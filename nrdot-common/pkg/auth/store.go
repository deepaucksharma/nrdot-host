package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// TokenStore manages API tokens and their metadata
type TokenStore struct {
	mu     sync.RWMutex
	tokens map[string]*TokenInfo
}

// TokenInfo contains token metadata
type TokenInfo struct {
	Token       string    `json:"token"`
	UserID      string    `json:"user_id"`
	Role        string    `json:"role"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	LastUsed    time.Time `json:"last_used,omitempty"`
	Revoked     bool      `json:"revoked"`
}

// NewTokenStore creates a new token store
func NewTokenStore() *TokenStore {
	return &TokenStore{
		tokens: make(map[string]*TokenInfo),
	}
}

// CreateToken creates a new API token
func (s *TokenStore) CreateToken(userID, role, description string, duration time.Duration) (*TokenInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Create token info
	now := time.Now()
	info := &TokenInfo{
		Token:       token,
		UserID:      userID,
		Role:        role,
		Description: description,
		CreatedAt:   now,
		ExpiresAt:   now.Add(duration),
		Revoked:     false,
	}

	// Store token
	s.tokens[token] = info

	return info, nil
}

// ValidateToken validates an API token
func (s *TokenStore) ValidateToken(token string) (*TokenInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	info, exists := s.tokens[token]
	if !exists {
		return nil, fmt.Errorf("token not found")
	}

	if info.Revoked {
		return nil, fmt.Errorf("token has been revoked")
	}

	if time.Now().After(info.ExpiresAt) {
		return nil, fmt.Errorf("token has expired")
	}

	// Update last used time
	info.LastUsed = time.Now()

	return info, nil
}

// RevokeToken revokes an API token
func (s *TokenStore) RevokeToken(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	info, exists := s.tokens[token]
	if !exists {
		return fmt.Errorf("token not found")
	}

	info.Revoked = true
	return nil
}

// ListTokens returns all tokens for a user
func (s *TokenStore) ListTokens(userID string) []*TokenInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tokens []*TokenInfo
	for _, info := range s.tokens {
		if info.UserID == userID {
			// Create a copy to avoid data races
			tokenCopy := *info
			tokens = append(tokens, &tokenCopy)
		}
	}

	return tokens
}

// CleanupExpired removes expired tokens
func (s *TokenStore) CleanupExpired() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	removed := 0

	for token, info := range s.tokens {
		if now.After(info.ExpiresAt) {
			delete(s.tokens, token)
			removed++
		}
	}

	return removed
}

// GetTokenInfo returns information about a specific token
func (s *TokenStore) GetTokenInfo(token string) (*TokenInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	info, exists := s.tokens[token]
	if !exists {
		return nil, fmt.Errorf("token not found")
	}

	// Return a copy to avoid data races
	infoCopy := *info
	return &infoCopy, nil
}

// Count returns the number of active tokens
func (s *TokenStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	now := time.Now()
	for _, info := range s.tokens {
		if !info.Revoked && now.Before(info.ExpiresAt) {
			count++
		}
	}

	return count
}

// HTTPMiddleware creates an HTTP middleware for API token authentication
func (s *TokenStore) HTTPMiddleware(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// Also check X-API-Key header
				authHeader = r.Header.Get("X-API-Key")
				if authHeader == "" {
					http.Error(w, "API key required", http.StatusUnauthorized)
					return
				}
			}

			// Remove Bearer prefix if present
			token := authHeader
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				token = authHeader[7:]
			}

			// Validate token
			info, err := s.ValidateToken(token)
			if err != nil {
				http.Error(w, fmt.Sprintf("Invalid token: %v", err), http.StatusUnauthorized)
				return
			}

			// Check role if required
			if requiredRole != "" && !hasRequiredRole(info.Role, requiredRole) {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			// Add token info to request context
			ctx := context.WithValue(r.Context(), "tokenInfo", info)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetTokenInfoFromContext extracts token info from request context
func GetTokenInfoFromContext(ctx context.Context) (*TokenInfo, bool) {
	info, ok := ctx.Value("tokenInfo").(*TokenInfo)
	return info, ok
}

// hasRequiredRole checks if user role meets the required role
func hasRequiredRole(userRole, requiredRole string) bool {
	// Role hierarchy: admin > operator > viewer
	roleLevel := map[string]int{
		RoleViewer:   1,
		RoleOperator: 2,
		RoleAdmin:    3,
	}

	userLevel, ok1 := roleLevel[userRole]
	requiredLevel, ok2 := roleLevel[requiredRole]

	if !ok1 || !ok2 {
		return false
	}

	return userLevel >= requiredLevel
}