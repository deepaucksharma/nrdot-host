# NRDOT Config Engine

The Config Engine is a comprehensive configuration management component that integrates schema validation with template generation to create a complete configuration pipeline for OpenTelemetry collectors.

## Features

- **File Watching**: Monitors configuration files for changes using fsnotify
- **Schema Validation**: Validates configurations using nrdot-schema before processing
- **Template Generation**: Generates OpenTelemetry configurations using nrdot-template-lib
- **Version Management**: Tracks configuration versions and maintains history
- **Change Notifications**: Supports hooks for notifying subscribers of configuration changes
- **Dry-Run Mode**: Validates configurations without generating output files

## Architecture

The Config Engine consists of three main components:

1. **Engine** (`engine.go`): Core configuration processing engine that integrates validation and generation
2. **Watcher** (`watcher.go`): File system watcher that monitors configuration files for changes
3. **Manager** (`manager.go`): Orchestrates the engine and watcher, manages configuration lifecycle

## Usage

### Command Line

```bash
# Build the engine
make build

# Run with configuration files
./bin/config-engine -config /path/to/configs -output /path/to/output

# Validate configurations without generating output
./bin/config-engine -config /path/to/configs -validate

# Run in dry-run mode
./bin/config-engine -config /path/to/configs -dry-run

# Watch multiple configuration paths
./bin/config-engine -config /path/to/config1.yaml,/path/to/config2.yaml
```

### Command Line Options

- `-config`: Comma-separated list of configuration files or directories to watch (required)
- `-output`: Output directory for generated configurations (default: `./output`)
- `-dry-run`: Validate configurations without generating output
- `-validate`: Only validate configurations and exit
- `-log-level`: Log level (debug, info, warn, error) (default: `info`)
- `-version`: Print version information

### Programmatic Usage

```go
import (
    configengine "github.com/newrelic/nrdot-host/nrdot-config-engine"
    "github.com/newrelic/nrdot-host/nrdot-config-engine/pkg/hooks"
)

// Create a manager
manager, err := configengine.NewManager(configengine.ManagerConfig{
    ConfigDir:   "./configs",
    OutputDir:   "./output",
    MaxVersions: 20,
    Logger:      logger,
    DryRun:      false,
})

// Register a custom hook
manager.GetEngine().RegisterHook(hooks.HookFunc(func(ctx context.Context, event hooks.ConfigChangeEvent) error {
    // Handle configuration change
    fmt.Printf("Config changed: %s -> %s\n", event.OldVersion, event.NewVersion)
    return nil
}))

// Start watching configurations
err = manager.Start(ctx, []string{"./configs"})

// Stop when done
err = manager.Stop()
```

## Hooks

The engine supports hooks for reacting to configuration changes:

```go
type MyHook struct{}

func (h *MyHook) OnConfigChange(ctx context.Context, event hooks.ConfigChangeEvent) error {
    // React to configuration change
    if event.Error != nil {
        return fmt.Errorf("config error: %w", event.Error)
    }
    
    // Reload collectors, notify services, etc.
    return nil
}

func (h *MyHook) Name() string {
    return "MyHook"
}

// Register the hook
engine.RegisterHook(&MyHook{})
```

## Development

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific test
go test -v -run TestEngine_ProcessConfig
```

### Building

```bash
# Build the binary
make build

# Clean build artifacts
make clean

# Run all checks (format, lint, test)
make check
```

## Dependencies

- `github.com/newrelic/nrdot-host/nrdot-schema`: Configuration schema validation
- `github.com/newrelic/nrdot-host/nrdot-template-lib`: Template generation for OTel configs
- `github.com/fsnotify/fsnotify`: Cross-platform file system notifications
- `go.uber.org/zap`: Structured logging

## Configuration File Format

The engine expects configuration files in YAML format that conform to the nrdot-schema specification:

```yaml
apiVersion: v1
kind: Pipeline
metadata:
  name: my-pipeline
spec:
  receivers:
    - type: otlp
      config:
        protocols:
          grpc:
            endpoint: "0.0.0.0:4317"
  processors:
    - type: batch
  exporters:
    - type: newrelic
      config:
        api_key: ${NEW_RELIC_API_KEY}
        endpoint: https://otlp.nr-data.net
```

## Version Management

The engine maintains a version history of processed configurations:

- Each successful configuration processing generates a new version
- Version history is maintained with a configurable maximum size
- Each version includes timestamp, source configuration path, and generated files

## Error Handling

The engine provides comprehensive error handling:

- Validation errors are reported with detailed messages
- File system errors are logged and don't crash the watcher
- Hook errors are logged but don't fail configuration processing
- Graceful shutdown on interrupt signals