// Package templates provides internal template generation for OpenTelemetry configs
package templates

import (
	"fmt"
	"strings"

	"github.com/newrelic/nrdot-host/nrdot-common/pkg/models"
)

// Generator generates OpenTelemetry collector configurations from NRDOT configs
type Generator struct {
	templates map[string]string
}

// NewGenerator creates a new template generator
func NewGenerator() *Generator {
	return &Generator{
		templates: defaultTemplates(),
	}
}

// Generate creates an OpenTelemetry configuration from NRDOT config
func (g *Generator) Generate(config *models.Config) (map[string]interface{}, []string, error) {
	otelConfig := make(map[string]interface{})
	templatesUsed := []string{}

	// Build receivers
	receivers := make(map[string]interface{})
	if config.Metrics.Enabled {
		if config.Metrics.HostMetrics {
			receivers["hostmetrics"] = g.buildHostMetricsReceiver(config)
			templatesUsed = append(templatesUsed, "hostmetrics_receiver")
		}
		if config.Metrics.ProcessMetrics {
			receivers["prometheus"] = g.buildPrometheusReceiver()
			templatesUsed = append(templatesUsed, "prometheus_receiver")
		}
	}
	
	if config.Logs.Enabled {
		receivers["filelog"] = g.buildFilelogReceiver(config)
		templatesUsed = append(templatesUsed, "filelog_receiver")
	}
	
	if config.Traces.Enabled {
		receivers["otlp"] = g.buildOTLPReceiver()
		templatesUsed = append(templatesUsed, "otlp_receiver")
	}

	// Build processors
	processors := g.buildProcessors(config)
	for name := range processors {
		templatesUsed = append(templatesUsed, name+"_processor")
	}

	// Build exporters
	exporters := make(map[string]interface{})
	exporters["otlp/newrelic"] = g.buildNewRelicExporter(config)
	templatesUsed = append(templatesUsed, "newrelic_exporter")

	// Build pipelines
	pipelines := g.buildPipelines(config, receivers, processors, exporters)

	// Build service
	service := map[string]interface{}{
		"pipelines": pipelines,
		"telemetry": map[string]interface{}{
			"logs": map[string]interface{}{
				"level": "info",
			},
			"metrics": map[string]interface{}{
				"address": ":8888",
			},
		},
	}

	// Add extensions
	extensions := map[string]interface{}{
		"health_check": map[string]interface{}{},
		"zpages":       map[string]interface{}{},
	}

	// Assemble final config
	otelConfig["receivers"] = receivers
	otelConfig["processors"] = processors
	otelConfig["exporters"] = exporters
	otelConfig["extensions"] = extensions
	otelConfig["service"] = service

	return otelConfig, templatesUsed, nil
}

// buildHostMetricsReceiver builds the host metrics receiver config
func (g *Generator) buildHostMetricsReceiver(config *models.Config) map[string]interface{} {
	return map[string]interface{}{
		"collection_interval": fmt.Sprintf("%ds", int(config.Metrics.Interval.Seconds())),
		"scrapers": map[string]interface{}{
			"cpu":        map[string]interface{}{},
			"disk":       map[string]interface{}{},
			"filesystem": map[string]interface{}{},
			"load":       map[string]interface{}{},
			"memory":     map[string]interface{}{},
			"network":    map[string]interface{}{},
			"process":    map[string]interface{}{},
		},
	}
}

// buildPrometheusReceiver builds the prometheus receiver config
func (g *Generator) buildPrometheusReceiver() map[string]interface{} {
	return map[string]interface{}{
		"config": map[string]interface{}{
			"scrape_configs": []interface{}{
				map[string]interface{}{
					"job_name":        "otel-collector",
					"scrape_interval": "30s",
					"static_configs": []interface{}{
						map[string]interface{}{
							"targets": []string{"localhost:8888"},
						},
					},
				},
			},
		},
	}
}

// buildFilelogReceiver builds the filelog receiver config
func (g *Generator) buildFilelogReceiver(config *models.Config) map[string]interface{} {
	cfg := map[string]interface{}{
		"include": config.Logs.Paths,
		"operators": []interface{}{
			map[string]interface{}{
				"type": "regex_parser",
				"regex": "^(?P<time>[^ ]*) (?P<severity>[^ ]*) (?P<message>.*)$",
				"timestamp": map[string]interface{}{
					"parse_from": "attributes.time",
					"layout":     "%Y-%m-%d %H:%M:%S",
				},
			},
		},
	}
	
	if config.Logs.IncludeStdout || config.Logs.IncludeStderr {
		// Add console logs
		cfg["include_stdout"] = config.Logs.IncludeStdout
		cfg["include_stderr"] = config.Logs.IncludeStderr
	}
	
	return cfg
}

// buildOTLPReceiver builds the OTLP receiver config
func (g *Generator) buildOTLPReceiver() map[string]interface{} {
	return map[string]interface{}{
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

// buildProcessors builds all configured processors
func (g *Generator) buildProcessors(config *models.Config) map[string]interface{} {
	processors := make(map[string]interface{})

	// Always add batch processor first
	processors["batch"] = map[string]interface{}{
		"timeout": "5s",
		"send_batch_size": 1000,
	}

	// Security processor
	if config.Security.RedactSecrets {
		processors["nrsecurity"] = map[string]interface{}{
			"enabled": true,
			"patterns": config.Security.RedactPatterns,
		}
	}

	// Enrichment processor
	if config.Processing.Enrich.AddHostMetadata || 
	   config.Processing.Enrich.AddCloudMetadata || 
	   config.Processing.Enrich.AddK8sMetadata {
		processors["nrenrich"] = map[string]interface{}{
			"host_metadata":  config.Processing.Enrich.AddHostMetadata,
			"cloud_metadata": config.Processing.Enrich.AddCloudMetadata,
			"k8s_metadata":   config.Processing.Enrich.AddK8sMetadata,
			"custom_tags":    config.Processing.Enrich.CustomTags,
		}
	}

	// Transform processor
	if config.Processing.Transform.ConvertUnits || 
	   len(config.Processing.Transform.Aggregations) > 0 ||
	   len(config.Processing.Transform.Calculations) > 0 {
		processors["nrtransform"] = map[string]interface{}{
			"convert_units": config.Processing.Transform.ConvertUnits,
			"aggregations":  config.Processing.Transform.Aggregations,
			"calculations":  config.Processing.Transform.Calculations,
		}
	}

	// Cardinality processor
	if config.Processing.Cardinality.Enabled {
		processors["nrcap"] = map[string]interface{}{
			"enabled":      true,
			"global_limit": config.Processing.Cardinality.GlobalLimit,
			"per_metric":   config.Processing.Cardinality.PerMetric,
			"action":       config.Processing.Cardinality.LimitAction,
		}
	}

	// Memory limiter (always last)
	processors["memory_limiter"] = map[string]interface{}{
		"check_interval": "1s",
		"limit_mib": 512,
		"spike_limit_mib": 128,
	}

	return processors
}

// buildNewRelicExporter builds the New Relic exporter config
func (g *Generator) buildNewRelicExporter(config *models.Config) map[string]interface{} {
	exporter := map[string]interface{}{
		"endpoint": "https://otlp.nr-data.net:4317",
		"headers": map[string]interface{}{
			"api-key": config.LicenseKey,
		},
		"compression": "gzip",
		"retry_on_failure": map[string]interface{}{
			"enabled": true,
			"initial_interval": "5s",
			"max_interval": "30s",
			"max_elapsed_time": "300s",
		},
	}

	// Add custom headers
	if config.Export.Headers != nil {
		for k, v := range config.Export.Headers {
			exporter["headers"].(map[string]interface{})[k] = v
		}
	}

	return exporter
}

// buildPipelines builds the pipeline configurations
func (g *Generator) buildPipelines(config *models.Config, receivers, processors, exporters map[string]interface{}) map[string]interface{} {
	pipelines := make(map[string]interface{})
	
	// Get processor names in order
	processorNames := g.getProcessorOrder(processors)

	if config.Metrics.Enabled && len(receivers) > 0 {
		var metricReceivers []string
		if _, ok := receivers["hostmetrics"]; ok {
			metricReceivers = append(metricReceivers, "hostmetrics")
		}
		if _, ok := receivers["prometheus"]; ok {
			metricReceivers = append(metricReceivers, "prometheus")
		}
		
		if len(metricReceivers) > 0 {
			pipelines["metrics"] = map[string]interface{}{
				"receivers":  metricReceivers,
				"processors": processorNames,
				"exporters":  []string{"otlp/newrelic"},
			}
		}
	}

	if config.Traces.Enabled {
		pipelines["traces"] = map[string]interface{}{
			"receivers":  []string{"otlp"},
			"processors": processorNames,
			"exporters":  []string{"otlp/newrelic"},
		}
	}

	if config.Logs.Enabled {
		pipelines["logs"] = map[string]interface{}{
			"receivers":  []string{"filelog"},
			"processors": processorNames,
			"exporters":  []string{"otlp/newrelic"},
		}
	}

	return pipelines
}

// getProcessorOrder returns processors in the correct order
func (g *Generator) getProcessorOrder(processors map[string]interface{}) []string {
	order := []string{}
	
	// Define the correct order
	processorOrder := []string{
		"batch",
		"nrsecurity",
		"nrenrich", 
		"nrtransform",
		"nrcap",
		"memory_limiter",
	}
	
	// Add only configured processors in order
	for _, name := range processorOrder {
		if _, exists := processors[name]; exists {
			order = append(order, name)
		}
	}
	
	return order
}

// defaultTemplates returns the default template configurations
func defaultTemplates() map[string]string {
	return map[string]string{
		"version": "1.0.0",
	}
}

// GetTemplateInfo returns information about available templates
func (g *Generator) GetTemplateInfo() map[string]string {
	info := make(map[string]string)
	info["generator_version"] = "2.0"
	info["otel_version"] = "0.96.0"
	info["template_count"] = fmt.Sprintf("%d", len(g.templates))
	return info
}