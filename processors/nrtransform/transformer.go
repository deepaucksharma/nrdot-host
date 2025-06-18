package nrtransform

import (
	"fmt"
	"sort"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

// Transformer handles metric transformations
type Transformer struct {
	config     *Config
	calculator *MetricCalculator
	logger     *zap.Logger
	programs   map[int]*vm.Program // Compiled expression programs
}

// NewTransformer creates a new transformer
func NewTransformer(config *Config, logger *zap.Logger) (*Transformer, error) {
	t := &Transformer{
		config:     config,
		calculator: NewMetricCalculator(),
		logger:     logger,
		programs:   make(map[int]*vm.Program),
	}

	// Pre-compile expressions
	for i, transform := range config.Transformations {
		if transform.Type == TransformTypeCombine && transform.Expression != "" {
			program, err := expr.Compile(transform.Expression)
			if err != nil {
				return nil, fmt.Errorf("failed to compile expression %s: %w", transform.Expression, err)
			}
			t.programs[i] = program
		}
	}

	return t, nil
}

// Transform applies all configured transformations to the metrics
func (t *Transformer) Transform(metrics pmetric.Metrics) error {
	// Process each resource
	resourceMetrics := metrics.ResourceMetrics()
	for i := 0; i < resourceMetrics.Len(); i++ {
		rm := resourceMetrics.At(i)
		scopeMetrics := rm.ScopeMetrics()
		
		for j := 0; j < scopeMetrics.Len(); j++ {
			sm := scopeMetrics.At(j)
			originalMetrics := sm.Metrics()
			
			// Collect all metrics
			allMetrics := make([]pmetric.Metric, 0, originalMetrics.Len())
			metricsToRemove := make(map[string]bool)
			
			// Copy existing metrics
			for k := 0; k < originalMetrics.Len(); k++ {
				metric := pmetric.NewMetric()
				originalMetrics.At(k).CopyTo(metric)
				allMetrics = append(allMetrics, metric)
			}
			
			// Build metric map for transformations
			metricMap := make(map[string]pmetric.Metric)
			for _, metric := range allMetrics {
				metricMap[metric.Name()] = metric
			}
			
			// Apply transformations
			for idx, transform := range t.config.Transformations {
				transformedMetrics, toRemove, err := t.applyTransformation(transform, originalMetrics, metricMap, idx)
				if err != nil {
					t.logger.Error("Failed to apply transformation",
						zap.Error(err),
						zap.String("type", string(transform.Type)),
						zap.String("metric", transform.MetricName))
					continue
				}
				
				allMetrics = append(allMetrics, transformedMetrics...)
				for _, name := range toRemove {
					metricsToRemove[name] = true
				}
				
				// Update metric map with new metrics
				for _, metric := range transformedMetrics {
					metricMap[metric.Name()] = metric
				}
			}
			
			// Create a new scope metrics to rebuild
			tempMetrics := make([]pmetric.Metric, 0)
			for _, metric := range allMetrics {
				if !metricsToRemove[metric.Name()] {
					tempMetrics = append(tempMetrics, metric)
				}
			}
			
			// Clear and rebuild the metrics slice
			// Since we can't directly clear, we'll work around it
			currentLen := originalMetrics.Len()
			
			// First, update existing slots
			for k := 0; k < len(tempMetrics) && k < currentLen; k++ {
				tempMetrics[k].CopyTo(originalMetrics.At(k))
			}
			
			// If we have more metrics than slots, append them
			if len(tempMetrics) > currentLen {
				for k := currentLen; k < len(tempMetrics); k++ {
					newMetric := originalMetrics.AppendEmpty()
					tempMetrics[k].CopyTo(newMetric)
				}
			}
			
			// If we have fewer metrics than slots, we can't remove them
			// This is a limitation of the pmetric API
			// Log a warning if this happens
			if len(tempMetrics) < currentLen {
				t.logger.Warn("Cannot reduce metric count due to pmetric API limitations",
					zap.Int("current", currentLen),
					zap.Int("desired", len(tempMetrics)))
			}
		}
	}

	return nil
}

func (t *Transformer) buildMetricMap(metrics pmetric.MetricSlice) map[string]pmetric.Metric {
	metricMap := make(map[string]pmetric.Metric)
	for i := 0; i < metrics.Len(); i++ {
		metric := metrics.At(i)
		metricMap[metric.Name()] = metric
	}
	return metricMap
}

func (t *Transformer) applyTransformation(
	transform TransformationConfig,
	metrics pmetric.MetricSlice,
	metricMap map[string]pmetric.Metric,
	idx int,
) ([]pmetric.Metric, []string, error) {
	var newMetrics []pmetric.Metric
	var toRemove []string

	switch transform.Type {
	case TransformTypeAggregate:
		metric, exists := metricMap[transform.MetricName]
		if !exists {
			return nil, nil, nil
		}
		
		aggregated, err := t.calculator.Aggregate(metric, transform.Aggregation, transform.GroupBy, transform.OutputMetric)
		if err != nil {
			return nil, nil, err
		}
		newMetrics = append(newMetrics, aggregated)

	case TransformTypeCalculateRate:
		metric, exists := metricMap[transform.MetricName]
		if !exists {
			return nil, nil, nil
		}
		
		rate, err := t.calculator.CalculateRate(metric, transform.OutputMetric)
		if err != nil {
			return nil, nil, err
		}
		newMetrics = append(newMetrics, rate)

	case TransformTypeCalculateDelta:
		metric, exists := metricMap[transform.MetricName]
		if !exists {
			return nil, nil, nil
		}
		
		delta, err := t.calculator.CalculateDelta(metric, transform.OutputMetric)
		if err != nil {
			return nil, nil, err
		}
		newMetrics = append(newMetrics, delta)

	case TransformTypeConvertUnit:
		metric, exists := metricMap[transform.MetricName]
		if !exists {
			return nil, nil, nil
		}
		
		converted, err := t.calculator.ConvertUnit(metric, transform.FromUnit, transform.ToUnit, transform.OutputMetric)
		if err != nil {
			return nil, nil, err
		}
		newMetrics = append(newMetrics, converted)

	case TransformTypeCombine:
		combined, err := t.combineMetrics(transform, metricMap, idx)
		if err != nil {
			return nil, nil, err
		}
		// Check if the metric is valid by checking if it has a type set
		if combined.Type() != pmetric.MetricTypeEmpty {
			newMetrics = append(newMetrics, combined)
		}

	case TransformTypeRename:
		for i := 0; i < metrics.Len(); i++ {
			metric := metrics.At(i)
			if metric.Name() == transform.MetricName {
				newMetric := pmetric.NewMetric()
				metric.CopyTo(newMetric)
				newMetric.SetName(transform.OutputMetric)
				newMetrics = append(newMetrics, newMetric)
				toRemove = append(toRemove, transform.MetricName)
			}
		}

	case TransformTypeFilter:
		_, removed, err := t.filterMetrics(transform, metrics)
		if err != nil {
			return nil, nil, err
		}
		toRemove = append(toRemove, removed...)

	case TransformTypeExtractLabel:
		metric, exists := metricMap[transform.MetricName]
		if !exists {
			return nil, nil, nil
		}
		
		extracted := t.extractLabel(metric, transform)
		if extracted.Type() != pmetric.MetricTypeEmpty {
			newMetrics = append(newMetrics, extracted)
		}
	}

	return newMetrics, toRemove, nil
}

func (t *Transformer) combineMetrics(transform TransformationConfig, metricMap map[string]pmetric.Metric, idx int) (pmetric.Metric, error) {
	// Get all metrics involved
	var baseMetric pmetric.Metric
	hasBase := false
	for _, metricName := range transform.Metrics {
		metric, exists := metricMap[metricName]
		if !exists {
			return pmetric.NewMetric(), nil // Skip if any metric is missing
		}
		if !hasBase {
			baseMetric = metric
			hasBase = true
		}
	}

	if !hasBase {
		return pmetric.NewMetric(), nil
	}

	// Create new metric
	newMetric := pmetric.NewMetric()
	newMetric.SetName(transform.OutputMetric)
	newMetric.SetDescription(fmt.Sprintf("Combined metric from: %s", strings.Join(transform.Metrics, ", ")))

	// Get compiled program
	program, ok := t.programs[idx]
	if !ok {
		return pmetric.Metric{}, fmt.Errorf("no compiled program for transformation %d", idx)
	}

	switch baseMetric.Type() {
	case pmetric.MetricTypeGauge:
		newMetric.SetEmptyGauge()
		t.combineGaugeMetrics(transform, metricMap, newMetric.Gauge(), program)

	case pmetric.MetricTypeSum:
		newMetric.SetEmptySum()
		newMetric.Sum().SetIsMonotonic(false)
		newMetric.Sum().SetAggregationTemporality(baseMetric.Sum().AggregationTemporality())
		t.combineSumMetrics(transform, metricMap, newMetric.Sum(), program)

	default:
		return pmetric.NewMetric(), fmt.Errorf("combine not supported for metric type: %s", baseMetric.Type())
	}

	return newMetric, nil
}

func (t *Transformer) combineGaugeMetrics(transform TransformationConfig, metricMap map[string]pmetric.Metric, gauge pmetric.Gauge, program *vm.Program) {
	// Group data points by attributes
	dpGroups := make(map[string]*DataPointGroup)

	for _, metricName := range transform.Metrics {
		metric, exists := metricMap[metricName]
		if !exists || metric.Type() != pmetric.MetricTypeGauge {
			continue
		}

		dataPoints := metric.Gauge().DataPoints()
		for i := 0; i < dataPoints.Len(); i++ {
			dp := dataPoints.At(i)
			key := t.attributeKey(dp.Attributes())
			
			group, exists := dpGroups[key]
			if !exists {
				group = &DataPointGroup{
					attributes: pcommon.NewMap(),
					values:     make(map[string]float64),
					timestamp:  dp.Timestamp(),
				}
				dp.Attributes().CopyTo(group.attributes)
				dpGroups[key] = group
			}
			
			// Use sanitized metric name as variable name in expression
			varName := strings.ReplaceAll(metricName, ".", "_")
			varName = strings.ReplaceAll(varName, "-", "_")
			group.values[varName] = dp.DoubleValue()
			if dp.Timestamp() > group.timestamp {
				group.timestamp = dp.Timestamp()
			}
		}
	}

	// Evaluate expression for each group
	for _, group := range dpGroups {
		result, err := vm.Run(program, group.values)
		if err != nil {
			t.logger.Warn("Failed to evaluate expression",
				zap.Error(err),
				zap.String("expression", transform.Expression))
			continue
		}

		if value, ok := result.(float64); ok {
			newDp := gauge.DataPoints().AppendEmpty()
			group.attributes.CopyTo(newDp.Attributes())
			newDp.SetTimestamp(group.timestamp)
			newDp.SetDoubleValue(value)
		}
	}
}

func (t *Transformer) combineSumMetrics(transform TransformationConfig, metricMap map[string]pmetric.Metric, sum pmetric.Sum, program *vm.Program) {
	// Similar to combineGaugeMetrics but for Sum type
	dpGroups := make(map[string]*DataPointGroup)

	for _, metricName := range transform.Metrics {
		metric, exists := metricMap[metricName]
		if !exists || metric.Type() != pmetric.MetricTypeSum {
			continue
		}

		dataPoints := metric.Sum().DataPoints()
		for i := 0; i < dataPoints.Len(); i++ {
			dp := dataPoints.At(i)
			key := t.attributeKey(dp.Attributes())
			
			group, exists := dpGroups[key]
			if !exists {
				group = &DataPointGroup{
					attributes: pcommon.NewMap(),
					values:     make(map[string]float64),
					timestamp:  dp.Timestamp(),
				}
				dp.Attributes().CopyTo(group.attributes)
				dpGroups[key] = group
			}
			
			// Use sanitized metric name as variable name in expression
			varName := strings.ReplaceAll(metricName, ".", "_")
			varName = strings.ReplaceAll(varName, "-", "_")
			group.values[varName] = dp.DoubleValue()
			if dp.Timestamp() > group.timestamp {
				group.timestamp = dp.Timestamp()
			}
		}
	}

	// Evaluate expression for each group
	for _, group := range dpGroups {
		result, err := vm.Run(program, group.values)
		if err != nil {
			t.logger.Warn("Failed to evaluate expression",
				zap.Error(err),
				zap.String("expression", transform.Expression))
			continue
		}

		if value, ok := result.(float64); ok {
			newDp := sum.DataPoints().AppendEmpty()
			group.attributes.CopyTo(newDp.Attributes())
			newDp.SetTimestamp(group.timestamp)
			newDp.SetDoubleValue(value)
		}
	}
}

func (t *Transformer) filterMetrics(transform TransformationConfig, metrics pmetric.MetricSlice) ([]pmetric.Metric, []string, error) {
	var toRemove []string
	
	program, err := expr.Compile(transform.Condition)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compile filter condition: %w", err)
	}

	for i := 0; i < metrics.Len(); i++ {
		metric := metrics.At(i)
		env := map[string]interface{}{
			"name": metric.Name(),
			"type": metric.Type().String(),
			"unit": metric.Unit(),
		}

		result, err := expr.Run(program, env)
		if err != nil {
			t.logger.Warn("Failed to evaluate filter condition",
				zap.Error(err),
				zap.String("metric", metric.Name()))
			continue
		}

		if keep, ok := result.(bool); ok && !keep {
			toRemove = append(toRemove, metric.Name())
		}
	}

	return nil, toRemove, nil
}

func (t *Transformer) extractLabel(metric pmetric.Metric, transform TransformationConfig) pmetric.Metric {
	newMetric := pmetric.NewMetric()
	newMetric.SetName(transform.OutputMetric)
	newMetric.SetDescription(fmt.Sprintf("Label %s extracted from %s", transform.LabelKey, metric.Name()))

	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		newMetric.SetEmptyGauge()
		t.extractLabelFromGauge(metric.Gauge(), newMetric.Gauge(), transform.LabelKey, transform.LabelValue)

	case pmetric.MetricTypeSum:
		newMetric.SetEmptySum()
		newMetric.Sum().SetIsMonotonic(metric.Sum().IsMonotonic())
		newMetric.Sum().SetAggregationTemporality(metric.Sum().AggregationTemporality())
		t.extractLabelFromSum(metric.Sum(), newMetric.Sum(), transform.LabelKey, transform.LabelValue)

	default:
		t.logger.Warn("Extract label not supported for metric type",
			zap.String("type", metric.Type().String()))
	}

	return newMetric
}

func (t *Transformer) extractLabelFromGauge(gauge pmetric.Gauge, newGauge pmetric.Gauge, labelKey, labelValue string) {
	dataPoints := gauge.DataPoints()
	for i := 0; i < dataPoints.Len(); i++ {
		dp := dataPoints.At(i)
		
		if labelValue == "" {
			// Extract all unique label values
			if _, ok := dp.Attributes().Get(labelKey); ok {
				newDp := newGauge.DataPoints().AppendEmpty()
				dp.CopyTo(newDp)
				// Set value to 1 for presence
				newDp.SetDoubleValue(1.0)
			}
		} else {
			// Extract specific label value
			if val, ok := dp.Attributes().Get(labelKey); ok && val.AsString() == labelValue {
				newDp := newGauge.DataPoints().AppendEmpty()
				dp.CopyTo(newDp)
			}
		}
	}
}

func (t *Transformer) extractLabelFromSum(sum pmetric.Sum, newSum pmetric.Sum, labelKey, labelValue string) {
	dataPoints := sum.DataPoints()
	for i := 0; i < dataPoints.Len(); i++ {
		dp := dataPoints.At(i)
		
		if labelValue == "" {
			// Extract all unique label values
			if _, ok := dp.Attributes().Get(labelKey); ok {
				newDp := newSum.DataPoints().AppendEmpty()
				dp.CopyTo(newDp)
				// Set value to 1 for presence
				newDp.SetDoubleValue(1.0)
			}
		} else {
			// Extract specific label value
			if val, ok := dp.Attributes().Get(labelKey); ok && val.AsString() == labelValue {
				newDp := newSum.DataPoints().AppendEmpty()
				dp.CopyTo(newDp)
			}
		}
	}
}

func (t *Transformer) removeMetrics(metrics pmetric.MetricSlice, toRemove map[string]bool) {
	// Since pmetric.MetricSlice doesn't support RemoveAt, we need to work around it
	// by not adding removed metrics when we rebuild the metrics list
	// This is handled in the Transform method by tracking which metrics to remove
	// and not copying them to the final output
}

func (t *Transformer) attributeKey(attrs pcommon.Map) string {
	keys := []string{}
	attrs.Range(func(k string, v pcommon.Value) bool {
		keys = append(keys, fmt.Sprintf("%s=%s", k, v.AsString()))
		return true
	})
	sort.Strings(keys)
	return strings.Join(keys, ",")
}

// DataPointGroup represents a group of data points with same attributes
type DataPointGroup struct {
	attributes pcommon.Map
	values     map[string]float64
	timestamp  pcommon.Timestamp
}