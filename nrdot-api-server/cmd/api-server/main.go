package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	apiserver "github.com/newrelic/nrdot-host/nrdot-api-server"
	"github.com/newrelic/nrdot-host/nrdot-api-server/pkg/handlers"
	"github.com/newrelic/nrdot-host/nrdot-api-server/pkg/models"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const version = "v1.0.0"

func main() {
	var (
		host       = flag.String("host", "127.0.0.1", "Host to bind to (localhost only)")
		port       = flag.Int("port", 8089, "Port to listen on")
		readOnly   = flag.Bool("read-only", false, "Enable read-only mode")
		enableCORS = flag.Bool("cors", true, "Enable CORS for localhost origins")
		debug      = flag.Bool("debug", false, "Enable debug logging")
		showVersion = flag.Bool("version", false, "Show version")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("nrdot-api-server %s\n", version)
		os.Exit(0)
	}

	// Initialize logger
	logger := initLogger(*debug)
	defer logger.Sync()

	// Create server config
	config := apiserver.Config{
		Host:        *host,
		Port:        *port,
		ReadOnly:    *readOnly,
		Version:     version,
		EnableCORS:  *enableCORS,
		EnableDebug: *debug,
	}

	// Create server
	server := apiserver.NewServer(config, logger)

	// Create mock providers for demonstration
	// In a real implementation, these would connect to actual systems
	statusProvider := &mockStatusProvider{}
	healthProvider := &mockHealthProvider{}
	configProvider := &mockConfigProvider{logger: logger}
	metricsProvider := &mockMetricsProvider{}

	// Set providers
	server.SetProviders(statusProvider, healthProvider, configProvider, metricsProvider)

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logger.Info("Received signal", zap.String("signal", sig.String()))
		cancel()
	}()

	// Start server
	if err := server.Start(ctx); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}

	// Wait for shutdown
	<-ctx.Done()

	// Shutdown server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Failed to shutdown server", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("Server stopped")
}

// initLogger initializes the logger
func initLogger(debug bool) *zap.Logger {
	config := zap.NewProductionConfig()
	
	if debug {
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		config.Development = true
	}

	// Use human-readable timestamps
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := config.Build()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}

	return logger
}

// Mock implementations for demonstration

type mockStatusProvider struct{}

func (m *mockStatusProvider) GetCollectorStatus() []models.CollectorStatus {
	return []models.CollectorStatus{
		{
			Name:   "hostmetrics",
			Type:   "receiver",
			Status: models.StatusRunning,
		},
		{
			Name:   "otlp",
			Type:   "exporter",
			Status: models.StatusRunning,
		},
	}
}

func (m *mockStatusProvider) GetConfigHash() string {
	return "abc123def456"
}

func (m *mockStatusProvider) GetLastReload() *time.Time {
	t := time.Now().Add(-1 * time.Hour)
	return &t
}

func (m *mockStatusProvider) GetErrors() []models.ErrorInfo {
	return nil
}

type mockHealthProvider struct{}

func (m *mockHealthProvider) GetComponentHealth() map[string]models.Health {
	return map[string]models.Health{
		"collector": {
			Status:  models.StatusHealthy,
			Message: "All pipelines running",
		},
		"api": {
			Status:  models.StatusHealthy,
			Message: "API server responding",
		},
	}
}

type mockConfigProvider struct {
	logger *zap.Logger
	config interface{}
	loadedAt time.Time
}

func (m *mockConfigProvider) GetCurrentConfig() (interface{}, string, time.Time) {
	if m.config == nil {
		m.config = map[string]interface{}{
			"service": map[string]interface{}{
				"name": "nrdot-host",
				"environment": "production",
			},
			"metrics": map[string]interface{}{
				"enabled": true,
				"interval": "60s",
			},
		}
		m.loadedAt = time.Now()
	}
	return m.config, "default", m.loadedAt
}

func (m *mockConfigProvider) ValidateConfig(config interface{}) *models.ValidationResult {
	// Simple validation
	return &models.ValidationResult{
		Valid:    true,
		Warnings: []string{"This is a mock validation"},
	}
}

func (m *mockConfigProvider) UpdateConfig(config interface{}, dryRun bool) error {
	if !dryRun {
		m.config = config
		m.loadedAt = time.Now()
		m.logger.Info("Mock config updated")
	}
	return nil
}

func (m *mockConfigProvider) ReloadConfig(force bool) error {
	m.loadedAt = time.Now()
	m.logger.Info("Mock config reloaded", zap.Bool("force", force))
	return nil
}

type mockMetricsProvider struct{}

func (m *mockMetricsProvider) GetCustomMetrics() []handlers.Metric {
	return []handlers.Metric{
		{
			Name:  "nrdot_collectors_running",
			Help:  "Number of running collectors",
			Type:  "gauge",
			Value: 2,
		},
		{
			Name:  "nrdot_config_reloads_total",
			Help:  "Total number of configuration reloads",
			Type:  "counter",
			Value: 5,
		},
	}
}