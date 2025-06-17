package nrtransform

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func TestCalculateRate(t *testing.T) {
	calculator := NewMetricCalculator()
	
	// Create a cumulative sum metric
	metric := pmetric.NewMetric()
	metric.SetName("requests.total")
	metric.SetEmptySum()
	metric.Sum().SetIsMonotonic(true)
	metric.Sum().SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	
	// First observation
	dp1 := metric.Sum().DataPoints().AppendEmpty()
	dp1.SetDoubleValue(100)
	dp1.SetTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(-10 * time.Second)))
	dp1.Attributes().PutStr("service", "api")
	
	// Calculate rate (should be empty as it's the first observation)
	rate1, err := calculator.CalculateRate(metric, "requests.rate")
	require.NoError(t, err)
	assert.Equal(t, "requests.rate", rate1.Name())
	assert.Equal(t, 0, rate1.Gauge().DataPoints().Len())
	
	// Second observation - create a new metric
	metric2 := pmetric.NewMetric()
	metric2.SetName("requests.total")
	metric2.SetEmptySum()
	metric2.Sum().SetIsMonotonic(true)
	metric2.Sum().SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	dp2 := metric2.Sum().DataPoints().AppendEmpty()
	dp2.SetDoubleValue(200)
	dp2.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	dp2.Attributes().PutStr("service", "api")
	
	// Calculate rate again
	rate2, err := calculator.CalculateRate(metric2, "requests.rate")
	require.NoError(t, err)
	assert.Equal(t, 1, rate2.Gauge().DataPoints().Len())
	
	// Rate should be approximately 10 req/s (100 requests in ~10 seconds)
	rateValue := rate2.Gauge().DataPoints().At(0).DoubleValue()
	assert.InDelta(t, 10.0, rateValue, 1.0)
}

func TestCalculateDelta(t *testing.T) {
	calculator := NewMetricCalculator()
	
	// Create a cumulative sum metric
	metric := pmetric.NewMetric()
	metric.SetName("bytes.total")
	metric.SetEmptySum()
	metric.Sum().SetIsMonotonic(true)
	metric.Sum().SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	
	// First observation
	dp1 := metric.Sum().DataPoints().AppendEmpty()
	dp1.SetDoubleValue(1000)
	dp1.SetTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(-5 * time.Second)))
	dp1.Attributes().PutStr("interface", "eth0")
	
	// Calculate delta (should be empty as it's the first observation)
	delta1, err := calculator.CalculateDelta(metric, "bytes.delta")
	require.NoError(t, err)
	assert.Equal(t, "bytes.delta", delta1.Name())
	assert.Equal(t, 0, delta1.Sum().DataPoints().Len())
	
	// Second observation - create a new metric
	metric2 := pmetric.NewMetric()
	metric2.SetName("bytes.total")
	metric2.SetEmptySum()
	metric2.Sum().SetIsMonotonic(true)
	metric2.Sum().SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	dp2 := metric2.Sum().DataPoints().AppendEmpty()
	dp2.SetDoubleValue(1500)
	dp2.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	dp2.Attributes().PutStr("interface", "eth0")
	
	// Calculate delta again
	delta2, err := calculator.CalculateDelta(metric2, "bytes.delta")
	require.NoError(t, err)
	assert.Equal(t, 1, delta2.Sum().DataPoints().Len())
	assert.Equal(t, 500.0, delta2.Sum().DataPoints().At(0).DoubleValue())
}

func TestCalculateDelta_CounterReset(t *testing.T) {
	calculator := NewMetricCalculator()
	
	metric := pmetric.NewMetric()
	metric.SetName("counter")
	metric.SetEmptySum()
	metric.Sum().SetIsMonotonic(true)
	
	// First observation
	dp1 := metric.Sum().DataPoints().AppendEmpty()
	dp1.SetDoubleValue(1000)
	dp1.SetTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(-5 * time.Second)))
	
	_, err := calculator.CalculateDelta(metric, "counter.delta")
	require.NoError(t, err)
	
	// Counter reset (value less than previous) - create a new metric
	metric2 := pmetric.NewMetric()
	metric2.SetName("counter")
	metric2.SetEmptySum()
	metric2.Sum().SetIsMonotonic(true)
	dp2 := metric2.Sum().DataPoints().AppendEmpty()
	dp2.SetDoubleValue(100)
	dp2.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	
	delta, err := calculator.CalculateDelta(metric2, "counter.delta")
	require.NoError(t, err)
	// On counter reset, delta should be the current value
	assert.Equal(t, 100.0, delta.Sum().DataPoints().At(0).DoubleValue())
}

func TestAggregate(t *testing.T) {
	calculator := NewMetricCalculator()
	
	tests := []struct {
		name        string
		aggregation AggregationType
		values      []float64
		expected    float64
	}{
		{
			name:        "sum",
			aggregation: AggregationSum,
			values:      []float64{1, 2, 3, 4, 5},
			expected:    15,
		},
		{
			name:        "average",
			aggregation: AggregationAvg,
			values:      []float64{10, 20, 30},
			expected:    20,
		},
		{
			name:        "min",
			aggregation: AggregationMin,
			values:      []float64{5, 2, 8, 1, 9},
			expected:    1,
		},
		{
			name:        "max",
			aggregation: AggregationMax,
			values:      []float64{5, 2, 8, 1, 9},
			expected:    9,
		},
		{
			name:        "count",
			aggregation: AggregationCount,
			values:      []float64{1, 2, 3, 4, 5},
			expected:    5,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metric := pmetric.NewMetric()
			metric.SetName("test.metric")
			metric.SetEmptyGauge()
			
			// Add data points
			for _, val := range tt.values {
				dp := metric.Gauge().DataPoints().AppendEmpty()
				dp.SetDoubleValue(val)
				dp.Attributes().PutStr("group", "A")
			}
			
			result, err := calculator.Aggregate(metric, tt.aggregation, []string{"group"}, "test.aggregated")
			require.NoError(t, err)
			
			assert.Equal(t, 1, result.Gauge().DataPoints().Len())
			assert.Equal(t, tt.expected, result.Gauge().DataPoints().At(0).DoubleValue())
		})
	}
}

func TestAggregate_GroupBy(t *testing.T) {
	calculator := NewMetricCalculator()
	
	metric := pmetric.NewMetric()
	metric.SetName("response.time")
	metric.SetEmptyGauge()
	
	// Add data points for different services
	services := []struct {
		service string
		values  []float64
	}{
		{"api", []float64{100, 200, 300}},
		{"web", []float64{50, 150}},
		{"db", []float64{10, 20, 30, 40}},
	}
	
	for _, svc := range services {
		for _, val := range svc.values {
			dp := metric.Gauge().DataPoints().AppendEmpty()
			dp.SetDoubleValue(val)
			dp.Attributes().PutStr("service", svc.service)
			dp.Attributes().PutStr("region", "us-east")
		}
	}
	
	result, err := calculator.Aggregate(metric, AggregationAvg, []string{"service"}, "response.time.avg")
	require.NoError(t, err)
	
	assert.Equal(t, 3, result.Gauge().DataPoints().Len())
	
	// Verify averages for each service
	expectedAvgs := map[string]float64{
		"api": 200,  // (100+200+300)/3
		"web": 100,  // (50+150)/2
		"db":  25,   // (10+20+30+40)/4
	}
	
	for i := 0; i < result.Gauge().DataPoints().Len(); i++ {
		dp := result.Gauge().DataPoints().At(i)
		service, _ := dp.Attributes().Get("service")
		expectedAvg := expectedAvgs[service.AsString()]
		assert.Equal(t, expectedAvg, dp.DoubleValue())
	}
}

func TestConvertUnit(t *testing.T) {
	calculator := NewMetricCalculator()
	
	tests := []struct {
		name     string
		fromUnit string
		toUnit   string
		value    float64
		expected float64
	}{
		{
			name:     "bytes to kilobytes",
			fromUnit: "bytes",
			toUnit:   "kilobytes",
			value:    1024,
			expected: 1,
		},
		{
			name:     "bytes to megabytes",
			fromUnit: "bytes",
			toUnit:   "mb",
			value:    1048576,
			expected: 1,
		},
		{
			name:     "milliseconds to seconds",
			fromUnit: "milliseconds",
			toUnit:   "seconds",
			value:    1000,
			expected: 1,
		},
		{
			name:     "seconds to milliseconds",
			fromUnit: "seconds",
			toUnit:   "ms",
			value:    1,
			expected: 1000,
		},
		{
			name:     "percent to ratio",
			fromUnit: "percent",
			toUnit:   "ratio",
			value:    50,
			expected: 0.5,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metric := pmetric.NewMetric()
			metric.SetName("test.metric")
			metric.SetUnit(tt.fromUnit)
			metric.SetEmptyGauge()
			
			dp := metric.Gauge().DataPoints().AppendEmpty()
			dp.SetDoubleValue(tt.value)
			
			result, err := calculator.ConvertUnit(metric, tt.fromUnit, tt.toUnit, "test.converted")
			require.NoError(t, err)
			
			assert.Equal(t, tt.toUnit, result.Unit())
			assert.Equal(t, tt.expected, result.Gauge().DataPoints().At(0).DoubleValue())
		})
	}
}

func TestConvertUnit_Histogram(t *testing.T) {
	calculator := NewMetricCalculator()
	
	metric := pmetric.NewMetric()
	metric.SetName("request.duration")
	metric.SetUnit("milliseconds")
	metric.SetEmptyHistogram()
	
	dp := metric.Histogram().DataPoints().AppendEmpty()
	dp.SetCount(100)
	dp.SetSum(5000) // 5000ms total
	dp.SetMin(10)   // 10ms min
	dp.SetMax(200)  // 200ms max
	
	// Set bucket bounds in milliseconds
	bounds := dp.ExplicitBounds()
	bounds.Append(50)
	bounds.Append(100)
	bounds.Append(150)
	
	result, err := calculator.ConvertUnit(metric, "milliseconds", "seconds", "request.duration.seconds")
	require.NoError(t, err)
	
	assert.Equal(t, "seconds", result.Unit())
	
	resultDp := result.Histogram().DataPoints().At(0)
	assert.Equal(t, 5.0, resultDp.Sum())     // 5000ms = 5s
	assert.Equal(t, 0.01, resultDp.Min())    // 10ms = 0.01s
	assert.Equal(t, 0.2, resultDp.Max())     // 200ms = 0.2s
	
	// Check bucket bounds
	resultBounds := resultDp.ExplicitBounds()
	assert.Equal(t, 3, resultBounds.Len())
	assert.InDelta(t, 0.05, resultBounds.At(0), 0.001)  // 50ms = 0.05s
	assert.InDelta(t, 0.1, resultBounds.At(1), 0.001)   // 100ms = 0.1s
	assert.InDelta(t, 0.15, resultBounds.At(2), 0.001)  // 150ms = 0.15s
}

func TestCalculatePercentile(t *testing.T) {
	tests := []struct {
		name       string
		values     []float64
		percentile float64
		expected   float64
	}{
		{
			name:       "median",
			values:     []float64{1, 2, 3, 4, 5},
			percentile: 50,
			expected:   3,
		},
		{
			name:       "p95",
			values:     []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			percentile: 95,
			expected:   9.5,
		},
		{
			name:       "p99",
			values:     []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			percentile: 99,
			expected:   9.9,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculatePercentile(tt.values, tt.percentile)
			assert.InDelta(t, tt.expected, result, 0.1)
		})
	}
}