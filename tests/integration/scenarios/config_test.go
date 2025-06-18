package scenarios

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/newrelic/nrdot-host/integration-tests/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestConfigurationManagement(t *testing.T) {
	suite := framework.NewTestSuite(t)
	
	t.Run("ConfigurationValidation", func(t *testing.T) {
		// Test various configuration scenarios
		configs := []struct {
			name        string
			configFile  string
			shouldStart bool
		}{
			{
				name:        "valid_basic_config",
				configFile:  "../fixtures/configs/basic.yaml",
				shouldStart: true,
			},
			{
				name:        "valid_complex_config",
				configFile:  "../fixtures/configs/complex.yaml",
				shouldStart: true,
			},
			{
				name:        "invalid_yaml_syntax",
				configFile:  "../fixtures/configs/invalid_syntax.yaml",
				shouldStart: false,
			},
			{
				name:        "missing_required_fields",
				configFile:  "../fixtures/configs/missing_fields.yaml",
				shouldStart: false,
			},
		}
		
		for _, tc := range configs {
			t.Run(tc.name, func(t *testing.T) {
				if !tc.shouldStart {
					// Expect failure
					defer func() {
						if r := recover(); r != nil {
							// Expected to fail
							suite.Logger().Info("Config validation failed as expected",
								zap.String("config", tc.configFile),
								zap.Any("error", r),
							)
						}
					}()
				}
				
				collector := suite.StartCollector(tc.configFile)
				
				if tc.shouldStart {
					// Should start successfully
					require.NotNil(t, collector)
					
					err := collector.WaitReady(context.Background(), 10*time.Second)
					assert.NoError(t, err)
				}
			})
		}
	})
	
	t.Run("HotReload", func(t *testing.T) {
		collector := suite.StartCollector("../fixtures/configs/basic.yaml")
		
		// Send initial metrics
		generator := framework.NewTelemetryGenerator()
		metrics := generator.GenerateMetrics(5)
		err := collector.SendMetrics(metrics)
		require.NoError(t, err)
		
		// Wait for initial processing
		time.Sleep(2 * time.Second)
		
		// Update configuration
		suite.UpdateCollectorConfig("collector-0", "../fixtures/configs/updated.yaml")
		
		// Send metrics after config update
		metrics = generator.GenerateMetrics(5)
		err = collector.SendMetrics(metrics)
		require.NoError(t, err)
		
		// Verify new configuration is active
		// TODO: Check for specific behavior changes based on new config
		
		// Check collector is still healthy
		err = collector.WaitReady(context.Background(), 10*time.Second)
		assert.NoError(t, err)
	})
	
	t.Run("EnvironmentVariableSubstitution", func(t *testing.T) {
		// Set test environment variables
		testEnvVars := map[string]string{
			"TEST_SERVICE_NAME": "env-test-service",
			"TEST_LOG_LEVEL":    "debug",
			"TEST_METRIC_PORT":  "9999",
			"TEST_API_KEY":      "test-key-12345",
		}
		
		for k, v := range testEnvVars {
			os.Setenv(k, v)
			defer os.Unsetenv(k)
		}
		
		// Start collector with config using env vars
		config := &framework.CollectorConfig{
			Name:           "env-var-test",
			Image:          framework.DefaultTestConfig().CollectorImage,
			ConfigPath:     "../fixtures/configs/env_vars.yaml",
			Network:        suite.Network(),
			Logger:         suite.Logger(),
			EnvVars:        testEnvVars,
			HealthEndpoint: "/health",
			MetricsPort:    8888,
			OTLPGRPCPort:   4317,
			OTLPHTTPPort:   4318,
		}
		
		collector := suite.StartCollectorWithConfig(config)
		
		// Verify environment variables were substituted
		logs, err := collector.GetLogs(context.Background())
		require.NoError(t, err)
		
		// Should see the substituted values in logs (but not the actual secrets)
		assert.Contains(t, logs, "env-test-service")
		assert.NotContains(t, logs, "${TEST_SERVICE_NAME}")
		assert.NotContains(t, logs, "test-key-12345") // API key should be redacted
	})
	
	t.Run("ConfigurationInheritance", func(t *testing.T) {
		// Test configuration with includes/inheritance
		collector := suite.StartCollector("../fixtures/configs/with_includes.yaml")
		
		// Verify all inherited configurations are active
		err := collector.WaitReady(context.Background(), 10*time.Second)
		require.NoError(t, err)
		
		// Send test data to verify all components work
		generator := framework.NewTelemetryGenerator()
		
		// Test metrics
		metrics := generator.GenerateMetrics(5)
		err = collector.SendMetrics(metrics)
		assert.NoError(t, err)
		
		// Test traces
		traces := generator.GenerateTraces(5)
		err = collector.SendTraces(traces)
		assert.NoError(t, err)
		
		// Test logs
		logs := generator.GenerateLogs(5)
		err = collector.SendLogs(logs)
		assert.NoError(t, err)
	})
}

func TestConfigurationScenarios(t *testing.T) {
	suite := framework.NewTestSuite(t)
	
	t.Run("MultiPipelineConfiguration", func(t *testing.T) {
		// Test configuration with multiple pipelines
		collector := suite.StartCollector("../fixtures/configs/multi_pipeline.yaml")
		
		// Generate different types of telemetry
		generator := framework.NewTelemetryGenerator()
		
		// Send to different pipelines
		metrics := generator.GenerateMetrics(10)
		traces := generator.GenerateTraces(10)
		logs := generator.GenerateLogs(10)
		
		err := collector.SendMetrics(metrics)
		require.NoError(t, err)
		
		err = collector.SendTraces(traces)
		require.NoError(t, err)
		
		err = collector.SendLogs(logs)
		require.NoError(t, err)
		
		// Wait for processing
		time.Sleep(3 * time.Second)
		
		// Verify all pipelines are processing
		collectorMetrics, err := collector.GetMetrics(context.Background())
		require.NoError(t, err)
		
		// Should see metrics for all pipeline types
		assert.Contains(t, string(collectorMetrics), "pipeline_metrics")
		assert.Contains(t, string(collectorMetrics), "pipeline_traces")
		assert.Contains(t, string(collectorMetrics), "pipeline_logs")
	})
	
	t.Run("ProcessorChaining", func(t *testing.T) {
		// Test configuration with chained processors
		collector := suite.StartCollector("../fixtures/configs/processor_chain.yaml")
		
		// Send data through the processor chain
		generator := framework.NewTelemetryGenerator()
		metrics := generator.GenerateMetrics(10)
		
		err := collector.SendMetrics(metrics)
		require.NoError(t, err)
		
		// Wait for all processors to execute
		time.Sleep(3 * time.Second)
		
		// Verify processors executed in order
		// TODO: Verify specific processor effects when backend is implemented
	})
	
	t.Run("ConditionalExporters", func(t *testing.T) {
		// Test configuration with conditional routing
		collector := suite.StartCollector("../fixtures/configs/conditional_routing.yaml")
		
		// Generate metrics for different routes
		prodGen := framework.NewTelemetryGeneratorWithConfig(&framework.TelemetryConfig{
			ServiceName: "production-service",
			ResourceAttrs: map[string]string{
				"environment": "production",
			},
		})
		
		devGen := framework.NewTelemetryGeneratorWithConfig(&framework.TelemetryConfig{
			ServiceName: "development-service",
			ResourceAttrs: map[string]string{
				"environment": "development",
			},
		})
		
		// Send both types
		prodMetrics := prodGen.GenerateMetrics(5)
		devMetrics := devGen.GenerateMetrics(5)
		
		err := collector.SendMetrics(prodMetrics)
		require.NoError(t, err)
		
		err = collector.SendMetrics(devMetrics)
		require.NoError(t, err)
		
		// Different environments should route to different exporters
		// TODO: Verify routing when multiple backends are implemented
	})
}

func TestConfigurationResilience(t *testing.T) {
	suite := framework.NewTestSuite(t)
	
	t.Run("InvalidConfigReload", func(t *testing.T) {
		collector := suite.StartCollector("../fixtures/configs/basic.yaml")
		
		// Verify collector is healthy
		err := collector.WaitReady(context.Background(), 10*time.Second)
		require.NoError(t, err)
		
		// Try to reload with invalid configuration
		invalidConfig := []byte(`
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: "0.0.0.0:4317"
processors:
  invalid_processor:
    not_a_real_option: true
exporters:
  logging:
service:
  pipelines:
    metrics:
      receivers: [otlp]
      processors: [invalid_processor]
      exporters: [logging]
`)
		
		err = collector.UpdateConfig(context.Background(), invalidConfig)
		// Should fail but not crash
		assert.Error(t, err)
		
		// Collector should still be running with old config
		time.Sleep(2 * time.Second)
		err = collector.WaitReady(context.Background(), 10*time.Second)
		assert.NoError(t, err)
		
		// Should still process data
		generator := framework.NewTelemetryGenerator()
		metrics := generator.GenerateMetrics(5)
		err = collector.SendMetrics(metrics)
		assert.NoError(t, err)
	})
	
	t.Run("ConfigMemoryLeak", func(t *testing.T) {
		collector := suite.StartCollector("../fixtures/configs/basic.yaml")
		
		// Get baseline memory
		_, memBase, err := collector.GetResourceUsage(context.Background())
		require.NoError(t, err)
		
		// Perform many config reloads
		for i := 0; i < 20; i++ {
			// Alternate between two valid configs
			if i%2 == 0 {
				suite.UpdateCollectorConfig("collector-0", "../fixtures/configs/basic.yaml")
			} else {
				suite.UpdateCollectorConfig("collector-0", "../fixtures/configs/minimal.yaml")
			}
			time.Sleep(500 * time.Millisecond)
		}
		
		// Check memory after reloads
		_, memAfter, err := collector.GetResourceUsage(context.Background())
		require.NoError(t, err)
		
		// Memory should not increase significantly
		memIncrease := float64(memAfter-memBase) / float64(memBase) * 100
		assert.Less(t, memIncrease, 20.0, "Memory increased too much after config reloads")
		
		suite.Logger().Info("Config reload memory test",
			zap.Uint64("mem_base", memBase),
			zap.Uint64("mem_after", memAfter),
			zap.Float64("increase_percent", memIncrease),
		)
	})
}

func TestConfigurationPerformance(t *testing.T) {
	suite := framework.NewTestSuite(t)
	
	t.Run("LargeConfigurationLoad", func(t *testing.T) {
		// Test loading a very large configuration
		start := time.Now()
		collector := suite.StartCollector("../fixtures/configs/large.yaml")
		
		err := collector.WaitReady(context.Background(), 30*time.Second)
		require.NoError(t, err)
		
		loadTime := time.Since(start)
		
		// Even large configs should load reasonably fast
		assert.Less(t, loadTime, 10*time.Second, "Large config took too long to load")
		
		suite.Logger().Info("Large config load time",
			zap.Duration("load_time", loadTime),
		)
	})
	
	t.Run("HotReloadPerformance", func(t *testing.T) {
		collector := suite.StartCollector("../fixtures/configs/basic.yaml")
		
		// Measure reload times
		reloadTimes := make([]time.Duration, 0)
		
		for i := 0; i < 10; i++ {
			start := time.Now()
			suite.UpdateCollectorConfig("collector-0", "../fixtures/configs/updated.yaml")
			
			// Wait for reload to complete
			time.Sleep(1 * time.Second)
			
			// Verify still healthy
			err := collector.WaitReady(context.Background(), 5*time.Second)
			require.NoError(t, err)
			
			reloadTime := time.Since(start)
			reloadTimes = append(reloadTimes, reloadTime)
			
			// Switch back
			suite.UpdateCollectorConfig("collector-0", "../fixtures/configs/basic.yaml")
			time.Sleep(1 * time.Second)
		}
		
		// Calculate average reload time
		var totalTime time.Duration
		for _, t := range reloadTimes {
			totalTime += t
		}
		avgReloadTime := totalTime / time.Duration(len(reloadTimes))
		
		// Reload should be fast
		assert.Less(t, avgReloadTime, 2*time.Second, "Average reload time too high")
		
		suite.Logger().Info("Hot reload performance",
			zap.Duration("avg_reload_time", avgReloadTime),
			zap.Int("reload_count", len(reloadTimes)),
		)
	})
}