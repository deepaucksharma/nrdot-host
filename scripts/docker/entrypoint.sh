#!/bin/sh
# NRDOT-HOST Docker entrypoint script

set -e

# Function to log messages
log() {
    echo "[entrypoint] $1"
}

# Check if running as root (needed for some operations)
if [ "$(id -u)" = "0" ]; then
    log "Running as root, will switch to nrdot user after setup"
    
    # Fix permissions if needed
    chown -R nrdot:nrdot /var/lib/nrdot /var/log/nrdot || true
    
    # Drop to nrdot user
    exec su-exec nrdot "$0" "$@"
fi

# Setup configuration
if [ ! -f "$NRDOT_CONFIG" ]; then
    log "No configuration found, copying default"
    cp /etc/nrdot/config.yaml.default "$NRDOT_CONFIG"
fi

# Handle license key from environment
if [ -n "$NEW_RELIC_LICENSE_KEY" ]; then
    log "Setting license key from environment"
    # Use sed to replace the license key in config
    sed -i "s/license_key:.*/license_key: \"$NEW_RELIC_LICENSE_KEY\"/" "$NRDOT_CONFIG"
fi

# Handle other environment variables
if [ -n "$NRDOT_ENVIRONMENT" ]; then
    sed -i "s/environment:.*/environment: \"$NRDOT_ENVIRONMENT\"/" "$NRDOT_CONFIG"
fi

if [ -n "$NRDOT_SERVICE_NAME" ]; then
    sed -i "s/name:.*/name: \"$NRDOT_SERVICE_NAME\"/" "$NRDOT_CONFIG"
fi

# Set log level if specified
if [ -n "$NRDOT_LOG_LEVEL" ]; then
    export LOG_LEVEL="$NRDOT_LOG_LEVEL"
fi

# Handle auto-config enable/disable
if [ "$NRDOT_AUTO_CONFIG" = "false" ]; then
    log "Disabling auto-configuration"
    sed -i "s/enabled: true/enabled: false/" "$NRDOT_CONFIG"
fi

# Service credentials from environment
# MySQL
if [ -n "$MYSQL_HOST" ]; then
    export MYSQL_MONITOR_USER="${MYSQL_MONITOR_USER:-monitoring}"
    export MYSQL_MONITOR_PASS="${MYSQL_MONITOR_PASS:-}"
    log "MySQL credentials configured"
fi

# PostgreSQL
if [ -n "$POSTGRES_HOST" ]; then
    export POSTGRES_MONITOR_USER="${POSTGRES_MONITOR_USER:-monitoring}"
    export POSTGRES_MONITOR_PASS="${POSTGRES_MONITOR_PASS:-}"
    export POSTGRES_MONITOR_DB="${POSTGRES_MONITOR_DB:-postgres}"
    log "PostgreSQL credentials configured"
fi

# Redis
if [ -n "$REDIS_HOST" ]; then
    export REDIS_PASSWORD="${REDIS_PASSWORD:-}"
    log "Redis credentials configured"
fi

# MongoDB
if [ -n "$MONGODB_HOST" ]; then
    export MONGODB_MONITOR_USER="${MONGODB_MONITOR_USER:-monitoring}"
    export MONGODB_MONITOR_PASS="${MONGODB_MONITOR_PASS:-}"
    log "MongoDB credentials configured"
fi

# Elasticsearch
if [ -n "$ELASTICSEARCH_HOST" ]; then
    export ELASTICSEARCH_USER="${ELASTICSEARCH_USER:-elastic}"
    export ELASTICSEARCH_PASS="${ELASTICSEARCH_PASS:-}"
    log "Elasticsearch credentials configured"
fi

# RabbitMQ
if [ -n "$RABBITMQ_HOST" ]; then
    export RABBITMQ_USER="${RABBITMQ_USER:-guest}"
    export RABBITMQ_PASS="${RABBITMQ_PASS:-guest}"
    log "RabbitMQ credentials configured"
fi

# Check if we need to wait for services
if [ -n "$WAIT_FOR_SERVICES" ]; then
    log "Waiting for services: $WAIT_FOR_SERVICES"
    for service in $(echo "$WAIT_FOR_SERVICES" | tr ',' ' '); do
        host=$(echo "$service" | cut -d: -f1)
        port=$(echo "$service" | cut -d: -f2)
        
        log "Waiting for $host:$port..."
        while ! nc -z "$host" "$port" 2>/dev/null; do
            sleep 1
        done
        log "$host:$port is available"
    done
fi

# Run pre-discovery if requested
if [ "$RUN_DISCOVERY_FIRST" = "true" ]; then
    log "Running initial service discovery"
    /usr/local/bin/nrdot-host discover || true
fi

# Execute nrdot-host with arguments
log "Starting NRDOT-HOST with args: $@"
exec /usr/local/bin/nrdot-host "$@"