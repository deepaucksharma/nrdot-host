package monitor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ProcessInfo contains information about a process
type ProcessInfo struct {
	PID         int32
	PPID        int32
	Name        string
	Cmdline     string
	Executable  string
	MemoryRSS   uint64  // Resident Set Size in bytes
	MemoryVMS   uint64  // Virtual Memory Size in bytes
	CPUPercent  float64
	Username    string
	UID         int32
	GID         int32
	CreateTime  int64
	NumThreads  int32
	State       string
}

// ProcessMonitor provides process monitoring capabilities
type ProcessMonitor struct {
	procPath string
}

// NewProcessMonitor creates a new process monitor
func NewProcessMonitor() *ProcessMonitor {
	return &ProcessMonitor{
		procPath: "/proc",
	}
}

// GetProcessInfo retrieves information about a specific process
func (pm *ProcessMonitor) GetProcessInfo(pid int32) (*ProcessInfo, error) {
	if pid <= 0 {
		return nil, fmt.Errorf("invalid PID: %d", pid)
	}

	pidPath := filepath.Join(pm.procPath, strconv.Itoa(int(pid)))
	
	// Check if process exists
	if _, err := os.Stat(pidPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("process %d not found", pid)
	}

	info := &ProcessInfo{
		PID: pid,
	}

	// Read process status
	if err := pm.readStatus(pidPath, info); err != nil {
		return nil, fmt.Errorf("failed to read status: %w", err)
	}

	// Read command line
	if err := pm.readCmdline(pidPath, info); err != nil {
		// Non-fatal: some processes may not have cmdline
		info.Cmdline = ""
	}

	// Read executable path
	if exe, err := os.Readlink(filepath.Join(pidPath, "exe")); err == nil {
		info.Executable = exe
	}

	// Read memory info
	if err := pm.readStatm(pidPath, info); err != nil {
		// Non-fatal: continue without memory info
	}

	return info, nil
}

// readStatus reads /proc/[pid]/status file
func (pm *ProcessMonitor) readStatus(pidPath string, info *ProcessInfo) error {
	statusPath := filepath.Join(pidPath, "status")
	file, err := os.Open(statusPath)
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
		case "Name:":
			info.Name = fields[1]
		case "State:":
			info.State = fields[1]
		case "PPid:":
			if ppid, err := strconv.ParseInt(fields[1], 10, 32); err == nil {
				info.PPID = int32(ppid)
			}
		case "Uid:":
			if uid, err := strconv.ParseInt(fields[1], 10, 32); err == nil {
				info.UID = int32(uid)
			}
		case "Gid:":
			if gid, err := strconv.ParseInt(fields[1], 10, 32); err == nil {
				info.GID = int32(gid)
			}
		case "Threads:":
			if threads, err := strconv.ParseInt(fields[1], 10, 32); err == nil {
				info.NumThreads = int32(threads)
			}
		case "VmRSS:":
			if len(fields) >= 3 && fields[2] == "kB" {
				if rss, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
					info.MemoryRSS = rss * 1024 // Convert to bytes
				}
			}
		case "VmSize:":
			if len(fields) >= 3 && fields[2] == "kB" {
				if vms, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
					info.MemoryVMS = vms * 1024 // Convert to bytes
				}
			}
		}
	}

	return scanner.Err()
}

// readCmdline reads /proc/[pid]/cmdline file
func (pm *ProcessMonitor) readCmdline(pidPath string, info *ProcessInfo) error {
	cmdlinePath := filepath.Join(pidPath, "cmdline")
	data, err := os.ReadFile(cmdlinePath)
	if err != nil {
		return err
	}

	// cmdline is null-separated, replace with spaces
	cmdline := strings.ReplaceAll(string(data), "\x00", " ")
	cmdline = strings.TrimSpace(cmdline)
	
	// Limit cmdline length for security
	const maxCmdlineLen = 4096
	if len(cmdline) > maxCmdlineLen {
		cmdline = cmdline[:maxCmdlineLen]
	}
	
	info.Cmdline = cmdline
	return nil
}

// readStatm reads /proc/[pid]/statm file for memory information
func (pm *ProcessMonitor) readStatm(pidPath string, info *ProcessInfo) error {
	statmPath := filepath.Join(pidPath, "statm")
	data, err := os.ReadFile(statmPath)
	if err != nil {
		return err
	}

	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return fmt.Errorf("invalid statm format")
	}

	// statm values are in pages, need to multiply by page size
	pageSize := int64(os.Getpagesize())

	// VmSize (total program size)
	if vms, err := strconv.ParseInt(fields[0], 10, 64); err == nil {
		info.MemoryVMS = uint64(vms * pageSize)
	}

	// VmRSS (resident set size)
	if rss, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
		info.MemoryRSS = uint64(rss * pageSize)
	}

	return nil
}

// ListProcesses returns a list of all process PIDs
func (pm *ProcessMonitor) ListProcesses() ([]int32, error) {
	entries, err := os.ReadDir(pm.procPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc: %w", err)
	}

	var pids []int32
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if directory name is a PID
		if pid, err := strconv.ParseInt(entry.Name(), 10, 32); err == nil && pid > 0 {
			pids = append(pids, int32(pid))
		}
	}

	return pids, nil
}

// GetProcessesByName returns all processes matching the given name
func (pm *ProcessMonitor) GetProcessesByName(name string) ([]*ProcessInfo, error) {
	pids, err := pm.ListProcesses()
	if err != nil {
		return nil, err
	}

	var processes []*ProcessInfo
	for _, pid := range pids {
		info, err := pm.GetProcessInfo(pid)
		if err != nil {
			// Process may have exited, continue
			continue
		}

		if info.Name == name {
			processes = append(processes, info)
		}
	}

	return processes, nil
}

// ValidatePID checks if a PID is valid and safe to query
func ValidatePID(pid int32) error {
	if pid <= 0 {
		return fmt.Errorf("invalid PID: must be positive")
	}
	
	// Additional security checks can be added here
	// For example, checking if PID is within reasonable range
	const maxPID = 4194304 // Default max PID on Linux
	if pid > maxPID {
		return fmt.Errorf("PID exceeds maximum allowed value")
	}
	
	return nil
}