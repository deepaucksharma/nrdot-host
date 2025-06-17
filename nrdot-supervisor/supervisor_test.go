package supervisor

import (
	"context"
	"testing"
	"time"

	"github.com/newrelic/nrdot-host/nrdot-supervisor/pkg/restart"
	telemetryclient "github.com/newrelic/nrdot-host/nrdot-telemetry-client"
	"go.uber.org/zap/zaptest"
)

func TestSupervisor_StartStop(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := Config{
		Collector:     DefaultCollectorConfig(),
		HealthChecker: DefaultHealthCheckerConfig(),
		Restart:       restart.DefaultConfig(),
		Telemetry:     telemetryclient.Config{
			Enabled:  false, // Disable telemetry for tests
			Interval: 10 * time.Second,
		},
	}

	// Use a non-existent binary to prevent actual process start
	config.Collector.BinaryPath = "/bin/false"
	config.Restart.Policy = restart.PolicyNever
	// Disable health checking for this test
	config.HealthChecker.Interval = 1 * time.Hour

	supervisor, err := New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create supervisor: %v", err)
	}

	ctx := context.Background()

	// Start supervisor
	err = supervisor.Start(ctx)
	if err != nil {
		// Expected to fail due to non-existent binary
		t.Logf("Expected start failure: %v", err)
	}

	// Give supervisor a moment to start
	time.Sleep(100 * time.Millisecond)

	// Stop supervisor
	stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	err = supervisor.Stop(stopCtx)
	if err != nil {
		t.Errorf("Failed to stop supervisor: %v", err)
	}
}

func TestSupervisor_MultipleStart(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	config.Collector.BinaryPath = "/bin/false"
	config.Restart.Policy = restart.PolicyNever
	config.HealthChecker.Interval = 1 * time.Hour
	config.Telemetry.Enabled = false

	supervisor, err := New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create supervisor: %v", err)
	}

	ctx := context.Background()

	// First start
	_ = supervisor.Start(ctx)
	
	// Give it a moment
	time.Sleep(100 * time.Millisecond)

	// Second start should fail
	err = supervisor.Start(ctx)
	if err == nil {
		t.Error("Expected error on second start, got nil")
	}

	supervisor.Stop(ctx)
}

func TestSupervisor_ReportMetric(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	config.Telemetry.Enabled = false

	supervisor, err := New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create supervisor: %v", err)
	}

	// Test metric reporting
	supervisor.reportMetric("test.metric", 42.0, map[string]string{"test": "tag"})
	
	// No assertion needed, just ensure no panic
}

func TestBuildConfig(t *testing.T) {
	config := DefaultConfig()

	// Verify default values
	if config.Collector.BinaryPath != "otelcol" {
		t.Errorf("Expected default binary path 'otelcol', got %s", config.Collector.BinaryPath)
	}

	if config.HealthChecker.Interval != 10*time.Second {
		t.Errorf("Expected default health interval 10s, got %v", config.HealthChecker.Interval)
	}

	if config.Restart.Policy != restart.PolicyOnFailure {
		t.Errorf("Expected default restart policy 'on-failure', got %v", config.Restart.Policy)
	}
}