package collectors

import (
	"runtime"
	"time"
)

// SystemMetrics contains system resource metrics
type SystemMetrics struct {
	CPUPercent   float64
	MemoryMB     float64
	GoroutineNum int
	Uptime       time.Duration
	StartTime    time.Time
}

// SystemCollector collects system metrics
type SystemCollector struct {
	startTime time.Time
}

// NewSystemCollector creates a new system metrics collector
func NewSystemCollector() *SystemCollector {
	return &SystemCollector{
		startTime: time.Now(),
	}
}

// Collect gathers current system metrics
func (sc *SystemCollector) Collect() SystemMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemMetrics{
		CPUPercent:   getCPUPercent(),
		MemoryMB:     float64(m.Alloc) / 1024 / 1024,
		GoroutineNum: runtime.NumGoroutine(),
		Uptime:       time.Since(sc.startTime),
		StartTime:    sc.startTime,
	}
}

// getCPUPercent returns an approximation of CPU usage
// Note: This is a simplified implementation. In production, you might want
// to use a more sophisticated approach like reading from /proc/stat
func getCPUPercent() float64 {
	// This is a placeholder implementation
	// In a real implementation, you would:
	// 1. Read CPU stats from /proc/stat (on Linux)
	// 2. Calculate the difference from the last reading
	// 3. Return the percentage
	
	// For now, return a simulated value
	return 0.0
}

// MemoryStats provides detailed memory statistics
type MemoryStats struct {
	Alloc      uint64  // Bytes allocated and still in use
	TotalAlloc uint64  // Bytes allocated (even if freed)
	Sys        uint64  // Bytes obtained from system
	NumGC      uint32  // Number of completed GC cycles
	AllocMB    float64 // Alloc in MB
	SysMB      float64 // Sys in MB
}

// GetMemoryStats returns detailed memory statistics
func GetMemoryStats() MemoryStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return MemoryStats{
		Alloc:      m.Alloc,
		TotalAlloc: m.TotalAlloc,
		Sys:        m.Sys,
		NumGC:      m.NumGC,
		AllocMB:    float64(m.Alloc) / 1024 / 1024,
		SysMB:      float64(m.Sys) / 1024 / 1024,
	}
}

// RuntimeInfo provides information about the Go runtime
type RuntimeInfo struct {
	Version    string
	NumCPU     int
	GOMAXPROCS int
	GOOS       string
	GOARCH     string
}

// GetRuntimeInfo returns information about the Go runtime
func GetRuntimeInfo() RuntimeInfo {
	return RuntimeInfo{
		Version:    runtime.Version(),
		NumCPU:     runtime.NumCPU(),
		GOMAXPROCS: runtime.GOMAXPROCS(0),
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
	}
}