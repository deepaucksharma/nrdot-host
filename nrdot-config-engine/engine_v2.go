package configengine

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/newrelic/nrdot-host/nrdot-common/pkg/interfaces"
	"github.com/newrelic/nrdot-host/nrdot-common/pkg/models"
	"github.com/newrelic/nrdot-host/nrdot-config-engine/internal/schema"
	"github.com/newrelic/nrdot-host/nrdot-config-engine/internal/templates"
	"github.com/newrelic/nrdot-host/nrdot-config-engine/pkg/hooks"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// versionRecord stores both the version metadata and the actual config
type versionRecord struct {
	Version    models.ConfigVersion
	UserConfig string
}

// EngineV2 is the unified configuration engine that consolidates
// schema validation, template generation, and config management
type EngineV2 struct {
	logger        *zap.Logger
	validator     *schema.Validator
	generator     *templates.Generator
	hookManager   *hooks.Manager
	
	mu             sync.RWMutex
	versions       []models.ConfigVersion
	versionMap     map[int]*versionRecord
	currentVersion int
	currentConfig  *models.Config
	currentOTel    string
	
	// Options
	maxVersions   int
	enableBackup  bool
}

// ConfigV2 holds the engine configuration
type ConfigV2 struct {
	Logger       *zap.Logger
	MaxVersions  int  // Maximum versions to keep in history
	EnableBackup bool // Enable automatic backups
}

// NewEngineV2 creates a new unified configuration engine
func NewEngineV2(cfg ConfigV2) (*EngineV2, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	
	if cfg.MaxVersions <= 0 {
		cfg.MaxVersions = 10
	}

	validator := schema.NewValidator()
	generator := templates.NewGenerator()

	return &EngineV2{
		logger:       cfg.Logger,
		validator:    validator,
		generator:    generator,
		hookManager:  hooks.NewManager(),
		versions:     make([]models.ConfigVersion, 0),
		versionMap:   make(map[int]*versionRecord),
		maxVersions:  cfg.MaxVersions,
		enableBackup: cfg.EnableBackup,
	}, nil
}

// ProcessUserConfig implements the unified configuration processing
func (e *EngineV2) ProcessUserConfig(ctx context.Context, userConfig []byte) (*models.GeneratedConfig, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Step 1: Validate user configuration
	validatedConfig, err := e.validator.Validate(userConfig)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Step 2: Generate OTel configuration from validated config
	otelConfig, templatesUsed, err := e.generator.Generate(validatedConfig)
	if err != nil {
		return nil, fmt.Errorf("generation failed: %w", err)
	}

	// Step 3: Marshal OTel config to YAML
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(otelConfig); err != nil {
		return nil, fmt.Errorf("failed to encode OTel config: %w", err)
	}
	otelYAML := buf.String()

	// Step 4: Calculate hash
	hash := e.calculateHash(otelYAML)

	// Step 5: Create result
	result := &models.GeneratedConfig{
		OTelConfig:   otelYAML,
		Hash:         hash,
		GeneratedAt:  time.Now(),
		Templates:    templatesUsed,
		Metadata: map[string]string{
			"generator_version": "2.0",
			"schema_version":    e.validator.GetSchemaVersion(),
		},
	}

	// Store current config
	e.currentConfig = validatedConfig
	e.currentOTel = otelYAML

	return result, nil
}

// ApplyConfig implements the ConfigProvider interface
func (e *EngineV2) ApplyConfig(ctx context.Context, update *models.ConfigUpdate) (*models.ConfigResult, error) {
	e.logger.Info("Applying configuration",
		zap.String("source", update.Source),
		zap.Bool("dryRun", update.DryRun))

	// Validate the configuration
	validationResult := &models.ValidationResult{Valid: true}
	_, err := e.validateUserConfig(update.Config, update.Format)
	if err != nil {
		validationResult.Valid = false
		validationResult.Errors = []models.ValidationError{
			{
				Path:    "/",
				Message: err.Error(),
				Code:    "VALIDATION_FAILED",
			},
		}
		
		return &models.ConfigResult{
			Success:          false,
			ValidationResult: validationResult,
			Error: models.NewError(
				models.ErrCodeConfigInvalid,
				"Configuration validation failed",
				models.ErrorCategoryConfig,
				models.SeverityError,
			).WithDetails(err.Error()),
		}, nil
	}

	// If dry run, return validation result only
	if update.DryRun {
		return &models.ConfigResult{
			Success:          true,
			ValidationResult: validationResult,
		}, nil
	}

	// Generate new configuration
	generated, err := e.ProcessUserConfig(ctx, update.Config)
	if err != nil {
		return &models.ConfigResult{
			Success: false,
			Error: models.NewError(
				models.ErrCodeInternalError,
				"Failed to generate configuration",
				models.ErrorCategoryInternal,
				models.SeverityError,
			).WithDetails(err.Error()),
		}, err
	}

	// Create new version
	e.mu.Lock()
	newVersion := e.currentVersion + 1
	configVersion := models.ConfigVersion{
		Version:     newVersion,
		AppliedAt:   time.Now(),
		Source:      update.Source,
		Author:      update.Author,
		Description: update.Description,
		Hash:        generated.Hash,
		Size:        int64(len(update.Config)),
		Metadata:    update.Metadata,
	}
	
	// Create version record with config
	record := &versionRecord{
		Version:    configVersion,
		UserConfig: string(update.Config),
	}
	
	// Add to version history
	e.versions = append(e.versions, configVersion)
	e.versionMap[newVersion] = record
	e.currentVersion = newVersion
	
	// Trim old versions if needed
	if len(e.versions) > e.maxVersions {
		oldVersion := e.versions[0]
		delete(e.versionMap, oldVersion.Version)
		e.versions = e.versions[1:]
	}
	e.mu.Unlock()

	// Notify hooks
	event := hooks.ConfigChangeEvent{
		OldVersion: fmt.Sprintf("%d", newVersion-1),
		NewVersion: fmt.Sprintf("%d", newVersion),
		GeneratedConfigs: []string{
			fmt.Sprintf("version-%d.yaml", newVersion),
		},
	}
	
	if err := e.hookManager.NotifyAll(ctx, event); err != nil {
		e.logger.Warn("Failed to notify hooks", zap.Error(err))
	}

	return &models.ConfigResult{
		Success:          true,
		Version:          newVersion,
		ValidationResult: validationResult,
		AppliedAt:        configVersion.AppliedAt,
	}, nil
}

// ValidateConfig implements the ConfigProvider interface
func (e *EngineV2) ValidateConfig(ctx context.Context, config []byte) (*models.ValidationResult, error) {
	_, err := e.validator.Validate(config)
	if err != nil {
		return &models.ValidationResult{
			Valid: false,
			Errors: []models.ValidationError{
				{
					Path:    "/",
					Message: err.Error(),
					Code:    "VALIDATION_FAILED",
				},
			},
		}, nil
	}

	return &models.ValidationResult{
		Valid: true,
		Info:  []string{"Configuration is valid"},
	}, nil
}

// GetCurrentConfig implements the ConfigProvider interface
func (e *EngineV2) GetCurrentConfig(ctx context.Context) (*models.Config, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	if e.currentConfig == nil {
		return nil, models.NewError(
			models.ErrCodeConfigMissing,
			"No configuration loaded",
			models.ErrorCategoryConfig,
			models.SeverityWarning,
		)
	}
	
	return e.currentConfig, nil
}

// GetConfigHistory implements the ConfigProvider interface
func (e *EngineV2) GetConfigHistory(ctx context.Context, limit int) ([]*models.ConfigVersion, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	if limit <= 0 || limit > len(e.versions) {
		limit = len(e.versions)
	}
	
	// Return most recent versions
	start := len(e.versions) - limit
	if start < 0 {
		start = 0
	}
	
	result := make([]*models.ConfigVersion, 0, limit)
	for i := start; i < len(e.versions); i++ {
		v := e.versions[i]
		result = append(result, &v)
	}
	
	return result, nil
}

// RollbackConfig implements the ConfigProvider interface
func (e *EngineV2) RollbackConfig(ctx context.Context, version int) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	// Find the version
	var targetVersion *models.ConfigVersion
	for _, v := range e.versions {
		if v.Version == version {
			targetVersion = &v
			break
		}
	}
	
	if targetVersion == nil {
		return models.NewError(
			models.ErrCodeResourceNotFound,
			fmt.Sprintf("Version %d not found", version),
			models.ErrorCategoryConfig,
			models.SeverityError,
		)
	}
	
	// In a real implementation, we would restore the config from backup
	// For now, just log the action
	e.logger.Info("Rolling back configuration",
		zap.Int("targetVersion", version),
		zap.String("hash", targetVersion.Hash))
	
	return nil
}

// RegisterHook registers a configuration change hook
func (e *EngineV2) RegisterHook(hook hooks.Hook) {
	e.hookManager.Register(hook)
}

// validateUserConfig validates user configuration in various formats
func (e *EngineV2) validateUserConfig(data []byte, format string) (*models.Config, error) {
	switch format {
	case "yaml", "yml":
		return e.validator.Validate(data)
	case "json":
		// Convert JSON to YAML first
		var obj interface{}
		if err := yaml.Unmarshal(data, &obj); err != nil {
			return nil, err
		}
		yamlData, err := yaml.Marshal(obj)
		if err != nil {
			return nil, err
		}
		return e.validator.Validate(yamlData)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// calculateHash calculates SHA256 hash of content
func (e *EngineV2) calculateHash(content string) string {
	h := sha256.New()
	h.Write([]byte(content))
	return hex.EncodeToString(h.Sum(nil))
}

// GetCapabilities returns the provider capabilities
func (e *EngineV2) GetCapabilities() []string {
	return []string{"ConfigProvider", "ConfigCommander"}
}

// DiffConfigs compares two configuration versions
func (e *EngineV2) DiffConfigs(ctx context.Context, oldVersion, newVersion int) (*models.ConfigDiff, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Check if versions exist
	ver1, exists1 := e.versionMap[oldVersion]
	ver2, exists2 := e.versionMap[newVersion]
	
	if !exists1 {
		return nil, fmt.Errorf("old version not found: %d", oldVersion)
	}
	if !exists2 {
		return nil, fmt.Errorf("new version not found: %d", newVersion)
	}

	// Create diff result
	diff := &models.ConfigDiff{
		OldVersion: oldVersion,
		NewVersion: newVersion,
		Added:      []string{},
		Removed:    []string{},
		Modified:   []string{},
	}

	// Simple comparison - just check if config changed
	if ver1.UserConfig != ver2.UserConfig {
		diff.Modified = append(diff.Modified, "configuration")
		diff.Summary = fmt.Sprintf("Configuration changed from version %d to %d", oldVersion, newVersion)
	} else {
		diff.Summary = "No changes detected"
	}

	return diff, nil
}

// ExportConfig exports the current configuration
func (e *EngineV2) ExportConfig(ctx context.Context, format string) ([]byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Get current version
	if e.currentVersion == 0 {
		return nil, fmt.Errorf("no configuration available")
	}

	ver, exists := e.versionMap[e.currentVersion]
	if !exists {
		return nil, fmt.Errorf("current version not found: %d", e.currentVersion)
	}

	// Return the user config in requested format
	// For now, we just return the raw config as it's already in YAML
	if format != "yaml" && format != "" {
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}

	return []byte(ver.UserConfig), nil
}

// GenerateConfig creates an OTel config from user config
func (e *EngineV2) GenerateConfig(ctx context.Context, userConfig []byte) (*models.GeneratedConfig, error) {
	return e.ProcessUserConfig(ctx, userConfig)
}

// ImportConfig imports a configuration from external source
func (e *EngineV2) ImportConfig(ctx context.Context, source string, data []byte) (*models.ConfigResult, error) {
	// Create a config update request
	update := &models.ConfigUpdate{
		Config: data,
		Format: "yaml",
		Source: source,
		DryRun: false,
	}

	// Apply the configuration
	return e.ApplyConfig(ctx, update)
}

// GetVersionHistory returns the configuration version history
func (e *EngineV2) GetVersionHistory(ctx context.Context, limit int) ([]*models.ConfigVersion, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Return up to limit versions
	versions := make([]*models.ConfigVersion, 0, limit)
	start := len(e.versions) - limit
	if start < 0 {
		start = 0
	}

	for i := start; i < len(e.versions); i++ {
		v := e.versions[i]
		versions = append(versions, &v)
	}

	return versions, nil
}

// Ensure EngineV2 implements the interfaces
var (
	_ interfaces.ConfigProvider  = (*EngineV2)(nil)
	_ interfaces.ConfigCommander = (*EngineV2)(nil)
)