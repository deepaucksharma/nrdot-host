package scenarios

import (
	"context"
	"testing"
	"time"

	"github.com/newrelic/nrdot-host/integration-tests/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"
)

func TestMetadataEnrichment(t *testing.T) {
	suite := framework.NewTestSuite(t)
	
	t.Run("HostMetadataEnrichment", func(t *testing.T) {
		// Start collector with enrichment configuration
		collector := suite.StartCollector("../fixtures/configs/enrichment.yaml")
		
		// Start a mock backend to receive enriched data
		backend := suite.StartMockBackend()
		
		// Generate metrics without host metadata
		generator := framework.NewTelemetryGenerator()
		metrics := generator.GenerateMetrics(10)
		
		// Send metrics to collector
		err := collector.SendMetrics(metrics)
		require.NoError(t, err)
		
		// Wait for processing
		time.Sleep(2 * time.Second)
		
		// Get received metrics from backend
		received := backend.GetReceived()
		require.Greater(t, len(received), 0)
		
		// Verify host metadata was added
		// Note: This is a simplified check - in real implementation,
		// we would parse the actual metrics and check attributes
		expectedHostAttrs := []string{
			"host.name",
			"host.id",
			"host.arch",
			"host.type",
			"os.type",
			"os.description",
		}
		
		// Check collector metrics for enrichment activity
		collectorMetrics, err := collector.GetMetrics(context.Background())
		require.NoError(t, err)
		
		// Verify enrichment processor metrics
		assert.Contains(t, string(collectorMetrics), "processor_enrichment")
		
		// TODO: When backend implementation is complete, verify actual attributes
		_ = expectedHostAttrs
	})
	
	t.Run("ContainerMetadataEnrichment", func(t *testing.T) {
		// This test requires the collector to be running in a container
		collector := suite.StartCollector("../fixtures/configs/enrichment.yaml")
		
		// Generate metrics from a containerized source
		generator := framework.NewTelemetryGeneratorWithConfig(&framework.TelemetryConfig{
			ServiceName: "containerized-service",
			HostName:    "container-host",
			ResourceAttrs: map[string]string{
				"container.name": "test-container",
				"container.id":   "abc123",
			},
		})
		
		metrics := generator.GenerateMetrics(5)
		
		// Send metrics
		err := collector.SendMetrics(metrics)
		require.NoError(t, err)
		
		// Wait for processing
		time.Sleep(2 * time.Second)
		
		// Expected container attributes after enrichment
		expectedContainerAttrs := []string{
			"container.name",
			"container.id",
			"container.runtime",
			"container.image.name",
			"container.image.tag",
		}
		
		// TODO: Verify actual enrichment when backend is implemented
		_ = expectedContainerAttrs
	})
	
	t.Run("KubernetesMetadataEnrichment", func(t *testing.T) {
		t.Skip("Requires Kubernetes environment")
		
		// This test would verify K8s metadata enrichment
		// Including pod, deployment, namespace, node info
	})
	
	t.Run("CustomAttributeEnrichment", func(t *testing.T) {
		collector := suite.StartCollector("../fixtures/configs/enrichment.yaml")
		backend := suite.StartMockBackend()
		
		// Generate metrics
		generator := framework.NewTelemetryGenerator()
		metrics := generator.GenerateMetrics(5)
		
		// Send metrics
		err := collector.SendMetrics(metrics)
		require.NoError(t, err)
		
		// Wait for processing
		time.Sleep(2 * time.Second)
		
		// Custom attributes that should be added by enrichment
		expectedCustomAttrs := map[string]string{
			"environment":     "test",
			"region":          "us-east-1",
			"deployment.type": "integration-test",
			"team":            "platform",
		}
		
		// TODO: Verify custom attributes when backend is implemented
		_ = expectedCustomAttrs
		_ = backend
	})
}

func TestEnrichmentScenarios(t *testing.T) {
	suite := framework.NewTestSuite(t)
	
	t.Run("ConditionalEnrichment", func(t *testing.T) {
		collector := suite.StartCollector("../fixtures/configs/enrichment.yaml")
		
		// Generate metrics with different service names
		generator1 := framework.NewTelemetryGeneratorWithConfig(&framework.TelemetryConfig{
			ServiceName: "frontend-service",
			HostName:    "web-host",
		})
		
		generator2 := framework.NewTelemetryGeneratorWithConfig(&framework.TelemetryConfig{
			ServiceName: "backend-service",
			HostName:    "api-host",
		})
		
		// Send metrics from both services
		metrics1 := generator1.GenerateMetrics(5)
		metrics2 := generator2.GenerateMetrics(5)
		
		err := collector.SendMetrics(metrics1)
		require.NoError(t, err)
		
		err = collector.SendMetrics(metrics2)
		require.NoError(t, err)
		
		// Wait for processing
		time.Sleep(2 * time.Second)
		
		// Different services should get different enrichment
		// Frontend should get "tier=web"
		// Backend should get "tier=api"
		// TODO: Verify conditional enrichment when backend is implemented
	})
	
	t.Run("EnrichmentFromExternalSource", func(t *testing.T) {
		t.Skip("Requires external metadata service")
		
		// This test would verify enrichment from external sources
		// like AWS metadata service, GCP metadata, etc.
	})
	
	t.Run("EnrichmentPerformance", func(t *testing.T) {
		collector := suite.StartCollector("../fixtures/configs/enrichment.yaml")
		
		// Get baseline performance
		cpuBefore, memBefore, err := collector.GetResourceUsage(context.Background())
		require.NoError(t, err)
		
		// Generate high volume of metrics
		generator := framework.NewTelemetryGenerator()
		
		start := time.Now()
		for i := 0; i < 100; i++ {
			metrics := generator.GenerateMetrics(100)
			err := collector.SendMetrics(metrics)
			require.NoError(t, err)
		}
		duration := time.Since(start)
		
		// Wait for processing
		time.Sleep(5 * time.Second)
		
		// Check resource usage
		cpuAfter, memAfter, err := collector.GetResourceUsage(context.Background())
		require.NoError(t, err)
		
		// Enrichment should not significantly impact performance
		assert.Less(t, duration, 30*time.Second)
		
		memIncrease := float64(memAfter-memBefore) / float64(memBefore) * 100
		assert.Less(t, memIncrease, 50.0, "Memory increased by more than 50%")
		
		suite.Logger().Info("Enrichment performance",
			zap.Duration("duration", duration),
			zap.Float64("cpu_before", cpuBefore),
			zap.Float64("cpu_after", cpuAfter),
			zap.Uint64("mem_before", memBefore),
			zap.Uint64("mem_after", memAfter),
		)
	})
}

func TestEnrichmentValidation(t *testing.T) {
	suite := framework.NewTestSuite(t)
	
	t.Run("PreserveOriginalAttributes", func(t *testing.T) {
		collector := suite.StartCollector("../fixtures/configs/enrichment.yaml")
		
		// Generate metrics with specific attributes
		originalAttrs := map[string]string{
			"custom.attr1": "value1",
			"custom.attr2": "value2",
			"service.name": "original-service",
		}
		
		generator := framework.NewTelemetryGeneratorWithConfig(&framework.TelemetryConfig{
			ServiceName:   "test-service",
			HostName:      "test-host",
			ResourceAttrs: originalAttrs,
		})
		
		metrics := generator.GenerateMetrics(5)
		
		// Send metrics
		err := collector.SendMetrics(metrics)
		require.NoError(t, err)
		
		// Wait for processing
		time.Sleep(2 * time.Second)
		
		// Original attributes should be preserved
		// TODO: Verify when backend is implemented
	})
	
	t.Run("AttributeOverrideHandling", func(t *testing.T) {
		collector := suite.StartCollector("../fixtures/configs/enrichment.yaml")
		
		// Generate metrics with attributes that might be overridden
		generator := framework.NewTelemetryGeneratorWithConfig(&framework.TelemetryConfig{
			ServiceName: "test-service",
			HostName:    "original-host",
			ResourceAttrs: map[string]string{
				"host.name":   "should-not-be-overridden",
				"environment": "production", // This might be overridden to "test"
			},
		})
		
		metrics := generator.GenerateMetrics(5)
		
		// Send metrics
		err := collector.SendMetrics(metrics)
		require.NoError(t, err)
		
		// Wait for processing
		time.Sleep(2 * time.Second)
		
		// Verify override behavior based on configuration
		// TODO: Verify when backend is implemented
	})
}

// Helper function to verify attributes in metrics
func verifyMetricAttributes(t *testing.T, metrics interface{}, expectedAttrs map[string]string) {
	// This would be implemented when we have actual metric parsing
	// For now, it's a placeholder
	t.Helper()
	
	// Parse metrics and check each resource for expected attributes
	// ...
}

// Helper function to check if attributes exist
func hasAttributes(attrs pcommon.Map, keys []string) bool {
	for _, key := range keys {
		if _, exists := attrs.Get(key); !exists {
			return false
		}
	}
	return true
}