package nrtransform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

func TestTransformer_ExtractLabel(t *testing.T) {
	config := &Config{
		Transformations: []TransformationConfig{
			{
				Type:         TransformTypeExtractLabel,
				MetricName:   "http.requests",
				LabelKey:     "status_code",
				OutputMetric: "http.requests.by_status",
			},
		},
	}

	logger := zap.NewNop()
	transformer, err := NewTransformer(config, logger)
	require.NoError(t, err)

	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	
	// Create metric with different status codes
	metric := sm.Metrics().AppendEmpty()
	metric.SetName("http.requests")
	metric.SetEmptySum()
	metric.Sum().SetIsMonotonic(true)
	
	// Add data points with different status codes
	statusCodes := []string{"200", "404", "500"}
	for i, code := range statusCodes {
		dp := metric.Sum().DataPoints().AppendEmpty()
		dp.SetDoubleValue(float64((i + 1) * 100))
		dp.Attributes().PutStr("status_code", code)
		dp.Attributes().PutStr("method", "GET")
	}

	err = transformer.Transform(metrics)
	require.NoError(t, err)

	// Verify we have the original metric plus the extracted one
	assert.Equal(t, 2, metrics.MetricCount())
	
	// Find the extracted metric
	var extracted pmetric.Metric
	outputMetrics := metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()
	for i := 0; i < outputMetrics.Len(); i++ {
		if outputMetrics.At(i).Name() == "http.requests.by_status" {
			extracted = outputMetrics.At(i)
			break
		}
	}
	
	require.NotEqual(t, "", extracted.Name())
	assert.Equal(t, 3, extracted.Sum().DataPoints().Len())
}

func TestTransformer_MultipleTransformations(t *testing.T) {
	config := &Config{
		Transformations: []TransformationConfig{
			{
				Type:         TransformTypeRename,
				MetricName:   "old.name",
				OutputMetric: "new.name",
			},
			{
				Type:         TransformTypeConvertUnit,
				MetricName:   "time.ms",
				OutputMetric: "time.seconds",
				FromUnit:     "milliseconds",
				ToUnit:       "seconds",
			},
		},
	}

	logger := zap.NewNop()
	transformer, err := NewTransformer(config, logger)
	require.NoError(t, err)

	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	
	// Create first metric to rename
	metric1 := sm.Metrics().AppendEmpty()
	metric1.SetName("old.name")
	metric1.SetEmptyGauge()
	dp1 := metric1.Gauge().DataPoints().AppendEmpty()
	dp1.SetDoubleValue(42.0)
	
	// Create second metric to convert units
	metric2 := sm.Metrics().AppendEmpty()
	metric2.SetName("time.ms")
	metric2.SetUnit("milliseconds")
	metric2.SetEmptyGauge()
	dp2 := metric2.Gauge().DataPoints().AppendEmpty()
	dp2.SetDoubleValue(1000.0)

	err = transformer.Transform(metrics)
	require.NoError(t, err)

	// Should have 3 metrics: renamed, original time.ms, and converted
	assert.Equal(t, 3, metrics.MetricCount())
	
	// Verify transformations
	outputMetrics := metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()
	
	foundRenamed := false
	foundConverted := false
	
	for i := 0; i < outputMetrics.Len(); i++ {
		metric := outputMetrics.At(i)
		switch metric.Name() {
		case "new.name":
			foundRenamed = true
			assert.Equal(t, 42.0, metric.Gauge().DataPoints().At(0).DoubleValue())
		case "time.seconds":
			foundConverted = true
			assert.Equal(t, "seconds", metric.Unit())
			assert.Equal(t, 1.0, metric.Gauge().DataPoints().At(0).DoubleValue())
		}
	}
	
	assert.True(t, foundRenamed, "renamed metric not found")
	assert.True(t, foundConverted, "converted metric not found")
}

func TestAttributeKey(t *testing.T) {
	transformer := &Transformer{logger: zap.NewNop()}
	
	attrs := pcommon.NewMap()
	attrs.PutStr("service", "api")
	attrs.PutStr("host", "server1")
	attrs.PutStr("env", "prod")
	
	key := transformer.attributeKey(attrs)
	// Should be sorted alphabetically
	assert.Equal(t, "env=prod,host=server1,service=api", key)
}

func TestTransformer_InvalidExpression(t *testing.T) {
	config := &Config{
		Transformations: []TransformationConfig{
			{
				Type:         TransformTypeCombine,
				Expression:   "invalid syntax {{",
				OutputMetric: "result",
				Metrics:      []string{"metric1"},
			},
		},
	}

	logger := zap.NewNop()
	_, err := NewTransformer(config, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to compile expression")
}