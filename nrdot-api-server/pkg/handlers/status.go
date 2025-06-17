package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/newrelic/nrdot-host/nrdot-api-server/pkg/models"
	"go.uber.org/zap"
)

// StatusHandler handles status requests
type StatusHandler struct {
	logger    *zap.Logger
	startTime time.Time
	version   string
	
	// Status provider interface (to be implemented by the actual system)
	statusProvider StatusProvider
}

// StatusProvider provides system status information
type StatusProvider interface {
	GetCollectorStatus() []models.CollectorStatus
	GetConfigHash() string
	GetLastReload() *time.Time
	GetErrors() []models.ErrorInfo
}

// NewStatusHandler creates a new status handler
func NewStatusHandler(logger *zap.Logger, version string, provider StatusProvider) *StatusHandler {
	return &StatusHandler{
		logger:         logger,
		startTime:      time.Now(),
		version:        version,
		statusProvider: provider,
	}
}

// ServeHTTP handles GET /v1/status
func (h *StatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Build status response
	status := h.buildStatus()

	// Encode response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		h.logger.Error("Failed to encode status response", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// buildStatus builds the status response
func (h *StatusHandler) buildStatus() *models.StatusResponse {
	uptime := time.Since(h.startTime)
	
	status := &models.StatusResponse{
		Status:     h.determineOverallStatus(),
		Version:    h.version,
		Uptime:     formatDuration(uptime),
		StartTime:  h.startTime,
		ConfigHash: h.statusProvider.GetConfigHash(),
		LastReload: h.statusProvider.GetLastReload(),
	}

	// Get collector status
	status.Collectors = h.statusProvider.GetCollectorStatus()

	// Get errors if any
	if errors := h.statusProvider.GetErrors(); len(errors) > 0 {
		status.Errors = errors
	}

	return status
}

// determineOverallStatus determines the overall system status
func (h *StatusHandler) determineOverallStatus() string {
	collectors := h.statusProvider.GetCollectorStatus()
	
	hasError := false
	hasDegraded := false
	
	for _, c := range collectors {
		switch c.Status {
		case models.StatusError:
			hasError = true
		case models.StatusDegraded:
			hasDegraded = true
		}
	}

	if hasError {
		return models.StatusUnhealthy
	}
	if hasDegraded {
		return models.StatusDegraded
	}
	return models.StatusHealthy
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return formatUnit(days, "day") + " " + 
			formatUnit(hours, "hour") + " " + 
			formatUnit(minutes, "minute")
	}
	if hours > 0 {
		return formatUnit(hours, "hour") + " " + 
			formatUnit(minutes, "minute") + " " + 
			formatUnit(seconds, "second")
	}
	if minutes > 0 {
		return formatUnit(minutes, "minute") + " " + 
			formatUnit(seconds, "second")
	}
	return formatUnit(seconds, "second")
}

// formatUnit formats a single time unit
func formatUnit(value int, unit string) string {
	if value == 1 {
		return "1 " + unit
	}
	return fmt.Sprintf("%d %ss", value, unit)
}