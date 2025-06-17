// Package interfaces defines the core provider interfaces used throughout NRDOT-HOST
// for inter-component communication and dependency injection.
package interfaces

import (
	"context"
	"time"

	"github.com/newrelic/nrdot-host/nrdot-common/pkg/models"
)

// StatusProvider provides real-time status information about the collector and system.
type StatusProvider interface {
	// GetStatus returns the current status of the collector
	GetStatus(ctx context.Context) (*models.CollectorStatus, error)

	// GetPipelineStatus returns status for a specific pipeline
	GetPipelineStatus(ctx context.Context, pipelineName string) (*models.PipelineStatus, error)

	// Subscribe allows components to receive status updates
	Subscribe(ctx context.Context, subscriber StatusSubscriber) error
}

// StatusSubscriber receives status update notifications
type StatusSubscriber interface {
	OnStatusChange(status *models.CollectorStatus)
}

// ConfigProvider manages configuration for the system.
type ConfigProvider interface {
	// GetCurrentConfig returns the active configuration
	GetCurrentConfig(ctx context.Context) (*models.Config, error)

	// ApplyConfig validates and applies a new configuration
	ApplyConfig(ctx context.Context, config *models.ConfigUpdate) (*models.ConfigResult, error)

	// ValidateConfig checks if a configuration is valid without applying it
	ValidateConfig(ctx context.Context, config []byte) (*models.ValidationResult, error)

	// GetConfigHistory returns previous configurations
	GetConfigHistory(ctx context.Context, limit int) ([]*models.ConfigVersion, error)

	// RollbackConfig reverts to a previous configuration version
	RollbackConfig(ctx context.Context, version int) error
}

// HealthProvider provides health and liveness information.
type HealthProvider interface {
	// GetHealth returns overall system health
	GetHealth(ctx context.Context) (*models.HealthStatus, error)

	// GetComponentHealth returns health for a specific component
	GetComponentHealth(ctx context.Context, component string) (*models.ComponentHealth, error)

	// RegisterHealthCheck adds a new health check
	RegisterHealthCheck(name string, check HealthCheck)
}

// HealthCheck is a function that performs a health check
type HealthCheck func(ctx context.Context) error

// MetricsProvider provides internal metrics about the system.
type MetricsProvider interface {
	// GetMetrics returns current internal metrics
	GetMetrics(ctx context.Context) (*models.Metrics, error)

	// GetMetricHistory returns historical metrics
	GetMetricHistory(ctx context.Context, metric string, duration time.Duration) ([]models.MetricPoint, error)
}

// LifecycleProvider manages component lifecycle operations.
type LifecycleProvider interface {
	// Start initiates the component
	Start(ctx context.Context) error

	// Stop gracefully shuts down the component
	Stop(ctx context.Context) error

	// Restart performs a restart operation
	Restart(ctx context.Context) error

	// GetState returns the current lifecycle state
	GetState() models.LifecycleState
}

// EventProvider allows components to emit and subscribe to events.
type EventProvider interface {
	// Emit publishes an event
	Emit(ctx context.Context, event models.Event) error

	// Subscribe registers an event handler
	Subscribe(eventType string, handler EventHandler) error

	// Unsubscribe removes an event handler
	Unsubscribe(eventType string, handler EventHandler) error
}

// EventHandler processes events
type EventHandler func(event models.Event)

// LogProvider provides centralized logging capabilities.
type LogProvider interface {
	// GetLogs retrieves recent log entries
	GetLogs(ctx context.Context, filter models.LogFilter) ([]*models.LogEntry, error)

	// StreamLogs provides real-time log streaming
	StreamLogs(ctx context.Context, filter models.LogFilter) (<-chan *models.LogEntry, error)
}

// Provider is a composite interface for components that provide multiple capabilities.
// Components can implement any subset of these interfaces based on their functionality.
type Provider interface {
	// GetCapabilities returns which provider interfaces this component implements
	GetCapabilities() []string
}