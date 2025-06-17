package main

import (
	"fmt"
	"log"

	"github.com/newrelic/nrdot-host/nrdot-schema"
	templatelib "github.com/newrelic/nrdot-host/nrdot-template-lib"
)

func main() {
	// Example: Create a production-ready configuration
	config := &schema.Config{
		Service: schema.ServiceConfig{
			Name:        "api-gateway",
			Environment: "production",
			Version:     "v2.1.0",
			Tags: map[string]string{
				"team":   "platform",
				"region": "us-east-1",
			},
		},
		LicenseKey: "${NEW_RELIC_LICENSE_KEY}",
		AccountID:  "${NEW_RELIC_ACCOUNT_ID}",
		Metrics: schema.MetricsConfig{
			Enabled:  true,
			Interval: "30s",
			Include:  []string{"http.*", "grpc.*", "db.*"},
			Exclude:  []string{"*.debug"},
		},
		Traces: schema.TracesConfig{
			Enabled:    true,
			SampleRate: 0.01, // 1% sampling for production
		},
		Logs: schema.LogsConfig{
			Enabled: true,
			Sources: []schema.LogSource{
				{
					Path:   "/var/log/app/*.log",
					Parser: "json",
					Attributes: map[string]string{
						"service": "api",
					},
				},
			},
		},
		Security: schema.SecurityConfig{
			RedactSecrets:     true,
			BlockedAttributes: []string{"password", "api_key", "token"},
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
			Format: "json",
		},
	}

	// Generate OTel configuration
	generator := templatelib.NewGenerator(config)
	otelConfig, err := generator.Generate()
	if err != nil {
		log.Fatalf("Failed to generate configuration: %v", err)
	}

	// Convert to YAML
	yamlData, err := otelConfig.ToYAML()
	if err != nil {
		log.Fatalf("Failed to convert to YAML: %v", err)
	}

	fmt.Println("Generated OpenTelemetry Collector Configuration:")
	fmt.Println("================================================")
	fmt.Println(string(yamlData))
}