# nrdot-test-harness

Comprehensive testing framework for NRDOT-Host components.

## Overview
Provides integration testing, mock collectors, and test orchestration for validating NRDOT functionality across different scenarios.

## Test Categories
- Unit tests for each component
- Integration tests
- Performance benchmarks
- Security validation
- Compatibility testing

## Features
- Mock OTel collector
- Test data generators
- Assertion frameworks
- Coverage reporting
- CI/CD integration

## Usage
```bash
# Run all tests
make test-all

# Run specific suite
make test-integration
make test-security
```

## Integration
- Used by all NRDOT repositories
- Integrates with `nrdot-benchmark-suite`
