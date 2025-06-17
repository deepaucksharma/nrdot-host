package models

import (
	"time"
)

// ReloadStrategy defines how configuration reloads are performed
type ReloadStrategy string

const (
	ReloadStrategyInPlace    ReloadStrategy = "in_place"    // SIGHUP-style reload
	ReloadStrategyBlueGreen  ReloadStrategy = "blue_green"  // Start new, switch, stop old
	ReloadStrategyGraceful   ReloadStrategy = "graceful"    // Stop and restart
	ReloadStrategyRolling    ReloadStrategy = "rolling"     // For multiple instances
)

// ReloadResult represents the outcome of a reload operation
type ReloadResult struct {
	Success       bool           `json:"success"`
	Strategy      ReloadStrategy `json:"strategy"`
	OldVersion    int            `json:"old_version"`
	NewVersion    int            `json:"new_version"`
	StartTime     time.Time      `json:"start_time"`
	EndTime       time.Time      `json:"end_time"`
	Duration      time.Duration  `json:"duration"`
	Error         *ErrorInfo     `json:"error,omitempty"`
	RollbackInfo  *RollbackInfo  `json:"rollback_info,omitempty"`
}

// RollbackInfo contains information about a rollback operation
type RollbackInfo struct {
	Triggered    bool      `json:"triggered"`
	Reason       string    `json:"reason"`
	FromVersion  int       `json:"from_version"`
	ToVersion    int       `json:"to_version"`
	Success      bool      `json:"success"`
	Timestamp    time.Time `json:"timestamp"`
}

// CollectorUpdate represents a collector binary update request
type CollectorUpdate struct {
	Version      string            `json:"version"`
	DownloadURL  string            `json:"download_url,omitempty"`
	Checksum     string            `json:"checksum"`
	ReleaseNotes string            `json:"release_notes,omitempty"`
	Mandatory    bool              `json:"mandatory"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// UpdateResult represents the outcome of an update operation
type UpdateResult struct {
	Success        bool          `json:"success"`
	OldVersion     string        `json:"old_version"`
	NewVersion     string        `json:"new_version"`
	UpdateDuration time.Duration `json:"update_duration"`
	Downtime       time.Duration `json:"downtime"`
	Error          *ErrorInfo    `json:"error,omitempty"`
}

// BackupInfo represents configuration backup information
type BackupInfo struct {
	ID           string            `json:"id"`
	Timestamp    time.Time         `json:"timestamp"`
	Version      int               `json:"version"`
	Size         int64             `json:"size"`
	Location     string            `json:"location"`
	Description  string            `json:"description,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// CleanupResult represents the outcome of a cleanup operation
type CleanupResult struct {
	Success        bool              `json:"success"`
	ItemsCleaned   int               `json:"items_cleaned"`
	SpaceReclaimed int64             `json:"space_reclaimed"`
	Duration       time.Duration     `json:"duration"`
	Details        map[string]int    `json:"details,omitempty"`
	Error          *ErrorInfo        `json:"error,omitempty"`
}

// OptimizationResult represents the outcome of optimization
type OptimizationResult struct {
	Success          bool                   `json:"success"`
	OptimizationsRun []string               `json:"optimizations_run"`
	Before           ResourceMetrics        `json:"metrics_before"`
	After            ResourceMetrics        `json:"metrics_after"`
	Improvements     map[string]float64     `json:"improvements"`
	Duration         time.Duration          `json:"duration"`
	Recommendations  []string               `json:"recommendations,omitempty"`
}

// FleetStatus represents fleet connectivity status
type FleetStatus struct {
	Connected       bool              `json:"connected"`
	FleetURL        string            `json:"fleet_url"`
	AgentID         string            `json:"agent_id"`
	LastSync        time.Time         `json:"last_sync"`
	NextSync        time.Time         `json:"next_sync"`
	ConfigVersion   int               `json:"config_version"`
	FleetVersion    int               `json:"fleet_version"`
	ConnectionError *ErrorInfo        `json:"connection_error,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// SyncResult represents the outcome of fleet synchronization
type SyncResult struct {
	Success        bool              `json:"success"`
	ConfigUpdated  bool              `json:"config_updated"`
	NewVersion     int               `json:"new_version,omitempty"`
	Changes        []string          `json:"changes,omitempty"`
	SyncTime       time.Time         `json:"sync_time"`
	Duration       time.Duration     `json:"duration"`
	Error          *ErrorInfo        `json:"error,omitempty"`
}