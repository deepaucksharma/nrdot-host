package scenarios

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/newrelic/nrdot-host/integration-tests/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestBasicFunctionality(t *testing.T) {
	suite := framework.NewTestSuite(t)
	
	t.Run("CollectorStartup", func(t *testing.T) {
		// Start collector with basic configuration
		collector := suite.StartCollector("../fixtures/configs/basic.yaml")
		require.NotNil(t, collector)
		
		// Verify collector is healthy
		ctx := context.Background()
		err := collector.WaitReady(ctx, 30*time.Second)
		assert.NoError(t, err)
		
		// Check metrics endpoint
		metrics, err := collector.GetMetrics(ctx)
		assert.NoError(t, err)
		assert.Contains(t, string(metrics), "otelcol_process_uptime")
	})
	
	t.Run("MetricReception", func(t *testing.T) {
		collector := suite.StartCollector("../fixtures/configs/basic.yaml")
		
		// Generate test metrics
		generator := framework.NewTelemetryGenerator()
		metrics := generator.GenerateMetrics(10)
		
		// Send metrics to collector
		err := collector.SendMetrics(metrics)
		require.NoError(t, err)
		
		// Wait for processing
		time.Sleep(2 * time.Second)
		
		// Verify metrics were received (check collector metrics)
		collectorMetrics, err := collector.GetMetrics(context.Background())
		require.NoError(t, err)
		assert.Contains(t, string(collectorMetrics), "otelcol_receiver_accepted_metric_points")
	})
	
	t.Run("TraceReception", func(t *testing.T) {
		collector := suite.StartCollector("../fixtures/configs/basic.yaml")
		
		// Generate test traces
		generator := framework.NewTelemetryGenerator()
		traces := generator.GenerateTraces(5)
		
		// Send traces to collector
		err := collector.SendTraces(traces)
		require.NoError(t, err)
		
		// Wait for processing
		time.Sleep(2 * time.Second)
		
		// Verify traces were received
		collectorMetrics, err := collector.GetMetrics(context.Background())
		require.NoError(t, err)
		assert.Contains(t, string(collectorMetrics), "otelcol_receiver_accepted_spans")
	})
	
	t.Run("LogReception", func(t *testing.T) {
		collector := suite.StartCollector("../fixtures/configs/basic.yaml")
		
		// Generate test logs
		generator := framework.NewTelemetryGenerator()
		logs := generator.GenerateLogs(10)
		
		// Send logs to collector
		err := collector.SendLogs(logs)
		require.NoError(t, err)
		
		// Wait for processing
		time.Sleep(2 * time.Second)
		
		// Verify logs were received
		collectorMetrics, err := collector.GetMetrics(context.Background())
		require.NoError(t, err)
		assert.Contains(t, string(collectorMetrics), "otelcol_receiver_accepted_log_records")
	})
	
	t.Run("CollectorRestart", func(t *testing.T) {
		collector := suite.StartCollector("../fixtures/configs/basic.yaml")
		
		// Send initial metrics
		generator := framework.NewTelemetryGenerator()
		metrics := generator.GenerateMetrics(5)
		err := collector.SendMetrics(metrics)
		require.NoError(t, err)
		
		// Restart collector
		err = collector.Restart(context.Background())
		require.NoError(t, err)
		
		// Wait for collector to be ready
		err = collector.WaitReady(context.Background(), 30*time.Second)
		require.NoError(t, err)
		
		// Send metrics after restart
		metrics = generator.GenerateMetrics(5)
		err = collector.SendMetrics(metrics)
		require.NoError(t, err)
		
		// Verify collector is functioning after restart
		collectorMetrics, err := collector.GetMetrics(context.Background())
		require.NoError(t, err)
		assert.Contains(t, string(collectorMetrics), "otelcol_process_uptime")
	})
	
	t.Run("MultipleCollectors", func(t *testing.T) {
		// Start multiple collectors
		collector1 := suite.StartCollector("../fixtures/configs/basic.yaml")
		collector2 := suite.StartCollector("../fixtures/configs/basic.yaml")
		
		require.NotNil(t, collector1)
		require.NotNil(t, collector2)
		
		// Send metrics to both
		generator := framework.NewTelemetryGenerator()
		metrics := generator.GenerateMetrics(5)
		
		err := collector1.SendMetrics(metrics)
		require.NoError(t, err)
		
		err = collector2.SendMetrics(metrics)
		require.NoError(t, err)
		
		// Verify both are functioning
		metrics1, err := collector1.GetMetrics(context.Background())
		require.NoError(t, err)
		assert.Contains(t, string(metrics1), "otelcol_receiver_accepted_metric_points")
		
		metrics2, err := collector2.GetMetrics(context.Background())
		require.NoError(t, err)
		assert.Contains(t, string(metrics2), "otelcol_receiver_accepted_metric_points")
	})
}

func TestHealthChecks(t *testing.T) {
	suite := framework.NewTestSuite(t)
	
	t.Run("HealthEndpoint", func(t *testing.T) {
		collector := suite.StartCollector("../fixtures/configs/basic.yaml")
		
		// Check health endpoint returns OK
		err := collector.WaitReady(context.Background(), 10*time.Second)
		assert.NoError(t, err)
	})
	
	t.Run("LivenessCheck", func(t *testing.T) {
		collector := suite.StartCollector("../fixtures/configs/basic.yaml")
		
		// Continuously check liveness over time
		for i := 0; i < 5; i++ {
			err := collector.WaitReady(context.Background(), 5*time.Second)
			assert.NoError(t, err)
			time.Sleep(1 * time.Second)
		}
	})
}

func TestResourceUsage(t *testing.T) {
	suite := framework.NewTestSuite(t)
	
	t.Run("MemoryUsage", func(t *testing.T) {
		collector := suite.StartCollector("../fixtures/configs/basic.yaml")
		
		// Get baseline memory usage
		cpu1, mem1, err := collector.GetResourceUsage(context.Background())
		require.NoError(t, err)
		
		// Send large amount of data
		generator := framework.NewTelemetryGenerator()
		for i := 0; i < 10; i++ {
			metrics := generator.GenerateMetrics(100)
			err := collector.SendMetrics(metrics)
			require.NoError(t, err)
		}
		
		// Wait for processing
		time.Sleep(5 * time.Second)
		
		// Check memory usage increase
		cpu2, mem2, err := collector.GetResourceUsage(context.Background())
		require.NoError(t, err)
		
		// Memory should increase but stay within reasonable bounds
		memIncrease := float64(mem2-mem1) / float64(mem1) * 100
		assert.Less(t, memIncrease, 50.0, "Memory increased by more than 50%")
		
		suite.Logger().Info("Resource usage",
			zap.Float64("cpu_before", cpu1),
			zap.Float64("cpu_after", cpu2),
			zap.Uint64("mem_before", mem1),
			zap.Uint64("mem_after", mem2),
		)
	})
}

func TestErrorHandling(t *testing.T) {
	suite := framework.NewTestSuite(t)
	
	t.Run("InvalidConfiguration", func(t *testing.T) {
		// This should fail to start
		defer func() {
			if r := recover(); r != nil {
				// Expected to panic/fail
				assert.Contains(t, fmt.Sprint(r), "failed to create container")
			}
		}()
		
		// Try to start with non-existent config
		suite.StartCollector("../fixtures/configs/invalid.yaml")
	})
	
	t.Run("ConnectionFailure", func(t *testing.T) {
		collector := suite.StartCollector("../fixtures/configs/basic.yaml")
		
		// Stop the collector
		err := collector.Stop(context.Background())
		require.NoError(t, err)
		
		// Try to send metrics (should fail)
		generator := framework.NewTelemetryGenerator()
		metrics := generator.GenerateMetrics(5)
		err = collector.SendMetrics(metrics)
		assert.Error(t, err)
	})
}