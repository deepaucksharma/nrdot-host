package hooks

import (
	"context"
	"fmt"
)

// ConfigChangeEvent represents a configuration change event
type ConfigChangeEvent struct {
	// ConfigPath is the path to the configuration file that changed
	ConfigPath string
	// OldVersion is the previous version of the configuration
	OldVersion string
	// NewVersion is the new version of the configuration
	NewVersion string
	// GeneratedConfigs contains paths to generated OTel configuration files
	GeneratedConfigs []string
	// Error contains any error that occurred during processing
	Error error
}

// Hook interface for configuration reload notifications
type Hook interface {
	// OnConfigChange is called when configuration changes
	OnConfigChange(ctx context.Context, event ConfigChangeEvent) error
	// Name returns the hook name for logging
	Name() string
}

// HookFunc is a function adapter for the Hook interface
type HookFunc func(ctx context.Context, event ConfigChangeEvent) error

// OnConfigChange implements the Hook interface
func (f HookFunc) OnConfigChange(ctx context.Context, event ConfigChangeEvent) error {
	return f(ctx, event)
}

// Name implements the Hook interface
func (f HookFunc) Name() string {
	return "HookFunc"
}

// Manager manages configuration change hooks
type Manager struct {
	hooks []Hook
}

// NewManager creates a new hook manager
func NewManager() *Manager {
	return &Manager{
		hooks: make([]Hook, 0),
	}
}

// Register adds a hook to the manager
func (m *Manager) Register(hook Hook) {
	m.hooks = append(m.hooks, hook)
}

// NotifyAll notifies all registered hooks of a configuration change
func (m *Manager) NotifyAll(ctx context.Context, event ConfigChangeEvent) error {
	var errs []error
	for _, hook := range m.hooks {
		if err := hook.OnConfigChange(ctx, event); err != nil {
			errs = append(errs, fmt.Errorf("hook %s failed: %w", hook.Name(), err))
		}
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("hook notifications failed: %v", errs)
	}
	return nil
}