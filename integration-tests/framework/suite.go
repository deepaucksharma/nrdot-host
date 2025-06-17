package framework

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// TestSuite provides a comprehensive test environment for integration tests
type TestSuite struct {
	t          *testing.T
	ctx        context.Context
	cancel     context.CancelFunc
	collectors map[string]*CollectorInstance
	mu         sync.Mutex
	logger     *zap.Logger
	network    testcontainers.Network
	cleanup    []func()
}

// TestConfig contains configuration for a test suite
type TestConfig struct {
	CollectorImage   string
	Timeout          time.Duration
	EnableDebugLogs  bool
	PreserveOnFailure bool
	NetworkName      string
}

// DefaultTestConfig returns default test configuration
func DefaultTestConfig() *TestConfig {
	return &TestConfig{
		CollectorImage:    getEnvOrDefault("COLLECTOR_IMAGE", "nrdot-host:latest"),
		Timeout:           5 * time.Minute,
		EnableDebugLogs:   os.Getenv("TEST_LOG_LEVEL") == "debug",
		PreserveOnFailure: os.Getenv("PRESERVE_ON_FAILURE") == "true",
		NetworkName:       "nrdot-test-network",
	}
}

// NewTestSuite creates a new test suite with default configuration
func NewTestSuite(t *testing.T) *TestSuite {
	return NewTestSuiteWithConfig(t, DefaultTestConfig())
}

// NewTestSuiteWithConfig creates a new test suite with custom configuration
func NewTestSuiteWithConfig(t *testing.T, config *TestConfig) *TestSuite {
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	
	// Setup logger
	logConfig := zap.NewDevelopmentConfig()
	if config.EnableDebugLogs {
		logConfig.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	} else {
		logConfig.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}
	logger, err := logConfig.Build()
	require.NoError(t, err)

	// Create test network
	network, err := testcontainers.GenericNetwork(ctx, testcontainers.GenericNetworkRequest{
		NetworkRequest: testcontainers.NetworkRequest{
			Name:           config.NetworkName,
			CheckDuplicate: true,
		},
	})
	require.NoError(t, err)

	suite := &TestSuite{
		t:          t,
		ctx:        ctx,
		cancel:     cancel,
		collectors: make(map[string]*CollectorInstance),
		logger:     logger,
		network:    network,
		cleanup:    []func(){},
	}

	// Register cleanup
	t.Cleanup(func() {
		if t.Failed() && config.PreserveOnFailure {
			suite.logger.Info("Test failed, preserving environment for debugging")
			suite.logger.Info("Run 'docker ps' to see running containers")
			return
		}
		suite.Cleanup()
	})

	return suite
}

// StartCollector starts a new collector instance with the given configuration
func (s *TestSuite) StartCollector(configPath string) *CollectorInstance {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := fmt.Sprintf("collector-%d", len(s.collectors))
	s.logger.Info("Starting collector", zap.String("name", name), zap.String("config", configPath))

	// Read config file
	configData, err := os.ReadFile(configPath)
	require.NoError(s.t, err)

	// Create temporary config file with test modifications
	tempConfig := s.createTempConfig(configData)
	defer os.Remove(tempConfig)

	// Start collector container
	collector, err := NewCollectorInstance(s.ctx, &CollectorConfig{
		Name:           name,
		Image:          getEnvOrDefault("COLLECTOR_IMAGE", "nrdot-host:latest"),
		ConfigPath:     tempConfig,
		Network:        s.network,
		Logger:         s.logger.With(zap.String("collector", name)),
		HealthEndpoint: "/health",
		MetricsPort:    8888,
		OTLPGRPCPort:   4317,
		OTLPHTTPPort:   4318,
	})
	require.NoError(s.t, err)

	// Wait for collector to be ready
	err = collector.WaitReady(s.ctx, 30*time.Second)
	require.NoError(s.t, err)

	s.collectors[name] = collector
	s.registerCleanup(func() {
		if err := collector.Stop(context.Background()); err != nil {
			s.logger.Error("Failed to stop collector", zap.Error(err))
		}
	})

	return collector
}

// StartCollectorWithConfig starts a collector with a custom configuration
func (s *TestSuite) StartCollectorWithConfig(config *CollectorConfig) *CollectorInstance {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Info("Starting collector with custom config", zap.String("name", config.Name))

	collector, err := NewCollectorInstance(s.ctx, config)
	require.NoError(s.t, err)

	err = collector.WaitReady(s.ctx, 30*time.Second)
	require.NoError(s.t, err)

	s.collectors[config.Name] = collector
	s.registerCleanup(func() {
		if err := collector.Stop(context.Background()); err != nil {
			s.logger.Error("Failed to stop collector", zap.Error(err))
		}
	})

	return collector
}

// GetCollector returns a collector instance by name
func (s *TestSuite) GetCollector(name string) *CollectorInstance {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.collectors[name]
}

// RestartCollector restarts a collector instance
func (s *TestSuite) RestartCollector(name string) *CollectorInstance {
	s.mu.Lock()
	collector := s.collectors[name]
	s.mu.Unlock()

	require.NotNil(s.t, collector, "Collector not found: %s", name)

	// Stop the collector
	err := collector.Stop(s.ctx)
	require.NoError(s.t, err)

	// Start it again
	err = collector.Start(s.ctx)
	require.NoError(s.t, err)

	// Wait for it to be ready
	err = collector.WaitReady(s.ctx, 30*time.Second)
	require.NoError(s.t, err)

	return collector
}

// UpdateCollectorConfig updates the configuration of a running collector
func (s *TestSuite) UpdateCollectorConfig(name string, configPath string) {
	collector := s.GetCollector(name)
	require.NotNil(s.t, collector)

	configData, err := os.ReadFile(configPath)
	require.NoError(s.t, err)

	err = collector.UpdateConfig(s.ctx, configData)
	require.NoError(s.t, err)

	// Wait for config reload
	time.Sleep(2 * time.Second)
}

// WaitForCondition waits for a condition to be true
func (s *TestSuite) WaitForCondition(timeout time.Duration, condition func() bool) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	s.t.Fatal("Timeout waiting for condition")
}

// Context returns the test context
func (s *TestSuite) Context() context.Context {
	return s.ctx
}

// Logger returns the test logger
func (s *TestSuite) Logger() *zap.Logger {
	return s.logger
}

// Network returns the test network
func (s *TestSuite) Network() testcontainers.Network {
	return s.network
}

// Cleanup cleans up all test resources
func (s *TestSuite) Cleanup() {
	s.logger.Info("Cleaning up test suite")

	// Run cleanup functions in reverse order
	for i := len(s.cleanup) - 1; i >= 0; i-- {
		s.cleanup[i]()
	}

	// Remove network
	if s.network != nil {
		if err := s.network.Remove(context.Background()); err != nil {
			s.logger.Error("Failed to remove network", zap.Error(err))
		}
	}

	// Cancel context
	s.cancel()

	// Sync logger
	_ = s.logger.Sync()
}

// registerCleanup registers a cleanup function
func (s *TestSuite) registerCleanup(fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanup = append(s.cleanup, fn)
}

// createTempConfig creates a temporary config file with test modifications
func (s *TestSuite) createTempConfig(configData []byte) string {
	tempFile, err := os.CreateTemp("", "collector-config-*.yaml")
	require.NoError(s.t, err)

	// TODO: Apply test-specific modifications to config
	// For now, just write the original config
	_, err = tempFile.Write(configData)
	require.NoError(s.t, err)

	err = tempFile.Close()
	require.NoError(s.t, err)

	return tempFile.Name()
}

// StartPrometheus starts a Prometheus instance for metric collection
func (s *TestSuite) StartPrometheus() *PrometheusInstance {
	prom, err := NewPrometheusInstance(s.ctx, s.network)
	require.NoError(s.t, err)

	s.registerCleanup(func() {
		if err := prom.Stop(context.Background()); err != nil {
			s.logger.Error("Failed to stop Prometheus", zap.Error(err))
		}
	})

	return prom
}

// StartMockBackend starts a mock backend for testing exporters
func (s *TestSuite) StartMockBackend() *MockBackend {
	backend, err := NewMockBackend(s.ctx, s.network)
	require.NoError(s.t, err)

	s.registerCleanup(func() {
		if err := backend.Stop(context.Background()); err != nil {
			s.logger.Error("Failed to stop mock backend", zap.Error(err))
		}
	})

	return backend
}

// Helper functions

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// PrometheusInstance represents a Prometheus container for testing
type PrometheusInstance struct {
	container testcontainers.Container
	endpoint  string
}

// NewPrometheusInstance creates a new Prometheus instance
func NewPrometheusInstance(ctx context.Context, network testcontainers.Network) (*PrometheusInstance, error) {
	req := testcontainers.ContainerRequest{
		Image:        "prom/prometheus:latest",
		ExposedPorts: []string{"9090/tcp"},
		Networks:     []string{network.(*testcontainers.DockerNetwork).Name},
		WaitingFor:   wait.ForHTTP("/-/ready").WithPort("9090/tcp"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	endpoint, err := container.Endpoint(ctx, "http")
	if err != nil {
		return nil, err
	}

	return &PrometheusInstance{
		container: container,
		endpoint:  endpoint,
	}, nil
}

// Stop stops the Prometheus instance
func (p *PrometheusInstance) Stop(ctx context.Context) error {
	return p.container.Terminate(ctx)
}

// Endpoint returns the Prometheus endpoint
func (p *PrometheusInstance) Endpoint() string {
	return p.endpoint
}

// MockBackend represents a mock backend for testing exporters
type MockBackend struct {
	container testcontainers.Container
	endpoint  string
	received  []interface{}
	mu        sync.Mutex
}

// NewMockBackend creates a new mock backend
func NewMockBackend(ctx context.Context, network testcontainers.Network) (*MockBackend, error) {
	// TODO: Implement mock backend container
	return &MockBackend{
		received: make([]interface{}, 0),
	}, nil
}

// Stop stops the mock backend
func (m *MockBackend) Stop(ctx context.Context) error {
	if m.container != nil {
		return m.container.Terminate(ctx)
	}
	return nil
}

// GetReceived returns all received data
func (m *MockBackend) GetReceived() []interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]interface{}{}, m.received...)
}