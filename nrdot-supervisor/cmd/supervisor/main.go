package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/newrelic/nrdot-host/nrdot-supervisor"
	"github.com/newrelic/nrdot-host/nrdot-supervisor/pkg/restart"
	telemetryclient "github.com/newrelic/nrdot-host/nrdot-telemetry-client"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Collector flags
	collectorBinary  = pflag.String("collector-binary", "otelcol", "Path to OTel collector binary")
	collectorConfig  = pflag.String("collector-config", "/etc/otel/config.yaml", "Path to collector config file")
	collectorArgs    = pflag.StringSlice("collector-args", []string{}, "Additional collector arguments")
	collectorWorkDir = pflag.String("collector-workdir", "", "Collector working directory")
	memoryLimit      = pflag.Uint64("memory-limit", 512*1024*1024, "Collector memory limit in bytes")

	// Health check flags
	healthEndpoint  = pflag.String("health-endpoint", "http://localhost:13133/health", "Collector health endpoint")
	healthInterval  = pflag.Duration("health-interval", 10*time.Second, "Health check interval")
	healthTimeout   = pflag.Duration("health-timeout", 5*time.Second, "Health check timeout")
	healthThreshold = pflag.Int("health-threshold", 3, "Consecutive health check failures before restart")

	// Restart flags
	restartPolicy      = pflag.String("restart-policy", "on-failure", "Restart policy (never, on-failure, always)")
	restartMaxRetries  = pflag.Int("restart-max-retries", 10, "Maximum restart attempts")
	restartInitialDelay = pflag.Duration("restart-initial-delay", time.Second, "Initial restart delay")
	restartMaxDelay    = pflag.Duration("restart-max-delay", 5*time.Minute, "Maximum restart delay")
	restartBackoff     = pflag.Float64("restart-backoff", 2.0, "Restart backoff multiplier")

	// Telemetry flags
	telemetryEndpoint = pflag.String("telemetry-endpoint", "http://localhost:4318/v1/metrics", "Telemetry endpoint")
	telemetryInterval = pflag.Duration("telemetry-interval", 10*time.Second, "Telemetry reporting interval")

	// General flags
	logLevel = pflag.String("log-level", "info", "Log level (debug, info, warn, error)")
	logJson  = pflag.Bool("log-json", false, "Output logs in JSON format")
)

func main() {
	pflag.Parse()

	// Setup logger
	logger, err := setupLogger(*logLevel, *logJson)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Build configuration
	config := buildConfig()

	// Create supervisor
	supervisor, err := supervisor.New(config, logger)
	if err != nil {
		logger.Fatal("Failed to create supervisor", zap.Error(err))
	}

	// Create context
	ctx := context.Background()

	// Start supervisor
	if err := supervisor.Start(ctx); err != nil {
		logger.Fatal("Failed to start supervisor", zap.Error(err))
	}

	logger.Info("Supervisor started successfully")

	// Wait for supervisor to finish
	supervisor.Wait()

	logger.Info("Supervisor exited")
}

func setupLogger(level string, jsonOutput bool) (*zap.Logger, error) {
	// Parse log level
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}

	// Build logger config
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zapLevel)
	
	if !jsonOutput {
		config.Encoding = "console"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	return config.Build()
}

func buildConfig() supervisor.Config {
	return supervisor.Config{
		Collector: supervisor.CollectorConfig{
			BinaryPath:      *collectorBinary,
			ConfigPath:      *collectorConfig,
			Args:            *collectorArgs,
			Env:             os.Environ(),
			WorkDir:         *collectorWorkDir,
			MemoryLimit:     *memoryLimit,
			ShutdownTimeout: 30 * time.Second,
		},
		HealthChecker: supervisor.HealthCheckerConfig{
			Endpoint:         *healthEndpoint,
			Interval:         *healthInterval,
			Timeout:          *healthTimeout,
			FailureThreshold: *healthThreshold,
		},
		Restart: restart.Config{
			Policy:            restart.Policy(*restartPolicy),
			MaxRetries:        *restartMaxRetries,
			InitialDelay:      *restartInitialDelay,
			MaxDelay:          *restartMaxDelay,
			BackoffMultiplier: *restartBackoff,
		},
		Telemetry: telemetryclient.Config{
			ServiceName:    "nrdot-supervisor",
			ServiceVersion: "1.0.0",
			Environment:    "production",
			Endpoint:       *telemetryEndpoint,
			Interval:       *telemetryInterval,
			Enabled:        true,
		},
	}
}