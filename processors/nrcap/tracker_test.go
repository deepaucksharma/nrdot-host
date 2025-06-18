package nrcap

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func TestNewCardinalityTracker(t *testing.T) {
	windowSize := 5 * time.Minute
	tracker := NewCardinalityTracker(windowSize)

	assert.NotNil(t, tracker)
	assert.Equal(t, windowSize, tracker.windowSize)
	assert.NotNil(t, tracker.metrics)
	assert.NotNil(t, tracker.metricCounts)
	assert.Equal(t, 0, tracker.globalCount)
	assert.NotNil(t, tracker.stats.MetricCardinalities)
	assert.NotNil(t, tracker.stats.HighCardinalityLabels)
}

func TestTrack(t *testing.T) {
	tracker := NewCardinalityTracker(5 * time.Minute)

	// Create a metric
	metric := createTestMetric("test_metric", map[string]string{
		"label1": "value1",
		"label2": "value2",
	})

	// First track should return true (new)
	isNew, hash := tracker.Track(metric)
	assert.True(t, isNew)
	assert.NotEqual(t, uint64(0), hash)
	assert.Equal(t, 1, tracker.GetCardinality("test_metric"))
	assert.Equal(t, 1, tracker.GetGlobalCardinality())

	// Second track with same labels should return false (existing)
	isNew2, hash2 := tracker.Track(metric)
	assert.False(t, isNew2)
	assert.Equal(t, hash, hash2)
	assert.Equal(t, 1, tracker.GetCardinality("test_metric"))
	assert.Equal(t, 1, tracker.GetGlobalCardinality())

	// Track with different labels should return true
	metric2 := createTestMetric("test_metric", map[string]string{
		"label1": "value1",
		"label2": "value3", // Different value
	})
	isNew3, hash3 := tracker.Track(metric2)
	assert.True(t, isNew3)
	assert.NotEqual(t, hash, hash3)
	assert.Equal(t, 2, tracker.GetCardinality("test_metric"))
	assert.Equal(t, 2, tracker.GetGlobalCardinality())
}

func TestTrackMultipleMetrics(t *testing.T) {
	tracker := NewCardinalityTracker(5 * time.Minute)

	// Track different metrics
	metric1 := createTestMetric("metric1", map[string]string{"env": "prod"})
	metric2 := createTestMetric("metric2", map[string]string{"env": "dev"})

	isNew1, _ := tracker.Track(metric1)
	assert.True(t, isNew1)

	isNew2, _ := tracker.Track(metric2)
	assert.True(t, isNew2)

	assert.Equal(t, 1, tracker.GetCardinality("metric1"))
	assert.Equal(t, 1, tracker.GetCardinality("metric2"))
	assert.Equal(t, 2, tracker.GetGlobalCardinality())
}

func TestCleanupOldEntries(t *testing.T) {
	windowSize := 100 * time.Millisecond
	tracker := NewCardinalityTracker(windowSize)

	// Track a metric
	metric := createTestMetric("test_metric", map[string]string{"label": "value"})
	tracker.Track(metric)
	assert.Equal(t, 1, tracker.GetCardinality("test_metric"))

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Cleanup should remove the old entry
	tracker.CleanupOldEntries()
	assert.Equal(t, 0, tracker.GetCardinality("test_metric"))
	assert.Equal(t, 0, tracker.GetGlobalCardinality())
}

func TestTrackerReset(t *testing.T) {
	tracker := NewCardinalityTracker(5 * time.Minute)

	// Add some metrics
	for i := 0; i < 5; i++ {
		metric := createTestMetric("test_metric", map[string]string{
			"label": string(rune('a' + i)),
		})
		tracker.Track(metric)
	}

	assert.Equal(t, 5, tracker.GetCardinality("test_metric"))
	assert.Equal(t, 5, tracker.GetGlobalCardinality())

	// Reset
	tracker.Reset()

	assert.Equal(t, 0, tracker.GetCardinality("test_metric"))
	assert.Equal(t, 0, tracker.GetGlobalCardinality())
	assert.NotNil(t, tracker.metrics)
	assert.NotNil(t, tracker.metricCounts)
}

func TestGetStats(t *testing.T) {
	tracker := NewCardinalityTracker(5 * time.Minute)

	// Track some metrics
	metric := createTestMetric("test_metric", map[string]string{"label": "value"})
	tracker.Track(metric)
	tracker.IncrementStats("total")
	tracker.IncrementStats("dropped")
	tracker.TrackLabelCardinality("label", 10)

	stats := tracker.GetStats()
	assert.Equal(t, int64(1), stats.TotalMetrics)
	assert.Equal(t, int64(1), stats.DroppedMetrics)
	assert.Equal(t, 10, stats.HighCardinalityLabels["label"])
	assert.NotZero(t, stats.LastReset)
}

func TestIncrementStats(t *testing.T) {
	tracker := NewCardinalityTracker(5 * time.Minute)

	tracker.IncrementStats("total")
	tracker.IncrementStats("total")
	tracker.IncrementStats("dropped")
	tracker.IncrementStats("aggregated")
	tracker.IncrementStats("sampled")

	stats := tracker.GetStats()
	assert.Equal(t, int64(2), stats.TotalMetrics)
	assert.Equal(t, int64(1), stats.DroppedMetrics)
	assert.Equal(t, int64(1), stats.AggregatedMetrics)
	assert.Equal(t, int64(1), stats.SampledMetrics)
}

func TestHashLabels(t *testing.T) {
	tracker := NewCardinalityTracker(5 * time.Minute)

	// Same labels should produce same hash
	metric1 := createTestMetric("test_metric", map[string]string{
		"label1": "value1",
		"label2": "value2",
	})
	metric2 := createTestMetric("test_metric", map[string]string{
		"label1": "value1",
		"label2": "value2",
	})

	hash1 := tracker.hashLabels(metric1)
	hash2 := tracker.hashLabels(metric2)
	assert.Equal(t, hash1, hash2)

	// Different labels should produce different hash
	metric3 := createTestMetric("test_metric", map[string]string{
		"label1": "value1",
		"label2": "value3", // Different
	})
	hash3 := tracker.hashLabels(metric3)
	assert.NotEqual(t, hash1, hash3)

	// Different metric names should produce different hash
	metric4 := createTestMetric("other_metric", map[string]string{
		"label1": "value1",
		"label2": "value2",
	})
	hash4 := tracker.hashLabels(metric4)
	assert.NotEqual(t, hash1, hash4)
}

func TestGetOldestEntries(t *testing.T) {
	tracker := NewCardinalityTracker(5 * time.Minute)

	// Track metrics with delays to ensure different timestamps
	for i := 0; i < 5; i++ {
		metric := createTestMetric("test_metric", map[string]string{
			"label": string(rune('a' + i)),
		})
		tracker.Track(metric)
		time.Sleep(10 * time.Millisecond)
	}

	// Get oldest 3 entries
	oldest := tracker.GetOldestEntries("test_metric", 3)
	assert.Len(t, oldest, 3)

	// Remove one of the oldest
	tracker.RemoveEntry("test_metric", oldest[0])
	assert.Equal(t, 4, tracker.GetCardinality("test_metric"))
}

func TestRemoveEntry(t *testing.T) {
	tracker := NewCardinalityTracker(5 * time.Minute)

	// Track a metric
	metric := createTestMetric("test_metric", map[string]string{"label": "value"})
	isNew, hash := tracker.Track(metric)
	assert.True(t, isNew)
	assert.Equal(t, 1, tracker.GetCardinality("test_metric"))

	// Remove the entry
	tracker.RemoveEntry("test_metric", hash)
	assert.Equal(t, 0, tracker.GetCardinality("test_metric"))
	assert.Equal(t, 0, tracker.GetGlobalCardinality())

	// Remove non-existent entry (should not panic)
	tracker.RemoveEntry("test_metric", hash)
	tracker.RemoveEntry("non_existent", 12345)
}

func TestDifferentMetricTypes(t *testing.T) {
	// Test with different metric types
	tests := []struct {
		name       string
		metricType pmetric.MetricType
		setupFunc  func(pmetric.Metric)
	}{
		{
			name:       "gauge",
			metricType: pmetric.MetricTypeGauge,
			setupFunc: func(m pmetric.Metric) {
				m.SetEmptyGauge()
				dp := m.Gauge().DataPoints().AppendEmpty()
				dp.Attributes().PutStr("test", "value")
			},
		},
		{
			name:       "sum",
			metricType: pmetric.MetricTypeSum,
			setupFunc: func(m pmetric.Metric) {
				m.SetEmptySum()
				dp := m.Sum().DataPoints().AppendEmpty()
				dp.Attributes().PutStr("test", "value")
			},
		},
		{
			name:       "histogram",
			metricType: pmetric.MetricTypeHistogram,
			setupFunc: func(m pmetric.Metric) {
				m.SetEmptyHistogram()
				dp := m.Histogram().DataPoints().AppendEmpty()
				dp.Attributes().PutStr("test", "value")
			},
		},
		{
			name:       "summary",
			metricType: pmetric.MetricTypeSummary,
			setupFunc: func(m pmetric.Metric) {
				m.SetEmptySummary()
				dp := m.Summary().DataPoints().AppendEmpty()
				dp.Attributes().PutStr("test", "value")
			},
		},
		{
			name:       "exponential_histogram",
			metricType: pmetric.MetricTypeExponentialHistogram,
			setupFunc: func(m pmetric.Metric) {
				m.SetEmptyExponentialHistogram()
				dp := m.ExponentialHistogram().DataPoints().AppendEmpty()
				dp.Attributes().PutStr("test", "value")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new tracker for each test to ensure isolation
			tracker := NewCardinalityTracker(5 * time.Minute)
			
			metrics := pmetric.NewMetrics()
			rm := metrics.ResourceMetrics().AppendEmpty()
			sm := rm.ScopeMetrics().AppendEmpty()
			metric := sm.Metrics().AppendEmpty()
			metric.SetName("test_metric")
			tt.setupFunc(metric)

			isNew, hash := tracker.Track(metric)
			assert.True(t, isNew)
			assert.NotEqual(t, uint64(0), hash)
		})
	}
}

// Helper function to create a test metric
func createTestMetric(name string, labels map[string]string) pmetric.Metric {
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	metric := sm.Metrics().AppendEmpty()
	metric.SetName(name)
	metric.SetEmptyGauge()

	dp := metric.Gauge().DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	dp.SetDoubleValue(1.0)

	for k, v := range labels {
		dp.Attributes().PutStr(k, v)
	}

	return metric
}