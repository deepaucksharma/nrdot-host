package nrcap

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

// capProcessor implements the cardinality protection processor
type capProcessor struct {
	component.StartFunc
	component.ShutdownFunc

	config *Config
	logger *zap.Logger

	limiter      *CardinalityLimiter
	nextConsumer consumer.Metrics

	// Reset ticker
	resetTicker *time.Ticker
	stopCh      chan struct{}
	wg          sync.WaitGroup

	// Stats reporting
	statsTicker *time.Ticker
}

// newCapProcessor creates a new processor instance
func newCapProcessor(cfg component.Config, logger *zap.Logger, nextConsumer consumer.Metrics) (*capProcessor, error) {
	processorCfg, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("invalid config type: %T", cfg)
	}

	return &capProcessor{
		config:       processorCfg,
		logger:       logger,
		limiter:      NewCardinalityLimiter(processorCfg, logger),
		nextConsumer: nextConsumer,
		stopCh:       make(chan struct{}),
	}, nil
}

// Capabilities returns the capabilities of the processor
func (p *capProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

// Start starts the processor
func (p *capProcessor) Start(ctx context.Context, host component.Host) error {
	p.logger.Info("Starting cardinality protection processor",
		zap.Int("global_limit", p.config.GlobalLimit),
		zap.String("strategy", string(p.config.Strategy)))

	// Start reset ticker
	p.resetTicker = time.NewTicker(p.config.ResetInterval)
	p.wg.Add(1)
	go p.resetLoop()

	// Start stats ticker if enabled
	if p.config.EnableStats {
		p.statsTicker = time.NewTicker(1 * time.Minute)
		p.wg.Add(1)
		go p.statsLoop()
	}

	return nil
}

// Shutdown shuts down the processor
func (p *capProcessor) Shutdown(ctx context.Context) error {
	p.logger.Info("Shutting down cardinality protection processor")

	close(p.stopCh)

	if p.resetTicker != nil {
		p.resetTicker.Stop()
	}
	if p.statsTicker != nil {
		p.statsTicker.Stop()
	}

	// Wait for goroutines to finish
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ConsumeMetrics processes metrics
func (p *capProcessor) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	// Apply cardinality protection
	protected, err := p.limiter.ProcessMetrics(md)
	if err != nil {
		return fmt.Errorf("failed to process metrics: %w", err)
	}

	// Pass to next consumer
	return p.nextConsumer.ConsumeMetrics(ctx, protected)
}

// resetLoop handles periodic resets
func (p *capProcessor) resetLoop() {
	defer p.wg.Done()

	for {
		select {
		case <-p.resetTicker.C:
			p.logger.Info("Resetting cardinality tracker")
			p.limiter.Reset()
		case <-p.stopCh:
			return
		}
	}
}

// statsLoop handles periodic stats reporting
func (p *capProcessor) statsLoop() {
	defer p.wg.Done()

	for {
		select {
		case <-p.statsTicker.C:
			stats := p.limiter.GetStats()
			p.logger.Info("Cardinality protection statistics",
				zap.Int64("total_metrics", stats.TotalMetrics),
				zap.Int64("dropped_metrics", stats.DroppedMetrics),
				zap.Int64("aggregated_metrics", stats.AggregatedMetrics),
				zap.Int64("sampled_metrics", stats.SampledMetrics),
				zap.Time("last_reset", stats.LastReset))

			// Log high cardinality metrics
			for metric, cardinality := range stats.MetricCardinalities {
				if cardinality > p.config.DefaultLimit {
					p.logger.Warn("High cardinality metric detected",
						zap.String("metric", metric),
						zap.Int("cardinality", cardinality),
						zap.Int("limit", p.getMetricLimit(metric)))
				}
			}

			// Log high cardinality labels
			for label, cardinality := range stats.HighCardinalityLabels {
				if cardinality > 100 {
					p.logger.Warn("High cardinality label detected",
						zap.String("label", label),
						zap.Int("unique_values", cardinality))
				}
			}
		case <-p.stopCh:
			return
		}
	}
}

// getMetricLimit returns the limit for a specific metric
func (p *capProcessor) getMetricLimit(metricName string) int {
	if limit, exists := p.config.MetricLimits[metricName]; exists {
		return limit
	}
	return p.config.DefaultLimit
}

// Ensure capProcessor implements the necessary interfaces
var (
	_ processor.Metrics   = (*capProcessor)(nil)
	_ component.Component = (*capProcessor)(nil)
)