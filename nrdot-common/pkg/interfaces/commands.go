// Package interfaces defines command interfaces for supervisor control operations
package interfaces

import (
	"context"
	"time"

	"github.com/newrelic/nrdot-host/nrdot-common/pkg/models"
)

// SupervisorCommander provides control operations for the supervisor
type SupervisorCommander interface {
	// ReloadCollector triggers a configuration reload
	ReloadCollector(ctx context.Context, strategy models.ReloadStrategy) (*models.ReloadResult, error)

	// RestartCollector performs a full restart
	RestartCollector(ctx context.Context, reason string) error

	// StopCollector gracefully stops the collector
	StopCollector(ctx context.Context, timeout time.Duration) error

	// StartCollector starts the collector with given config
	StartCollector(ctx context.Context, configPath string) error

	// UpdateCollector performs a collector binary update
	UpdateCollector(ctx context.Context, update *models.CollectorUpdate) (*models.UpdateResult, error)
}

// ConfigCommander provides configuration management operations
type ConfigCommander interface {
	// GenerateConfig creates an OTel config from user config
	GenerateConfig(ctx context.Context, userConfig []byte) (*models.GeneratedConfig, error)

	// DiffConfigs shows differences between two configurations
	DiffConfigs(ctx context.Context, oldVersion, newVersion int) (*models.ConfigDiff, error)

	// ExportConfig exports the current configuration
	ExportConfig(ctx context.Context, format string) ([]byte, error)

	// ImportConfig imports a configuration from external source
	ImportConfig(ctx context.Context, source string, data []byte) (*models.ConfigResult, error)
}

// DiagnosticCommander provides diagnostic and troubleshooting operations
type DiagnosticCommander interface {
	// CollectDiagnostics gathers system diagnostic information
	CollectDiagnostics(ctx context.Context) (*models.DiagnosticBundle, error)

	// RunHealthCheck performs a comprehensive health check
	RunHealthCheck(ctx context.Context) (*models.HealthReport, error)

	// GetDebugInfo returns detailed debug information
	GetDebugInfo(ctx context.Context, component string) (*models.DebugInfo, error)

	// EnableDebugMode toggles debug logging
	EnableDebugMode(ctx context.Context, enable bool, duration time.Duration) error
}

// MaintenanceCommander provides maintenance operations
type MaintenanceCommander interface {
	// BackupConfiguration creates a configuration backup
	BackupConfiguration(ctx context.Context) (*models.BackupInfo, error)

	// RestoreConfiguration restores from a backup
	RestoreConfiguration(ctx context.Context, backupID string) error

	// CleanupOldData removes old logs, metrics, etc.
	CleanupOldData(ctx context.Context, before time.Time) (*models.CleanupResult, error)

	// OptimizePerformance runs performance optimization
	OptimizePerformance(ctx context.Context) (*models.OptimizationResult, error)
}

// FleetCommander provides fleet management operations (future)
type FleetCommander interface {
	// RegisterWithFleet registers this agent with fleet manager
	RegisterWithFleet(ctx context.Context, fleetURL string) error

	// GetFleetStatus returns fleet connectivity status
	GetFleetStatus(ctx context.Context) (*models.FleetStatus, error)

	// SyncWithFleet synchronizes configuration with fleet
	SyncWithFleet(ctx context.Context) (*models.SyncResult, error)
}

// Commander is a composite interface for all command operations
type Commander interface {
	SupervisorCommander
	ConfigCommander
	DiagnosticCommander
	MaintenanceCommander
}