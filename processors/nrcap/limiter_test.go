package nrcap

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

func TestNewCardinalityLimiter(t *testing.T) {
	cfg := &Config{
		GlobalLimit:   100,
		DefaultLimit:  10,
		Strategy:      StrategyDrop,
		WindowSize:    5 * time.Minute,
		ResetInterval: time.Hour,
	}
	logger := zap.NewNop()

	limiter := NewCardinalityLimiter(cfg, logger)
	assert.NotNil(t, limiter)
	assert.Equal(t, cfg, limiter.config)
	assert.Equal(t, logger, limiter.logger)
	assert.NotNil(t, limiter.tracker)
	assert.NotNil(t, limiter.labelCardinality)
	assert.NotNil(t, limiter.rand)
}

func TestProcessMetricsDropStrategy(t *testing.T) {
	cfg := &Config{
		GlobalLimit:   100,
		DefaultLimit:  3,
		Strategy:      StrategyDrop,
		WindowSize:    5 * time.Minute,
		ResetInterval: time.Hour,
	}
	logger := zap.NewNop()
	limiter := NewCardinalityLimiter(cfg, logger)

	// Create metrics with cardinality of 5
	metrics := generateMetricsWithLabels("test_metric", []map[string]string{
		{"label": "a"},
		{"label": "b"},
		{"label": "c"},
		{"label": "d"}, // This should be dropped
		{"label": "e"}, // This should be dropped
	})

	result, err := limiter.ProcessMetrics(metrics)
	require.NoError(t, err)

	// Count resulting data points
	count := countDataPoints(result)
	assert.Equal(t, 3, count)

	stats := limiter.GetStats()
	assert.Equal(t, int64(2), stats.DroppedMetrics)
}

func TestProcessMetricsAggregateStrategy(t *testing.T) {
	cfg := &Config{
		GlobalLimit:       100,
		DefaultLimit:      2,
		Strategy:          StrategyAggregate,
		WindowSize:        5 * time.Minute,
		ResetInterval:     time.Hour,
		AggregationLabels: []string{"service"},
	}
	logger := zap.NewNop()
	limiter := NewCardinalityLimiter(cfg, logger)

	// Create metrics with high cardinality labels
	metrics := generateMetricsWithLabels("test_metric", []map[string]string{
		{"service": "api", "trace_id": "123", "session": "abc"},
		{"service": "api", "trace_id": "456", "session": "def"},
		{"service": "web", "trace_id": "789", "session": "ghi"},
	})

	result, err := limiter.ProcessMetrics(metrics)
	require.NoError(t, err)

	// All metrics should be present but aggregated
	count := countDataPoints(result)
	assert.Equal(t, 3, count)

	// Check that only service label remains
	rm := result.ResourceMetrics().At(0)
	sm := rm.ScopeMetrics().At(0)
	metric := sm.Metrics().At(0)
	
	for i := 0; i < metric.Gauge().DataPoints().Len(); i++ {
		dp := metric.Gauge().DataPoints().At(i)
		attrs := dp.Attributes()
		assert.Equal(t, 1, attrs.Len())
		_, exists := attrs.Get("service")
		assert.True(t, exists)
	}
}

func TestProcessMetricsSampleStrategy(t *testing.T) {
	cfg := &Config{
		GlobalLimit:   100,
		DefaultLimit:  1,
		Strategy:      StrategySample,
		SampleRate:    0.5,
		WindowSize:    5 * time.Minute,
		ResetInterval: time.Hour,
	}
	logger := zap.NewNop()
	limiter := NewCardinalityLimiter(cfg, logger)

	// Create many metrics
	var labels []map[string]string
	for i := 0; i < 100; i++ {
		labels = append(labels, map[string]string{
			"label": string(rune('a' + i%26)),
			"index": string(rune('0' + i)),
		})
	}
	metrics := generateMetricsWithLabels("test_metric", labels)

	result, err := limiter.ProcessMetrics(metrics)
	require.NoError(t, err)

	// With 50% sample rate, we expect roughly half (but could vary)
	count := countDataPoints(result)
	assert.Greater(t, count, 0)
	assert.Less(t, count, 100)

	stats := limiter.GetStats()
	assert.Greater(t, stats.SampledMetrics, int64(0))
}

func TestProcessMetricsOldestStrategy(t *testing.T) {
	cfg := &Config{
		GlobalLimit:   100,
		DefaultLimit:  3,
		Strategy:      StrategyOldest,
		WindowSize:    5 * time.Minute,
		ResetInterval: time.Hour,
	}
	logger := zap.NewNop()
	limiter := NewCardinalityLimiter(cfg, logger)

	// Process metrics in sequence
	for i := 0; i < 5; i++ {
		metrics := generateMetricsWithLabels("test_metric", []map[string]string{
			{"label": string(rune('a' + i))},
		})
		
		// Add small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
		
		_, err := limiter.ProcessMetrics(metrics)
		require.NoError(t, err)
	}

	// Should have kept the 3 most recent
	cardinality := limiter.tracker.GetCardinality("test_metric")
	assert.Equal(t, 3, cardinality)
}

func TestProcessMetricsDenyLabels(t *testing.T) {
	cfg := &Config{
		GlobalLimit:   100,
		DefaultLimit:  10,
		Strategy:      StrategyAggregate,
		DenyLabels:    []string{"trace_id", "session_id"},
		WindowSize:    5 * time.Minute,
		ResetInterval: time.Hour,
	}
	logger := zap.NewNop()
	limiter := NewCardinalityLimiter(cfg, logger)

	metrics := generateMetricsWithLabels("test_metric", []map[string]string{
		{"service": "api", "trace_id": "123", "session_id": "abc"},
		{"service": "web", "trace_id": "456", "session_id": "def"},
	})

	result, err := limiter.ProcessMetrics(metrics)
	require.NoError(t, err)

	// Check that denied labels are removed
	rm := result.ResourceMetrics().At(0)
	sm := rm.ScopeMetrics().At(0)
	metric := sm.Metrics().At(0)
	
	for i := 0; i < metric.Gauge().DataPoints().Len(); i++ {
		dp := metric.Gauge().DataPoints().At(i)
		attrs := dp.Attributes()
		
		_, exists := attrs.Get("trace_id")
		assert.False(t, exists)
		
		_, exists = attrs.Get("session_id")
		assert.False(t, exists)
		
		_, exists = attrs.Get("service")
		assert.True(t, exists)
	}
}

func TestProcessMetricsGlobalLimit(t *testing.T) {
	cfg := &Config{
		GlobalLimit:   5,
		DefaultLimit:  10,
		Strategy:      StrategyDrop,
		WindowSize:    5 * time.Minute,
		ResetInterval: time.Hour,
	}
	logger := zap.NewNop()
	limiter := NewCardinalityLimiter(cfg, logger)

	// Create multiple metrics that together exceed global limit
	metrics1 := generateMetrics("metric1", 3)
	metrics2 := generateMetrics("metric2", 3)

	result1, err := limiter.ProcessMetrics(metrics1)
	require.NoError(t, err)
	assert.Equal(t, 3, countDataPoints(result1))

	result2, err := limiter.ProcessMetrics(metrics2)
	require.NoError(t, err)
	assert.Equal(t, 2, countDataPoints(result2)) // Only 2 because global limit is 5

	stats := limiter.GetStats()
	assert.Equal(t, int64(1), stats.DroppedMetrics)
}

func TestLabelCardinalityTracking(t *testing.T) {
	cfg := &Config{
		GlobalLimit:   100,
		DefaultLimit:  10,
		Strategy:      StrategyDrop,
		WindowSize:    5 * time.Minute,
		ResetInterval: time.Hour,
	}
	logger := zap.NewNop()
	limiter := NewCardinalityLimiter(cfg, logger)

	// Create metrics with various label values
	var labels []map[string]string
	for i := 0; i < 5; i++ {
		labels = append(labels, map[string]string{
			"service":    "api",
			"request_id": string(rune('a' + i)),
		})
	}
	metrics := generateMetricsWithLabels("test_metric", labels)

	_, err := limiter.ProcessMetrics(metrics)
	require.NoError(t, err)

	// Check label cardinality tracking
	stats := limiter.GetStats()
	assert.Equal(t, 1, stats.HighCardinalityLabels["service"])    // Only one value
	assert.Equal(t, 5, stats.HighCardinalityLabels["request_id"]) // Five different values
}

func TestReset(t *testing.T) {
	cfg := &Config{
		GlobalLimit:   100,
		DefaultLimit:  10,
		Strategy:      StrategyDrop,
		WindowSize:    5 * time.Minute,
		ResetInterval: time.Hour,
	}
	logger := zap.NewNop()
	limiter := NewCardinalityLimiter(cfg, logger)

	// Add some metrics
	metrics := generateMetrics("test_metric", 5)
	_, err := limiter.ProcessMetrics(metrics)
	require.NoError(t, err)

	// Verify data exists
	assert.Greater(t, limiter.tracker.GetGlobalCardinality(), 0)

	// Reset
	limiter.Reset()

	// Verify data is cleared
	assert.Equal(t, 0, limiter.tracker.GetGlobalCardinality())
	assert.Equal(t, 0, len(limiter.labelCardinality))
}

// Helper function to generate metrics with specific labels
func generateMetricsWithLabels(name string, labels []map[string]string) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	metric := sm.Metrics().AppendEmpty()
	metric.SetName(name)
	metric.SetEmptyGauge()

	for i, labelMap := range labels {
		dp := metric.Gauge().DataPoints().AppendEmpty()
		dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		dp.SetDoubleValue(float64(i))
		
		for k, v := range labelMap {
			dp.Attributes().PutStr(k, v)
		}
	}

	return metrics
}

// Helper function to count data points in metrics
func countDataPoints(metrics pmetric.Metrics) int {
	count := 0
	rm := metrics.ResourceMetrics()
	for i := 0; i < rm.Len(); i++ {
		sm := rm.At(i).ScopeMetrics()
		for j := 0; j < sm.Len(); j++ {
			m := sm.At(j).Metrics()
			for k := 0; k < m.Len(); k++ {
				metric := m.At(k)
				switch metric.Type() {
				case pmetric.MetricTypeGauge:
					count += metric.Gauge().DataPoints().Len()
				case pmetric.MetricTypeSum:
					count += metric.Sum().DataPoints().Len()
				case pmetric.MetricTypeHistogram:
					count += metric.Histogram().DataPoints().Len()
				case pmetric.MetricTypeSummary:
					count += metric.Summary().DataPoints().Len()
				case pmetric.MetricTypeExponentialHistogram:
					count += metric.ExponentialHistogram().DataPoints().Len()
				}
			}
		}
	}
	return count
}