package supervisor

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/newrelic/nrdot-host/nrdot-common/pkg/auth"
	"go.uber.org/zap"
)

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Role      string    `json:"role"`
}

// TokenRequest represents a token creation request
type TokenRequest struct {
	UserID      string        `json:"user_id"`
	Role        string        `json:"role"`
	Description string        `json:"description"`
	Duration    time.Duration `json:"duration,omitempty"`
}

// TokenResponse represents a token response
type TokenResponse struct {
	Token       string    `json:"token"`
	UserID      string    `json:"user_id"`
	Role        string    `json:"role"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// handleLogin handles user login requests
func (s *UnifiedSupervisor) handleLogin(jwtManager *auth.JWTManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// TODO: Implement proper user authentication
		// For now, we'll use a simple hardcoded check
		// In production, this should validate against a user database
		
		var role string
		switch req.Username {
		case "admin":
			if req.Password != "admin123" { // This should be hashed in production
				http.Error(w, "Invalid credentials", http.StatusUnauthorized)
				return
			}
			role = auth.RoleAdmin
		case "operator":
			if req.Password != "operator123" {
				http.Error(w, "Invalid credentials", http.StatusUnauthorized)
				return
			}
			role = auth.RoleOperator
		case "viewer":
			if req.Password != "viewer123" {
				http.Error(w, "Invalid credentials", http.StatusUnauthorized)
				return
			}
			role = auth.RoleViewer
		default:
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		// Generate JWT token
		token, err := jwtManager.GenerateToken(req.Username, role)
		if err != nil {
			s.logger.Error("Failed to generate token", zap.Error(err))
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		// Send response
		resp := LoginResponse{
			Token:     token,
			ExpiresAt: time.Now().Add(24 * time.Hour), // Should match JWT duration
			Role:      role,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// handleRefreshToken handles token refresh requests
func (s *UnifiedSupervisor) handleRefreshToken(jwtManager *auth.JWTManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get current token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || len(authHeader) <= 7 || authHeader[:7] != "Bearer " {
			http.Error(w, "Bearer token required", http.StatusBadRequest)
			return
		}

		oldToken := authHeader[7:]
		
		// Refresh the token
		newToken, err := jwtManager.RefreshToken(oldToken)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Validate new token to get expiration
		claims, err := jwtManager.ValidateToken(newToken)
		if err != nil {
			http.Error(w, "Failed to validate new token", http.StatusInternalServerError)
			return
		}

		resp := LoginResponse{
			Token:     newToken,
			ExpiresAt: claims.ExpiresAt.Time,
			Role:      claims.Role,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// handleListTokens lists all API tokens
func (s *UnifiedSupervisor) handleListTokens(store *auth.TokenStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user ID from query parameter (optional)
		userID := r.URL.Query().Get("user_id")
		
		var tokens []*auth.TokenInfo
		if userID != "" {
			tokens = store.ListTokens(userID)
		} else {
			// For admin, list all tokens
			// This is a simplified implementation
			tokens = store.ListTokens("")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokens)
	}
}

// handleCreateToken creates a new API token
func (s *UnifiedSupervisor) handleCreateToken(store *auth.TokenStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req TokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate role
		switch req.Role {
		case auth.RoleAdmin, auth.RoleOperator, auth.RoleViewer:
			// Valid roles
		default:
			http.Error(w, "Invalid role", http.StatusBadRequest)
			return
		}

		// Use default duration if not specified
		duration := req.Duration
		if duration == 0 {
			duration = 90 * 24 * time.Hour // 90 days
		}

		// Create token
		info, err := store.CreateToken(req.UserID, req.Role, req.Description, duration)
		if err != nil {
			s.logger.Error("Failed to create token", zap.Error(err))
			http.Error(w, "Failed to create token", http.StatusInternalServerError)
			return
		}

		resp := TokenResponse{
			Token:       info.Token,
			UserID:      info.UserID,
			Role:        info.Role,
			Description: info.Description,
			CreatedAt:   info.CreatedAt,
			ExpiresAt:   info.ExpiresAt,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}
}

// handleRevokeToken revokes an API token
func (s *UnifiedSupervisor) handleRevokeToken(store *auth.TokenStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		token := vars["token"]
		
		if token == "" {
			http.Error(w, "Token required", http.StatusBadRequest)
			return
		}

		if err := store.RevokeToken(token); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}