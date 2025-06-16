# nrdot-container-images

Container image definitions and build pipelines for NRDOT-Host.

## Overview
Multi-architecture container images for deploying NRDOT in containerized environments.

## Images
- `newrelic/nrdot-host:latest` - Main image
- `newrelic/nrdot-host:alpine` - Minimal Alpine
- `newrelic/nrdot-host:debug` - With debug tools

## Architectures
- linux/amd64
- linux/arm64
- linux/arm/v7

## Features
- Multi-stage builds
- Security scanning
- Minimal attack surface
- Non-root by default

## Build
```bash
# Build all architectures
make docker-buildx

# Push to registry
make docker-push
```

## Integration
- Used by `nrdot-k8s-operator`
- Deployed via `nrdot-helm-chart`
