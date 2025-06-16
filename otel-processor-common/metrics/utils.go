package metrics

import (
	"fmt"
	"math"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// RateCalculator calculates rates for counter metrics
type RateCalculator struct {
	cache map[string]*metricState
}

type metricState struct {
	lastValue     float64
	lastTimestamp time.Time
}

// NewRateCalculator creates a new rate calculator
func NewRateCalculator() *RateCalculator {
	return &RateCalculator{
		cache: make(map[string]*metricState),
	}
}

// CalculateRate computes the rate of change for a metric
func (r *RateCalculator) CalculateRate(metric pmetric.Metric, dp pmetric.NumberDataPoint) (float64, bool) {
	// Generate unique key for the metric
	key := generateMetricKey(metric, dp.Attributes())
	
	value := dp.DoubleValue()
	timestamp := dp.Timestamp().AsTime()
	
	state, exists := r.cache[key]
	if !exists {
		// First observation, store it
		r.cache[key] = &metricState{
			lastValue:     value,
			lastTimestamp: timestamp,
		}
		return 0, false
	}
	
	// Calculate rate
	timeDiff := timestamp.Sub(state.lastTimestamp).Seconds()
	if timeDiff <= 0 {
		return 0, false
	}
	
	valueDiff := value - state.lastValue
	
	// Handle counter resets
	if valueDiff < 0 {
		valueDiff = value
	}
	
	rate := valueDiff / timeDiff
	
	// Update cache
	state.lastValue = value
	state.lastTimestamp = timestamp
	
	return rate, true
}

// generateMetricKey creates a unique identifier for a metric
func generateMetricKey(metric pmetric.Metric, attrs pcommon.Map) string {
	key := fmt.Sprintf("%s:", metric.Name())
	
	// Sort attributes for consistent key generation
	attrs.Range(func(k string, v pcommon.Value) bool {
		key += fmt.Sprintf("%s=%s,", k, v.AsString())
		return true
	})
	
	return key
}

// CardinalityLimiter helps prevent cardinality explosion
type CardinalityLimiter struct {
	maxCardinality int
	metricCounts   map[string]int
}

// NewCardinalityLimiter creates a new cardinality limiter
func NewCardinalityLimiter(maxCardinality int) *CardinalityLimiter {
	return &CardinalityLimiter{
		maxCardinality: maxCardinality,
		metricCounts:   make(map[string]int),
	}
}

// ShouldKeep determines if a metric should be kept based on cardinality
func (c *CardinalityLimiter) ShouldKeep(metric pmetric.Metric, attrs pcommon.Map) bool {
	metricName := metric.Name()
	
	// Count unique attribute combinations
	key := generateMetricKey(metric, attrs)
	
	count, exists := c.metricCounts[metricName]
	if !exists {
		c.metricCounts[metricName] = 1
		return true
	}
	
	if count >= c.maxCardinality {
		return false
	}
	
	c.metricCounts[metricName]++
	return true
}

// MetricAggregator provides metric aggregation utilities
type MetricAggregator struct {
	aggregations map[string]*aggregationState
}

type aggregationState struct {
	sum   float64
	count int64
	min   float64
	max   float64
}

// NewMetricAggregator creates a new metric aggregator
func NewMetricAggregator() *MetricAggregator {
	return &MetricAggregator{
		aggregations: make(map[string]*aggregationState),
	}
}

// AddValue adds a value to the aggregation
func (a *MetricAggregator) AddValue(key string, value float64) {
	state, exists := a.aggregations[key]
	if !exists {
		state = &aggregationState{
			sum:   value,
			count: 1,
			min:   value,
			max:   value,
		}
		a.aggregations[key] = state
		return
	}
	
	state.sum += value
	state.count++
	state.min = math.Min(state.min, value)
	state.max = math.Max(state.max, value)
}

// GetAggregations returns the current aggregations
func (a *MetricAggregator) GetAggregations() map[string]AggregationResult {
	results := make(map[string]AggregationResult)
	
	for key, state := range a.aggregations {
		results[key] = AggregationResult{
			Sum:   state.sum,
			Count: state.count,
			Min:   state.min,
			Max:   state.max,
			Avg:   state.sum / float64(state.count),
		}
	}
	
	return results
}

// AggregationResult contains aggregated metric values
type AggregationResult struct {
	Sum   float64
	Count int64
	Min   float64
	Max   float64
	Avg   float64
}

// FilterMetrics filters metrics based on a predicate function
func FilterMetrics(metrics pmetric.Metrics, predicate func(pmetric.Metric) bool) pmetric.Metrics {
	filtered := pmetric.NewMetrics()
	
	resourceMetrics := metrics.ResourceMetrics()
	for i := 0; i < resourceMetrics.Len(); i++ {
		rm := resourceMetrics.At(i)
		
		newRM := filtered.ResourceMetrics().AppendEmpty()
		rm.Resource().CopyTo(newRM.Resource())
		
		scopeMetrics := rm.ScopeMetrics()
		for j := 0; j < scopeMetrics.Len(); j++ {
			sm := scopeMetrics.At(j)
			
			newSM := newRM.ScopeMetrics().AppendEmpty()
			sm.Scope().CopyTo(newSM.Scope())
			
			metrics := sm.Metrics()
			for k := 0; k < metrics.Len(); k++ {
				metric := metrics.At(k)
				if predicate(metric) {
					newMetric := newSM.Metrics().AppendEmpty()
					metric.CopyTo(newMetric)
				}
			}
			
			// Remove empty scope metrics
			if newSM.Metrics().Len() == 0 {
				newRM.ScopeMetrics().RemoveIf(func(_ pmetric.ScopeMetrics) bool {
					return true
				})
			}
		}
		
		// Remove empty resource metrics
		if newRM.ScopeMetrics().Len() == 0 {
			filtered.ResourceMetrics().RemoveIf(func(_ pmetric.ResourceMetrics) bool {
				return true
			})
		}
	}
	
	return filtered
}