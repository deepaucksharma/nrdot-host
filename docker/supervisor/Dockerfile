# NRDOT Supervisor Image
# Process lifecycle management for NRDOT components

ARG BASE_IMAGE=docker.io/newrelic/nrdot-base:latest

# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the supervisor
RUN CGO_ENABLED=0 go build -o nrdot-supervisor \
    -ldflags="-w -s -X main.version=1.0.0" \
    ./cmd/supervisor

# Runtime stage
FROM ${BASE_IMAGE}

# Switch to root for installation
USER root

# Install additional dependencies for process management
RUN apk add --no-cache procps

# Copy supervisor binary
COPY --from=builder /build/nrdot-supervisor /usr/local/bin/nrdot-supervisor

# Copy entrypoint script
COPY supervisor/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Create supervisor-specific directories
RUN mkdir -p /var/lib/nrdot/supervisor /var/run/nrdot && \
    chown -R nrdot:nrdot /var/lib/nrdot/supervisor /var/run/nrdot

# Health check
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD /usr/local/bin/nrdot-supervisor health || exit 1

# Environment variables
ENV NRDOT_SUPERVISOR_CONFIG="/etc/nrdot/supervisor.yaml" \
    NRDOT_SUPERVISOR_SOCKET="/var/run/nrdot/supervisor.sock" \
    NRDOT_SUPERVISOR_STATE_DIR="/var/lib/nrdot/supervisor" \
    NRDOT_RESTART_STRATEGY="exponential" \
    NRDOT_MAX_RESTARTS="5"

# Volume for runtime data
VOLUME ["/var/lib/nrdot/supervisor", "/var/run/nrdot"]

# Switch back to non-root user
USER nrdot

# Labels
LABEL org.opencontainers.image.title="NRDOT Supervisor" \
      org.opencontainers.image.description="Process lifecycle manager for NRDOT components" \
      org.opencontainers.image.version="1.0.0"

# Use the entrypoint script
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]