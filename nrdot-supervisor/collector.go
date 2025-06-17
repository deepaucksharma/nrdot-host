package supervisor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// CollectorProcess manages the OTel Collector process
type CollectorProcess struct {
	mu              sync.Mutex
	binaryPath      string
	configPath      string
	args            []string
	env             []string
	workDir         string
	logger          *zap.Logger
	cmd             *exec.Cmd
	stdout          io.ReadCloser
	stderr          io.ReadCloser
	memoryLimit     uint64 // in bytes
	shutdownTimeout time.Duration
}

// CollectorConfig holds collector process configuration
type CollectorConfig struct {
	BinaryPath      string
	ConfigPath      string
	Args            []string
	Env             []string
	WorkDir         string
	MemoryLimit     uint64
	ShutdownTimeout time.Duration
}

// DefaultCollectorConfig returns default collector configuration
func DefaultCollectorConfig() CollectorConfig {
	return CollectorConfig{
		BinaryPath:      "otelcol",
		ConfigPath:      "/etc/otel/config.yaml",
		Args:            []string{},
		Env:             os.Environ(),
		WorkDir:         "",
		MemoryLimit:     512 * 1024 * 1024, // 512MB
		ShutdownTimeout: 30 * time.Second,
	}
}

// NewCollectorProcess creates a new collector process manager
func NewCollectorProcess(config CollectorConfig, logger *zap.Logger) *CollectorProcess {
	return &CollectorProcess{
		binaryPath:      config.BinaryPath,
		configPath:      config.ConfigPath,
		args:            config.Args,
		env:             config.Env,
		workDir:         config.WorkDir,
		logger:          logger,
		memoryLimit:     config.MemoryLimit,
		shutdownTimeout: config.ShutdownTimeout,
	}
}

// Start starts the collector process
func (c *CollectorProcess) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cmd != nil && c.cmd.Process != nil {
		return fmt.Errorf("collector process already running")
	}

	// Build command arguments
	args := append([]string{"--config", c.configPath}, c.args...)
	
	cmd := exec.CommandContext(ctx, c.binaryPath, args...)
	cmd.Env = c.env
	cmd.Dir = c.workDir
	
	// Set process group to enable killing all child processes
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Setup stdout pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating stdout pipe: %w", err)
	}
	c.stdout = stdout

	// Setup stderr pipe
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("creating stderr pipe: %w", err)
	}
	c.stderr = stderr

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting collector process: %w", err)
	}

	c.cmd = cmd
	c.logger.Info("Collector process started",
		zap.Int("pid", cmd.Process.Pid),
		zap.String("binary", c.binaryPath),
		zap.String("config", c.configPath),
	)

	// Start log readers
	go c.readLogs("stdout", c.stdout)
	go c.readLogs("stderr", c.stderr)

	return nil
}

// Stop stops the collector process
func (c *CollectorProcess) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cmd == nil || c.cmd.Process == nil {
		return nil
	}

	c.logger.Info("Stopping collector process",
		zap.Int("pid", c.cmd.Process.Pid),
	)

	// First try graceful shutdown with SIGTERM
	if err := c.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		c.logger.Warn("Failed to send SIGTERM", zap.Error(err))
	}

	// Wait for graceful shutdown or timeout
	done := make(chan error, 1)
	go func() {
		done <- c.cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		// Context cancelled, force kill
		c.logger.Warn("Context cancelled, force killing collector")
		return c.forceKill()
	case err := <-done:
		// Process exited
		c.cmd = nil
		if err != nil {
			c.logger.Warn("Collector process exited with error", zap.Error(err))
		} else {
			c.logger.Info("Collector process stopped gracefully")
		}
		return nil
	case <-time.After(c.shutdownTimeout):
		// Timeout reached, force kill
		c.logger.Warn("Graceful shutdown timeout exceeded, force killing collector")
		return c.forceKill()
	}
}

// forceKill forcefully kills the collector process
func (c *CollectorProcess) forceKill() error {
	if c.cmd == nil || c.cmd.Process == nil {
		return nil
	}

	// Kill the entire process group
	pgid, err := syscall.Getpgid(c.cmd.Process.Pid)
	if err == nil {
		syscall.Kill(-pgid, syscall.SIGKILL)
	} else {
		c.cmd.Process.Kill()
	}

	c.cmd.Wait()
	c.cmd = nil
	return nil
}

// IsRunning returns whether the collector process is running
func (c *CollectorProcess) IsRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cmd == nil || c.cmd.Process == nil {
		return false
	}

	// Check if process is still alive
	err := c.cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

// Wait waits for the collector process to exit
func (c *CollectorProcess) Wait() error {
	c.mu.Lock()
	cmd := c.cmd
	c.mu.Unlock()

	if cmd == nil {
		return nil
	}

	err := cmd.Wait()
	
	c.mu.Lock()
	c.cmd = nil
	c.mu.Unlock()

	return err
}

// Signal sends a signal to the collector process
func (c *CollectorProcess) Signal(sig os.Signal) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cmd == nil || c.cmd.Process == nil {
		return fmt.Errorf("collector process not running")
	}

	return c.cmd.Process.Signal(sig)
}

// readLogs reads and logs output from the collector
func (c *CollectorProcess) readLogs(source string, reader io.ReadCloser) {
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		c.logger.Info("Collector output",
			zap.String("source", source),
			zap.String("line", line),
		)
	}

	if err := scanner.Err(); err != nil {
		c.logger.Error("Error reading collector logs",
			zap.String("source", source),
			zap.Error(err),
		)
	}
}

// GetMemoryUsage returns the current memory usage of the collector process
func (c *CollectorProcess) GetMemoryUsage() (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cmd == nil || c.cmd.Process == nil {
		return 0, fmt.Errorf("collector process not running")
	}

	// Read process memory info from /proc
	statmPath := fmt.Sprintf("/proc/%d/statm", c.cmd.Process.Pid)
	data, err := os.ReadFile(statmPath)
	if err != nil {
		return 0, fmt.Errorf("reading process stats: %w", err)
	}

	// Parse RSS (resident set size) from statm file
	// RSS is the 2nd field in /proc/PID/statm (in pages)
	var size, rss uint64
	fmt.Sscanf(string(data), "%d %d", &size, &rss)
	
	// Convert pages to bytes
	pageSize := uint64(os.Getpagesize())
	return rss * pageSize, nil
}

// CheckMemoryLimit checks if the collector is exceeding memory limit
func (c *CollectorProcess) CheckMemoryLimit() (bool, uint64, error) {
	usage, err := c.GetMemoryUsage()
	if err != nil {
		return false, 0, err
	}

	exceeded := c.memoryLimit > 0 && usage > c.memoryLimit
	return exceeded, usage, nil
}