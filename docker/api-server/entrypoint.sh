#!/bin/sh
set -e

# NRDOT API Server Entrypoint Script

# Function to log messages
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [API-SERVER] $1"
}

# Handle shutdown gracefully
shutdown() {
    log "Received shutdown signal, stopping API server..."
    kill -TERM "$PID" 2>/dev/null
    wait "$PID"
    exit 0
}

# Set up signal handlers
trap shutdown TERM INT

# Configure TLS if enabled
TLS_ARGS=""
if [ "$NRDOT_API_TLS_ENABLED" = "true" ]; then
    if [ ! -f "$NRDOT_API_TLS_CERT" ] || [ ! -f "$NRDOT_API_TLS_KEY" ]; then
        log "ERROR: TLS enabled but certificate or key not found"
        exit 1
    fi
    TLS_ARGS="--tls-cert $NRDOT_API_TLS_CERT --tls-key $NRDOT_API_TLS_KEY"
    log "TLS enabled"
fi

# Configure authentication if enabled
AUTH_ARGS=""
if [ "$NRDOT_API_AUTH_ENABLED" = "true" ]; then
    if [ -z "$NRDOT_API_AUTH_TOKEN" ] && [ ! -f "$NRDOT_API_AUTH_TOKEN_FILE" ]; then
        log "ERROR: Authentication enabled but no token provided"
        exit 1
    fi
    AUTH_ARGS="--auth-enabled"
    log "Authentication enabled"
fi

# Configure CORS
CORS_ARGS=""
if [ "$NRDOT_API_CORS_ENABLED" = "true" ]; then
    CORS_ARGS="--cors-enabled"
    if [ -n "$NRDOT_API_CORS_ORIGINS" ]; then
        CORS_ARGS="$CORS_ARGS --cors-origins $NRDOT_API_CORS_ORIGINS"
    fi
    log "CORS enabled"
fi

# Enable debug logging if requested
if [ "$NRDOT_DEBUG" = "true" ]; then
    export NRDOT_LOG_LEVEL="debug"
fi

# Configure connection to other components
COMPONENT_ARGS=""
if [ -n "$NRDOT_SUPERVISOR_SOCKET" ]; then
    COMPONENT_ARGS="$COMPONENT_ARGS --supervisor-socket $NRDOT_SUPERVISOR_SOCKET"
fi
if [ -n "$NRDOT_CONFIG_ENGINE_URL" ]; then
    COMPONENT_ARGS="$COMPONENT_ARGS --config-engine-url $NRDOT_CONFIG_ENGINE_URL"
fi

log "Starting NRDOT API Server..."
log "Listen address: $NRDOT_API_HOST:$NRDOT_API_PORT"
log "Metrics enabled: $NRDOT_API_METRICS_ENABLED"
log "Read timeout: $NRDOT_API_READ_TIMEOUT"
log "Write timeout: $NRDOT_API_WRITE_TIMEOUT"

# Start the API server
/usr/local/bin/nrdot-api-server \
    --host "$NRDOT_API_HOST" \
    --port "$NRDOT_API_PORT" \
    --read-timeout "$NRDOT_API_READ_TIMEOUT" \
    --write-timeout "$NRDOT_API_WRITE_TIMEOUT" \
    --idle-timeout "$NRDOT_API_IDLE_TIMEOUT" \
    --max-header-bytes "$NRDOT_API_MAX_HEADER_BYTES" \
    $TLS_ARGS \
    $AUTH_ARGS \
    $CORS_ARGS \
    $COMPONENT_ARGS &

PID=$!

# Wait for the server to start
sleep 2

# Check if server started successfully
if ! kill -0 "$PID" 2>/dev/null; then
    log "ERROR: API server failed to start"
    exit 1
fi

# Wait for server to be ready
MAX_ATTEMPTS=30
ATTEMPT=0
while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    if curl -sf "http://localhost:$NRDOT_API_PORT/health" >/dev/null 2>&1; then
        log "API server is ready"
        break
    fi
    ATTEMPT=$((ATTEMPT + 1))
    sleep 1
done

if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
    log "ERROR: API server failed to become ready"
    exit 1
fi

log "API server started successfully (PID: $PID)"

# Monitor the server process
while true; do
    if ! kill -0 "$PID" 2>/dev/null; then
        log "API server process terminated unexpectedly"
        exit 1
    fi
    sleep 5
done