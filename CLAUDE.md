# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

NRDOT-HOST is an enterprise-grade OpenTelemetry distribution for host monitoring. It's a monorepo containing 30 modular components built in Go, designed to provide a hardened, opinionated telemetry solution combining OpenTelemetry's flexibility with New Relic's operational excellence.

## Essential Commands

### Building
```bash
make all              # Build all components
make build-<component> # Build specific component (e.g., make build-control-plane)
make clean            # Clean build artifacts
```

### Testing
```bash
make test             # Run all unit tests
make test-<component> # Test specific component
make test-integration # Run integration tests
make lint             # Run linters
make coverage         # Generate coverage reports
```

### Development
```bash
make dev              # Start development environment
make watch            # Watch for changes and rebuild
./quickstart.sh       # Quick setup for new developers
```

### Packaging & Deployment
```bash
make package          # Create RPM/DEB/MSI packages
make docker           # Build Docker images
make push             # Push images to registry
make deploy-k8s       # Deploy to Kubernetes
```

## Architecture & Component Structure

The codebase follows a 7-layer architecture as defined in DEPENDENCIES.md:

### Core Components & Dependencies

1. **User Interface Layer**
   - `nrdot-ctl`: Main CLI that orchestrates all operations
   - Depends on: config-engine, supervisor, api-server

2. **Configuration Layer**
   - `nrdot-config-engine`: Central configuration processor (depends on schema, template-lib)
   - `nrdot-schema`: YAML validation
   - `nrdot-template-lib`: OTel config generation
   - `nrdot-remote-config`: Dynamic updates and feature flags

3. **Execution Layer**
   - `nrdot-supervisor`: Manages OTel Collector lifecycle (depends on telemetry-client)
   - `nrdot-telemetry-client`: Self-monitoring capabilities

4. **Processor Layer** (all depend on `otel-processor-common`)
   - `otel-processor-nrsecurity`: Secret redaction (uses privileged-helper for non-root)
   - `otel-processor-nrenrich`: Metadata injection
   - `otel-processor-nrtransform`: Metric calculations
   - `otel-processor-nrcap`: Cardinality protection

5. **Testing Layer**
   - `guardian-fleet-infra`: 24/7 validation infrastructure
   - `nrdot-workload-simulators`: Load generation
   - `nrdot-compliance-validator`: Security compliance testing
   - `nrdot-benchmark-suite`: Performance comparison

6. **Deployment Layer**
   - `nrdot-packaging`: RPM/DEB/MSI packages
   - `nrdot-container-images`: Docker images
   - `nrdot-k8s-operator`, `nrdot-helm-chart`: Kubernetes deployment
   - `nrdot-ansible-role`: Automation

7. **Tools Layer**
   - `nrdot-migrate`: Migration from NR Agent/vanilla OTel
   - `nrdot-debug-tools`: Diagnostics via API
   - `nrdot-sdk-go`: Custom processor development
   - `nrdot-health-analyzer`: KPI analysis
   - `nrdot-cost-calculator`: Cardinality optimization

### Key Architectural Patterns

- **Three-Layer Translation**: Simple YAML → Config Engine → Template Library → Full OTel Config
- **Security-First Pipeline**: Data → Security → Enrichment → Transform → Cap → Export
- **Self-Monitoring Loop**: Components monitor their own health via telemetry-client

## Key Design Principles

- Security-first: Automatic secret redaction, secure defaults
- Zero-config: Works out of the box with sensible defaults
- Modular: Each component can be developed and tested independently
- Enterprise-ready: Multi-platform support, comprehensive monitoring
- Self-healing: Built-in supervision and recovery

## Development Workflow

1. Components use Go 1.21+ with modules
2. Each component has its own Makefile that integrates with the master Makefile
3. Testing is mandatory - unit tests for all new code, integration tests for cross-component features
4. Follow the dependency graph in DEPENDENCIES.md to understand component interactions
5. Use `make lint` before committing to ensure code quality

## Component Integration Notes

- **Fleet Management**: `nrdot-fleet-protocol` enables coordinated updates across thousands of hosts
- **Non-Root Capability**: `nrdot-privileged-helper` allows process monitoring without root access
- **Config Flow**: User YAML → config-engine → schema validation → template generation → OTel config
- **Processor Pipeline**: All data flows through security → enrichment → transform → cardinality cap
- **Self-Monitoring**: Every component reports health metrics via telemetry-client

## Important Files

- `Makefile`: Master build orchestration
- `DEPENDENCIES.md`: Component dependency graph and integration architecture
- `PROJECT_STATUS.md`: Current development status and roadmap
- `quickstart.sh`: Developer onboarding script
- Each component's `README.md`: Component-specific documentation

## Testing Strategy

- Unit tests: In each component's `*_test.go` files
- Integration tests: In `nrdot-test-integration/`
- 24/7 validation: `nrdot-validation-fleet/` contains continuous validation suite
- Compliance tests: `nrdot-test-compliance/` for standards verification