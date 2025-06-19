package supervisor

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/newrelic/nrdot-host/nrdot-common/pkg/models"
	"go.uber.org/zap"
)

// Handlers provides HTTP handler functions for the API server
type Handlers struct {
	Supervisor *UnifiedSupervisor
	Logger     *zap.Logger
}

// Health handles GET /health
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	health, err := h.Supervisor.GetHealth(ctx)
	if err != nil {
		h.Logger.Error("Failed to get health", zap.Error(err))
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Extract collector state from components
	collectorState := "unknown"
	for _, comp := range health.Components {
		if comp.Name == "collector" {
			collectorState = string(comp.State)
			break
		}
	}
	
	response := map[string]interface{}{
		"status": string(health.State),
		"checks": map[string]interface{}{
			"collector": collectorState,
			"api":       "healthy",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Ready handles GET /ready
func (h *Handlers) Ready(w http.ResponseWriter, r *http.Request) {
	// Simple ready check - if we can respond, we're ready
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ready": true})
}

// Status handles GET /v1/status
func (h *Handlers) Status(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status, err := h.Supervisor.GetStatus(ctx)
	if err != nil {
		h.Logger.Error("Failed to get status", zap.Error(err))
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// GetConfig handles GET /v1/config
func (h *Handlers) GetConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	config, err := h.Supervisor.GetCurrentConfig(ctx)
	if err != nil {
		h.Logger.Error("Failed to get config", zap.Error(err))
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// UpdateConfig handles POST/PUT /v1/config
func (h *Handlers) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	// Decode the update request
	var update models.ConfigUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Apply the configuration
	result, err := h.Supervisor.ApplyConfig(r.Context(), &update)
	if err != nil {
		h.Logger.Error("Failed to apply config", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ValidateConfig handles POST /v1/config/validate
func (h *Handlers) ValidateConfig(w http.ResponseWriter, r *http.Request) {
	// Use ConfigUpdate for validation
	var update models.ConfigUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set dry run to true for validation
	update.DryRun = true
	
	// Validate using supervisor
	result, err := h.Supervisor.ApplyConfig(r.Context(), &update)
	if err != nil {
		h.Logger.Error("Config validation failed", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetMetrics handles GET /metrics (Prometheus format)
func (h *Handlers) GetMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status, err := h.Supervisor.GetStatus(ctx)
	if err != nil {
		h.Logger.Error("Failed to get metrics", zap.Error(err))
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Generate Prometheus-style metrics
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	
	// Helper function to write metric
	writeMetric := func(name, help, metricType string, value interface{}) {
		fmt.Fprintf(w, "# HELP %s %s\n", name, help)
		fmt.Fprintf(w, "# TYPE %s %s\n", name, metricType)
		fmt.Fprintf(w, "%s %v\n", name, value)
	}

	// Write metrics
	writeMetric("nrdot_supervisor_uptime_seconds", "Time since supervisor started", "counter", status.Uptime.Seconds())
	writeMetric("nrdot_collector_running", "Whether the collector is running", "gauge", boolToFloat(status.State == models.CollectorStateRunning))
	writeMetric("nrdot_config_version", "Current configuration version", "gauge", status.ConfigVersion)
	writeMetric("nrdot_restart_count", "Number of collector restarts", "counter", status.RestartCount)
	
	// Resource metrics
	writeMetric("nrdot_cpu_percent", "CPU usage percentage", "gauge", status.ResourceMetrics.CPUPercent)
	writeMetric("nrdot_memory_bytes", "Memory usage in bytes", "gauge", status.ResourceMetrics.MemoryBytes)
	writeMetric("nrdot_goroutine_count", "Number of goroutines", "gauge", status.ResourceMetrics.GoroutineCount)
}

// RestartCollector handles POST /v1/collector/restart
func (h *Handlers) RestartCollector(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	err := h.Supervisor.RestartCollector(ctx, "API request")
	if err != nil {
		h.Logger.Error("Failed to restart collector", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"message": "Collector restart initiated",
	})
}

// ReloadCollector handles POST /v1/collector/reload
func (h *Handlers) ReloadCollector(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	result, err := h.Supervisor.ReloadCollector(ctx, models.ReloadStrategyBlueGreen)
	if err != nil {
		h.Logger.Error("Failed to reload collector", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetVersionHistory handles GET /v1/config/history
func (h *Handlers) GetVersionHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit := 10 // Default limit
	
	// TODO: Parse limit from query params
	
	history, err := h.Supervisor.GetConfigHistory(ctx, limit)
	if err != nil {
		h.Logger.Error("Failed to get version history", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// boolToFloat converts bool to float64 for Prometheus metrics
func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}