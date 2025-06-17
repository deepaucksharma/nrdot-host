# NRDOT Collector Image
# OpenTelemetry Collector with custom processors

ARG BASE_IMAGE=docker.io/newrelic/nrdot-base:latest

# Build stage
FROM golang:1.21-alpine AS builder

# Install dependencies
RUN apk add --no-cache make git gcc musl-dev linux-headers

# Set working directory
WORKDIR /build

# Copy build configuration
COPY otelcol-builder.yaml .
COPY Makefile .

# Copy processor code if building from monorepo context
# These will be copied if building from the root context
COPY otel-processor-* /build/ || true
COPY nrdot-privileged-helper /build/nrdot-privileged-helper || true

# Build the collector
RUN make build

# Runtime stage
FROM ${BASE_IMAGE}

# Switch to root for installation
USER root

# Create additional directories
RUN mkdir -p /etc/otel /var/lib/otel/checkpoint && \
    chown -R nrdot:nrdot /etc/otel /var/lib/otel

# Copy collector binary
COPY --from=builder /build/nrdot-collector /usr/local/bin/nrdot-collector

# Copy entrypoint script
COPY collector/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Set up health check endpoint
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:13133/health || exit 1

# Environment variables
ENV OTEL_RESOURCE_ATTRIBUTES="service.name=nrdot-collector" \
    NRDOT_COLLECTOR_CONFIG="/etc/otel/config.yaml" \
    NRDOT_CHECKPOINT_DIR="/var/lib/otel/checkpoint" \
    GOMAXPROCS=2

# Expose ports
# OTLP gRPC receiver
EXPOSE 4317
# OTLP HTTP receiver
EXPOSE 4318
# Prometheus metrics
EXPOSE 8888
# Health check
EXPOSE 13133
# pprof endpoint
EXPOSE 1777
# zpages
EXPOSE 55679

# Volume for configuration
VOLUME ["/etc/otel", "/var/lib/otel"]

# Switch back to non-root user
USER nrdot

# Labels
LABEL org.opencontainers.image.title="NRDOT Collector" \
      org.opencontainers.image.description="OpenTelemetry Collector with New Relic custom processors" \
      org.opencontainers.image.version="1.0.0"

# Use the entrypoint script
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["--config", "/etc/otel/config.yaml"]