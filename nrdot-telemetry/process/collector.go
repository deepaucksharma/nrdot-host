package process

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

// ProcessInfo represents detailed information about a process
type ProcessInfo struct {
	PID         int32
	PPID        int32
	Name        string
	Cmdline     string
	User        string
	UID         int32
	CPUPercent  float64
	MemoryRSS   uint64 // Resident Set Size in bytes
	MemoryVMS   uint64 // Virtual Memory Size in bytes
	OpenFiles   int
	ThreadCount int32
	CreateTime  int64 // Unix timestamp
	State       string
	CPUTime     float64 // Total CPU time in seconds
}

// ProcessCollector collects detailed process metrics from /proc
type ProcessCollector struct {
	logger       *zap.Logger
	procPath     string
	topN         int
	interval     time.Duration
	cache        *processCache
	lastCPUTimes map[int32]float64
	mu           sync.RWMutex
}

type processCache struct {
	processes map[int32]*ProcessInfo
	mu        sync.RWMutex
}

// NewProcessCollector creates a new process collector
func NewProcessCollector(logger *zap.Logger, procPath string, topN int, interval time.Duration) *ProcessCollector {
	return &ProcessCollector{
		logger:       logger,
		procPath:     procPath,
		topN:         topN,
		interval:     interval,
		cache:        &processCache{processes: make(map[int32]*ProcessInfo)},
		lastCPUTimes: make(map[int32]float64),
	}
}

// Collect gathers process metrics
func (c *ProcessCollector) Collect(ctx context.Context) ([]*ProcessInfo, error) {
	entries, err := ioutil.ReadDir(c.procPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc: %w", err)
	}

	var processes []*ProcessInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pid, err := strconv.ParseInt(entry.Name(), 10, 32)
		if err != nil {
			continue // Not a PID directory
		}

		info, err := c.collectProcessInfo(int32(pid))
		if err != nil {
			c.logger.Debug("Failed to collect process info", 
				zap.Int32("pid", int32(pid)), 
				zap.Error(err))
			continue
		}

		processes = append(processes, info)
	}

	// Calculate CPU percentages
	c.calculateCPUPercentages(processes)

	// Sort and return top N by CPU usage
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].CPUPercent > processes[j].CPUPercent
	})

	if len(processes) > c.topN {
		processes = processes[:c.topN]
	}

	return processes, nil
}

// collectProcessInfo gathers information about a single process
func (c *ProcessCollector) collectProcessInfo(pid int32) (*ProcessInfo, error) {
	procDir := filepath.Join(c.procPath, strconv.Itoa(int(pid)))

	info := &ProcessInfo{PID: pid}

	// Read /proc/[pid]/stat
	if err := c.readProcStat(procDir, info); err != nil {
		return nil, err
	}

	// Read /proc/[pid]/status
	if err := c.readProcStatus(procDir, info); err != nil {
		return nil, err
	}

	// Read /proc/[pid]/cmdline
	if err := c.readProcCmdline(procDir, info); err != nil {
		// Non-fatal, some processes may not have cmdline
		c.logger.Debug("Failed to read cmdline", zap.Int32("pid", pid))
	}

	// Count open files
	info.OpenFiles = c.countOpenFiles(procDir)

	return info, nil
}

// readProcStat reads /proc/[pid]/stat
func (c *ProcessCollector) readProcStat(procDir string, info *ProcessInfo) error {
	data, err := ioutil.ReadFile(filepath.Join(procDir, "stat"))
	if err != nil {
		return err
	}

	// Parse stat file - format is complex due to comm field potentially containing spaces/parens
	content := string(data)
	
	// Find the last ')' which closes the comm field
	lastParen := strings.LastIndex(content, ")")
	if lastParen == -1 {
		return fmt.Errorf("invalid stat format")
	}

	// Extract comm (process name)
	firstParen := strings.Index(content, "(")
	if firstParen != -1 && lastParen > firstParen {
		info.Name = content[firstParen+1:lastParen]
	}

	// Parse remaining fields after comm
	fields := strings.Fields(content[lastParen+1:])
	if len(fields) < 20 {
		return fmt.Errorf("insufficient fields in stat")
	}

	// Field indices (0-based after comm)
	info.State = fields[0]
	ppid, _ := strconv.ParseInt(fields[1], 10, 32)
	info.PPID = int32(ppid)

	// CPU times (utime + stime) in clock ticks
	utime, _ := strconv.ParseUint(fields[11], 10, 64)
	stime, _ := strconv.ParseUint(fields[12], 10, 64)
	info.CPUTime = float64(utime+stime) / 100.0 // Convert to seconds (assuming 100Hz)

	// Thread count
	threads, _ := strconv.ParseInt(fields[17], 10, 32)
	info.ThreadCount = int32(threads)

	// Start time
	starttime, _ := strconv.ParseUint(fields[19], 10, 64)
	info.CreateTime = c.getBootTime() + int64(starttime/100)

	// Memory
	vsize, _ := strconv.ParseUint(fields[20], 10, 64)
	info.MemoryVMS = vsize

	rss, _ := strconv.ParseInt(fields[21], 10, 64)
	info.MemoryRSS = uint64(rss) * uint64(os.Getpagesize())

	return nil
}

// readProcStatus reads /proc/[pid]/status for additional info
func (c *ProcessCollector) readProcStatus(procDir string, info *ProcessInfo) error {
	file, err := os.Open(filepath.Join(procDir, "status"))
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
		case "Uid:":
			uid, _ := strconv.ParseInt(fields[1], 10, 32)
			info.UID = int32(uid)
			info.User = c.getUsername(int32(uid))
		}
	}

	return scanner.Err()
}

// readProcCmdline reads /proc/[pid]/cmdline
func (c *ProcessCollector) readProcCmdline(procDir string, info *ProcessInfo) error {
	data, err := ioutil.ReadFile(filepath.Join(procDir, "cmdline"))
	if err != nil {
		return err
	}

	// Replace null bytes with spaces
	info.Cmdline = strings.ReplaceAll(string(data), "\x00", " ")
	info.Cmdline = strings.TrimSpace(info.Cmdline)

	return nil
}

// countOpenFiles counts open file descriptors
func (c *ProcessCollector) countOpenFiles(procDir string) int {
	fdDir := filepath.Join(procDir, "fd")
	entries, err := ioutil.ReadDir(fdDir)
	if err != nil {
		return 0
	}
	return len(entries)
}

// calculateCPUPercentages calculates CPU usage percentages
func (c *ProcessCollector) calculateCPUPercentages(processes []*ProcessInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	currentTime := time.Now()
	
	for _, proc := range processes {
		lastCPU, exists := c.lastCPUTimes[proc.PID]
		if exists {
			// Calculate percentage based on CPU time difference
			cpuDiff := proc.CPUTime - lastCPU
			timeDiff := c.interval.Seconds()
			if timeDiff > 0 {
				proc.CPUPercent = (cpuDiff / timeDiff) * 100.0
			}
		}
		c.lastCPUTimes[proc.PID] = proc.CPUTime
	}

	// Clean up old PIDs
	currentPIDs := make(map[int32]bool)
	for _, proc := range processes {
		currentPIDs[proc.PID] = true
	}
	
	for pid := range c.lastCPUTimes {
		if !currentPIDs[pid] {
			delete(c.lastCPUTimes, pid)
		}
	}
}

// getBootTime gets system boot time
func (c *ProcessCollector) getBootTime() int64 {
	data, err := ioutil.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[0] == "btime" {
			btime, _ := strconv.ParseInt(fields[1], 10, 64)
			return btime
		}
	}

	return 0
}

// getUsername gets username from UID
func (c *ProcessCollector) getUsername(uid int32) string {
	data, err := ioutil.ReadFile("/etc/passwd")
	if err != nil {
		return fmt.Sprintf("uid:%d", uid)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ":")
		if len(fields) >= 3 {
			if uidStr := fields[2]; uidStr == strconv.Itoa(int(uid)) {
				return fields[0]
			}
		}
	}

	return fmt.Sprintf("uid:%d", uid)
}

// ConvertToMetrics converts process info to OpenTelemetry metrics
func (c *ProcessCollector) ConvertToMetrics(processes []*ProcessInfo) pmetric.Metrics {
	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("nrdot.process")
	sm.Scope().SetVersion("1.0.0")

	timestamp := pcommon.NewTimestampFromTime(time.Now())

	// Create metrics for each process
	for _, proc := range processes {
		// CPU percentage metric
		cpuMetric := sm.Metrics().AppendEmpty()
		cpuMetric.SetName("process.cpu.percent")
		cpuMetric.SetUnit("%")
		gauge := cpuMetric.SetEmptyGauge()
		dp := gauge.DataPoints().AppendEmpty()
		dp.SetTimestamp(timestamp)
		dp.SetDoubleValue(proc.CPUPercent)
		c.setProcessAttributes(dp.Attributes(), proc)

		// Memory RSS metric
		memRSSMetric := sm.Metrics().AppendEmpty()
		memRSSMetric.SetName("process.memory.rss")
		memRSSMetric.SetUnit("By")
		memGauge := memRSSMetric.SetEmptyGauge()
		memDP := memGauge.DataPoints().AppendEmpty()
		memDP.SetTimestamp(timestamp)
		memDP.SetIntValue(int64(proc.MemoryRSS))
		c.setProcessAttributes(memDP.Attributes(), proc)

		// Memory VMS metric
		memVMSMetric := sm.Metrics().AppendEmpty()
		memVMSMetric.SetName("process.memory.vms")
		memVMSMetric.SetUnit("By")
		vmsGauge := memVMSMetric.SetEmptyGauge()
		vmsDP := vmsGauge.DataPoints().AppendEmpty()
		vmsDP.SetTimestamp(timestamp)
		vmsDP.SetIntValue(int64(proc.MemoryVMS))
		c.setProcessAttributes(vmsDP.Attributes(), proc)

		// Open files metric
		filesMetric := sm.Metrics().AppendEmpty()
		filesMetric.SetName("process.open_files")
		filesMetric.SetUnit("1")
		filesGauge := filesMetric.SetEmptyGauge()
		filesDP := filesGauge.DataPoints().AppendEmpty()
		filesDP.SetTimestamp(timestamp)
		filesDP.SetIntValue(int64(proc.OpenFiles))
		c.setProcessAttributes(filesDP.Attributes(), proc)

		// Thread count metric
		threadsMetric := sm.Metrics().AppendEmpty()
		threadsMetric.SetName("process.threads")
		threadsMetric.SetUnit("1")
		threadsGauge := threadsMetric.SetEmptyGauge()
		threadsDP := threadsGauge.DataPoints().AppendEmpty()
		threadsDP.SetTimestamp(timestamp)
		threadsDP.SetIntValue(int64(proc.ThreadCount))
		c.setProcessAttributes(threadsDP.Attributes(), proc)
	}

	return md
}

// setProcessAttributes sets common process attributes
func (c *ProcessCollector) setProcessAttributes(attrs pcommon.Map, proc *ProcessInfo) {
	attrs.PutInt("process.pid", int64(proc.PID))
	attrs.PutInt("process.ppid", int64(proc.PPID))
	attrs.PutStr("process.name", proc.Name)
	attrs.PutStr("process.user", proc.User)
	attrs.PutStr("process.state", proc.State)
	if proc.Cmdline != "" {
		attrs.PutStr("process.cmdline", proc.Cmdline)
	}
	attrs.PutInt("process.create_time", proc.CreateTime)
}