package output

import (
	"encoding/json"
	"fmt"

	"github.com/newrelic/nrdot-host/nrdot-ctl/pkg/client"
	"gopkg.in/yaml.v3"
)

// Formatter handles output formatting
type Formatter struct {
	format string
}

// NewFormatter creates a new formatter
func NewFormatter(format string) *Formatter {
	return &Formatter{format: format}
}

// FormatStatus formats status output
func (f *Formatter) FormatStatus(status *client.Status) error {
	switch f.format {
	case "json":
		return f.formatJSON(status)
	case "yaml":
		return f.formatYAML(status)
	default:
		return formatStatusTable(status)
	}
}

// FormatValidationResult formats validation result output
func (f *Formatter) FormatValidationResult(result *client.ValidationResult) error {
	switch f.format {
	case "json":
		return f.formatJSON(result)
	case "yaml":
		return f.formatYAML(result)
	default:
		return formatValidationTable(result)
	}
}

// FormatOperationResult formats operation result output
func (f *Formatter) FormatOperationResult(result *client.OperationResult) error {
	switch f.format {
	case "json":
		return f.formatJSON(result)
	case "yaml":
		return f.formatYAML(result)
	default:
		return formatOperationMessage(result)
	}
}

// FormatApplyResult formats apply result output
func (f *Formatter) FormatApplyResult(result *client.ApplyResult) error {
	switch f.format {
	case "json":
		return f.formatJSON(result)
	case "yaml":
		return f.formatYAML(result)
	default:
		return formatApplyMessage(result)
	}
}

// FormatMetrics formats metrics output
func (f *Formatter) FormatMetrics(metrics *client.Metrics) error {
	switch f.format {
	case "json":
		return f.formatJSON(metrics)
	case "yaml":
		return f.formatYAML(metrics)
	default:
		return formatMetricsTable(metrics)
	}
}

// FormatVersion formats version output
func (f *Formatter) FormatVersion(info *VersionInfo) error {
	switch f.format {
	case "json":
		return f.formatJSON(info)
	case "yaml":
		return f.formatYAML(info)
	default:
		return formatVersionTable(info)
	}
}

// Helper methods

func (f *Formatter) formatJSON(v interface{}) error {
	encoder := json.NewEncoder(outputWriter)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

func (f *Formatter) formatYAML(v interface{}) error {
	encoder := yaml.NewEncoder(outputWriter)
	defer encoder.Close()
	return encoder.Encode(v)
}

// VersionInfo represents version information
type VersionInfo struct {
	Version   string `json:"version" yaml:"version"`
	BuildTime string `json:"build_time" yaml:"build_time"`
	GoVersion string `json:"go_version" yaml:"go_version"`
	OS        string `json:"os" yaml:"os"`
	Arch      string `json:"arch" yaml:"arch"`
}

func formatOperationMessage(result *client.OperationResult) error {
	if result.Success {
		fmt.Fprintln(outputWriter, successColor(result.Message))
	} else {
		fmt.Fprintln(outputWriter, errorColor("Error: " + result.Error))
	}
	return nil
}

func formatApplyMessage(result *client.ApplyResult) error {
	if result.Success {
		fmt.Fprintln(outputWriter, successColor(result.Message))
		fmt.Fprintf(outputWriter, "Previous version: %s\n", result.PreviousVersion)
		fmt.Fprintf(outputWriter, "New version: %s\n", result.NewVersion)
	} else {
		fmt.Fprintln(outputWriter, errorColor("Error: " + result.Error))
	}
	return nil
}