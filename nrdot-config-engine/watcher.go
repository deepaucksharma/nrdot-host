package configengine

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

// Watcher watches configuration files for changes
type Watcher struct {
	logger   *zap.Logger
	engine   *Engine
	watcher  *fsnotify.Watcher
	paths    map[string]bool
	mu       sync.RWMutex
	debounce time.Duration
}

// WatcherConfig holds watcher configuration
type WatcherConfig struct {
	// Engine to process configuration changes
	Engine *Engine
	// Logger for the watcher
	Logger *zap.Logger
	// Debounce duration to avoid processing rapid changes
	Debounce time.Duration
}

// NewWatcher creates a new configuration file watcher
func NewWatcher(cfg WatcherConfig) (*Watcher, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	if cfg.Debounce == 0 {
		cfg.Debounce = 500 * time.Millisecond
	}

	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	return &Watcher{
		logger:   cfg.Logger,
		engine:   cfg.Engine,
		watcher:  fsWatcher,
		paths:    make(map[string]bool),
		debounce: cfg.Debounce,
	}, nil
}

// Watch adds a configuration file or directory to watch
func (w *Watcher) Watch(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Check if already watching
	if w.paths[path] {
		return nil
	}

	// Add to fsnotify watcher
	if err := w.watcher.Add(path); err != nil {
		return fmt.Errorf("failed to watch path %s: %w", path, err)
	}

	w.paths[path] = true
	w.logger.Info("Started watching path", zap.String("path", path))

	return nil
}

// Unwatch removes a path from watching
func (w *Watcher) Unwatch(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.paths[path] {
		return nil
	}

	if err := w.watcher.Remove(path); err != nil {
		return fmt.Errorf("failed to unwatch path %s: %w", path, err)
	}

	delete(w.paths, path)
	w.logger.Info("Stopped watching path", zap.String("path", path))

	return nil
}

// Start begins watching for file changes
func (w *Watcher) Start(ctx context.Context) error {
	// Debounce timer to avoid rapid reprocessing
	debounceTimers := make(map[string]*time.Timer)
	timerMu := sync.Mutex{}

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Watcher shutting down")
			return ctx.Err()

		case event, ok := <-w.watcher.Events:
			if !ok {
				return fmt.Errorf("watcher events channel closed")
			}

			// Only process write and create events for YAML files
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}

			if !isConfigFile(event.Name) {
				continue
			}

			w.logger.Debug("File event detected",
				zap.String("file", event.Name),
				zap.String("operation", event.Op.String()))

			// Debounce the event
			timerMu.Lock()
			if timer, exists := debounceTimers[event.Name]; exists {
				timer.Stop()
			}

			debounceTimers[event.Name] = time.AfterFunc(w.debounce, func() {
				timerMu.Lock()
				delete(debounceTimers, event.Name)
				timerMu.Unlock()

				w.handleFileChange(ctx, event.Name)
			})
			timerMu.Unlock()

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return fmt.Errorf("watcher errors channel closed")
			}
			w.logger.Error("Watcher error", zap.Error(err))
		}
	}
}

// handleFileChange processes a configuration file change
func (w *Watcher) handleFileChange(ctx context.Context, path string) {
	w.logger.Info("Processing configuration change", zap.String("path", path))

	if err := w.engine.ProcessConfig(ctx, path); err != nil {
		w.logger.Error("Failed to process configuration",
			zap.String("path", path),
			zap.Error(err))
		return
	}

	w.logger.Info("Configuration change processed successfully",
		zap.String("path", path))
}

// Close stops the watcher and cleans up resources
func (w *Watcher) Close() error {
	return w.watcher.Close()
}

// GetWatchedPaths returns a list of currently watched paths
func (w *Watcher) GetWatchedPaths() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	paths := make([]string, 0, len(w.paths))
	for path := range w.paths {
		paths = append(paths, path)
	}
	return paths
}

// isConfigFile checks if a file is a configuration file
func isConfigFile(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".yaml" || ext == ".yml"
}