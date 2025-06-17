package configengine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ConfigVersion represents a version of the configuration
type ConfigVersion struct {
	Version   string
	Timestamp time.Time
	ConfigPath string
	GeneratedFiles []string
}

// Manager manages the configuration lifecycle including versioning and rollback
type Manager struct {
	logger   *zap.Logger
	engine   *Engine
	watcher  *Watcher
	
	// Version history
	versions []ConfigVersion
	maxVersions int
	
	// State management
	mu       sync.RWMutex
	running  bool
	cancelFn context.CancelFunc
}

// ManagerConfig holds manager configuration
type ManagerConfig struct {
	// ConfigDir is the directory containing configuration files
	ConfigDir string
	// OutputDir is where generated configs will be written
	OutputDir string
	// MaxVersions is the maximum number of versions to keep
	MaxVersions int
	// Logger for the manager
	Logger *zap.Logger
	// DryRun mode
	DryRun bool
}

// NewManager creates a new configuration manager
func NewManager(cfg ManagerConfig) (*Manager, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	if cfg.MaxVersions == 0 {
		cfg.MaxVersions = 10
	}

	// Create output directory if it doesn't exist
	if !cfg.DryRun {
		if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Create engine
	engine, err := NewEngine(Config{
		OutputDir: cfg.OutputDir,
		DryRun:    cfg.DryRun,
		Logger:    cfg.Logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create engine: %w", err)
	}

	// Create watcher
	watcher, err := NewWatcher(WatcherConfig{
		Engine: engine,
		Logger: cfg.Logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	return &Manager{
		logger:      cfg.Logger,
		engine:      engine,
		watcher:     watcher,
		versions:    make([]ConfigVersion, 0, cfg.MaxVersions),
		maxVersions: cfg.MaxVersions,
	}, nil
}

// Start begins managing configuration lifecycle
func (m *Manager) Start(ctx context.Context, configPaths []string) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return fmt.Errorf("manager already running")
	}
	m.running = true
	
	// Create cancellable context
	ctx, cancel := context.WithCancel(ctx)
	m.cancelFn = cancel
	m.mu.Unlock()

	// Process initial configurations
	for _, path := range configPaths {
		if err := m.processInitialConfig(ctx, path); err != nil {
			m.logger.Error("Failed to process initial config",
				zap.String("path", path),
				zap.Error(err))
			// Continue with other configs
		}

		// Add to watcher
		if err := m.watcher.Watch(path); err != nil {
			m.logger.Error("Failed to watch config",
				zap.String("path", path),
				zap.Error(err))
		}
	}

	// Start watching for changes
	go func() {
		if err := m.watcher.Start(ctx); err != nil {
			m.logger.Error("Watcher stopped with error", zap.Error(err))
		}
	}()

	m.logger.Info("Configuration manager started",
		zap.Int("configs", len(configPaths)))

	return nil
}

// Stop stops the configuration manager
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	if m.cancelFn != nil {
		m.cancelFn()
	}

	if err := m.watcher.Close(); err != nil {
		return fmt.Errorf("failed to close watcher: %w", err)
	}

	m.running = false
	m.logger.Info("Configuration manager stopped")

	return nil
}

// processInitialConfig processes a configuration file at startup
func (m *Manager) processInitialConfig(ctx context.Context, path string) error {
	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat config file: %w", err)
	}

	if info.IsDir() {
		// If directory, find all config files
		configs, err := m.findConfigFiles(path)
		if err != nil {
			return fmt.Errorf("failed to find config files: %w", err)
		}

		for _, configPath := range configs {
			if err := m.engine.ProcessConfig(ctx, configPath); err != nil {
				m.logger.Error("Failed to process config",
					zap.String("path", configPath),
					zap.Error(err))
			} else {
				m.addVersion(configPath, m.engine.GetCurrentVersion(), []string{})
			}
		}
	} else {
		// Process single file
		if err := m.engine.ProcessConfig(ctx, path); err != nil {
			return err
		}
		m.addVersion(path, m.engine.GetCurrentVersion(), []string{})
	}

	return nil
}

// findConfigFiles finds all configuration files in a directory
func (m *Manager) findConfigFiles(dir string) ([]string, error) {
	var configs []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && isConfigFile(path) {
			configs = append(configs, path)
		}

		return nil
	})

	return configs, err
}

// addVersion adds a new version to the history
func (m *Manager) addVersion(configPath, version string, generatedFiles []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.versions = append(m.versions, ConfigVersion{
		Version:        version,
		Timestamp:      time.Now(),
		ConfigPath:     configPath,
		GeneratedFiles: generatedFiles,
	})

	// Trim old versions
	if len(m.versions) > m.maxVersions {
		m.versions = m.versions[len(m.versions)-m.maxVersions:]
	}
}

// GetVersionHistory returns the configuration version history
func (m *Manager) GetVersionHistory() []ConfigVersion {
	m.mu.RLock()
	defer m.mu.RUnlock()

	history := make([]ConfigVersion, len(m.versions))
	copy(history, m.versions)
	return history
}

// GetEngine returns the configuration engine
func (m *Manager) GetEngine() *Engine {
	return m.engine
}

// GetWatcher returns the file watcher
func (m *Manager) GetWatcher() *Watcher {
	return m.watcher
}

// IsRunning returns whether the manager is currently running
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// ValidateAll validates all configuration files without processing
func (m *Manager) ValidateAll(configPaths []string) error {
	var errors []error

	for _, path := range configPaths {
		info, err := os.Stat(path)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to stat %s: %w", path, err))
			continue
		}

		if info.IsDir() {
			configs, err := m.findConfigFiles(path)
			if err != nil {
				errors = append(errors, fmt.Errorf("failed to find configs in %s: %w", path, err))
				continue
			}

			for _, configPath := range configs {
				if err := m.engine.ValidateConfig(configPath); err != nil {
					errors = append(errors, fmt.Errorf("validation failed for %s: %w", configPath, err))
				}
			}
		} else {
			if err := m.engine.ValidateConfig(path); err != nil {
				errors = append(errors, fmt.Errorf("validation failed for %s: %w", path, err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation errors: %v", errors)
	}

	return nil
}