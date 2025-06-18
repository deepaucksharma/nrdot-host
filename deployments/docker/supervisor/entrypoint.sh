#!/bin/sh
set -e

# NRDOT Supervisor Entrypoint Script

# Function to log messages
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [SUPERVISOR] $1"
}

# Handle shutdown gracefully
shutdown() {
    log "Received shutdown signal, stopping supervisor..."
    
    # Send termination signal to all managed processes
    if [ -S "$NRDOT_SUPERVISOR_SOCKET" ]; then
        /usr/local/bin/nrdot-supervisor stop-all || true
    fi
    
    # Wait for processes to terminate
    sleep 2
    
    # Kill the supervisor
    kill -TERM "$PID" 2>/dev/null
    wait "$PID"
    exit 0
}

# Set up signal handlers
trap shutdown TERM INT

# Ensure socket directory exists
SOCKET_DIR=$(dirname "$NRDOT_SUPERVISOR_SOCKET")
if [ ! -d "$SOCKET_DIR" ]; then
    log "Creating socket directory: $SOCKET_DIR"
    mkdir -p "$SOCKET_DIR"
fi

# Ensure state directory exists
if [ ! -d "$NRDOT_SUPERVISOR_STATE_DIR" ]; then
    log "Creating state directory: $NRDOT_SUPERVISOR_STATE_DIR"
    mkdir -p "$NRDOT_SUPERVISOR_STATE_DIR"
fi

# Check configuration
CONFIG_FILE="${NRDOT_SUPERVISOR_CONFIG:-/etc/nrdot/supervisor.yaml}"
if [ -f "$CONFIG_FILE" ]; then
    log "Using configuration: $CONFIG_FILE"
    CONFIG_ARGS="--config $CONFIG_FILE"
else
    log "No configuration file found, using defaults"
    CONFIG_ARGS=""
fi

# Enable debug logging if requested
if [ "$NRDOT_DEBUG" = "true" ]; then
    export NRDOT_LOG_LEVEL="debug"
fi

log "Starting NRDOT Supervisor..."
log "Socket: $NRDOT_SUPERVISOR_SOCKET"
log "State directory: $NRDOT_SUPERVISOR_STATE_DIR"
log "Restart strategy: $NRDOT_RESTART_STRATEGY"
log "Max restarts: $NRDOT_MAX_RESTARTS"

# Start the supervisor
/usr/local/bin/nrdot-supervisor \
    --socket "$NRDOT_SUPERVISOR_SOCKET" \
    --state-dir "$NRDOT_SUPERVISOR_STATE_DIR" \
    --restart-strategy "$NRDOT_RESTART_STRATEGY" \
    --max-restarts "$NRDOT_MAX_RESTARTS" \
    $CONFIG_ARGS &

PID=$!

# Wait for the supervisor to start
sleep 2

# Check if supervisor started successfully
if ! kill -0 "$PID" 2>/dev/null; then
    log "ERROR: Supervisor failed to start"
    exit 1
fi

log "Supervisor started successfully (PID: $PID)"

# Monitor the supervisor process
while true; do
    if ! kill -0 "$PID" 2>/dev/null; then
        log "Supervisor process terminated unexpectedly"
        exit 1
    fi
    
    # Log supervisor status periodically
    if [ $(($(date +%s) % 60)) -eq 0 ]; then
        STATUS=$(/usr/local/bin/nrdot-supervisor status 2>/dev/null || echo "unknown")
        log "Status: $STATUS"
    fi
    
    sleep 5
done