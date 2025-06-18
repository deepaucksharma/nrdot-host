package nrsecurity

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.uber.org/zap"
)

func TestProcessorCapabilities(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	p, err := newProcessor(cfg, zap.NewNop())
	require.NoError(t, err)

	caps := p.Capabilities()
	assert.True(t, caps.MutatesData)
}

func TestProcessorStartShutdown(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	p, err := newProcessor(cfg, zap.NewNop())
	require.NoError(t, err)

	err = p.Start(context.Background(), nil)
	assert.NoError(t, err)

	err = p.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestProcessTracesDisabled(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Enabled = false

	p, err := newProcessor(cfg, zap.NewNop())
	require.NoError(t, err)

	td := generateTestTraces()
	result, err := p.processTraces(context.Background(), td)
	require.NoError(t, err)

	// Should pass through unchanged
	assert.Equal(t, td, result)
}

func TestProcessTracesRedaction(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Enabled = true

	p, err := newProcessor(cfg, zap.NewNop())
	require.NoError(t, err)

	td := generateTestTraces()
	
	// Add sensitive data
	rs := td.ResourceSpans().At(0)
	rs.Resource().Attributes().PutStr("api_key", "AKIA1234567890ABCDEF")
	rs.Resource().Attributes().PutStr("password", "secret123")
	rs.Resource().Attributes().PutStr("service.name", "test-service")

	span := rs.ScopeSpans().At(0).Spans().At(0)
	span.Attributes().PutStr("http.url", "https://user:password123@example.com")
	span.Attributes().PutStr("credit_card", "4111111111111111")

	result, err := p.processTraces(context.Background(), td)
	require.NoError(t, err)

	// Check redaction
	attrs := result.ResourceSpans().At(0).Resource().Attributes()
	
	// These should be redacted
	val, _ := attrs.Get("api_key")
	assert.Equal(t, "[REDACTED]", val.Str())
	
	val, _ = attrs.Get("password")
	assert.Equal(t, "[REDACTED]", val.Str())
	
	// This should not be redacted (in allow list)
	val, _ = attrs.Get("service.name")
	assert.Equal(t, "test-service", val.Str())

	// Check span attributes
	spanAttrs := result.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).Attributes()
	val, _ = spanAttrs.Get("http.url")
	assert.Contains(t, val.Str(), "[REDACTED]")
	
	val, _ = spanAttrs.Get("credit_card")
	assert.Equal(t, "[REDACTED]", val.Str())
}

func TestProcessMetricsRedaction(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Enabled = true

	p, err := newProcessor(cfg, zap.NewNop())
	require.NoError(t, err)

	md := generateTestMetrics()
	
	// Add sensitive data
	rm := md.ResourceMetrics().At(0)
	rm.Resource().Attributes().PutStr("db.connection_string", "postgres://user:pass123@localhost:5432/db")
	
	metric := rm.ScopeMetrics().At(0).Metrics().At(0)
	dp := metric.Gauge().DataPoints().At(0)
	dp.Attributes().PutStr("auth_token", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U")

	result, err := p.processMetrics(context.Background(), md)
	require.NoError(t, err)

	// Check redaction
	attrs := result.ResourceMetrics().At(0).Resource().Attributes()
	
	val, _ := attrs.Get("db.connection_string")
	assert.Equal(t, "[REDACTED]", val.Str())

	dpAttrs := result.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Gauge().DataPoints().At(0).Attributes()
	val, _ = dpAttrs.Get("auth_token")
	assert.Equal(t, "[REDACTED]", val.Str())
}

func TestProcessLogsRedaction(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Enabled = true

	p, err := newProcessor(cfg, zap.NewNop())
	require.NoError(t, err)

	ld := generateTestLogs()
	
	// Add sensitive data
	rl := ld.ResourceLogs().At(0)
	rl.Resource().Attributes().PutStr("aws_secret_key", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
	
	log := rl.ScopeLogs().At(0).LogRecords().At(0)
	log.Attributes().PutStr("ssn", "123-45-6789")
	log.Body().SetStr("User logged in with password: secret123")

	result, err := p.processLogs(context.Background(), ld)
	require.NoError(t, err)

	// Check redaction
	attrs := result.ResourceLogs().At(0).Resource().Attributes()
	
	val, _ := attrs.Get("aws_secret_key")
	assert.Equal(t, "[REDACTED]", val.Str())

	logAttrs := result.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Attributes()
	val, _ = logAttrs.Get("ssn")
	assert.Equal(t, "[REDACTED]", val.Str())

	// Check log body redaction
	body := result.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Body().Str()
	assert.Contains(t, body, "[REDACTED]")
	assert.NotContains(t, body, "secret123")
}

func TestNestedAttributesRedaction(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Enabled = true

	p, err := newProcessor(cfg, zap.NewNop())
	require.NoError(t, err)

	td := generateTestTraces()
	
	// Add nested attributes with sensitive data
	rs := td.ResourceSpans().At(0)
	nested := rs.Resource().Attributes().PutEmptyMap("user")
	nested.PutStr("name", "John Doe")
	nested.PutStr("password", "secret123")
	nested.PutStr("email", "john@example.com")

	result, err := p.processTraces(context.Background(), td)
	require.NoError(t, err)

	// Check redaction
	attrs := result.ResourceSpans().At(0).Resource().Attributes()
	
	userMap, _ := attrs.Get("user")
	userAttrs := userMap.Map()
	
	val, _ := userAttrs.Get("name")
	assert.Equal(t, "John Doe", val.Str())
	
	val, _ = userAttrs.Get("password")
	assert.Equal(t, "[REDACTED]", val.Str())
	
	val, _ = userAttrs.Get("email")
	assert.Equal(t, "john@example.com", val.Str()) // Email not redacted by default
}

func TestCustomPatternsRedaction(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Enabled = true
	cfg.Patterns = []PatternConfig{
		{
			Name:  "custom_id",
			Regex: `CUSTOM-[0-9]{8}`,
		},
	}

	p, err := newProcessor(cfg, zap.NewNop())
	require.NoError(t, err)

	td := generateTestTraces()
	
	// Add custom pattern data
	rs := td.ResourceSpans().At(0)
	rs.Resource().Attributes().PutStr("custom_field", "ID: CUSTOM-12345678")

	result, err := p.processTraces(context.Background(), td)
	require.NoError(t, err)

	// Check redaction
	attrs := result.ResourceSpans().At(0).Resource().Attributes()
	
	val, _ := attrs.Get("custom_field")
	assert.Equal(t, "[REDACTED]", val.Str())
}

func TestEmailAndIPRedaction(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Enabled = true
	cfg.RedactEmails = true
	cfg.RedactIPs = true

	p, err := newProcessor(cfg, zap.NewNop())
	require.NoError(t, err)

	td := generateTestTraces()
	
	// Add email and IP data
	rs := td.ResourceSpans().At(0)
	rs.Resource().Attributes().PutStr("user_email", "user@example.com")
	rs.Resource().Attributes().PutStr("client_ip", "192.168.1.100")

	result, err := p.processTraces(context.Background(), td)
	require.NoError(t, err)

	// Check redaction
	attrs := result.ResourceSpans().At(0).Resource().Attributes()
	
	val, _ := attrs.Get("user_email")
	assert.Equal(t, "[REDACTED]", val.Str())
	
	val, _ = attrs.Get("client_ip")
	assert.Equal(t, "[REDACTED]", val.Str())
}

// Helper functions to generate test data

func generateTestTraces() ptrace.Traces {
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("test", "value")
	
	ils := rs.ScopeSpans().AppendEmpty()
	span := ils.Spans().AppendEmpty()
	span.SetName("test-span")
	span.SetTraceID(pcommon.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))
	span.SetSpanID(pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}))
	
	return td
}

func generateTestMetrics() pmetric.Metrics {
	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().PutStr("test", "value")
	
	ilm := rm.ScopeMetrics().AppendEmpty()
	metric := ilm.Metrics().AppendEmpty()
	metric.SetName("test-metric")
	metric.SetEmptyGauge()
	
	dp := metric.Gauge().DataPoints().AppendEmpty()
	dp.SetDoubleValue(123.45)
	
	return md
}

func generateTestLogs() plog.Logs {
	ld := plog.NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()
	rl.Resource().Attributes().PutStr("test", "value")
	
	ill := rl.ScopeLogs().AppendEmpty()
	log := ill.LogRecords().AppendEmpty()
	log.Body().SetStr("test log message")
	
	return ld
}

func TestFactory(t *testing.T) {
	factory := NewFactory()
	assert.Equal(t, component.Type(TypeStr), factory.Type())
	
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg)
	
	// Test config validation
	err := cfg.(*Config).Validate()
	assert.NoError(t, err)
	
	// Test invalid config
	invalidCfg := &Config{
		Enabled:         true,
		ReplacementText: "",
	}
	err = invalidCfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "replacement_text cannot be empty")
}

func TestCreateProcessors(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	ctx := context.Background()
	set := processortest.NewNopCreateSettings()

	// Test traces processor
	tp, err := factory.CreateTracesProcessor(ctx, set, cfg, consumertest.NewNop())
	assert.NoError(t, err)
	assert.NotNil(t, tp)

	// Test metrics processor
	mp, err := factory.CreateMetricsProcessor(ctx, set, cfg, consumertest.NewNop())
	assert.NoError(t, err)
	assert.NotNil(t, mp)

	// Test logs processor
	lp, err := factory.CreateLogsProcessor(ctx, set, cfg, consumertest.NewNop())
	assert.NoError(t, err)
	assert.NotNil(t, lp)
}