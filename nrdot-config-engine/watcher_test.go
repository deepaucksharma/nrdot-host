package configengine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestWatcher_NewWatcher(t *testing.T) {
	engine, err := NewEngine(Config{
		OutputDir: t.TempDir(),
		Logger:    zaptest.NewLogger(t),
	})
	require.NoError(t, err)

	tests := []struct {
		name    string
		config  WatcherConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: WatcherConfig{
				Engine:   engine,
				Logger:   zaptest.NewLogger(t),
				Debounce: 100 * time.Millisecond,
			},
			wantErr: false,
		},
		{
			name: "nil logger",
			config: WatcherConfig{
				Engine: engine,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			watcher, err := NewWatcher(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, watcher)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, watcher)
				watcher.Close()
			}
		})
	}
}

func TestWatcher_Watch(t *testing.T) {
	tempDir := t.TempDir()
	
	engine, err := NewEngine(Config{
		OutputDir: filepath.Join(tempDir, "output"),
		Logger:    zaptest.NewLogger(t),
	})
	require.NoError(t, err)

	watcher, err := NewWatcher(WatcherConfig{
		Engine: engine,
		Logger: zaptest.NewLogger(t),
	})
	require.NoError(t, err)
	defer watcher.Close()

	// Create test file
	testFile := filepath.Join(tempDir, "test.yaml")
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

	// Test watching file
	err = watcher.Watch(testFile)
	assert.NoError(t, err)

	// Test watching same file again (should be idempotent)
	err = watcher.Watch(testFile)
	assert.NoError(t, err)

	// Verify path is being watched
	paths := watcher.GetWatchedPaths()
	assert.Contains(t, paths, testFile)
}

func TestWatcher_Unwatch(t *testing.T) {
	tempDir := t.TempDir()
	
	engine, err := NewEngine(Config{
		OutputDir: filepath.Join(tempDir, "output"),
		Logger:    zaptest.NewLogger(t),
	})
	require.NoError(t, err)

	watcher, err := NewWatcher(WatcherConfig{
		Engine: engine,
		Logger: zaptest.NewLogger(t),
	})
	require.NoError(t, err)
	defer watcher.Close()

	// Create and watch test file
	testFile := filepath.Join(tempDir, "test.yaml")
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))
	require.NoError(t, watcher.Watch(testFile))

	// Unwatch the file
	err = watcher.Unwatch(testFile)
	assert.NoError(t, err)

	// Verify path is no longer being watched
	paths := watcher.GetWatchedPaths()
	assert.NotContains(t, paths, testFile)

	// Test unwatching non-watched file (should be idempotent)
	err = watcher.Unwatch(testFile)
	assert.NoError(t, err)
}

func TestWatcher_FileChange(t *testing.T) {
	tempDir := t.TempDir()
	outputDir := filepath.Join(tempDir, "output")
	
	engine, err := NewEngine(Config{
		OutputDir: outputDir,
		Logger:    zaptest.NewLogger(t),
	})
	require.NoError(t, err)

	watcher, err := NewWatcher(WatcherConfig{
		Engine:   engine,
		Logger:   zaptest.NewLogger(t),
		Debounce: 50 * time.Millisecond,
	})
	require.NoError(t, err)
	defer watcher.Close()

	// Create initial config file
	configFile := filepath.Join(tempDir, "config.yaml")
	configContent := `---
apiVersion: v1
kind: Pipeline
metadata:
  name: test
spec:
  receivers:
    - type: otlp
  processors:
    - type: batch
  exporters:
    - type: newrelic
`
	require.NoError(t, os.WriteFile(configFile, []byte(configContent), 0644))

	// Watch the file
	require.NoError(t, watcher.Watch(configFile))

	// Start watcher in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcherDone := make(chan error, 1)
	go func() {
		watcherDone <- watcher.Start(ctx)
	}()

	// Give watcher time to start
	time.Sleep(100 * time.Millisecond)

	// Modify the file
	updatedContent := `---
apiVersion: v1
kind: Pipeline
metadata:
  name: test-updated
spec:
  receivers:
    - type: otlp
  processors:
    - type: batch
  exporters:
    - type: newrelic
`
	require.NoError(t, os.WriteFile(configFile, []byte(updatedContent), 0644))

	// Wait for debounce and processing
	time.Sleep(200 * time.Millisecond)

	// Cancel context to stop watcher
	cancel()

	// Wait for watcher to stop
	select {
	case err := <-watcherDone:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("Watcher did not stop in time")
	}
}

func TestWatcher_IsConfigFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"config.yaml", true},
		{"config.yml", true},
		{"test.json", false},
		{"README.md", false},
		{"/path/to/config.yaml", true},
		{"/path/to/file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isConfigFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}