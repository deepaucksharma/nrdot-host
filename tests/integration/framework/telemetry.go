package framework

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TelemetryGenerator generates test telemetry data
type TelemetryGenerator struct {
	logger         *zap.Logger
	resource       *resource.Resource
	serviceName    string
	hostName       string
	includeSecrets bool
}

// TelemetryConfig configures telemetry generation
type TelemetryConfig struct {
	ServiceName      string
	HostName         string
	IncludeSecrets   bool
	ResourceAttrs    map[string]string
	HighCardinality  bool
	CardinalityLevel int
}

// NewTelemetryGenerator creates a new telemetry generator
func NewTelemetryGenerator() *TelemetryGenerator {
	return NewTelemetryGeneratorWithConfig(&TelemetryConfig{
		ServiceName: "test-service",
		HostName:    "test-host",
	})
}

// NewTelemetryGeneratorWithConfig creates a telemetry generator with custom config
func NewTelemetryGeneratorWithConfig(config *TelemetryConfig) *TelemetryGenerator {
	logger, _ := zap.NewDevelopment()

	// Build resource attributes
	attrs := []attribute.KeyValue{
		attribute.String("service.name", config.ServiceName),
		attribute.String("host.name", config.HostName),
		attribute.String("service.version", "1.0.0"),
		attribute.String("telemetry.sdk.name", "opentelemetry"),
		attribute.String("telemetry.sdk.language", "go"),
	}

	for k, v := range config.ResourceAttrs {
		attrs = append(attrs, attribute.String(k, v))
	}

	res, _ := resource.New(context.Background(),
		resource.WithAttributes(attrs...),
	)

	return &TelemetryGenerator{
		logger:         logger,
		resource:       res,
		serviceName:    config.ServiceName,
		hostName:       config.HostName,
		includeSecrets: config.IncludeSecrets,
	}
}

// GenerateMetrics generates test metrics
func (g *TelemetryGenerator) GenerateMetrics(count int) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	
	// Set resource attributes
	g.resource.Attributes().Range(func(kv attribute.KeyValue) bool {
		rm.Resource().Attributes().PutStr(string(kv.Key), kv.Value.AsString())
		return true
	})

	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("test-instrumentation")
	sm.Scope().SetVersion("1.0.0")

	// Generate different metric types
	for i := 0; i < count; i++ {
		switch i % 4 {
		case 0:
			g.generateGaugeMetric(sm.Metrics().AppendEmpty(), i)
		case 1:
			g.generateSumMetric(sm.Metrics().AppendEmpty(), i)
		case 2:
			g.generateHistogramMetric(sm.Metrics().AppendEmpty(), i)
		case 3:
			g.generateSummaryMetric(sm.Metrics().AppendEmpty(), i)
		}
	}

	return metrics
}

// GenerateHighCardinalityMetrics generates metrics with high cardinality
func (g *TelemetryGenerator) GenerateHighCardinalityMetrics(metricCount, labelCount int) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	
	g.resource.Attributes().Range(func(kv attribute.KeyValue) bool {
		rm.Resource().Attributes().PutStr(string(kv.Key), kv.Value.AsString())
		return true
	})

	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("high-cardinality-test")

	for i := 0; i < metricCount; i++ {
		metric := sm.Metrics().AppendEmpty()
		metric.SetName(fmt.Sprintf("high_cardinality_metric_%d", i))
		metric.SetUnit("1")
		
		gauge := metric.SetEmptyGauge()
		
		// Create multiple data points with unique label combinations
		for j := 0; j < labelCount; j++ {
			dp := gauge.DataPoints().AppendEmpty()
			dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
			dp.SetDoubleValue(rand.Float64() * 100)
			
			// Add unique labels
			for k := 0; k < 5; k++ {
				dp.Attributes().PutStr(
					fmt.Sprintf("label_%d", k),
					fmt.Sprintf("value_%d_%d_%d", i, j, k),
				)
			}
		}
	}

	return metrics
}

// GenerateTraces generates test traces
func (g *TelemetryGenerator) GenerateTraces(spanCount int) ptrace.Traces {
	traces := ptrace.NewTraces()
	rs := traces.ResourceSpans().AppendEmpty()
	
	g.resource.Attributes().Range(func(kv attribute.KeyValue) bool {
		rs.Resource().Attributes().PutStr(string(kv.Key), kv.Value.AsString())
		return true
	})

	ss := rs.ScopeSpans().AppendEmpty()
	ss.Scope().SetName("test-tracer")
	ss.Scope().SetVersion("1.0.0")

	// Create a trace with multiple spans
	traceID := pcommon.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	rootSpanID := pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8})

	// Root span
	rootSpan := ss.Spans().AppendEmpty()
	rootSpan.SetTraceID(traceID)
	rootSpan.SetSpanID(rootSpanID)
	rootSpan.SetName("root-operation")
	rootSpan.SetKind(ptrace.SpanKindServer)
	rootSpan.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(-time.Minute)))
	rootSpan.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	rootSpan.Status().SetCode(ptrace.StatusCodeOk)
	
	// Add attributes
	rootSpan.Attributes().PutStr("http.method", "GET")
	rootSpan.Attributes().PutStr("http.url", "/api/test")
	rootSpan.Attributes().PutInt("http.status_code", 200)
	
	if g.includeSecrets {
		rootSpan.Attributes().PutStr("api_key", "secret-api-key-12345")
		rootSpan.Attributes().PutStr("password", "my-secret-password")
	}

	// Child spans
	for i := 1; i < spanCount; i++ {
		span := ss.Spans().AppendEmpty()
		span.SetTraceID(traceID)
		span.SetSpanID(pcommon.SpanID([8]byte{byte(i + 1), 2, 3, 4, 5, 6, 7, 8}))
		span.SetParentSpanID(rootSpanID)
		span.SetName(fmt.Sprintf("child-operation-%d", i))
		span.SetKind(ptrace.SpanKindClient)
		span.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(-30 * time.Second)))
		span.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(-10 * time.Second)))
		
		// Add events
		event := span.Events().AppendEmpty()
		event.SetName("processing.start")
		event.SetTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(-20 * time.Second)))
		event.Attributes().PutStr("processor", fmt.Sprintf("processor-%d", i))
		
		// Add status
		if i%3 == 0 {
			span.Status().SetCode(ptrace.StatusCodeError)
			span.Status().SetMessage("Simulated error")
		} else {
			span.Status().SetCode(ptrace.StatusCodeOk)
		}
	}

	return traces
}

// GenerateLogs generates test logs
func (g *TelemetryGenerator) GenerateLogs(recordCount int) plog.Logs {
	logs := plog.NewLogs()
	rl := logs.ResourceLogs().AppendEmpty()
	
	g.resource.Attributes().Range(func(kv attribute.KeyValue) bool {
		rl.Resource().Attributes().PutStr(string(kv.Key), kv.Value.AsString())
		return true
	})

	sl := rl.ScopeLogs().AppendEmpty()
	sl.Scope().SetName("test-logger")
	sl.Scope().SetVersion("1.0.0")

	severities := []plog.SeverityNumber{
		plog.SeverityNumberDebug,
		plog.SeverityNumberInfo,
		plog.SeverityNumberWarn,
		plog.SeverityNumberError,
		plog.SeverityNumberFatal,
	}

	for i := 0; i < recordCount; i++ {
		record := sl.LogRecords().AppendEmpty()
		record.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		record.SetSeverityNumber(severities[i%len(severities)])
		record.SetSeverityText(severities[i%len(severities)].String())
		
		// Set body
		if g.includeSecrets {
			record.Body().SetStr(fmt.Sprintf("Log message %d with password=secret123 and api_key=key456", i))
		} else {
			record.Body().SetStr(fmt.Sprintf("Log message %d", i))
		}
		
		// Add attributes
		record.Attributes().PutStr("log.file", fmt.Sprintf("/var/log/app-%d.log", i%3))
		record.Attributes().PutInt("log.line", int64(100+i))
		record.Attributes().PutStr("component", fmt.Sprintf("component-%d", i%5))
		
		if g.includeSecrets {
			record.Attributes().PutStr("database_password", "db-secret-pass")
			record.Attributes().PutStr("auth_token", "Bearer secret-token-xyz")
		}
		
		// Add trace context for some logs
		if i%3 == 0 {
			record.SetTraceID(pcommon.TraceID([16]byte{byte(i), 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))
			record.SetSpanID(pcommon.SpanID([8]byte{byte(i), 2, 3, 4, 5, 6, 7, 8}))
		}
	}

	return logs
}

// Helper methods for generating specific metric types

func (g *TelemetryGenerator) generateGaugeMetric(metric pmetric.Metric, index int) {
	metric.SetName(fmt.Sprintf("test_gauge_%d", index))
	metric.SetDescription("A test gauge metric")
	metric.SetUnit("bytes")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	dp.SetDoubleValue(rand.Float64() * 1000)
	
	dp.Attributes().PutStr("gauge_type", "memory")
	dp.Attributes().PutStr("environment", "test")
	dp.Attributes().PutInt("index", int64(index))
}

func (g *TelemetryGenerator) generateSumMetric(metric pmetric.Metric, index int) {
	metric.SetName(fmt.Sprintf("test_counter_%d", index))
	metric.SetDescription("A test counter metric")
	metric.SetUnit("1")
	
	sum := metric.SetEmptySum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	
	dp := sum.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	dp.SetDoubleValue(float64(rand.Intn(1000)))
	dp.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(-time.Hour)))
	
	dp.Attributes().PutStr("counter_type", "requests")
	dp.Attributes().PutStr("method", "GET")
	dp.Attributes().PutInt("status_code", int64(200))
}

func (g *TelemetryGenerator) generateHistogramMetric(metric pmetric.Metric, index int) {
	metric.SetName(fmt.Sprintf("test_histogram_%d", index))
	metric.SetDescription("A test histogram metric")
	metric.SetUnit("ms")
	
	histogram := metric.SetEmptyHistogram()
	histogram.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	
	dp := histogram.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	dp.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(-time.Hour)))
	dp.SetCount(uint64(rand.Intn(1000)))
	dp.SetSum(rand.Float64() * 10000)
	
	// Set bucket counts and bounds
	dp.BucketCounts().FromRaw([]uint64{10, 20, 30, 40, 50})
	dp.ExplicitBounds().FromRaw([]float64{0.1, 1, 10, 100, 1000})
	
	dp.Attributes().PutStr("histogram_type", "latency")
	dp.Attributes().PutStr("operation", fmt.Sprintf("operation_%d", index%5))
}

func (g *TelemetryGenerator) generateSummaryMetric(metric pmetric.Metric, index int) {
	metric.SetName(fmt.Sprintf("test_summary_%d", index))
	metric.SetDescription("A test summary metric")
	metric.SetUnit("bytes")
	
	summary := metric.SetEmptySummary()
	
	dp := summary.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	dp.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(-time.Hour)))
	dp.SetCount(uint64(rand.Intn(1000)))
	dp.SetSum(rand.Float64() * 10000)
	
	// Add quantiles
	q1 := dp.QuantileValues().AppendEmpty()
	q1.SetQuantile(0.5)
	q1.SetValue(rand.Float64() * 100)
	
	q2 := dp.QuantileValues().AppendEmpty()
	q2.SetQuantile(0.95)
	q2.SetValue(rand.Float64() * 1000)
	
	q3 := dp.QuantileValues().AppendEmpty()
	q3.SetQuantile(0.99)
	q3.SetValue(rand.Float64() * 10000)
	
	dp.Attributes().PutStr("summary_type", "size")
	dp.Attributes().PutStr("resource", fmt.Sprintf("resource_%d", index%3))
}

// OTLPSender sends telemetry data via OTLP
type OTLPSender struct {
	endpoint       string
	logger         *zap.Logger
	metricExporter metric.Exporter
	traceExporter  trace.SpanExporter
}

// NewOTLPSender creates a new OTLP sender
func NewOTLPSender(endpoint string) (*OTLPSender, error) {
	logger, _ := zap.NewDevelopment()

	// Create GRPC connection
	conn, err := grpc.Dial(endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to OTLP endpoint: %w", err)
	}

	// Create metric exporter
	metricExporter, err := otlpmetricgrpc.New(context.Background(),
		otlpmetricgrpc.WithGRPCConn(conn),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	// Create trace exporter
	traceExporter, err := otlptracegrpc.New(context.Background(),
		otlptracegrpc.WithGRPCConn(conn),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	return &OTLPSender{
		endpoint:       endpoint,
		logger:         logger,
		metricExporter: metricExporter,
		traceExporter:  traceExporter,
	}, nil
}

// SendMetrics sends metrics via OTLP
func (s *OTLPSender) SendMetrics(ctx context.Context, metrics pmetric.Metrics) error {
	// TODO: Convert pdata metrics to OTLP and send
	return nil
}

// SendTraces sends traces via OTLP
func (s *OTLPSender) SendTraces(ctx context.Context, traces ptrace.Traces) error {
	// TODO: Convert pdata traces to OTLP and send
	return nil
}

// SendLogs sends logs via OTLP
func (s *OTLPSender) SendLogs(ctx context.Context, logs plog.Logs) error {
	// TODO: Convert pdata logs to OTLP and send
	return nil
}

// Close closes the OTLP sender
func (s *OTLPSender) Close() error {
	if s.metricExporter != nil {
		if err := s.metricExporter.Shutdown(context.Background()); err != nil {
			return err
		}
	}
	if s.traceExporter != nil {
		if err := s.traceExporter.Shutdown(context.Background()); err != nil {
			return err
		}
	}
	return nil
}