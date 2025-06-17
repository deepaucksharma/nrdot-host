package integrationtests

import (
	"context"
	"testing"
	"time"

	"github.com/newrelic/nrdot-host/integration-tests/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ExampleBasicIntegrationTest demonstrates how to write an integration test
func ExampleBasicIntegrationTest(t *testing.T) {
	// Create a test suite
	suite := framework.NewTestSuite(t)
	
	// Start a collector with a specific configuration
	collector := suite.StartCollector("fixtures/configs/basic.yaml")
	
	// Generate test telemetry
	generator := framework.NewTelemetryGenerator()
	metrics := generator.GenerateMetrics(10)
	
	// Send metrics to the collector
	err := collector.SendMetrics(metrics)
	require.NoError(t, err)
	
	// Wait for processing
	time.Sleep(2 * time.Second)
	
	// Verify the collector processed the metrics
	collectorMetrics, err := collector.GetMetrics(context.Background())
	require.NoError(t, err)
	
	// Check that metrics were received
	assert.Contains(t, string(collectorMetrics), "otelcol_receiver_accepted_metric_points")
}

// ExampleCustomConfiguration demonstrates testing with custom configuration
func ExampleCustomConfiguration(t *testing.T) {
	suite := framework.NewTestSuite(t)
	
	// Create a custom collector configuration
	config := &framework.CollectorConfig{
		Name:           "custom-collector",
		Image:          "nrdot-host:latest",
		ConfigPath:     "fixtures/configs/enrichment.yaml",
		Network:        suite.Network(),
		Logger:         suite.Logger(),
		HealthEndpoint: "/health",
		MetricsPort:    8888,
		OTLPGRPCPort:   4317,
		OTLPHTTPPort:   4318,
		EnvVars: map[string]string{
			"LOG_LEVEL": "debug",
			"REGION":    "us-east-1",
		},
	}
	
	// Start the collector
	collector := suite.StartCollectorWithConfig(config)
	
	// Test enrichment functionality
	generator := framework.NewTelemetryGeneratorWithConfig(&framework.TelemetryConfig{
		ServiceName: "test-service",
		HostName:    "test-host",
		ResourceAttrs: map[string]string{
			"custom.attribute": "test-value",
		},
	})
	
	metrics := generator.GenerateMetrics(5)
	err := collector.SendMetrics(metrics)
	require.NoError(t, err)
	
	// Verify enrichment
	framework.AssertMetricsEnriched(t, metrics, "environment", "region", "team")
}

// ExampleHighCardinalityTest demonstrates testing cardinality limits
func ExampleHighCardinalityTest(t *testing.T) {
	suite := framework.NewTestSuite(t)
	collector := suite.StartCollector("fixtures/configs/cardinality.yaml")
	
	// Generate high cardinality metrics
	generator := framework.NewTelemetryGenerator()
	highCardMetrics := generator.GenerateHighCardinalityMetrics(10, 100)
	
	// Send metrics
	err := collector.SendMetrics(highCardMetrics)
	require.NoError(t, err)
	
	// Wait for processing
	time.Sleep(3 * time.Second)
	
	// Verify cardinality limiting
	framework.AssertCardinalityWithinLimits(t, highCardMetrics, 10000)
	
	// Check resource usage
	cpu, memory, err := collector.GetResourceUsage(context.Background())
	require.NoError(t, err)
	
	// Memory should be controlled
	assert.Less(t, memory, uint64(500*1024*1024), "Memory usage too high")
	
	suite.Logger().Info("Resource usage after high cardinality test",
		zap.Float64("cpu_percent", cpu),
		zap.Uint64("memory_bytes", memory),
	)
}

// ExampleSecurityTest demonstrates testing security features
func ExampleSecurityTest(t *testing.T) {
	suite := framework.NewTestSuite(t)
	collector := suite.StartCollector("fixtures/configs/security.yaml")
	
	// Generate logs with secrets
	generator := framework.NewTelemetryGeneratorWithConfig(&framework.TelemetryConfig{
		ServiceName:    "secure-service",
		HostName:       "secure-host",
		IncludeSecrets: true,
	})
	
	logs := generator.GenerateLogs(10)
	
	// Send logs
	err := collector.SendLogs(logs)
	require.NoError(t, err)
	
	// Wait for processing
	time.Sleep(2 * time.Second)
	
	// Verify secrets are redacted
	secretPatterns := []string{"password", "api_key", "token", "secret"}
	framework.AssertSecretsRedacted(t, logs, secretPatterns)
}