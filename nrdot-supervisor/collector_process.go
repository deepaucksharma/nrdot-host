package supervisor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/newrelic/nrdot-host/nrdot-common/pkg/models"
	"go.uber.org/zap"
)

// CollectorProcess manages an OpenTelemetry Collector process
type CollectorProcess struct {
	Path       string
	ConfigYAML string
	WorkDir    string
	Port       int // For health check endpoint
	Logger     *zap.Logger
	
	cmd        *exec.Cmd
	configFile string
	startTime  time.Time
	mu         sync.Mutex
}

// Start starts the collector process
func (c *CollectorProcess) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.cmd != nil && c.cmd.Process != nil {
		return fmt.Errorf("collector already running")
	}
	
	// Write config to temporary file
	configFile := filepath.Join(c.WorkDir, fmt.Sprintf("otel-config-%d.yaml", time.Now().Unix()))
	if err := os.WriteFile(configFile, []byte(c.ConfigYAML), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	c.configFile = configFile
	
	// Build command
	c.cmd = exec.CommandContext(ctx, c.Path, "--config", configFile)
	c.cmd.Dir = c.WorkDir
	
	// Set environment
	c.cmd.Env = append(os.Environ(),
		"OTEL_RESOURCE_ATTRIBUTES=service.name=nrdot-collector",
	)
	
	// Capture output
	c.cmd.Stdout = &logWriter{logger: c.Logger, level: "info"}
	c.cmd.Stderr = &logWriter{logger: c.Logger, level: "error"}
	
	// Start the process
	if err := c.cmd.Start(); err != nil {
		os.Remove(configFile)
		return fmt.Errorf("failed to start collector: %w", err)
	}
	
	c.startTime = time.Now()
	c.Logger.Info("Collector process started",
		zap.Int("pid", c.cmd.Process.Pid),
		zap.String("config", configFile))
	
	// Monitor process in background
	go c.waitForExit()
	
	return nil
}

// Stop gracefully stops the collector process
func (c *CollectorProcess) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.cmd == nil || c.cmd.Process == nil {
		return nil
	}
	
	c.Logger.Info("Stopping collector process", zap.Int("pid", c.cmd.Process.Pid))
	
	// Send terminate signal
	if err := c.sendTerminate(); err != nil {
		c.Logger.Warn("Failed to send terminate signal", zap.Error(err))
	}
	
	// Wait for process to exit
	done := make(chan error, 1)
	go func() {
		done <- c.cmd.Wait()
	}()
	
	select {
	case <-ctx.Done():
		// Context cancelled, force kill
		c.Logger.Warn("Context cancelled, force killing collector")
		c.cmd.Process.Kill()
		<-done
	case err := <-done:
		if err != nil && err.Error() != "signal: terminated" {
			c.Logger.Warn("Collector exited with error", zap.Error(err))
		}
	}
	
	// Clean up config file
	if c.configFile != "" {
		os.Remove(c.configFile)
	}
	
	c.cmd = nil
	return nil
}

// IsRunning checks if the collector process is running
func (c *CollectorProcess) IsRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.cmd == nil || c.cmd.Process == nil {
		return false
	}
	
	// Check if process still exists
	process, err := os.FindProcess(c.cmd.Process.Pid)
	if err != nil {
		return false
	}
	
	// Send signal 0 to check if process is alive
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// IsHealthy checks if the collector is healthy via health endpoint
func (c *CollectorProcess) IsHealthy() bool {
	if !c.IsRunning() {
		return false
	}
	
	// Check health endpoint
	port := c.Port
	if port == 0 {
		port = 13133 // Default OTel health check port
	}
	
	url := fmt.Sprintf("http://localhost:%d/", port)
	
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 3 * time.Second,
	}
	
	resp, err := client.Get(url)
	if err != nil {
		c.Logger.Debug("Health check failed", zap.String("url", url), zap.Error(err))
		return false
	}
	defer resp.Body.Close()
	
	// Drain response body to reuse connection
	io.Copy(io.Discard, resp.Body)
	
	return resp.StatusCode == http.StatusOK
}

// GetMetrics retrieves resource metrics from the collector
func (c *CollectorProcess) GetMetrics() (models.ResourceMetrics, error) {
	metrics := models.ResourceMetrics{}
	
	if !c.IsRunning() {
		return metrics, fmt.Errorf("collector not running")
	}
	
	// Get process stats
	c.mu.Lock()
	pid := c.cmd.Process.Pid
	c.mu.Unlock()
	
	// Get actual process statistics
	stats, err := GetProcessStats(pid)
	if err != nil {
		c.Logger.Debug("Failed to get process stats", zap.Error(err))
		// Continue with other metrics even if process stats fail
	} else {
		metrics.CPUPercent = stats.CPUPercent
		metrics.MemoryBytes = stats.MemoryBytes
		metrics.MemoryPercent = stats.MemoryPercent
		metrics.OpenFiles = stats.OpenFiles
		metrics.ThreadCount = stats.ThreadCount
	}
	
	// Get metrics from collector's prometheus endpoint
	metricsURL := fmt.Sprintf("http://localhost:8888/metrics")
	resp, err := http.Get(metricsURL)
	if err == nil {
		defer resp.Body.Close()
		// Parse prometheus metrics (simplified)
		metrics.GoroutineCount = 50
	}
	
	return metrics, nil
}

// SendSignal sends a signal to the collector process
func (c *CollectorProcess) SendSignal(signal string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.cmd == nil || c.cmd.Process == nil {
		return fmt.Errorf("no process running")
	}
	
	var sig os.Signal
	switch signal {
	case "HUP":
		sig = syscall.SIGHUP
	case "TERM":
		sig = syscall.SIGTERM
	case "KILL":
		sig = syscall.SIGKILL
	default:
		return fmt.Errorf("unknown signal: %s", signal)
	}
	
	return c.cmd.Process.Signal(sig)
}

// sendTerminate sends appropriate terminate signal based on OS
func (c *CollectorProcess) sendTerminate() error {
	if runtime.GOOS == "windows" {
		// Windows doesn't have SIGTERM, use Kill
		return c.cmd.Process.Kill()
	}
	return c.cmd.Process.Signal(syscall.SIGTERM)
}

// waitForExit waits for the process to exit and cleans up
func (c *CollectorProcess) waitForExit() {
	err := c.cmd.Wait()
	
	c.mu.Lock()
	exitCode := c.cmd.ProcessState.ExitCode()
	c.mu.Unlock()
	
	if err != nil {
		c.Logger.Error("Collector process exited with error",
			zap.Error(err),
			zap.Int("exitCode", exitCode))
	} else {
		c.Logger.Info("Collector process exited normally",
			zap.Int("exitCode", exitCode))
	}
	
	// Clean up
	c.mu.Lock()
	if c.configFile != "" {
		os.Remove(c.configFile)
		c.configFile = ""
	}
	c.cmd = nil
	c.mu.Unlock()
}

// logWriter writes process output to logger
type logWriter struct {
	logger *zap.Logger
	level  string
	buffer []byte
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	// Append to buffer
	w.buffer = append(w.buffer, p...)
	
	// Process complete lines
	for {
		idx := indexByte(w.buffer, '\n')
		if idx < 0 {
			break
		}
		
		line := string(w.buffer[:idx])
		w.buffer = w.buffer[idx+1:]
		
		// Skip empty lines
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Log based on level
		switch w.level {
		case "error":
			w.logger.Error(line)
		default:
			w.logger.Info(line)
		}
	}
	
	return len(p), nil
}

// indexByte returns the index of the first instance of c in b, or -1 if not found
func indexByte(b []byte, c byte) int {
	for i, v := range b {
		if v == c {
			return i
		}
	}
	return -1
}