package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/newrelic/nrdot-host/nrdot-privileged-helper/pkg/socket"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	defaultSocketPath = "/var/run/nrdot/privileged-helper.sock"
	defaultLogPath    = "/var/log/nrdot/privileged-helper.log"
)

func main() {
	var (
		socketPath = flag.String("socket", defaultSocketPath, "Unix domain socket path")
		logPath    = flag.String("log", defaultLogPath, "Log file path")
		debug      = flag.Bool("debug", false, "Enable debug logging")
		version    = flag.Bool("version", false, "Show version")
	)
	flag.Parse()

	if *version {
		fmt.Println("nrdot-privileged-helper v1.0.0")
		os.Exit(0)
	}

	// Initialize logger
	logger, err := initLogger(*logPath, *debug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Log startup
	logger.Info("Starting nrdot-privileged-helper",
		zap.String("version", "v1.0.0"),
		zap.String("socket", *socketPath),
		zap.Int("pid", os.Getpid()),
		zap.Int("uid", os.Getuid()),
		zap.Int("gid", os.Getgid()),
	)

	// Check if running with appropriate privileges
	if os.Getuid() != 0 && os.Geteuid() != 0 {
		logger.Warn("Not running as root or with setuid privileges. Some operations may fail.")
	}

	// Create server
	server, err := socket.NewServer(*socketPath, logger)
	if err != nil {
		logger.Fatal("Failed to create server", zap.Error(err))
	}

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

	// Wait for shutdown signal
	<-ctx.Done()

	// Stop server
	if err := server.Stop(); err != nil {
		logger.Error("Failed to stop server cleanly", zap.Error(err))
	}

	logger.Info("Privileged helper stopped")
}

// initLogger initializes the logger
func initLogger(logPath string, debug bool) (*zap.Logger, error) {
	// Ensure log directory exists
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Configure logger
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{logPath}
	config.ErrorOutputPaths = []string{logPath}
	
	if debug {
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	} else {
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	// Use human-readable timestamps
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return logger, nil
}