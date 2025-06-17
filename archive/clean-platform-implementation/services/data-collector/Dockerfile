# Use FIPS-compliant base image from newrelic-dokken
FROM cf-registry.nr-ops.net/newrelic/python-3.11-fips:latest

# Install security updates
USER root
RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt && \
    pip install newrelic psutil

# Copy application code
COPY . .

# Make startup script executable
RUN chmod +x start.sh

# Create non-root user (UID must be > 10000 for security policies)
RUN groupadd -r -g 10001 appuser && \
    useradd -r -u 10001 -g appuser appuser && \
    chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Health check endpoints
EXPOSE 8080 8081 9090

# Configure New Relic agent
ENV NEW_RELIC_CONFIG_FILE=newrelic.ini
ENV NEW_RELIC_ENVIRONMENT=production
ENV NEW_RELIC_LOG=stdout
ENV NEW_RELIC_LOG_LEVEL=info

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD python -c "import requests; requests.get('http://localhost:8081/healthz').raise_for_status()"

# Run startup script
CMD ["./start.sh"]