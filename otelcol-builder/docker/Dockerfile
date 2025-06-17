# Build stage
FROM golang:1.21-alpine AS builder

# Install dependencies
RUN apk add --no-cache make git

# Set working directory
WORKDIR /build

# Copy all necessary files
COPY otelcol-builder.yaml .
COPY Makefile .

# Copy the nrdot-ctl directory with processors
COPY ../nrdot-ctl /nrdot-ctl

# Build the collector
RUN make build

# Runtime stage
FROM alpine:3.19

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates

# Create non-root user
RUN addgroup -g 10001 -S otel && \
    adduser -u 10001 -S otel -G otel

# Set working directory
WORKDIR /otel

# Copy binary from builder
COPY --from=builder /build/nrdot-collector /otel/nrdot-collector

# Copy test configuration (can be overridden)
COPY test-config.yaml /etc/otel/config.yaml

# Change ownership
RUN chown -R otel:otel /otel /etc/otel

# Switch to non-root user
USER otel

# Expose ports
# OTLP gRPC
EXPOSE 4317
# OTLP HTTP
EXPOSE 4318
# Metrics
EXPOSE 8888
# Prometheus metrics
EXPOSE 8889
# Health check
EXPOSE 13133
# pprof
EXPOSE 1777
# zPages
EXPOSE 55679

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/otel/nrdot-collector", "validate", "--config=/etc/otel/config.yaml"]

# Default command
ENTRYPOINT ["/otel/nrdot-collector"]
CMD ["--config", "/etc/otel/config.yaml"]