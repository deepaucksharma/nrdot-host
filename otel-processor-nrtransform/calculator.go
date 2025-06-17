package nrtransform

import (
	"fmt"
	"math"
	"sync"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// MetricCalculator handles metric calculations like rate and delta
type MetricCalculator struct {
	stateStore *StateStore
}

// NewMetricCalculator creates a new metric calculator
func NewMetricCalculator() *MetricCalculator {
	return &MetricCalculator{
		stateStore: NewStateStore(),
	}
}

// CalculateRate converts cumulative metrics to rate
func (mc *MetricCalculator) CalculateRate(metric pmetric.Metric, outputName string) (pmetric.Metric, error) {
	newMetric := pmetric.NewMetric()
	newMetric.SetName(outputName)
	newMetric.SetDescription(fmt.Sprintf("Rate of %s", metric.Description()))
	newMetric.SetUnit(metric.Unit() + "/s")

	switch metric.Type() {
	case pmetric.MetricTypeSum:
		if !metric.Sum().IsMonotonic() {
			return newMetric, fmt.Errorf("can only calculate rate for monotonic sums")
		}
		newMetric.SetEmptyGauge()
		if err := mc.calculateSumRate(metric.Sum(), newMetric.Gauge()); err != nil {
			return newMetric, err
		}

	default:
		return newMetric, fmt.Errorf("rate calculation only supported for cumulative sum metrics")
	}

	return newMetric, nil
}

// CalculateDelta converts cumulative metrics to delta
func (mc *MetricCalculator) CalculateDelta(metric pmetric.Metric, outputName string) (pmetric.Metric, error) {
	newMetric := pmetric.NewMetric()
	newMetric.SetName(outputName)
	newMetric.SetDescription(fmt.Sprintf("Delta of %s", metric.Description()))
	newMetric.SetUnit(metric.Unit())

	switch metric.Type() {
	case pmetric.MetricTypeSum:
		if !metric.Sum().IsMonotonic() {
			return newMetric, fmt.Errorf("can only calculate delta for monotonic sums")
		}
		newMetric.SetEmptySum()
		newMetric.Sum().SetIsMonotonic(false)
		newMetric.Sum().SetAggregationTemporality(pmetric.AggregationTemporalityDelta)
		if err := mc.calculateSumDelta(metric.Sum(), newMetric.Sum()); err != nil {
			return newMetric, err
		}

	default:
		return newMetric, fmt.Errorf("delta calculation only supported for cumulative sum metrics")
	}

	return newMetric, nil
}

func (mc *MetricCalculator) calculateSumRate(sum pmetric.Sum, gauge pmetric.Gauge) error {
	dataPoints := sum.DataPoints()
	for i := 0; i < dataPoints.Len(); i++ {
		dp := dataPoints.At(i)

		// Generate unique key for this data point
		key := mc.generateDataPointKey(sum.DataPoints().At(i))

		// Get previous state
		prevState := mc.stateStore.Get(key)
		if prevState == nil {
			// First observation, store state but don't produce rate
			mc.stateStore.Set(key, &DataPointState{
				Value:     dp.DoubleValue(),
				Timestamp: dp.Timestamp(),
			})
			continue
		}

		// Calculate rate
		timeDiff := float64(dp.Timestamp()-prevState.Timestamp) / float64(time.Second.Nanoseconds())
		if timeDiff <= 0 {
			continue
		}

		valueDiff := dp.DoubleValue() - prevState.Value
		if valueDiff < 0 {
			// Counter reset, skip this calculation
			mc.stateStore.Set(key, &DataPointState{
				Value:     dp.DoubleValue(),
				Timestamp: dp.Timestamp(),
			})
			continue
		}

		// Create new data point with rate
		newDp := gauge.DataPoints().AppendEmpty()
		dp.Attributes().CopyTo(newDp.Attributes())
		newDp.SetTimestamp(dp.Timestamp())
		
		rate := valueDiff / timeDiff
		newDp.SetDoubleValue(rate)

		// Update state
		mc.stateStore.Set(key, &DataPointState{
			Value:     dp.DoubleValue(),
			Timestamp: dp.Timestamp(),
		})
	}

	return nil
}

func (mc *MetricCalculator) calculateSumDelta(sum pmetric.Sum, newSum pmetric.Sum) error {
	dataPoints := sum.DataPoints()
	for i := 0; i < dataPoints.Len(); i++ {
		dp := dataPoints.At(i)
		
		// Generate unique key for this data point
		key := mc.generateDataPointKey(dp)

		// Get previous state
		prevState := mc.stateStore.Get(key)
		if prevState == nil {
			// First observation, store state but don't produce delta
			mc.stateStore.Set(key, &DataPointState{
				Value:     dp.DoubleValue(),
				Timestamp: dp.Timestamp(),
			})
			continue
		}

		// Calculate delta
		valueDiff := dp.DoubleValue() - prevState.Value
		if valueDiff < 0 {
			// Counter reset, use current value as delta
			valueDiff = dp.DoubleValue()
		}

		// Create new data point with delta value
		newDp := newSum.DataPoints().AppendEmpty()
		dp.Attributes().CopyTo(newDp.Attributes())
		newDp.SetTimestamp(dp.Timestamp())
		newDp.SetStartTimestamp(prevState.Timestamp)
		newDp.SetDoubleValue(valueDiff)

		// Update state
		mc.stateStore.Set(key, &DataPointState{
			Value:     dp.DoubleValue(),
			Timestamp: dp.Timestamp(),
		})
	}

	return nil
}

func (mc *MetricCalculator) generateDataPointKey(dp pmetric.NumberDataPoint) string {
	// Create a unique key based on attributes
	attrs := dp.Attributes()
	key := ""
	attrs.Range(func(k string, v pcommon.Value) bool {
		key += k + "=" + v.AsString() + ","
		return true
	})
	return key
}

// Aggregate performs aggregation operations on metrics
func (mc *MetricCalculator) Aggregate(metric pmetric.Metric, agg AggregationType, groupBy []string, outputName string) (pmetric.Metric, error) {
	newMetric := pmetric.NewMetric()
	newMetric.SetName(outputName)
	newMetric.SetDescription(fmt.Sprintf("%s of %s", agg, metric.Description()))
	newMetric.SetUnit(metric.Unit())

	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		newMetric.SetEmptyGauge()
		mc.aggregateGauge(metric.Gauge(), newMetric.Gauge(), agg, groupBy)
		return newMetric, nil

	case pmetric.MetricTypeSum:
		newMetric.SetEmptySum()
		newMetric.Sum().SetIsMonotonic(metric.Sum().IsMonotonic())
		newMetric.Sum().SetAggregationTemporality(metric.Sum().AggregationTemporality())
		mc.aggregateSum(metric.Sum(), newMetric.Sum(), agg, groupBy)
		return newMetric, nil

	default:
		return newMetric, fmt.Errorf("aggregation not supported for metric type: %s", metric.Type())
	}
}

func (mc *MetricCalculator) aggregateGauge(gauge pmetric.Gauge, newGauge pmetric.Gauge, agg AggregationType, groupBy []string) {
	groups := make(map[string]*AggregationGroup)

	dataPoints := gauge.DataPoints()
	for i := 0; i < dataPoints.Len(); i++ {
		dp := dataPoints.At(i)
		groupKey := mc.generateGroupKey(dp.Attributes(), groupBy)
		
		group, exists := groups[groupKey]
		if !exists {
			group = &AggregationGroup{
				attributes: mc.filterAttributes(dp.Attributes(), groupBy),
				values:     []float64{},
				timestamp:  dp.Timestamp(),
			}
			groups[groupKey] = group
		}
		
		group.values = append(group.values, dp.DoubleValue())
		if dp.Timestamp() > group.timestamp {
			group.timestamp = dp.Timestamp()
		}
	}

	// Create aggregated data points
	for _, group := range groups {
		newDp := newGauge.DataPoints().AppendEmpty()
		group.attributes.CopyTo(newDp.Attributes())
		newDp.SetTimestamp(group.timestamp)
		newDp.SetDoubleValue(mc.calculateAggregationValue(group.values, agg))
	}
}

func (mc *MetricCalculator) aggregateSum(sum pmetric.Sum, newSum pmetric.Sum, agg AggregationType, groupBy []string) {
	groups := make(map[string]*AggregationGroup)

	dataPoints := sum.DataPoints()
	for i := 0; i < dataPoints.Len(); i++ {
		dp := dataPoints.At(i)
		groupKey := mc.generateGroupKey(dp.Attributes(), groupBy)
		
		group, exists := groups[groupKey]
		if !exists {
			group = &AggregationGroup{
				attributes: mc.filterAttributes(dp.Attributes(), groupBy),
				values:     []float64{},
				timestamp:  dp.Timestamp(),
			}
			groups[groupKey] = group
		}
		
		group.values = append(group.values, dp.DoubleValue())
		if dp.Timestamp() > group.timestamp {
			group.timestamp = dp.Timestamp()
		}
	}

	// Create aggregated data points
	for _, group := range groups {
		newDp := newSum.DataPoints().AppendEmpty()
		group.attributes.CopyTo(newDp.Attributes())
		newDp.SetTimestamp(group.timestamp)
		newDp.SetDoubleValue(mc.calculateAggregationValue(group.values, agg))
	}
}

func (mc *MetricCalculator) generateGroupKey(attrs pcommon.Map, groupBy []string) string {
	if len(groupBy) == 0 {
		return "all"
	}

	key := ""
	for _, attr := range groupBy {
		if val, ok := attrs.Get(attr); ok {
			key += attr + "=" + val.AsString() + ","
		}
	}
	return key
}

func (mc *MetricCalculator) filterAttributes(attrs pcommon.Map, keepAttrs []string) pcommon.Map {
	newAttrs := pcommon.NewMap()
	for _, attr := range keepAttrs {
		if val, ok := attrs.Get(attr); ok {
			val.CopyTo(newAttrs.PutEmpty(attr))
		}
	}
	return newAttrs
}

func (mc *MetricCalculator) calculateAggregationValue(values []float64, agg AggregationType) float64 {
	if len(values) == 0 {
		return 0
	}

	switch agg {
	case AggregationSum:
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		return sum

	case AggregationAvg:
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		return sum / float64(len(values))

	case AggregationMin:
		min := values[0]
		for _, v := range values[1:] {
			if v < min {
				min = v
			}
		}
		return min

	case AggregationMax:
		max := values[0]
		for _, v := range values[1:] {
			if v > max {
				max = v
			}
		}
		return max

	case AggregationCount:
		return float64(len(values))

	default:
		return 0
	}
}

// ConvertUnit converts metric values between units
func (mc *MetricCalculator) ConvertUnit(metric pmetric.Metric, fromUnit, toUnit, outputName string) (pmetric.Metric, error) {
	conversionFactor, err := getConversionFactor(fromUnit, toUnit)
	if err != nil {
		return pmetric.Metric{}, err
	}

	newMetric := pmetric.NewMetric()
	newMetric.SetName(outputName)
	newMetric.SetDescription(metric.Description())
	newMetric.SetUnit(toUnit)

	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		newMetric.SetEmptyGauge()
		mc.convertGaugeUnit(metric.Gauge(), newMetric.Gauge(), conversionFactor)

	case pmetric.MetricTypeSum:
		newMetric.SetEmptySum()
		newMetric.Sum().SetIsMonotonic(metric.Sum().IsMonotonic())
		newMetric.Sum().SetAggregationTemporality(metric.Sum().AggregationTemporality())
		mc.convertSumUnit(metric.Sum(), newMetric.Sum(), conversionFactor)

	case pmetric.MetricTypeHistogram:
		newMetric.SetEmptyHistogram()
		newMetric.Histogram().SetAggregationTemporality(metric.Histogram().AggregationTemporality())
		mc.convertHistogramUnit(metric.Histogram(), newMetric.Histogram(), conversionFactor)

	default:
		return newMetric, fmt.Errorf("unit conversion not supported for metric type: %s", metric.Type())
	}

	return newMetric, nil
}

func (mc *MetricCalculator) convertGaugeUnit(gauge pmetric.Gauge, newGauge pmetric.Gauge, factor float64) {
	dataPoints := gauge.DataPoints()
	for i := 0; i < dataPoints.Len(); i++ {
		dp := dataPoints.At(i)
		newDp := newGauge.DataPoints().AppendEmpty()
		dp.CopyTo(newDp)
		newDp.SetDoubleValue(dp.DoubleValue() * factor)
	}
}

func (mc *MetricCalculator) convertSumUnit(sum pmetric.Sum, newSum pmetric.Sum, factor float64) {
	dataPoints := sum.DataPoints()
	for i := 0; i < dataPoints.Len(); i++ {
		dp := dataPoints.At(i)
		newDp := newSum.DataPoints().AppendEmpty()
		dp.CopyTo(newDp)
		newDp.SetDoubleValue(dp.DoubleValue() * factor)
	}
}

func (mc *MetricCalculator) convertHistogramUnit(hist pmetric.Histogram, newHist pmetric.Histogram, factor float64) {
	dataPoints := hist.DataPoints()
	for i := 0; i < dataPoints.Len(); i++ {
		dp := dataPoints.At(i)
		newDp := newHist.DataPoints().AppendEmpty()
		
		// Copy attributes and timestamps
		dp.Attributes().CopyTo(newDp.Attributes())
		newDp.SetTimestamp(dp.Timestamp())
		newDp.SetStartTimestamp(dp.StartTimestamp())
		newDp.SetCount(dp.Count())
		
		// Convert sum
		newDp.SetSum(dp.Sum() * factor)
		
		// Convert min/max if present
		if dp.HasMin() {
			newDp.SetMin(dp.Min() * factor)
		}
		if dp.HasMax() {
			newDp.SetMax(dp.Max() * factor)
		}
		
		// Copy bucket counts
		bucketCounts := dp.BucketCounts()
		newBucketCounts := newDp.BucketCounts()
		newBucketCounts.EnsureCapacity(bucketCounts.Len())
		for j := 0; j < bucketCounts.Len(); j++ {
			newBucketCounts.Append(bucketCounts.At(j))
		}
		
		// Convert bucket bounds
		bounds := dp.ExplicitBounds()
		newBounds := newDp.ExplicitBounds()
		newBounds.EnsureCapacity(bounds.Len())
		for j := 0; j < bounds.Len(); j++ {
			newBounds.Append(bounds.At(j) * factor)
		}
		
		// Copy exemplars if any
		exemplars := dp.Exemplars()
		newExemplars := newDp.Exemplars()
		for j := 0; j < exemplars.Len(); j++ {
			exemplar := exemplars.At(j)
			newExemplar := newExemplars.AppendEmpty()
			exemplar.CopyTo(newExemplar)
			// Convert exemplar value
			switch newExemplar.ValueType() {
			case pmetric.ExemplarValueTypeDouble:
				newExemplar.SetDoubleValue(newExemplar.DoubleValue() * factor)
			case pmetric.ExemplarValueTypeInt:
				newExemplar.SetIntValue(int64(float64(newExemplar.IntValue()) * factor))
			}
		}
	}
}

func getConversionFactor(fromUnit, toUnit string) (float64, error) {
	conversions := map[string]map[string]float64{
		"bytes": {
			"kilobytes": 1.0 / 1024,
			"megabytes": 1.0 / (1024 * 1024),
			"gigabytes": 1.0 / (1024 * 1024 * 1024),
			"kb":        1.0 / 1024,
			"mb":        1.0 / (1024 * 1024),
			"gb":        1.0 / (1024 * 1024 * 1024),
		},
		"milliseconds": {
			"seconds": 0.001,
			"minutes": 0.001 / 60,
			"hours":   0.001 / 3600,
			"ms":      1.0,
			"s":       0.001,
			"m":       0.001 / 60,
			"h":       0.001 / 3600,
		},
		"seconds": {
			"milliseconds": 1000,
			"minutes":      1.0 / 60,
			"hours":        1.0 / 3600,
			"ms":           1000,
			"m":            1.0 / 60,
			"h":            1.0 / 3600,
		},
		"percent": {
			"ratio": 0.01,
		},
		"ratio": {
			"percent": 100,
		},
	}

	if fromConversions, ok := conversions[fromUnit]; ok {
		if factor, ok := fromConversions[toUnit]; ok {
			return factor, nil
		}
	}

	// Try reverse conversion
	if toConversions, ok := conversions[toUnit]; ok {
		if factor, ok := toConversions[fromUnit]; ok {
			return 1.0 / factor, nil
		}
	}

	return 0, fmt.Errorf("unsupported unit conversion: %s to %s", fromUnit, toUnit)
}

// StateStore manages state for rate and delta calculations
type StateStore struct {
	mu    sync.RWMutex
	store map[string]*DataPointState
}

// DataPointState stores the state of a data point
type DataPointState struct {
	Value     float64
	Timestamp pcommon.Timestamp
}

// NewStateStore creates a new state store
func NewStateStore() *StateStore {
	return &StateStore{
		store: make(map[string]*DataPointState),
	}
}

// Get retrieves state for a key
func (ss *StateStore) Get(key string) *DataPointState {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return ss.store[key]
}

// Set stores state for a key
func (ss *StateStore) Set(key string, state *DataPointState) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.store[key] = state
}

// AggregationGroup represents a group of values to aggregate
type AggregationGroup struct {
	attributes pcommon.Map
	values     []float64
	timestamp  pcommon.Timestamp
}

// CalculatePercentile calculates the percentile value from a sorted slice
func CalculatePercentile(values []float64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}

	index := percentile / 100 * float64(len(values)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return values[lower]
	}

	// Linear interpolation
	weight := index - float64(lower)
	return values[lower]*(1-weight) + values[upper]*weight
}