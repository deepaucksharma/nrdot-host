// +build linux

package supervisor

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ProcessStats contains process statistics
type ProcessStats struct {
	CPUPercent    float64
	MemoryBytes   int64
	MemoryPercent float64
	OpenFiles     int
	ThreadCount   int
}

// GetProcessStats retrieves process statistics on Linux
func GetProcessStats(pid int) (*ProcessStats, error) {
	stats := &ProcessStats{}
	
	// Read CPU stats from /proc/[pid]/stat
	if err := readCPUStats(pid, stats); err != nil {
		return nil, fmt.Errorf("failed to read CPU stats: %w", err)
	}
	
	// Read memory stats from /proc/[pid]/status
	if err := readMemoryStats(pid, stats); err != nil {
		return nil, fmt.Errorf("failed to read memory stats: %w", err)
	}
	
	// Read file descriptor count
	if err := readFDCount(pid, stats); err != nil {
		// Non-fatal, just log
		stats.OpenFiles = -1
	}
	
	return stats, nil
}

func readCPUStats(pid int, stats *ProcessStats) error {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return err
	}
	
	// Parse stat file (simplified - just get thread count for now)
	fields := strings.Fields(string(data))
	if len(fields) < 20 {
		return fmt.Errorf("invalid stat format")
	}
	
	// Field 19 is number of threads
	if threads, err := strconv.Atoi(fields[19]); err == nil {
		stats.ThreadCount = threads
	}
	
	// CPU percent calculation requires sampling over time
	// For now, return a placeholder
	stats.CPUPercent = 0.0
	
	return nil
}

func readMemoryStats(pid int, stats *ProcessStats) error {
	file, err := os.Open(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		
		switch fields[0] {
		case "VmRSS:":
			// Resident Set Size in KB
			if kb, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
				stats.MemoryBytes = kb * 1024
			}
		case "VmSize:":
			// Virtual Memory Size (could also track this)
		}
	}
	
	// Calculate memory percentage
	if totalMem := getTotalMemory(); totalMem > 0 {
		stats.MemoryPercent = float64(stats.MemoryBytes) / float64(totalMem) * 100
	}
	
	return scanner.Err()
}

func readFDCount(pid int, stats *ProcessStats) error {
	fdDir := fmt.Sprintf("/proc/%d/fd", pid)
	entries, err := os.ReadDir(fdDir)
	if err != nil {
		return err
	}
	
	stats.OpenFiles = len(entries)
	return nil
}

func getTotalMemory() int64 {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if kb, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
					return kb * 1024
				}
			}
		}
	}
	
	return 0
}