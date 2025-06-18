#!/bin/bash
# Build script for NRDOT unified Docker image

set -e

# Get version info
VERSION=${VERSION:-2.0.0}
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build the Docker image
echo "Building NRDOT unified image v${VERSION}..."
docker build \
  --build-arg VERSION=${VERSION} \
  --build-arg BUILD_TIME="${BUILD_TIME}" \
  --build-arg GIT_COMMIT=${GIT_COMMIT} \
  -t nrdot-host:${VERSION} \
  -t nrdot-host:latest \
  -f docker/unified/Dockerfile \
  ../..

echo "Build complete!"
echo "Run with: docker run -v /path/to/config:/etc/nrdot nrdot-host:latest"