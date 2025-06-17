package supervisor

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/newrelic/nrdot-host/nrdot-supervisor/pkg/restart"
	telemetryclient "github.com/newrelic/nrdot-host/nrdot-telemetry-client"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

// Supervisor manages the OTel Collector lifecycle
type Supervisor struct {
	collector       *CollectorProcess
	healthChecker   *HealthChecker
	restartStrategy restart.Strategy
	telemetryClient telemetryclient.TelemetryClient
	logger          *zap.Logger
	config          Config
	mu              sync.Mutex
	running         bool
	stopCh          chan struct{}
	doneCh          chan struct{}
}

// Config holds supervisor configuration
type Config struct {
	Collector     CollectorConfig
	HealthChecker HealthCheckerConfig
	Restart       restart.Config
	Telemetry     telemetryclient.Config
}

// DefaultConfig returns default supervisor configuration
func DefaultConfig() Config {
	return Config{
		Collector:     DefaultCollectorConfig(),
		HealthChecker: DefaultHealthCheckerConfig(),
		Restart:       restart.DefaultConfig(),
		Telemetry:     telemetryclient.Config{
			Enabled:  true,
			Interval: 10 * time.Second,
		},
	}
}

// New creates a new supervisor
func New(config Config, logger *zap.Logger) (*Supervisor, error) {
	// Create telemetry client
	telemetryClient, err := telemetryclient.NewTelemetryClient(config.Telemetry, logger)
	if err != nil {
		return nil, fmt.Errorf("creating telemetry client: %w", err)
	}

	// Create restart strategy
	restartFactory := restart.NewFactory(config.Restart)
	
	return &Supervisor{
		collector:       NewCollectorProcess(config.Collector, logger),
		healthChecker:   NewHealthChecker(config.HealthChecker, logger),
		restartStrategy: restartFactory.Create(),
		telemetryClient: telemetryClient,
		logger:          logger,
		config:          config,
		stopCh:          make(chan struct{}),
		doneCh:          make(chan struct{}),
	}, nil
}

// Start starts the supervisor
func (s *Supervisor) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("supervisor already running")
	}
	s.running = true
	s.mu.Unlock()

	s.logger.Info("Starting supervisor")

	// Telemetry client starts automatically with NewTelemetryClient

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

	// Start supervision loop
	go s.supervise(ctx, sigCh)

	return nil
}

// Stop stops the supervisor
func (s *Supervisor) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	s.mu.Unlock()

	s.logger.Info("Stopping supervisor")

	// Signal stop
	close(s.stopCh)

	// Wait for supervision loop to finish
	select {
	case <-s.doneCh:
	case <-ctx.Done():
		return ctx.Err()
	}

	// Stop collector
	if err := s.collector.Stop(ctx); err != nil {
		s.logger.Error("Error stopping collector", zap.Error(err))
	}

	// Shutdown telemetry client
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.telemetryClient.Shutdown(shutdownCtx); err != nil {
		s.logger.Error("Error shutting down telemetry client", zap.Error(err))
	}

	s.logger.Info("Supervisor stopped")
	return nil
}

// supervise is the main supervision loop
func (s *Supervisor) supervise(ctx context.Context, sigCh <-chan os.Signal) {
	defer close(s.doneCh)

	// Start collector initially
	if err := s.startCollector(ctx); err != nil {
		s.logger.Error("Failed to start collector initially", zap.Error(err))
		s.reportMetric("supervisor.start.failed", 1, map[string]string{"reason": "initial_start"})
		return
	}

	// Start health monitoring
	healthErrCh := s.healthChecker.Monitor(ctx)

	// Start memory monitoring
	memoryCheckTicker := time.NewTicker(30 * time.Second)
	defer memoryCheckTicker.Stop()

	for {
		select {
		case <-s.stopCh:
			s.logger.Info("Supervisor stop requested")
			return

		case <-ctx.Done():
			s.logger.Info("Context cancelled")
			return

		case sig := <-sigCh:
			s.handleSignal(ctx, sig)

		case err := <-healthErrCh:
			if err != nil {
				s.logger.Error("Health check error", zap.Error(err))
				s.reportMetric("supervisor.health_check.failed", 1, nil)
				s.handleCollectorFailure(ctx, "health_check_failed")
			}

		case <-memoryCheckTicker.C:
			s.checkMemoryUsage(ctx)
		}

		// Check if collector exited unexpectedly
		if !s.collector.IsRunning() {
			s.logger.Warn("Collector process exited unexpectedly")
			s.reportMetric("supervisor.collector.unexpected_exit", 1, nil)
			s.handleCollectorFailure(ctx, "unexpected_exit")
		}
	}
}

// startCollector starts the collector process
func (s *Supervisor) startCollector(ctx context.Context) error {
	s.logger.Info("Starting collector process")
	
	startTime := time.Now()
	if err := s.collector.Start(ctx); err != nil {
		s.reportMetric("supervisor.collector.start.failed", 1, nil)
		return err
	}

	// Wait for collector to become healthy
	if err := s.healthChecker.WaitForHealthy(ctx, 30*time.Second); err != nil {
		s.logger.Error("Collector failed to become healthy", zap.Error(err))
		s.collector.Stop(ctx)
		s.reportMetric("supervisor.collector.start.unhealthy", 1, nil)
		return err
	}

	s.restartStrategy.RecordSuccess()
	s.reportMetric("supervisor.collector.start.success", 1, nil)
	s.reportMetric("supervisor.collector.start.duration", time.Since(startTime).Seconds(), nil)
	
	return nil
}

// handleCollectorFailure handles collector failures
func (s *Supervisor) handleCollectorFailure(ctx context.Context, reason string) {
	s.restartStrategy.RecordFailure()
	s.reportMetric("supervisor.collector.failure", 1, map[string]string{"reason": reason})

	// Check restart strategy
	delay, shouldRestart := s.restartStrategy.NextDelay()
	if !shouldRestart {
		s.logger.Error("Restart strategy exhausted, not restarting collector")
		s.reportMetric("supervisor.restart.exhausted", 1, nil)
		return
	}

	s.logger.Info("Restarting collector",
		zap.Duration("delay", delay),
		zap.String("reason", reason),
	)

	// Stop collector if still running
	if err := s.collector.Stop(ctx); err != nil {
		s.logger.Error("Error stopping failed collector", zap.Error(err))
	}

	// Wait for restart delay
	if err := restart.WaitForRestart(ctx, delay); err != nil {
		s.logger.Error("Error waiting for restart", zap.Error(err))
		return
	}

	// Restart collector
	if err := s.startCollector(ctx); err != nil {
		s.logger.Error("Failed to restart collector", zap.Error(err))
		s.reportMetric("supervisor.restart.failed", 1, nil)
		// Will retry on next iteration
	} else {
		s.reportMetric("supervisor.restart.success", 1, nil)
	}
}

// handleSignal handles OS signals
func (s *Supervisor) handleSignal(ctx context.Context, sig os.Signal) {
	switch sig {
	case syscall.SIGTERM, syscall.SIGINT:
		s.logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
		s.Stop(ctx)

	case syscall.SIGHUP:
		s.logger.Info("Received reload signal")
		s.reportMetric("supervisor.reload.requested", 1, nil)
		
		// Forward signal to collector for config reload
		if err := s.collector.Signal(sig); err != nil {
			s.logger.Error("Failed to forward reload signal", zap.Error(err))
			s.reportMetric("supervisor.reload.failed", 1, nil)
		} else {
			s.reportMetric("supervisor.reload.success", 1, nil)
		}
	}
}

// checkMemoryUsage checks collector memory usage
func (s *Supervisor) checkMemoryUsage(ctx context.Context) {
	exceeded, usage, err := s.collector.CheckMemoryLimit()
	if err != nil {
		s.logger.Debug("Failed to check memory usage", zap.Error(err))
		return
	}

	// Report memory usage metric
	s.reportMetric("supervisor.collector.memory.usage", float64(usage), nil)

	if exceeded {
		s.logger.Warn("Collector memory limit exceeded",
			zap.Uint64("usage", usage),
			zap.Uint64("limit", s.config.Collector.MemoryLimit),
		)
		s.reportMetric("supervisor.collector.memory.limit_exceeded", 1, nil)
		s.handleCollectorFailure(ctx, "memory_limit_exceeded")
	}
}

// reportMetric reports a metric via telemetry client
func (s *Supervisor) reportMetric(name string, value float64, tags map[string]string) {
	// Convert tags to attributes
	attrs := make([]attribute.KeyValue, 0, len(tags))
	for k, v := range tags {
		attrs = append(attrs, attribute.String(k, v))
	}
	
	if err := s.telemetryClient.RecordMetric(name, value, attrs...); err != nil {
		s.logger.Debug("Failed to send metric",
			zap.String("metric", name),
			zap.Error(err),
		)
	}
}

// Wait waits for the supervisor to finish
func (s *Supervisor) Wait() {
	<-s.doneCh
}