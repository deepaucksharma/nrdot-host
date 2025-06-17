package telemetryclient

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// TelemetryClient provides self-monitoring capabilities for NRDOT
type TelemetryClient interface {
	// Health monitoring
	RecordHealth(sample HealthSample) error
	RecordRestart(reason string) error
	RecordConfigChange(oldVersion, newVersion string) error
	RecordFeatureFlag(name string, enabled bool) error
	
	// Metrics
	RecordMetric(name string, value float64, attributes ...attribute.KeyValue) error
	IncrementCounter(name string, attributes ...attribute.KeyValue) error
	RecordDuration(name string, duration time.Duration, attributes ...attribute.KeyValue) error
	
	// Shutdown
	Shutdown(ctx context.Context) error
}

// HealthSample represents a health check data point
type HealthSample struct {
	Timestamp    time.Time
	CPUPercent   float64
	MemoryMB     float64
	GoroutineNum int
	ErrorCount   int64
	RestartCount int64
	Uptime       time.Duration
	Version      string
	ConfigHash   string
}

// Config holds the telemetry client configuration
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	Endpoint       string
	APIKey         string
	Interval       time.Duration
	Enabled        bool
}

// client implements TelemetryClient
type client struct {
	config      Config
	logger      *zap.Logger
	tracer      trace.Tracer
	meter       metric.Meter
	resource    *resource.Resource
	
	// Metrics
	healthGauge       metric.Float64ObservableGauge
	restartCounter    metric.Int64Counter
	configChangeCount metric.Int64Counter
	errorCounter      metric.Int64Counter
	featureFlagGauge  metric.Float64ObservableGauge
	
	// State
	mu              sync.RWMutex
	lastHealth      HealthSample
	featureFlags    map[string]bool
	restartReasons  []string
	configVersions  []string
}

// NewTelemetryClient creates a new telemetry client
func NewTelemetryClient(config Config, logger *zap.Logger) (TelemetryClient, error) {
	if !config.Enabled {
		return &noopClient{}, nil
	}

	// Create resource
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			attribute.String("service.name", config.ServiceName),
			attribute.String("service.version", config.ServiceVersion),
			attribute.String("deployment.environment", config.Environment),
		),
		resource.WithHost(),
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithContainer(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Initialize OpenTelemetry
	tracerProvider, err := initTracer(config, res)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracer: %w", err)
	}
	otel.SetTracerProvider(tracerProvider)

	meterProvider, err := initMeter(config, res)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize meter: %w", err)
	}
	otel.SetMeterProvider(meterProvider)

	// Create client
	c := &client{
		config:         config,
		logger:         logger,
		tracer:         otel.Tracer("nrdot-telemetry"),
		meter:          otel.Meter("nrdot-telemetry"),
		resource:       res,
		featureFlags:   make(map[string]bool),
		restartReasons: make([]string, 0),
		configVersions: make([]string, 0),
	}

	// Initialize metrics
	if err := c.initMetrics(); err != nil {
		return nil, fmt.Errorf("failed to initialize metrics: %w", err)
	}

	return c, nil
}

// initMetrics initializes all metric instruments
func (c *client) initMetrics() error {
	var err error

	// Health gauge (observable)
	c.healthGauge, err = c.meter.Float64ObservableGauge(
		"nrdot.health",
		metric.WithDescription("NRDOT health metrics"),
		metric.WithFloat64Callback(c.observeHealth),
	)
	if err != nil {
		return fmt.Errorf("failed to create health gauge: %w", err)
	}

	// Restart counter
	c.restartCounter, err = c.meter.Int64Counter(
		"nrdot.restarts",
		metric.WithDescription("Number of NRDOT restarts"),
	)
	if err != nil {
		return fmt.Errorf("failed to create restart counter: %w", err)
	}

	// Config change counter
	c.configChangeCount, err = c.meter.Int64Counter(
		"nrdot.config.changes",
		metric.WithDescription("Number of configuration changes"),
	)
	if err != nil {
		return fmt.Errorf("failed to create config change counter: %w", err)
	}

	// Error counter
	c.errorCounter, err = c.meter.Int64Counter(
		"nrdot.errors",
		metric.WithDescription("Number of errors encountered"),
	)
	if err != nil {
		return fmt.Errorf("failed to create error counter: %w", err)
	}

	// Feature flag gauge (observable)
	c.featureFlagGauge, err = c.meter.Float64ObservableGauge(
		"nrdot.feature.flag",
		metric.WithDescription("Feature flag states (1=enabled, 0=disabled)"),
		metric.WithFloat64Callback(c.observeFeatureFlags),
	)
	if err != nil {
		return fmt.Errorf("failed to create feature flag gauge: %w", err)
	}

	return nil
}

// observeHealth is the callback for health metrics
func (c *client) observeHealth(_ context.Context, observer metric.Float64Observer) error {
	c.mu.RLock()
	health := c.lastHealth
	c.mu.RUnlock()

	observer.Observe(health.CPUPercent, metric.WithAttributes(
		attribute.String("metric", "cpu_percent"),
	))
	observer.Observe(health.MemoryMB, metric.WithAttributes(
		attribute.String("metric", "memory_mb"),
	))
	observer.Observe(float64(health.GoroutineNum), metric.WithAttributes(
		attribute.String("metric", "goroutine_count"),
	))
	observer.Observe(float64(health.ErrorCount), metric.WithAttributes(
		attribute.String("metric", "error_count"),
	))
	observer.Observe(health.Uptime.Seconds(), metric.WithAttributes(
		attribute.String("metric", "uptime_seconds"),
	))

	return nil
}

// observeFeatureFlags is the callback for feature flag metrics
func (c *client) observeFeatureFlags(_ context.Context, observer metric.Float64Observer) error {
	c.mu.RLock()
	flags := make(map[string]bool)
	for k, v := range c.featureFlags {
		flags[k] = v
	}
	c.mu.RUnlock()

	for name, enabled := range flags {
		value := 0.0
		if enabled {
			value = 1.0
		}
		observer.Observe(value, metric.WithAttributes(
			attribute.String("flag_name", name),
		))
	}

	return nil
}

// RecordHealth records a health sample
func (c *client) RecordHealth(sample HealthSample) error {
	c.mu.Lock()
	c.lastHealth = sample
	c.mu.Unlock()

	// Also record as an event
	ctx := context.Background()
	_, span := c.tracer.Start(ctx, "health.check")
	defer span.End()

	span.SetAttributes(
		attribute.Float64("cpu_percent", sample.CPUPercent),
		attribute.Float64("memory_mb", sample.MemoryMB),
		attribute.Int("goroutine_count", sample.GoroutineNum),
		attribute.Int64("error_count", sample.ErrorCount),
		attribute.Int64("restart_count", sample.RestartCount),
		attribute.Float64("uptime_seconds", sample.Uptime.Seconds()),
		attribute.String("version", sample.Version),
		attribute.String("config_hash", sample.ConfigHash),
	)

	return nil
}

// RecordRestart records a restart event
func (c *client) RecordRestart(reason string) error {
	ctx := context.Background()
	c.restartCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("reason", reason),
	))

	c.mu.Lock()
	c.restartReasons = append(c.restartReasons, fmt.Sprintf("%s: %s", time.Now().Format(time.RFC3339), reason))
	// Keep only last 10 reasons
	if len(c.restartReasons) > 10 {
		c.restartReasons = c.restartReasons[len(c.restartReasons)-10:]
	}
	c.mu.Unlock()

	// Also create a span for the restart event
	_, span := c.tracer.Start(ctx, "nrdot.restart")
	defer span.End()
	span.SetAttributes(attribute.String("reason", reason))

	c.logger.Info("Recorded restart", zap.String("reason", reason))
	return nil
}

// RecordConfigChange records a configuration change
func (c *client) RecordConfigChange(oldVersion, newVersion string) error {
	ctx := context.Background()
	c.configChangeCount.Add(ctx, 1, metric.WithAttributes(
		attribute.String("old_version", oldVersion),
		attribute.String("new_version", newVersion),
	))

	c.mu.Lock()
	c.configVersions = append(c.configVersions, newVersion)
	// Keep only last 10 versions
	if len(c.configVersions) > 10 {
		c.configVersions = c.configVersions[len(c.configVersions)-10:]
	}
	c.mu.Unlock()

	// Create a span for the config change
	_, span := c.tracer.Start(ctx, "nrdot.config.change")
	defer span.End()
	span.SetAttributes(
		attribute.String("old_version", oldVersion),
		attribute.String("new_version", newVersion),
	)

	c.logger.Info("Recorded config change", 
		zap.String("old_version", oldVersion),
		zap.String("new_version", newVersion))
	return nil
}

// RecordFeatureFlag records a feature flag state
func (c *client) RecordFeatureFlag(name string, enabled bool) error {
	c.mu.Lock()
	c.featureFlags[name] = enabled
	c.mu.Unlock()

	c.logger.Debug("Recorded feature flag", 
		zap.String("name", name),
		zap.Bool("enabled", enabled))
	return nil
}

// RecordMetric records a generic metric value
func (c *client) RecordMetric(name string, value float64, attributes ...attribute.KeyValue) error {
	// Use a histogram for generic metrics
	histogram, err := c.meter.Float64Histogram(
		fmt.Sprintf("nrdot.custom.%s", name),
		metric.WithDescription("Custom metric"),
	)
	if err != nil {
		return fmt.Errorf("failed to create histogram: %w", err)
	}

	ctx := context.Background()
	histogram.Record(ctx, value, metric.WithAttributes(attributes...))
	return nil
}

// IncrementCounter increments a counter metric
func (c *client) IncrementCounter(name string, attributes ...attribute.KeyValue) error {
	counter, err := c.meter.Int64Counter(
		fmt.Sprintf("nrdot.custom.%s", name),
		metric.WithDescription("Custom counter"),
	)
	if err != nil {
		return fmt.Errorf("failed to create counter: %w", err)
	}

	ctx := context.Background()
	counter.Add(ctx, 1, metric.WithAttributes(attributes...))
	return nil
}

// RecordDuration records a duration metric
func (c *client) RecordDuration(name string, duration time.Duration, attributes ...attribute.KeyValue) error {
	histogram, err := c.meter.Float64Histogram(
		fmt.Sprintf("nrdot.duration.%s", name),
		metric.WithDescription("Duration metric"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return fmt.Errorf("failed to create histogram: %w", err)
	}

	ctx := context.Background()
	histogram.Record(ctx, float64(duration.Milliseconds()), metric.WithAttributes(attributes...))
	return nil
}

// Shutdown gracefully shuts down the telemetry client
func (c *client) Shutdown(ctx context.Context) error {
	// The actual shutdown is handled by the providers
	// This is a placeholder for any cleanup needed
	c.logger.Info("Shutting down telemetry client")
	return nil
}

// noopClient is a no-op implementation for when telemetry is disabled
type noopClient struct{}

func (n *noopClient) RecordHealth(sample HealthSample) error                        { return nil }
func (n *noopClient) RecordRestart(reason string) error                             { return nil }
func (n *noopClient) RecordConfigChange(oldVersion, newVersion string) error        { return nil }
func (n *noopClient) RecordFeatureFlag(name string, enabled bool) error             { return nil }
func (n *noopClient) RecordMetric(name string, value float64, attributes ...attribute.KeyValue) error { return nil }
func (n *noopClient) IncrementCounter(name string, attributes ...attribute.KeyValue) error { return nil }
func (n *noopClient) RecordDuration(name string, duration time.Duration, attributes ...attribute.KeyValue) error { return nil }
func (n *noopClient) Shutdown(ctx context.Context) error                            { return nil }