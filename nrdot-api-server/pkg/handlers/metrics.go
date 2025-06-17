package handlers

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"go.uber.org/zap"
)

// MetricsHandler handles Prometheus metrics requests
type MetricsHandler struct {
	logger          *zap.Logger
	metricsProvider MetricsProvider
	startTime       time.Time
	version         string
}

// MetricsProvider provides custom metrics
type MetricsProvider interface {
	GetCustomMetrics() []Metric
}

// Metric represents a single metric
type Metric struct {
	Name   string
	Help   string
	Type   string // counter, gauge, histogram
	Value  float64
	Labels map[string]string
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler(logger *zap.Logger, version string, provider MetricsProvider) *MetricsHandler {
	return &MetricsHandler{
		logger:          logger,
		metricsProvider: provider,
		startTime:       time.Now(),
		version:         version,
	}
}

// ServeHTTP handles GET /v1/metrics
func (h *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set Prometheus content type
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	// Write metrics
	h.writeMetrics(w)
}

// writeMetrics writes all metrics in Prometheus format
func (h *MetricsHandler) writeMetrics(w http.ResponseWriter) {
	// Standard Go metrics
	h.writeGoMetrics(w)
	
	// Process metrics
	h.writeProcessMetrics(w)
	
	// API server metrics
	h.writeAPIMetrics(w)
	
	// Custom metrics from provider
	if h.metricsProvider != nil {
		h.writeCustomMetrics(w)
	}
}

// writeGoMetrics writes standard Go runtime metrics
func (h *MetricsHandler) writeGoMetrics(w http.ResponseWriter) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Memory metrics
	fmt.Fprintf(w, "# HELP go_memstats_alloc_bytes Number of bytes allocated and still in use.\n")
	fmt.Fprintf(w, "# TYPE go_memstats_alloc_bytes gauge\n")
	fmt.Fprintf(w, "go_memstats_alloc_bytes %d\n", m.Alloc)

	fmt.Fprintf(w, "# HELP go_memstats_sys_bytes Number of bytes obtained from system.\n")
	fmt.Fprintf(w, "# TYPE go_memstats_sys_bytes gauge\n")
	fmt.Fprintf(w, "go_memstats_sys_bytes %d\n", m.Sys)

	fmt.Fprintf(w, "# HELP go_memstats_gc_cpu_fraction The fraction of this program's available CPU time used by the GC since the program started.\n")
	fmt.Fprintf(w, "# TYPE go_memstats_gc_cpu_fraction gauge\n")
	fmt.Fprintf(w, "go_memstats_gc_cpu_fraction %f\n", m.GCCPUFraction)

	// Goroutines
	fmt.Fprintf(w, "# HELP go_goroutines Number of goroutines that currently exist.\n")
	fmt.Fprintf(w, "# TYPE go_goroutines gauge\n")
	fmt.Fprintf(w, "go_goroutines %d\n", runtime.NumGoroutine())

	// GC metrics
	fmt.Fprintf(w, "# HELP go_gc_duration_seconds A summary of the pause duration of garbage collection cycles.\n")
	fmt.Fprintf(w, "# TYPE go_gc_duration_seconds summary\n")
	if m.NumGC > 0 {
		fmt.Fprintf(w, "go_gc_duration_seconds{quantile=\"0\"} %f\n", float64(m.PauseNs[(m.NumGC+255)%256])/1e9)
		fmt.Fprintf(w, "go_gc_duration_seconds{quantile=\"1\"} %f\n", float64(m.PauseNs[(m.NumGC+255)%256])/1e9)
		fmt.Fprintf(w, "go_gc_duration_seconds_sum %f\n", float64(m.PauseTotalNs)/1e9)
		fmt.Fprintf(w, "go_gc_duration_seconds_count %d\n", m.NumGC)
	}
}

// writeProcessMetrics writes process-level metrics
func (h *MetricsHandler) writeProcessMetrics(w http.ResponseWriter) {
	uptime := time.Since(h.startTime).Seconds()

	fmt.Fprintf(w, "# HELP process_start_time_seconds Start time of the process since unix epoch in seconds.\n")
	fmt.Fprintf(w, "# TYPE process_start_time_seconds gauge\n")
	fmt.Fprintf(w, "process_start_time_seconds %f\n", float64(h.startTime.Unix()))

	fmt.Fprintf(w, "# HELP process_uptime_seconds Number of seconds since the process started.\n")
	fmt.Fprintf(w, "# TYPE process_uptime_seconds gauge\n")
	fmt.Fprintf(w, "process_uptime_seconds %f\n", uptime)
}

// writeAPIMetrics writes API server specific metrics
func (h *MetricsHandler) writeAPIMetrics(w http.ResponseWriter) {
	fmt.Fprintf(w, "# HELP nrdot_api_server_info API server information.\n")
	fmt.Fprintf(w, "# TYPE nrdot_api_server_info gauge\n")
	fmt.Fprintf(w, "nrdot_api_server_info{version=\"%s\"} 1\n", h.version)

	fmt.Fprintf(w, "# HELP nrdot_api_server_up Whether the API server is up.\n")
	fmt.Fprintf(w, "# TYPE nrdot_api_server_up gauge\n")
	fmt.Fprintf(w, "nrdot_api_server_up 1\n")
}

// writeCustomMetrics writes custom metrics from the provider
func (h *MetricsHandler) writeCustomMetrics(w http.ResponseWriter) {
	metrics := h.metricsProvider.GetCustomMetrics()
	
	for _, metric := range metrics {
		// Write help and type
		if metric.Help != "" {
			fmt.Fprintf(w, "# HELP %s %s\n", metric.Name, metric.Help)
		}
		if metric.Type != "" {
			fmt.Fprintf(w, "# TYPE %s %s\n", metric.Name, metric.Type)
		}

		// Write metric value with labels
		labelStr := formatLabels(metric.Labels)
		if labelStr != "" {
			fmt.Fprintf(w, "%s{%s} %f\n", metric.Name, labelStr, metric.Value)
		} else {
			fmt.Fprintf(w, "%s %f\n", metric.Name, metric.Value)
		}
	}
}

// formatLabels formats labels for Prometheus
func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}

	var parts []string
	for k, v := range labels {
		// Escape quotes in label values
		escapedValue := escapeQuotes(v)
		parts = append(parts, fmt.Sprintf(`%s="%s"`, k, escapedValue))
	}

	result := ""
	for i, part := range parts {
		if i > 0 {
			result += ","
		}
		result += part
	}
	return result
}

// escapeQuotes escapes quotes in label values
func escapeQuotes(s string) string {
	result := ""
	for _, c := range s {
		switch c {
		case '"':
			result += `\"`
		case '\\':
			result += `\\`
		case '\n':
			result += `\n`
		default:
			result += string(c)
		}
	}
	return result
}