package templatelib

import (
	"testing"

	"github.com/newrelic/nrdot-host/nrdot-schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGenerator(t *testing.T) {
	t.Run("minimal config", func(t *testing.T) {
		config := &schema.Config{
			Service: schema.ServiceConfig{
				Name:        "test-service",
				Environment: "production",
			},
			Metrics: schema.MetricsConfig{
				Enabled:  true,
				Interval: "60s",
			},
			Security: schema.SecurityConfig{
				RedactSecrets: true,
			},
			Processing: schema.ProcessingConfig{
				CardinalityLimit: 10000,
				Enrichment: schema.EnrichmentConfig{
					AddHostMetadata: true,
				},
			},
			Export: schema.ExportConfig{
				Endpoint:    "https://otlp.nr-data.net",
				Region:      "US",
				Compression: "gzip",
				Timeout:     "30s",
				Retry: schema.RetryConfig{
					Enabled:     true,
					MaxAttempts: 3,
					Backoff:     "5s",
				},
			},
			Logging: schema.LoggingConfig{
				Level:  "info",
				Format: "text",
			},
		}

		gen := NewGenerator(config)
		otelConfig, err := gen.Generate()
		require.NoError(t, err)

		// Verify extensions
		assert.Contains(t, otelConfig.Extensions, "health_check")
		assert.NotContains(t, otelConfig.Extensions, "pprof") // not in debug mode

		// Verify receivers
		assert.Contains(t, otelConfig.Receivers, "hostmetrics")
		assert.Contains(t, otelConfig.Receivers, "prometheus")
		assert.NotContains(t, otelConfig.Receivers, "otlp") // traces not enabled

		// Verify processors
		assert.Contains(t, otelConfig.Processors, "memory_limiter")
		assert.Contains(t, otelConfig.Processors, "batch")
		assert.Contains(t, otelConfig.Processors, "nrsecurity")
		assert.Contains(t, otelConfig.Processors, "nrenrich")
		assert.Contains(t, otelConfig.Processors, "nrcap")
		assert.Contains(t, otelConfig.Processors, "resource")

		// Verify exporters
		assert.Contains(t, otelConfig.Exporters, "otlp")
		otlpExporter := otelConfig.Exporters["otlp"].(map[string]interface{})
		assert.Equal(t, "https://otlp.nr-data.net", otlpExporter["endpoint"])
		assert.Equal(t, "gzip", otlpExporter["compression"])

		// Verify service pipelines
		assert.Contains(t, otelConfig.Service.Pipelines, "metrics")
		metricsPipeline := otelConfig.Service.Pipelines["metrics"]
		assert.Contains(t, metricsPipeline.Receivers, "hostmetrics")
		assert.Contains(t, metricsPipeline.Processors, "nrsecurity")
		assert.Contains(t, metricsPipeline.Exporters, "otlp")
	})

	t.Run("full config with all features", func(t *testing.T) {
		config := &schema.Config{
			Service: schema.ServiceConfig{
				Name:        "full-service",
				Environment: "staging",
				Version:     "v1.2.3",
				Tags: map[string]string{
					"team":   "platform",
					"region": "us-east",
				},
			},
			LicenseKey: "test-license-key",
			AccountID:  "123456",
			Metrics: schema.MetricsConfig{
				Enabled:  true,
				Interval: "30s",
				Include:  []string{"app.*", "custom.*"},
				Exclude:  []string{"*.debug"},
			},
			Traces: schema.TracesConfig{
				Enabled:    true,
				SampleRate: 0.1,
			},
			Logs: schema.LogsConfig{
				Enabled: true,
				Sources: []schema.LogSource{
					{
						Path:   "/var/log/app.log",
						Parser: "json",
						Attributes: map[string]string{
							"app": "main",
						},
					},
				},
			},
			Security: schema.SecurityConfig{
				RedactSecrets:     true,
				AllowedAttributes: []string{"user.id", "request.id"},
				BlockedAttributes: []string{"password", "token"},
				CustomRedactionPatterns: []string{
					"ssn:\\d{3}-\\d{2}-\\d{4}",
				},
			},
			Processing: schema.ProcessingConfig{
				CardinalityLimit: 50000,
				Enrichment: schema.EnrichmentConfig{
					AddHostMetadata:       true,
					AddCloudMetadata:      true,
					AddKubernetesMetadata: true,
				},
			},
			Export: schema.ExportConfig{
				Endpoint:    "https://otlp.nr-data.net",
				Region:      "EU",
				Compression: "gzip",
				Timeout:     "45s",
				Retry: schema.RetryConfig{
					Enabled:     true,
					MaxAttempts: 5,
					Backoff:     "10s",
				},
			},
			Logging: schema.LoggingConfig{
				Level:  "debug",
				Format: "json",
			},
		}

		gen := NewGenerator(config)
		otelConfig, err := gen.Generate()
		require.NoError(t, err)

		// Verify debug features
		assert.Contains(t, otelConfig.Extensions, "pprof")
		assert.Contains(t, otelConfig.Exporters, "debug")

		// Verify all receivers
		assert.Contains(t, otelConfig.Receivers, "hostmetrics")
		assert.Contains(t, otelConfig.Receivers, "otlp")
		assert.Contains(t, otelConfig.Receivers, "filelog")

		// Verify filter processor
		assert.Contains(t, otelConfig.Processors, "filter")
		filterProc := otelConfig.Processors["filter"].(map[string]interface{})
		metricsConfig := filterProc["metrics"].(map[string]interface{})
		includeConfig := metricsConfig["include"].(map[string]interface{})
		assert.Equal(t, config.Metrics.Include, includeConfig["metric_names"])

		// Verify security processor configuration
		securityProc := otelConfig.Processors["nrsecurity"].(map[string]interface{})
		assert.Equal(t, config.Security.AllowedAttributes, securityProc["allowed_attributes"])
		assert.Equal(t, config.Security.BlockedAttributes, securityProc["blocked_attributes"])

		// Verify EU endpoint
		otlpExporter := otelConfig.Exporters["otlp"].(map[string]interface{})
		assert.Equal(t, "https://otlp.eu01.nr-data.net", otlpExporter["endpoint"])
		headers := otlpExporter["headers"].(map[string]string)
		assert.Equal(t, "test-license-key", headers["api-key"])

		// Verify all pipelines
		assert.Contains(t, otelConfig.Service.Pipelines, "metrics")
		assert.Contains(t, otelConfig.Service.Pipelines, "traces")
		assert.Contains(t, otelConfig.Service.Pipelines, "logs")

		// Verify trace pipeline has sampler
		tracesPipeline := otelConfig.Service.Pipelines["traces"]
		assert.Contains(t, tracesPipeline.Processors, "probabilistic_sampler")
		samplerProc := otelConfig.Processors["probabilistic_sampler"].(map[string]interface{})
		assert.Equal(t, float64(10), samplerProc["sampling_percentage"]) // 0.1 * 100
	})

	t.Run("disabled features", func(t *testing.T) {
		config := &schema.Config{
			Service: schema.ServiceConfig{
				Name:        "minimal",
				Environment: "test",
			},
			Metrics: schema.MetricsConfig{
				Enabled: false,
			},
			Traces: schema.TracesConfig{
				Enabled: false,
			},
			Logs: schema.LogsConfig{
				Enabled: false,
			},
			Security: schema.SecurityConfig{
				RedactSecrets: false,
			},
			Processing: schema.ProcessingConfig{
				Enrichment: schema.EnrichmentConfig{
					AddHostMetadata:       false,
					AddCloudMetadata:      false,
					AddKubernetesMetadata: false,
				},
			},
			Export: schema.ExportConfig{
				Endpoint: "https://otlp.nr-data.net",
			},
			Logging: schema.LoggingConfig{
				Level:  "info",
				Format: "text",
			},
		}

		gen := NewGenerator(config)
		otelConfig, err := gen.Generate()
		require.NoError(t, err)

		// Should have minimal receivers
		assert.NotContains(t, otelConfig.Receivers, "hostmetrics")
		assert.NotContains(t, otelConfig.Receivers, "otlp")
		assert.NotContains(t, otelConfig.Receivers, "filelog")

		// Should not have optional processors
		assert.NotContains(t, otelConfig.Processors, "nrsecurity")
		assert.NotContains(t, otelConfig.Processors, "nrenrich")
		assert.NotContains(t, otelConfig.Processors, "filter")

		// Should have no pipelines
		assert.Empty(t, otelConfig.Service.Pipelines)
	})

	t.Run("YAML output", func(t *testing.T) {
		config := &schema.Config{
			Service: schema.ServiceConfig{
				Name:        "yaml-test",
				Environment: "production",
			},
			Metrics: schema.MetricsConfig{
				Enabled:  true,
				Interval: "60s",
			},
			Export: schema.ExportConfig{
				Endpoint: "https://otlp.nr-data.net",
			},
			Logging: schema.LoggingConfig{
				Level:  "info",
				Format: "text",
			},
		}

		gen := NewGenerator(config)
		otelConfig, err := gen.Generate()
		require.NoError(t, err)

		// Convert to YAML
		yamlData, err := otelConfig.ToYAML()
		require.NoError(t, err)
		require.NotEmpty(t, yamlData)

		// Parse back to verify it's valid YAML
		var parsed map[string]interface{}
		err = yaml.Unmarshal(yamlData, &parsed)
		require.NoError(t, err)

		// Verify structure
		assert.Contains(t, parsed, "receivers")
		assert.Contains(t, parsed, "processors")
		assert.Contains(t, parsed, "exporters")
		assert.Contains(t, parsed, "service")
	})
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"30s", "30s"},
		{"5m", "5m0s"},
		{"1h", "1h0m0s"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			d, err := parseDuration(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, d.String())
		})
	}
}