#!/bin/sh
set -e

# NRDOT Privileged Helper Entrypoint Script
# WARNING: This runs as root and performs privileged operations

# Function to log messages
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [PRIVILEGED-HELPER] $1"
}

# Handle shutdown gracefully
shutdown() {
    log "Received shutdown signal, stopping privileged helper..."
    
    # Remove socket file
    rm -f "$NRDOT_PRIVILEGED_SOCKET"
    
    kill -TERM "$PID" 2>/dev/null
    wait "$PID"
    exit 0
}

# Set up signal handlers
trap shutdown TERM INT

# Security checks
if [ "$(id -u)" != "0" ]; then
    log "ERROR: Privileged helper must run as root"
    exit 1
fi

# Verify capabilities
if ! command -v capsh >/dev/null 2>&1; then
    log "WARNING: capsh not available, cannot verify capabilities"
else
    CAPS=$(capsh --print | grep "Current:" | cut -d' ' -f3-)
    log "Running with capabilities: $CAPS"
fi

# Ensure socket directory exists
SOCKET_DIR=$(dirname "$NRDOT_PRIVILEGED_SOCKET")
if [ ! -d "$SOCKET_DIR" ]; then
    log "Creating socket directory: $SOCKET_DIR"
    mkdir -p "$SOCKET_DIR"
    chmod 755 "$SOCKET_DIR"
fi

# Remove old socket if it exists
if [ -S "$NRDOT_PRIVILEGED_SOCKET" ]; then
    log "Removing old socket file"
    rm -f "$NRDOT_PRIVILEGED_SOCKET"
fi

# Set up audit logging
if [ -n "$NRDOT_PRIVILEGED_AUDIT_LOG" ]; then
    AUDIT_DIR=$(dirname "$NRDOT_PRIVILEGED_AUDIT_LOG")
    if [ ! -d "$AUDIT_DIR" ]; then
        mkdir -p "$AUDIT_DIR"
        chmod 700 "$AUDIT_DIR"
    fi
    touch "$NRDOT_PRIVILEGED_AUDIT_LOG"
    chmod 600 "$NRDOT_PRIVILEGED_AUDIT_LOG"
fi

# Enable debug logging if requested
if [ "$NRDOT_DEBUG" = "true" ]; then
    export NRDOT_LOG_LEVEL="debug"
fi

# Validate allowed UIDs
if [ -z "$NRDOT_PRIVILEGED_ALLOWED_UIDS" ]; then
    log "WARNING: No allowed UIDs specified, defaulting to 10001"
    NRDOT_PRIVILEGED_ALLOWED_UIDS="10001"
fi

log "Starting NRDOT Privileged Helper..."
log "Socket: $NRDOT_PRIVILEGED_SOCKET"
log "Allowed UIDs: $NRDOT_PRIVILEGED_ALLOWED_UIDS"
log "Max connections: $NRDOT_PRIVILEGED_MAX_CONNECTIONS"
log "Rate limit: $NRDOT_PRIVILEGED_RATE_LIMIT"
log "Audit log: ${NRDOT_PRIVILEGED_AUDIT_LOG:-disabled}"

# Start the privileged helper
/usr/local/bin/nrdot-privileged-helper \
    --socket "$NRDOT_PRIVILEGED_SOCKET" \
    --allowed-uids "$NRDOT_PRIVILEGED_ALLOWED_UIDS" \
    --max-connections "$NRDOT_PRIVILEGED_MAX_CONNECTIONS" \
    --timeout "$NRDOT_PRIVILEGED_TIMEOUT" \
    --rate-limit "$NRDOT_PRIVILEGED_RATE_LIMIT" \
    ${NRDOT_PRIVILEGED_AUDIT_LOG:+--audit-log "$NRDOT_PRIVILEGED_AUDIT_LOG"} &

PID=$!

# Wait for the helper to start
sleep 2

# Check if helper started successfully
if ! kill -0 "$PID" 2>/dev/null; then
    log "ERROR: Privileged helper failed to start"
    exit 1
fi

# Set socket permissions
if [ -S "$NRDOT_PRIVILEGED_SOCKET" ]; then
    chmod 660 "$NRDOT_PRIVILEGED_SOCKET"
    # If running in a container with the nrdot group, set group ownership
    if getent group nrdot >/dev/null 2>&1; then
        chgrp nrdot "$NRDOT_PRIVILEGED_SOCKET"
    fi
    log "Socket permissions configured"
fi

log "Privileged helper started successfully (PID: $PID)"

# Monitor the helper process
while true; do
    if ! kill -0 "$PID" 2>/dev/null; then
        log "Privileged helper process terminated unexpectedly"
        exit 1
    fi
    
    # Check socket exists
    if [ ! -S "$NRDOT_PRIVILEGED_SOCKET" ]; then
        log "ERROR: Socket file disappeared"
        exit 1
    fi
    
    sleep 5
done