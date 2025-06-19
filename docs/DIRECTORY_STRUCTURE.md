# NRDOT-HOST Directory Structure

This document describes the organization of the NRDOT-HOST repository after the v2.0 reorganization.

## Root Directory

```
NRDOT-HOST/
├── cmd/                    # Command-line applications
│   └── nrdot-host/        # Unified binary (v2.0)
├── nrdot-*/               # Core components
├── processors/            # OpenTelemetry processors
├── deployments/           # Deployment configurations
├── tests/                 # Test suites
├── docs/                  # Documentation
├── build/                 # Build artifacts (gitignored)
├── archive/               # Historical/deprecated content
└── scripts/               # Utility scripts
```

## Core Components

- **nrdot-common/**: Shared interfaces, models, and utilities
- **nrdot-supervisor/**: Process supervisor with embedded API server
- **nrdot-config-engine/**: Configuration validation and generation
- **nrdot-api-server/**: REST API server components
- **nrdot-ctl/**: Command-line control tool
- **nrdot-schema/**: Configuration schema definitions
- **nrdot-telemetry-client/**: Telemetry client library
- **nrdot-template-lib/**: Template processing library
- **nrdot-privileged-helper/**: Privileged operations helper

## Processors Directory

All OpenTelemetry processors are consolidated under `processors/`:

```
processors/
├── common/        # Shared processor code
├── nrsecurity/    # Security processor
├── nrenrich/      # Enrichment processor
├── nrtransform/   # Transform processor
└── nrcap/         # CAP processor
```

## Deployments Directory

All deployment configurations are consolidated under `deployments/`:

```
deployments/
├── docker/        # Docker images and compose files
│   ├── unified/   # Unified binary Docker image
│   └── legacy/    # Legacy microservices images
├── kubernetes/    # Kubernetes manifests
│   ├── helm/      # Helm charts
│   └── kustomize/ # Kustomize overlays
└── systemd/       # SystemD service files
```

## Tests Directory

All test suites are consolidated under `tests/`:

```
tests/
├── unit/          # Unit tests (component-specific)
├── integration/   # Integration tests
├── e2e/          # End-to-end test scenarios
├── benchmarks/    # Performance benchmarks
└── fixtures/      # Test data and fixtures
```

## Documentation Organization

Documentation is organized under `docs/`:

```
docs/
├── architecture/  # Architecture documents
├── guides/        # User and deployment guides
├── development/   # Development documentation
└── api/          # API documentation
```

## Build Directory

The `build/` directory (gitignored) contains:

```
build/
├── bin/          # Compiled binaries
├── dist/         # Distribution packages
└── reports/      # Test/coverage reports
```

## Key Changes from v1.0

1. **Processors Consolidated**: All `otel-processor-*` directories moved to `processors/`
2. **Deployments Unified**: `docker/`, `kubernetes/`, `systemd/` moved to `deployments/`
3. **Tests Organized**: `integration-tests/`, `e2e-tests/` consolidated into `tests/`
4. **Documentation Structured**: Project docs moved from root to `docs/`
5. **Build Artifacts**: All binaries output to `build/bin/`

## Import Path Updates

After reorganization, import paths have changed:

- Old: `github.com/newrelic/nrdot-host/otel-processor-nrsecurity`
- New: `github.com/newrelic/nrdot-host/processors/nrsecurity`

## Makefile Targets

The main Makefile has been updated to handle the new structure:

- `make build`: Builds all components
- `make test`: Runs all tests
- `make docker-unified`: Builds the unified Docker image
- `make clean`: Cleans build artifacts

## Development Workflow

1. **Building**: Run `make build` from root to build all components
2. **Testing**: Run `make test` for all tests or `make test-<component>` for specific tests
3. **Docker**: Use `make docker-unified` for the v2.0 unified image
4. **Deployment**: Find all deployment configs in `deployments/`