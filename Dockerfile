# Multi-stage build for NRDOT-HOST
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make build-base

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binaries
RUN make build-host build-helper build-collector

# Final stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    libc6-compat \
    libcap \
    su-exec \
    tzdata

# Create user and directories
RUN addgroup -g 1000 -S nrdot && \
    adduser -u 1000 -S nrdot -G nrdot && \
    mkdir -p /etc/nrdot /var/lib/nrdot /var/log/nrdot && \
    chown -R nrdot:nrdot /var/lib/nrdot /var/log/nrdot

# Copy binaries from builder
COPY --from=builder /build/bin/nrdot-host /usr/local/bin/nrdot-host
COPY --from=builder /build/bin/nrdot-helper /usr/local/bin/nrdot-helper
COPY --from=builder /build/bin/otelcol-nrdot /usr/local/bin/otelcol-nrdot

# Set capabilities for helper
RUN chmod 755 /usr/local/bin/nrdot-host && \
    chmod 4755 /usr/local/bin/nrdot-helper && \
    setcap cap_sys_ptrace,cap_dac_read_search+ep /usr/local/bin/nrdot-helper

# Copy default configuration
COPY examples/config/docker.yaml /etc/nrdot/config.yaml.default

# Copy entrypoint script
COPY scripts/docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Environment variables
ENV NRDOT_CONFIG=/etc/nrdot/config.yaml \
    NRDOT_DATA_DIR=/var/lib/nrdot \
    NRDOT_LOG_DIR=/var/log/nrdot \
    NRDOT_LOG_LEVEL=info

# Volumes
VOLUME ["/etc/nrdot", "/var/lib/nrdot", "/var/log/nrdot"]

# Expose ports
EXPOSE 8080 4317 4318

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/usr/local/bin/nrdot-host", "status"] || exit 1

# Run as nrdot user
USER nrdot

# Entrypoint
ENTRYPOINT ["/entrypoint.sh"]
CMD ["run", "--mode=all"]