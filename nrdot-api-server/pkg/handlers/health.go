package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/newrelic/nrdot-host/nrdot-api-server/pkg/models"
	"go.uber.org/zap"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	logger         *zap.Logger
	healthProvider HealthProvider
}

// HealthProvider provides health information
type HealthProvider interface {
	GetComponentHealth() map[string]models.Health
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(logger *zap.Logger, provider HealthProvider) *HealthHandler {
	return &HealthHandler{
		logger:         logger,
		healthProvider: provider,
	}
}

// ServeHTTP handles GET /v1/health
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Build health response
	health := h.buildHealth()

	// Set appropriate status code
	statusCode := http.StatusOK
	if health.Status == models.StatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	} else if health.Status == models.StatusDegraded {
		statusCode = http.StatusOK // Still return 200 for degraded
	}

	// Encode response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(health); err != nil {
		h.logger.Error("Failed to encode health response", zap.Error(err))
		// Can't change status code after WriteHeader
	}
}

// buildHealth builds the health response
func (h *HealthHandler) buildHealth() *models.HealthResponse {
	components := h.healthProvider.GetComponentHealth()
	
	return &models.HealthResponse{
		Status:     h.determineOverallHealth(components),
		Components: components,
		Timestamp:  time.Now(),
	}
}

// determineOverallHealth determines overall health from component health
func (h *HealthHandler) determineOverallHealth(components map[string]models.Health) string {
	if len(components) == 0 {
		return models.StatusHealthy
	}

	hasUnhealthy := false
	hasDegraded := false

	for _, health := range components {
		switch health.Status {
		case models.StatusUnhealthy:
			hasUnhealthy = true
		case models.StatusDegraded:
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		return models.StatusUnhealthy
	}
	if hasDegraded {
		return models.StatusDegraded
	}
	return models.StatusHealthy
}