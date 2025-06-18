# NRDOT Config Engine Image
# Configuration management and templating engine

ARG BASE_IMAGE=docker.io/newrelic/nrdot-base:latest

# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the config engine
RUN CGO_ENABLED=0 go build -o nrdot-config-engine \
    -ldflags="-w -s -X main.version=1.0.0" \
    ./cmd/config-engine

# Runtime stage
FROM ${BASE_IMAGE}

# Switch to root for installation
USER root

# Install additional dependencies for template processing
RUN apk add --no-cache jq yq

# Copy config engine binary
COPY --from=builder /build/nrdot-config-engine /usr/local/bin/nrdot-config-engine

# Copy entrypoint script
COPY config-engine/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Create config engine specific directories
RUN mkdir -p /var/lib/nrdot/config-engine/templates \
             /var/lib/nrdot/config-engine/cache \
             /var/lib/nrdot/config-engine/output && \
    chown -R nrdot:nrdot /var/lib/nrdot/config-engine

# Copy default templates
COPY nrdot-template-lib/templates /var/lib/nrdot/config-engine/templates/

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD /usr/local/bin/nrdot-config-engine health || exit 1

# Environment variables
ENV NRDOT_CONFIG_ENGINE_MODE="watch" \
    NRDOT_CONFIG_PATH="/etc/nrdot/nrdot-config.yaml" \
    NRDOT_TEMPLATE_DIR="/var/lib/nrdot/config-engine/templates" \
    NRDOT_OUTPUT_DIR="/var/lib/nrdot/config-engine/output" \
    NRDOT_CACHE_DIR="/var/lib/nrdot/config-engine/cache" \
    NRDOT_WATCH_INTERVAL="30s" \
    NRDOT_VALIDATION_ENABLED="true"

# Volume for configuration and output
VOLUME ["/etc/nrdot", "/var/lib/nrdot/config-engine"]

# Switch back to non-root user
USER nrdot

# Labels
LABEL org.opencontainers.image.title="NRDOT Config Engine" \
      org.opencontainers.image.description="Configuration management and templating for NRDOT" \
      org.opencontainers.image.version="1.0.0"

# Use the entrypoint script
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]