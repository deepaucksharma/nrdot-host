package nrcap

import (
	"math/rand"
	"sync"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

// CardinalityLimiter handles cardinality limiting logic
type CardinalityLimiter struct {
	config  *Config
	tracker *CardinalityTracker
	logger  *zap.Logger

	// Label value tracking for high cardinality detection
	labelCardinality map[string]map[string]struct{}
	labelMutex       sync.RWMutex

	// Random source for sampling
	rand *rand.Rand

	// Alert tracking
	alertsSent map[string]time.Time
	alertMutex sync.Mutex
}

// NewCardinalityLimiter creates a new cardinality limiter
func NewCardinalityLimiter(cfg *Config, logger *zap.Logger) *CardinalityLimiter {
	return &CardinalityLimiter{
		config:           cfg,
		tracker:          NewCardinalityTracker(cfg.WindowSize),
		logger:           logger,
		labelCardinality: make(map[string]map[string]struct{}),
		rand:             rand.New(rand.NewSource(time.Now().UnixNano())),
		alertsSent:       make(map[string]time.Time),
	}
}

// ProcessMetrics applies cardinality limits to metrics
func (cl *CardinalityLimiter) ProcessMetrics(metrics pmetric.Metrics) (pmetric.Metrics, error) {
	// First, remove deny labels from all metrics
	cl.removeDenyLabelsFromMetrics(metrics)
	
	// Track high cardinality labels
	cl.trackLabelCardinality(metrics)

	// Create output metrics
	output := pmetric.NewMetrics()
	
	resourceMetrics := metrics.ResourceMetrics()
	for i := 0; i < resourceMetrics.Len(); i++ {
		rm := resourceMetrics.At(i)
		outputRM := output.ResourceMetrics().AppendEmpty()
		rm.Resource().CopyTo(outputRM.Resource())

		scopeMetrics := rm.ScopeMetrics()
		for j := 0; j < scopeMetrics.Len(); j++ {
			sm := scopeMetrics.At(j)
			outputSM := outputRM.ScopeMetrics().AppendEmpty()
			sm.Scope().CopyTo(outputSM.Scope())

			metrics := sm.Metrics()
			for k := 0; k < metrics.Len(); k++ {
				metric := metrics.At(k)
				cl.processMetric(metric, outputSM.Metrics())
			}
		}
	}

	// Periodic cleanup
	cl.tracker.CleanupOldEntries()

	// Check for alerts
	cl.checkAlerts()

	return output, nil
}

// processMetric processes a single metric
func (cl *CardinalityLimiter) processMetric(metric pmetric.Metric, output pmetric.MetricSlice) {
	metricName := metric.Name()
	
	// Get limit for this metric
	limit := cl.getMetricLimit(metricName)
	
	// Apply limiting strategy
	switch cl.config.Strategy {
	case StrategyDrop:
		cl.handleDrop(metric, output, limit)
	case StrategyAggregate:
		cl.handleAggregate(metric, output, limit)
	case StrategySample:
		cl.handleSample(metric, output, limit)
	case StrategyOldest:
		cl.handleOldest(metric, output, metricName, limit)
	}
}

// handleDrop handles the drop strategy
func (cl *CardinalityLimiter) handleDrop(metric pmetric.Metric, output pmetric.MetricSlice, limit int) {
	// For drop strategy, we need to check each data point separately
	outputMetric := output.AppendEmpty()
	metric.CopyTo(outputMetric)
	
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		cl.filterDataPoints(metric, outputMetric.Gauge().DataPoints(), limit)
	case pmetric.MetricTypeSum:
		cl.filterDataPoints(metric, outputMetric.Sum().DataPoints(), limit)
	case pmetric.MetricTypeHistogram:
		cl.filterHistogramDataPoints(metric, outputMetric.Histogram().DataPoints(), limit)
	case pmetric.MetricTypeSummary:
		cl.filterSummaryDataPoints(metric, outputMetric.Summary().DataPoints(), limit)
	case pmetric.MetricTypeExponentialHistogram:
		cl.filterExponentialHistogramDataPoints(metric, outputMetric.ExponentialHistogram().DataPoints(), limit)
	}
	
	// Remove metric if no data points remain
	if cl.getDataPointCount(outputMetric) == 0 {
		output.RemoveIf(func(m pmetric.Metric) bool {
			return m.Name() == outputMetric.Name()
		})
	}
}

// filterDataPoints filters number data points based on cardinality
func (cl *CardinalityLimiter) filterDataPoints(metric pmetric.Metric, dps pmetric.NumberDataPointSlice, limit int) {
	toRemove := []int{}
	metricName := metric.Name()
	
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		
		// Create a temporary metric with just this data point for tracking
		tempMetric := pmetric.NewMetric()
		metric.CopyTo(tempMetric)
		
		// Clear all data points and add just this one
		switch tempMetric.Type() {
		case pmetric.MetricTypeGauge:
			tempMetric.Gauge().DataPoints().RemoveIf(func(_ pmetric.NumberDataPoint) bool { return true })
			dp.CopyTo(tempMetric.Gauge().DataPoints().AppendEmpty())
		case pmetric.MetricTypeSum:
			tempMetric.Sum().DataPoints().RemoveIf(func(_ pmetric.NumberDataPoint) bool { return true })
			dp.CopyTo(tempMetric.Sum().DataPoints().AppendEmpty())
		}
		
		// Check current cardinality before tracking
		currentCardinality := cl.tracker.GetCardinality(metricName)
		globalCardinality := cl.tracker.GetGlobalCardinality()
		
		isNew, _ := cl.tracker.TrackDataPoint(metricName, dp.Attributes())
		cl.tracker.IncrementStats("total")
		
		// Check if we're over limits
		overMetricLimit := currentCardinality >= limit && isNew
		overGlobalLimit := globalCardinality >= cl.config.GlobalLimit && isNew
		
		if isNew && (overMetricLimit || overGlobalLimit) {
			// New data point that would exceed limit, mark for removal
			toRemove = append(toRemove, i)
			cl.tracker.IncrementStats("dropped")
			cl.logger.Debug("Dropping data point due to cardinality limit",
				zap.String("metric", metric.Name()),
				zap.Int("data_point_index", i))
		}
	}
	
	// Remove data points in reverse order to maintain indices
	for i := len(toRemove) - 1; i >= 0; i-- {
		dps.RemoveIf(func(dp pmetric.NumberDataPoint) bool {
			return dps.At(toRemove[i]).Timestamp() == dp.Timestamp()
		})
	}
}

// Similar filter functions for other data point types...
func (cl *CardinalityLimiter) filterHistogramDataPoints(metric pmetric.Metric, dps pmetric.HistogramDataPointSlice, limit int) {
	// Similar implementation as filterDataPoints
	toRemove := []int{}
	metricName := metric.Name()
	
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		
		// Check current cardinality before tracking
		currentCardinality := cl.tracker.GetCardinality(metricName)
		globalCardinality := cl.tracker.GetGlobalCardinality()
		
		isNew, _ := cl.tracker.TrackDataPoint(metricName, dp.Attributes())
		cl.tracker.IncrementStats("total")
		
		// Check if we're over limits
		overMetricLimit := currentCardinality >= limit && isNew
		overGlobalLimit := globalCardinality >= cl.config.GlobalLimit && isNew
		
		if isNew && (overMetricLimit || overGlobalLimit) {
			toRemove = append(toRemove, i)
			cl.tracker.IncrementStats("dropped")
		}
	}
	
	for i := len(toRemove) - 1; i >= 0; i-- {
		dps.RemoveIf(func(dp pmetric.HistogramDataPoint) bool {
			return dps.At(toRemove[i]).Timestamp() == dp.Timestamp()
		})
	}
}

func (cl *CardinalityLimiter) filterSummaryDataPoints(metric pmetric.Metric, dps pmetric.SummaryDataPointSlice, limit int) {
	toRemove := []int{}
	metricName := metric.Name()
	
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		
		// Check current cardinality before tracking
		currentCardinality := cl.tracker.GetCardinality(metricName)
		globalCardinality := cl.tracker.GetGlobalCardinality()
		
		isNew, _ := cl.tracker.TrackDataPoint(metricName, dp.Attributes())
		cl.tracker.IncrementStats("total")
		
		// Check if we're over limits
		overMetricLimit := currentCardinality >= limit && isNew
		overGlobalLimit := globalCardinality >= cl.config.GlobalLimit && isNew
		
		if isNew && (overMetricLimit || overGlobalLimit) {
			toRemove = append(toRemove, i)
			cl.tracker.IncrementStats("dropped")
		}
	}
	
	for i := len(toRemove) - 1; i >= 0; i-- {
		dps.RemoveIf(func(dp pmetric.SummaryDataPoint) bool {
			return dps.At(toRemove[i]).Timestamp() == dp.Timestamp()
		})
	}
}

func (cl *CardinalityLimiter) filterExponentialHistogramDataPoints(metric pmetric.Metric, dps pmetric.ExponentialHistogramDataPointSlice, limit int) {
	toRemove := []int{}
	metricName := metric.Name()
	
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		
		// Check current cardinality before tracking
		currentCardinality := cl.tracker.GetCardinality(metricName)
		globalCardinality := cl.tracker.GetGlobalCardinality()
		
		isNew, _ := cl.tracker.TrackDataPoint(metricName, dp.Attributes())
		cl.tracker.IncrementStats("total")
		
		// Check if we're over limits
		overMetricLimit := currentCardinality >= limit && isNew
		overGlobalLimit := globalCardinality >= cl.config.GlobalLimit && isNew
		
		if isNew && (overMetricLimit || overGlobalLimit) {
			toRemove = append(toRemove, i)
			cl.tracker.IncrementStats("dropped")
		}
	}
	
	for i := len(toRemove) - 1; i >= 0; i-- {
		dps.RemoveIf(func(dp pmetric.ExponentialHistogramDataPoint) bool {
			return dps.At(toRemove[i]).Timestamp() == dp.Timestamp()
		})
	}
}

// getDataPointCount returns the number of data points in a metric
func (cl *CardinalityLimiter) getDataPointCount(metric pmetric.Metric) int {
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		return metric.Gauge().DataPoints().Len()
	case pmetric.MetricTypeSum:
		return metric.Sum().DataPoints().Len()
	case pmetric.MetricTypeHistogram:
		return metric.Histogram().DataPoints().Len()
	case pmetric.MetricTypeSummary:
		return metric.Summary().DataPoints().Len()
	case pmetric.MetricTypeExponentialHistogram:
		return metric.ExponentialHistogram().DataPoints().Len()
	}
	return 0
}

// handleAggregate handles the aggregate strategy
func (cl *CardinalityLimiter) handleAggregate(metric pmetric.Metric, output pmetric.MetricSlice, limit int) {
	metricName := metric.Name()
	
	// Create aggregated metric
	outputMetric := output.AppendEmpty()
	metric.CopyTo(outputMetric)
	
	// Always apply aggregation labels if specified, or remove high cardinality labels when over limit
	if len(cl.config.AggregationLabels) > 0 || cl.shouldAggregate(metricName, limit) {
		cl.removeHighCardinalityLabels(outputMetric)
		cl.tracker.IncrementStats("aggregated")
	}
	
	// Track all data points after aggregation
	cl.trackAllDataPoints(outputMetric)
}

// handleSample handles the sample strategy
func (cl *CardinalityLimiter) handleSample(metric pmetric.Metric, output pmetric.MetricSlice, limit int) {
	// Create output metric
	outputMetric := output.AppendEmpty()
	metric.CopyTo(outputMetric)
	
	// Process each data point
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		cl.sampleDataPoints(metric, outputMetric.Gauge().DataPoints(), limit)
	case pmetric.MetricTypeSum:
		cl.sampleDataPoints(metric, outputMetric.Sum().DataPoints(), limit)
	case pmetric.MetricTypeHistogram:
		cl.sampleHistogramDataPoints(metric, outputMetric.Histogram().DataPoints(), limit)
	case pmetric.MetricTypeSummary:
		cl.sampleSummaryDataPoints(metric, outputMetric.Summary().DataPoints(), limit)
	case pmetric.MetricTypeExponentialHistogram:
		cl.sampleExponentialHistogramDataPoints(metric, outputMetric.ExponentialHistogram().DataPoints(), limit)
	}
	
	// Remove metric if no data points remain
	if cl.getDataPointCount(outputMetric) == 0 {
		output.RemoveIf(func(m pmetric.Metric) bool {
			return m.Name() == outputMetric.Name()
		})
	}
}

// sampleDataPoints samples number data points based on sample rate
func (cl *CardinalityLimiter) sampleDataPoints(metric pmetric.Metric, dps pmetric.NumberDataPointSlice, limit int) {
	toRemove := []int{}
	metricName := metric.Name()
	
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		
		// Check current cardinality before tracking
		currentCardinality := cl.tracker.GetCardinality(metricName)
		globalCardinality := cl.tracker.GetGlobalCardinality()
		
		isNew, _ := cl.tracker.TrackDataPoint(metricName, dp.Attributes())
		cl.tracker.IncrementStats("total")
		
		// Check if we're over limits
		overMetricLimit := currentCardinality >= limit && isNew
		overGlobalLimit := globalCardinality >= cl.config.GlobalLimit && isNew
		
		if isNew && (overMetricLimit || overGlobalLimit) {
			// Sample based on configured rate
			if cl.rand.Float64() >= cl.config.SampleRate {
				toRemove = append(toRemove, i)
				cl.tracker.IncrementStats("dropped")
			} else {
				cl.tracker.IncrementStats("sampled")
			}
		}
	}
	
	// Remove data points in reverse order to maintain indices
	for i := len(toRemove) - 1; i >= 0; i-- {
		dps.RemoveIf(func(dp pmetric.NumberDataPoint) bool {
			return dps.At(toRemove[i]).Timestamp() == dp.Timestamp()
		})
	}
}

// sampleHistogramDataPoints samples histogram data points based on sample rate
func (cl *CardinalityLimiter) sampleHistogramDataPoints(metric pmetric.Metric, dps pmetric.HistogramDataPointSlice, limit int) {
	toRemove := []int{}
	metricName := metric.Name()
	
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		
		// Check current cardinality before tracking
		currentCardinality := cl.tracker.GetCardinality(metricName)
		globalCardinality := cl.tracker.GetGlobalCardinality()
		
		isNew, _ := cl.tracker.TrackDataPoint(metricName, dp.Attributes())
		cl.tracker.IncrementStats("total")
		
		// Check if we're over limits
		overMetricLimit := currentCardinality >= limit && isNew
		overGlobalLimit := globalCardinality >= cl.config.GlobalLimit && isNew
		
		if isNew && (overMetricLimit || overGlobalLimit) {
			// Sample based on configured rate
			if cl.rand.Float64() >= cl.config.SampleRate {
				toRemove = append(toRemove, i)
				cl.tracker.IncrementStats("dropped")
			} else {
				cl.tracker.IncrementStats("sampled")
			}
		}
	}
	
	// Remove data points in reverse order to maintain indices
	for i := len(toRemove) - 1; i >= 0; i-- {
		dps.RemoveIf(func(dp pmetric.HistogramDataPoint) bool {
			return dps.At(toRemove[i]).Timestamp() == dp.Timestamp()
		})
	}
}

// sampleSummaryDataPoints samples summary data points based on sample rate
func (cl *CardinalityLimiter) sampleSummaryDataPoints(metric pmetric.Metric, dps pmetric.SummaryDataPointSlice, limit int) {
	toRemove := []int{}
	metricName := metric.Name()
	
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		
		// Check current cardinality before tracking
		currentCardinality := cl.tracker.GetCardinality(metricName)
		globalCardinality := cl.tracker.GetGlobalCardinality()
		
		isNew, _ := cl.tracker.TrackDataPoint(metricName, dp.Attributes())
		cl.tracker.IncrementStats("total")
		
		// Check if we're over limits
		overMetricLimit := currentCardinality >= limit && isNew
		overGlobalLimit := globalCardinality >= cl.config.GlobalLimit && isNew
		
		if isNew && (overMetricLimit || overGlobalLimit) {
			// Sample based on configured rate
			if cl.rand.Float64() >= cl.config.SampleRate {
				toRemove = append(toRemove, i)
				cl.tracker.IncrementStats("dropped")
			} else {
				cl.tracker.IncrementStats("sampled")
			}
		}
	}
	
	// Remove data points in reverse order to maintain indices
	for i := len(toRemove) - 1; i >= 0; i-- {
		dps.RemoveIf(func(dp pmetric.SummaryDataPoint) bool {
			return dps.At(toRemove[i]).Timestamp() == dp.Timestamp()
		})
	}
}

// sampleExponentialHistogramDataPoints samples exponential histogram data points based on sample rate
func (cl *CardinalityLimiter) sampleExponentialHistogramDataPoints(metric pmetric.Metric, dps pmetric.ExponentialHistogramDataPointSlice, limit int) {
	toRemove := []int{}
	metricName := metric.Name()
	
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		
		// Check current cardinality before tracking
		currentCardinality := cl.tracker.GetCardinality(metricName)
		globalCardinality := cl.tracker.GetGlobalCardinality()
		
		isNew, _ := cl.tracker.TrackDataPoint(metricName, dp.Attributes())
		cl.tracker.IncrementStats("total")
		
		// Check if we're over limits
		overMetricLimit := currentCardinality >= limit && isNew
		overGlobalLimit := globalCardinality >= cl.config.GlobalLimit && isNew
		
		if isNew && (overMetricLimit || overGlobalLimit) {
			// Sample based on configured rate
			if cl.rand.Float64() >= cl.config.SampleRate {
				toRemove = append(toRemove, i)
				cl.tracker.IncrementStats("dropped")
			} else {
				cl.tracker.IncrementStats("sampled")
			}
		}
	}
	
	// Remove data points in reverse order to maintain indices
	for i := len(toRemove) - 1; i >= 0; i-- {
		dps.RemoveIf(func(dp pmetric.ExponentialHistogramDataPoint) bool {
			return dps.At(toRemove[i]).Timestamp() == dp.Timestamp()
		})
	}
}

// handleOldest handles the oldest strategy
func (cl *CardinalityLimiter) handleOldest(metric pmetric.Metric, output pmetric.MetricSlice, metricName string, limit int) {
	// Track all data points and remove oldest if needed
	outputMetric := output.AppendEmpty()
	metric.CopyTo(outputMetric)
	
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		cl.processOldestDataPoints(metric, outputMetric.Gauge().DataPoints(), limit)
	case pmetric.MetricTypeSum:
		cl.processOldestDataPoints(metric, outputMetric.Sum().DataPoints(), limit)
	case pmetric.MetricTypeHistogram:
		cl.processOldestHistogramDataPoints(metric, outputMetric.Histogram().DataPoints(), limit)
	case pmetric.MetricTypeSummary:
		cl.processOldestSummaryDataPoints(metric, outputMetric.Summary().DataPoints(), limit)
	case pmetric.MetricTypeExponentialHistogram:
		cl.processOldestExponentialHistogramDataPoints(metric, outputMetric.ExponentialHistogram().DataPoints(), limit)
	}
}

// processOldestDataPoints processes data points using oldest strategy
func (cl *CardinalityLimiter) processOldestDataPoints(metric pmetric.Metric, dps pmetric.NumberDataPointSlice, limit int) {
	metricName := metric.Name()
	
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		
		// Check current cardinality before tracking
		currentCardinality := cl.tracker.GetCardinality(metricName)
		
		isNew, _ := cl.tracker.TrackDataPoint(metricName, dp.Attributes())
		cl.tracker.IncrementStats("total")
		
		if isNew && currentCardinality >= limit {
			// Remove oldest entry to make room
			oldest := cl.tracker.GetOldestEntries(metricName, 1)
			if len(oldest) > 0 {
				cl.tracker.RemoveEntry(metricName, oldest[0])
			}
			// Re-track this new entry
			cl.tracker.TrackDataPoint(metricName, dp.Attributes())
		}
	}
}

// processOldestHistogramDataPoints processes histogram data points using oldest strategy
func (cl *CardinalityLimiter) processOldestHistogramDataPoints(metric pmetric.Metric, dps pmetric.HistogramDataPointSlice, limit int) {
	metricName := metric.Name()
	
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		
		// Check current cardinality before tracking
		currentCardinality := cl.tracker.GetCardinality(metricName)
		
		isNew, _ := cl.tracker.TrackDataPoint(metricName, dp.Attributes())
		cl.tracker.IncrementStats("total")
		
		if isNew && currentCardinality >= limit {
			// Remove oldest entry to make room
			oldest := cl.tracker.GetOldestEntries(metricName, 1)
			if len(oldest) > 0 {
				cl.tracker.RemoveEntry(metricName, oldest[0])
			}
			// Re-track this new entry
			cl.tracker.TrackDataPoint(metricName, dp.Attributes())
		}
	}
}

// processOldestSummaryDataPoints processes summary data points using oldest strategy
func (cl *CardinalityLimiter) processOldestSummaryDataPoints(metric pmetric.Metric, dps pmetric.SummaryDataPointSlice, limit int) {
	metricName := metric.Name()
	
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		
		// Check current cardinality before tracking
		currentCardinality := cl.tracker.GetCardinality(metricName)
		
		isNew, _ := cl.tracker.TrackDataPoint(metricName, dp.Attributes())
		cl.tracker.IncrementStats("total")
		
		if isNew && currentCardinality >= limit {
			// Remove oldest entry to make room
			oldest := cl.tracker.GetOldestEntries(metricName, 1)
			if len(oldest) > 0 {
				cl.tracker.RemoveEntry(metricName, oldest[0])
			}
			// Re-track this new entry
			cl.tracker.TrackDataPoint(metricName, dp.Attributes())
		}
	}
}

// processOldestExponentialHistogramDataPoints processes exponential histogram data points using oldest strategy
func (cl *CardinalityLimiter) processOldestExponentialHistogramDataPoints(metric pmetric.Metric, dps pmetric.ExponentialHistogramDataPointSlice, limit int) {
	metricName := metric.Name()
	
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		
		// Check current cardinality before tracking
		currentCardinality := cl.tracker.GetCardinality(metricName)
		
		isNew, _ := cl.tracker.TrackDataPoint(metricName, dp.Attributes())
		cl.tracker.IncrementStats("total")
		
		if isNew && currentCardinality >= limit {
			// Remove oldest entry to make room
			oldest := cl.tracker.GetOldestEntries(metricName, 1)
			if len(oldest) > 0 {
				cl.tracker.RemoveEntry(metricName, oldest[0])
			}
			// Re-track this new entry
			cl.tracker.TrackDataPoint(metricName, dp.Attributes())
		}
	}
}

// removeHighCardinalityLabels removes labels based on configuration
func (cl *CardinalityLimiter) removeHighCardinalityLabels(metric pmetric.Metric) {
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		cl.removeLabelsFromDataPoints(metric.Gauge().DataPoints())
	case pmetric.MetricTypeSum:
		cl.removeLabelsFromDataPoints(metric.Sum().DataPoints())
	case pmetric.MetricTypeHistogram:
		cl.removeLabelsFromHistogramDataPoints(metric.Histogram().DataPoints())
	case pmetric.MetricTypeSummary:
		cl.removeLabelsFromSummaryDataPoints(metric.Summary().DataPoints())
	case pmetric.MetricTypeExponentialHistogram:
		cl.removeLabelsFromExponentialHistogramDataPoints(metric.ExponentialHistogram().DataPoints())
	}
}

// removeLabelsFromDataPoints removes labels from number data points
func (cl *CardinalityLimiter) removeLabelsFromDataPoints(dps pmetric.NumberDataPointSlice) {
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		attrs := dp.Attributes()
		
		// Keep only allowed labels or remove denied labels
		newAttrs := pcommon.NewMap()
		
		if len(cl.config.AggregationLabels) > 0 {
			// Keep only aggregation labels
			for _, label := range cl.config.AggregationLabels {
				if val, ok := attrs.Get(label); ok {
					newAttrs.PutStr(label, val.AsString())
				}
			}
		} else {
			// Remove denied labels
			attrs.Range(func(k string, v pcommon.Value) bool {
				if !cl.isDeniedLabel(k) {
					v.CopyTo(newAttrs.PutEmpty(k))
				}
				return true
			})
		}
		
		newAttrs.CopyTo(attrs)
	}
}

// removeLabelsFromHistogramDataPoints removes labels from histogram data points
func (cl *CardinalityLimiter) removeLabelsFromHistogramDataPoints(dps pmetric.HistogramDataPointSlice) {
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		attrs := dp.Attributes()
		
		newAttrs := pcommon.NewMap()
		
		if len(cl.config.AggregationLabels) > 0 {
			for _, label := range cl.config.AggregationLabels {
				if val, ok := attrs.Get(label); ok {
					newAttrs.PutStr(label, val.AsString())
				}
			}
		} else {
			attrs.Range(func(k string, v pcommon.Value) bool {
				if !cl.isDeniedLabel(k) {
					v.CopyTo(newAttrs.PutEmpty(k))
				}
				return true
			})
		}
		
		newAttrs.CopyTo(attrs)
	}
}

// removeLabelsFromSummaryDataPoints removes labels from summary data points
func (cl *CardinalityLimiter) removeLabelsFromSummaryDataPoints(dps pmetric.SummaryDataPointSlice) {
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		attrs := dp.Attributes()
		
		newAttrs := pcommon.NewMap()
		
		if len(cl.config.AggregationLabels) > 0 {
			for _, label := range cl.config.AggregationLabels {
				if val, ok := attrs.Get(label); ok {
					newAttrs.PutStr(label, val.AsString())
				}
			}
		} else {
			attrs.Range(func(k string, v pcommon.Value) bool {
				if !cl.isDeniedLabel(k) {
					v.CopyTo(newAttrs.PutEmpty(k))
				}
				return true
			})
		}
		
		newAttrs.CopyTo(attrs)
	}
}

// removeLabelsFromExponentialHistogramDataPoints removes labels from exponential histogram data points
func (cl *CardinalityLimiter) removeLabelsFromExponentialHistogramDataPoints(dps pmetric.ExponentialHistogramDataPointSlice) {
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		attrs := dp.Attributes()
		
		newAttrs := pcommon.NewMap()
		
		if len(cl.config.AggregationLabels) > 0 {
			for _, label := range cl.config.AggregationLabels {
				if val, ok := attrs.Get(label); ok {
					newAttrs.PutStr(label, val.AsString())
				}
			}
		} else {
			attrs.Range(func(k string, v pcommon.Value) bool {
				if !cl.isDeniedLabel(k) {
					v.CopyTo(newAttrs.PutEmpty(k))
				}
				return true
			})
		}
		
		newAttrs.CopyTo(attrs)
	}
}

// isDeniedLabel checks if a label is in the deny list
func (cl *CardinalityLimiter) isDeniedLabel(label string) bool {
	for _, denied := range cl.config.DenyLabels {
		if label == denied {
			return true
		}
	}
	return false
}

// getMetricLimit returns the limit for a specific metric
func (cl *CardinalityLimiter) getMetricLimit(metricName string) int {
	if limit, exists := cl.config.MetricLimits[metricName]; exists {
		return limit
	}
	return cl.config.DefaultLimit
}

// trackLabelCardinality tracks unique values per label
func (cl *CardinalityLimiter) trackLabelCardinality(metrics pmetric.Metrics) {
	cl.labelMutex.Lock()
	defer cl.labelMutex.Unlock()

	resourceMetrics := metrics.ResourceMetrics()
	for i := 0; i < resourceMetrics.Len(); i++ {
		rm := resourceMetrics.At(i)
		scopeMetrics := rm.ScopeMetrics()
		for j := 0; j < scopeMetrics.Len(); j++ {
			sm := scopeMetrics.At(j)
			metrics := sm.Metrics()
			for k := 0; k < metrics.Len(); k++ {
				metric := metrics.At(k)
				cl.trackMetricLabels(metric)
			}
		}
	}

	// Update tracker with high cardinality labels
	for label, values := range cl.labelCardinality {
		cl.tracker.TrackLabelCardinality(label, len(values))
	}
}

// trackMetricLabels tracks labels from a single metric
func (cl *CardinalityLimiter) trackMetricLabels(metric pmetric.Metric) {
	var processAttributes func(attrs pcommon.Map)
	processAttributes = func(attrs pcommon.Map) {
		attrs.Range(func(k string, v pcommon.Value) bool {
			if _, exists := cl.labelCardinality[k]; !exists {
				cl.labelCardinality[k] = make(map[string]struct{})
			}
			cl.labelCardinality[k][v.AsString()] = struct{}{}
			return true
		})
	}

	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		dps := metric.Gauge().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			processAttributes(dps.At(i).Attributes())
		}
	case pmetric.MetricTypeSum:
		dps := metric.Sum().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			processAttributes(dps.At(i).Attributes())
		}
	case pmetric.MetricTypeHistogram:
		dps := metric.Histogram().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			processAttributes(dps.At(i).Attributes())
		}
	case pmetric.MetricTypeSummary:
		dps := metric.Summary().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			processAttributes(dps.At(i).Attributes())
		}
	case pmetric.MetricTypeExponentialHistogram:
		dps := metric.ExponentialHistogram().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			processAttributes(dps.At(i).Attributes())
		}
	}
}

// checkAlerts checks if alerts should be sent
func (cl *CardinalityLimiter) checkAlerts() {
	cl.alertMutex.Lock()
	defer cl.alertMutex.Unlock()

	globalCardinality := cl.tracker.GetGlobalCardinality()
	threshold := float64(cl.config.GlobalLimit) * float64(cl.config.AlertThreshold) / 100.0

	if float64(globalCardinality) > threshold {
		// Check if we've sent an alert recently
		lastAlert, exists := cl.alertsSent["global"]
		if !exists || time.Since(lastAlert) > 5*time.Minute {
			cl.logger.Warn("Global cardinality threshold exceeded",
				zap.Int("current", globalCardinality),
				zap.Int("limit", cl.config.GlobalLimit),
				zap.Int("threshold_percent", cl.config.AlertThreshold))
			cl.alertsSent["global"] = time.Now()
		}
	}
}

// GetStats returns current statistics
func (cl *CardinalityLimiter) GetStats() CardinalityStats {
	return cl.tracker.GetStats()
}

// removeDenyLabelsFromMetrics removes deny labels from all metrics
func (cl *CardinalityLimiter) removeDenyLabelsFromMetrics(metrics pmetric.Metrics) {
	if len(cl.config.DenyLabels) == 0 {
		return
	}
	
	resourceMetrics := metrics.ResourceMetrics()
	for i := 0; i < resourceMetrics.Len(); i++ {
		rm := resourceMetrics.At(i)
		scopeMetrics := rm.ScopeMetrics()
		for j := 0; j < scopeMetrics.Len(); j++ {
			sm := scopeMetrics.At(j)
			metrics := sm.Metrics()
			for k := 0; k < metrics.Len(); k++ {
				metric := metrics.At(k)
				cl.removeDenyLabelsFromMetric(metric)
			}
		}
	}
}

// removeDenyLabelsFromMetric removes deny labels from a single metric
func (cl *CardinalityLimiter) removeDenyLabelsFromMetric(metric pmetric.Metric) {
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		dps := metric.Gauge().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			cl.removeDenyLabelsFromAttributes(dps.At(i).Attributes())
		}
	case pmetric.MetricTypeSum:
		dps := metric.Sum().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			cl.removeDenyLabelsFromAttributes(dps.At(i).Attributes())
		}
	case pmetric.MetricTypeHistogram:
		dps := metric.Histogram().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			cl.removeDenyLabelsFromAttributes(dps.At(i).Attributes())
		}
	case pmetric.MetricTypeSummary:
		dps := metric.Summary().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			cl.removeDenyLabelsFromAttributes(dps.At(i).Attributes())
		}
	case pmetric.MetricTypeExponentialHistogram:
		dps := metric.ExponentialHistogram().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			cl.removeDenyLabelsFromAttributes(dps.At(i).Attributes())
		}
	}
}

// removeDenyLabelsFromAttributes removes deny labels from attributes
func (cl *CardinalityLimiter) removeDenyLabelsFromAttributes(attrs pcommon.Map) {
	for _, label := range cl.config.DenyLabels {
		attrs.Remove(label)
	}
}

// trackAllDataPoints tracks all data points in a metric
func (cl *CardinalityLimiter) trackAllDataPoints(metric pmetric.Metric) {
	metricName := metric.Name()
	
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		dps := metric.Gauge().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			cl.tracker.TrackDataPoint(metricName, dps.At(i).Attributes())
			cl.tracker.IncrementStats("total")
		}
	case pmetric.MetricTypeSum:
		dps := metric.Sum().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			cl.tracker.TrackDataPoint(metricName, dps.At(i).Attributes())
			cl.tracker.IncrementStats("total")
		}
	case pmetric.MetricTypeHistogram:
		dps := metric.Histogram().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			cl.tracker.TrackDataPoint(metricName, dps.At(i).Attributes())
			cl.tracker.IncrementStats("total")
		}
	case pmetric.MetricTypeSummary:
		dps := metric.Summary().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			cl.tracker.TrackDataPoint(metricName, dps.At(i).Attributes())
			cl.tracker.IncrementStats("total")
		}
	case pmetric.MetricTypeExponentialHistogram:
		dps := metric.ExponentialHistogram().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			cl.tracker.TrackDataPoint(metricName, dps.At(i).Attributes())
			cl.tracker.IncrementStats("total")
		}
	}
}

// shouldAggregate checks if aggregation is needed based on cardinality
func (cl *CardinalityLimiter) shouldAggregate(metricName string, limit int) bool {
	currentCardinality := cl.tracker.GetCardinality(metricName)
	globalCardinality := cl.tracker.GetGlobalCardinality()
	return currentCardinality >= limit || globalCardinality >= cl.config.GlobalLimit
}

// Reset resets the limiter state
func (cl *CardinalityLimiter) Reset() {
	cl.tracker.Reset()
	
	cl.labelMutex.Lock()
	cl.labelCardinality = make(map[string]map[string]struct{})
	cl.labelMutex.Unlock()
	
	cl.alertMutex.Lock()
	cl.alertsSent = make(map[string]time.Time)
	cl.alertMutex.Unlock()
}