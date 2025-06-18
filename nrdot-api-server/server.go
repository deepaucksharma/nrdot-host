package apiserver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/newrelic/nrdot-host/nrdot-api-server/pkg/handlers"
	"github.com/newrelic/nrdot-host/nrdot-api-server/pkg/middleware"
	"go.uber.org/zap"
)

// Server represents the API server
type Server struct {
	config     Config
	logger     *zap.Logger
	router     *mux.Router
	httpServer *http.Server
	
	// Providers
	statusProvider handlers.StatusProvider
	healthProvider handlers.HealthProvider
	configProvider handlers.ConfigProvider
	metricsProvider handlers.MetricsProvider
}

// Config represents server configuration
type Config struct {
	Host        string
	Port        int
	ReadOnly    bool
	Version     string
	EnableCORS  bool
	EnableDebug bool
}

// NewServer creates a new API server
func NewServer(config Config, logger *zap.Logger) *Server {
	s := &Server{
		config: config,
		logger: logger,
		router: mux.NewRouter(),
	}

	// Setup routes
	s.setupRoutes()

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		Handler:      s.buildHandler(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// SetProviders sets the various providers
func (s *Server) SetProviders(
	status handlers.StatusProvider,
	health handlers.HealthProvider,
	config handlers.ConfigProvider,
	metrics handlers.MetricsProvider,
) {
	s.statusProvider = status
	s.healthProvider = health
	s.configProvider = config
	s.metricsProvider = metrics
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// API v1 routes
	v1 := s.router.PathPrefix("/v1").Subrouter()

	// Status endpoint
	statusHandler := handlers.NewStatusHandler(s.logger, s.config.Version, s.statusProvider)
	v1.Handle("/status", statusHandler).Methods("GET")

	// Health endpoint
	healthHandler := handlers.NewHealthHandler(s.logger, s.healthProvider)
	v1.Handle("/health", healthHandler).Methods("GET")

	// Config endpoints
	configHandler := handlers.NewConfigHandler(s.logger, s.configProvider, s.config.ReadOnly)
	v1.Handle("/config", configHandler).Methods("GET", "POST")

	// Reload endpoint
	reloadHandler := handlers.NewReloadHandler(s.logger, s.configProvider, s.config.ReadOnly)
	v1.Handle("/reload", reloadHandler).Methods("POST")

	// Metrics endpoint
	metricsHandler := handlers.NewMetricsHandler(s.logger, s.config.Version, s.metricsProvider)
	v1.Handle("/metrics", metricsHandler).Methods("GET")

	// Root health check (for simple monitoring)
	s.router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "NRDOT API Server v%s\n", s.config.Version)
	}).Methods("GET")

	// Prometheus metrics endpoint at root (for standard Prometheus scraping)
	rootMetricsHandler := handlers.NewMetricsHandler(s.logger, s.config.Version, s.metricsProvider)
	s.router.Handle("/metrics", rootMetricsHandler).Methods("GET")
}

// buildHandler builds the HTTP handler with middleware
func (s *Server) buildHandler() http.Handler {
	// Start with the router
	handler := http.Handler(s.router)

	// Add middleware in reverse order (innermost first)
	
	// Request ID
	handler = middleware.RequestIDMiddleware()(handler)

	// Recovery (catch panics)
	handler = middleware.RecoveryMiddleware(s.logger)(handler)

	// CORS if enabled
	if s.config.EnableCORS {
		handler = middleware.CORSMiddleware()(handler)
	}

	// Localhost only restriction
	handler = middleware.LocalhostOnlyMiddleware(s.logger)(handler)

	// Logging (outermost)
	handler = middleware.LoggingMiddleware(s.logger)(handler)

	return handler
}

// Start starts the API server
func (s *Server) Start(ctx context.Context) error {
	// Verify providers are set
	if s.statusProvider == nil || s.healthProvider == nil || 
	   s.configProvider == nil || s.metricsProvider == nil {
		return fmt.Errorf("providers not set")
	}

	// Verify localhost only binding
	ip := net.ParseIP(s.config.Host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("API server must bind to localhost only, got: %s", s.config.Host)
	}

	s.logger.Info("Starting API server",
		zap.String("address", s.httpServer.Addr),
		zap.Bool("read_only", s.config.ReadOnly),
		zap.String("version", s.config.Version),
	)

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		return s.Shutdown(context.Background())
	case err := <-errCh:
		return fmt.Errorf("server failed to start: %w", err)
	case <-time.After(100 * time.Millisecond):
		// Give server time to fail fast if port is already in use
		select {
		case err := <-errCh:
			return fmt.Errorf("server failed to start: %w", err)
		default:
			s.logger.Info("API server started successfully")
			return nil
		}
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down API server")

	// Set a timeout if context doesn't have one
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	s.logger.Info("API server shut down successfully")
	return nil
}

// Wait blocks until the server is shut down
func (s *Server) Wait() error {
	// This is handled by Start() method
	// This method is here for compatibility
	return nil
}

// GetAddress returns the server address
func (s *Server) GetAddress() string {
	return s.httpServer.Addr
}