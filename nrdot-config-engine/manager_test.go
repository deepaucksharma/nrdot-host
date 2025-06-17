package configengine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestManager_NewManager(t *testing.T) {
	tests := []struct {
		name    string
		config  ManagerConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: ManagerConfig{
				ConfigDir:   t.TempDir(),
				OutputDir:   filepath.Join(t.TempDir(), "output"),
				MaxVersions: 10,
				Logger:      zaptest.NewLogger(t),
				DryRun:      false,
			},
			wantErr: false,
		},
		{
			name: "dry run mode",
			config: ManagerConfig{
				ConfigDir:   t.TempDir(),
				OutputDir:   "/invalid/path/that/wont/be/created",
				MaxVersions: 5,
				Logger:      zaptest.NewLogger(t),
				DryRun:      true,
			},
			wantErr: false,
		},
		{
			name: "default max versions",
			config: ManagerConfig{
				ConfigDir: t.TempDir(),
				OutputDir: filepath.Join(t.TempDir(), "output"),
				Logger:    zaptest.NewLogger(t),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewManager(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, manager)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, manager)
				assert.NotNil(t, manager.GetEngine())
				assert.NotNil(t, manager.GetWatcher())
			}
		})
	}
}

func TestManager_StartStop(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "configs")
	outputDir := filepath.Join(tempDir, "output")
	
	require.NoError(t, os.MkdirAll(configDir, 0755))

	// Create test config file
	configFile := filepath.Join(configDir, "test.yaml")
	configContent := `---
service:
  name: test-service
  environment: test
metrics:
  enabled: true
`
	require.NoError(t, os.WriteFile(configFile, []byte(configContent), 0644))

	manager, err := NewManager(ManagerConfig{
		ConfigDir:   configDir,
		OutputDir:   outputDir,
		MaxVersions: 5,
		Logger:      zaptest.NewLogger(t),
	})
	require.NoError(t, err)

	// Test starting
	ctx := context.Background()
	err = manager.Start(ctx, []string{configFile})
	assert.NoError(t, err)
	assert.True(t, manager.IsRunning())

	// Test starting when already running
	err = manager.Start(ctx, []string{configFile})
	assert.Error(t, err)

	// Give some time for processing
	time.Sleep(100 * time.Millisecond)

	// Test stopping
	err = manager.Stop()
	assert.NoError(t, err)
	assert.False(t, manager.IsRunning())

	// Test stopping when already stopped
	err = manager.Stop()
	assert.NoError(t, err)
}

func TestManager_ProcessDirectory(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "configs")
	outputDir := filepath.Join(tempDir, "output")
	
	require.NoError(t, os.MkdirAll(configDir, 0755))

	// Create multiple config files
	configs := []struct {
		name    string
		content string
	}{
		{
			name: "service1.yaml",
			content: `---
service:
  name: service1
  environment: production
metrics:
  enabled: true
  interval: 30s
traces:
  enabled: true
`,
		},
		{
			name: "service2.yml",
			content: `---
service:
  name: service2
  environment: staging
logs:
  enabled: true
  sources:
    - path: /var/log/app.log
`,
		},
		{
			name: "not-a-config.txt",
			content: "This should be ignored",
		},
	}

	for _, cfg := range configs {
		path := filepath.Join(configDir, cfg.name)
		require.NoError(t, os.WriteFile(path, []byte(cfg.content), 0644))
	}

	manager, err := NewManager(ManagerConfig{
		ConfigDir:   configDir,
		OutputDir:   outputDir,
		MaxVersions: 10,
		Logger:      zaptest.NewLogger(t),
	})
	require.NoError(t, err)

	// Start with directory path
	ctx := context.Background()
	err = manager.Start(ctx, []string{configDir})
	assert.NoError(t, err)

	// Give time for processing
	time.Sleep(100 * time.Millisecond)

	// Check version history
	history := manager.GetVersionHistory()
	assert.Len(t, history, 2) // Should have processed 2 YAML files

	// Stop manager
	require.NoError(t, manager.Stop())
}

func TestManager_ValidateAll(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "configs")
	
	require.NoError(t, os.MkdirAll(configDir, 0755))

	// Create valid and invalid config files
	validConfig := filepath.Join(configDir, "valid.yaml")
	validContent := `---
service:
  name: valid-service
  environment: test
metrics:
  enabled: true
`
	require.NoError(t, os.WriteFile(validConfig, []byte(validContent), 0644))

	invalidConfig := filepath.Join(configDir, "invalid.yaml")
	invalidContent := `---
metrics:
  enabled: true
# Missing required 'service' field
`
	require.NoError(t, os.WriteFile(invalidConfig, []byte(invalidContent), 0644))

	manager, err := NewManager(ManagerConfig{
		ConfigDir: configDir,
		OutputDir: filepath.Join(tempDir, "output"),
		Logger:    zaptest.NewLogger(t),
		DryRun:    true,
	})
	require.NoError(t, err)

	// Test validation of all files
	err = manager.ValidateAll([]string{configDir})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")

	// Test validation of valid file only
	err = manager.ValidateAll([]string{validConfig})
	assert.NoError(t, err)

	// Test validation of invalid file
	err = manager.ValidateAll([]string{invalidConfig})
	assert.Error(t, err)
}

func TestManager_VersionHistory(t *testing.T) {
	tempDir := t.TempDir()
	
	manager, err := NewManager(ManagerConfig{
		ConfigDir:   tempDir,
		OutputDir:   filepath.Join(tempDir, "output"),
		MaxVersions: 3,
		Logger:      zaptest.NewLogger(t),
	})
	require.NoError(t, err)

	// Add versions manually to test trimming
	for i := 0; i < 5; i++ {
		manager.addVersion(
			"test.yaml",
			fmt.Sprintf("v%d", i),
			[]string{fmt.Sprintf("output%d.yaml", i)},
		)
	}

	// Check that only last 3 versions are kept
	history := manager.GetVersionHistory()
	assert.Len(t, history, 3)
	assert.Equal(t, "v2", history[0].Version)
	assert.Equal(t, "v3", history[1].Version)
	assert.Equal(t, "v4", history[2].Version)
}