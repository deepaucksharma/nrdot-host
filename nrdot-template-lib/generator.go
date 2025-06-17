package templatelib

import (
	"fmt"
	"strings"
	"time"

	"github.com/newrelic/nrdot-host/nrdot-schema"
	"gopkg.in/yaml.v3"
)

// OTelConfig represents a complete OpenTelemetry Collector configuration
type OTelConfig struct {
	Extensions map[string]interface{} `yaml:"extensions,omitempty"`
	Receivers  map[string]interface{} `yaml:"receivers"`
	Processors map[string]interface{} `yaml:"processors"`
	Exporters  map[string]interface{} `yaml:"exporters"`
	Service    ServiceConfig          `yaml:"service"`
}

// ServiceConfig defines the OTel service configuration
type ServiceConfig struct {
	Extensions []string                          `yaml:"extensions,omitempty"`
	Pipelines  map[string]PipelineConfig         `yaml:"pipelines"`
	Telemetry  map[string]interface{}            `yaml:"telemetry,omitempty"`
}

// PipelineConfig defines a telemetry pipeline
type PipelineConfig struct {
	Receivers  []string `yaml:"receivers"`
	Processors []string `yaml:"processors"`
	Exporters  []string `yaml:"exporters"`
}

// Generator creates OTel configurations from NRDOT configs
type Generator struct {
	config *schema.Config
}

// NewGenerator creates a new configuration generator
func NewGenerator(config *schema.Config) *Generator {
	return &Generator{
		config: config,
	}
}

// Generate creates a complete OTel configuration
func (g *Generator) Generate() (*OTelConfig, error) {
	otelConfig := &OTelConfig{
		Extensions: g.generateExtensions(),
		Receivers:  g.generateReceivers(),
		Processors: g.generateProcessors(),
		Exporters:  g.generateExporters(),
		Service:    g.generateService(),
	}

	return otelConfig, nil
}

// generateExtensions creates OTel extensions configuration
func (g *Generator) generateExtensions() map[string]interface{} {
	extensions := make(map[string]interface{})

	// Health check extension
	extensions["health_check"] = map[string]interface{}{
		"endpoint": "0.0.0.0:13133",
		"path":     "/health",
	}

	// Performance profiler for debugging
	if g.config.Logging.Level == "debug" {
		extensions["pprof"] = map[string]interface{}{
			"endpoint": "0.0.0.0:1777",
		}
	}

	return extensions
}

// generateReceivers creates receiver configurations
func (g *Generator) generateReceivers() map[string]interface{} {
	receivers := make(map[string]interface{})

	// Host metrics receiver
	if g.config.Metrics.Enabled {
		interval, _ := parseDuration(g.config.Metrics.Interval)
		
		receivers["hostmetrics"] = map[string]interface{}{
			"collection_interval": interval.String(),
			"scrapers": map[string]interface{}{
				"cpu":        map[string]interface{}{},
				"disk":       map[string]interface{}{},
				"filesystem": map[string]interface{}{},
				"load":       map[string]interface{}{},
				"memory":     map[string]interface{}{},
				"network":    map[string]interface{}{},
				"paging":     map[string]interface{}{},
				"processes":  map[string]interface{}{},
			},
		}

		// Add Prometheus receiver for app metrics
		receivers["prometheus"] = map[string]interface{}{
			"config": map[string]interface{}{
				"scrape_configs": []map[string]interface{}{
					{
						"job_name":        "otel-collector",
						"scrape_interval": interval.String(),
						"static_configs": []map[string]interface{}{
							{
								"targets": []string{"0.0.0.0:8888"},
							},
						},
					},
				},
			},
		}
	}

	// OTLP receiver for traces
	if g.config.Traces.Enabled {
		receivers["otlp"] = map[string]interface{}{
			"protocols": map[string]interface{}{
				"grpc": map[string]interface{}{
					"endpoint": "0.0.0.0:4317",
				},
				"http": map[string]interface{}{
					"endpoint": "0.0.0.0:4318",
				},
			},
		}
	}

	// File log receiver
	if g.config.Logs.Enabled && len(g.config.Logs.Sources) > 0 {
		var operators []map[string]interface{}
		
		for _, source := range g.config.Logs.Sources {
			operator := map[string]interface{}{
				"type":    "file_input",
				"include": []string{source.Path},
			}
			
			// Add parser if specified
			if source.Parser != "" {
				operator["multiline"] = g.getMultilineConfig(source.Parser)
			}
			
			// Add attributes
			if len(source.Attributes) > 0 {
				operator["attributes"] = source.Attributes
			}
			
			operators = append(operators, operator)
		}
		
		receivers["filelog"] = map[string]interface{}{
			"operators": operators,
		}
	}

	return receivers
}

// generateProcessors creates processor configurations
func (g *Generator) generateProcessors() map[string]interface{} {
	processors := make(map[string]interface{})

	// Memory limiter (always enabled)
	processors["memory_limiter"] = map[string]interface{}{
		"check_interval":        "1s",
		"limit_percentage":      75,
		"spike_limit_percentage": 25,
	}

	// Batch processor (always enabled)
	processors["batch"] = map[string]interface{}{
		"send_batch_size":    10000,
		"timeout":            "10s",
		"send_batch_max_size": 11000,
	}

	// NR Security processor
	if g.config.Security.RedactSecrets {
		securityConfig := map[string]interface{}{
			"redact_secrets": true,
		}
		
		if len(g.config.Security.AllowedAttributes) > 0 {
			securityConfig["allowed_attributes"] = g.config.Security.AllowedAttributes
		}
		
		if len(g.config.Security.BlockedAttributes) > 0 {
			securityConfig["blocked_attributes"] = g.config.Security.BlockedAttributes
		}
		
		if len(g.config.Security.CustomRedactionPatterns) > 0 {
			securityConfig["custom_patterns"] = g.config.Security.CustomRedactionPatterns
		}
		
		processors["nrsecurity"] = securityConfig
	}

	// NR Enrichment processor
	if g.config.Processing.Enrichment.AddHostMetadata ||
		g.config.Processing.Enrichment.AddCloudMetadata ||
		g.config.Processing.Enrichment.AddKubernetesMetadata {
		
		enrichConfig := map[string]interface{}{}
		
		if g.config.Processing.Enrichment.AddHostMetadata {
			enrichConfig["add_host_metadata"] = true
		}
		
		if g.config.Processing.Enrichment.AddCloudMetadata {
			enrichConfig["add_cloud_metadata"] = true
		}
		
		if g.config.Processing.Enrichment.AddKubernetesMetadata {
			enrichConfig["add_kubernetes_metadata"] = true
		}
		
		processors["nrenrich"] = enrichConfig
	}

	// NR Transform processor (for calculated metrics)
	processors["nrtransform"] = map[string]interface{}{
		"error_mode": "ignore",
		"metric_statements": []map[string]interface{}{
			{
				"context": "datapoint",
				"statements": []string{
					// Add standard calculations here
				},
			},
		},
	}

	// NR Cardinality Cap processor
	if g.config.Processing.CardinalityLimit > 0 {
		processors["nrcap"] = map[string]interface{}{
			"cardinality_limit": g.config.Processing.CardinalityLimit,
		}
	}

	// Resource processor for service identification
	resourceAttrs := map[string]interface{}{
		"service.name":        g.config.Service.Name,
		"service.environment": g.config.Service.Environment,
	}
	
	if g.config.Service.Version != "" {
		resourceAttrs["service.version"] = g.config.Service.Version
	}
	
	// Add custom tags
	for k, v := range g.config.Service.Tags {
		resourceAttrs[fmt.Sprintf("tags.%s", k)] = v
	}
	
	processors["resource"] = map[string]interface{}{
		"attributes": []map[string]interface{}{
			{
				"key":    "service.name",
				"value":  g.config.Service.Name,
				"action": "insert",
			},
			{
				"key":    "deployment.environment", 
				"value":  g.config.Service.Environment,
				"action": "insert",
			},
		},
	}

	// Filter processor for include/exclude patterns
	if len(g.config.Metrics.Include) > 0 || len(g.config.Metrics.Exclude) > 0 {
		filterConfig := map[string]interface{}{
			"error_mode": "ignore",
		}
		
		if len(g.config.Metrics.Include) > 0 {
			filterConfig["metrics"] = map[string]interface{}{
				"include": map[string]interface{}{
					"match_type": "regexp",
					"metric_names": g.config.Metrics.Include,
				},
			}
		}
		
		if len(g.config.Metrics.Exclude) > 0 {
			if filterConfig["metrics"] == nil {
				filterConfig["metrics"] = map[string]interface{}{}
			}
			metricsConfig := filterConfig["metrics"].(map[string]interface{})
			metricsConfig["exclude"] = map[string]interface{}{
				"match_type": "regexp",
				"metric_names": g.config.Metrics.Exclude,
			}
		}
		
		processors["filter"] = filterConfig
	}

	// Probabilistic sampler for traces
	if g.config.Traces.Enabled && g.config.Traces.SampleRate < 1.0 {
		processors["probabilistic_sampler"] = map[string]interface{}{
			"sampling_percentage": g.config.Traces.SampleRate * 100,
		}
	}

	return processors
}

// generateExporters creates exporter configurations
func (g *Generator) generateExporters() map[string]interface{} {
	exporters := make(map[string]interface{})

	// OTLP exporter to New Relic
	endpoint := g.config.Export.Endpoint
	if endpoint == "" {
		endpoint = "https://otlp.nr-data.net"
	}
	
	// Adjust endpoint for EU region
	if g.config.Export.Region == "EU" {
		endpoint = strings.Replace(endpoint, "otlp.nr-data.net", "otlp.eu01.nr-data.net", 1)
	}

	headers := map[string]string{}
	
	// Add API key if provided
	if g.config.LicenseKey != "" {
		headers["api-key"] = g.config.LicenseKey
	}

	otlpConfig := map[string]interface{}{
		"endpoint": endpoint,
		"headers":  headers,
	}

	// Compression settings
	if g.config.Export.Compression != "none" {
		otlpConfig["compression"] = g.config.Export.Compression
	}

	// Timeout settings
	if g.config.Export.Timeout != "" {
		timeout, _ := parseDuration(g.config.Export.Timeout)
		otlpConfig["timeout"] = timeout.String()
	}

	// Retry settings
	if g.config.Export.Retry.Enabled {
		otlpConfig["retry_on_failure"] = map[string]interface{}{
			"enabled":         true,
			"max_attempts":    g.config.Export.Retry.MaxAttempts,
			"initial_interval": g.config.Export.Retry.Backoff,
		}
	}

	exporters["otlp"] = otlpConfig

	// Debug exporter for development
	if g.config.Logging.Level == "debug" {
		exporters["debug"] = map[string]interface{}{
			"verbosity": "detailed",
			"sampling_initial": 10,
			"sampling_thereafter": 100,
		}
	}

	return exporters
}

// generateService creates the service configuration
func (g *Generator) generateService() ServiceConfig {
	service := ServiceConfig{
		Extensions: []string{"health_check"},
		Pipelines:  make(map[string]PipelineConfig),
		Telemetry: map[string]interface{}{
			"logs": map[string]interface{}{
				"level": g.config.Logging.Level,
				"encoding": g.config.Logging.Format,
			},
			"metrics": map[string]interface{}{
				"level": "detailed",
				"address": "0.0.0.0:8888",
			},
		},
	}

	if g.config.Logging.Level == "debug" {
		service.Extensions = append(service.Extensions, "pprof")
	}

	// Build processor pipeline
	baseProcessors := []string{"memory_limiter", "batch"}
	
	// Metrics pipeline
	if g.config.Metrics.Enabled {
		processors := append([]string{}, baseProcessors...)
		
		// Add processors in order
		if _, exists := g.generateProcessors()["filter"]; exists {
			processors = append(processors, "filter")
		}
		if g.config.Security.RedactSecrets {
			processors = append(processors, "nrsecurity")
		}
		if _, exists := g.generateProcessors()["nrenrich"]; exists {
			processors = append(processors, "nrenrich")
		}
		processors = append(processors, "nrtransform")
		if g.config.Processing.CardinalityLimit > 0 {
			processors = append(processors, "nrcap")
		}
		processors = append(processors, "resource")
		
		exporters := []string{"otlp"}
		if g.config.Logging.Level == "debug" {
			exporters = append(exporters, "debug")
		}
		
		service.Pipelines["metrics"] = PipelineConfig{
			Receivers:  []string{"hostmetrics", "prometheus"},
			Processors: processors,
			Exporters:  exporters,
		}
	}

	// Traces pipeline
	if g.config.Traces.Enabled {
		processors := append([]string{}, baseProcessors...)
		
		if g.config.Traces.SampleRate < 1.0 {
			processors = append(processors, "probabilistic_sampler")
		}
		if g.config.Security.RedactSecrets {
			processors = append(processors, "nrsecurity")
		}
		if _, exists := g.generateProcessors()["nrenrich"]; exists {
			processors = append(processors, "nrenrich")
		}
		processors = append(processors, "resource")
		
		exporters := []string{"otlp"}
		if g.config.Logging.Level == "debug" {
			exporters = append(exporters, "debug")
		}
		
		service.Pipelines["traces"] = PipelineConfig{
			Receivers:  []string{"otlp"},
			Processors: processors,
			Exporters:  exporters,
		}
	}

	// Logs pipeline
	if g.config.Logs.Enabled && len(g.config.Logs.Sources) > 0 {
		processors := append([]string{}, baseProcessors...)
		
		if g.config.Security.RedactSecrets {
			processors = append(processors, "nrsecurity")
		}
		if _, exists := g.generateProcessors()["nrenrich"]; exists {
			processors = append(processors, "nrenrich")
		}
		processors = append(processors, "resource")
		
		exporters := []string{"otlp"}
		if g.config.Logging.Level == "debug" {
			exporters = append(exporters, "debug")
		}
		
		service.Pipelines["logs"] = PipelineConfig{
			Receivers:  []string{"filelog"},
			Processors: processors,
			Exporters:  exporters,
		}
	}

	return service
}

// getMultilineConfig returns multiline configuration for log parsers
func (g *Generator) getMultilineConfig(parser string) map[string]interface{} {
	switch parser {
	case "multiline":
		return map[string]interface{}{
			"line_start_pattern": `^\d{4}-\d{2}-\d{2}`,
		}
	default:
		return nil
	}
}

// parseDuration parses a duration string
func parseDuration(s string) (time.Duration, error) {
	// Handle simple formats like "30s", "5m"
	return time.ParseDuration(s)
}

// ToYAML converts the OTel configuration to YAML
func (c *OTelConfig) ToYAML() ([]byte, error) {
	return yaml.Marshal(c)
}