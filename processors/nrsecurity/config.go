package nrsecurity

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
)

// Config holds the configuration for the NR Security processor
type Config struct {
	// Enabled controls whether the processor is active
	Enabled bool `mapstructure:"enabled"`

	// ReplacementText is the text used to replace redacted values
	ReplacementText string `mapstructure:"replacement_text"`

	// RedactEmails controls whether email addresses should be redacted
	RedactEmails bool `mapstructure:"redact_emails"`

	// RedactIPs controls whether IP addresses should be redacted
	RedactIPs bool `mapstructure:"redact_ips"`

	// Patterns contains custom regex patterns for redaction
	Patterns []PatternConfig `mapstructure:"patterns"`

	// Keywords contains attribute name keywords that trigger redaction
	Keywords []string `mapstructure:"keywords"`

	// AllowList contains attribute names that should never be redacted
	AllowList []string `mapstructure:"allow_list"`

	// DenyList contains attribute names that should always be redacted
	DenyList []string `mapstructure:"deny_list"`
}

// PatternConfig defines a custom redaction pattern
type PatternConfig struct {
	Name  string `mapstructure:"name"`
	Regex string `mapstructure:"regex"`
}

// Validate checks if the configuration is valid
func (cfg *Config) Validate() error {
	if !cfg.Enabled {
		return nil
	}

	if cfg.ReplacementText == "" {
		return fmt.Errorf("replacement_text cannot be empty")
	}

	// Validate custom patterns
	for i, pattern := range cfg.Patterns {
		if pattern.Name == "" {
			return fmt.Errorf("pattern at index %d has empty name", i)
		}
		if pattern.Regex == "" {
			return fmt.Errorf("pattern '%s' has empty regex", pattern.Name)
		}
	}

	return nil
}

// createDefaultConfig creates the default configuration for the processor
func createDefaultConfig() component.Config {
	return &Config{
		Enabled:         true,
		ReplacementText: "[REDACTED]",
		RedactEmails:    false,
		RedactIPs:       false,
		Patterns:        []PatternConfig{},
		Keywords: []string{
			"password",
			"passwd",
			"secret",
			"token",
			"key",
			"credential",
			"auth",
			"authorization",
			"api_key",
			"apikey",
			"access_token",
			"private_key",
			"client_secret",
		},
		AllowList: []string{
			"service.name",
			"service.version",
			"service.instance.id",
			"span.kind",
			"span.name",
			"http.method",
			"http.status_code",
			"http.scheme",
			"net.peer.name",
			"net.peer.port",
			"process.pid",
			"process.command",
			"process.runtime.name",
			"process.runtime.version",
		},
		DenyList: []string{
			"http.request.header.authorization",
			"http.request.header.x-api-key",
			"http.request.header.x-auth-token",
			"db.connection_string",
			"messaging.url",
		},
	}
}