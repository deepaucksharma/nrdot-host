package models

import (
	"time"
)

// EventType represents the type of system event
type EventType string

const (
	// Lifecycle events
	EventTypeStarted         EventType = "component.started"
	EventTypeStopped         EventType = "component.stopped"
	EventTypeReloaded        EventType = "component.reloaded"
	EventTypeUpdated         EventType = "component.updated"
	EventTypeCrashed         EventType = "component.crashed"
	
	// Configuration events
	EventTypeConfigChanged   EventType = "config.changed"
	EventTypeConfigValidated EventType = "config.validated"
	EventTypeConfigRejected  EventType = "config.rejected"
	EventTypeConfigRolledBack EventType = "config.rolled_back"
	
	// Health events
	EventTypeHealthChanged   EventType = "health.changed"
	EventTypeHealthDegraded  EventType = "health.degraded"
	EventTypeHealthRecovered EventType = "health.recovered"
	
	// Resource events
	EventTypeResourceHigh    EventType = "resource.high"
	EventTypeResourceNormal  EventType = "resource.normal"
	EventTypeResourceExhausted EventType = "resource.exhausted"
	
	// Data events
	EventTypeDataLoss        EventType = "data.loss"
	EventTypeBackpressure    EventType = "data.backpressure"
	EventTypeCardinalityHigh EventType = "data.cardinality_high"
	
	// Security events
	EventTypeSecurityViolation EventType = "security.violation"
	EventTypeAuthFailure      EventType = "security.auth_failure"
	EventTypeCertExpiring     EventType = "security.cert_expiring"
)

// Event represents a system event
type Event struct {
	ID          string                 `json:"id"`
	Type        EventType              `json:"type"`
	Timestamp   time.Time              `json:"timestamp"`
	Component   string                 `json:"component"`
	Severity    EventSeverity          `json:"severity"`
	Summary     string                 `json:"summary"`
	Details     string                 `json:"details,omitempty"`
	Source      EventSource            `json:"source"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Error       *ErrorInfo             `json:"error,omitempty"`
}

// EventSeverity represents the severity of an event
type EventSeverity string

const (
	EventSeverityInfo     EventSeverity = "info"
	EventSeverityWarning  EventSeverity = "warning"
	EventSeverityError    EventSeverity = "error"
	EventSeverityCritical EventSeverity = "critical"
)

// EventSource contains information about event origin
type EventSource struct {
	Component string `json:"component"`
	Host      string `json:"host"`
	Process   string `json:"process,omitempty"`
	Version   string `json:"version,omitempty"`
}

// LogEntry represents a log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Component string                 `json:"component"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Error     *ErrorInfo             `json:"error,omitempty"`
}

// LogFilter defines criteria for log filtering
type LogFilter struct {
	StartTime  time.Time `json:"start_time,omitempty"`
	EndTime    time.Time `json:"end_time,omitempty"`
	Levels     []string  `json:"levels,omitempty"`
	Components []string  `json:"components,omitempty"`
	Contains   string    `json:"contains,omitempty"`
	Limit      int       `json:"limit,omitempty"`
}

// Metrics represents internal system metrics
type Metrics struct {
	Timestamp   time.Time                      `json:"timestamp"`
	Counters    map[string]int64               `json:"counters"`
	Gauges      map[string]float64             `json:"gauges"`
	Histograms  map[string]HistogramMetric     `json:"histograms"`
	Rates       map[string]float64             `json:"rates"`
}

// HistogramMetric represents histogram metric data
type HistogramMetric struct {
	Count  int64   `json:"count"`
	Sum    float64 `json:"sum"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Mean   float64 `json:"mean"`
	StdDev float64 `json:"stddev"`
	P50    float64 `json:"p50"`
	P90    float64 `json:"p90"`
	P95    float64 `json:"p95"`
	P99    float64 `json:"p99"`
}

// MetricPoint represents a single metric data point
type MetricPoint struct {
	Timestamp time.Time              `json:"timestamp"`
	Value     float64                `json:"value"`
	Labels    map[string]string      `json:"labels,omitempty"`
}