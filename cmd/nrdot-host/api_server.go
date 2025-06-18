package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/newrelic/nrdot-host/nrdot-common/pkg/models"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

// standaloneAPIServer runs the API server in standalone mode
type standaloneAPIServer struct {
	logger     *zap.Logger
	listenAddr string
	configFile string
	httpServer *http.Server
	
	// Connection to supervisor/collector
	collectorEndpoint string
	supervisorSocket  string
}

// Start starts the standalone API server
func (s *standaloneAPIServer) Start(ctx context.Context) error {
	// Load configuration to find collector/supervisor endpoints
	if err := s.loadConfig(); err != nil {
		s.logger.Warn("Failed to load config, using defaults", zap.Error(err))
		// Use defaults
		s.collectorEndpoint = "http://localhost:8888"
		s.supervisorSocket = "/var/run/nrdot/supervisor.sock"
	}
	
	// Setup routes
	router := mux.NewRouter()
	
	// Health endpoints (always available)
	router.HandleFunc("/health", s.handleHealth).Methods("GET")
	router.HandleFunc("/ready", s.handleReady).Methods("GET")
	
	// API v1 routes
	v1 := router.PathPrefix("/v1").Subrouter()
	
	// Status endpoint - queries collector
	v1.HandleFunc("/status", s.handleStatus).Methods("GET")
	
	// Config endpoints - read-only in standalone mode
	v1.HandleFunc("/config", s.handleGetConfig).Methods("GET")
	v1.HandleFunc("/config/validate", s.handleValidateConfig).Methods("POST")
	
	// Metrics endpoint - proxies to collector
	v1.HandleFunc("/metrics", s.handleMetrics).Methods("GET")
	
	// Info endpoint
	v1.HandleFunc("/info", s.handleInfo).Methods("GET")
	
	// Apply middleware
	router.Use(s.loggingMiddleware)
	router.Use(s.corsMiddleware)
	
	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:         s.listenAddr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	// Start server in goroutine
	go func() {
		s.logger.Info("Starting standalone API server", zap.String("addr", s.listenAddr))
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("API server error", zap.Error(err))
		}
	}()
	
	return nil
}

// Stop stops the API server
func (s *standaloneAPIServer) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// loadConfig loads configuration from file
func (s *standaloneAPIServer) loadConfig() error {
	data, err := ioutil.ReadFile(s.configFile)
	if err != nil {
		return err
	}
	
	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}
	
	// Extract API server config
	if apiConfig, ok := config["api_server"].(map[string]interface{}); ok {
		if endpoint, ok := apiConfig["collector_endpoint"].(string); ok {
			s.collectorEndpoint = endpoint
		}
		if socket, ok := apiConfig["supervisor_socket"].(string); ok {
			s.supervisorSocket = socket
		}
	}
	
	return nil
}

// Handler implementations

func (s *standaloneAPIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Simple health check - API server is running
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"mode":   "api",
	})
}

func (s *standaloneAPIServer) handleReady(w http.ResponseWriter, r *http.Request) {
	// Check if we can connect to collector
	resp, err := http.Get(s.collectorEndpoint + "/")
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "not ready",
			"error":  err.Error(),
		})
		return
	}
	resp.Body.Close()
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ready",
	})
}

func (s *standaloneAPIServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	// Query collector for status
	resp, err := http.Get(s.collectorEndpoint + "/")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query collector: %v", err), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	
	// For now, return a simple status
	// In production, this would parse collector metrics and status
	status := models.CollectorStatus{
		State:         models.CollectorStateRunning,
		Version:       version,
		ConfigVersion: 1,
		StartTime:     time.Now().Add(-1 * time.Hour), // Placeholder
		Uptime:        time.Hour,
		Health: models.HealthStatus{
			Healthy: true,
			LastCheck: time.Now(),
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *standaloneAPIServer) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	// Read current config file
	data, err := ioutil.ReadFile(s.configFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read config: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/yaml")
	w.Write(data)
}

func (s *standaloneAPIServer) handleValidateConfig(w http.ResponseWriter, r *http.Request) {
	// Read config from request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	
	// Basic YAML validation
	var config map[string]interface{}
	if err := yaml.Unmarshal(body, &config); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid": false,
			"error": err.Error(),
		})
		return
	}
	
	// TODO: Add more sophisticated validation
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid": true,
	})
}

func (s *standaloneAPIServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	// Proxy request to collector metrics endpoint
	resp, err := http.Get(s.collectorEndpoint + "/metrics")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get metrics: %v", err), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	
	// Copy headers
	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	
	// Copy status code
	w.WriteHeader(resp.StatusCode)
	
	// Copy body
	body, _ := ioutil.ReadAll(resp.Body)
	w.Write(body)
}

func (s *standaloneAPIServer) handleInfo(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"version": version,
		"commit":  commit,
		"build":   buildDate,
		"mode":    "api",
		"endpoints": map[string]string{
			"health":  "/health",
			"ready":   "/ready",
			"status":  "/v1/status",
			"config":  "/v1/config",
			"metrics": "/v1/metrics",
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// Middleware

func (s *standaloneAPIServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(wrapped, r)
		
		s.logger.Info("API request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", wrapped.statusCode),
			zap.Duration("duration", time.Since(start)),
			zap.String("remote", r.RemoteAddr),
		)
	})
}

func (s *standaloneAPIServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}