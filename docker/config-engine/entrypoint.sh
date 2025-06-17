#!/bin/sh
set -e

# NRDOT Config Engine Entrypoint Script

# Function to log messages
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [CONFIG-ENGINE] $1"
}

# Handle shutdown gracefully
shutdown() {
    log "Received shutdown signal, stopping config engine..."
    kill -TERM "$PID" 2>/dev/null
    wait "$PID"
    exit 0
}

# Set up signal handlers
trap shutdown TERM INT

# Validate environment
if [ ! -d "$NRDOT_TEMPLATE_DIR" ]; then
    log "ERROR: Template directory not found: $NRDOT_TEMPLATE_DIR"
    exit 1
fi

# Create output directory if it doesn't exist
if [ ! -d "$NRDOT_OUTPUT_DIR" ]; then
    log "Creating output directory: $NRDOT_OUTPUT_DIR"
    mkdir -p "$NRDOT_OUTPUT_DIR"
fi

# Create cache directory if it doesn't exist
if [ ! -d "$NRDOT_CACHE_DIR" ]; then
    log "Creating cache directory: $NRDOT_CACHE_DIR"
    mkdir -p "$NRDOT_CACHE_DIR"
fi

# Check if config file exists
if [ ! -f "$NRDOT_CONFIG_PATH" ]; then
    log "WARNING: Configuration file not found: $NRDOT_CONFIG_PATH"
    log "Waiting for configuration to be provided..."
    
    # In watch mode, wait for config file
    if [ "$NRDOT_CONFIG_ENGINE_MODE" = "watch" ]; then
        while [ ! -f "$NRDOT_CONFIG_PATH" ]; do
            sleep 5
        done
        log "Configuration file detected: $NRDOT_CONFIG_PATH"
    else
        log "ERROR: Configuration file required in non-watch mode"
        exit 1
    fi
fi

# Enable debug logging if requested
if [ "$NRDOT_DEBUG" = "true" ]; then
    export NRDOT_LOG_LEVEL="debug"
fi

# Set up webhook server if enabled
WEBHOOK_ARGS=""
if [ "$NRDOT_WEBHOOK_ENABLED" = "true" ]; then
    WEBHOOK_ARGS="--webhook-port ${NRDOT_WEBHOOK_PORT:-8081}"
    log "Webhook server enabled on port ${NRDOT_WEBHOOK_PORT:-8081}"
fi

# Set up validation
VALIDATION_ARGS=""
if [ "$NRDOT_VALIDATION_ENABLED" = "true" ]; then
    VALIDATION_ARGS="--validate"
    log "Configuration validation enabled"
fi

log "Starting NRDOT Config Engine..."
log "Mode: $NRDOT_CONFIG_ENGINE_MODE"
log "Config path: $NRDOT_CONFIG_PATH"
log "Template directory: $NRDOT_TEMPLATE_DIR"
log "Output directory: $NRDOT_OUTPUT_DIR"
log "Watch interval: $NRDOT_WATCH_INTERVAL"

# Start the config engine based on mode
case "$NRDOT_CONFIG_ENGINE_MODE" in
    "watch")
        /usr/local/bin/nrdot-config-engine watch \
            --config "$NRDOT_CONFIG_PATH" \
            --template-dir "$NRDOT_TEMPLATE_DIR" \
            --output-dir "$NRDOT_OUTPUT_DIR" \
            --cache-dir "$NRDOT_CACHE_DIR" \
            --interval "$NRDOT_WATCH_INTERVAL" \
            $VALIDATION_ARGS \
            $WEBHOOK_ARGS &
        ;;
    "generate")
        /usr/local/bin/nrdot-config-engine generate \
            --config "$NRDOT_CONFIG_PATH" \
            --template-dir "$NRDOT_TEMPLATE_DIR" \
            --output-dir "$NRDOT_OUTPUT_DIR" \
            $VALIDATION_ARGS
        log "Configuration generated successfully"
        exit 0
        ;;
    "validate")
        /usr/local/bin/nrdot-config-engine validate \
            --config "$NRDOT_CONFIG_PATH"
        log "Configuration validated successfully"
        exit 0
        ;;
    *)
        log "ERROR: Unknown mode: $NRDOT_CONFIG_ENGINE_MODE"
        exit 1
        ;;
esac

PID=$!

# Wait for the engine to start
sleep 2

# Check if engine started successfully
if ! kill -0 "$PID" 2>/dev/null; then
    log "ERROR: Config engine failed to start"
    exit 1
fi

log "Config engine started successfully (PID: $PID)"

# Monitor the engine process
while true; do
    if ! kill -0 "$PID" 2>/dev/null; then
        log "Config engine process terminated unexpectedly"
        exit 1
    fi
    sleep 5
done