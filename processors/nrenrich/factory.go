package nrenrich

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	// "go.opentelemetry.io/collector/processor/processorhelper"
)

const (
	// typeStr is the type name of the processor
	typeStr = "nrenrich"
	// stability is the stability level of the processor
	stability = component.StabilityLevelBeta
)

// NewFactory creates a new processor factory
func NewFactory() processor.Factory {
	return processor.NewFactory(
		typeStr,
		createDefaultConfig,
		processor.WithTraces(createTracesProcessor, stability),
		processor.WithMetrics(createMetricsProcessor, stability),
		processor.WithLogs(createLogsProcessor, stability),
	)
}

// createDefaultConfig creates the default configuration for the processor
func createDefaultConfig() component.Config {
	return &Config{
		StaticAttributes: make(map[string]interface{}),
		Environment: EnvironmentConfig{
			Enabled:       true,
			Hostname:      true,
			CloudProvider: true,
			Kubernetes:    true,
			System:        true,
		},
		Process: ProcessConfig{
			Enabled:        false,
			HelperEndpoint: "unix:///var/run/nrdot/helper.sock",
			Timeout:        5 * time.Second,
		},
		Rules:   []EnrichmentRule{},
		Dynamic: []DynamicAttribute{},
		Cache: CacheConfig{
			TTL:     5 * time.Minute,
			MaxSize: 1000,
		},
	}
}

// createTracesProcessor creates a traces processor
func createTracesProcessor(
	ctx context.Context,
	set processor.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error) {
	return newTracesProcessor(ctx, set, cfg, nextConsumer)
}

// createMetricsProcessor creates a metrics processor
func createMetricsProcessor(
	ctx context.Context,
	set processor.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	return newMetricsProcessor(ctx, set, cfg, nextConsumer)
}

// createLogsProcessor creates a logs processor
func createLogsProcessor(
	ctx context.Context,
	set processor.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	return newLogsProcessor(ctx, set, cfg, nextConsumer)
}