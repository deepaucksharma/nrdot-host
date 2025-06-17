package nrenrich

import (
	"context"
	"fmt"

	// common "github.com/newrelic/nrdot-host/otel-processor-common"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.uber.org/zap"
)

// nrenrichProcessor implements the OpenTelemetry processor interface
type nrenrichProcessor struct {
	config   *Config
	logger   *zap.Logger
	enricher *Enricher
	telemetrySettings component.TelemetrySettings
}

// newTracesProcessor creates a new traces processor
func newTracesProcessor(
	ctx context.Context,
	set processor.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error) {
	p, err := newProcessor(cfg, set)
	if err != nil {
		return nil, err
	}

	return processorhelper.NewTracesProcessor(
		ctx,
		set,
		cfg,
		nextConsumer,
		p.processTraces,
		processorhelper.WithCapabilities(consumer.Capabilities{MutatesData: true}),
		processorhelper.WithStart(p.start),
		processorhelper.WithShutdown(p.shutdown),
	)
}

// newMetricsProcessor creates a new metrics processor
func newMetricsProcessor(
	ctx context.Context,
	set processor.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	p, err := newProcessor(cfg, set)
	if err != nil {
		return nil, err
	}

	return processorhelper.NewMetricsProcessor(
		ctx,
		set,
		cfg,
		nextConsumer,
		p.processMetrics,
		processorhelper.WithCapabilities(consumer.Capabilities{MutatesData: true}),
		processorhelper.WithStart(p.start),
		processorhelper.WithShutdown(p.shutdown),
	)
}

// newLogsProcessor creates a new logs processor
func newLogsProcessor(
	ctx context.Context,
	set processor.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	p, err := newProcessor(cfg, set)
	if err != nil {
		return nil, err
	}

	return processorhelper.NewLogsProcessor(
		ctx,
		set,
		cfg,
		nextConsumer,
		p.processLogs,
		processorhelper.WithCapabilities(consumer.Capabilities{MutatesData: true}),
		processorhelper.WithStart(p.start),
		processorhelper.WithShutdown(p.shutdown),
	)
}

// newProcessor creates a new processor instance
func newProcessor(cfg component.Config, set processor.CreateSettings) (*nrenrichProcessor, error) {
	config := cfg.(*Config)

	enricher, err := NewEnricher(config, set.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create enricher: %w", err)
	}

	return &nrenrichProcessor{
		config:            config,
		logger:            set.Logger,
		enricher:          enricher,
		telemetrySettings: set.TelemetrySettings,
	}, nil
}

// start starts the processor
func (p *nrenrichProcessor) start(ctx context.Context, host component.Host) error {
	p.logger.Info("Starting nrenrich processor")
	return nil
}

// shutdown shuts down the processor
func (p *nrenrichProcessor) shutdown(ctx context.Context) error {
	p.logger.Info("Shutting down nrenrich processor")
	return p.enricher.Close()
}

// processTraces processes and enriches traces
func (p *nrenrichProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	if td.SpanCount() == 0 {
		return td, nil
	}

	if err := p.enricher.EnrichTraces(ctx, td); err != nil {
		p.logger.Error("Failed to enrich traces", zap.Error(err))
		// Continue processing even if enrichment fails
	}

	return td, nil
}

// processMetrics processes and enriches metrics
func (p *nrenrichProcessor) processMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	if md.MetricCount() == 0 {
		return md, nil
	}

	if err := p.enricher.EnrichMetrics(ctx, md); err != nil {
		p.logger.Error("Failed to enrich metrics", zap.Error(err))
		// Continue processing even if enrichment fails
	}

	return md, nil
}

// processLogs processes and enriches logs
func (p *nrenrichProcessor) processLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	if ld.LogRecordCount() == 0 {
		return ld, nil
	}

	if err := p.enricher.EnrichLogs(ctx, ld); err != nil {
		p.logger.Error("Failed to enrich logs", zap.Error(err))
		// Continue processing even if enrichment fails
	}

	return ld, nil
}

// TODO: Implement common.Processor interface when available
// var _ common.Processor = (*nrenrichProcessor)(nil)

// GetConfig returns the processor configuration
func (p *nrenrichProcessor) GetConfig() component.Config {
	return p.config
}

// GetLogger returns the processor logger
func (p *nrenrichProcessor) GetLogger() *zap.Logger {
	return p.logger
}