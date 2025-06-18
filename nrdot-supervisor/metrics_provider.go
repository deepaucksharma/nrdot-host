package supervisor

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/newrelic/nrdot-host/nrdot-api-server/pkg/handlers"
)

// MetricsCollector collects metrics for the supervisor
type MetricsCollector struct {
	mu sync.RWMutex

	// Request counters
	totalRequests      atomic.Int64
	configReloads      atomic.Int64
	failedReloads      atomic.Int64
	collectorRestarts  atomic.Int64
	
	// Timing metrics
	lastReloadDuration time.Duration
	lastHealthCheck    time.Time
	
	// State metrics
	collectorRunning bool
	apiEnabled       bool
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		lastHealthCheck: time.Now(),
	}
}

// IncrementRequests increments the total request counter
func (m *MetricsCollector) IncrementRequests() {
	m.totalRequests.Add(1)
}

// IncrementConfigReloads increments the config reload counter
func (m *MetricsCollector) IncrementConfigReloads() {
	m.configReloads.Add(1)
}

// IncrementFailedReloads increments the failed reload counter
func (m *MetricsCollector) IncrementFailedReloads() {
	m.failedReloads.Add(1)
}

// IncrementCollectorRestarts increments the collector restart counter
func (m *MetricsCollector) IncrementCollectorRestarts() {
	m.collectorRestarts.Add(1)
}

// SetReloadDuration sets the last reload duration
func (m *MetricsCollector) SetReloadDuration(d time.Duration) {
	m.mu.Lock()
	m.lastReloadDuration = d
	m.mu.Unlock()
}

// SetHealthCheckTime sets the last health check time
func (m *MetricsCollector) SetHealthCheckTime(t time.Time) {
	m.mu.Lock()
	m.lastHealthCheck = t
	m.mu.Unlock()
}

// SetCollectorRunning sets whether the collector is running
func (m *MetricsCollector) SetCollectorRunning(running bool) {
	m.mu.Lock()
	m.collectorRunning = running
	m.mu.Unlock()
}

// SetAPIEnabled sets whether the API is enabled
func (m *MetricsCollector) SetAPIEnabled(enabled bool) {
	m.mu.Lock()
	m.apiEnabled = enabled
	m.mu.Unlock()
}

// GetCustomMetrics implements the MetricsProvider interface
func (m *MetricsCollector) GetCustomMetrics() []handlers.Metric {
	m.mu.RLock()
	collectorRunning := m.collectorRunning
	apiEnabled := m.apiEnabled
	lastReloadDuration := m.lastReloadDuration
	lastHealthCheck := m.lastHealthCheck
	m.mu.RUnlock()

	metrics := []handlers.Metric{
		// Request metrics
		{
			Name:  "nrdot_supervisor_requests_total",
			Help:  "Total number of API requests handled",
			Type:  "counter",
			Value: float64(m.totalRequests.Load()),
		},
		{
			Name:  "nrdot_supervisor_config_reloads_total",
			Help:  "Total number of configuration reloads",
			Type:  "counter",
			Value: float64(m.configReloads.Load()),
		},
		{
			Name:  "nrdot_supervisor_failed_reloads_total",
			Help:  "Total number of failed configuration reloads",
			Type:  "counter",
			Value: float64(m.failedReloads.Load()),
		},
		{
			Name:  "nrdot_supervisor_collector_restarts_total",
			Help:  "Total number of collector process restarts",
			Type:  "counter",
			Value: float64(m.collectorRestarts.Load()),
		},
		
		// State metrics
		{
			Name:  "nrdot_collector_running",
			Help:  "Whether the OpenTelemetry collector is running",
			Type:  "gauge",
			Value: boolToFloat64(collectorRunning),
		},
		{
			Name:  "nrdot_api_enabled",
			Help:  "Whether the API server is enabled",
			Type:  "gauge",
			Value: boolToFloat64(apiEnabled),
		},
		
		// Timing metrics
		{
			Name:  "nrdot_last_reload_duration_seconds",
			Help:  "Duration of the last configuration reload in seconds",
			Type:  "gauge",
			Value: lastReloadDuration.Seconds(),
		},
		{
			Name:  "nrdot_time_since_last_health_check_seconds",
			Help:  "Time since the last health check in seconds",
			Type:  "gauge",
			Value: time.Since(lastHealthCheck).Seconds(),
		},
		
		// Resource metrics
		{
			Name:  "nrdot_supervisor_open_file_descriptors",
			Help:  "Number of open file descriptors",
			Type:  "gauge",
			Value: float64(getOpenFDs()),
		},
		{
			Name:  "nrdot_supervisor_max_file_descriptors",
			Help:  "Maximum number of file descriptors",
			Type:  "gauge",
			Value: float64(getMaxFDs()),
		},
		{
			Name:  "nrdot_supervisor_threads",
			Help:  "Number of OS threads",
			Type:  "gauge",
			Value: float64(runtime.GOMAXPROCS(0)),
		},
	}

	// Add collector process metrics if running
	if collectorRunning {
		// This would be populated from actual collector metrics
		metrics = append(metrics, handlers.Metric{
			Name:  "nrdot_collector_pipelines_running",
			Help:  "Number of running telemetry pipelines",
			Type:  "gauge",
			Value: 3, // placeholder - would be from actual collector status
		})
	}

	return metrics
}

// boolToFloat64 converts a boolean to float64 (1.0 for true, 0.0 for false)
func boolToFloat64(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

// getOpenFDs returns the number of open file descriptors (Linux specific)
func getOpenFDs() int {
	// This is a simplified version - in production you'd read /proc/self/fd
	return -1
}

// getMaxFDs returns the maximum number of file descriptors (Linux specific)
func getMaxFDs() int {
	// This is a simplified version - in production you'd read /proc/self/limits
	return -1
}