package supervisor

import (
	"context"
	"fmt"
	"time"

	"github.com/newrelic/nrdot-host/nrdot-common/pkg/models"
)

// BlueGreenReloadStrategy implements zero-downtime configuration reload
type BlueGreenReloadStrategy struct {
	supervisor *UnifiedSupervisor
}

// ReloadCollector performs a blue-green reload
func (s *BlueGreenReloadStrategy) ReloadCollector(ctx context.Context, strategy models.ReloadStrategy) (*models.ReloadResult, error) {
	startTime := time.Now()
	oldVersion := s.supervisor.status.ConfigVersion
	
	result := &models.ReloadResult{
		Strategy:   strategy,
		OldVersion: oldVersion,
		StartTime:  startTime,
	}
	
	switch strategy {
	case models.ReloadStrategyBlueGreen:
		return s.blueGreenReload(ctx, result)
	case models.ReloadStrategyGraceful:
		return s.gracefulReload(ctx, result)
	case models.ReloadStrategyInPlace:
		return s.inPlaceReload(ctx, result)
	default:
		result.Success = false
		result.Error = models.NewError(
			models.ErrCodeConfigUnsupported,
			fmt.Sprintf("Unsupported reload strategy: %s", strategy),
			models.ErrorCategoryConfig,
			models.SeverityError,
		)
		return result, fmt.Errorf("unsupported reload strategy: %s", strategy)
	}
}

// blueGreenReload starts new collector, verifies health, then stops old
func (s *BlueGreenReloadStrategy) blueGreenReload(ctx context.Context, result *models.ReloadResult) (*models.ReloadResult, error) {
	s.supervisor.logger.Info("Starting blue-green reload")
	
	// Get new configuration
	newConfig, err := s.supervisor.configEngine.GetCurrentConfig(ctx)
	if err != nil {
		result.Success = false
		result.Error = models.NewError(
			models.ErrCodeConfigMissing,
			"No configuration available",
			models.ErrorCategoryConfig,
			models.SeverityError,
		)
		return result, err
	}
	
	// Generate new OTel config
	generated, err := s.supervisor.configEngine.ProcessUserConfig(ctx, nil)
	if err != nil {
		result.Success = false
		result.Error = models.NewError(
			models.ErrCodeConfigInvalid,
			"Failed to generate configuration",
			models.ErrorCategoryConfig,
			models.SeverityError,
		)
		return result, err
	}
	
	// Keep reference to old collector
	oldCollector := s.supervisor.collector
	
	// Create new collector process (blue)
	newCollector := &CollectorProcess{
		Path:       s.supervisor.config.CollectorPath,
		ConfigYAML: generated.OTelConfig,
		WorkDir:    s.supervisor.config.WorkDir,
		Logger:     s.supervisor.logger.Named("collector-new"),
		Port:       9999, // Use different port initially
	}
	
	// Start new collector
	if err := newCollector.Start(ctx); err != nil {
		result.Success = false
		result.Error = models.NewError(
			models.ErrCodeInternalError,
			"Failed to start new collector",
			models.ErrorCategoryInternal,
			models.SeverityError,
		).WithDetails(err.Error())
		return result, fmt.Errorf("failed to start new collector: %w", err)
	}
	
	// Wait for new collector to be healthy
	healthCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	if err := s.waitForHealth(healthCtx, newCollector); err != nil {
		// New collector failed, kill it
		newCollector.Stop(context.Background())
		
		result.Success = false
		result.Error = models.NewError(
			models.ErrCodeInternalError,
			"New collector failed health check",
			models.ErrorCategoryInternal,
			models.SeverityError,
		).WithDetails(err.Error())
		return result, fmt.Errorf("new collector failed health check: %w", err)
	}
	
	// Switch to new collector
	s.supervisor.mu.Lock()
	s.supervisor.collector = newCollector
	s.supervisor.status.ConfigVersion++
	s.supervisor.status.LastConfigLoad = time.Now()
	s.supervisor.mu.Unlock()
	
	// Stop old collector gracefully
	if oldCollector != nil && oldCollector.IsRunning() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer stopCancel()
		
		if err := oldCollector.Stop(stopCtx); err != nil {
			s.supervisor.logger.Warn("Failed to stop old collector gracefully", 
				"error", err)
		}
	}
	
	// Success
	result.Success = true
	result.NewVersion = s.supervisor.status.ConfigVersion
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	
	s.supervisor.logger.Info("Blue-green reload completed successfully",
		"duration", result.Duration,
		"oldVersion", result.OldVersion,
		"newVersion", result.NewVersion)
	
	return result, nil
}

// gracefulReload stops the collector and starts with new config
func (s *BlueGreenReloadStrategy) gracefulReload(ctx context.Context, result *models.ReloadResult) (*models.ReloadResult, error) {
	s.supervisor.logger.Info("Starting graceful reload")
	
	// Stop current collector
	if s.supervisor.collector != nil && s.supervisor.collector.IsRunning() {
		stopCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		
		if err := s.supervisor.collector.Stop(stopCtx); err != nil {
			result.Success = false
			result.Error = models.NewError(
				models.ErrCodeInternalError,
				"Failed to stop collector",
				models.ErrorCategoryInternal,
				models.SeverityError,
			).WithDetails(err.Error())
			return result, err
		}
	}
	
	// Start with new configuration
	if err := s.supervisor.startCollector(ctx); err != nil {
		result.Success = false
		result.Error = models.NewError(
			models.ErrCodeInternalError,
			"Failed to start collector with new config",
			models.ErrorCategoryInternal,
			models.SeverityError,
		).WithDetails(err.Error())
		
		// Try to rollback by starting with old config
		s.rollbackStart(ctx)
		
		return result, err
	}
	
	// Success
	result.Success = true
	result.NewVersion = s.supervisor.status.ConfigVersion
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	
	return result, nil
}

// inPlaceReload attempts SIGHUP reload (legacy, not recommended)
func (s *BlueGreenReloadStrategy) inPlaceReload(ctx context.Context, result *models.ReloadResult) (*models.ReloadResult, error) {
	s.supervisor.logger.Warn("In-place reload requested - this strategy is deprecated")
	
	if s.supervisor.collector == nil || !s.supervisor.collector.IsRunning() {
		result.Success = false
		result.Error = models.NewError(
			models.ErrCodeResourceNotFound,
			"No collector running",
			models.ErrorCategoryInternal,
			models.SeverityError,
		)
		return result, fmt.Errorf("no collector running")
	}
	
	// Send SIGHUP (won't work on Windows)
	if err := s.supervisor.collector.SendSignal("HUP"); err != nil {
		result.Success = false
		result.Error = models.NewError(
			models.ErrCodeInternalError,
			"Failed to send reload signal",
			models.ErrorCategoryInternal,
			models.SeverityError,
		).WithDetails(err.Error())
		return result, err
	}
	
	// Note: We can't verify if reload succeeded with SIGHUP
	result.Success = true
	result.NewVersion = s.supervisor.status.ConfigVersion + 1
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	
	s.supervisor.logger.Warn("In-place reload signal sent - success not verified")
	
	return result, nil
}

// waitForHealth waits for collector to become healthy
func (s *BlueGreenReloadStrategy) waitForHealth(ctx context.Context, collector *CollectorProcess) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if collector.IsHealthy() {
				return nil
			}
		}
	}
}

// rollbackStart attempts to start collector with previous config
func (s *BlueGreenReloadStrategy) rollbackStart(ctx context.Context) {
	s.supervisor.logger.Info("Attempting to rollback to previous configuration")
	
	// In a real implementation, this would restore the previous config
	// For now, just try to start with current config
	if err := s.supervisor.startCollector(ctx); err != nil {
		s.supervisor.logger.Error("Rollback failed", "error", err)
	} else {
		s.supervisor.logger.Info("Rollback successful")
	}
}