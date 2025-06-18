# NRDOT API Server Image
# REST API for local management

ARG BASE_IMAGE=docker.io/newrelic/nrdot-base:latest

# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the API server
RUN CGO_ENABLED=0 go build -o nrdot-api-server \
    -ldflags="-w -s -X main.version=1.0.0" \
    ./cmd/api-server

# Runtime stage
FROM ${BASE_IMAGE}

# Switch to root for installation
USER root

# Copy API server binary
COPY --from=builder /build/nrdot-api-server /usr/local/bin/nrdot-api-server

# Copy entrypoint script
COPY api-server/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Create API server specific directories
RUN mkdir -p /var/lib/nrdot/api-server && \
    chown -R nrdot:nrdot /var/lib/nrdot/api-server

# Health check
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:${NRDOT_API_PORT:-8080}/health || exit 1

# Environment variables
ENV NRDOT_API_PORT="8080" \
    NRDOT_API_HOST="0.0.0.0" \
    NRDOT_API_READ_TIMEOUT="30s" \
    NRDOT_API_WRITE_TIMEOUT="30s" \
    NRDOT_API_IDLE_TIMEOUT="120s" \
    NRDOT_API_MAX_HEADER_BYTES="1048576" \
    NRDOT_API_CORS_ENABLED="true" \
    NRDOT_API_AUTH_ENABLED="false" \
    NRDOT_API_TLS_ENABLED="false" \
    NRDOT_API_METRICS_ENABLED="true"

# Expose API port
EXPOSE 8080

# Volume for API data
VOLUME ["/var/lib/nrdot/api-server"]

# Switch back to non-root user
USER nrdot

# Labels
LABEL org.opencontainers.image.title="NRDOT API Server" \
      org.opencontainers.image.description="REST API for NRDOT management" \
      org.opencontainers.image.version="1.0.0"

# Use the entrypoint script
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]