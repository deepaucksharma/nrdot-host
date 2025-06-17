package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/newrelic/nrdot-host/nrdot-api-server/pkg/models"
	"go.uber.org/zap"
)

// ConfigHandler handles configuration requests
type ConfigHandler struct {
	logger         *zap.Logger
	configProvider ConfigProvider
	readOnly       bool
}

// ConfigProvider provides configuration operations
type ConfigProvider interface {
	GetCurrentConfig() (interface{}, string, time.Time)
	ValidateConfig(config interface{}) *models.ValidationResult
	UpdateConfig(config interface{}, dryRun bool) error
	ReloadConfig(force bool) error
}

// NewConfigHandler creates a new config handler
func NewConfigHandler(logger *zap.Logger, provider ConfigProvider, readOnly bool) *ConfigHandler {
	return &ConfigHandler{
		logger:         logger,
		configProvider: provider,
		readOnly:       readOnly,
	}
}

// ServeHTTP handles /v1/config requests
func (h *ConfigHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r)
	case http.MethodPost:
		h.handlePost(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGet handles GET /v1/config
func (h *ConfigHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	config, source, loadedAt := h.configProvider.GetCurrentConfig()
	
	response := &models.ConfigResponse{
		Active:   config,
		Source:   source,
		Version:  "1.0", // TODO: Get actual version
		LoadedAt: loadedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode config response", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// handlePost handles POST /v1/config
func (h *ConfigHandler) handlePost(w http.ResponseWriter, r *http.Request) {
	if h.readOnly {
		http.Error(w, "Configuration updates are disabled", http.StatusForbidden)
		return
	}

	// Read request body
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB limit
	if err != nil {
		h.logger.Error("Failed to read request body", zap.Error(err))
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse request
	var req models.ConfigUpdateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.logger.Error("Failed to parse config update request", zap.Error(err))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate configuration
	validation := h.configProvider.ValidateConfig(req.Config)
	
	response := &models.ConfigUpdateResponse{
		ValidationResult: validation,
	}

	if !validation.Valid {
		response.Success = false
		response.Message = "Configuration validation failed"
		
		// Extract error messages
		for _, err := range validation.Errors {
			response.Errors = append(response.Errors, err.Message)
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Apply configuration if not dry run
	if !req.DryRun {
		if err := h.configProvider.UpdateConfig(req.Config, false); err != nil {
			h.logger.Error("Failed to update configuration", zap.Error(err))
			response.Success = false
			response.Message = "Failed to apply configuration"
			response.Errors = []string{err.Error()}
			
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
		
		h.logger.Info("Configuration updated", zap.String("comment", req.Comment))
	}

	// Success response
	response.Success = true
	if req.DryRun {
		response.Message = "Configuration is valid (dry run)"
	} else {
		response.Message = "Configuration updated successfully"
	}
	response.Warnings = validation.Warnings

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ReloadHandler handles configuration reload requests
type ReloadHandler struct {
	logger         *zap.Logger
	configProvider ConfigProvider
	readOnly       bool
}

// NewReloadHandler creates a new reload handler
func NewReloadHandler(logger *zap.Logger, provider ConfigProvider, readOnly bool) *ReloadHandler {
	return &ReloadHandler{
		logger:         logger,
		configProvider: provider,
		readOnly:       readOnly,
	}
}

// ServeHTTP handles POST /v1/reload
func (h *ReloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.readOnly {
		http.Error(w, "Reload operations are disabled", http.StatusForbidden)
		return
	}

	// Parse request
	var req models.ReloadRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.logger.Error("Failed to parse reload request", zap.Error(err))
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
	}

	// Get current config version
	_, _, loadedAt := h.configProvider.GetCurrentConfig()
	oldVersion := loadedAt.Format(time.RFC3339)

	// Perform reload
	if err := h.configProvider.ReloadConfig(req.Force); err != nil {
		h.logger.Error("Failed to reload configuration", zap.Error(err))
		
		response := &models.ReloadResponse{
			Success:    false,
			Message:    "Failed to reload configuration: " + err.Error(),
			OldVersion: oldVersion,
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get new config version
	_, _, newLoadedAt := h.configProvider.GetCurrentConfig()
	
	response := &models.ReloadResponse{
		Success:      true,
		Message:      "Configuration reloaded successfully",
		ReloadedAt:   time.Now(),
		OldVersion:   oldVersion,
		NewVersion:   newLoadedAt.Format(time.RFC3339),
	}

	h.logger.Info("Configuration reloaded", 
		zap.String("old_version", oldVersion),
		zap.String("new_version", response.NewVersion),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}