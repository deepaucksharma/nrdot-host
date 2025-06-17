// Package schema provides internal schema validation for NRDOT configurations
package schema

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/newrelic/nrdot-host/nrdot-common/pkg/models"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

// Validator validates user configurations against the NRDOT schema
type Validator struct {
	schema        *gojsonschema.Schema
	schemaVersion string
}

// NewValidator creates a new validator with embedded schema
func NewValidator() *Validator {
	// In production, this would load from nrdot-schema module
	// For now, using embedded schema
	schemaJSON := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["service"],
		"properties": {
			"service": {
				"type": "object",
				"required": ["name"],
				"properties": {
					"name": {"type": "string"},
					"environment": {"type": "string"},
					"version": {"type": "string"},
					"tags": {"type": "object"}
				}
			},
			"license_key": {"type": "string"},
			"metrics": {
				"type": "object",
				"properties": {
					"enabled": {"type": "boolean"},
					"interval": {"type": "string"},
					"host_metrics": {"type": "boolean"},
					"process_metrics": {"type": "boolean"}
				}
			},
			"traces": {
				"type": "object",
				"properties": {
					"enabled": {"type": "boolean"},
					"sample_rate": {"type": "number", "minimum": 0, "maximum": 1}
				}
			},
			"logs": {
				"type": "object",
				"properties": {
					"enabled": {"type": "boolean"},
					"paths": {"type": "array", "items": {"type": "string"}},
					"include_stdout": {"type": "boolean"},
					"include_stderr": {"type": "boolean"}
				}
			},
			"security": {
				"type": "object",
				"properties": {
					"redact_secrets": {"type": "boolean"},
					"redact_patterns": {"type": "array", "items": {"type": "string"}}
				}
			},
			"processing": {
				"type": "object",
				"properties": {
					"enrich": {
						"type": "object",
						"properties": {
							"add_host_metadata": {"type": "boolean"},
							"add_cloud_metadata": {"type": "boolean"},
							"add_k8s_metadata": {"type": "boolean"}
						}
					},
					"cardinality": {
						"type": "object",
						"properties": {
							"enabled": {"type": "boolean"},
							"global_limit": {"type": "integer", "minimum": 1000}
						}
					}
				}
			}
		}
	}`

	schemaLoader := gojsonschema.NewStringLoader(schemaJSON)
	schema, _ := gojsonschema.NewSchema(schemaLoader)

	return &Validator{
		schema:        schema,
		schemaVersion: "1.0.0",
	}
}

// Validate validates YAML configuration and returns parsed config
func (v *Validator) Validate(yamlData []byte) (*models.Config, error) {
	// Parse YAML to interface{}
	var data interface{}
	if err := yaml.Unmarshal(yamlData, &data); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}

	// Convert to JSON for schema validation
	jsonData, err := json.Marshal(data)
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
		var errors []string
		for _, err := range result.Errors() {
			errors = append(errors, fmt.Sprintf("%s: %s", err.Field(), err.Description()))
		}
		return nil, fmt.Errorf("schema validation failed: %v", errors)
	}

	// Parse into Config struct
	var config models.Config
	decoder := yaml.NewDecoder(bytes.NewReader(yamlData))
	decoder.KnownFields(true)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Apply defaults
	v.applyDefaults(&config)

	return &config, nil
}

// GetSchemaVersion returns the schema version
func (v *Validator) GetSchemaVersion() string {
	return v.schemaVersion
}

// applyDefaults applies default values to the configuration
func (v *Validator) applyDefaults(config *models.Config) {
	// Metrics defaults
	if config.Metrics.Interval == 0 {
		config.Metrics.Interval = 60 // 60 seconds
	}
	
	// Security defaults
	if !config.Security.RedactSecrets {
		config.Security.RedactSecrets = true // Enable by default
	}
	
	// Processing defaults
	if !config.Processing.Enrich.AddHostMetadata {
		config.Processing.Enrich.AddHostMetadata = true
	}
	
	// Cardinality defaults
	if config.Processing.Cardinality.Enabled && config.Processing.Cardinality.GlobalLimit == 0 {
		config.Processing.Cardinality.GlobalLimit = 100000
	}
}