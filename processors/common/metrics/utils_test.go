package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func TestRateCalculator(t *testing.T) {
	calc := NewRateCalculator()
	
	// Create test metric
	metric := pmetric.NewMetric()
	metric.SetName("test.counter")
	
	attrs := pcommon.NewMap()
	attrs.PutStr("host", "server1")
	
	t.Run("first observation returns no rate", func(t *testing.T) {
		dp := pmetric.NewNumberDataPoint()
		dp.SetDoubleValue(100)
		dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		attrs.CopyTo(dp.Attributes())
		
		rate, ok := calc.CalculateRate(metric, dp)
		assert.False(t, ok)
		assert.Equal(t, float64(0), rate)
	})
	
	t.Run("second observation calculates rate", func(t *testing.T) {
		// Wait a bit to ensure time difference
		time.Sleep(100 * time.Millisecond)
		
		dp := pmetric.NewNumberDataPoint()
		dp.SetDoubleValue(150)
		dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		attrs.CopyTo(dp.Attributes())
		
		rate, ok := calc.CalculateRate(metric, dp)
		assert.True(t, ok)
		assert.Greater(t, rate, float64(0))
		// Rate should be approximately 500/sec (50 increase over 0.1 seconds)
		assert.InDelta(t, 500, rate, 100)
	})
	
	t.Run("handles counter reset", func(t *testing.T) {
		time.Sleep(100 * time.Millisecond)
		
		dp := pmetric.NewNumberDataPoint()
		dp.SetDoubleValue(25) // Reset to lower value
		dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		attrs.CopyTo(dp.Attributes())
		
		rate, ok := calc.CalculateRate(metric, dp)
		assert.True(t, ok)
		assert.Greater(t, rate, float64(0))
		// Rate should be approximately 250/sec
		assert.InDelta(t, 250, rate, 100)
	})
	
	t.Run("different attributes track separately", func(t *testing.T) {
		attrs2 := pcommon.NewMap()
		attrs2.PutStr("host", "server2")
		
		dp := pmetric.NewNumberDataPoint()
		dp.SetDoubleValue(200)
		dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		attrs2.CopyTo(dp.Attributes())
		
		rate, ok := calc.CalculateRate(metric, dp)
		assert.False(t, ok) // First observation for this attribute set
		assert.Equal(t, float64(0), rate)
	})
}

func TestCardinalityLimiter(t *testing.T) {
	limiter := NewCardinalityLimiter(3)
	
	metric := pmetric.NewMetric()
	metric.SetName("test.metric")
	
	t.Run("allows metrics under limit", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			attrs := pcommon.NewMap()
			attrs.PutStr("label", string(rune('a'+i)))
			
			allowed := limiter.ShouldKeep(metric, attrs)
			assert.True(t, allowed)
		}
	})
	
	t.Run("blocks metrics over limit", func(t *testing.T) {
		attrs := pcommon.NewMap()
		attrs.PutStr("label", "d")
		
		allowed := limiter.ShouldKeep(metric, attrs)
		assert.False(t, allowed)
	})
	
	t.Run("tracks different metrics separately", func(t *testing.T) {
		metric2 := pmetric.NewMetric()
		metric2.SetName("other.metric")
		
		attrs := pcommon.NewMap()
		attrs.PutStr("label", "a")
		
		allowed := limiter.ShouldKeep(metric2, attrs)
		assert.True(t, allowed)
	})
}

func TestMetricAggregator(t *testing.T) {
	agg := NewMetricAggregator()
	
	t.Run("aggregates values correctly", func(t *testing.T) {
		// Add values for key1
		agg.AddValue("key1", 10)
		agg.AddValue("key1", 20)
		agg.AddValue("key1", 30)
		
		// Add values for key2
		agg.AddValue("key2", 5)
		agg.AddValue("key2", 15)
		
		results := agg.GetAggregations()
		
		// Check key1 aggregations
		key1Result := results["key1"]
		assert.Equal(t, float64(60), key1Result.Sum)
		assert.Equal(t, int64(3), key1Result.Count)
		assert.Equal(t, float64(10), key1Result.Min)
		assert.Equal(t, float64(30), key1Result.Max)
		assert.Equal(t, float64(20), key1Result.Avg)
		
		// Check key2 aggregations
		key2Result := results["key2"]
		assert.Equal(t, float64(20), key2Result.Sum)
		assert.Equal(t, int64(2), key2Result.Count)
		assert.Equal(t, float64(5), key2Result.Min)
		assert.Equal(t, float64(15), key2Result.Max)
		assert.Equal(t, float64(10), key2Result.Avg)
	})
}

func TestFilterMetrics(t *testing.T) {
	// Create test metrics
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	
	// Add various metrics
	metric1 := sm.Metrics().AppendEmpty()
	metric1.SetName("cpu.usage")
	metric1.SetEmptyGauge()
	
	metric2 := sm.Metrics().AppendEmpty()
	metric2.SetName("memory.usage")
	metric2.SetEmptyGauge()
	
	metric3 := sm.Metrics().AppendEmpty()
	metric3.SetName("disk.usage")
	metric3.SetEmptyGauge()
	
	t.Run("filter by name prefix", func(t *testing.T) {
		filtered := FilterMetrics(metrics, func(m pmetric.Metric) bool {
			return m.Name() == "cpu.usage" || m.Name() == "memory.usage"
		})
		
		assert.Equal(t, 1, filtered.ResourceMetrics().Len())
		assert.Equal(t, 1, filtered.ResourceMetrics().At(0).ScopeMetrics().Len())
		assert.Equal(t, 2, filtered.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().Len())
		
		// Check that correct metrics were kept
		filteredMetrics := filtered.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()
		names := []string{
			filteredMetrics.At(0).Name(),
			filteredMetrics.At(1).Name(),
		}
		assert.Contains(t, names, "cpu.usage")
		assert.Contains(t, names, "memory.usage")
		assert.NotContains(t, names, "disk.usage")
	})
	
	t.Run("filter all metrics removes empty structures", func(t *testing.T) {
		filtered := FilterMetrics(metrics, func(m pmetric.Metric) bool {
			return false // Filter out everything
		})
		
		assert.Equal(t, 0, filtered.ResourceMetrics().Len())
	})
}