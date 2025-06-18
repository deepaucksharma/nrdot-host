package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims represents the claims in a JWT token
type JWTClaims struct {
	jwt.RegisteredClaims
	Role        string   `json:"role"`
	Permissions []string `json:"permissions,omitempty"`
}

// Role definitions
const (
	RoleAdmin    = "admin"
	RoleOperator = "operator"
	RoleViewer   = "viewer"
)

// Permission definitions
const (
	PermissionRead   = "read"
	PermissionWrite  = "write"
	PermissionDelete = "delete"
	PermissionReload = "reload"
)

// JWTManager handles JWT token creation and validation
type JWTManager struct {
	secretKey     []byte
	tokenDuration time.Duration
	issuer        string
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(secretKey string, tokenDuration time.Duration, issuer string) (*JWTManager, error) {
	if secretKey == "" {
		// Generate a random secret if none provided
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, fmt.Errorf("failed to generate secret key: %w", err)
		}
		secretKey = base64.StdEncoding.EncodeToString(key)
	}

	return &JWTManager{
		secretKey:     []byte(secretKey),
		tokenDuration: tokenDuration,
		issuer:        issuer,
	}, nil
}

// GenerateToken creates a new JWT token for the given user and role
func (m *JWTManager) GenerateToken(userID string, role string) (string, error) {
	permissions := m.getPermissionsForRole(role)
	
	claims := &JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        generateTokenID(),
		},
		Role:        role,
		Permissions: permissions,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

// ValidateToken validates a JWT token and returns the claims
func (m *JWTManager) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	// Additional validation
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("token has expired")
	}

	if claims.Issuer != m.issuer {
		return nil, errors.New("invalid token issuer")
	}

	return claims, nil
}

// RefreshToken creates a new token with extended expiration
func (m *JWTManager) RefreshToken(oldToken string) (string, error) {
	claims, err := m.ValidateToken(oldToken)
	if err != nil {
		return "", fmt.Errorf("cannot refresh invalid token: %w", err)
	}

	// Check if token is close to expiration (within 1 hour)
	if claims.ExpiresAt.After(time.Now().Add(time.Hour)) {
		return "", errors.New("token is not close to expiration")
	}

	return m.GenerateToken(claims.Subject, claims.Role)
}

// HTTPMiddleware creates an HTTP middleware for JWT authentication
func (m *JWTManager) HTTPMiddleware(requiredPermissions ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			// Check for Bearer prefix
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Authorization header must be Bearer token", http.StatusUnauthorized)
				return
			}

			// Validate token
			claims, err := m.ValidateToken(parts[1])
			if err != nil {
				http.Error(w, fmt.Sprintf("Invalid token: %v", err), http.StatusUnauthorized)
				return
			}

			// Check permissions
			if len(requiredPermissions) > 0 {
				if !m.hasPermissions(claims.Permissions, requiredPermissions) {
					http.Error(w, "Insufficient permissions", http.StatusForbidden)
					return
				}
			}

			// Add claims to request context
			ctx := context.WithValue(r.Context(), "claims", claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetClaimsFromContext extracts JWT claims from request context
func GetClaimsFromContext(ctx context.Context) (*JWTClaims, bool) {
	claims, ok := ctx.Value("claims").(*JWTClaims)
	return claims, ok
}

// getPermissionsForRole returns permissions based on role
func (m *JWTManager) getPermissionsForRole(role string) []string {
	switch role {
	case RoleAdmin:
		return []string{PermissionRead, PermissionWrite, PermissionDelete, PermissionReload}
	case RoleOperator:
		return []string{PermissionRead, PermissionWrite, PermissionReload}
	case RoleViewer:
		return []string{PermissionRead}
	default:
		return []string{}
	}
}

// hasPermissions checks if user has all required permissions
func (m *JWTManager) hasPermissions(userPerms, requiredPerms []string) bool {
	permMap := make(map[string]bool)
	for _, perm := range userPerms {
		permMap[perm] = true
	}

	for _, required := range requiredPerms {
		if !permMap[required] {
			return false
		}
	}
	return true
}

// generateTokenID creates a unique token ID
func generateTokenID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}