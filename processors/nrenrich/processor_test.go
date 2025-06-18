package nrenrich

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	// "go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor/processortest"
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory()
	assert.Equal(t, component.Type(typeStr), factory.Type())
	assert.Equal(t, stability, factory.TracesProcessorStability())
	assert.Equal(t, stability, factory.MetricsProcessorStability())
	assert.Equal(t, stability, factory.LogsProcessorStability())
}

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg)
	assert.NoError(t, componenttest.CheckConfigStruct(cfg))

	config := cfg.(*Config)
	assert.True(t, config.Environment.Enabled)
	assert.True(t, config.Environment.Hostname)
	assert.False(t, config.Process.Enabled)
	assert.Equal(t, 5*time.Minute, config.Cache.TTL)
}

func TestCreateTracesProcessor(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	ctx := context.Background()
	set := processortest.NewNopCreateSettings()

	// Test with valid configuration
	tp, err := factory.CreateTracesProcessor(ctx, set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NotNil(t, tp)

	// Start and shutdown
	err = tp.Start(ctx, componenttest.NewNopHost())
	assert.NoError(t, err)
	err = tp.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestCreateMetricsProcessor(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	ctx := context.Background()
	set := processortest.NewNopCreateSettings()

	// Test with valid configuration
	mp, err := factory.CreateMetricsProcessor(ctx, set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NotNil(t, mp)

	// Start and shutdown
	err = mp.Start(ctx, componenttest.NewNopHost())
	assert.NoError(t, err)
	err = mp.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestCreateLogsProcessor(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	ctx := context.Background()
	set := processortest.NewNopCreateSettings()

	// Test with valid configuration
	lp, err := factory.CreateLogsProcessor(ctx, set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NotNil(t, lp)

	// Start and shutdown
	err = lp.Start(ctx, componenttest.NewNopHost())
	assert.NoError(t, err)
	err = lp.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestProcessorEnrichTraces(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.StaticAttributes = map[string]interface{}{
		"env":  "test",
		"team": "platform",
	}

	ctx := context.Background()
	set := processortest.NewNopCreateSettings()
	nextConsumer := &consumertest.TracesSink{}

	tp, err := factory.CreateTracesProcessor(ctx, set, cfg, nextConsumer)
	require.NoError(t, err)

	err = tp.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)
	defer tp.Shutdown(ctx)

	// Create test traces
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	ss := rs.ScopeSpans().AppendEmpty()
	span := ss.Spans().AppendEmpty()
	span.SetName("test-span")
	span.Attributes().PutStr("existing", "value")

	// Process traces
	err = tp.ConsumeTraces(ctx, td)
	require.NoError(t, err)

	// Verify enrichment
	processedTraces := nextConsumer.AllTraces()
	require.Len(t, processedTraces, 1)

	processedTd := processedTraces[0]
	processedSpan := processedTd.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)

	// Check that static attributes were added
	val, exists := processedSpan.Attributes().Get("env")
	assert.True(t, exists)
	assert.Equal(t, "test", val.Str())

	val, exists = processedSpan.Attributes().Get("team")
	assert.True(t, exists)
	assert.Equal(t, "platform", val.Str())

	// Check that existing attributes were preserved
	val, exists = processedSpan.Attributes().Get("existing")
	assert.True(t, exists)
	assert.Equal(t, "value", val.Str())
}

func TestProcessorEnrichMetrics(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.StaticAttributes = map[string]interface{}{
		"env": "test",
	}

	ctx := context.Background()
	set := processortest.NewNopCreateSettings()
	nextConsumer := &consumertest.MetricsSink{}

	mp, err := factory.CreateMetricsProcessor(ctx, set, cfg, nextConsumer)
	require.NoError(t, err)

	err = mp.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)
	defer mp.Shutdown(ctx)

	// Create test metrics
	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	metric := sm.Metrics().AppendEmpty()
	metric.SetName("test.metric")
	dp := metric.SetEmptyGauge().DataPoints().AppendEmpty()
	dp.SetDoubleValue(123.45)
	dp.Attributes().PutStr("existing", "value")

	// Process metrics
	err = mp.ConsumeMetrics(ctx, md)
	require.NoError(t, err)

	// Verify enrichment
	processedMetrics := nextConsumer.AllMetrics()
	require.Len(t, processedMetrics, 1)

	processedMd := processedMetrics[0]
	processedDp := processedMd.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Gauge().DataPoints().At(0)

	// Check that static attributes were added
	val, exists := processedDp.Attributes().Get("env")
	assert.True(t, exists)
	assert.Equal(t, "test", val.Str())
}

func TestProcessorEnrichLogs(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.StaticAttributes = map[string]interface{}{
		"env": "test",
	}

	ctx := context.Background()
	set := processortest.NewNopCreateSettings()
	nextConsumer := &consumertest.LogsSink{}

	lp, err := factory.CreateLogsProcessor(ctx, set, cfg, nextConsumer)
	require.NoError(t, err)

	err = lp.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)
	defer lp.Shutdown(ctx)

	// Create test logs
	ld := plog.NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()
	log := sl.LogRecords().AppendEmpty()
	log.Body().SetStr("test log message")
	log.Attributes().PutStr("existing", "value")

	// Process logs
	err = lp.ConsumeLogs(ctx, ld)
	require.NoError(t, err)

	// Verify enrichment
	processedLogs := nextConsumer.AllLogs()
	require.Len(t, processedLogs, 1)

	processedLd := processedLogs[0]
	processedLog := processedLd.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)

	// Check that static attributes were added
	val, exists := processedLog.Attributes().Get("env")
	assert.True(t, exists)
	assert.Equal(t, "test", val.Str())
}