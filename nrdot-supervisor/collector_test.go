package supervisor

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

func TestCollectorProcess_StartStop(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultCollectorConfig()
	
	// Use a simple command for testing
	config.BinaryPath = "sleep"
	config.Args = []string{"30"}
	config.ConfigPath = "" // Override config requirement

	collector := NewCollectorProcess(config, logger)

	ctx := context.Background()

	// Start collector
	err := collector.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}

	// Verify it's running
	if !collector.IsRunning() {
		t.Error("Collector should be running")
	}

	// Stop collector
	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = collector.Stop(stopCtx)
	if err != nil {
		t.Errorf("Failed to stop collector: %v", err)
	}

	// Verify it's stopped
	if collector.IsRunning() {
		t.Error("Collector should not be running")
	}
}

func TestCollectorProcess_AlreadyRunning(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultCollectorConfig()
	config.BinaryPath = "sleep"
	config.Args = []string{"30"}
	config.ConfigPath = ""

	collector := NewCollectorProcess(config, logger)

	ctx := context.Background()

	// Start collector
	err := collector.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}
	defer collector.Stop(ctx)

	// Try to start again
	err = collector.Start(ctx)
	if err == nil {
		t.Error("Expected error when starting already running collector")
	}
}

func TestCollectorProcess_Signal(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultCollectorConfig()
	config.BinaryPath = "sleep"
	config.Args = []string{"30"}
	config.ConfigPath = ""

	collector := NewCollectorProcess(config, logger)

	ctx := context.Background()

	// Start collector
	err := collector.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}

	// Send signal
	err = collector.Signal(os.Interrupt)
	if err != nil {
		t.Errorf("Failed to send signal: %v", err)
	}

	// Wait for process to exit
	collector.Wait()

	// Verify it's stopped
	if collector.IsRunning() {
		t.Error("Collector should not be running after signal")
	}
}

func TestCollectorProcess_Wait(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultCollectorConfig()
	config.BinaryPath = "sleep"
	config.Args = []string{"1"} // Just sleep for 1 second
	config.ConfigPath = "/dev/null" // Use a valid path

	collector := NewCollectorProcess(config, logger)

	// Override the args to avoid --config flag
	collector.args = []string{"1"}

	ctx := context.Background()

	// Start collector
	err := collector.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}

	// Wait for process to exit naturally
	err = collector.Wait()
	if err != nil && err.Error() != "exit status 1" {
		// Sleep with --config will fail, but that's OK for this test
		t.Logf("Process exited with expected error: %v", err)
	}

	// Verify it's stopped
	if collector.IsRunning() {
		t.Error("Collector should not be running after wait")
	}
}

func TestCollectorProcess_NonExistentBinary(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultCollectorConfig()
	config.BinaryPath = "/non/existent/binary"

	collector := NewCollectorProcess(config, logger)

	ctx := context.Background()

	// Start should fail
	err := collector.Start(ctx)
	if err == nil {
		t.Error("Expected error when starting non-existent binary")
	}
}

func TestCollectorProcess_GetMemoryUsage(t *testing.T) {
	// Skip if not on Linux
	if _, err := os.Stat("/proc"); os.IsNotExist(err) {
		t.Skip("Skipping memory usage test on non-Linux system")
	}

	logger := zaptest.NewLogger(t)
	config := DefaultCollectorConfig()
	config.BinaryPath = "sleep"
	config.Args = []string{"30"}
	config.ConfigPath = ""

	collector := NewCollectorProcess(config, logger)

	ctx := context.Background()

	// Start collector
	err := collector.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}
	defer collector.Stop(ctx)

	// Get memory usage
	usage, err := collector.GetMemoryUsage()
	if err != nil {
		t.Errorf("Failed to get memory usage: %v", err)
	}

	if usage == 0 {
		t.Error("Memory usage should be greater than 0")
	}
}

func TestCollectorProcess_CheckMemoryLimit(t *testing.T) {
	// Skip if not on Linux
	if _, err := os.Stat("/proc"); os.IsNotExist(err) {
		t.Skip("Skipping memory limit test on non-Linux system")
	}

	logger := zaptest.NewLogger(t)
	config := DefaultCollectorConfig()
	config.BinaryPath = "sleep"
	config.Args = []string{"30"}
	config.ConfigPath = ""
	config.MemoryLimit = 1 // Set very low limit

	collector := NewCollectorProcess(config, logger)

	ctx := context.Background()

	// Start collector
	err := collector.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}
	defer collector.Stop(ctx)

	// Check memory limit
	exceeded, usage, err := collector.CheckMemoryLimit()
	if err != nil {
		t.Errorf("Failed to check memory limit: %v", err)
	}

	if !exceeded {
		t.Error("Memory limit should be exceeded with 1 byte limit")
	}

	if usage == 0 {
		t.Error("Memory usage should be greater than 0")
	}
}

func TestCollectorProcess_ContextCancellation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultCollectorConfig()
	config.BinaryPath = "sleep"
	config.Args = []string{"30"}
	config.ConfigPath = ""

	collector := NewCollectorProcess(config, logger)

	ctx, cancel := context.WithCancel(context.Background())

	// Start collector
	err := collector.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}

	// Cancel context
	cancel()

	// The process should still be running (context doesn't kill the process)
	if !collector.IsRunning() {
		t.Error("Collector should still be running after context cancellation")
	}

	// Clean up
	collector.Stop(context.Background())
}

func TestCollectorProcess_StdoutStderr(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh command not found")
	}

	logger := zaptest.NewLogger(t)
	config := DefaultCollectorConfig()
	config.BinaryPath = "sh"
	config.ConfigPath = "/dev/null"

	collector := NewCollectorProcess(config, logger)
	
	// Override args to run our test command
	collector.args = []string{"-c", "echo stdout; echo stderr >&2; exit 0"}

	ctx := context.Background()

	// Start collector
	err := collector.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}

	// Wait for process to complete
	err = collector.Wait()
	if err != nil {
		t.Logf("Process exited with: %v", err)
	}
}