package models

import (
	"time"
)

// HealthState represents the health state of a component
type HealthState string

const (
	HealthStateHealthy   HealthState = "healthy"
	HealthStateDegraded  HealthState = "degraded"
	HealthStateUnhealthy HealthState = "unhealthy"
	HealthStateUnknown   HealthState = "unknown"
)

// HealthStatus represents overall system health
type HealthStatus struct {
	State          HealthState        `json:"state"`
	Timestamp      time.Time          `json:"timestamp"`
	Components     []ComponentHealth  `json:"components"`
	Checks         []HealthCheckResult `json:"checks"`
	Summary        string             `json:"summary"`
	ReadinessProbe bool               `json:"readiness"`
	LivenessProbe  bool               `json:"liveness"`
}

// ComponentHealth represents health of a single component
type ComponentHealth struct {
	Name         string              `json:"name"`
	Type         string              `json:"type"`
	State        HealthState         `json:"state"`
	LastCheck    time.Time           `json:"last_check"`
	Message      string              `json:"message,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
	Dependencies []string            `json:"dependencies,omitempty"`
}

// HealthCheckResult represents result of a health check
type HealthCheckResult struct {
	Name      string          `json:"name"`
	Type      string          `json:"type"` // readiness, liveness, startup
	Status    string          `json:"status"` // pass, fail, warn
	Timestamp time.Time       `json:"timestamp"`
	Duration  time.Duration   `json:"duration"`
	Message   string          `json:"message,omitempty"`
	Error     *ErrorInfo      `json:"error,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// HealthReport represents a comprehensive health report
type HealthReport struct {
	GeneratedAt    time.Time           `json:"generated_at"`
	SystemHealth   *HealthStatus       `json:"system_health"`
	CollectorHealth *CollectorHealth   `json:"collector_health"`
	ResourceHealth  *ResourceHealth    `json:"resource_health"`
	NetworkHealth   *NetworkHealth     `json:"network_health"`
	Recommendations []string           `json:"recommendations,omitempty"`
}

// CollectorHealth represents collector-specific health metrics
type CollectorHealth struct {
	State             HealthState       `json:"state"`
	PipelinesHealthy  int               `json:"pipelines_healthy"`
	PipelinesTotal    int               `json:"pipelines_total"`
	DataLoss          bool              `json:"data_loss"`
	BackPressure      bool              `json:"back_pressure"`
	ConfigErrors      int               `json:"config_errors"`
	ProcessorErrors   map[string]int    `json:"processor_errors"`
	ExporterErrors    map[string]int    `json:"exporter_errors"`
}

// ResourceHealth represents system resource health
type ResourceHealth struct {
	State           HealthState     `json:"state"`
	CPUHealthy      bool            `json:"cpu_healthy"`
	CPUThreshold    float64         `json:"cpu_threshold"`
	MemoryHealthy   bool            `json:"memory_healthy"`
	MemoryThreshold float64         `json:"memory_threshold"`
	DiskHealthy     bool            `json:"disk_healthy"`
	DiskThreshold   float64         `json:"disk_threshold"`
	Alerts          []ResourceAlert `json:"alerts,omitempty"`
}

// ResourceAlert represents a resource-related alert
type ResourceAlert struct {
	Type      string    `json:"type"` // cpu, memory, disk, fd
	Severity  string    `json:"severity"` // warning, critical
	Message   string    `json:"message"`
	Value     float64   `json:"value"`
	Threshold float64   `json:"threshold"`
	Timestamp time.Time `json:"timestamp"`
}

// NetworkHealth represents network connectivity health
type NetworkHealth struct {
	State              HealthState           `json:"state"`
	ExporterReachable  bool                  `json:"exporter_reachable"`
	DNSResolution      bool                  `json:"dns_resolution"`
	InternetConnectivity bool                `json:"internet_connectivity"`
	Endpoints          []EndpointHealth      `json:"endpoints"`
	Latency            map[string]time.Duration `json:"latency,omitempty"`
}

// EndpointHealth represents health of a network endpoint
type EndpointHealth struct {
	Name        string        `json:"name"`
	URL         string        `json:"url"`
	Reachable   bool          `json:"reachable"`
	StatusCode  int           `json:"status_code,omitempty"`
	Latency     time.Duration `json:"latency"`
	LastCheck   time.Time     `json:"last_check"`
	Error       string        `json:"error,omitempty"`
}

// DiagnosticBundle contains comprehensive diagnostic information
type DiagnosticBundle struct {
	ID            string                 `json:"id"`
	GeneratedAt   time.Time              `json:"generated_at"`
	SystemInfo    SystemInfo             `json:"system_info"`
	Configuration map[string]interface{} `json:"configuration"`
	HealthReport  *HealthReport          `json:"health_report"`
	Logs          []LogEntry             `json:"recent_logs"`
	Metrics       map[string]interface{} `json:"metrics_snapshot"`
	Traces        []interface{}          `json:"trace_samples,omitempty"`
	Goroutines    string                 `json:"goroutines,omitempty"`
	MemoryProfile string                 `json:"memory_profile,omitempty"`
	CPUProfile    string                 `json:"cpu_profile,omitempty"`
}

// DebugInfo contains detailed debug information for a component
type DebugInfo struct {
	Component     string                 `json:"component"`
	DebugLevel    string                 `json:"debug_level"`
	InternalState map[string]interface{} `json:"internal_state"`
	Counters      map[string]int64       `json:"counters"`
	Timers        map[string]time.Duration `json:"timers"`
	LastErrors    []ErrorInfo            `json:"last_errors"`
	StackTrace    string                 `json:"stack_trace,omitempty"`
}