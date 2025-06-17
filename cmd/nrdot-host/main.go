// Package main provides the unified NRDOT-HOST binary with mode selection
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/newrelic/nrdot-host/nrdot-supervisor"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Version information (set by build flags)
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

// RunMode defines the operating mode
type RunMode string

const (
	ModeAll        RunMode = "all"        // Everything in one process (default)
	ModeAgent      RunMode = "agent"      // Just collector + supervisor
	ModeAPI        RunMode = "api"        // Just API server
	ModeCollector  RunMode = "collector"  // Just collector (standalone)
	ModeVersion    RunMode = "version"    // Print version and exit
)

func main() {
	// Parse flags
	var (
		mode          = flag.String("mode", "all", "Run mode: all, agent, api, collector, version")
		configFile    = flag.String("config", "/etc/nrdot/config.yaml", "Configuration file path")
		collectorPath = flag.String("collector", "/usr/bin/otelcol-nrdot", "Path to collector binary")
		workDir       = flag.String("workdir", "/var/lib/nrdot", "Working directory")
		apiAddr       = flag.String("api-addr", "127.0.0.1:8080", "API server listen address")
		logLevel      = flag.String("log-level", "info", "Log level: debug, info, warn, error")
		logFormat     = flag.String("log-format", "console", "Log format: console, json")
		enableTelemetry = flag.Bool("telemetry", true, "Enable self-telemetry")
	)
	
	flag.Parse()
	
	// Handle version mode
	runMode := RunMode(*mode)
	if runMode == ModeVersion {
		printVersion()
		os.Exit(0)
	}
	
	// Setup logging
	logger := setupLogger(*logLevel, *logFormat)
	defer logger.Sync()
	
	// Log startup info
	logger.Info("Starting NRDOT-HOST",
		zap.String("version", version),
		zap.String("commit", commit),
		zap.String("mode", string(runMode)),
		zap.String("config", *configFile),
	)
	
	// Setup context with signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), 
		os.Interrupt, syscall.SIGTERM)
	defer cancel()
	
	// Run based on mode
	var err error
	switch runMode {
	case ModeAll:
		err = runAll(ctx, logger, *configFile, *collectorPath, *workDir, *apiAddr, *enableTelemetry)
	case ModeAgent:
		err = runAgent(ctx, logger, *configFile, *collectorPath, *workDir, *enableTelemetry)
	case ModeAPI:
		err = runAPI(ctx, logger, *configFile, *apiAddr)
	case ModeCollector:
		err = runCollector(ctx, logger, *configFile)
	default:
		err = fmt.Errorf("unknown mode: %s", runMode)
	}
	
	if err != nil {
		logger.Fatal("Failed to run", zap.Error(err))
	}
	
	logger.Info("NRDOT-HOST shutdown complete")
}

// runAll runs all components in a single process
func runAll(ctx context.Context, logger *zap.Logger, configFile, collectorPath, workDir, apiAddr string, enableTelemetry bool) error {
	logger.Info("Running in ALL mode - unified process")
	
	// Create unified supervisor with everything embedded
	config := supervisor.SupervisorConfig{
		CollectorPath:       collectorPath,
		ConfigPath:          configFile,
		WorkDir:             workDir,
		APIEnabled:          true,
		APIListenAddr:       apiAddr,
		RestartDelay:        5,
		MaxRestarts:         10,
		HealthCheckInterval: 30,
		EnableTelemetry:     enableTelemetry,
		Logger:              logger,
	}
	
	sup, err := supervisor.NewUnifiedSupervisor(config)
	if err != nil {
		return fmt.Errorf("failed to create supervisor: %w", err)
	}
	
	// Start supervisor (includes API server, config engine, collector)
	if err := sup.Start(ctx); err != nil {
		return fmt.Errorf("failed to start supervisor: %w", err)
	}
	
	// Wait for shutdown signal
	<-ctx.Done()
	logger.Info("Received shutdown signal")
	
	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30)
	defer cancel()
	
	return sup.Stop(shutdownCtx)
}

// runAgent runs just the collector and supervisor (no API)
func runAgent(ctx context.Context, logger *zap.Logger, configFile, collectorPath, workDir string, enableTelemetry bool) error {
	logger.Info("Running in AGENT mode - collector only")
	
	config := supervisor.SupervisorConfig{
		CollectorPath:       collectorPath,
		ConfigPath:          configFile,
		WorkDir:             workDir,
		APIEnabled:          false, // No API in agent mode
		RestartDelay:        5,
		MaxRestarts:         10,
		HealthCheckInterval: 30,
		EnableTelemetry:     enableTelemetry,
		Logger:              logger,
	}
	
	sup, err := supervisor.NewUnifiedSupervisor(config)
	if err != nil {
		return fmt.Errorf("failed to create supervisor: %w", err)
	}
	
	if err := sup.Start(ctx); err != nil {
		return fmt.Errorf("failed to start supervisor: %w", err)
	}
	
	<-ctx.Done()
	
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30)
	defer cancel()
	
	return sup.Stop(shutdownCtx)
}

// runAPI runs just the API server (connects to existing supervisor)
func runAPI(ctx context.Context, logger *zap.Logger, configFile, apiAddr string) error {
	logger.Info("Running in API mode - API server only")
	
	// In a real implementation, this would connect to an existing supervisor
	// via Unix socket or network. For now, return error.
	return fmt.Errorf("standalone API mode not yet implemented")
}

// runCollector runs just the collector without supervisor
func runCollector(ctx context.Context, logger *zap.Logger, configFile string) error {
	logger.Info("Running in COLLECTOR mode - standalone collector")
	
	// This would exec the actual OTel collector binary
	// For now, return error
	return fmt.Errorf("standalone collector mode not yet implemented")
}

// setupLogger configures the logger
func setupLogger(level, format string) *zap.Logger {
	// Parse level
	var zapLevel zapcore.Level
	switch strings.ToLower(level) {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn", "warning":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}
	
	// Create config
	var config zap.Config
	if format == "json" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}
	
	config.Level = zap.NewAtomicLevelAt(zapLevel)
	
	logger, err := config.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to create logger: %v", err))
	}
	
	return logger
}

// printVersion prints version information
func printVersion() {
	fmt.Printf("NRDOT-HOST %s\n", version)
	fmt.Printf("  Commit:     %s\n", commit)
	fmt.Printf("  Build Date: %s\n", buildDate)
	fmt.Printf("  Go Version: %s\n", os.Getenv("GO_VERSION"))
	fmt.Printf("\nModes:\n")
	fmt.Printf("  all       - Run all components in one process (default)\n")
	fmt.Printf("  agent     - Run collector with supervisor only\n")
	fmt.Printf("  api       - Run API server only\n")
	fmt.Printf("  collector - Run collector standalone\n")
}