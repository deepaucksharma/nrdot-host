# NRDOT Privileged Helper Image
# Runs as root to perform privileged operations

ARG BASE_IMAGE=docker.io/newrelic/nrdot-base:latest

# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the privileged helper with security features
RUN CGO_ENABLED=1 go build -o nrdot-privileged-helper \
    -ldflags="-w -s -X main.version=1.0.0" \
    -tags "netgo osusergo" \
    ./cmd/privileged-helper

# Runtime stage - Note: We don't use the base image here for security
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    libcap \
    tini \
    && rm -rf /var/cache/apk/*

# Copy privileged helper binary
COPY --from=builder /build/nrdot-privileged-helper /usr/local/bin/nrdot-privileged-helper

# Copy entrypoint script
COPY privileged-helper/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Set capabilities on the binary
RUN setcap 'cap_sys_ptrace,cap_dac_read_search,cap_sys_admin+eip' /usr/local/bin/nrdot-privileged-helper

# Create directories (as root)
RUN mkdir -p /var/run/nrdot /var/lib/nrdot/privileged-helper

# Health check
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD test -S /var/run/nrdot/privileged-helper.sock || exit 1

# Environment variables
ENV NRDOT_PRIVILEGED_SOCKET="/var/run/nrdot/privileged-helper.sock" \
    NRDOT_PRIVILEGED_ALLOWED_UIDS="10001" \
    NRDOT_PRIVILEGED_MAX_CONNECTIONS="10" \
    NRDOT_PRIVILEGED_TIMEOUT="30s" \
    NRDOT_PRIVILEGED_AUDIT_LOG="/var/log/nrdot/privileged-audit.log" \
    NRDOT_PRIVILEGED_RATE_LIMIT="100/minute"

# Volume for socket
VOLUME ["/var/run/nrdot"]

# Note: This container MUST run as root for privileged operations
USER root

# Labels
LABEL org.opencontainers.image.title="NRDOT Privileged Helper" \
      org.opencontainers.image.description="Privileged operations helper for NRDOT" \
      org.opencontainers.image.version="1.0.0" \
      security.privileged="true"

# Use tini and entrypoint
ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/entrypoint.sh"]