package models

import (
	"time"
)

// StatusResponse represents the system status
type StatusResponse struct {
	Status      string            `json:"status"`
	Version     string            `json:"version"`
	Uptime      string            `json:"uptime"`
	StartTime   time.Time         `json:"start_time"`
	Collectors  []CollectorStatus `json:"collectors"`
	ConfigHash  string            `json:"config_hash"`
	LastReload  *time.Time        `json:"last_reload,omitempty"`
	Errors      []ErrorInfo       `json:"errors,omitempty"`
}

// CollectorStatus represents the status of a collector
type CollectorStatus struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// ConfigResponse represents the current configuration
type ConfigResponse struct {
	Active   interface{} `json:"active"`
	Source   string      `json:"source"`
	Version  string      `json:"version"`
	LoadedAt time.Time   `json:"loaded_at"`
}

// ConfigUpdateRequest represents a configuration update request
type ConfigUpdateRequest struct {
	Config  interface{} `json:"config"`
	DryRun  bool        `json:"dry_run,omitempty"`
	Comment string      `json:"comment,omitempty"`
}

// ConfigUpdateResponse represents the response to a configuration update
type ConfigUpdateResponse struct {
	Success     bool              `json:"success"`
	Message     string            `json:"message"`
	Warnings    []string          `json:"warnings,omitempty"`
	Errors      []string          `json:"errors,omitempty"`
	ValidationResult *ValidationResult `json:"validation,omitempty"`
}

// ValidationResult represents configuration validation results
type ValidationResult struct {
	Valid    bool              `json:"valid"`
	Errors   []ValidationError `json:"errors,omitempty"`
	Warnings []string          `json:"warnings,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ReloadRequest represents a configuration reload request
type ReloadRequest struct {
	Force bool `json:"force,omitempty"`
}

// ReloadResponse represents the response to a reload request
type ReloadResponse struct {
	Success      bool      `json:"success"`
	Message      string    `json:"message"`
	ReloadedAt   time.Time `json:"reloaded_at"`
	OldVersion   string    `json:"old_version"`
	NewVersion   string    `json:"new_version"`
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status     string            `json:"status"` // "healthy", "degraded", "unhealthy"
	Components map[string]Health `json:"components"`
	Timestamp  time.Time         `json:"timestamp"`
}

// Health represents component health
type Health struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// MetricsResponse represents Prometheus metrics
// This is typically returned as text/plain in Prometheus format
type MetricsResponse string

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string    `json:"error"`
	Code    string    `json:"code,omitempty"`
	Details string    `json:"details,omitempty"`
	Time    time.Time `json:"time"`
}

// ErrorInfo represents error information
type ErrorInfo struct {
	Component string    `json:"component"`
	Error     string    `json:"error"`
	Count     int       `json:"count"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}

// Constants for status values
const (
	StatusHealthy   = "healthy"
	StatusDegraded  = "degraded"
	StatusUnhealthy = "unhealthy"
	StatusRunning   = "running"
	StatusStopped   = "stopped"
	StatusError     = "error"
	StatusUnknown   = "unknown"
)