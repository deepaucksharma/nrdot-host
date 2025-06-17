package client

import (
	"time"
)

// Status represents the current status of the NRDOT system
type Status struct {
	State            string    `json:"state"`
	Uptime           time.Duration `json:"uptime"`
	ConfigVersion    string    `json:"config_version"`
	Health           Health    `json:"health"`
	LastError        string    `json:"last_error,omitempty"`
	LastErrorTime    time.Time `json:"last_error_time,omitempty"`
	CollectorVersion string    `json:"collector_version"`
}

// Health represents the health status of the system
type Health struct {
	Status     string            `json:"status"`
	Checks     map[string]Check  `json:"checks"`
	LastUpdate time.Time         `json:"last_update"`
}

// Check represents a single health check
type Check struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// ValidationResult represents the result of config validation
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

// OperationResult represents the result of an operation
type OperationResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// ApplyResult represents the result of applying configuration
type ApplyResult struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	PreviousVersion string `json:"previous_version"`
	NewVersion    string `json:"new_version"`
	Error         string `json:"error,omitempty"`
}

// Metrics represents system metrics
type Metrics struct {
	ReceivedMetrics   int64          `json:"received_metrics"`
	SentMetrics       int64          `json:"sent_metrics"`
	DroppedMetrics    int64          `json:"dropped_metrics"`
	ProcessingRate    float64        `json:"processing_rate"`
	ErrorRate         float64        `json:"error_rate"`
	ResourceUsage     ResourceUsage  `json:"resource_usage"`
	PipelineMetrics   map[string]PipelineMetric `json:"pipeline_metrics"`
}

// ResourceUsage represents resource usage metrics
type ResourceUsage struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryMB      float64 `json:"memory_mb"`
	GoroutineCount int    `json:"goroutine_count"`
}

// PipelineMetric represents metrics for a single pipeline
type PipelineMetric struct {
	Received int64   `json:"received"`
	Sent     int64   `json:"sent"`
	Dropped  int64   `json:"dropped"`
	Errors   int64   `json:"errors"`
}