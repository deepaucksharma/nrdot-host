package nrtransform

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

// nrTransformProcessor implements the metrics processor
type nrTransformProcessor struct {
	config      *Config
	logger      *zap.Logger
	transformer *Transformer
}

// newProcessor creates a new processor
func newProcessor(cfg *Config, logger *zap.Logger) *nrTransformProcessor {
	return &nrTransformProcessor{
		config: cfg,
		logger: logger,
	}
}

// processMetrics processes the metrics
func (p *nrTransformProcessor) processMetrics(ctx context.Context, metrics pmetric.Metrics) (pmetric.Metrics, error) {
	// Initialize transformer if not already done
	if p.transformer == nil {
		transformer, err := NewTransformer(p.config, p.logger)
		if err != nil {
			return metrics, fmt.Errorf("failed to create transformer: %w", err)
		}
		p.transformer = transformer
	}

	// Apply transformations
	if err := p.transformer.Transform(metrics); err != nil {
		return metrics, fmt.Errorf("failed to transform metrics: %w", err)
	}

	return metrics, nil
}