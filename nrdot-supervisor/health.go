package supervisor

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// HealthChecker monitors the health of the OTel Collector
type HealthChecker struct {
	endpoint        string
	interval        time.Duration
	timeout         time.Duration
	failureThreshold int
	logger          *zap.Logger
	client          *http.Client
}

// HealthCheckerConfig holds health checker configuration
type HealthCheckerConfig struct {
	Endpoint         string
	Interval         time.Duration
	Timeout          time.Duration
	FailureThreshold int
}

// DefaultHealthCheckerConfig returns default health checker configuration
func DefaultHealthCheckerConfig() HealthCheckerConfig {
	return HealthCheckerConfig{
		Endpoint:         "http://localhost:13133/health",
		Interval:         10 * time.Second,
		Timeout:          5 * time.Second,
		FailureThreshold: 3,
	}
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(config HealthCheckerConfig, logger *zap.Logger) *HealthChecker {
	return &HealthChecker{
		endpoint:         config.Endpoint,
		interval:         config.Interval,
		timeout:          config.Timeout,
		failureThreshold: config.FailureThreshold,
		logger:          logger,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Check performs a single health check
func (h *HealthChecker) Check(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.endpoint, nil)
	if err != nil {
		return fmt.Errorf("creating health check request: %w", err)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}

// Monitor continuously monitors collector health
func (h *HealthChecker) Monitor(ctx context.Context) <-chan error {
	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)

		ticker := time.NewTicker(h.interval)
		defer ticker.Stop()

		consecutiveFailures := 0

		// Initial delay to allow collector to start
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				err := h.Check(ctx)
				if err != nil {
					consecutiveFailures++
					h.logger.Warn("Health check failed",
						zap.Error(err),
						zap.Int("consecutive_failures", consecutiveFailures),
					)

					if consecutiveFailures >= h.failureThreshold {
						h.logger.Error("Health check failure threshold exceeded",
							zap.Int("threshold", h.failureThreshold),
						)
						select {
						case errCh <- fmt.Errorf("health check failed %d consecutive times", consecutiveFailures):
						case <-ctx.Done():
							return
						}
						consecutiveFailures = 0 // Reset after reporting
					}
				} else {
					if consecutiveFailures > 0 {
						h.logger.Info("Health check recovered",
							zap.Int("previous_failures", consecutiveFailures),
						)
					}
					consecutiveFailures = 0
				}
			}
		}
	}()

	return errCh
}

// WaitForHealthy waits for the collector to become healthy
func (h *HealthChecker) WaitForHealthy(ctx context.Context, maxWait time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, maxWait)
	defer cancel()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for collector to become healthy")
		case <-ticker.C:
			if err := h.Check(ctx); err == nil {
				h.logger.Info("Collector is healthy")
				return nil
			}
		}
	}
}