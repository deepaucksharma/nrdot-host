# NRDOT-HOST Development Guide

This guide covers development setup, workflows, and best practices for contributing to NRDOT-HOST.

## Table of Contents

- [Development Environment](#development-environment)
- [Project Structure](#project-structure)
- [Building Components](#building-components)
- [Testing](#testing)
- [Creating New Components](#creating-new-components)
- [Debugging](#debugging)
- [Performance Profiling](#performance-profiling)
- [Release Process](#release-process)
- [Best Practices](#best-practices)

## Development Environment

### Prerequisites

- **Go**: 1.21 or later
- **Docker**: 20.10 or later
- **Make**: GNU Make 4.0 or later
- **Git**: 2.25 or later
- **OpenTelemetry Collector Builder**: 0.90.0 or later

### Initial Setup

1. **Clone the Repository**
   ```bash
   git clone https://github.com/deepaucksharma/nrdot-host.git
   cd nrdot-host
   ```

2. **Install Development Dependencies**
   ```bash
   # Install Go tools
   make install-tools
   
   # Install pre-commit hooks
   make install-hooks
   
   # Verify setup
   make check-tools
   ```

3. **Set Up Environment**
   ```bash
   # Copy environment template
   cp .env.example .env
   
   # Edit with your settings
   vi .env
   
   # Source environment
   source .env
   ```

### IDE Setup

#### VS Code

```json
// .vscode/settings.json
{
  "go.useLanguageServer": true,
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "package",
  "go.formatTool": "goimports",
  "go.testFlags": ["-v", "-race"],
  "files.associations": {
    "*.yml": "yaml",
    "Dockerfile*": "dockerfile"
  }
}
```

#### GoLand/IntelliJ

1. Open project root
2. Configure Go SDK (1.21+)
3. Enable Go modules
4. Set up file watchers for formatting

## Project Structure

```
nrdot-host/
├── cmd/                        # Main applications
├── docs/                       # Documentation
├── examples/                   # Example configurations
├── scripts/                    # Build and utility scripts
├── tools/                      # Development tools
│
├── nrdot-ctl/                 # CLI tool
│   ├── cmd/                   # CLI commands
│   ├── pkg/                   # CLI packages
│   └── README.md
│
├── nrdot-supervisor/          # Process supervisor
│   ├── cmd/                   # Main application
│   ├── pkg/                   # Supervisor logic
│   └── README.md
│
├── otel-processor-*/          # Custom processors
│   ├── pkg/                   # Processor implementation
│   ├── factory.go            # Factory pattern
│   └── README.md
│
├── integration-tests/         # Integration test suite
├── e2e-tests/                # End-to-end tests
├── kubernetes/               # K8s manifests
├── docker/                   # Docker configurations
│
├── Makefile                  # Build automation
├── go.mod                    # Go module definition
└── README.md                 # Project documentation
```

### Component Structure

Each component follows this structure:

```
component-name/
├── cmd/                      # Main application(s)
│   └── component/
│       └── main.go
├── pkg/                      # Public packages
│   ├── api/                 # API definitions
│   ├── config/              # Configuration
│   └── service/             # Core logic
├── internal/                 # Private packages
├── test/                     # Test fixtures
├── Makefile                  # Component Makefile
├── go.mod                    # Go module
├── README.md                # Component docs
└── Dockerfile               # Container image
```

## Building Components

### Building Everything

```bash
# Build all components
make all

# Build with specific flags
make all GOFLAGS="-v" LDFLAGS="-X main.version=dev"

# Build for different platforms
make all GOOS=linux GOARCH=arm64
```

### Building Individual Components

```bash
# Build specific component
make build-nrdot-ctl
make build-nrdot-supervisor
make build-processors

# Build and install
make install-nrdot-ctl

# Build Docker images
make docker-build
make docker-build-nrdot-ctl
```

### Building the Collector

```bash
# Build custom OTel collector
cd otelcol-builder
make build

# Test the collector
./bin/nrdot-collector --config=../examples/basic/config.yaml
```

### Cross-Platform Builds

```bash
# Build for all platforms
make release-build

# Specific platform
make build-nrdot-ctl GOOS=darwin GOARCH=arm64

# Windows build
make build-nrdot-ctl GOOS=windows GOARCH=amd64
```

## Testing

### Unit Tests

```bash
# Run all unit tests
make test

# Run tests for specific component
make test-nrdot-ctl
make test-processors

# Run with coverage
make test-coverage

# Run with race detection
make test-race

# Verbose output
make test TESTFLAGS="-v"
```

### Integration Tests

```bash
# Run integration tests
make test-integration

# Run specific integration test
make test-integration TEST="TestConfigEngine"

# With custom timeout
make test-integration TIMEOUT=10m
```

### End-to-End Tests

```bash
# Run E2E tests
make test-e2e

# Run specific scenario
make test-e2e SCENARIO=microservices

# With custom environment
make test-e2e ENV=staging
```

### Test Coverage

```bash
# Generate coverage report
make coverage

# View coverage in browser
make coverage-html
open coverage.html

# Check coverage threshold
make coverage-check THRESHOLD=80
```

### Writing Tests

#### Unit Test Example

```go
// pkg/service/service_test.go
package service

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestService_Start(t *testing.T) {
    tests := []struct {
        name    string
        config  Config
        wantErr bool
    }{
        {
            name:    "valid config",
            config:  Config{Port: 8080},
            wantErr: false,
        },
        {
            name:    "invalid port",
            config:  Config{Port: -1},
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            svc := New(tt.config)
            err := svc.Start()
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                defer svc.Stop()
            }
        })
    }
}
```

#### Integration Test Example

```go
// integration-tests/config_test.go
//go:build integration

package integration

import (
    "testing"
    "time"
)

func TestConfigReload(t *testing.T) {
    // Start supervisor
    supervisor := startSupervisor(t)
    defer supervisor.Stop()
    
    // Wait for startup
    require.Eventually(t, func() bool {
        return supervisor.IsHealthy()
    }, 30*time.Second, time.Second)
    
    // Modify config
    updateConfig(t, "new-config.yaml")
    
    // Verify reload
    assert.Eventually(t, func() bool {
        return supervisor.ConfigVersion() == "v2"
    }, 10*time.Second, time.Second)
}
```

## Creating New Components

### Creating a New Processor

1. **Generate Boilerplate**
   ```bash
   make new-processor NAME=myprocessor
   ```

2. **Implement Processor Interface**
   ```go
   // otel-processor-myprocessor/processor.go
   package myprocessor

   import (
       "context"
       "github.com/deepaucksharma/nrdot-host/otel-processor-common"
   )

   type processor struct {
       config *Config
   }

   func (p *processor) ProcessMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
       // Implementation
   }
   ```

3. **Create Factory**
   ```go
   // otel-processor-myprocessor/factory.go
   package myprocessor

   func NewFactory() processor.Factory {
       return processor.NewFactory(
           typeStr,
           createDefaultConfig,
           processor.WithMetrics(createMetricsProcessor, stability),
       )
   }
   ```

4. **Add Tests**
   ```go
   // otel-processor-myprocessor/processor_test.go
   func TestProcessMetrics(t *testing.T) {
       // Test implementation
   }
   ```

5. **Register in Builder**
   ```yaml
   # otelcol-builder/builder-config.yaml
   processors:
     - gomod: github.com/deepaucksharma/nrdot-host/otel-processor-myprocessor v0.0.0
   ```

### Creating a New Service

```bash
# Generate service skeleton
make new-service NAME=myservice

# This creates:
# - myservice/
#   - cmd/myservice/main.go
#   - pkg/
#   - internal/
#   - Dockerfile
#   - Makefile
#   - README.md
```

## Debugging

### Debug Builds

```bash
# Build with debug symbols
make build-debug

# Build with specific debug flags
make build GCFLAGS="all=-N -l"
```

### Using Delve

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug a component
dlv debug ./nrdot-ctl/cmd/nrdot-ctl -- status

# Attach to running process
dlv attach $(pgrep nrdot-supervisor)

# Debug test
dlv test ./nrdot-ctl/pkg/config -- -test.run TestValidate
```

### Debug Logging

```go
// Enable debug logging in code
import "github.com/deepaucksharma/nrdot-host/pkg/logging"

logger := logging.NewLogger(logging.DebugLevel)
logger.Debug("Processing request", "id", requestID)
```

### Remote Debugging

```yaml
# docker-compose.debug.yml
services:
  nrdot:
    build:
      target: debug
    ports:
      - "2345:2345"  # Delve port
    environment:
      - DEBUG=true
```

## Performance Profiling

### CPU Profiling

```bash
# Enable CPU profiling
NRDOT_PROFILE_CPU=true ./bin/nrdot-supervisor

# Or via API
curl -X POST localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof

# Analyze profile
go tool pprof cpu.prof
```

### Memory Profiling

```bash
# Enable memory profiling
NRDOT_PROFILE_MEM=true ./bin/nrdot-supervisor

# Or via API
curl localhost:8080/debug/pprof/heap > mem.prof

# Analyze
go tool pprof -http=:8081 mem.prof
```

### Trace Profiling

```bash
# Enable tracing
NRDOT_PROFILE_TRACE=true ./bin/nrdot-supervisor

# Or via API
curl localhost:8080/debug/pprof/trace?seconds=5 > trace.out

# Analyze
go tool trace trace.out
```

### Benchmarking

```bash
# Run benchmarks
make bench

# Run specific benchmark
go test -bench=BenchmarkProcessor ./otel-processor-nrsecurity/...

# With memory allocation stats
go test -bench=. -benchmem

# Compare benchmarks
go install golang.org/x/perf/cmd/benchstat@latest
benchstat old.txt new.txt
```

## Release Process

### Version Numbering

We follow [Semantic Versioning](https://semver.org/):
- MAJOR: Breaking changes
- MINOR: New features (backward compatible)
- PATCH: Bug fixes

### Creating a Release

1. **Update Version**
   ```bash
   # Update version in version.go files
   make update-version VERSION=v1.2.3
   ```

2. **Update Changelog**
   ```bash
   # Generate changelog
   make changelog VERSION=v1.2.3
   
   # Review and edit
   vi CHANGELOG.md
   ```

3. **Run Release Tests**
   ```bash
   # Full test suite
   make release-test
   
   # Build all artifacts
   make release-build
   ```

4. **Create Git Tag**
   ```bash
   git tag -a v1.2.3 -m "Release v1.2.3"
   git push origin v1.2.3
   ```

5. **GitHub Release**
   - GitHub Actions will automatically:
     - Build binaries for all platforms
     - Create Docker images
     - Generate release notes
     - Upload artifacts

### Pre-release Testing

```bash
# Create pre-release
git tag v1.2.3-rc1
git push origin v1.2.3-rc1

# Test pre-release
make test-release VERSION=v1.2.3-rc1
```

## Best Practices

### Code Style

1. **Follow Go Standards**
   - Use `gofmt` and `goimports`
   - Follow [Effective Go](https://golang.org/doc/effective_go.html)
   - Use `golangci-lint` for additional checks

2. **Error Handling**
   ```go
   // Good
   if err != nil {
       return fmt.Errorf("failed to process: %w", err)
   }
   
   // Bad
   if err != nil {
       return err
   }
   ```

3. **Context Usage**
   ```go
   // Always accept context as first parameter
   func Process(ctx context.Context, data []byte) error {
       // Check context cancellation
       select {
       case <-ctx.Done():
           return ctx.Err()
       default:
       }
       // Process...
   }
   ```

### Documentation

1. **Code Comments**
   ```go
   // Package service provides the core business logic for NRDOT.
   package service
   
   // Config holds the service configuration.
   // It is validated during initialization.
   type Config struct {
       // Port is the TCP port to listen on.
       // Must be between 1 and 65535.
       Port int `json:"port" validate:"min=1,max=65535"`
   }
   ```

2. **README Files**
   - Each component must have a README
   - Include: purpose, usage, configuration, examples

3. **API Documentation**
   ```go
   // Start initializes and starts the service.
   // It returns an error if the service cannot be started.
   //
   // The service will listen on the configured port and handle
   // incoming requests until Stop is called or the context is canceled.
   func (s *Service) Start(ctx context.Context) error {
   ```

### Testing Guidelines

1. **Test Coverage**
   - Aim for 80%+ coverage
   - Focus on critical paths
   - Test error conditions

2. **Test Organization**
   ```go
   func TestService_Feature(t *testing.T) {
       t.Run("success case", func(t *testing.T) {
           // Test happy path
       })
       
       t.Run("error case", func(t *testing.T) {
           // Test error handling
       })
   }
   ```

3. **Use Test Fixtures**
   ```go
   // testdata/config.yaml
   // testdata/expected_output.json
   
   data, err := os.ReadFile("testdata/config.yaml")
   require.NoError(t, err)
   ```

### Performance Guidelines

1. **Avoid Allocations**
   ```go
   // Good - reuse buffers
   var buf bytes.Buffer
   for _, item := range items {
       buf.Reset()
       processItem(&buf, item)
   }
   
   // Bad - allocate each time
   for _, item := range items {
       buf := new(bytes.Buffer)
       processItem(buf, item)
   }
   ```

2. **Use Sync Pools**
   ```go
   var bufferPool = sync.Pool{
       New: func() interface{} {
           return new(bytes.Buffer)
       },
   }
   ```

3. **Profile Regularly**
   - Run benchmarks before optimization
   - Profile to find actual bottlenecks
   - Measure impact of changes

### Security Practices

1. **Input Validation**
   ```go
   func ValidateInput(input string) error {
       if len(input) > MaxInputLength {
           return errors.New("input too long")
       }
       if !validPattern.MatchString(input) {
           return errors.New("invalid input format")
       }
       return nil
   }
   ```

2. **No Hardcoded Secrets**
   ```go
   // Good
   apiKey := os.Getenv("API_KEY")
   
   // Bad
   apiKey := "sk-1234567890abcdef"
   ```

3. **Secure Defaults**
   - TLS enabled by default
   - Authentication required
   - Minimal permissions

## Development Workflow

### Feature Development

1. Create feature branch
   ```bash
   git checkout -b feature/new-processor
   ```

2. Make changes with tests
   ```bash
   # Develop
   # Test
   make test
   # Lint
   make lint
   ```

3. Commit with conventional commits
   ```bash
   git commit -m "feat(processors): add new filtering processor"
   ```

4. Push and create PR
   ```bash
   git push origin feature/new-processor
   # Create PR on GitHub
   ```

### Bug Fixes

1. Create issue first
2. Reference issue in PR
3. Add regression test
4. Follow same workflow as features

### Code Review

1. **Before Submitting PR**
   - Run all tests
   - Run linters
   - Update documentation
   - Add/update tests

2. **Review Checklist**
   - [ ] Tests pass
   - [ ] Code follows style guide
   - [ ] Documentation updated
   - [ ] No security issues
   - [ ] Performance impact considered

## Useful Commands

```bash
# Development
make dev                 # Start development environment
make watch              # Watch for changes and rebuild
make clean              # Clean build artifacts

# Testing
make test-short         # Run short tests only
make test-verbose       # Verbose test output
make test-failfast      # Stop on first failure

# Debugging
make debug-env          # Print environment
make debug-build        # Build with debug info
make trace              # Enable tracing

# Documentation
make docs               # Generate documentation
make docs-serve         # Serve documentation locally

# Utilities
make fmt                # Format code
make lint               # Run linters
make generate           # Generate code
make vendor             # Update vendored dependencies
```

## Getting Help

- **Development Chat**: [Discord](https://discord.gg/nrdot)
- **Issues**: [GitHub Issues](https://github.com/deepaucksharma/nrdot-host/issues)
- **Discussions**: [GitHub Discussions](https://github.com/deepaucksharma/nrdot-host/discussions)
- **Security**: security@newrelic.com