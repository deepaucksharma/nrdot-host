# otel-processor-nrcap

OpenTelemetry processor that provides cardinality protection for metrics.

## Overview
Monitors and controls metric cardinality to prevent cost overruns and performance issues from high-cardinality data.

## Features
- Per-metric cardinality limits
- Global cardinality limit enforcement
- Multiple limiting strategies (drop, aggregate, sample, oldest)
- High-cardinality label detection and filtering
- Time-based cardinality windows
- Memory-efficient tracking using xxhash
- Configurable reset intervals
- Cardinality statistics reporting

## Configuration
```yaml
processors:
  nrcap:
    # Global cardinality limit
    global_limit: 100000
    
    # Per-metric limits
    metric_limits:
      http_requests_total: 10000
      db_connections: 5000
      process.cpu.time: 1000
      custom.metric: 5000
    
    # Default limit for unlisted metrics
    default_limit: 1000
    
    # Limiting strategy: drop, aggregate, sample, oldest
    strategy: drop
    
    # High-cardinality labels to remove
    deny_labels:
      - request_id
      - session_id
      - trace_id
    
    # Labels to always keep
    allow_labels:
      - service
      - environment
      - host
    
    # Reset interval for cardinality tracking
    reset_interval: 1h
    
    # Enable cardinality statistics
    enable_stats: true
```

## Limiting Strategies

- **drop**: Drop metrics that exceed cardinality limit
- **aggregate**: Remove labels to reduce cardinality
- **sample**: Randomly sample metrics over the limit
- **oldest**: Drop oldest label combinations

## Usage

Add the processor to your OpenTelemetry Collector configuration:

```yaml
receivers:
  otlp:
    protocols:
      grpc:

processors:
  nrcap:
    global_limit: 50000
    strategy: aggregate

exporters:
  otlp:
    endpoint: localhost:4317

service:
  pipelines:
    metrics:
      receivers: [otlp]
      processors: [nrcap]
      exporters: [otlp]
```

## Integration
- Works with `nrdot-cost-calculator`
- Alerts via `nrdot-telemetry-client`
