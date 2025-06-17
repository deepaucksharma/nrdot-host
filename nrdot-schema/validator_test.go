package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidator(t *testing.T) {
	validator, err := NewValidator()
	require.NoError(t, err)

	t.Run("valid minimal config", func(t *testing.T) {
		yaml := `
service:
  name: my-service
`
		config, err := validator.ValidateYAML([]byte(yaml))
		require.NoError(t, err)
		assert.Equal(t, "my-service", config.Service.Name)
		assert.Equal(t, "production", config.Service.Environment) // default
	})

	t.Run("valid full config", func(t *testing.T) {
		yaml := `
service:
  name: api-gateway
  environment: staging
  version: v1.2.3
  tags:
    team: platform
    region: us-east-1

license_key: 1234567890abcdef1234567890abcdef12345678
account_id: "123456"

metrics:
  enabled: true
  interval: 30s
  include:
    - "system.*"
    - "app.*"
  exclude:
    - "*.debug"

traces:
  enabled: true
  sample_rate: 0.5

logs:
  enabled: true
  sources:
    - path: /var/log/app/*.log
      parser: json
      attributes:
        app: gateway
    - path: /var/log/nginx/access.log
      parser: nginx

security:
  redact_secrets: true
  blocked_attributes:
    - password
    - credit_card
  custom_redaction_patterns:
    - "SSN:\\s*\\d{3}-\\d{2}-\\d{4}"

processing:
  cardinality_limit: 50000
  enrichment:
    add_host_metadata: true
    add_cloud_metadata: false

export:
  endpoint: https://otlp.eu.nr-data.net
  region: EU
  compression: gzip
  timeout: 45s
  retry:
    enabled: true
    max_attempts: 5
    backoff: 10s

logging:
  level: debug
  format: json
`
		config, err := validator.ValidateYAML([]byte(yaml))
		require.NoError(t, err)
		
		// Service validation
		assert.Equal(t, "api-gateway", config.Service.Name)
		assert.Equal(t, "staging", config.Service.Environment)
		assert.Equal(t, "v1.2.3", config.Service.Version)
		assert.Equal(t, "platform", config.Service.Tags["team"])
		
		// Metrics validation
		assert.True(t, config.Metrics.Enabled)
		assert.Equal(t, "30s", config.Metrics.Interval)
		assert.Contains(t, config.Metrics.Include, "system.*")
		
		// Security validation
		assert.True(t, config.Security.RedactSecrets)
		assert.Contains(t, config.Security.BlockedAttributes, "password")
		
		// Export validation
		assert.Equal(t, "https://otlp.eu.nr-data.net", config.Export.Endpoint)
		assert.Equal(t, "EU", config.Export.Region)
		assert.Equal(t, 5, config.Export.Retry.MaxAttempts)
	})

	t.Run("invalid service name", func(t *testing.T) {
		yaml := `
service:
  name: "invalid service name!"
`
		_, err := validator.ValidateYAML([]byte(yaml))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "service.name")
		assert.Contains(t, err.Error(), "pattern")
	})

	t.Run("missing required field", func(t *testing.T) {
		yaml := `
metrics:
  enabled: true
`
		_, err := validator.ValidateYAML([]byte(yaml))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "service")
		assert.Contains(t, err.Error(), "required")
	})

	t.Run("invalid license key", func(t *testing.T) {
		yaml := `
service:
  name: my-service
license_key: invalid-key
`
		_, err := validator.ValidateYAML([]byte(yaml))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "license_key")
	})

	t.Run("environment variable placeholder", func(t *testing.T) {
		yaml := `
service:
  name: my-service
license_key: ${NEW_RELIC_LICENSE_KEY}
account_id: ${NEW_RELIC_ACCOUNT_ID}
`
		config, err := validator.ValidateYAML([]byte(yaml))
		require.NoError(t, err)
		assert.Equal(t, "${NEW_RELIC_LICENSE_KEY}", config.LicenseKey)
		assert.Equal(t, "${NEW_RELIC_ACCOUNT_ID}", config.AccountID)
	})

	t.Run("invalid interval format", func(t *testing.T) {
		yaml := `
service:
  name: my-service
metrics:
  interval: 30
`
		_, err := validator.ValidateYAML([]byte(yaml))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "interval")
	})

	t.Run("sample rate validation", func(t *testing.T) {
		yaml := `
service:
  name: my-service
traces:
  sample_rate: 1.5
`
		_, err := validator.ValidateYAML([]byte(yaml))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sample_rate")
		assert.Contains(t, err.Error(), "less than or equal to")
	})

	t.Run("JSON validation", func(t *testing.T) {
		json := `{
  "service": {
    "name": "json-service",
    "environment": "test"
  },
  "metrics": {
    "enabled": false
  }
}`
		config, err := validator.ValidateJSON([]byte(json))
		require.NoError(t, err)
		assert.Equal(t, "json-service", config.Service.Name)
		assert.Equal(t, "test", config.Service.Environment)
		assert.False(t, config.Metrics.Enabled)
	})

	// Note: JSON Schema additionalProperties validation doesn't prevent
	// YAML parsing of unknown fields, it only validates the schema.
	// Unknown field rejection would need to be handled at the YAML parsing level.
}

func TestApplyDefaults(t *testing.T) {
	validator, err := NewValidator()
	require.NoError(t, err)

	yaml := `
service:
  name: minimal
`
	config, err := validator.ValidateYAML([]byte(yaml))
	require.NoError(t, err)

	// Check all defaults are applied
	assert.Equal(t, "production", config.Service.Environment)
	assert.Equal(t, "60s", config.Metrics.Interval)
	assert.Equal(t, 0.1, config.Traces.SampleRate)
	assert.True(t, config.Security.RedactSecrets)
	assert.Equal(t, 10000, config.Processing.CardinalityLimit)
	assert.True(t, config.Processing.Enrichment.AddHostMetadata)
	assert.Equal(t, "https://otlp.nr-data.net", config.Export.Endpoint)
	assert.Equal(t, "US", config.Export.Region)
	assert.Equal(t, "gzip", config.Export.Compression)
	assert.Equal(t, "30s", config.Export.Timeout)
	assert.True(t, config.Export.Retry.Enabled)
	assert.Equal(t, 3, config.Export.Retry.MaxAttempts)
	assert.Equal(t, "5s", config.Export.Retry.Backoff)
	assert.Equal(t, "info", config.Logging.Level)
	assert.Equal(t, "text", config.Logging.Format)
}