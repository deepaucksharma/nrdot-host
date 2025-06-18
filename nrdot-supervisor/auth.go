package supervisor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/newrelic/nrdot-host/nrdot-common/pkg/auth"
	"go.uber.org/zap"
)

// AuthConfig extends SupervisorConfig with authentication settings
type AuthConfig struct {
	SupervisorConfig
	Auth auth.Config
}

// SetupAuthenticatedAPIServer configures the API server with authentication
func (s *UnifiedSupervisor) SetupAuthenticatedAPIServer(authConfig auth.Config) error {
	// Validate auth config
	if err := authConfig.Validate(); err != nil {
		return fmt.Errorf("invalid auth config: %w", err)
	}

	// Create auth managers based on config
	var jwtManager *auth.JWTManager
	var tokenStore *auth.TokenStore

	if authConfig.Enabled {
		switch authConfig.Type {
		case auth.AuthTypeJWT, auth.AuthTypeBoth:
			// Create JWT manager
			var err error
			jwtManager, err = auth.NewJWTManager(
				authConfig.JWT.SecretKey,
				authConfig.JWT.Duration,
				authConfig.JWT.Issuer,
			)
			if err != nil {
				return fmt.Errorf("failed to create JWT manager: %w", err)
			}
			s.logger.Info("JWT authentication enabled")

		case auth.AuthTypeAPIKey, auth.AuthTypeBoth:
			// Create token store
			tokenStore = auth.NewTokenStore()
			
			// Create default admin token if configured
			if authConfig.DefaultAdmin.APIKey != "" {
				info, err := tokenStore.CreateToken(
					authConfig.DefaultAdmin.Username,
					auth.RoleAdmin,
					"Default admin token",
					authConfig.APIKey.DefaultExpiration,
				)
				if err != nil {
					return fmt.Errorf("failed to create default admin token: %w", err)
				}
				s.logger.Info("Default admin API key created", 
					zap.String("token", info.Token),
					zap.String("user", info.UserID))
			}
			s.logger.Info("API key authentication enabled")
		}
	}

	// Set up routes with authentication
	router := mux.NewRouter()

	// Apply global middleware
	if authConfig.Enabled {
		router.Use(s.createAuthMiddleware(authConfig, jwtManager, tokenStore))
	}

	// Always allow health checks without auth
	router.HandleFunc("/health", s.apiHandlers.Health).Methods("GET")
	router.HandleFunc("/ready", s.apiHandlers.Ready).Methods("GET")

	// API v1 routes with auth
	v1 := router.PathPrefix("/v1").Subrouter()
	
	// Read-only endpoints
	v1.HandleFunc("/status", s.apiHandlers.Status).Methods("GET")
	v1.HandleFunc("/config", s.apiHandlers.GetConfig).Methods("GET")
	v1.HandleFunc("/metrics", s.apiHandlers.Metrics).Methods("GET")

	// Write endpoints (require higher permissions)
	if authConfig.Enabled {
		// These endpoints require operator or admin role
		v1.HandleFunc("/config", s.requireRole(auth.RoleOperator, s.apiHandlers.UpdateConfig)).Methods("POST", "PUT")
		v1.HandleFunc("/config/validate", s.requireRole(auth.RoleOperator, s.apiHandlers.ValidateConfig)).Methods("POST")
		v1.HandleFunc("/control/reload", s.requireRole(auth.RoleOperator, s.handleReload)).Methods("POST")
		v1.HandleFunc("/control/restart", s.requireRole(auth.RoleAdmin, s.handleRestart)).Methods("POST")
	} else {
		// No auth required
		v1.HandleFunc("/config", s.apiHandlers.UpdateConfig).Methods("POST", "PUT")
		v1.HandleFunc("/config/validate", s.apiHandlers.ValidateConfig).Methods("POST")
		v1.HandleFunc("/control/reload", s.handleReload).Methods("POST")
		v1.HandleFunc("/control/restart", s.handleRestart).Methods("POST")
	}

	// Auth management endpoints (only when auth is enabled)
	if authConfig.Enabled {
		authRouter := v1.PathPrefix("/auth").Subrouter()
		
		// JWT endpoints
		if jwtManager != nil {
			authRouter.HandleFunc("/login", s.handleLogin(jwtManager)).Methods("POST")
			authRouter.HandleFunc("/refresh", s.handleRefreshToken(jwtManager)).Methods("POST")
		}
		
		// API key endpoints
		if tokenStore != nil {
			authRouter.HandleFunc("/tokens", s.requireRole(auth.RoleAdmin, s.handleListTokens(tokenStore))).Methods("GET")
			authRouter.HandleFunc("/tokens", s.requireRole(auth.RoleAdmin, s.handleCreateToken(tokenStore))).Methods("POST")
			authRouter.HandleFunc("/tokens/{token}", s.requireRole(auth.RoleAdmin, s.handleRevokeToken(tokenStore))).Methods("DELETE")
		}
	}

	// Update API server with authenticated router
	s.apiServer.Handler = router
	
	return nil
}

// createAuthMiddleware creates the appropriate auth middleware based on config
func (s *UnifiedSupervisor) createAuthMiddleware(config auth.Config, jwtManager *auth.JWTManager, tokenStore *auth.TokenStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for health endpoints
			if r.URL.Path == "/health" || r.URL.Path == "/ready" {
				next.ServeHTTP(w, r)
				return
			}

			authenticated := false
			
			// Try JWT authentication
			if (config.Type == auth.AuthTypeJWT || config.Type == auth.AuthTypeBoth) && jwtManager != nil {
				authHeader := r.Header.Get("Authorization")
				if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
					token := authHeader[7:]
					if claims, err := jwtManager.ValidateToken(token); err == nil {
						// Add claims to context
						ctx := r.Context()
						ctx = context.WithValue(ctx, "claims", claims)
						r = r.WithContext(ctx)
						authenticated = true
					}
				}
			}

			// Try API key authentication if not authenticated yet
			if !authenticated && (config.Type == auth.AuthTypeAPIKey || config.Type == auth.AuthTypeBoth) && tokenStore != nil {
				apiKey := r.Header.Get(config.APIKey.HeaderName)
				if apiKey == "" && config.APIKey.AllowQueryParam {
					apiKey = r.URL.Query().Get(config.APIKey.QueryParamName)
				}
				
				if apiKey != "" {
					if info, err := tokenStore.ValidateToken(apiKey); err == nil {
						// Create claims from token info
						claims := &auth.JWTClaims{
							Role: info.Role,
						}
						claims.Subject = info.UserID
						
						ctx := r.Context()
						ctx = context.WithValue(ctx, "claims", claims)
						ctx = context.WithValue(ctx, "tokenInfo", info)
						r = r.WithContext(ctx)
						authenticated = true
					}
				}
			}

			if !authenticated {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// requireRole creates a middleware that checks for specific role
func (s *UnifiedSupervisor) requireRole(requiredRole string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetClaimsFromContext(r.Context())
		if !ok {
			http.Error(w, "No authentication claims found", http.StatusUnauthorized)
			return
		}

		// Check role hierarchy
		if !hasRequiredRole(claims.Role, requiredRole) {
			http.Error(w, "Insufficient permissions", http.StatusForbidden)
			return
		}

		handler(w, r)
	}
}

// hasRequiredRole checks role hierarchy
func hasRequiredRole(userRole, requiredRole string) bool {
	roleLevel := map[string]int{
		auth.RoleViewer:   1,
		auth.RoleOperator: 2,
		auth.RoleAdmin:    3,
	}

	userLevel, ok1 := roleLevel[userRole]
	requiredLevel, ok2 := roleLevel[requiredRole]

	if !ok1 || !ok2 {
		return false
	}

	return userLevel >= requiredLevel
}