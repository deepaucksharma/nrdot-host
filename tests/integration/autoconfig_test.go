package integration

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/newrelic/nrdot-host/nrdot-autoconfig"
	"github.com/newrelic/nrdot-host/nrdot-common/config"
	"github.com/newrelic/nrdot-host/nrdot-discovery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestServiceDiscovery(t *testing.T) {
	logger := zaptest.NewLogger(t)
	discovery := discovery.NewServiceDiscovery(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Run discovery
	services, err := discovery.Discover(ctx)
	require.NoError(t, err)

	// Should find at least some services on a typical system
	t.Logf("Found %d services", len(services))
	for _, svc := range services {
		t.Logf("  - %s (confidence: %s, discovered by: %v)", 
			svc.Type, svc.Confidence, svc.DiscoveredBy)
	}
}

func TestConfigGeneration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	generator := autoconfig.NewConfigGenerator(logger)

	// Create test services
	services := []discovery.ServiceInfo{
		{
			Type: "mysql",
			Endpoints: []discovery.Endpoint{
				{Address: "localhost", Port: 3306, Protocol: "tcp"},
			},
			DiscoveredBy: []string{"port", "process"},
			Confidence:   "HIGH",
		},
		{
			Type: "redis",
			Endpoints: []discovery.Endpoint{
				{Address: "localhost", Port: 6379, Protocol: "tcp"},
			},
			DiscoveredBy: []string{"port"},
			Confidence:   "MEDIUM",
		},
	}

	ctx := context.Background()
	config, err := generator.GenerateConfig(ctx, services)
	require.NoError(t, err)

	assert.NotEmpty(t, config.Config)
	assert.NotEmpty(t, config.Version)
	assert.NotEmpty(t, config.Signature)
	assert.Len(t, config.DiscoveredServices, 2)

	// Check required variables
	assert.Contains(t, config.RequiredVariables, "NEW_RELIC_LICENSE_KEY")
	assert.Contains(t, config.RequiredVariables, "MYSQL_MONITOR_USER")
	assert.Contains(t, config.RequiredVariables, "MYSQL_MONITOR_PASS")

	t.Logf("Generated config version: %s", config.Version)
	t.Logf("Required variables: %v", config.RequiredVariables)
}

func TestConfigSignatureVerification(t *testing.T) {
	logger := zaptest.NewLogger(t)
	signer := autoconfig.NewConfigSigner(logger)

	// Test data
	data := []byte("test configuration data")

	// Sign
	signature, err := signer.Sign(data)
	require.NoError(t, err)
	assert.NotEmpty(t, signature)

	// Get public key
	publicKeyPEM := signer.GetPublicKeyPEM()
	assert.NotEmpty(t, publicKeyPEM)

	// Verify
	err = autoconfig.VerifySignature(data, signature, publicKeyPEM)
	assert.NoError(t, err)

	// Verify with tampered data should fail
	tamperedData := []byte("tampered configuration data")
	err = autoconfig.VerifySignature(tamperedData, signature, publicKeyPEM)
	assert.Error(t, err)
}

func TestAutoConfigOrchestrator(t *testing.T) {
	// Skip if not running as root (required for some discovery methods)
	if os.Geteuid() != 0 {
		t.Skip("Skipping test that requires root privileges")
	}

	logger := zaptest.NewLogger(t)
	
	// Create temp directory
	tempDir, err := ioutil.TempDir("", "nrdot-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test config
	cfg := &config.Config{
		LicenseKey: "test-license-key",
		DataDir:    tempDir,
		ConfigPath: filepath.Join(tempDir, "config.yaml"),
		AutoConfig: config.AutoConfigSettings{
			Enabled:      true,
			ScanInterval: 10 * time.Second,
		},
	}

	// Create mock supervisor
	supervisor := &mockSupervisor{
		reloadFunc: func(configPath string) error {
			// Verify config file exists
			_, err := os.Stat(configPath)
			return err
		},
	}

	// Create orchestrator
	orchestrator := autoconfig.NewAutoConfigOrchestrator(logger, cfg, supervisor)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start orchestrator
	err = orchestrator.Start(ctx)
	require.NoError(t, err)

	// Give it time to run initial discovery
	time.Sleep(2 * time.Second)

	// Get status
	status := orchestrator.GetStatus()
	assert.True(t, status.Enabled)
	assert.NotNil(t, status.LastScan)

	// Stop
	orchestrator.Stop()
}

func TestProcessTelemetry(t *testing.T) {
	logger := zaptest.NewLogger(t)
	collector := process.NewProcessCollector(logger, "/proc", 10, time.Second)

	ctx := context.Background()
	processes, err := collector.Collect(ctx)
	require.NoError(t, err)

	// Should find at least init process
	assert.NotEmpty(t, processes)

	// Check first process has required fields
	if len(processes) > 0 {
		p := processes[0]
		assert.NotZero(t, p.PID)
		assert.NotEmpty(t, p.Name)
		assert.NotEmpty(t, p.User)
		assert.NotZero(t, p.MemoryRSS)

		t.Logf("Top process: PID=%d, Name=%s, CPU=%.2f%%, RSS=%d", 
			p.PID, p.Name, p.CPUPercent, p.MemoryRSS)
	}

	// Test service detection
	detector := process.NewServiceDetector()
	for _, p := range processes {
		if service, confidence := detector.DetectService(p); service != "" {
			t.Logf("Detected service: %s (confidence: %s) from process %s", 
				service, confidence, p.Name)
		}
	}
}

func TestProcessMetricsConversion(t *testing.T) {
	logger := zaptest.NewLogger(t)
	collector := process.NewProcessCollector(logger, "/proc", 5, time.Second)

	ctx := context.Background()
	processes, err := collector.Collect(ctx)
	require.NoError(t, err)

	// Convert to metrics
	metrics := collector.ConvertToMetrics(processes)
	assert.NotNil(t, metrics)

	// Verify metrics structure
	resourceMetrics := metrics.ResourceMetrics()
	assert.Greater(t, resourceMetrics.Len(), 0)

	if resourceMetrics.Len() > 0 {
		scopeMetrics := resourceMetrics.At(0).ScopeMetrics()
		assert.Greater(t, scopeMetrics.Len(), 0)

		if scopeMetrics.Len() > 0 {
			scope := scopeMetrics.At(0).Scope()
			assert.Equal(t, "nrdot.process", scope.Name())
			assert.Equal(t, "1.0.0", scope.Version())

			// Check metrics
			metrics := scopeMetrics.At(0).Metrics()
			assert.Greater(t, metrics.Len(), 0)

			// Verify metric names
			metricNames := make(map[string]bool)
			for i := 0; i < metrics.Len(); i++ {
				metric := metrics.At(i)
				metricNames[metric.Name()] = true
			}

			assert.Contains(t, metricNames, "process.cpu.percent")
			assert.Contains(t, metricNames, "process.memory.rss")
			assert.Contains(t, metricNames, "process.threads")
		}
	}
}

// Mock supervisor for testing
type mockSupervisor struct {
	reloadFunc func(string) error
}

func (m *mockSupervisor) ReloadConfig(configPath string) error {
	if m.reloadFunc != nil {
		return m.reloadFunc(configPath)
	}
	return nil
}

func (m *mockSupervisor) Start(ctx context.Context) error {
	return nil
}

func (m *mockSupervisor) Stop(ctx context.Context) error {
	return nil
}