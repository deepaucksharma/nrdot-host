package telemetryclient

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewTelemetryClient(t *testing.T) {
	logger := zap.NewNop()

	t.Run("disabled client returns noop", func(t *testing.T) {
		config := Config{
			Enabled: false,
		}
		
		client, err := NewTelemetryClient(config, logger)
		require.NoError(t, err)
		
		_, ok := client.(*noopClient)
		assert.True(t, ok, "Expected noop client when disabled")
	})

	t.Run("enabled client with invalid endpoint fails", func(t *testing.T) {
		config := Config{
			Enabled:        true,
			ServiceName:    "test-service",
			ServiceVersion: "v1.0.0",
			Environment:    "test",
			Endpoint:       "invalid:endpoint:format",
			APIKey:         "test-key",
		}
		
		_, err := NewTelemetryClient(config, logger)
		assert.Error(t, err)
	})
}

func TestNoopClient(t *testing.T) {
	client := &noopClient{}
	
	// All methods should return nil
	assert.NoError(t, client.RecordHealth(HealthSample{}))
	assert.NoError(t, client.RecordRestart("test"))
	assert.NoError(t, client.RecordConfigChange("v1", "v2"))
	assert.NoError(t, client.RecordFeatureFlag("test", true))
	assert.NoError(t, client.RecordMetric("test", 1.0))
	assert.NoError(t, client.IncrementCounter("test"))
	assert.NoError(t, client.RecordDuration("test", time.Second))
	assert.NoError(t, client.Shutdown(context.Background()))
}

func TestHealthSample(t *testing.T) {
	sample := HealthSample{
		Timestamp:    time.Now(),
		CPUPercent:   45.5,
		MemoryMB:     256.7,
		GoroutineNum: 42,
		ErrorCount:   3,
		RestartCount: 1,
		Uptime:       5 * time.Minute,
		Version:      "v1.0.0",
		ConfigHash:   "abc123",
	}
	
	// Verify all fields are set correctly
	assert.InDelta(t, 45.5, sample.CPUPercent, 0.01)
	assert.InDelta(t, 256.7, sample.MemoryMB, 0.01)
	assert.Equal(t, 42, sample.GoroutineNum)
	assert.Equal(t, int64(3), sample.ErrorCount)
	assert.Equal(t, int64(1), sample.RestartCount)
	assert.Equal(t, 5*time.Minute, sample.Uptime)
	assert.Equal(t, "v1.0.0", sample.Version)
	assert.Equal(t, "abc123", sample.ConfigHash)
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	assert.Equal(t, "nrdot-host", config.ServiceName)
	assert.Equal(t, "unknown", config.ServiceVersion)
	assert.Equal(t, "production", config.Environment)
	assert.Equal(t, "otlp.nr-data.net:4317", config.Endpoint)
	assert.Equal(t, 60*time.Second, config.Interval)
	assert.True(t, config.Enabled)
}

// MockTelemetryClient is a mock implementation for testing
type MockTelemetryClient struct {
	HealthSamples    []HealthSample
	Restarts         []string
	ConfigChanges    []struct{ Old, New string }
	FeatureFlags     map[string]bool
	Metrics          []struct{ Name string; Value float64 }
	Counters         []string
	Durations        []struct{ Name string; Duration time.Duration }
	ShutdownCalled   bool
}

func NewMockTelemetryClient() *MockTelemetryClient {
	return &MockTelemetryClient{
		HealthSamples: make([]HealthSample, 0),
		Restarts:      make([]string, 0),
		FeatureFlags:  make(map[string]bool),
		Metrics:       make([]struct{ Name string; Value float64 }, 0),
		Counters:      make([]string, 0),
		Durations:     make([]struct{ Name string; Duration time.Duration }, 0),
	}
}

func (m *MockTelemetryClient) RecordHealth(sample HealthSample) error {
	m.HealthSamples = append(m.HealthSamples, sample)
	return nil
}

func (m *MockTelemetryClient) RecordRestart(reason string) error {
	m.Restarts = append(m.Restarts, reason)
	return nil
}

func (m *MockTelemetryClient) RecordConfigChange(oldVersion, newVersion string) error {
	m.ConfigChanges = append(m.ConfigChanges, struct{ Old, New string }{oldVersion, newVersion})
	return nil
}

func (m *MockTelemetryClient) RecordFeatureFlag(name string, enabled bool) error {
	m.FeatureFlags[name] = enabled
	return nil
}

func (m *MockTelemetryClient) RecordMetric(name string, value float64, _ ...interface{}) error {
	m.Metrics = append(m.Metrics, struct{ Name string; Value float64 }{name, value})
	return nil
}

func (m *MockTelemetryClient) IncrementCounter(name string, _ ...interface{}) error {
	m.Counters = append(m.Counters, name)
	return nil
}

func (m *MockTelemetryClient) RecordDuration(name string, duration time.Duration, _ ...interface{}) error {
	m.Durations = append(m.Durations, struct{ Name string; Duration time.Duration }{name, duration})
	return nil
}

func (m *MockTelemetryClient) Shutdown(ctx context.Context) error {
	m.ShutdownCalled = true
	return nil
}

func TestMockTelemetryClient(t *testing.T) {
	mock := NewMockTelemetryClient()
	
	// Test RecordHealth
	sample := HealthSample{CPUPercent: 50.0}
	assert.NoError(t, mock.RecordHealth(sample))
	assert.Len(t, mock.HealthSamples, 1)
	assert.Equal(t, 50.0, mock.HealthSamples[0].CPUPercent)
	
	// Test RecordRestart
	assert.NoError(t, mock.RecordRestart("manual"))
	assert.Contains(t, mock.Restarts, "manual")
	
	// Test RecordConfigChange
	assert.NoError(t, mock.RecordConfigChange("v1", "v2"))
	assert.Len(t, mock.ConfigChanges, 1)
	assert.Equal(t, "v1", mock.ConfigChanges[0].Old)
	assert.Equal(t, "v2", mock.ConfigChanges[0].New)
	
	// Test RecordFeatureFlag
	assert.NoError(t, mock.RecordFeatureFlag("new-feature", true))
	assert.True(t, mock.FeatureFlags["new-feature"])
	
	// Test Shutdown
	assert.NoError(t, mock.Shutdown(context.Background()))
	assert.True(t, mock.ShutdownCalled)
}