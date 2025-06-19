// Package supervisor provides the unified supervisor with embedded API server
package supervisor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/newrelic/nrdot-host/nrdot-api-server/pkg/middleware"
	"github.com/newrelic/nrdot-host/nrdot-common/pkg/interfaces"
	"github.com/newrelic/nrdot-host/nrdot-common/pkg/models"
	configengine "github.com/newrelic/nrdot-host/nrdot-config-engine"
	telemetryclient "github.com/newrelic/nrdot-host/nrdot-telemetry-client"
	"go.uber.org/zap"
)

// UnifiedSupervisor combines supervisor, API server, and config engine
type UnifiedSupervisor struct {
	logger        *zap.Logger
	configEngine  *configengine.EngineV2
	telemetry     telemetryclient.TelemetryClient
	
	// Collector management
	collector     *CollectorProcess
	reloadStrategy interfaces.SupervisorCommander
	
	// API Server
	apiServer     *http.Server
	apiHandlers   *Handlers
	
	// State
	mu            sync.RWMutex
	status        models.CollectorStatus
	health        models.HealthStatus
	startTime     time.Time
	
	// Metrics collection
	metrics       *MetricsCollector
	
	// Options
	config        SupervisorConfig
}

// SupervisorConfig holds configuration for the unified supervisor
type SupervisorConfig struct {
	// Core settings
	CollectorPath   string
	ConfigPath      string
	WorkDir         string
	
	// API settings
	APIEnabled      bool
	APIListenAddr   string
	
	// Behavior settings
	RestartDelay    time.Duration
	MaxRestarts     int
	HealthCheckInterval time.Duration
	
	// Features
	EnableTelemetry bool
	EnableDebug     bool
	
	// Rate limiting
	RateLimitEnabled   bool
	RateLimitRate      int           // requests per interval
	RateLimitInterval  time.Duration // interval duration
	RateLimitBurst     int           // burst size
	
	Logger          *zap.Logger
}

// NewUnifiedSupervisor creates a new supervisor with all components embedded
func NewUnifiedSupervisor(config SupervisorConfig) (*UnifiedSupervisor, error) {
	if config.Logger == nil {
		config.Logger = zap.NewNop()
	}
	
	// Create config engine
	engineConfig := configengine.ConfigV2{
		Logger:      config.Logger.Named("config-engine"),
		MaxVersions: 20,
		EnableBackup: true,
	}
	
	engine, err := configengine.NewEngineV2(engineConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create config engine: %w", err)
	}
	
	// Create telemetry client if enabled
	var telemetry telemetryclient.TelemetryClient
	if config.EnableTelemetry {
		// Use a no-op client for now
		telemetry = telemetryclient.NewNoOpClient()
	}
	
	s := &UnifiedSupervisor{
		logger:       config.Logger,
		configEngine: engine,
		telemetry:    telemetry,
		metrics:      NewMetricsCollector(),
		config:       config,
		startTime:    time.Now(),
		status: models.CollectorStatus{
			State:         models.CollectorStateStopped,
			Version:       "unknown",
			ConfigVersion: 0,
			StartTime:     time.Now(),
		},
		health: models.HealthStatus{
			State:     models.HealthStateUnknown,
			Timestamp: time.Now(),
		},
	}
	
	// Set initial metrics state
	s.metrics.SetAPIEnabled(config.APIEnabled)
	
	// Set up reload strategy
	s.reloadStrategy = &BlueGreenReloadStrategy{supervisor: s}
	
	// Set up API server if enabled
	if config.APIEnabled {
		s.setupAPIServer()
	}
	
	return s, nil
}

// Start starts the supervisor and all embedded components
func (s *UnifiedSupervisor) Start(ctx context.Context) error {
	s.logger.Info("Starting unified supervisor")
	
	// Telemetry client is ready (no-op client doesn't need starting)
	
	// Start API server if enabled
	if s.config.APIEnabled {
		go s.startAPIServer()
	}
	
	// Load initial configuration
	if s.config.ConfigPath != "" {
		if err := s.loadConfiguration(ctx); err != nil {
			return fmt.Errorf("failed to load initial configuration: %w", err)
		}
	}
	
	// Start the collector
	if err := s.startCollector(ctx); err != nil {
		return fmt.Errorf("failed to start collector: %w", err)
	}
	
	// Start health monitoring
	go s.healthMonitorLoop(ctx)
	
	// Start restart monitor
	go s.restartMonitorLoop(ctx)
	
	s.logger.Info("Unified supervisor started successfully")
	return nil
}

// Stop gracefully stops all components
func (s *UnifiedSupervisor) Stop(ctx context.Context) error {
	s.logger.Info("Stopping unified supervisor")
	
	// Stop collector
	if err := s.StopCollector(ctx, 30*time.Second); err != nil {
		s.logger.Error("Failed to stop collector", zap.Error(err))
	}
	
	// Stop API server
	if s.apiServer != nil {
		if err := s.apiServer.Shutdown(ctx); err != nil {
			s.logger.Error("Failed to stop API server", zap.Error(err))
		}
	}
	
	// Telemetry client cleanup (no-op client doesn't need stopping)
	
	return nil
}

// setupAPIServer configures the embedded API server
func (s *UnifiedSupervisor) setupAPIServer() {
	// Create handlers
	s.apiHandlers = &Handlers{
		Supervisor: s,
		Logger:     s.logger.Named("api"),
	}
	
	// Set up routes
	router := mux.NewRouter()
	
	// Apply rate limiting if configured
	if s.config.RateLimitEnabled {
		rateLimiter := middleware.NewRateLimiter(
			s.config.RateLimitRate,
			s.config.RateLimitInterval,
			s.config.RateLimitBurst,
			s.logger.Named("ratelimit"),
		)
		
		// Rate limit by IP address for supervisor API
		router.Use(rateLimiter.RateLimitMiddleware(middleware.IPKeyFunc))
	}
	
	// Health endpoints
	router.HandleFunc("/health", s.apiHandlers.Health).Methods("GET")
	router.HandleFunc("/ready", s.apiHandlers.Ready).Methods("GET")
	
	// Prometheus metrics endpoint at root level
	router.HandleFunc("/metrics", s.apiHandlers.GetMetrics).Methods("GET")
	
	// API v1 routes
	v1 := router.PathPrefix("/v1").Subrouter()
	v1.HandleFunc("/status", s.apiHandlers.Status).Methods("GET")
	v1.HandleFunc("/config", s.apiHandlers.GetConfig).Methods("GET")
	v1.HandleFunc("/config", s.apiHandlers.UpdateConfig).Methods("POST", "PUT")
	v1.HandleFunc("/config/validate", s.apiHandlers.ValidateConfig).Methods("POST")
	v1.HandleFunc("/metrics", s.apiHandlers.GetMetrics).Methods("GET")
	
	// Control endpoints (new)
	v1.HandleFunc("/control/reload", s.handleReload).Methods("POST")
	v1.HandleFunc("/control/restart", s.handleRestart).Methods("POST")
	
	s.apiServer = &http.Server{
		Addr:         s.config.APIListenAddr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}
}

// startAPIServer starts the embedded API server
func (s *UnifiedSupervisor) startAPIServer() {
	s.logger.Info("Starting API server", zap.String("addr", s.config.APIListenAddr))
	if err := s.apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.logger.Error("API server error", zap.Error(err))
	}
}

// Implement StatusProvider interface
func (s *UnifiedSupervisor) GetStatus(ctx context.Context) (*models.CollectorStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	status := s.status
	status.Uptime = time.Since(s.startTime)
	
	// Get real-time metrics if collector is running
	if s.collector != nil && s.collector.IsRunning() {
		// Placeholder for real metrics
		status.ResourceMetrics = models.ResourceMetrics{
			CPUPercent:     10.5,
			MemoryBytes:    256 * 1024 * 1024,
			GoroutineCount: 42,
		}
	}
	
	return &status, nil
}

// Implement HealthProvider interface
func (s *UnifiedSupervisor) GetHealth(ctx context.Context) (*models.HealthStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	health := s.health
	health.Timestamp = time.Now()
	
	// Add component health
	health.Components = []models.ComponentHealth{
		{
			Name:      "supervisor",
			Type:      "core",
			State:     models.HealthStateHealthy,
			LastCheck: time.Now(),
		},
		{
			Name:      "config-engine",
			Type:      "core", 
			State:     models.HealthStateHealthy,
			LastCheck: time.Now(),
		},
		{
			Name:      "collector",
			Type:      "core",
			State:     s.getCollectorHealthState(),
			LastCheck: time.Now(),
		},
	}
	
	// Overall health based on components
	if s.getCollectorHealthState() == models.HealthStateHealthy {
		health.State = models.HealthStateHealthy
		health.ReadinessProbe = true
		health.LivenessProbe = true
	} else {
		health.State = models.HealthStateDegraded
		health.ReadinessProbe = false
		health.LivenessProbe = true
	}
	
	return &health, nil
}

// Implement SupervisorCommander interface
func (s *UnifiedSupervisor) ReloadCollector(ctx context.Context, strategy models.ReloadStrategy) (*models.ReloadResult, error) {
	s.logger.Info("Reloading collector", zap.String("strategy", string(strategy)))
	
	// Track reload timing
	_ = time.Now()
	_ = s.status.ConfigVersion
	
	// Use the configured reload strategy
	result, err := s.reloadStrategy.ReloadCollector(ctx, strategy)
	if err != nil {
		s.recordEvent(models.EventTypeConfigRejected, models.EventSeverityError, 
			"Configuration reload failed", err.Error())
		return result, err
	}
	
	// Update status
	s.mu.Lock()
	s.status.ConfigVersion = result.NewVersion
	s.status.LastConfigLoad = time.Now()
	s.mu.Unlock()
	
	s.recordEvent(models.EventTypeReloaded, models.EventSeverityInfo,
		"Configuration reloaded successfully", 
		fmt.Sprintf("Version %d -> %d", result.OldVersion, result.NewVersion))
	
	return result, nil
}

// loadConfiguration loads configuration from file
func (s *UnifiedSupervisor) loadConfiguration(ctx context.Context) error {
	data, err := os.ReadFile(s.config.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	
	update := &models.ConfigUpdate{
		Config: data,
		Format: "yaml",
		Source: "file",
		Author: "supervisor",
	}
	
	result, err := s.configEngine.ApplyConfig(ctx, update)
	if err != nil {
		return err
	}
	
	if !result.Success {
		return fmt.Errorf("configuration is invalid")
	}
	
	s.mu.Lock()
	s.status.ConfigVersion = result.Version
	s.mu.Unlock()
	
	return nil
}

// startCollector starts the collector process
func (s *UnifiedSupervisor) startCollector(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.collector != nil && s.collector.IsRunning() {
		return fmt.Errorf("collector already running")
	}
	
	// Generate OTel config
	generated, err := s.configEngine.ProcessUserConfig(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to generate config: %w", err)
	}
	
	// Write config to file
	configPath := fmt.Sprintf("%s/config.yaml", s.config.WorkDir)
	if err := os.WriteFile(configPath, []byte(generated.OTelConfig), 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	
	// Create collector process
	s.collector = &CollectorProcess{
		binaryPath: s.config.CollectorPath,
		configPath: configPath,
		workDir:    s.config.WorkDir,
		logger:     s.logger.Named("collector"),
	}
	
	// Start the collector
	if err := s.collector.Start(ctx); err != nil {
		return fmt.Errorf("failed to start collector: %w", err)
	}
	
	// Update status
	s.status.State = models.CollectorStateRunning
	s.status.StartTime = time.Now()
	
	// Update metrics
	s.metrics.SetCollectorRunning(true)
	
	s.recordEvent(models.EventTypeStarted, models.EventSeverityInfo,
		"Collector started", fmt.Sprintf("PID: %d", s.collector.cmd.Process.Pid))
	
	return nil
}

// API control endpoints
func (s *UnifiedSupervisor) handleReload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Track API request
	s.metrics.IncrementRequests()
	
	// Default to blue-green strategy
	strategy := models.ReloadStrategyBlueGreen
	
	startTime := time.Now()
	result, err := s.ReloadCollector(ctx, strategy)
	duration := time.Since(startTime)
	
	// Update metrics
	s.metrics.SetReloadDuration(duration)
	if err != nil {
		s.metrics.IncrementFailedReloads()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	s.metrics.IncrementConfigReloads()
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

func (s *UnifiedSupervisor) handleRestart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Track API request
	s.metrics.IncrementRequests()
	
	if err := s.RestartCollector(ctx, "API request"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	s.metrics.IncrementCollectorRestarts()
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// Helper methods
func (s *UnifiedSupervisor) getCollectorHealthState() models.HealthState {
	if s.collector == nil || !s.collector.IsRunning() {
		return models.HealthStateUnhealthy
	}
	
	// Check collector health endpoint or metrics
	// For now, just check if process is running
	return models.HealthStateHealthy
}

func (s *UnifiedSupervisor) recordEvent(eventType models.EventType, severity models.EventSeverity, summary, details string) {
	_ = models.Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Component: "supervisor",
		Severity:  severity,
		Summary:   summary,
		Details:   details,
		Source: models.EventSource{
			Component: "nrdot-supervisor",
			Host:      s.getHostname(),
			Version:   "2.0",
		},
	}
	
	// Log the event
	switch severity {
	case models.EventSeverityError, models.EventSeverityCritical:
		s.logger.Error(summary, zap.String("details", details))
	case models.EventSeverityWarning:
		s.logger.Warn(summary, zap.String("details", details))
	default:
		s.logger.Info(summary, zap.String("details", details))
	}
	
	// Send to telemetry if enabled
	// TODO: Add telemetry event recording when method is available
}

func (s *UnifiedSupervisor) getHostname() string {
	hostname, _ := os.Hostname()
	return hostname
}

// RestartCollector implements SupervisorCommander interface
func (s *UnifiedSupervisor) RestartCollector(ctx context.Context, reason string) error {
	s.logger.Info("Restarting collector", zap.String("reason", reason))
	
	// Stop existing collector
	if s.collector != nil && s.collector.IsRunning() {
		if err := s.collector.Stop(ctx); err != nil {
			s.logger.Warn("Failed to stop collector cleanly", zap.Error(err))
		}
	}
	
	// Small delay before restart
	time.Sleep(2 * time.Second)
	
	// Start new collector
	return s.startCollector(ctx)
}

// StopCollector implements SupervisorCommander interface
func (s *UnifiedSupervisor) StopCollector(ctx context.Context, timeout time.Duration) error {
	s.logger.Info("Stopping collector", zap.Duration("timeout", timeout))
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.collector == nil || !s.collector.IsRunning() {
		return nil
	}
	
	// Create timeout context
	stopCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	// Stop the collector
	if err := s.collector.Stop(stopCtx); err != nil {
		return fmt.Errorf("failed to stop collector: %w", err)
	}
	
	// Update state
	s.status.State = models.CollectorStateStopped
	s.metrics.SetCollectorRunning(false)
	
	s.recordEvent(models.EventTypeStopped, models.EventSeverityInfo,
		"Collector stopped", "Graceful shutdown completed")
	
	return nil
}

// StartCollector implements SupervisorCommander interface
func (s *UnifiedSupervisor) StartCollector(ctx context.Context, configPath string) error {
	s.logger.Info("Starting collector", zap.String("config", configPath))
	
	// Load configuration if path provided
	if configPath != "" {
		if _, err := s.loadInitialConfig(ctx); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}
	
	return s.startCollector(ctx)
}

// UpdateCollector implements SupervisorCommander interface
func (s *UnifiedSupervisor) UpdateCollector(ctx context.Context, update *models.CollectorUpdate) (*models.UpdateResult, error) {
	// Not implemented yet - would handle binary updates
	return nil, fmt.Errorf("collector updates not implemented")
}

// GetPipelineStatus returns status for a specific pipeline
func (s *UnifiedSupervisor) GetPipelineStatus(ctx context.Context, pipelineName string) (*models.PipelineStatus, error) {
	// For now, return a simple status
	return &models.PipelineStatus{
		Name:  pipelineName,
		Type:  "metrics",
		State: "running",
	}, nil
}

// GetComponentHealth returns health status for a specific component
func (s *UnifiedSupervisor) GetComponentHealth(ctx context.Context, componentName string) (*models.ComponentHealth, error) {
	return &models.ComponentHealth{
		Name:      componentName,
		Type:      "service",
		State:     models.HealthStateHealthy,
		LastCheck: time.Now(),
	}, nil
}

// ApplyConfig applies a new configuration
func (s *UnifiedSupervisor) ApplyConfig(ctx context.Context, update *models.ConfigUpdate) (*models.ConfigResult, error) {
	// Delegate to config engine
	return s.configEngine.ApplyConfig(ctx, update)
}

// Subscribe allows components to receive status updates
func (s *UnifiedSupervisor) Subscribe(ctx context.Context, subscriber interfaces.StatusSubscriber) error {
	// Not implemented yet
	return nil
}

// RegisterHealthCheck registers a health check function
func (s *UnifiedSupervisor) RegisterHealthCheck(name string, check interfaces.HealthCheck) {
	// Not implemented yet
}

// GetConfigHistory returns the configuration history
func (s *UnifiedSupervisor) GetConfigHistory(ctx context.Context, limit int) ([]*models.ConfigVersion, error) {
	// Delegate to config engine
	return s.configEngine.GetVersionHistory(ctx, limit)
}

// GetCurrentConfig returns the current configuration
func (s *UnifiedSupervisor) GetCurrentConfig(ctx context.Context) (*models.Config, error) {
	// Return a simple config for now
	return &models.Config{
		Version: 1,
	}, nil
}

// RollbackConfig rolls back to a previous configuration version
func (s *UnifiedSupervisor) RollbackConfig(ctx context.Context, version int) error {
	// Not implemented yet
	return fmt.Errorf("rollback not implemented")
}

// ValidateConfig validates a configuration without applying it
func (s *UnifiedSupervisor) ValidateConfig(ctx context.Context, config []byte) (*models.ValidationResult, error) {
	// Delegate to config engine
	update := &models.ConfigUpdate{
		Config: config,
		Format: "yaml",
		DryRun: true,
	}
	
	result, err := s.configEngine.ApplyConfig(ctx, update)
	if err != nil {
		return nil, err
	}
	
	return result.ValidationResult, nil
}

// healthMonitorLoop monitors collector health
func (s *UnifiedSupervisor) healthMonitorLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkHealth(ctx)
		}
	}
}

// restartMonitorLoop monitors for restart conditions
func (s *UnifiedSupervisor) restartMonitorLoop(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkRestartConditions(ctx)
		}
	}
}

// checkHealth performs health checks
func (s *UnifiedSupervisor) checkHealth(ctx context.Context) {
	// Simple health check - is collector running?
	if s.collector != nil && !s.collector.IsRunning() {
		s.logger.Warn("Collector is not running, attempting restart")
		if err := s.startCollector(ctx); err != nil {
			s.logger.Error("Failed to restart collector", zap.Error(err))
		}
	}
}

// checkRestartConditions checks if restart is needed
func (s *UnifiedSupervisor) checkRestartConditions(ctx context.Context) {
	// Placeholder for restart logic
	// Could check memory usage, error rates, etc.
}

// loadInitialConfig loads the initial configuration
func (s *UnifiedSupervisor) loadInitialConfig(ctx context.Context) (*models.Config, error) {
	// For now, return a default config
	return &models.Config{
		Version: 1,
	}, nil
}

// Ensure UnifiedSupervisor implements required interfaces
var (
	_ interfaces.StatusProvider      = (*UnifiedSupervisor)(nil)
	_ interfaces.HealthProvider      = (*UnifiedSupervisor)(nil)
	_ interfaces.ConfigProvider      = (*UnifiedSupervisor)(nil)
	_ interfaces.SupervisorCommander = (*UnifiedSupervisor)(nil)
)