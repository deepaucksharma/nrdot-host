package schema

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

//go:embed schemas/nrdot-config.json
var nrdotConfigSchema string

// Config represents the validated NRDOT configuration
type Config struct {
	Service    ServiceConfig    `yaml:"service" json:"service"`
	LicenseKey string           `yaml:"license_key,omitempty" json:"license_key,omitempty"`
	AccountID  string           `yaml:"account_id,omitempty" json:"account_id,omitempty"`
	Metrics    MetricsConfig    `yaml:"metrics,omitempty" json:"metrics,omitempty"`
	Traces     TracesConfig     `yaml:"traces,omitempty" json:"traces,omitempty"`
	Logs       LogsConfig       `yaml:"logs,omitempty" json:"logs,omitempty"`
	Security   SecurityConfig   `yaml:"security,omitempty" json:"security,omitempty"`
	Processing ProcessingConfig `yaml:"processing,omitempty" json:"processing,omitempty"`
	Export     ExportConfig     `yaml:"export,omitempty" json:"export,omitempty"`
	Logging    LoggingConfig    `yaml:"logging,omitempty" json:"logging,omitempty"`
}

// ServiceConfig defines service identification
type ServiceConfig struct {
	Name        string            `yaml:"name" json:"name"`
	Environment string            `yaml:"environment,omitempty" json:"environment,omitempty"`
	Version     string            `yaml:"version,omitempty" json:"version,omitempty"`
	Tags        map[string]string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// MetricsConfig defines metrics collection settings
type MetricsConfig struct {
	Enabled  bool     `yaml:"enabled" json:"enabled"`
	Interval string   `yaml:"interval,omitempty" json:"interval,omitempty"`
	Include  []string `yaml:"include,omitempty" json:"include,omitempty"`
	Exclude  []string `yaml:"exclude,omitempty" json:"exclude,omitempty"`
}

// TracesConfig defines trace collection settings
type TracesConfig struct {
	Enabled    bool    `yaml:"enabled" json:"enabled"`
	SampleRate float64 `yaml:"sample_rate,omitempty" json:"sample_rate,omitempty"`
}

// LogsConfig defines log collection settings
type LogsConfig struct {
	Enabled bool        `yaml:"enabled" json:"enabled"`
	Sources []LogSource `yaml:"sources,omitempty" json:"sources,omitempty"`
}

// LogSource defines a log file source
type LogSource struct {
	Path       string            `yaml:"path" json:"path"`
	Parser     string            `yaml:"parser,omitempty" json:"parser,omitempty"`
	Attributes map[string]string `yaml:"attributes,omitempty" json:"attributes,omitempty"`
}

// SecurityConfig defines security settings
type SecurityConfig struct {
	RedactSecrets           bool     `yaml:"redact_secrets" json:"redact_secrets"`
	AllowedAttributes       []string `yaml:"allowed_attributes,omitempty" json:"allowed_attributes,omitempty"`
	BlockedAttributes       []string `yaml:"blocked_attributes,omitempty" json:"blocked_attributes,omitempty"`
	CustomRedactionPatterns []string `yaml:"custom_redaction_patterns,omitempty" json:"custom_redaction_patterns,omitempty"`
}

// ProcessingConfig defines data processing settings
type ProcessingConfig struct {
	CardinalityLimit int              `yaml:"cardinality_limit,omitempty" json:"cardinality_limit,omitempty"`
	Enrichment       EnrichmentConfig `yaml:"enrichment,omitempty" json:"enrichment,omitempty"`
}

// EnrichmentConfig defines enrichment settings
type EnrichmentConfig struct {
	AddHostMetadata       bool `yaml:"add_host_metadata" json:"add_host_metadata"`
	AddCloudMetadata      bool `yaml:"add_cloud_metadata" json:"add_cloud_metadata"`
	AddKubernetesMetadata bool `yaml:"add_kubernetes_metadata" json:"add_kubernetes_metadata"`
}

// ExportConfig defines export settings
type ExportConfig struct {
	Endpoint    string       `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
	Region      string       `yaml:"region,omitempty" json:"region,omitempty"`
	Compression string       `yaml:"compression,omitempty" json:"compression,omitempty"`
	Timeout     string       `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Retry       RetryConfig  `yaml:"retry,omitempty" json:"retry,omitempty"`
}

// RetryConfig defines retry settings
type RetryConfig struct {
	Enabled     bool   `yaml:"enabled" json:"enabled"`
	MaxAttempts int    `yaml:"max_attempts,omitempty" json:"max_attempts,omitempty"`
	Backoff     string `yaml:"backoff,omitempty" json:"backoff,omitempty"`
}

// LoggingConfig defines logging settings
type LoggingConfig struct {
	Level  string `yaml:"level,omitempty" json:"level,omitempty"`
	Format string `yaml:"format,omitempty" json:"format,omitempty"`
}

// Validator provides configuration validation
type Validator struct {
	schema *gojsonschema.Schema
}

// NewValidator creates a new configuration validator
func NewValidator() (*Validator, error) {
	schemaLoader := gojsonschema.NewStringLoader(nrdotConfigSchema)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema: %w", err)
	}

	return &Validator{
		schema: schema,
	}, nil
}

// ValidateYAML validates a YAML configuration
func (v *Validator) ValidateYAML(yamlData []byte) (*Config, error) {
	// First, parse YAML to get a generic structure
	var rawConfig interface{}
	decoder := yaml.NewDecoder(bytes.NewReader(yamlData))
	decoder.KnownFields(true)
	if err := decoder.Decode(&rawConfig); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}

	// Convert to JSON for schema validation
	jsonData, err := json.Marshal(rawConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to JSON: %w", err)
	}

	// Validate against schema
	documentLoader := gojsonschema.NewBytesLoader(jsonData)
	result, err := v.schema.Validate(documentLoader)
	if err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	if !result.Valid() {
		return nil, v.formatValidationErrors(result.Errors())
	}

	// Parse into struct with defaults
	config := &Config{}
	if err := yaml.Unmarshal(yamlData, config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Apply defaults
	v.applyDefaults(config)

	return config, nil
}

// ValidateJSON validates a JSON configuration
func (v *Validator) ValidateJSON(jsonData []byte) (*Config, error) {
	documentLoader := gojsonschema.NewBytesLoader(jsonData)
	result, err := v.schema.Validate(documentLoader)
	if err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	if !result.Valid() {
		return nil, v.formatValidationErrors(result.Errors())
	}

	config := &Config{}
	if err := json.Unmarshal(jsonData, config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	v.applyDefaults(config)

	return config, nil
}

// formatValidationErrors formats schema validation errors
func (v *Validator) formatValidationErrors(errors []gojsonschema.ResultError) error {
	var messages []string
	for _, err := range errors {
		field := err.Field()
		if field == "(root)" {
			field = "configuration"
		}
		messages = append(messages, fmt.Sprintf("- %s: %s", field, err.Description()))
	}
	return fmt.Errorf("configuration validation failed:\n%s", strings.Join(messages, "\n"))
}

// applyDefaults applies default values to the configuration
func (v *Validator) applyDefaults(config *Config) {
	// Service defaults
	if config.Service.Environment == "" {
		config.Service.Environment = "production"
	}

	// Metrics defaults
	if config.Metrics.Interval == "" {
		config.Metrics.Interval = "60s"
	}

	// Traces defaults
	if config.Traces.SampleRate == 0 {
		config.Traces.SampleRate = 0.1
	}

	// Security defaults
	config.Security.RedactSecrets = true

	// Processing defaults
	if config.Processing.CardinalityLimit == 0 {
		config.Processing.CardinalityLimit = 10000
	}
	config.Processing.Enrichment.AddHostMetadata = true
	config.Processing.Enrichment.AddCloudMetadata = true
	config.Processing.Enrichment.AddKubernetesMetadata = true

	// Export defaults
	if config.Export.Endpoint == "" {
		config.Export.Endpoint = "https://otlp.nr-data.net"
	}
	if config.Export.Region == "" {
		config.Export.Region = "US"
	}
	if config.Export.Compression == "" {
		config.Export.Compression = "gzip"
	}
	if config.Export.Timeout == "" {
		config.Export.Timeout = "30s"
	}
	config.Export.Retry.Enabled = true
	if config.Export.Retry.MaxAttempts == 0 {
		config.Export.Retry.MaxAttempts = 3
	}
	if config.Export.Retry.Backoff == "" {
		config.Export.Retry.Backoff = "5s"
	}

	// Logging defaults
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}
	if config.Logging.Format == "" {
		config.Logging.Format = "text"
	}
}

// GetSchema returns the raw JSON schema
func GetSchema() string {
	return nrdotConfigSchema
}