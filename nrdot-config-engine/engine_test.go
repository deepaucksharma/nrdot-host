package configengine

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/newrelic/nrdot-host/nrdot-config-engine/pkg/hooks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestEngine_NewEngine(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				OutputDir: t.TempDir(),
				DryRun:    false,
				Logger:    zaptest.NewLogger(t),
			},
			wantErr: false,
		},
		{
			name: "nil logger",
			config: Config{
				OutputDir: t.TempDir(),
				DryRun:    false,
				Logger:    nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewEngine(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, engine)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, engine)
			}
		})
	}
}

func TestEngine_ProcessConfig(t *testing.T) {
	tempDir := t.TempDir()
	outputDir := filepath.Join(tempDir, "output")

	// Create test configuration file
	configPath := filepath.Join(tempDir, "test-config.yaml")
	configContent := `---
service:
  name: test-service
  environment: production
  version: v1.0.0
license_key: "${NEW_RELIC_LICENSE_KEY}"
metrics:
  enabled: true
  interval: 60s
traces:
  enabled: true
  sample_rate: 0.1
logs:
  enabled: true
  sources:
    - path: /var/log/app.log
      parser: json
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	engine, err := NewEngine(Config{
		OutputDir: outputDir,
		DryRun:    false,
		Logger:    zaptest.NewLogger(t),
	})
	require.NoError(t, err)

	// Register a test hook
	hookCalled := false
	engine.RegisterHook(hooks.HookFunc(func(ctx context.Context, event hooks.ConfigChangeEvent) error {
		hookCalled = true
		assert.Equal(t, configPath, event.ConfigPath)
		assert.NotEmpty(t, event.NewVersion)
		assert.Empty(t, event.OldVersion)
		assert.Nil(t, event.Error)
		return nil
	}))

	// Process configuration
	ctx := context.Background()
	err = engine.ProcessConfig(ctx, configPath)
	assert.NoError(t, err)
	assert.True(t, hookCalled)
	assert.NotEmpty(t, engine.GetCurrentVersion())
}

func TestEngine_ProcessConfig_DryRun(t *testing.T) {
	tempDir := t.TempDir()
	outputDir := filepath.Join(tempDir, "output")

	// Create test configuration file
	configPath := filepath.Join(tempDir, "test-config.yaml")
	configContent := `---
service:
  name: test-service
  environment: test
metrics:
  enabled: true
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	engine, err := NewEngine(Config{
		OutputDir: outputDir,
		DryRun:    true,
		Logger:    zaptest.NewLogger(t),
	})
	require.NoError(t, err)

	// Process configuration in dry-run mode
	ctx := context.Background()
	err = engine.ProcessConfig(ctx, configPath)
	assert.NoError(t, err)

	// Check that no files were created
	_, err = os.Stat(outputDir)
	assert.True(t, os.IsNotExist(err))
}

func TestEngine_ValidateConfig(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		configYAML  string
		expectError bool
	}{
		{
			name: "valid config",
			configYAML: `---
service:
  name: test-service
metrics:
  enabled: true
traces:
  enabled: true
logs:
  enabled: false
`,
			expectError: false,
		},
		{
			name: "invalid config - missing required fields",
			configYAML: `---
metrics:
  enabled: true
`,
			expectError: true,
		},
		{
			name:        "invalid yaml",
			configYAML:  `invalid: [yaml content`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(tempDir, "config.yaml")
			require.NoError(t, os.WriteFile(configPath, []byte(tt.configYAML), 0644))

			engine, err := NewEngine(Config{
				OutputDir: tempDir,
				Logger:    zaptest.NewLogger(t),
			})
			require.NoError(t, err)

			err = engine.ValidateConfig(configPath)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEngine_GetOutputDir(t *testing.T) {
	outputDir := t.TempDir()
	engine, err := NewEngine(Config{
		OutputDir: outputDir,
		Logger:    zaptest.NewLogger(t),
	})
	require.NoError(t, err)

	assert.Equal(t, outputDir, engine.GetOutputDir())
}