#!/bin/bash
# Startup script for data-collector service
# Runs both main application and health check server

set -e

echo "Starting data-collector service..."

# Start health check server in background
echo "Starting health check server on port 8081..."
python health_server.py &
HEALTH_PID=$!

# Give health server time to start
sleep 2

# Function to handle shutdown
shutdown() {
    echo "Shutting down..."
    kill $HEALTH_PID 2>/dev/null || true
    exit 0
}

# Register shutdown handlers
trap shutdown SIGTERM SIGINT

# Start main application with New Relic
echo "Starting main application on port 8080..."
if [ -n "$NEW_RELIC_LICENSE_KEY" ]; then
    exec newrelic-admin run-program gunicorn \
        --bind 0.0.0.0:8080 \
        --workers ${GUNICORN_WORKERS:-4} \
        --threads ${GUNICORN_THREADS:-4} \
        --timeout 30 \
        --access-logfile - \
        --error-logfile - \
        --log-level ${LOG_LEVEL:-info} \
        app:app
else
    echo "WARNING: NEW_RELIC_LICENSE_KEY not set, running without APM"
    exec gunicorn \
        --bind 0.0.0.0:8080 \
        --workers ${GUNICORN_WORKERS:-4} \
        --threads ${GUNICORN_THREADS:-4} \
        --timeout 30 \
        --access-logfile - \
        --error-logfile - \
        --log-level ${LOG_LEVEL:-info} \
        app:app
fi