#!/bin/sh
set -e

# NRDOT Collector Entrypoint Script

# Function to log messages
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Handle shutdown gracefully
shutdown() {
    log "Received shutdown signal, stopping collector..."
    kill -TERM "$PID" 2>/dev/null
    wait "$PID"
    exit 0
}

# Set up signal handlers
trap shutdown TERM INT

# Validate configuration if requested
if [ "$1" = "validate" ]; then
    exec /usr/local/bin/nrdot-collector validate --config="${NRDOT_COLLECTOR_CONFIG:-/etc/otel/config.yaml}"
fi

# Check if config file exists
CONFIG_FILE="${NRDOT_COLLECTOR_CONFIG:-/etc/otel/config.yaml}"
if [ ! -f "$CONFIG_FILE" ]; then
    log "ERROR: Configuration file not found: $CONFIG_FILE"
    exit 1
fi

# Set up feature gates if specified
if [ -n "$NRDOT_FEATURE_GATES" ]; then
    export OTEL_COLLECTOR_FEATURE_GATES="$NRDOT_FEATURE_GATES"
fi

# Configure memory limits
if [ -n "$NRDOT_MEM_LIMIT_MIB" ]; then
    export GOMEMLIMIT="${NRDOT_MEM_LIMIT_MIB}MiB"
fi

# Enable debug logging if requested
if [ "$NRDOT_DEBUG" = "true" ]; then
    export OTEL_LOG_LEVEL="debug"
fi

# Create checkpoint directory if it doesn't exist
if [ -n "$NRDOT_CHECKPOINT_DIR" ]; then
    mkdir -p "$NRDOT_CHECKPOINT_DIR"
fi

log "Starting NRDOT Collector..."
log "Configuration: $CONFIG_FILE"
log "Checkpoint directory: ${NRDOT_CHECKPOINT_DIR:-none}"
log "Memory limit: ${GOMEMLIMIT:-default}"
log "Log level: ${OTEL_LOG_LEVEL:-info}"

# Start the collector in the background
/usr/local/bin/nrdot-collector "$@" &
PID=$!

# Wait for the collector to start
sleep 2

# Check if collector started successfully
if ! kill -0 "$PID" 2>/dev/null; then
    log "ERROR: Collector failed to start"
    exit 1
fi

log "Collector started successfully (PID: $PID)"

# Monitor the collector process
while true; do
    if ! kill -0 "$PID" 2>/dev/null; then
        log "Collector process terminated unexpectedly"
        exit 1
    fi
    sleep 5
done