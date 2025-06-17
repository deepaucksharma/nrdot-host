package nrcap

import (
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
)

// Strategy defines the limiting strategy
type Strategy string

const (
	// StrategyDrop drops metrics that exceed cardinality limit
	StrategyDrop Strategy = "drop"
	// StrategyAggregate removes labels to reduce cardinality
	StrategyAggregate Strategy = "aggregate"
	// StrategySample randomly samples metrics over the limit
	StrategySample Strategy = "sample"
	// StrategyOldest drops oldest label combinations
	StrategyOldest Strategy = "oldest"
)

// Config configures the cardinality protection processor
type Config struct {
	// GlobalLimit is the maximum total cardinality across all metrics
	GlobalLimit int `mapstructure:"global_limit"`

	// MetricLimits defines per-metric cardinality limits
	MetricLimits map[string]int `mapstructure:"metric_limits"`

	// DefaultLimit is the default cardinality limit for unlisted metrics
	DefaultLimit int `mapstructure:"default_limit"`

	// Strategy defines how to handle metrics exceeding limits
	Strategy Strategy `mapstructure:"strategy"`

	// DenyLabels are high-cardinality labels to remove
	DenyLabels []string `mapstructure:"deny_labels"`

	// AllowLabels are labels to always keep
	AllowLabels []string `mapstructure:"allow_labels"`

	// ResetInterval is how often to reset cardinality tracking
	ResetInterval time.Duration `mapstructure:"reset_interval"`

	// EnableStats enables cardinality statistics reporting
	EnableStats bool `mapstructure:"enable_stats"`

	// SampleRate for sample strategy (0.0-1.0)
	SampleRate float64 `mapstructure:"sample_rate"`

	// AggregationLabels to keep when using aggregate strategy
	AggregationLabels []string `mapstructure:"aggregation_labels"`

	// WindowSize for time-based cardinality windows
	WindowSize time.Duration `mapstructure:"window_size"`

	// AlertThreshold percentage (0-100) to trigger alerts
	AlertThreshold int `mapstructure:"alert_threshold"`
}

// createDefaultConfig returns the default config
func createDefaultConfig() component.Config {
	return &Config{
		GlobalLimit:    100000,
		DefaultLimit:   1000,
		Strategy:       StrategyDrop,
		ResetInterval:  1 * time.Hour,
		EnableStats:    true,
		SampleRate:     0.1,
		WindowSize:     5 * time.Minute,
		AlertThreshold: 90,
		MetricLimits:   make(map[string]int),
		DenyLabels:     []string{},
		AllowLabels:    []string{},
		AggregationLabels: []string{
			"service",
			"environment",
			"host",
			"region",
		},
	}
}

// Validate checks if the configuration is valid
func (cfg *Config) Validate() error {
	if cfg.GlobalLimit <= 0 {
		return errors.New("global_limit must be positive")
	}

	if cfg.DefaultLimit <= 0 {
		return errors.New("default_limit must be positive")
	}

	for metric, limit := range cfg.MetricLimits {
		if limit <= 0 {
			return errors.New("metric limit for " + metric + " must be positive")
		}
	}

	switch cfg.Strategy {
	case StrategyDrop, StrategyAggregate, StrategySample, StrategyOldest:
		// valid strategies
	default:
		return errors.New("invalid strategy: " + string(cfg.Strategy))
	}

	if cfg.Strategy == StrategySample {
		if cfg.SampleRate <= 0 || cfg.SampleRate > 1 {
			return errors.New("sample_rate must be between 0 and 1")
		}
	}

	if cfg.ResetInterval <= 0 {
		return errors.New("reset_interval must be positive")
	}

	if cfg.WindowSize <= 0 {
		return errors.New("window_size must be positive")
	}

	if cfg.AlertThreshold < 0 || cfg.AlertThreshold > 100 {
		return errors.New("alert_threshold must be between 0 and 100")
	}

	return nil
}