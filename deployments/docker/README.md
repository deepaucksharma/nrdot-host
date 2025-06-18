# NRDOT-HOST Docker Images

This directory contains Docker configurations for all NRDOT-HOST components, providing a complete containerized deployment solution.

## Version 2.0 - Unified Architecture (Recommended)

Starting with v2.0, NRDOT-HOST uses a unified binary architecture that consolidates all components into a single process, providing:

- **40% less memory usage** (300MB vs 500MB)
- **60% less CPU overhead**
- **80% fewer processes** (1 vs 5)
- **Zero IPC complexity**
- **50x faster configuration reloads**

### Quick Start v2.0

```bash
# Build unified image
make docker-unified

# Run with Docker Compose
docker-compose -f docker-compose.v2.yaml up -d

# Access services
# API: http://localhost:8080
# OTLP: localhost:4317
# Metrics: http://localhost:8888
```

### v2.0 Files
- `unified/Dockerfile` - Unified binary Docker image
- `docker-compose.v2.yaml` - v2.0 deployment configuration
- `configs/nrdot-unified.yaml` - OpenTelemetry collector configuration

## Version 1.0 - Microservices Architecture (Legacy)

The original architecture with separate components is still available for compatibility.

## Architecture

All images follow these principles:
- Multi-stage builds for size optimization
- Non-root execution (except privileged-helper)
- Health checks for container orchestration
- Signal handling for graceful shutdown
- Configurable via environment variables
- Security scanning integration

## Components

### Base Image (`base/`)
Shared base image with common dependencies and security hardening.

### Core Components
- **collector/** - OpenTelemetry Collector with custom processors
- **supervisor/** - Process lifecycle management
- **config-engine/** - Configuration management and templating
- **api-server/** - REST API for local management
- **privileged-helper/** - Privileged operations helper (runs as root)
- **ctl/** - Command-line interface tool

## Quick Start

### Build All Images
```bash
make build-all
```

### Build Individual Component
```bash
make build-collector
make build-supervisor
# etc...
```

### Run Development Stack
```bash
docker-compose up -d
```

### Run Production Stack
```bash
docker-compose -f docker-compose.prod.yaml up -d
```

## Image Details

### Size Optimization
All images use multi-stage builds and Alpine Linux base to minimize size:
- Base image: ~15MB
- Component images: ~30-50MB each
- Total deployment: <300MB

### Security Features
- Non-root user execution (UID 10001)
- Read-only root filesystem support
- No shell in production images
- Minimal attack surface
- Regular security scanning

### Health Checks
Each component includes appropriate health checks:
- HTTP endpoints for web services
- Process checks for daemons
- Configuration validation

## Environment Variables

### Common Variables
- `NRDOT_LOG_LEVEL` - Logging level (debug, info, warn, error)
- `NRDOT_CONFIG_PATH` - Configuration file path
- `NRDOT_DATA_DIR` - Data directory path

### Component-Specific Variables
See individual component documentation for specific environment variables.

## Networking

### Development Mode
- All components on `nrdot-dev` network
- Service discovery via container names
- Exposed ports for debugging

### Production Mode
- Isolated networks per component type
- TLS encryption between components
- Minimal port exposure

## Volumes

### Persistent Data
- `/var/lib/nrdot` - Runtime data
- `/etc/nrdot` - Configuration files
- `/var/log/nrdot` - Log files (optional)

### Configuration
Mount configuration files to `/etc/nrdot/` in containers.

## Building

### Prerequisites
- Docker 20.10+
- BuildKit enabled
- Make 4.0+

### Build Commands
```bash
# Build all images
make build-all

# Build with custom registry
make build-all REGISTRY=myregistry.com

# Build with custom tag
make build-all TAG=v1.2.3

# Build for multiple architectures
make build-multiarch
```

## Registry Push

### Push All Images
```bash
make push-all REGISTRY=myregistry.com TAG=v1.0.0
```

### Push Individual Image
```bash
make push-collector REGISTRY=myregistry.com TAG=v1.0.0
```

## Security Scanning

### Run Security Scans
```bash
make scan-all
```

### View Scan Results
```bash
make scan-report
```

## Troubleshooting

### Debug Mode
Set `NRDOT_DEBUG=true` to enable debug logging and additional diagnostics.

### Container Logs
```bash
docker-compose logs -f collector
docker-compose logs -f supervisor
```

### Health Status
```bash
docker-compose ps
curl http://localhost:8080/health
```

## Production Deployment

### Kubernetes
See `../nrdot-helm-chart` for Kubernetes deployment.

### Docker Swarm
```bash
docker stack deploy -c docker-compose.prod.yaml nrdot
```

### Docker Compose
```bash
docker-compose -f docker-compose.prod.yaml up -d
```

## Maintenance

### Update Images
```bash
make pull-all
docker-compose down
docker-compose up -d
```

### Cleanup
```bash
make clean
docker system prune -a
```