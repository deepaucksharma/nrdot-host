# NRDOT CTL Image
# Command-line interface tool

ARG BASE_IMAGE=docker.io/newrelic/nrdot-base:latest

# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the CTL tool
RUN CGO_ENABLED=0 go build -o nrdot-ctl \
    -ldflags="-w -s -X github.com/newrelic/nrdot-ctl/cmd.version=1.0.0" \
    .

# Runtime stage - minimal image for CLI tool
FROM ${BASE_IMAGE}

# Switch to root for installation
USER root

# Copy CTL binary
COPY --from=builder /build/nrdot-ctl /usr/local/bin/nrdot-ctl

# Create config directory for CTL
RUN mkdir -p /home/nrdot/.nrdot && \
    chown -R nrdot:nrdot /home/nrdot/.nrdot

# Switch back to non-root user
USER nrdot

# Set home directory
ENV HOME=/home/nrdot

# Environment variables for CTL
ENV NRDOT_CTL_CONFIG_DIR="/home/nrdot/.nrdot" \
    NRDOT_CTL_OUTPUT_FORMAT="table" \
    NRDOT_CTL_NO_COLOR="false" \
    NRDOT_CTL_API_TIMEOUT="30s"

# Volume for CTL configuration
VOLUME ["/home/nrdot/.nrdot"]

# Labels
LABEL org.opencontainers.image.title="NRDOT CTL" \
      org.opencontainers.image.description="Command-line interface for NRDOT" \
      org.opencontainers.image.version="1.0.0"

# Default to showing help
ENTRYPOINT ["/usr/local/bin/nrdot-ctl"]
CMD ["--help"]