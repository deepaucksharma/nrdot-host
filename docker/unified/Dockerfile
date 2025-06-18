# NRDOT Unified Binary Image
# Single binary containing all NRDOT components (v2.0 architecture)

# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make ca-certificates

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the unified binary with version info
ARG VERSION=2.0.0
ARG BUILD_TIME
ARG GIT_COMMIT
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s \
    -X main.version=${VERSION} \
    -X main.buildTime=${BUILD_TIME} \
    -X main.gitCommit=${GIT_COMMIT}" \
    -o nrdot-host ./cmd/nrdot-host

# Create minimal config for health checks
RUN echo 'receivers:\n  hostmetrics:\n    collection_interval: 10s\nexporters:\n  logging:\n    loglevel: error\nservice:\n  pipelines:\n    metrics:\n      receivers: [hostmetrics]\n      exporters: [logging]' > /build/health-check.yaml

# Runtime stage - minimal Alpine image
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -g 10001 -S nrdot && \
    adduser -u 10001 -S nrdot -G nrdot

# Copy binary from builder
COPY --from=builder /build/nrdot-host /usr/local/bin/nrdot-host
COPY --from=builder /build/health-check.yaml /etc/nrdot/health-check.yaml

# Create necessary directories
RUN mkdir -p /etc/nrdot /var/lib/nrdot /var/log/nrdot /var/run/nrdot && \
    chown -R nrdot:nrdot /etc/nrdot /var/lib/nrdot /var/log/nrdot /var/run/nrdot

# Copy default configuration template
COPY configs/base.yaml /etc/nrdot/config.yaml.template

# Health check - uses minimal config to verify collector is running
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD /usr/local/bin/nrdot-host --mode=collector --config=/etc/nrdot/health-check.yaml --dry-run || exit 1

# Environment variables for unified binary
ENV NRDOT_MODE="all" \
    NRDOT_CONFIG="/etc/nrdot/config.yaml" \
    NRDOT_LOG_LEVEL="info" \
    NRDOT_API_ADDRESS="127.0.0.1:8080" \
    NRDOT_STATE_DIR="/var/lib/nrdot" \
    NRDOT_LOG_DIR="/var/log/nrdot" \
    NRDOT_RELOAD_STRATEGY="blue-green" \
    GOMEMLIMIT="300MiB"

# Volumes for persistent data
VOLUME ["/etc/nrdot", "/var/lib/nrdot", "/var/log/nrdot"]

# Expose ports
# 8080: API server (when mode=all or mode=api)
# 4317: OTLP gRPC receiver
# 4318: OTLP HTTP receiver
# 8888: OTel collector internal metrics
# 13133: Health check extension
# 55679: zPages extension
EXPOSE 8080 4317 4318 8888 13133 55679

# Run as non-root user
USER nrdot

# Labels
LABEL org.opencontainers.image.title="NRDOT Host" \
      org.opencontainers.image.description="Unified OpenTelemetry distribution for enterprise host monitoring" \
      org.opencontainers.image.version="2.0.0" \
      org.opencontainers.image.vendor="New Relic" \
      org.opencontainers.image.source="https://github.com/newrelic/nrdot-host" \
      org.opencontainers.image.documentation="https://github.com/newrelic/nrdot-host/blob/main/README.md" \
      org.opencontainers.image.licenses="Apache-2.0"

# Default command - run in unified mode
ENTRYPOINT ["/usr/local/bin/nrdot-host"]
CMD ["--mode=all"]