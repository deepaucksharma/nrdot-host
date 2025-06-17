package models_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/newrelic/nrdot-host/nrdot-common/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectorStatus(t *testing.T) {
	now := time.Now()
	status := &models.CollectorStatus{
		State:          models.CollectorStateRunning,
		Version:        "1.0.0",
		ConfigVersion:  5,
		StartTime:      now,
		Uptime:         time.Hour,
		LastConfigLoad: now.Add(-10 * time.Minute),
		RestartCount:   2,
		Pipelines: []models.PipelineStatus{
			{
				Name:  "metrics/host",
				Type:  "metrics",
				State: "running",
				ComponentsHealth: map[string]string{
					"receiver/hostmetrics": "healthy",
					"processor/batch":      "healthy",
					"exporter/newrelic":    "healthy",
				},
				Metrics: models.PipelineMetrics{
					ItemsReceived:  1000,
					ItemsProcessed: 950,
					ItemsDropped:   10,
					ItemsExported:  940,
					ProcessingRate: 15.5,
					ErrorRate:      0.1,
					Latency: models.LatencyMetrics{
						P50:  time.Millisecond,
						P95:  5 * time.Millisecond,
						P99:  10 * time.Millisecond,
						Max:  20 * time.Millisecond,
						Mean: 2 * time.Millisecond,
					},
				},
			},
		},
		ResourceMetrics: models.ResourceMetrics{
			CPUPercent:       25.5,
			MemoryBytes:      104857600, // 100MB
			MemoryPercent:    12.5,
			GoroutineCount:   150,
			OpenFileCount:    25,
			NetworkBytesRecv: 1048576,
			NetworkBytesSent: 2097152,
		},
		Features: map[string]bool{
			"security_processor": true,
			"enrichment":         true,
			"cardinality_cap":    true,
		},
	}

	// Test JSON serialization
	data, err := json.MarshalIndent(status, "", "  ")
	require.NoError(t, err)
	
	var decoded models.CollectorStatus
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	
	assert.Equal(t, status.State, decoded.State)
	assert.Equal(t, status.Version, decoded.Version)
	assert.Equal(t, status.ConfigVersion, decoded.ConfigVersion)
	assert.Equal(t, status.RestartCount, decoded.RestartCount)
	assert.Len(t, decoded.Pipelines, 1)
	assert.Equal(t, status.Features["security_processor"], decoded.Features["security_processor"])
}

func TestLifecycleStates(t *testing.T) {
	states := []models.LifecycleState{
		models.StateUnknown,
		models.StateInitializing,
		models.StateRunning,
		models.StateStopping,
		models.StateStopped,
		models.StateError,
		models.StateReloading,
		models.StateUpdating,
	}

	for _, state := range states {
		// Verify string representation
		assert.NotEmpty(t, string(state))
		
		// Test JSON marshaling
		data, err := json.Marshal(state)
		require.NoError(t, err)
		
		var decoded models.LifecycleState
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Equal(t, state, decoded)
	}
}

func TestSystemStatus(t *testing.T) {
	status := &models.SystemStatus{
		Timestamp: time.Now(),
		CollectorStatus: &models.CollectorStatus{
			State:   models.CollectorStateRunning,
			Version: "1.0.0",
		},
		Components: []models.ComponentStatus{
			{
				Name:      "supervisor",
				Type:      "core",
				State:     models.StateRunning,
				Version:   "1.0.0",
				StartTime: time.Now().Add(-time.Hour),
			},
			{
				Name:      "api-server",
				Type:      "core",
				State:     models.StateRunning,
				Version:   "1.0.0",
				StartTime: time.Now().Add(-time.Hour),
			},
		},
		SystemInfo: models.SystemInfo{
			Hostname:      "test-host",
			OS:            "linux",
			Architecture:  "amd64",
			CPUCount:      8,
			TotalMemory:   17179869184, // 16GB
			KernelVersion: "5.15.0",
			CloudProvider: "aws",
			CloudRegion:   "us-east-1",
			Labels: map[string]string{
				"env":  "production",
				"team": "platform",
			},
		},
		ClusterInfo: &models.ClusterInfo{
			ClusterName: "prod-cluster",
			NodeName:    "node-1",
			Namespace:   "monitoring",
			PodName:     "nrdot-host-abc123",
			Labels: map[string]string{
				"app": "nrdot-host",
			},
		},
	}

	// Test full serialization
	data, err := json.MarshalIndent(status, "", "  ")
	require.NoError(t, err)
	
	var decoded models.SystemStatus
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	
	assert.Equal(t, status.SystemInfo.Hostname, decoded.SystemInfo.Hostname)
	assert.Equal(t, status.SystemInfo.CPUCount, decoded.SystemInfo.CPUCount)
	assert.NotNil(t, decoded.ClusterInfo)
	assert.Equal(t, status.ClusterInfo.ClusterName, decoded.ClusterInfo.ClusterName)
	assert.Len(t, decoded.Components, 2)
}

func TestPipelineMetrics(t *testing.T) {
	metrics := models.PipelineMetrics{
		ItemsReceived:  1000000,
		ItemsProcessed: 999000,
		ItemsDropped:   100,
		ItemsExported:  998900,
		ProcessingRate: 1000.5,
		ErrorRate:      0.01,
		Latency: models.LatencyMetrics{
			P50:  100 * time.Microsecond,
			P95:  500 * time.Microsecond,
			P99:  time.Millisecond,
			Max:  5 * time.Millisecond,
			Mean: 200 * time.Microsecond,
		},
	}

	// Calculate drop rate
	dropRate := float64(metrics.ItemsDropped) / float64(metrics.ItemsReceived) * 100
	assert.InDelta(t, 0.01, dropRate, 0.001)

	// Calculate success rate
	successRate := float64(metrics.ItemsExported) / float64(metrics.ItemsReceived) * 100
	assert.InDelta(t, 99.89, successRate, 0.01)

	// Verify latency ordering
	assert.True(t, metrics.Latency.P50 <= metrics.Latency.P95)
	assert.True(t, metrics.Latency.P95 <= metrics.Latency.P99)
	assert.True(t, metrics.Latency.P99 <= metrics.Latency.Max)
}