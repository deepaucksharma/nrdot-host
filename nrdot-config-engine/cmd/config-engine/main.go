package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	configengine "github.com/newrelic/nrdot-host/nrdot-config-engine"
	"github.com/newrelic/nrdot-host/nrdot-config-engine/pkg/hooks"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var (
		configPaths   = flag.String("config", "", "Comma-separated list of config files or directories to watch")
		outputDir     = flag.String("output", "./output", "Output directory for generated configurations")
		dryRun        = flag.Bool("dry-run", false, "Validate configurations without generating output")
		validateOnly  = flag.Bool("validate", false, "Only validate configurations and exit")
		logLevel      = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
		versionFlag   = flag.Bool("version", false, "Print version information")
	)

	flag.Parse()

	if *versionFlag {
		fmt.Printf("nrdot-config-engine %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	// Setup logger
	logger, err := setupLogger(*logLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Parse config paths
	if *configPaths == "" {
		logger.Fatal("No configuration paths specified")
	}

	paths := strings.Split(*configPaths, ",")
	for i := range paths {
		paths[i] = strings.TrimSpace(paths[i])
	}

	// Create manager
	manager, err := configengine.NewManager(configengine.ManagerConfig{
		ConfigDir:   ".",
		OutputDir:   *outputDir,
		MaxVersions: 20,
		Logger:      logger,
		DryRun:      *dryRun,
	})
	if err != nil {
		logger.Fatal("Failed to create manager", zap.Error(err))
	}

	// Validate only mode
	if *validateOnly {
		logger.Info("Running in validation-only mode")
		if err := manager.ValidateAll(paths); err != nil {
			logger.Error("Validation failed", zap.Error(err))
			os.Exit(1)
		}
		logger.Info("All configurations are valid")
		os.Exit(0)
	}

	// Register example hook for logging
	manager.GetEngine().RegisterHook(&loggingHook{logger: logger})

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start manager
	if err := manager.Start(ctx, paths); err != nil {
		logger.Fatal("Failed to start manager", zap.Error(err))
	}

	logger.Info("Configuration engine started",
		zap.Strings("paths", paths),
		zap.String("outputDir", *outputDir),
		zap.Bool("dryRun", *dryRun))

	// Wait for shutdown signal
	select {
	case sig := <-sigChan:
		logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
	case <-ctx.Done():
		logger.Info("Context cancelled")
	}

	// Shutdown
	if err := manager.Stop(); err != nil {
		logger.Error("Failed to stop manager", zap.Error(err))
	}

	logger.Info("Configuration engine stopped")
}

// setupLogger creates a zap logger with the specified level
func setupLogger(level string) (*zap.Logger, error) {
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}

	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zapLevel)
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	return config.Build()
}

// loggingHook is an example hook that logs configuration changes
type loggingHook struct {
	logger *zap.Logger
}

func (h *loggingHook) OnConfigChange(ctx context.Context, event hooks.ConfigChangeEvent) error {
	if event.Error != nil {
		h.logger.Error("Configuration change failed",
			zap.String("config", event.ConfigPath),
			zap.Error(event.Error))
		return nil
	}

	h.logger.Info("Configuration changed",
		zap.String("config", event.ConfigPath),
		zap.String("oldVersion", event.OldVersion),
		zap.String("newVersion", event.NewVersion),
		zap.Int("generatedFiles", len(event.GeneratedConfigs)))

	for _, file := range event.GeneratedConfigs {
		h.logger.Debug("Generated file", zap.String("path", file))
	}

	return nil
}

func (h *loggingHook) Name() string {
	return "LoggingHook"
}