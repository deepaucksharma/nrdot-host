package nrtransform

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
)

// Config represents the configuration for the nrtransform processor
type Config struct {
	// Transformations is the list of transformations to apply
	Transformations []TransformationConfig `mapstructure:"transformations"`
}

// TransformationConfig represents a single transformation configuration
type TransformationConfig struct {
	// Type of transformation
	Type TransformationType `mapstructure:"type"`

	// Common fields
	MetricName   string `mapstructure:"metric_name"`
	OutputMetric string `mapstructure:"output_metric"`

	// Aggregation specific
	Aggregation AggregationType `mapstructure:"aggregation"`
	GroupBy     []string        `mapstructure:"group_by"`

	// Unit conversion specific
	FromUnit string `mapstructure:"from_unit"`
	ToUnit   string `mapstructure:"to_unit"`

	// Combine specific
	Expression string   `mapstructure:"expression"`
	Metrics    []string `mapstructure:"metrics"`

	// Filter specific
	Condition string `mapstructure:"condition"`

	// Extract label specific
	LabelKey   string `mapstructure:"label_key"`
	LabelValue string `mapstructure:"label_value"`

	// Histogram specific
	Buckets []float64 `mapstructure:"buckets"`

	// Summary specific
	Percentiles []float64 `mapstructure:"percentiles"`
}

// TransformationType defines the type of transformation
type TransformationType string

const (
	TransformTypeAggregate      TransformationType = "aggregate"
	TransformTypeCalculateRate  TransformationType = "calculate_rate"
	TransformTypeCalculateDelta TransformationType = "calculate_delta"
	TransformTypeConvertUnit    TransformationType = "convert_unit"
	TransformTypeCombine        TransformationType = "combine"
	TransformTypeRename         TransformationType = "rename"
	TransformTypeFilter         TransformationType = "filter"
	TransformTypeExtractLabel   TransformationType = "extract_label"
)

// AggregationType defines the type of aggregation
type AggregationType string

const (
	AggregationSum   AggregationType = "sum"
	AggregationAvg   AggregationType = "avg"
	AggregationMin   AggregationType = "min"
	AggregationMax   AggregationType = "max"
	AggregationCount AggregationType = "count"
)

// Validate checks if the configuration is valid
func (cfg *Config) Validate() error {
	if len(cfg.Transformations) == 0 {
		return fmt.Errorf("at least one transformation must be specified")
	}

	for i, transform := range cfg.Transformations {
		if err := validateTransformation(transform); err != nil {
			return fmt.Errorf("transformation %d: %w", i, err)
		}
	}

	return nil
}

func validateTransformation(t TransformationConfig) error {
	switch t.Type {
	case TransformTypeAggregate:
		if t.MetricName == "" {
			return fmt.Errorf("metric_name is required for aggregate transformation")
		}
		if t.OutputMetric == "" {
			return fmt.Errorf("output_metric is required for aggregate transformation")
		}
		if t.Aggregation == "" {
			return fmt.Errorf("aggregation is required for aggregate transformation")
		}
		if !isValidAggregation(t.Aggregation) {
			return fmt.Errorf("invalid aggregation type: %s", t.Aggregation)
		}

	case TransformTypeCalculateRate, TransformTypeCalculateDelta:
		if t.MetricName == "" {
			return fmt.Errorf("metric_name is required for %s transformation", t.Type)
		}
		if t.OutputMetric == "" {
			return fmt.Errorf("output_metric is required for %s transformation", t.Type)
		}

	case TransformTypeConvertUnit:
		if t.MetricName == "" {
			return fmt.Errorf("metric_name is required for convert_unit transformation")
		}
		if t.OutputMetric == "" {
			return fmt.Errorf("output_metric is required for convert_unit transformation")
		}
		if t.FromUnit == "" || t.ToUnit == "" {
			return fmt.Errorf("from_unit and to_unit are required for convert_unit transformation")
		}

	case TransformTypeCombine:
		if t.Expression == "" {
			return fmt.Errorf("expression is required for combine transformation")
		}
		if t.OutputMetric == "" {
			return fmt.Errorf("output_metric is required for combine transformation")
		}
		if len(t.Metrics) == 0 {
			return fmt.Errorf("metrics list is required for combine transformation")
		}

	case TransformTypeRename:
		if t.MetricName == "" {
			return fmt.Errorf("metric_name is required for rename transformation")
		}
		if t.OutputMetric == "" {
			return fmt.Errorf("output_metric is required for rename transformation")
		}

	case TransformTypeFilter:
		if t.Condition == "" {
			return fmt.Errorf("condition is required for filter transformation")
		}

	case TransformTypeExtractLabel:
		if t.MetricName == "" {
			return fmt.Errorf("metric_name is required for extract_label transformation")
		}
		if t.LabelKey == "" {
			return fmt.Errorf("label_key is required for extract_label transformation")
		}
		if t.OutputMetric == "" {
			return fmt.Errorf("output_metric is required for extract_label transformation")
		}

	default:
		return fmt.Errorf("unsupported transformation type: %s", t.Type)
	}

	return nil
}

func isValidAggregation(agg AggregationType) bool {
	switch agg {
	case AggregationSum, AggregationAvg, AggregationMin, AggregationMax, AggregationCount:
		return true
	default:
		return false
	}
}

var _ component.Config = (*Config)(nil)