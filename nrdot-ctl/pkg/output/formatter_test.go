package output

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/newrelic/nrdot-host/nrdot-ctl/pkg/client"
	"gopkg.in/yaml.v3"
)

func TestFormatStatus(t *testing.T) {
	status := &client.Status{
		State:         "running",
		Uptime:        2 * time.Hour,
		ConfigVersion: "v1.2.3",
		Health: client.Health{
			Status: "healthy",
			Checks: map[string]client.Check{
				"collector": {Status: "healthy"},
				"api":       {Status: "healthy"},
			},
			LastUpdate: time.Now(),
		},
		CollectorVersion: "0.88.0",
	}

	tests := []struct {
		name   string
		format string
		check  func(t *testing.T, output string)
	}{
		{
			name:   "table format",
			format: "table",
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, "running") {
					t.Error("Expected output to contain 'running'")
				}
				if !strings.Contains(output, "2h") {
					t.Error("Expected output to contain uptime")
				}
			},
		},
		{
			name:   "json format",
			format: "json",
			check: func(t *testing.T, output string) {
				var result client.Status
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("Failed to unmarshal JSON: %v", err)
				}
				if result.State != "running" {
					t.Errorf("Expected state 'running', got %s", result.State)
				}
			},
		},
		{
			name:   "yaml format",
			format: "yaml",
			check: func(t *testing.T, output string) {
				var result client.Status
				if err := yaml.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("Failed to unmarshal YAML: %v", err)
				}
				if result.State != "running" {
					t.Errorf("Expected state 'running', got %s", result.State)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			SetOutput(buf)
			defer SetOutput(os.Stdout)

			formatter := NewFormatter(tt.format)
			err := formatter.FormatStatus(status)
			if err != nil {
				t.Errorf("FormatStatus() error = %v", err)
				return
			}

			tt.check(t, buf.String())
		})
	}
}

func TestFormatValidationResult(t *testing.T) {
	result := &client.ValidationResult{
		Valid: false,
		Errors: []client.ValidationError{
			{Field: "license_key", Message: "license key is required"},
			{Field: "collector.receivers", Message: "at least one receiver must be enabled"},
		},
		Warnings: []string{
			"debug mode is enabled",
			"resource limits are not configured",
		},
	}

	buf := new(bytes.Buffer)
	SetOutput(buf)
	defer SetOutput(os.Stdout)

	formatter := NewFormatter("table")
	err := formatter.FormatValidationResult(result)
	if err != nil {
		t.Errorf("FormatValidationResult() error = %v", err)
		return
	}

	output := buf.String()
	if !strings.Contains(output, "invalid") {
		t.Error("Expected output to indicate invalid configuration")
	}
	if !strings.Contains(output, "license_key") {
		t.Error("Expected output to contain error field")
	}
	if !strings.Contains(output, "debug mode") {
		t.Error("Expected output to contain warning")
	}
}

func TestFormatMetrics(t *testing.T) {
	metrics := &client.Metrics{
		ReceivedMetrics: 10000,
		SentMetrics:     9500,
		DroppedMetrics:  500,
		ProcessingRate:  100.5,
		ErrorRate:       5.0,
		ResourceUsage: client.ResourceUsage{
			CPUPercent:     25.5,
			MemoryMB:       128.0,
			GoroutineCount: 50,
		},
		PipelineMetrics: map[string]client.PipelineMetric{
			"metrics/hostmetrics": {
				Received: 5000,
				Sent:     4800,
				Dropped:  200,
				Errors:   10,
			},
			"metrics/prometheus": {
				Received: 5000,
				Sent:     4700,
				Dropped:  300,
				Errors:   20,
			},
		},
	}

	buf := new(bytes.Buffer)
	SetOutput(buf)
	defer SetOutput(os.Stdout)

	formatter := NewFormatter("table")
	err := formatter.FormatMetrics(metrics)
	if err != nil {
		t.Errorf("FormatMetrics() error = %v", err)
		return
	}

	output := buf.String()
	if !strings.Contains(output, "10000") {
		t.Error("Expected output to contain received metrics count")
	}
	if !strings.Contains(output, "25.5%") {
		t.Error("Expected output to contain CPU percentage")
	}
	if !strings.Contains(output, "hostmetrics") {
		t.Error("Expected output to contain pipeline names")
	}
}