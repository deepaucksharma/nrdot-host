package configengine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/newrelic/nrdot-host/nrdot-config-engine/pkg/hooks"
	schema "github.com/newrelic/nrdot-host/nrdot-schema"
	templatelib "github.com/newrelic/nrdot-host/nrdot-template-lib"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// Engine is the main configuration engine that integrates schema validation
// with template generation
type Engine struct {
	logger       *zap.Logger
	validator    *schema.Validator
	generator    *templatelib.Generator
	hookManager  *hooks.Manager
	outputDir    string
	dryRun       bool
	mu           sync.RWMutex
	
	// Current configuration version
	currentVersion string
}

// Config holds the engine configuration
type Config struct {
	// OutputDir is where generated OTel configs will be written
	OutputDir string
	// DryRun if true, validates but doesn't write files
	DryRun bool
	// Logger for the engine
	Logger *zap.Logger
}

// NewEngine creates a new configuration engine
func NewEngine(cfg Config) (*Engine, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	validator, err := schema.NewValidator()
	if err != nil {
		return nil, fmt.Errorf("failed to create validator: %w", err)
	}

	// Generator will be created per config when processing
	// as it needs the parsed config structure

	return &Engine{
		logger:      cfg.Logger,
		validator:   validator,
		generator:   nil, // Will be set during ProcessConfig
		hookManager: hooks.NewManager(),
		outputDir:   cfg.OutputDir,
		dryRun:      cfg.DryRun,
	}, nil
}

// ProcessConfig processes a configuration file through validation and generation
func (e *Engine) ProcessConfig(ctx context.Context, configPath string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.logger.Info("Processing configuration", zap.String("path", configPath))

	// Read the configuration file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Validate the configuration
	config, err := e.validator.ValidateYAML(data)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Generate new version
	oldVersion := e.currentVersion
	newVersion := generateVersion()

	// If dry run, stop here
	if e.dryRun {
		e.logger.Info("Dry run complete - configuration is valid",
			zap.String("version", newVersion))
		return nil
	}

	// Create generator with validated config
	generator := templatelib.NewGenerator(config)
	
	// Generate OTel configuration
	otelConfig, err := generator.Generate()
	if err != nil {
		return fmt.Errorf("failed to generate configuration: %w", err)
	}
	
	// Write OTel config to file
	otelConfigPath := filepath.Join(e.outputDir, "otel-config.yaml")
	otelConfigData, err := yaml.Marshal(otelConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal OTel config: %w", err)
	}
	
	if err := os.MkdirAll(e.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	if err := os.WriteFile(otelConfigPath, otelConfigData, 0644); err != nil {
		return fmt.Errorf("failed to write OTel config: %w", err)
	}
	
	generatedFiles := []string{otelConfigPath}

	// Update current version
	e.currentVersion = newVersion

	// Notify hooks
	event := hooks.ConfigChangeEvent{
		ConfigPath:       configPath,
		OldVersion:       oldVersion,
		NewVersion:       newVersion,
		GeneratedConfigs: generatedFiles,
	}

	if err := e.hookManager.NotifyAll(ctx, event); err != nil {
		e.logger.Error("Failed to notify hooks", zap.Error(err))
		// Don't fail the operation if hooks fail
	}

	e.logger.Info("Configuration processed successfully",
		zap.String("version", newVersion),
		zap.Int("generatedFiles", len(generatedFiles)))

	return nil
}

// ValidateConfig validates a configuration file without generating outputs
func (e *Engine) ValidateConfig(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	_, err = e.validator.ValidateYAML(data)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	return nil
}

// RegisterHook registers a configuration change hook
func (e *Engine) RegisterHook(hook hooks.Hook) {
	e.hookManager.Register(hook)
}

// GetCurrentVersion returns the current configuration version
func (e *Engine) GetCurrentVersion() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.currentVersion
}

// GetOutputDir returns the output directory for generated configs
func (e *Engine) GetOutputDir() string {
	return e.outputDir
}

// generateVersion generates a new version string
func generateVersion() string {
	// In a real implementation, this might use git commit hash or timestamp
	return fmt.Sprintf("v%d", time.Now().Unix())
}

