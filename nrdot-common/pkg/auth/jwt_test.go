package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJWTManager(t *testing.T) {
	tests := []struct {
		name          string
		secretKey     string
		tokenDuration time.Duration
		issuer        string
		wantErr       bool
	}{
		{
			name:          "valid configuration",
			secretKey:     "test-secret-key-123",
			tokenDuration: time.Hour,
			issuer:        "nrdot-test",
			wantErr:       false,
		},
		{
			name:          "empty secret key generates random",
			secretKey:     "",
			tokenDuration: time.Hour,
			issuer:        "nrdot-test",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewJWTManager(tt.secretKey, tt.tokenDuration, tt.issuer)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, manager)
			}
		})
	}
}

func TestGenerateAndValidateToken(t *testing.T) {
	manager, err := NewJWTManager("test-secret", time.Hour, "nrdot-test")
	require.NoError(t, err)

	tests := []struct {
		name   string
		userID string
		role   string
	}{
		{
			name:   "admin token",
			userID: "admin-user",
			role:   RoleAdmin,
		},
		{
			name:   "operator token",
			userID: "operator-user",
			role:   RoleOperator,
		},
		{
			name:   "viewer token",
			userID: "viewer-user",
			role:   RoleViewer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate token
			token, err := manager.GenerateToken(tt.userID, tt.role)
			assert.NoError(t, err)
			assert.NotEmpty(t, token)

			// Validate token
			claims, err := manager.ValidateToken(token)
			assert.NoError(t, err)
			assert.NotNil(t, claims)
			assert.Equal(t, tt.userID, claims.Subject)
			assert.Equal(t, tt.role, claims.Role)
			assert.Equal(t, "nrdot-test", claims.Issuer)
			assert.NotEmpty(t, claims.Permissions)

			// Check permissions based on role
			switch tt.role {
			case RoleAdmin:
				assert.Contains(t, claims.Permissions, PermissionRead)
				assert.Contains(t, claims.Permissions, PermissionWrite)
				assert.Contains(t, claims.Permissions, PermissionDelete)
				assert.Contains(t, claims.Permissions, PermissionReload)
			case RoleOperator:
				assert.Contains(t, claims.Permissions, PermissionRead)
				assert.Contains(t, claims.Permissions, PermissionWrite)
				assert.Contains(t, claims.Permissions, PermissionReload)
				assert.NotContains(t, claims.Permissions, PermissionDelete)
			case RoleViewer:
				assert.Contains(t, claims.Permissions, PermissionRead)
				assert.NotContains(t, claims.Permissions, PermissionWrite)
			}
		})
	}
}

func TestValidateToken_Invalid(t *testing.T) {
	manager, err := NewJWTManager("test-secret", time.Hour, "nrdot-test")
	require.NoError(t, err)

	tests := []struct {
		name        string
		token       string
		expectedErr string
	}{
		{
			name:        "empty token",
			token:       "",
			expectedErr: "token is malformed",
		},
		{
			name:        "invalid format",
			token:       "not.a.token",
			expectedErr: "token is malformed",
		},
		{
			name:        "wrong signature",
			token:       generateTokenWithWrongSignature(t),
			expectedErr: "signature is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := manager.ValidateToken(tt.token)
			assert.Error(t, err)
			assert.Nil(t, claims)
			assert.Contains(t, err.Error(), "invalid token")
		})
	}
}

func TestValidateToken_Expired(t *testing.T) {
	// Create manager with very short duration
	manager, err := NewJWTManager("test-secret", 1*time.Millisecond, "nrdot-test")
	require.NoError(t, err)

	// Generate token
	token, err := manager.GenerateToken("test-user", RoleViewer)
	require.NoError(t, err)

	// Wait for token to expire
	time.Sleep(2 * time.Millisecond)

	// Try to validate expired token
	claims, err := manager.ValidateToken(token)
	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Contains(t, err.Error(), "token has expired")
}

func TestRefreshToken(t *testing.T) {
	manager, err := NewJWTManager("test-secret", 2*time.Hour, "nrdot-test")
	require.NoError(t, err)

	// Generate initial token
	token, err := manager.GenerateToken("test-user", RoleOperator)
	require.NoError(t, err)

	// Try to refresh too early (should fail)
	_, err = manager.RefreshToken(token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not close to expiration")

	// Create a token that's close to expiration
	shortManager, err := NewJWTManager("test-secret", 30*time.Minute, "nrdot-test")
	require.NoError(t, err)
	
	shortToken, err := shortManager.GenerateToken("test-user", RoleOperator)
	require.NoError(t, err)

	// Now refresh should work (using original manager)
	manager.tokenDuration = 30 * time.Minute
	newToken, err := manager.RefreshToken(shortToken)
	assert.NoError(t, err)
	assert.NotEmpty(t, newToken)
	assert.NotEqual(t, shortToken, newToken)

	// Validate new token
	claims, err := manager.ValidateToken(newToken)
	assert.NoError(t, err)
	assert.Equal(t, "test-user", claims.Subject)
	assert.Equal(t, RoleOperator, claims.Role)
}

func TestHTTPMiddleware(t *testing.T) {
	manager, err := NewJWTManager("test-secret", time.Hour, "nrdot-test")
	require.NoError(t, err)

	// Generate tokens for different roles
	adminToken, _ := manager.GenerateToken("admin", RoleAdmin)
	operatorToken, _ := manager.GenerateToken("operator", RoleOperator)
	viewerToken, _ := manager.GenerateToken("viewer", RoleViewer)

	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := GetClaimsFromContext(r.Context())
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(claims.Role))
	})

	tests := []struct {
		name               string
		token              string
		requiredPerms      []string
		expectedStatus     int
		expectedResponse   string
	}{
		{
			name:               "no auth header",
			token:              "",
			requiredPerms:      []string{},
			expectedStatus:     http.StatusUnauthorized,
		},
		{
			name:               "invalid token format",
			token:              "invalid-token",
			requiredPerms:      []string{},
			expectedStatus:     http.StatusUnauthorized,
		},
		{
			name:               "admin with no required perms",
			token:              adminToken,
			requiredPerms:      []string{},
			expectedStatus:     http.StatusOK,
			expectedResponse:   RoleAdmin,
		},
		{
			name:               "admin with write permission",
			token:              adminToken,
			requiredPerms:      []string{PermissionWrite},
			expectedStatus:     http.StatusOK,
			expectedResponse:   RoleAdmin,
		},
		{
			name:               "viewer with write permission",
			token:              viewerToken,
			requiredPerms:      []string{PermissionWrite},
			expectedStatus:     http.StatusForbidden,
		},
		{
			name:               "operator with reload permission",
			token:              operatorToken,
			requiredPerms:      []string{PermissionReload},
			expectedStatus:     http.StatusOK,
			expectedResponse:   RoleOperator,
		},
		{
			name:               "operator with delete permission",
			token:              operatorToken,
			requiredPerms:      []string{PermissionDelete},
			expectedStatus:     http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create middleware
			middleware := manager.HTTPMiddleware(tt.requiredPerms...)
			handler := middleware(testHandler)

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.token != "" {
				if tt.token == "invalid-token" {
					req.Header.Set("Authorization", tt.token)
				} else {
					req.Header.Set("Authorization", "Bearer "+tt.token)
				}
			}

			// Execute request
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			// Check response
			assert.Equal(t, tt.expectedStatus, rec.Code)
			if tt.expectedResponse != "" {
				assert.Equal(t, tt.expectedResponse, rec.Body.String())
			}
		})
	}
}

func TestGetClaimsFromContext(t *testing.T) {
	// Test with claims in context
	claims := &JWTClaims{
		Role: RoleAdmin,
	}
	ctx := context.WithValue(context.Background(), "claims", claims)
	
	gotClaims, ok := GetClaimsFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, claims, gotClaims)

	// Test without claims in context
	emptyCtx := context.Background()
	gotClaims, ok = GetClaimsFromContext(emptyCtx)
	assert.False(t, ok)
	assert.Nil(t, gotClaims)
}

// Helper function to generate a token with wrong signature
func generateTokenWithWrongSignature(t *testing.T) string {
	manager1, _ := NewJWTManager("secret1", time.Hour, "nrdot-test")
	manager2, _ := NewJWTManager("secret2", time.Hour, "nrdot-test")
	
	// Generate token with first manager
	token, err := manager1.GenerateToken("user", RoleViewer)
	require.NoError(t, err)
	
	// Try to validate with second manager (different secret)
	// This will fail, but we just want the token string
	return token
}