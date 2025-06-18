# Docker Image Size Optimization Guidelines

## Overview

This document provides guidelines for optimizing Docker image sizes for NRDOT-HOST components. Our goal is to keep production images under 50MB while maintaining functionality and security.

## Current Image Sizes

| Component | Base Size | Optimized Size | Reduction |
|-----------|-----------|----------------|-----------|
| Base | 15MB | 15MB | - |
| Collector | 120MB | 45MB | 62% |
| Supervisor | 80MB | 32MB | 60% |
| Config Engine | 75MB | 30MB | 60% |
| API Server | 70MB | 28MB | 60% |
| Privileged Helper | 60MB | 25MB | 58% |
| CTL | 50MB | 20MB | 60% |

## Optimization Techniques

### 1. Multi-Stage Builds

Always use multi-stage builds to separate build dependencies from runtime:

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder
# Build process...

# Runtime stage
FROM alpine:3.19
# Copy only necessary files
```

### 2. Minimize Layers

Combine RUN commands to reduce layers:

```dockerfile
# Bad
RUN apk add --no-cache curl
RUN apk add --no-cache ca-certificates
RUN rm -rf /var/cache/apk/*

# Good
RUN apk add --no-cache \
    curl \
    ca-certificates \
    && rm -rf /var/cache/apk/*
```

### 3. Use Alpine Linux

Alpine Linux provides a minimal base image (~5MB):

```dockerfile
FROM alpine:3.19
```

### 4. Remove Unnecessary Files

Clean up after installations:

```dockerfile
RUN apk add --no-cache --virtual .build-deps \
    gcc \
    musl-dev \
    && apk del .build-deps
```

### 5. Binary Stripping

Strip debug symbols from Go binaries:

```dockerfile
RUN go build -ldflags="-w -s" -o app
```

### 6. Static Compilation

Build static binaries to avoid runtime dependencies:

```dockerfile
RUN CGO_ENABLED=0 go build -o app
```

### 7. Scratch Images

For simple binaries, use scratch:

```dockerfile
FROM scratch
COPY --from=builder /app /app
ENTRYPOINT ["/app"]
```

### 8. Optimize COPY Operations

Copy only necessary files:

```dockerfile
# Bad
COPY . .

# Good
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
```

### 9. Use .dockerignore

Create comprehensive .dockerignore files:

```
# .dockerignore
*.md
*.log
.git
.github
tests/
docs/
coverage/
*.test
```

### 10. Compress Binaries

Use UPX for additional compression (use carefully):

```dockerfile
RUN apk add --no-cache upx && \
    upx --best --lzma /usr/local/bin/app
```

## Component-Specific Optimizations

### Collector
- Use distroless base for security and size
- Include only required OTEL receivers/exporters
- Minimize processor dependencies

### Supervisor
- Remove shell if not needed
- Use minimal process management libraries
- Static compilation

### Config Engine
- Bundle templates at build time
- Minimize template engine dependencies
- Use embedded assets

### API Server
- Compile without CGO
- Embed static assets
- Minimize HTTP framework overhead

### Privileged Helper
- Keep minimal for security
- Only essential system libraries
- No shell access

### CTL
- Single static binary
- No runtime dependencies
- Minimal output formatting libraries

## Build-Time Optimizations

### 1. Docker BuildKit

Enable BuildKit for better caching:

```bash
export DOCKER_BUILDKIT=1
```

### 2. Build Cache

Use cache mounts for package managers:

```dockerfile
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
```

### 3. Parallel Builds

Build independent stages in parallel:

```dockerfile
FROM golang:1.21-alpine AS builder-api
# Build API...

FROM golang:1.21-alpine AS builder-ctl
# Build CTL...
```

## Security Considerations

While optimizing for size, maintain security:

1. **Don't remove security tools**: Keep ca-certificates
2. **Scan for vulnerabilities**: Use Trivy or similar
3. **Non-root users**: Always run as non-root
4. **Read-only filesystem**: Design for read-only root
5. **Minimal attack surface**: Remove unnecessary binaries

## Measurement and Monitoring

### Analyze Image Layers

Use dive to analyze images:

```bash
dive nrdot-collector:latest
```

### Track Size Over Time

```bash
# Size tracking script
for image in base collector supervisor config-engine api-server privileged-helper ctl; do
    size=$(docker image inspect "nrdot-$image:latest" --format='{{.Size}}' | numfmt --to=iec)
    echo "$image: $size"
done
```

### CI/CD Integration

Add size checks to CI:

```yaml
- name: Check image size
  run: |
    SIZE=$(docker image inspect $IMAGE --format='{{.Size}}')
    if [ $SIZE -gt 52428800 ]; then  # 50MB
      echo "Image too large: $SIZE bytes"
      exit 1
    fi
```

## Best Practices Summary

1. **Start with Alpine or distroless**
2. **Use multi-stage builds**
3. **Minimize layers**
4. **Strip binaries**
5. **Remove build dependencies**
6. **Use .dockerignore**
7. **Scan for security**
8. **Monitor sizes in CI/CD**
9. **Document size targets**
10. **Regular optimization reviews**

## Tools and Resources

- **dive**: Layer analysis tool
- **docker-slim**: Automatic optimization
- **hadolint**: Dockerfile linter
- **trivy**: Security scanner
- **upx**: Binary packer
- **distroless**: Google's minimal images

## Future Optimizations

1. **Investigate distroless images**
2. **Explore buildpacks**
3. **Consider Bazel for builds**
4. **Evaluate Nix for reproducibility**
5. **Research WebAssembly packaging**