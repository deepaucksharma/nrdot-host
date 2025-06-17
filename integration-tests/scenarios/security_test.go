package scenarios

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/newrelic/nrdot-host/integration-tests/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurityFeatures(t *testing.T) {
	suite := framework.NewTestSuite(t)
	
	t.Run("SecretRedactionInLogs", func(t *testing.T) {
		// Start collector with security configuration
		collector := suite.StartCollector("../fixtures/configs/security.yaml")
		
		// Generate logs with secrets
		generator := framework.NewTelemetryGeneratorWithConfig(&framework.TelemetryConfig{
			ServiceName:    "test-service",
			HostName:       "test-host",
			IncludeSecrets: true,
		})
		
		logs := generator.GenerateLogs(10)
		
		// Send logs to collector
		err := collector.SendLogs(logs)
		require.NoError(t, err)
		
		// Wait for processing
		time.Sleep(2 * time.Second)
		
		// Get collector logs to verify redaction
		collectorLogs, err := collector.GetLogs(context.Background())
		require.NoError(t, err)
		
		// Verify secrets are redacted
		secretPatterns := []string{
			"password", "api_key", "secret", "token",
			"database_password", "auth_token",
		}
		
		for _, pattern := range secretPatterns {
			// Should not contain actual secret values
			assert.NotContains(t, collectorLogs, "secret123")
			assert.NotContains(t, collectorLogs, "key456")
			assert.NotContains(t, collectorLogs, "db-secret-pass")
			assert.NotContains(t, collectorLogs, "secret-token-xyz")
		}
	})
	
	t.Run("SecretRedactionInTraces", func(t *testing.T) {
		collector := suite.StartCollector("../fixtures/configs/security.yaml")
		
		// Generate traces with secrets
		generator := framework.NewTelemetryGeneratorWithConfig(&framework.TelemetryConfig{
			ServiceName:    "test-service",
			HostName:       "test-host",
			IncludeSecrets: true,
		})
		
		traces := generator.GenerateTraces(5)
		
		// Send traces to collector
		err := collector.SendTraces(traces)
		require.NoError(t, err)
		
		// Wait for processing
		time.Sleep(2 * time.Second)
		
		// Verify secrets are not exposed in collector logs
		collectorLogs, err := collector.GetLogs(context.Background())
		require.NoError(t, err)
		
		// Check for secret values
		assert.NotContains(t, collectorLogs, "secret-api-key-12345")
		assert.NotContains(t, collectorLogs, "my-secret-password")
	})
	
	t.Run("SecretRedactionInMetrics", func(t *testing.T) {
		collector := suite.StartCollector("../fixtures/configs/security.yaml")
		
		// Generate metrics with sensitive labels
		generator := framework.NewTelemetryGeneratorWithConfig(&framework.TelemetryConfig{
			ServiceName: "test-service",
			HostName:    "test-host",
			ResourceAttrs: map[string]string{
				"api_key":  "sensitive-api-key",
				"password": "sensitive-password",
				"token":    "Bearer secret-token",
			},
		})
		
		metrics := generator.GenerateMetrics(10)
		
		// Send metrics to collector
		err := collector.SendMetrics(metrics)
		require.NoError(t, err)
		
		// Wait for processing
		time.Sleep(2 * time.Second)
		
		// Get exported metrics
		exportedMetrics, err := collector.GetMetrics(context.Background())
		require.NoError(t, err)
		
		// Verify sensitive values are not exposed
		assert.NotContains(t, string(exportedMetrics), "sensitive-api-key")
		assert.NotContains(t, string(exportedMetrics), "sensitive-password")
		assert.NotContains(t, string(exportedMetrics), "secret-token")
	})
	
	t.Run("ConfigurationSecrets", func(t *testing.T) {
		// Start collector and check that configuration secrets are not logged
		collector := suite.StartCollector("../fixtures/configs/security.yaml")
		
		// Get collector logs
		logs, err := collector.GetLogs(context.Background())
		require.NoError(t, err)
		
		// Configuration should not expose sensitive values
		assert.NotContains(t, logs, "actual-api-key-value")
		assert.NotContains(t, logs, "actual-secret-value")
		
		// But should show redacted indicators
		if strings.Contains(logs, "api_key") {
			assert.Contains(t, logs, "***")
		}
	})
}

func TestSecurityPatterns(t *testing.T) {
	suite := framework.NewTestSuite(t)
	
	t.Run("CommonSecretPatterns", func(t *testing.T) {
		collector := suite.StartCollector("../fixtures/configs/security.yaml")
		
		// Test various secret patterns
		secretPatterns := map[string]string{
			"AWS_ACCESS_KEY_ID":     "AKIAIOSFODNN7EXAMPLE",
			"AWS_SECRET_ACCESS_KEY": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			"DATABASE_URL":          "postgres://user:password@localhost/db",
			"GITHUB_TOKEN":          "ghp_1234567890abcdefghijklmnopqrstuvwxyz",
			"PRIVATE_KEY":           "-----BEGIN PRIVATE KEY-----\nMIIEvQ...",
			"JWT_TOKEN":             "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
			"CREDIT_CARD":           "4111111111111111",
			"SSN":                   "123-45-6789",
		}
		
		// Generate logs with these patterns
		generator := framework.NewTelemetryGenerator()
		logs := generator.GenerateLogs(1)
		
		// Add secret patterns to log attributes
		logRecord := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
		for key, value := range secretPatterns {
			logRecord.Attributes().PutStr(key, value)
		}
		
		// Send logs
		err := collector.SendLogs(logs)
		require.NoError(t, err)
		
		// Wait for processing
		time.Sleep(2 * time.Second)
		
		// Verify none of the actual secret values appear in logs
		collectorLogs, err := collector.GetLogs(context.Background())
		require.NoError(t, err)
		
		for _, secretValue := range secretPatterns {
			assert.NotContains(t, collectorLogs, secretValue,
				"Found unredacted secret value: %s", secretValue)
		}
	})
	
	t.Run("EnvironmentVariableSecrets", func(t *testing.T) {
		// Start collector with environment variables containing secrets
		config := &framework.CollectorConfig{
			Name:       "security-env-test",
			Image:      framework.DefaultTestConfig().CollectorImage,
			ConfigPath: "../fixtures/configs/security.yaml",
			Network:    suite.Network(),
			Logger:     suite.Logger(),
			EnvVars: map[string]string{
				"NEW_RELIC_API_KEY": "test-api-key-12345",
				"DATABASE_PASSWORD": "super-secret-password",
				"AUTH_TOKEN":        "Bearer secret-token",
			},
			HealthEndpoint: "/health",
			MetricsPort:    8888,
			OTLPGRPCPort:   4317,
			OTLPHTTPPort:   4318,
		}
		
		collector := suite.StartCollectorWithConfig(config)
		
		// Get collector logs
		logs, err := collector.GetLogs(context.Background())
		require.NoError(t, err)
		
		// Verify environment secrets are not exposed
		assert.NotContains(t, logs, "test-api-key-12345")
		assert.NotContains(t, logs, "super-secret-password")
		assert.NotContains(t, logs, "secret-token")
	})
}

func TestSecretRedactionPerformance(t *testing.T) {
	suite := framework.NewTestSuite(t)
	
	t.Run("HighVolumeSecretRedaction", func(t *testing.T) {
		collector := suite.StartCollector("../fixtures/configs/security.yaml")
		
		// Get baseline resource usage
		cpuBefore, memBefore, err := collector.GetResourceUsage(context.Background())
		require.NoError(t, err)
		
		// Generate high volume of logs with secrets
		generator := framework.NewTelemetryGeneratorWithConfig(&framework.TelemetryConfig{
			ServiceName:    "test-service",
			HostName:       "test-host",
			IncludeSecrets: true,
		})
		
		// Send many logs with secrets
		start := time.Now()
		for i := 0; i < 100; i++ {
			logs := generator.GenerateLogs(100)
			err := collector.SendLogs(logs)
			require.NoError(t, err)
		}
		duration := time.Since(start)
		
		// Wait for processing
		time.Sleep(5 * time.Second)
		
		// Check resource usage after
		cpuAfter, memAfter, err := collector.GetResourceUsage(context.Background())
		require.NoError(t, err)
		
		// Performance should still be reasonable
		assert.Less(t, duration, 30*time.Second, "Processing took too long")
		
		// Memory increase should be reasonable
		memIncrease := float64(memAfter-memBefore) / float64(memBefore) * 100
		assert.Less(t, memIncrease, 100.0, "Memory increased by more than 100%")
		
		suite.Logger().Info("Secret redaction performance",
			zap.Duration("duration", duration),
			zap.Float64("cpu_before", cpuBefore),
			zap.Float64("cpu_after", cpuAfter),
			zap.Uint64("mem_before", memBefore),
			zap.Uint64("mem_after", memAfter),
		)
	})
}