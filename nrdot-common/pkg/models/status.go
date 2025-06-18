// Package models defines shared data structures used across NRDOT-HOST components
package models

import (
	"time"
)

// LifecycleState represents the current state of a component
type LifecycleState string

const (
	StateUnknown      LifecycleState = "unknown"
	StateInitializing LifecycleState = "initializing"
	StateRunning      LifecycleState = "running"
	StateStopping     LifecycleState = "stopping"
	StateStopped      LifecycleState = "stopped"
	StateError        LifecycleState = "error"
	StateReloading    LifecycleState = "reloading"
	StateUpdating     LifecycleState = "updating"
)

// CollectorState represents the operational state of the collector
type CollectorState string

const (
	CollectorStateStarting CollectorState = "starting"
	CollectorStateRunning  CollectorState = "running"
	CollectorStateDegraded CollectorState = "degraded"
	CollectorStateFailed   CollectorState = "failed"
	CollectorStateStopped  CollectorState = "stopped"
)

// CollectorStatus represents the complete status of the OpenTelemetry Collector
type CollectorStatus struct {
	State           CollectorState    `json:"state"`
	Version         string            `json:"version"`
	ConfigVersion   int               `json:"config_version"`
	StartTime       time.Time         `json:"start_time"`
	Uptime          time.Duration     `json:"uptime"`
	LastConfigLoad  time.Time         `json:"last_config_load"`
	RestartCount    int               `json:"restart_count"`
	Pipelines       []PipelineStatus  `json:"pipelines"`
	ResourceMetrics ResourceMetrics   `json:"resource_metrics"`
	LastError       *ErrorInfo        `json:"last_error,omitempty"`
	Features        map[string]bool   `json:"features"`
}

// PipelineStatus represents the status of a single telemetry pipeline
type PipelineStatus struct {
	Name             string                 `json:"name"`
	Type             string                 `json:"type"` // metrics, traces, logs
	State            string                 `json:"state"`
	ComponentsHealth map[string]string      `json:"components_health"`
	Metrics          PipelineMetrics        `json:"metrics"`
	LastError        *ErrorInfo             `json:"last_error,omitempty"`
}

// PipelineMetrics contains runtime metrics for a pipeline
type PipelineMetrics struct {
	ItemsReceived   int64         `json:"items_received"`
	ItemsProcessed  int64         `json:"items_processed"`
	ItemsDropped    int64         `json:"items_dropped"`
	ItemsExported   int64         `json:"items_exported"`
	ProcessingRate  float64       `json:"processing_rate"`  // items/sec
	ErrorRate       float64       `json:"error_rate"`       // errors/sec
	Latency         LatencyMetrics `json:"latency"`
}

// LatencyMetrics represents latency statistics
type LatencyMetrics struct {
	P50  time.Duration `json:"p50"`
	P95  time.Duration `json:"p95"`
	P99  time.Duration `json:"p99"`
	Max  time.Duration `json:"max"`
	Mean time.Duration `json:"mean"`
}

// ResourceMetrics represents resource utilization
type ResourceMetrics struct {
	CPUPercent        float64       `json:"cpu_percent"`
	MemoryBytes       int64         `json:"memory_bytes"`
	MemoryPercent     float64       `json:"memory_percent"`
	GoroutineCount    int           `json:"goroutine_count"`
	OpenFileCount     int           `json:"open_file_count"`
	OpenFiles         int           `json:"open_files"`         // Alias for OpenFileCount
	ThreadCount       int           `json:"thread_count"`       // OS thread count
	NetworkBytesRecv  int64         `json:"network_bytes_recv"`
	NetworkBytesSent  int64         `json:"network_bytes_sent"`
}

// ComponentStatus represents the status of a system component
type ComponentStatus struct {
	Name          string            `json:"name"`
	Type          string            `json:"type"`
	State         LifecycleState    `json:"state"`
	Version       string            `json:"version"`
	StartTime     time.Time         `json:"start_time"`
	LastHeartbeat time.Time         `json:"last_heartbeat"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// SystemStatus represents the overall system status
type SystemStatus struct {
	Timestamp       time.Time          `json:"timestamp"`
	CollectorStatus *CollectorStatus   `json:"collector"`
	Components      []ComponentStatus  `json:"components"`
	SystemInfo      SystemInfo         `json:"system_info"`
	ClusterInfo     *ClusterInfo       `json:"cluster_info,omitempty"`
}

// SystemInfo contains host system information
type SystemInfo struct {
	Hostname       string            `json:"hostname"`
	OS             string            `json:"os"`
	Architecture   string            `json:"architecture"`
	CPUCount       int               `json:"cpu_count"`
	TotalMemory    int64             `json:"total_memory"`
	KernelVersion  string            `json:"kernel_version"`
	ContainerID    string            `json:"container_id,omitempty"`
	CloudProvider  string            `json:"cloud_provider,omitempty"`
	CloudRegion    string            `json:"cloud_region,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
}

// ClusterInfo contains Kubernetes cluster information if available
type ClusterInfo struct {
	ClusterName   string            `json:"cluster_name"`
	NodeName      string            `json:"node_name"`
	Namespace     string            `json:"namespace"`
	PodName       string            `json:"pod_name"`
	Labels        map[string]string `json:"labels,omitempty"`
	Annotations   map[string]string `json:"annotations,omitempty"`
}