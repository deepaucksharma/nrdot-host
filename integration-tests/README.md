# NRDOT-HOST Integration Tests

This directory contains the integration test framework for NRDOT-HOST, designed to test all components working together in realistic scenarios.

## Overview

The integration test framework provides comprehensive testing for:
- Basic collector functionality
- Security features (secret redaction)
- Metadata enrichment
- Metric transformations
- Cardinality limiting
- Configuration management and hot-reload

## Structure

```
integration-tests/
├── framework/          # Test framework utilities
│   ├── suite.go       # Test suite runner
│   ├── collector.go   # Collector management
│   ├── assertions.go  # Custom assertions
│   └── telemetry.go   # Telemetry generation
├── scenarios/         # Test scenarios
│   ├── basic_test.go
│   ├── security_test.go
│   ├── enrichment_test.go
│   ├── transform_test.go
│   ├── cardinality_test.go
│   └── config_test.go
├── fixtures/          # Test data
│   ├── configs/      # Test configurations
│   └── expected/     # Expected outputs
└── scripts/          # Setup/teardown scripts
```

## Prerequisites

- Docker and Docker Compose
- Go 1.21+
- Make

## Running Tests

### Quick Start

```bash
# Run all integration tests
make test

# Run specific test scenario
make test-basic
make test-security
make test-enrichment

# Run with coverage
make coverage

# Run benchmarks
make benchmark
```

### Advanced Usage

```bash
# Run tests with race detector
make test-race

# Run tests in parallel
make test-parallel

# Run with verbose logging
make test-verbose

# Memory leak detection
make test-memory

# End-to-end tests with external systems
make test-e2e
```

## Test Scenarios

### Basic Functionality Tests
- Collector startup/shutdown
- Basic telemetry reception (metrics, traces, logs)
- Data forwarding to exporters
- Health check endpoints

### Security Tests
- Secret redaction in logs
- Sensitive data filtering
- API key validation
- Secure communication

### Enrichment Tests
- Host metadata enrichment
- Container metadata enrichment
- Kubernetes metadata enrichment
- Custom attribute addition

### Transform Tests
- Metric unit conversion
- Metric aggregation
- Label manipulation
- Data type transformations

### Cardinality Tests
- High cardinality detection
- Cardinality limiting
- Performance under high cardinality load
- Memory usage validation

### Configuration Tests
- Configuration validation
- Hot-reload functionality
- Invalid configuration handling
- Environment variable substitution

## Writing New Tests

### Test Structure

```go
func TestNewFeature(t *testing.T) {
    suite := framework.NewTestSuite(t)
    defer suite.Cleanup()
    
    // Start collector with test config
    collector := suite.StartCollector("fixtures/configs/feature.yaml")
    
    // Generate test telemetry
    telemetry := framework.NewTelemetryGenerator()
    metrics := telemetry.GenerateMetrics(100)
    
    // Send telemetry
    err := collector.SendMetrics(metrics)
    require.NoError(t, err)
    
    // Verify results
    received := collector.GetReceivedMetrics()
    assertions.AssertMetricsProcessed(t, received, expected)
}
```

### Custom Assertions

```go
// Assert metrics were enriched
assertions.AssertMetricsEnriched(t, metrics, "host.name", "container.id")

// Assert secrets were redacted
assertions.AssertSecretsRedacted(t, logs, []string{"password", "api_key"})

// Assert cardinality limits
assertions.AssertCardinalityWithinLimits(t, metrics, 10000)
```

## Performance Testing

### Benchmarks

```bash
# Run all benchmarks
make benchmark

# Run specific benchmark
go test -bench=BenchmarkHighVolume -benchtime=30s ./scenarios
```

### Load Testing

The framework includes load testing capabilities:
- Configurable telemetry generation rate
- Concurrent sender simulation
- Resource usage monitoring
- Latency measurements

## Debugging

### Enable Debug Logging

```bash
TEST_LOG_LEVEL=debug make test
```

### Collector Logs

Test collector logs are available in:
- Container logs: `docker logs <container-id>`
- Test output: Enable verbose mode with `-v` flag
- Log files: `./test-output/collector.log`

### Failed Test Artifacts

When tests fail, the following artifacts are preserved:
- Collector configuration
- Collector logs
- Received telemetry data
- System metrics

## CI/CD Integration

### GitHub Actions

```yaml
- name: Run Integration Tests
  run: |
    cd integration-tests
    make build
    make test
    make coverage
```

### Jenkins

```groovy
stage('Integration Tests') {
    steps {
        dir('integration-tests') {
            sh 'make build'
            sh 'make test'
            sh 'make coverage'
        }
    }
}
```

## Troubleshooting

### Common Issues

1. **Docker permission errors**
   ```bash
   sudo usermod -aG docker $USER
   newgrp docker
   ```

2. **Port conflicts**
   - Check for conflicting services
   - Adjust port mappings in test configs

3. **Timeout errors**
   - Increase TEST_TIMEOUT in Makefile
   - Check Docker resource limits

4. **Memory issues**
   - Increase Docker memory allocation
   - Run tests sequentially instead of parallel

## Contributing

When adding new integration tests:
1. Follow existing test patterns
2. Add appropriate fixtures
3. Update documentation
4. Ensure tests are idempotent
5. Add cleanup logic
6. Consider performance impact