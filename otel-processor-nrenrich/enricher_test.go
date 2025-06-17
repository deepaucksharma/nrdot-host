package nrenrich

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

func TestNewEnricher(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				StaticAttributes: map[string]interface{}{"env": "test"},
				Environment: EnvironmentConfig{
					Enabled: true,
					System:  true,
				},
				Cache: CacheConfig{TTL: 300, MaxSize: 100},
			},
			wantErr: false,
		},
		{
			name: "with process enrichment disabled",
			config: &Config{
				Process: ProcessConfig{
					Enabled: false,
				},
				Cache: CacheConfig{TTL: 300, MaxSize: 100},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enricher, err := NewEnricher(tt.config, logger)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, enricher)
			}
		})
	}
}

func TestEnricherEnrichTraces(t *testing.T) {
	logger := zap.NewNop()
	config := &Config{
		StaticAttributes: map[string]interface{}{
			"static.env":  "test",
			"static.team": "platform",
		},
		Cache: CacheConfig{TTL: 300, MaxSize: 100},
	}

	enricher, err := NewEnricher(config, logger)
	require.NoError(t, err)

	// Create test traces
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("service.name", "test-service")
	ss := rs.ScopeSpans().AppendEmpty()
	span := ss.Spans().AppendEmpty()
	span.SetName("test-span")
	span.Attributes().PutStr("existing", "value")

	// Enrich traces
	err = enricher.EnrichTraces(context.Background(), td)
	assert.NoError(t, err)

	// Verify enrichment
	enrichedSpan := td.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)

	// Check static attributes
	val, exists := enrichedSpan.Attributes().Get("static.env")
	assert.True(t, exists)
	assert.Equal(t, "test", val.Str())

	val, exists = enrichedSpan.Attributes().Get("static.team")
	assert.True(t, exists)
	assert.Equal(t, "platform", val.Str())

	// Check existing attributes preserved
	val, exists = enrichedSpan.Attributes().Get("existing")
	assert.True(t, exists)
	assert.Equal(t, "value", val.Str())

	// Check resource attributes
	enrichedResource := td.ResourceSpans().At(0).Resource()
	val, exists = enrichedResource.Attributes().Get("static.env")
	assert.True(t, exists)
	assert.Equal(t, "test", val.Str())
}

func TestEnricherEnrichMetrics(t *testing.T) {
	logger := zap.NewNop()
	config := &Config{
		StaticAttributes: map[string]interface{}{
			"static.env": "prod",
		},
		Cache: CacheConfig{TTL: 300, MaxSize: 100},
	}

	enricher, err := NewEnricher(config, logger)
	require.NoError(t, err)

	// Create test metrics
	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	metric := sm.Metrics().AppendEmpty()
	metric.SetName("test.metric")
	dp := metric.SetEmptyGauge().DataPoints().AppendEmpty()
	dp.SetDoubleValue(123.45)
	dp.Attributes().PutStr("metric.type", "gauge")

	// Enrich metrics
	err = enricher.EnrichMetrics(context.Background(), md)
	assert.NoError(t, err)

	// Verify enrichment
	enrichedDp := md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Gauge().DataPoints().At(0)

	// Check static attributes
	val, exists := enrichedDp.Attributes().Get("static.env")
	assert.True(t, exists)
	assert.Equal(t, "prod", val.Str())

	// Check existing attributes preserved
	val, exists = enrichedDp.Attributes().Get("metric.type")
	assert.True(t, exists)
	assert.Equal(t, "gauge", val.Str())
}

func TestEnricherEnrichLogs(t *testing.T) {
	logger := zap.NewNop()
	config := &Config{
		StaticAttributes: map[string]interface{}{
			"static.env":     "staging",
			"static.version": "1.2.3",
		},
		Cache: CacheConfig{TTL: 300, MaxSize: 100},
	}

	enricher, err := NewEnricher(config, logger)
	require.NoError(t, err)

	// Create test logs
	ld := plog.NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()
	log := sl.LogRecords().AppendEmpty()
	log.Body().SetStr("test log message")
	log.Attributes().PutStr("level", "info")

	// Enrich logs
	err = enricher.EnrichLogs(context.Background(), ld)
	assert.NoError(t, err)

	// Verify enrichment
	enrichedLog := ld.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)

	// Check static attributes
	val, exists := enrichedLog.Attributes().Get("static.env")
	assert.True(t, exists)
	assert.Equal(t, "staging", val.Str())

	val, exists = enrichedLog.Attributes().Get("static.version")
	assert.True(t, exists)
	assert.Equal(t, "1.2.3", val.Str())

	// Check existing attributes preserved
	val, exists = enrichedLog.Attributes().Get("level")
	assert.True(t, exists)
	assert.Equal(t, "info", val.Str())
}

func TestSetAttributeValue(t *testing.T) {
	attrs := pcommon.NewMap()

	tests := []struct {
		name     string
		key      string
		value    interface{}
		expected func(pcommon.Map) bool
	}{
		{
			name:  "string value",
			key:   "str",
			value: "test",
			expected: func(m pcommon.Map) bool {
				v, ok := m.Get("str")
				return ok && v.Type() == pcommon.ValueTypeStr && v.Str() == "test"
			},
		},
		{
			name:  "int value",
			key:   "int",
			value: 42,
			expected: func(m pcommon.Map) bool {
				v, ok := m.Get("int")
				return ok && v.Type() == pcommon.ValueTypeInt && v.Int() == 42
			},
		},
		{
			name:  "float value",
			key:   "float",
			value: 3.14,
			expected: func(m pcommon.Map) bool {
				v, ok := m.Get("float")
				return ok && v.Type() == pcommon.ValueTypeDouble && v.Double() == 3.14
			},
		},
		{
			name:  "bool value",
			key:   "bool",
			value: true,
			expected: func(m pcommon.Map) bool {
				v, ok := m.Get("bool")
				return ok && v.Type() == pcommon.ValueTypeBool && v.Bool() == true
			},
		},
		{
			name:  "string slice",
			key:   "slice",
			value: []string{"a", "b", "c"},
			expected: func(m pcommon.Map) bool {
				v, ok := m.Get("slice")
				if !ok || v.Type() != pcommon.ValueTypeSlice {
					return false
				}
				slice := v.Slice()
				return slice.Len() == 3 &&
					slice.At(0).Str() == "a" &&
					slice.At(1).Str() == "b" &&
					slice.At(2).Str() == "c"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs.Clear()
			setAttributeValue(attrs, tt.key, tt.value)
			assert.True(t, tt.expected(attrs))
		})
	}
}