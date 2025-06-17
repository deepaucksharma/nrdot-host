# NRDOT OpenTelemetry Collector Builder

This directory contains the configuration and tooling to build a custom OpenTelemetry Collector distribution that includes all NRDOT processors.

## Overview

The NRDOT Collector is a custom distribution of the OpenTelemetry Collector that includes:

### Standard Components
- **Receivers**: OTLP, Host Metrics, Prometheus
- **Exporters**: OTLP, Prometheus, Logging
- **Extensions**: Health Check, pprof, zPages
- **Processors**: Batch, Memory Limiter

### NRDOT Custom Processors
- **nrsecurity**: Security validation and enforcement
- **nrenrich**: Data enrichment and augmentation
- **nrtransform**: Advanced data transformation
- **nrcap**: Cardinality analysis and protection

## Prerequisites

- Go 1.21 or later
- Docker and Docker Compose (for containerized deployment)
- Make

## Quick Start

1. **Build the collector**:
   ```bash
   make build
   ```

2. **Test the collector**:
   ```bash
   make test
   ```

3. **Run with Docker**:
   ```bash
   make docker-build
   make docker-run
   ```

## Building

### Local Build

The simplest way to build the collector:

```bash
make build
```

This will:
1. Install the OpenTelemetry Collector Builder if not present
2. Build the custom collector binary as `nrdot-collector`

### Production Build

For optimized production builds:

```bash
make prod
```

This creates a smaller binary with debug symbols stripped.

### Docker Build

To build a Docker image:

```bash
make docker-build
```

## Configuration

### Builder Configuration

The `otelcol-builder.yaml` file defines:
- Which components to include
- Module versions
- Local path replacements for NRDOT processors

### Test Configuration

The `test-config.yaml` provides a basic configuration for testing the built collector with all NRDOT processors enabled.

## Usage

### Running Locally

```bash
./nrdot-collector --config=test-config.yaml
```

### Running in Docker

```bash
# Start the container
make docker-run

# View logs
docker-compose -f docker/docker-compose.yaml logs -f

# Stop the container
make docker-stop
```

### Validating Configuration

```bash
./nrdot-collector validate --config=your-config.yaml
```

## Development

### Updating NRDOT Processors

When making changes to NRDOT processors:

1. Update the processor code in `../nrdot-ctl/`
2. Update dependencies:
   ```bash
   make update-deps
   ```
3. Rebuild the collector:
   ```bash
   make dev
   ```

### Adding New Components

To add new components to the distribution:

1. Edit `otelcol-builder.yaml`
2. Add the component's gomod reference
3. Update the version as needed
4. Rebuild the collector

## Troubleshooting

### Build Failures

1. **Module not found**: Ensure all NRDOT processors are present in `../nrdot-ctl/`
2. **Version conflicts**: Check that all components use compatible versions
3. **Permission denied**: Ensure you have write permissions in the build directory

### Runtime Issues

1. **Configuration errors**: Use `validate` command to check config
2. **Port conflicts**: Check that required ports are available
3. **Memory issues**: Adjust memory_limiter processor settings

## Directory Structure

```
otelcol-builder/
├── otelcol-builder.yaml    # Builder configuration
├── Makefile                # Build automation
├── README.md               # This file
├── test-config.yaml        # Test configuration
├── .gitignore              # Git ignore rules
├── docker/
│   ├── Dockerfile          # Container image definition
│   └── docker-compose.yaml # Local testing setup
└── nrdot-collector         # Built binary (git ignored)
```

## Version Management

The collector version is managed in `otelcol-builder.yaml`. To update:

1. Change `otelcol_version` for core components
2. Update individual component versions as needed
3. Rebuild the collector

## Contributing

When contributing changes:

1. Test with `make dev` before committing
2. Ensure Docker build works: `make docker-build`
3. Update documentation if adding new features
4. Follow the existing code structure and conventions