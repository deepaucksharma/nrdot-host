// Package main provides the unified NRDOT-HOST binary with mode selection
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/newrelic/nrdot-host/nrdot-common/pkg/auth"
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
		enableAuth    = flag.Bool("auth", false, "Enable API authentication")
		authType      = flag.String("auth-type", "jwt", "Authentication type: jwt, api-key, both")
		authSecret    = flag.String("auth-secret", "", "Authentication secret key (auto-generated if empty)")
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
	
	// Build auth config
	authConfig := buildAuthConfig(*enableAuth, *authType, *authSecret)
	
	// Run based on mode
	var err error
	switch runMode {
	case ModeAll:
		err = runAll(ctx, logger, *configFile, *collectorPath, *workDir, *apiAddr, *enableTelemetry, authConfig)
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
func runAll(ctx context.Context, logger *zap.Logger, configFile, collectorPath, workDir, apiAddr string, enableTelemetry bool, authConfig auth.Config) error {
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
	
	// Configure authentication if enabled
	if authConfig.Enabled {
		logger.Info("Configuring API authentication", 
			zap.String("type", authConfig.Type),
			zap.Bool("jwt", authConfig.Type == auth.AuthTypeJWT || authConfig.Type == auth.AuthTypeBoth),
			zap.Bool("api-key", authConfig.Type == auth.AuthTypeAPIKey || authConfig.Type == auth.AuthTypeBoth))
		
		if err := sup.SetupAuthenticatedAPIServer(authConfig); err != nil {
			return fmt.Errorf("failed to setup authentication: %w", err)
		}
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
	
	// Create a minimal API server configuration
	// In production, this would read from config file to get supervisor connection details
	
	// For standalone API mode, we create a lightweight server that can:
	// 1. Connect to existing collector via its API
	// 2. Provide read-only access to status and metrics
	// 3. Forward configuration changes to the supervisor
	
	server := &standaloneAPIServer{
		logger:     logger,
		listenAddr: apiAddr,
		configFile: configFile,
	}
	
	if err := server.Start(ctx); err != nil {
		return fmt.Errorf("failed to start API server: %w", err)
	}
	
	// Wait for shutdown
	<-ctx.Done()
	logger.Info("Shutting down API server")
	
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	return server.Stop(shutdownCtx)
}

// runCollector runs just the collector without supervisor
func runCollector(ctx context.Context, logger *zap.Logger, configFile string) error {
	logger.Info("Running in COLLECTOR mode - standalone collector")
	
	// Find the collector binary
	collectorPath := os.Getenv("OTELCOL_BINARY")
	if collectorPath == "" {
		// Try common locations
		paths := []string{
			"/usr/bin/otelcol-nrdot",
			"/usr/local/bin/otelcol-nrdot",
			"./bin/otelcol-nrdot",
			"otelcol-nrdot",
		}
		
		for _, path := range paths {
			if _, err := os.Stat(path); err == nil {
				collectorPath = path
				break
			}
		}
		
		if collectorPath == "" {
			return fmt.Errorf("collector binary not found in PATH or common locations")
		}
	}
	
	logger.Info("Found collector binary", zap.String("path", collectorPath))
	
	// Build collector command
	cmd := exec.Command(collectorPath, "--config", configFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	
	// Start collector
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start collector: %w", err)
	}
	
	logger.Info("Collector started", zap.Int("pid", cmd.Process.Pid))
	
	// Handle signals
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	
	select {
	case <-ctx.Done():
		logger.Info("Stopping collector")
		// Try graceful shutdown first
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			logger.Warn("Failed to send SIGTERM, killing process", zap.Error(err))
			cmd.Process.Kill()
		}
		
		// Wait for process to exit
		select {
		case err := <-done:
			if err != nil {
				logger.Error("Collector exited with error", zap.Error(err))
			}
		case <-time.After(10 * time.Second):
			logger.Warn("Collector did not stop gracefully, force killing")
			cmd.Process.Kill()
			<-done
		}
		
		return nil
		
	case err := <-done:
		if err != nil {
			return fmt.Errorf("collector exited unexpectedly: %w", err)
		}
		return nil
	}
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

// buildAuthConfig builds authentication configuration from command line flags
func buildAuthConfig(enabled bool, authType, secretKey string) auth.Config {
	if !enabled {
		return auth.DefaultAuthConfig() // Disabled by default
	}
	
	config := auth.DefaultAuthConfig()
	config.Enabled = true
	config.Type = authType
	
	// Set JWT configuration
	if authType == auth.AuthTypeJWT || authType == auth.AuthTypeBoth {
		config.JWT.SecretKey = secretKey
		config.JWT.Duration = 24 * time.Hour
		config.JWT.Issuer = "nrdot-host"
	}
	
	// Set API key configuration
	if authType == auth.AuthTypeAPIKey || authType == auth.AuthTypeBoth {
		config.APIKey.HeaderName = "X-API-Key"
		config.APIKey.DefaultExpiration = 90 * 24 * time.Hour
		
		// Generate a default admin API key if none provided
		if config.DefaultAdmin.APIKey == "" && secretKey != "" {
			config.DefaultAdmin.APIKey = secretKey
		}
	}
	
	return config
}