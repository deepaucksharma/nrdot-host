# NRDOT-HOST Test Suites

This directory contains all test suites for NRDOT-HOST.

## Test Organization

### unit/
Unit tests for individual components. These are typically kept with the component source code but can be added here for cross-component unit tests.

### integration/
Integration tests that verify interactions between multiple components:
- API integration tests
- Configuration engine tests
- Supervisor and collector integration
- End-to-end configuration flow

### e2e/
End-to-end test scenarios that simulate real-world usage:
- Host monitoring scenarios
- Kubernetes deployment tests
- Security compliance validation
- High cardinality data handling
- Microservices monitoring

### benchmarks/
Performance benchmarks and load tests:
- Processor performance benchmarks
- API throughput tests
- Memory and CPU usage tests
- Data pipeline performance

### fixtures/
Test data and configuration files used across test suites:
- Sample configurations
- Mock telemetry data
- Test certificates
- Expected outputs

## Running Tests

### All Tests
```bash
make test
```

### Integration Tests Only
```bash
cd integration && make test
```

### E2E Tests
```bash
cd e2e && ./run-all-tests.sh
```

### Benchmarks
```bash
make benchmark
```

## Test Requirements

- Go 1.21+
- Docker (for E2E tests)
- Kind or Minikube (for Kubernetes tests)
- Make

## Writing Tests

1. **Unit Tests**: Place next to the code being tested, follow Go conventions
2. **Integration Tests**: Add to `integration/` with clear scenario names
3. **E2E Tests**: Create scenario directories under `e2e/scenarios/`
4. **Benchmarks**: Use Go benchmark conventions, place in appropriate component

## CI/CD Integration

Tests are automatically run in CI/CD pipelines:
- Unit tests on every commit
- Integration tests on PR
- E2E tests on merge to main
- Benchmarks tracked for performance regression