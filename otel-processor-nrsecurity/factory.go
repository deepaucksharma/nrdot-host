package nrsecurity

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

const (
	// TypeStr is the type string for this processor
	TypeStr = "nrsecurity"
	// stability is the stability level of this processor
	stability = component.StabilityLevelBeta
)

// NewFactory creates a new processor factory
func NewFactory() processor.Factory {
	return processor.NewFactory(
		TypeStr,
		createDefaultConfig,
		processor.WithTraces(createTracesProcessor, stability),
		processor.WithMetrics(createMetricsProcessor, stability),
		processor.WithLogs(createLogsProcessor, stability),
	)
}

// createTracesProcessor creates a traces processor
func createTracesProcessor(
	ctx context.Context,
	set processor.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error) {
	p, err := newProcessor(cfg, set.Logger)
	if err != nil {
		return nil, err
	}
	
	return processorhelper.NewTracesProcessor(
		ctx,
		set,
		cfg,
		nextConsumer,
		func(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
			return p.processTraces(ctx, td)
		},
		processorhelper.WithStart(p.Start),
		processorhelper.WithShutdown(p.Shutdown),
		processorhelper.WithCapabilities(p.Capabilities()),
	)
}

// createMetricsProcessor creates a metrics processor
func createMetricsProcessor(
	ctx context.Context,
	set processor.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	p, err := newProcessor(cfg, set.Logger)
	if err != nil {
		return nil, err
	}
	
	return processorhelper.NewMetricsProcessor(
		ctx,
		set,
		cfg,
		nextConsumer,
		func(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
			return p.processMetrics(ctx, md)
		},
		processorhelper.WithStart(p.Start),
		processorhelper.WithShutdown(p.Shutdown),
		processorhelper.WithCapabilities(p.Capabilities()),
	)
}

// createLogsProcessor creates a logs processor
func createLogsProcessor(
	ctx context.Context,
	set processor.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	p, err := newProcessor(cfg, set.Logger)
	if err != nil {
		return nil, err
	}
	
	return processorhelper.NewLogsProcessor(
		ctx,
		set,
		cfg,
		nextConsumer,
		func(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
			return p.processLogs(ctx, ld)
		},
		processorhelper.WithStart(p.Start),
		processorhelper.WithShutdown(p.Shutdown),
		processorhelper.WithCapabilities(p.Capabilities()),
	)
}