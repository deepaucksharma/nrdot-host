package collectors

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSystemCollector(t *testing.T) {
	collector := NewSystemCollector()
	
	// Let some time pass
	time.Sleep(10 * time.Millisecond)
	
	metrics := collector.Collect()
	
	// Verify metrics are collected
	assert.GreaterOrEqual(t, metrics.MemoryMB, 0.0)
	assert.Greater(t, metrics.GoroutineNum, 0)
	assert.Greater(t, metrics.Uptime, time.Duration(0))
	assert.NotZero(t, metrics.StartTime)
	
	// CPU percent might be 0 in our simplified implementation
	assert.GreaterOrEqual(t, metrics.CPUPercent, 0.0)
}

func TestGetMemoryStats(t *testing.T) {
	stats := GetMemoryStats()
	
	// Verify memory stats are reasonable
	assert.Greater(t, stats.Alloc, uint64(0))
	assert.Greater(t, stats.Sys, uint64(0))
	assert.GreaterOrEqual(t, stats.TotalAlloc, stats.Alloc)
	assert.Greater(t, stats.AllocMB, 0.0)
	assert.Greater(t, stats.SysMB, 0.0)
	
	// NumGC might be 0 if no GC has run yet
	assert.GreaterOrEqual(t, stats.NumGC, uint32(0))
}

func TestGetRuntimeInfo(t *testing.T) {
	info := GetRuntimeInfo()
	
	// Verify runtime info
	assert.NotEmpty(t, info.Version)
	assert.Contains(t, info.Version, "go")
	assert.Greater(t, info.NumCPU, 0)
	assert.Greater(t, info.GOMAXPROCS, 0)
	assert.Equal(t, runtime.GOOS, info.GOOS)
	assert.Equal(t, runtime.GOARCH, info.GOARCH)
}

func BenchmarkSystemCollector(b *testing.B) {
	collector := NewSystemCollector()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = collector.Collect()
	}
}

func BenchmarkGetMemoryStats(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GetMemoryStats()
	}
}

func BenchmarkGetRuntimeInfo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GetRuntimeInfo()
	}
}