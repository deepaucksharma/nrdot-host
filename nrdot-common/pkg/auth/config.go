package auth

import (
	"fmt"
	"time"
)

// Config represents authentication configuration
type Config struct {
	// Enabled determines if authentication is required
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Type specifies the authentication type (jwt, api-key, both)
	Type string `json:"type" yaml:"type"`

	// JWT configuration
	JWT JWTConfig `json:"jwt" yaml:"jwt"`

	// API Key configuration
	APIKey APIKeyConfig `json:"api_key" yaml:"api_key"`

	// Default admin credentials (for initial setup)
	DefaultAdmin DefaultAdminConfig `json:"default_admin" yaml:"default_admin"`
}

// JWTConfig represents JWT-specific configuration
type JWTConfig struct {
	// Secret key for signing tokens (will be generated if empty)
	SecretKey string `json:"secret_key" yaml:"secret_key"`

	// Token duration
	Duration time.Duration `json:"duration" yaml:"duration"`

	// Issuer name
	Issuer string `json:"issuer" yaml:"issuer"`

	// Refresh token settings
	RefreshEnabled  bool          `json:"refresh_enabled" yaml:"refresh_enabled"`
	RefreshDuration time.Duration `json:"refresh_duration" yaml:"refresh_duration"`
}

// APIKeyConfig represents API key configuration
type APIKeyConfig struct {
	// Header name for API key (default: X-API-Key)
	HeaderName string `json:"header_name" yaml:"header_name"`

	// Allow API keys in query parameter
	AllowQueryParam bool `json:"allow_query_param" yaml:"allow_query_param"`

	// Query parameter name (default: api_key)
	QueryParamName string `json:"query_param_name" yaml:"query_param_name"`

	// Default expiration for API keys
	DefaultExpiration time.Duration `json:"default_expiration" yaml:"default_expiration"`
}

// DefaultAdminConfig represents default admin user configuration
type DefaultAdminConfig struct {
	// Username for default admin
	Username string `json:"username" yaml:"username"`

	// Password for default admin (should be changed on first login)
	Password string `json:"password" yaml:"password"`

	// API key for default admin
	APIKey string `json:"api_key" yaml:"api_key"`

	// Force password change on first login
	ForcePasswordChange bool `json:"force_password_change" yaml:"force_password_change"`
}

// AuthType constants
const (
	AuthTypeJWT    = "jwt"
	AuthTypeAPIKey = "api-key"
	AuthTypeBoth   = "both"
	AuthTypeNone   = "none"
)

// DefaultAuthConfig returns a default authentication configuration
func DefaultAuthConfig() Config {
	return Config{
		Enabled: false,
		Type:    AuthTypeNone,
		JWT: JWTConfig{
			Duration:        24 * time.Hour,
			Issuer:          "nrdot-host",
			RefreshEnabled:  true,
			RefreshDuration: 7 * 24 * time.Hour,
		},
		APIKey: APIKeyConfig{
			HeaderName:        "X-API-Key",
			AllowQueryParam:   false,
			QueryParamName:    "api_key",
			DefaultExpiration: 90 * 24 * time.Hour, // 90 days
		},
		DefaultAdmin: DefaultAdminConfig{
			Username:            "admin",
			Password:            "changeme",
			ForcePasswordChange: true,
		},
	}
}

// Validate validates the authentication configuration
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	switch c.Type {
	case AuthTypeJWT, AuthTypeAPIKey, AuthTypeBoth:
		// Valid types
	default:
		return fmt.Errorf("invalid auth type: %s", c.Type)
	}

	if c.Type == AuthTypeJWT || c.Type == AuthTypeBoth {
		if c.JWT.Duration <= 0 {
			return fmt.Errorf("JWT duration must be positive")
		}
		if c.JWT.Issuer == "" {
			return fmt.Errorf("JWT issuer cannot be empty")
		}
	}

	if c.Type == AuthTypeAPIKey || c.Type == AuthTypeBoth {
		if c.APIKey.HeaderName == "" {
			return fmt.Errorf("API key header name cannot be empty")
		}
		if c.APIKey.AllowQueryParam && c.APIKey.QueryParamName == "" {
			return fmt.Errorf("API key query parameter name cannot be empty when query params are allowed")
		}
	}

	return nil
}