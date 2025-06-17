package models

import (
	"time"
)

// Config represents the complete NRDOT configuration
type Config struct {
	Version      int                    `json:"version"`
	Service      ServiceConfig          `json:"service"`
	LicenseKey   string                 `json:"license_key,omitempty"`
	Metrics      MetricsConfig          `json:"metrics"`
	Traces       TracesConfig           `json:"traces"`
	Logs         LogsConfig             `json:"logs"`
	Security     SecurityConfig         `json:"security"`
	Processing   ProcessingConfig       `json:"processing"`
	Export       ExportConfig           `json:"export"`
	Advanced     map[string]interface{} `json:"advanced,omitempty"`
}

// ServiceConfig contains service identification
type ServiceConfig struct {
	Name        string            `json:"name"`
	Environment string            `json:"environment"`
	Version     string            `json:"version,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
}

// MetricsConfig contains metrics collection settings
type MetricsConfig struct {
	Enabled      bool              `json:"enabled"`
	Interval     time.Duration     `json:"interval"`
	HostMetrics  bool              `json:"host_metrics"`
	ProcessMetrics bool            `json:"process_metrics"`
	CustomMetrics map[string]interface{} `json:"custom_metrics,omitempty"`
}

// TracesConfig contains tracing settings
type TracesConfig struct {
	Enabled     bool    `json:"enabled"`
	SampleRate  float64 `json:"sample_rate"`
	Propagators []string `json:"propagators,omitempty"`
}

// LogsConfig contains log collection settings
type LogsConfig struct {
	Enabled      bool     `json:"enabled"`
	Paths        []string `json:"paths,omitempty"`
	IncludeStdout bool    `json:"include_stdout"`
	IncludeStderr bool    `json:"include_stderr"`
}

// SecurityConfig contains security settings
type SecurityConfig struct {
	RedactSecrets   bool     `json:"redact_secrets"`
	RedactPatterns  []string `json:"redact_patterns,omitempty"`
	AllowedDomains  []string `json:"allowed_domains,omitempty"`
	TLSConfig       *TLSConfig `json:"tls,omitempty"`
}

// TLSConfig contains TLS settings
type TLSConfig struct {
	Enabled    bool   `json:"enabled"`
	CertFile   string `json:"cert_file,omitempty"`
	KeyFile    string `json:"key_file,omitempty"`
	CAFile     string `json:"ca_file,omitempty"`
	SkipVerify bool   `json:"skip_verify"`
}

// ProcessingConfig contains data processing settings
type ProcessingConfig struct {
	Enrich      EnrichConfig      `json:"enrich"`
	Transform   TransformConfig   `json:"transform"`
	Cardinality CardinalityConfig `json:"cardinality"`
}

// EnrichConfig contains enrichment settings
type EnrichConfig struct {
	AddHostMetadata  bool              `json:"add_host_metadata"`
	AddCloudMetadata bool              `json:"add_cloud_metadata"`
	AddK8sMetadata   bool              `json:"add_k8s_metadata"`
	CustomTags       map[string]string `json:"custom_tags,omitempty"`
}

// TransformConfig contains transformation settings
type TransformConfig struct {
	ConvertUnits    bool                   `json:"convert_units"`
	Aggregations    []AggregationConfig    `json:"aggregations,omitempty"`
	Calculations    []CalculationConfig    `json:"calculations,omitempty"`
}

// CardinalityConfig contains cardinality control settings
type CardinalityConfig struct {
	Enabled      bool                       `json:"enabled"`
	GlobalLimit  int                        `json:"global_limit"`
	PerMetric    map[string]int             `json:"per_metric_limits,omitempty"`
	LimitAction  string                     `json:"limit_action"` // drop, sample, aggregate
}

// ExportConfig contains export destination settings
type ExportConfig struct {
	Endpoint     string            `json:"endpoint"`
	Headers      map[string]string `json:"headers,omitempty"`
	Timeout      time.Duration     `json:"timeout"`
	RetryConfig  RetryConfig       `json:"retry"`
	Compression  string            `json:"compression,omitempty"`
}

// RetryConfig contains retry settings
type RetryConfig struct {
	Enabled         bool          `json:"enabled"`
	InitialInterval time.Duration `json:"initial_interval"`
	MaxInterval     time.Duration `json:"max_interval"`
	MaxElapsedTime  time.Duration `json:"max_elapsed_time"`
}

// ConfigUpdate represents a configuration change request
type ConfigUpdate struct {
	Config      []byte            `json:"config"`
	Format      string            `json:"format"` // yaml, json
	DryRun      bool              `json:"dry_run"`
	Source      string            `json:"source"` // cli, api, file, remote
	Author      string            `json:"author,omitempty"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ConfigResult represents the result of a configuration operation
type ConfigResult struct {
	Success         bool               `json:"success"`
	Version         int                `json:"version"`
	ValidationResult *ValidationResult `json:"validation,omitempty"`
	AppliedAt       time.Time          `json:"applied_at,omitempty"`
	Error           *ErrorInfo         `json:"error,omitempty"`
	Warnings        []string           `json:"warnings,omitempty"`
}

// ValidationResult contains configuration validation details
type ValidationResult struct {
	Valid    bool              `json:"valid"`
	Errors   []ValidationError `json:"errors,omitempty"`
	Warnings []string          `json:"warnings,omitempty"`
	Info     []string          `json:"info,omitempty"`
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// ConfigVersion represents a configuration version entry
type ConfigVersion struct {
	Version     int               `json:"version"`
	AppliedAt   time.Time         `json:"applied_at"`
	Source      string            `json:"source"`
	Author      string            `json:"author,omitempty"`
	Description string            `json:"description,omitempty"`
	Hash        string            `json:"hash"`
	Size        int64             `json:"size"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// GeneratedConfig represents the output of config generation
type GeneratedConfig struct {
	OTelConfig   string            `json:"otel_config"`
	Hash         string            `json:"hash"`
	GeneratedAt  time.Time         `json:"generated_at"`
	Templates    []string          `json:"templates_used"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// ConfigDiff represents differences between configurations
type ConfigDiff struct {
	OldVersion int      `json:"old_version"`
	NewVersion int      `json:"new_version"`
	Added      []string `json:"added"`
	Removed    []string `json:"removed"`
	Modified   []string `json:"modified"`
	Summary    string   `json:"summary"`
}

// AggregationConfig defines metric aggregation rules
type AggregationConfig struct {
	Name       string   `json:"name"`
	Metrics    []string `json:"metrics"`
	Method     string   `json:"method"` // sum, avg, min, max
	Dimensions []string `json:"dimensions"`
	Window     time.Duration `json:"window"`
}

// CalculationConfig defines metric calculation rules
type CalculationConfig struct {
	Name       string `json:"name"`
	Expression string `json:"expression"`
	Unit       string `json:"unit,omitempty"`
}