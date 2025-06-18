package nrenrich

import (
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
)

// Config holds the configuration for the nrenrich processor
type Config struct {
	// StaticAttributes are attributes added to all telemetry data
	StaticAttributes map[string]interface{} `mapstructure:"static_attributes"`

	// Environment configuration for collecting environment metadata
	Environment EnvironmentConfig `mapstructure:"environment"`

	// Process configuration for collecting process metadata
	Process ProcessConfig `mapstructure:"process"`

	// Rules for conditional enrichment
	Rules []EnrichmentRule `mapstructure:"rules"`

	// Dynamic attribute computation
	Dynamic []DynamicAttribute `mapstructure:"dynamic"`

	// Cache configuration
	Cache CacheConfig `mapstructure:"cache"`
}

// EnvironmentConfig configures environment metadata collection
type EnvironmentConfig struct {
	// Enabled determines if environment metadata collection is enabled
	Enabled bool `mapstructure:"enabled"`

	// Hostname enables hostname collection
	Hostname bool `mapstructure:"hostname"`

	// CloudProvider enables cloud provider metadata collection
	CloudProvider bool `mapstructure:"cloud_provider"`

	// Kubernetes enables Kubernetes metadata collection
	Kubernetes bool `mapstructure:"kubernetes"`

	// System enables system information collection
	System bool `mapstructure:"system"`
}

// ProcessConfig configures process metadata collection
type ProcessConfig struct {
	// Enabled determines if process metadata collection is enabled
	Enabled bool `mapstructure:"enabled"`

	// HelperEndpoint is the endpoint for the privileged helper
	HelperEndpoint string `mapstructure:"helper_endpoint"`

	// Timeout for helper requests
	Timeout time.Duration `mapstructure:"timeout"`
}

// EnrichmentRule defines a conditional enrichment rule
type EnrichmentRule struct {
	// Condition is a CEL expression that determines if the rule applies
	Condition string `mapstructure:"condition"`

	// Attributes to add when the condition matches
	Attributes map[string]interface{} `mapstructure:"attributes"`

	// Priority of the rule (higher priority rules are evaluated first)
	Priority int `mapstructure:"priority"`
}

// DynamicAttribute defines dynamic attribute computation
type DynamicAttribute struct {
	// Target is the attribute name to set
	Target string `mapstructure:"target"`

	// Source is the attribute to read from
	Source string `mapstructure:"source"`

	// Transform is a script/expression to transform the source value
	Transform string `mapstructure:"transform"`
}

// CacheConfig configures metadata caching
type CacheConfig struct {
	// TTL is the time-to-live for cached metadata
	TTL time.Duration `mapstructure:"ttl"`

	// MaxSize is the maximum number of entries in the cache
	MaxSize int `mapstructure:"max_size"`
}

// Validate checks if the configuration is valid
func (cfg *Config) Validate() error {
	if cfg.Process.Enabled && cfg.Process.HelperEndpoint == "" {
		return errors.New("helper_endpoint must be specified when process enrichment is enabled")
	}

	for i, rule := range cfg.Rules {
		if rule.Condition == "" {
			return errors.New("enrichment rule must have a condition")
		}
		if len(rule.Attributes) == 0 {
			return errors.New("enrichment rule must specify attributes to add")
		}
		cfg.Rules[i] = rule
	}

	for _, dynamic := range cfg.Dynamic {
		if dynamic.Target == "" {
			return errors.New("dynamic attribute must have a target")
		}
		if dynamic.Source == "" && dynamic.Transform == "" {
			return errors.New("dynamic attribute must have either source or transform")
		}
	}

	if cfg.Cache.TTL == 0 {
		cfg.Cache.TTL = 5 * time.Minute
	}
	if cfg.Cache.MaxSize == 0 {
		cfg.Cache.MaxSize = 1000
	}

	return nil
}

var _ component.Config = (*Config)(nil)