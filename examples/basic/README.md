# Basic NRDOT-HOST Example

This example demonstrates a basic NRDOT-HOST deployment suitable for development, testing, and small-scale production use.

## Overview

This configuration sets up:
- HTTP webhook receiver for incoming data
- File watcher for batch processing
- Data validation and enrichment pipeline
- Console and file outputs
- Basic monitoring and health checks

## Prerequisites

- Docker and Docker Compose installed
- At least 2GB of available RAM
- Port 8080 available

## Quick Start

1. **Clone the configuration**:
   ```bash
   cp -r examples/basic /path/to/your/deployment
   cd /path/to/your/deployment
   ```

2. **Create data directories**:
   ```bash
   mkdir -p data/input data/output
   ```

3. **Start the services**:
   ```bash
   docker-compose up -d
   ```

4. **Verify deployment**:
   ```bash
   # Check health
   curl http://localhost:8080/health
   
   # View metrics
   curl http://localhost:8080/metrics
   ```

## Configuration Details

### Data Sources

1. **HTTP Webhook** (`/webhook`):
   - Accepts POST requests with JSON payloads
   - No authentication (configure for production!)
   - Example:
     ```bash
     curl -X POST http://localhost:8080/webhook \
       -H "Content-Type: application/json" \
       -d '{"id": "123", "timestamp": "2024-01-01T00:00:00Z", "payload": {"message": "test"}}'
     ```

2. **File Watcher**:
   - Monitors `/data/input` directory
   - Processes `*.json` files
   - Polls every 10 seconds

### Processing Pipeline

1. **Validator**: Ensures data has required fields (`id`, `timestamp`)
2. **Enricher**: Adds metadata (processing time, environment)
3. **Transformer**: Restructures data for output format

### Outputs

1. **Console**: Pretty-printed JSON for debugging
2. **File**: JSONL format with rotation (100MB or 24 hours)

## Customization

### Adding Kafka Source

Add to `sources` section:
```yaml
- name: "kafka-events"
  type: "kafka"
  config:
    brokers: ["kafka:9092"]
    topics: ["events"]
    consumer_group: "nrdot-basic"
```

### Adding Elasticsearch Output

Add to `outputs` section:
```yaml
- name: "elasticsearch"
  type: "elasticsearch"
  config:
    hosts: ["http://elasticsearch:9200"]
    index: "nrdot-events"
    bulk_size: 1000
    flush_interval: "5s"
```

### Enabling Authentication

Update HTTP source:
```yaml
config:
  endpoint: "/webhook"
  method: "POST"
  auth:
    type: "bearer"
    token: "${WEBHOOK_TOKEN}"
```

## Production Considerations

1. **Security**:
   - Enable authentication on all endpoints
   - Use HTTPS with proper certificates
   - Restrict CORS origins

2. **Performance**:
   - Increase resource limits based on load
   - Tune garbage collection settings
   - Enable connection pooling

3. **Persistence**:
   - Configure PostgreSQL for state storage
   - Use Redis for caching
   - Set up proper backup strategies

## Monitoring

- **Health Check**: `http://localhost:8080/health`
- **Metrics**: `http://localhost:8080/metrics` (Prometheus format)
- **Logs**: `docker-compose logs -f nrdot-host`

## Troubleshooting

### Container won't start
```bash
# Check logs
docker-compose logs nrdot-host

# Verify configuration
docker-compose config
```

### High memory usage
- Adjust `resources.max_memory` in config
- Tune `gc_percent` for more aggressive garbage collection

### Slow processing
- Check processor pipeline for bottlenecks
- Monitor metrics for queue buildup
- Consider horizontal scaling

## Next Steps

- Review security settings for production use
- Set up proper monitoring and alerting
- Configure persistent storage
- Implement proper backup procedures
- Consider Kubernetes deployment for scaling