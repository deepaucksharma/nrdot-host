// +build !linux

package supervisor

import "fmt"

// ProcessStats contains process statistics
type ProcessStats struct {
	CPUPercent    float64
	MemoryBytes   int64
	MemoryPercent float64
	OpenFiles     int
	ThreadCount   int
}

// GetProcessStats retrieves process statistics on non-Linux systems
func GetProcessStats(pid int) (*ProcessStats, error) {
	// On non-Linux systems, return placeholder values
	// In production, this could use platform-specific APIs
	return &ProcessStats{
		CPUPercent:    0.0,
		MemoryBytes:   0,
		MemoryPercent: 0.0,
		OpenFiles:     -1,
		ThreadCount:   1,
	}, fmt.Errorf("process metrics not implemented for this platform")
}