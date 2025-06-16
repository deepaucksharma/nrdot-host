package common

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

// BaseProcessor defines the common interface for all NRDOT processors
type BaseProcessor interface {
	component.Component
	GetCapabilities() Capabilities
}

// Capabilities describes what a processor can handle
type Capabilities struct {
	SupportsMetrics bool
	SupportsTraces  bool
	SupportsLogs    bool
}

// ProcessorConfig is the base configuration for all processors
type ProcessorConfig struct {
	// Enabled allows disabling the processor
	Enabled bool `mapstructure:"enabled"`
	
	// ErrorMode determines how to handle processing errors
	ErrorMode ErrorMode `mapstructure:"error_mode"`
	
	// Timeout for processing operations
	Timeout time.Duration `mapstructure:"timeout"`
}

// ErrorMode defines how processors handle errors
type ErrorMode string

const (
	// ErrorModePropagateError stops processing and returns the error
	ErrorModePropagateError ErrorMode = "propagate"
	
	// ErrorModeIgnore logs the error and continues processing
	ErrorModeIgnore ErrorMode = "ignore"
	
	// ErrorModeSilent continues processing without logging
	ErrorModeSilent ErrorMode = "silent"
)

// BaseProcessorImpl provides common functionality for all processors
type BaseProcessorImpl struct {
	logger       *zap.Logger
	capabilities Capabilities
	config       ProcessorConfig
	telemetry    component.TelemetrySettings
}

// NewBaseProcessor creates a new base processor instance
func NewBaseProcessor(logger *zap.Logger, config ProcessorConfig, telemetry component.TelemetrySettings) *BaseProcessorImpl {
	return &BaseProcessorImpl{
		logger:    logger,
		config:    config,
		telemetry: telemetry,
	}
}

// Start implements component.Component
func (p *BaseProcessorImpl) Start(_ context.Context, _ component.Host) error {
	if !p.config.Enabled {
		p.logger.Info("Processor is disabled")
		return nil
	}
	p.logger.Info("Starting processor")
	return nil
}

// Shutdown implements component.Component
func (p *BaseProcessorImpl) Shutdown(_ context.Context) error {
	p.logger.Info("Shutting down processor")
	return nil
}

// GetCapabilities returns the processor's capabilities
func (p *BaseProcessorImpl) GetCapabilities() Capabilities {
	return p.capabilities
}

// HandleError processes errors according to the configured error mode
func (p *BaseProcessorImpl) HandleError(err error, message string) error {
	if err == nil {
		return nil
	}

	switch p.config.ErrorMode {
	case ErrorModePropagateError:
		p.logger.Error(message, zap.Error(err))
		return err
	case ErrorModeIgnore:
		p.logger.Warn(message, zap.Error(err))
		return nil
	case ErrorModeSilent:
		// Do nothing
		return nil
	default:
		p.logger.Error(message, zap.Error(err))
		return err
	}
}

// WithTimeout executes a function with the configured timeout
func (p *BaseProcessorImpl) WithTimeout(ctx context.Context, fn func(context.Context) error) error {
	if p.config.Timeout <= 0 {
		return fn(ctx)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, p.config.Timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- fn(timeoutCtx)
	}()

	select {
	case err := <-done:
		return err
	case <-timeoutCtx.Done():
		return errors.New("processor operation timed out")
	}
}

// MetricProcessor is the base for metric processors
type MetricProcessor interface {
	BaseProcessor
	consumer.Metrics
	ProcessMetrics(context.Context, pmetric.Metrics) (pmetric.Metrics, error)
}

// TraceProcessor is the base for trace processors
type TraceProcessor interface {
	BaseProcessor
	consumer.Traces
	ProcessTraces(context.Context, ptrace.Traces) (ptrace.Traces, error)
}

// LogProcessor is the base for log processors
type LogProcessor interface {
	BaseProcessor
	consumer.Logs
	ProcessLogs(context.Context, plog.Logs) (plog.Logs, error)
}