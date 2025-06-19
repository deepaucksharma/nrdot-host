package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEndToEndAutoConfiguration tests the complete auto-configuration flow
func TestEndToEndAutoConfiguration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// This test requires:
	// 1. nrdot-host binary to be built
	// 2. Running as root or with appropriate permissions
	// 3. Some services installed (will be mocked if not available)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Setup test environment
	testDir, err := ioutil.TempDir("", "nrdot-e2e-*")
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	configPath := filepath.Join(testDir, "config.yaml")
	dataDir := filepath.Join(testDir, "data")
	logFile := filepath.Join(testDir, "nrdot.log")

	// Create minimal config
	config := `
license_key: test-license-key
service:
  name: e2e-test
  environment: testing

auto_config:
  enabled: true
  scan_interval: 10s

data_dir: ` + dataDir + `

logging:
  level: debug
  file: ` + logFile

	err = ioutil.WriteFile(configPath, []byte(config), 0644)
	require.NoError(t, err)

	// Build the binary if needed
	binaryPath := filepath.Join(testDir, "nrdot-host")
	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, "../../cmd/nrdot-host/main_v2.go")
	buildCmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Logf("Build output: %s", output)
		t.Skip("Failed to build binary, skipping E2E test")
	}

	// Start nrdot-host
	cmd := exec.CommandContext(ctx, binaryPath, "run", 
		"--config", configPath,
		"--mode", "all",
		"--api-addr", "127.0.0.1:18080",
		"--log-level", "debug",
	)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Start()
	require.NoError(t, err)

	// Ensure cleanup
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()

	// Wait for startup
	time.Sleep(5 * time.Second)

	// Test API endpoints
	t.Run("HealthCheck", func(t *testing.T) {
		resp, err := http.Get("http://localhost:18080/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var health map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&health)
		require.NoError(t, err)

		assert.Equal(t, "healthy", health["status"])
	})

	t.Run("Discovery", func(t *testing.T) {
		resp, err := http.Get("http://localhost:18080/v1/discovery")
		require.NoError(t, err)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := ioutil.ReadAll(resp.Body)
			t.Logf("Discovery response: %s", body)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var discovery map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&discovery)
		require.NoError(t, err)

		services, ok := discovery["discovered_services"].([]interface{})
		assert.True(t, ok)
		t.Logf("Discovered %d services", len(services))
	})

	t.Run("Status", func(t *testing.T) {
		resp, err := http.Get("http://localhost:18080/v1/status")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var status map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&status)
		require.NoError(t, err)

		service, ok := status["service"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "e2e-test", service["name"])
	})

	t.Run("ConfigPreview", func(t *testing.T) {
		// Request config preview for specific services
		reqBody := `{"services": ["redis", "nginx"]}`
		
		resp, err := http.Post("http://localhost:18080/v1/discovery/preview",
			"application/json",
			bytes.NewReader([]byte(reqBody)))
		
		require.NoError(t, err)
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var preview map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&preview)
			require.NoError(t, err)

			assert.Contains(t, preview, "preview")
			assert.Contains(t, preview, "variables_required")
		}
	})

	// Check logs for auto-config activity
	time.Sleep(2 * time.Second)
	
	if logData, err := ioutil.ReadFile(logFile); err == nil {
		logs := string(logData)
		assert.Contains(t, logs, "Starting auto-configuration")
		t.Logf("Log sample:\n%s", logs[:min(len(logs), 500)])
	}
}

// TestCLICommands tests various CLI commands
func TestCLICommands(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Build binary
	binaryPath := "./nrdot-host-test"
	defer os.Remove(binaryPath)

	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, "../../cmd/nrdot-host/main_v2.go")
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Logf("Build failed: %s", output)
		t.Skip("Failed to build binary")
	}

	tests := []struct {
		name     string
		args     []string
		wantExit int
		contains string
	}{
		{
			name:     "Version",
			args:     []string{"version"},
			wantExit: 0,
			contains: "NRDOT-HOST",
		},
		{
			name:     "Help",
			args:     []string{"help"},
			wantExit: 0,
			contains: "Available Commands:",
		},
		{
			name:     "Discover",
			args:     []string{"discover", "--output=json"},
			wantExit: 0,
			contains: "discovered_services",
		},
		{
			name:     "ValidateConfig",
			args:     []string{"validate", "--config=../../examples/config/basic.yaml"},
			wantExit: 0,
			contains: "Configuration",
		},
		{
			name:     "Processes",
			args:     []string{"processes", "--top=5"},
			wantExit: 0,
			contains: "PID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.CommandContext(ctx, binaryPath, tt.args...)
			output, err := cmd.CombinedOutput()

			if tt.wantExit == 0 {
				assert.NoError(t, err, "Command output: %s", output)
			}

			if tt.contains != "" {
				assert.Contains(t, string(output), tt.contains)
			}
		})
	}
}

// TestServiceDetectionAccuracy tests the accuracy of service detection
func TestServiceDetectionAccuracy(t *testing.T) {
	// This test checks if common services are detected correctly
	// when they are actually running

	services := []struct {
		name        string
		processName string
		port        int
		checkCmd    string
	}{
		{"mysql", "mysqld", 3306, "pgrep mysqld"},
		{"postgresql", "postgres", 5432, "pgrep postgres"},
		{"redis", "redis-server", 6379, "pgrep redis-server"},
		{"nginx", "nginx", 80, "pgrep nginx"},
		{"mongodb", "mongod", 27017, "pgrep mongod"},
	}

	for _, svc := range services {
		t.Run(svc.name, func(t *testing.T) {
			// Check if service is running
			cmd := exec.Command("sh", "-c", svc.checkCmd)
			if err := cmd.Run(); err != nil {
				t.Skipf("%s not running, skipping", svc.name)
			}

			// Run discovery
			binaryPath := "./nrdot-host-test"
			cmd = exec.Command(binaryPath, "discover", "--output=json")
			output, err := cmd.Output()
			require.NoError(t, err)

			var result map[string]interface{}
			err = json.Unmarshal(output, &result)
			require.NoError(t, err)

			// Check if service was discovered
			services := result["discovered_services"].([]interface{})
			found := false
			
			for _, s := range services {
				service := s.(map[string]interface{})
				if service["type"] == svc.name {
					found = true
					assert.NotEmpty(t, service["confidence"])
					assert.NotEmpty(t, service["discovered_by"])
					break
				}
			}

			assert.True(t, found, "%s service not discovered", svc.name)
		})
	}
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}