package nrcap

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
)

var errInvalidConfig = errors.New("invalid configuration")

const (
	// typeStr is the type string for this processor
	typeStr = "nrcap"
	// stability is the stability level of this processor
	stability = component.StabilityLevelBeta
)

// NewFactory returns a new factory for the cardinality protection processor
func NewFactory() processor.Factory {
	return processor.NewFactory(
		typeStr,
		createDefaultConfig,
		processor.WithMetrics(createMetricsProcessor, stability),
	)
}

// createMetricsProcessor creates a metrics processor
func createMetricsProcessor(
	ctx context.Context,
	set processor.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	processorCfg, ok := cfg.(*Config)
	if !ok {
		return nil, errInvalidConfig
	}

	if err := processorCfg.Validate(); err != nil {
		return nil, err
	}

	proc, err := newCapProcessor(cfg, set.Logger, nextConsumer)
	if err != nil {
		return nil, err
	}

	return proc, nil
}