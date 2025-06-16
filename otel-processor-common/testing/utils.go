package testing

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

// TestTelemetrySettings creates telemetry settings for testing
func TestTelemetrySettings() component.TelemetrySettings {
	return component.TelemetrySettings{
		Logger:         zap.NewNop(),
		TracerProvider: componenttest.NewNopTelemetrySettings().TracerProvider,
		MeterProvider:  componenttest.NewNopTelemetrySettings().MeterProvider,
	}
}

// MockConsumer implements all consumer interfaces for testing
type MockConsumer struct {
	Metrics []pmetric.Metrics
	Traces  []ptrace.Traces
	Logs    []plog.Logs
}

// NewMockConsumer creates a new mock consumer
func NewMockConsumer() *MockConsumer {
	return &MockConsumer{
		Metrics: make([]pmetric.Metrics, 0),
		Traces:  make([]ptrace.Traces, 0),
		Logs:    make([]plog.Logs, 0),
	}
}

// ConsumeMetrics implements consumer.Metrics
func (m *MockConsumer) ConsumeMetrics(_ context.Context, md pmetric.Metrics) error {
	m.Metrics = append(m.Metrics, md)
	return nil
}

// ConsumeTraces implements consumer.Traces
func (m *MockConsumer) ConsumeTraces(_ context.Context, td ptrace.Traces) error {
	m.Traces = append(m.Traces, td)
	return nil
}

// ConsumeLogs implements consumer.Logs
func (m *MockConsumer) ConsumeLogs(_ context.Context, ld plog.Logs) error {
	m.Logs = append(m.Logs, ld)
	return nil
}

// Capabilities implements consumer interfaces
func (m *MockConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// MetricBuilder helps create test metrics
type MetricBuilder struct {
	metrics pmetric.Metrics
	rm      pmetric.ResourceMetrics
	sm      pmetric.ScopeMetrics
}

// NewMetricBuilder creates a new metric builder
func NewMetricBuilder() *MetricBuilder {
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	
	return &MetricBuilder{
		metrics: metrics,
		rm:      rm,
		sm:      sm,
	}
}

// AddResourceAttribute adds a resource attribute
func (mb *MetricBuilder) AddResourceAttribute(key string, value interface{}) *MetricBuilder {
	switch v := value.(type) {
	case string:
		mb.rm.Resource().Attributes().PutStr(key, v)
	case int:
		mb.rm.Resource().Attributes().PutInt(key, int64(v))
	case float64:
		mb.rm.Resource().Attributes().PutDouble(key, v)
	case bool:
		mb.rm.Resource().Attributes().PutBool(key, v)
	}
	return mb
}

// AddGaugeMetric adds a gauge metric
func (mb *MetricBuilder) AddGaugeMetric(name string, value float64, attrs map[string]string) *MetricBuilder {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName(name)
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetDoubleValue(value)
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	
	for k, v := range attrs {
		dp.Attributes().PutStr(k, v)
	}
	
	return mb
}

// AddCounterMetric adds a counter metric
func (mb *MetricBuilder) AddCounterMetric(name string, value float64, attrs map[string]string) *MetricBuilder {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName(name)
	
	sum := metric.SetEmptySum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	
	dp := sum.DataPoints().AppendEmpty()
	dp.SetDoubleValue(value)
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	
	for k, v := range attrs {
		dp.Attributes().PutStr(k, v)
	}
	
	return mb
}

// AddHistogramMetric adds a histogram metric
func (mb *MetricBuilder) AddHistogramMetric(name string, count uint64, sum float64, bounds []float64, buckets []uint64, attrs map[string]string) *MetricBuilder {
	metric := mb.sm.Metrics().AppendEmpty()
	metric.SetName(name)
	
	histogram := metric.SetEmptyHistogram()
	histogram.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	
	dp := histogram.DataPoints().AppendEmpty()
	dp.SetCount(count)
	dp.SetSum(sum)
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	
	dp.ExplicitBounds().FromRaw(bounds)
	dp.BucketCounts().FromRaw(buckets)
	
	for k, v := range attrs {
		dp.Attributes().PutStr(k, v)
	}
	
	return mb
}

// Build returns the built metrics
func (mb *MetricBuilder) Build() pmetric.Metrics {
	return mb.metrics
}

// AssertMetricsEqual asserts that two metric sets are equal
func AssertMetricsEqual(t *testing.T, expected, actual pmetric.Metrics) {
	require.Equal(t, expected.ResourceMetrics().Len(), actual.ResourceMetrics().Len(), "Resource metrics count mismatch")
	
	for i := 0; i < expected.ResourceMetrics().Len(); i++ {
		expectedRM := expected.ResourceMetrics().At(i)
		actualRM := actual.ResourceMetrics().At(i)
		
		// Compare resource attributes
		assert.Equal(t, expectedRM.Resource().Attributes().AsRaw(), actualRM.Resource().Attributes().AsRaw())
		
		require.Equal(t, expectedRM.ScopeMetrics().Len(), actualRM.ScopeMetrics().Len(), "Scope metrics count mismatch")
		
		for j := 0; j < expectedRM.ScopeMetrics().Len(); j++ {
			expectedSM := expectedRM.ScopeMetrics().At(j)
			actualSM := actualRM.ScopeMetrics().At(j)
			
			require.Equal(t, expectedSM.Metrics().Len(), actualSM.Metrics().Len(), "Metrics count mismatch")
			
			for k := 0; k < expectedSM.Metrics().Len(); k++ {
				expectedMetric := expectedSM.Metrics().At(k)
				actualMetric := actualSM.Metrics().At(k)
				
				assert.Equal(t, expectedMetric.Name(), actualMetric.Name())
				assert.Equal(t, expectedMetric.Type(), actualMetric.Type())
			}
		}
	}
}

// ProcessorTestCase defines a test case for processor testing
type ProcessorTestCase struct {
	Name     string
	Input    interface{} // pmetric.Metrics, ptrace.Traces, or plog.Logs
	Expected interface{}
	WantErr  bool
}

// RunProcessorTests runs a set of processor test cases
func RunProcessorTests(t *testing.T, processor interface{}, testCases []ProcessorTestCase) {
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := context.Background()
			
			var err error
			var result interface{}
			
			switch p := processor.(type) {
			case interface{ ProcessMetrics(context.Context, pmetric.Metrics) (pmetric.Metrics, error) }:
				result, err = p.ProcessMetrics(ctx, tc.Input.(pmetric.Metrics))
			case interface{ ProcessTraces(context.Context, ptrace.Traces) (ptrace.Traces, error) }:
				result, err = p.ProcessTraces(ctx, tc.Input.(ptrace.Traces))
			case interface{ ProcessLogs(context.Context, plog.Logs) (plog.Logs, error) }:
				result, err = p.ProcessLogs(ctx, tc.Input.(plog.Logs))
			default:
				t.Fatalf("Unknown processor type")
			}
			
			if tc.WantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				
				switch expected := tc.Expected.(type) {
				case pmetric.Metrics:
					AssertMetricsEqual(t, expected, result.(pmetric.Metrics))
				case ptrace.Traces:
					assert.Equal(t, expected, result)
				case plog.Logs:
					assert.Equal(t, expected, result)
				}
			}
		})
	}
}