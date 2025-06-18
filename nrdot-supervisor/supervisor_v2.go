// Package supervisor provides the unified supervisor with embedded API server
package supervisor

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/newrelic/nrdot-host/nrdot-api-server/pkg/handlers"
	"github.com/newrelic/nrdot-host/nrdot-common/pkg/interfaces"
	"github.com/newrelic/nrdot-host/nrdot-common/pkg/models"
	configengine "github.com/newrelic/nrdot-host/nrdot-config-engine"
	"github.com/newrelic/nrdot-host/nrdot-telemetry-client"
	"go.uber.org/zap"
)

// UnifiedSupervisor combines supervisor, API server, and config engine
type UnifiedSupervisor struct {
	logger        *zap.Logger
	configEngine  *configengine.EngineV2
	telemetry     *telemetryclient.Client
	
	// Collector management
	collector     *CollectorProcess
	reloadStrategy interfaces.SupervisorCommander
	
	// API Server
	apiServer     *http.Server
	apiHandlers   *handlers.Handlers
	
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
	var telemetry *telemetryclient.Client
	if config.EnableTelemetry {
		telemetry = telemetryclient.New(telemetryclient.Config{
			ServiceName: "nrdot-supervisor",
			Logger:      config.Logger.Named("telemetry"),
		})
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
	
	// Start telemetry if enabled
	if s.telemetry != nil {
		if err := s.telemetry.Start(ctx); err != nil {
			s.logger.Warn("Failed to start telemetry", zap.Error(err))
		}
	}
	
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
	
	// Stop telemetry
	if s.telemetry != nil {
		s.telemetry.Stop()
	}
	
	return nil
}

// setupAPIServer configures the embedded API server
func (s *UnifiedSupervisor) setupAPIServer() {
	// Create handlers with direct access to supervisor
	s.apiHandlers = &handlers.Handlers{
		StatusProvider:  s,
		ConfigProvider:  s.configEngine,
		HealthProvider:  s,
		MetricsProvider: s.metrics,
		Logger:          s.logger.Named("api"),
	}
	
	// Set up routes
	router := mux.NewRouter()
	
	// Health endpoints
	router.HandleFunc("/health", s.apiHandlers.Health).Methods("GET")
	router.HandleFunc("/ready", s.apiHandlers.Ready).Methods("GET")
	
	// Prometheus metrics endpoint at root level
	router.HandleFunc("/metrics", s.apiHandlers.Metrics).Methods("GET")
	
	// API v1 routes
	v1 := router.PathPrefix("/v1").Subrouter()
	v1.HandleFunc("/status", s.apiHandlers.Status).Methods("GET")
	v1.HandleFunc("/config", s.apiHandlers.GetConfig).Methods("GET")
	v1.HandleFunc("/config", s.apiHandlers.UpdateConfig).Methods("POST", "PUT")
	v1.HandleFunc("/config/validate", s.apiHandlers.ValidateConfig).Methods("POST")
	v1.HandleFunc("/metrics", s.apiHandlers.Metrics).Methods("GET")
	
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
		metrics, err := s.collector.GetMetrics()
		if err == nil {
			status.ResourceMetrics = metrics
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
	
	startTime := time.Now()
	oldVersion := s.status.ConfigVersion
	
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
		fmt.Sprintf("Version %d -> %d", oldVersion, result.NewVersion))
	
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
	
	// Get current OTel config from engine
	config, err := s.configEngine.GetCurrentConfig(ctx)
	if err != nil {
		return fmt.Errorf("no configuration loaded")
	}
	
	// Generate OTel config
	generated, err := s.configEngine.ProcessUserConfig(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to generate config: %w", err)
	}
	
	// Create collector process
	s.collector = &CollectorProcess{
		Path:       s.config.CollectorPath,
		ConfigYAML: generated.OTelConfig,
		WorkDir:    s.config.WorkDir,
		Logger:     s.logger.Named("collector"),
	}
	
	// Start the collector
	if err := s.collector.Start(ctx); err != nil {
		return fmt.Errorf("failed to start collector: %w", err)
	}
	
	// Update status
	s.status.State = models.CollectorStateRunning
	s.status.StartTime = time.Now()
	
	s.recordEvent(models.EventTypeStarted, models.EventSeverityInfo,
		"Collector started", fmt.Sprintf("PID: %d", s.collector.cmd.Process.Pid))
	
	return nil
}

// API control endpoints
func (s *UnifiedSupervisor) handleReload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Default to blue-green strategy
	strategy := models.ReloadStrategyBlueGreen
	
	result, err := s.ReloadCollector(ctx, strategy)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// Return result as JSON
}

func (s *UnifiedSupervisor) handleRestart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	if err := s.RestartCollector(ctx, "API request"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
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
	event := models.Event{
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
	if s.telemetry != nil {
		s.telemetry.RecordEvent(event)
	}
}

func (s *UnifiedSupervisor) getHostname() string {
	hostname, _ := os.Hostname()
	return hostname
}

// Ensure UnifiedSupervisor implements required interfaces
var (
	_ interfaces.StatusProvider      = (*UnifiedSupervisor)(nil)
	_ interfaces.HealthProvider      = (*UnifiedSupervisor)(nil)
	_ interfaces.ConfigProvider      = (*UnifiedSupervisor)(nil)
	_ interfaces.SupervisorCommander = (*UnifiedSupervisor)(nil)
)