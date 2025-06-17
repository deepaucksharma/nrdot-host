# NRDOT-HOST Performance Tuning Guide

This guide helps you optimize NRDOT-HOST for maximum performance and efficiency.

## Table of Contents

- [Performance Overview](#performance-overview)
- [Baseline Performance](#baseline-performance)
- [System Requirements](#system-requirements)
- [Configuration Tuning](#configuration-tuning)
- [Processor Optimization](#processor-optimization)
- [Memory Management](#memory-management)
- [CPU Optimization](#cpu-optimization)
- [Network Optimization](#network-optimization)
- [Monitoring Performance](#monitoring-performance)
- [Troubleshooting Performance Issues](#troubleshooting-performance-issues)
- [Best Practices](#best-practices)

## Performance Overview

NRDOT-HOST is designed to handle high-volume telemetry data efficiently. Key performance characteristics:

- **Throughput**: 1M+ data points/second per instance
- **Latency**: <1ms processing time (P99)
- **Memory**: 256MB-1GB typical usage
- **CPU**: 1-4 cores depending on load

### Performance Factors

1. **Data Volume**: Number of metrics, traces, logs per second
2. **Cardinality**: Number of unique time series
3. **Processing Complexity**: Enabled processors and their configuration
4. **Network**: Bandwidth and latency to exporters
5. **Hardware**: CPU, memory, disk I/O capabilities

## Baseline Performance

### Test Environment

```yaml
# Test configuration
Hardware:
  CPU: 4 cores (Intel Xeon E5-2686 v4)
  Memory: 8GB
  Network: 10 Gbps
  
Configuration:
  Collectors: 1
  Processors: All enabled
  Exporters: New Relic OTLP
```

### Performance Results

| Metric Type | Rate (per second) | CPU Usage | Memory Usage | P99 Latency |
|-------------|-------------------|-----------|--------------|-------------|
| Metrics | 1,000,000 | 85% | 512MB | 0.8ms |
| Traces | 50,000 | 45% | 384MB | 1.2ms |
| Logs | 200,000 | 60% | 448MB | 0.9ms |
| Combined | 500K/25K/100K | 90% | 768MB | 1.5ms |

## System Requirements

### Minimum Requirements

```yaml
# For <100K metrics/second
CPU: 1 core
Memory: 256MB
Disk: 1GB (for queuing)
Network: 1 Mbps
```

### Recommended Requirements

```yaml
# For 100K-1M metrics/second
CPU: 2-4 cores
Memory: 512MB-1GB
Disk: 10GB SSD (for queuing)
Network: 10 Mbps
```

### High-Volume Requirements

```yaml
# For >1M metrics/second
CPU: 4-8 cores
Memory: 2-4GB
Disk: 50GB SSD (for queuing)
Network: 100 Mbps
```

## Configuration Tuning

### Basic Optimization

```yaml
# Optimized for throughput
service:
  telemetry:
    metrics:
      level: none  # Disable self-telemetry

processors:
  batch:
    size: 2000        # Larger batches
    timeout: 200ms    # Lower timeout
    
  memory_limiter:
    check_interval: 1s
    limit_mib: 1024
    spike_limit_mib: 256

exporters:
  newrelic:
    compression: gzip  # Enable compression
    retry_on_failure:
      enabled: true
      initial_interval: 1s
      max_interval: 30s
```

### High-Performance Configuration

```yaml
# Maximum performance configuration
service:
  extensions: [memory_ballast]
  
extensions:
  memory_ballast:
    size_mib: 256  # Pre-allocate memory

processors:
  # Disable expensive processors if not needed
  nrsecurity:
    enabled: false  # If secrets already redacted
    
  nrenrich:
    host_metadata:
      enabled: true
    kubernetes:
      enabled: false  # If not in K8s
      
  # Optimize batch processor
  batch:
    size: 5000
    timeout: 100ms
    send_batch_max_size: 5000
    
  # Aggressive memory limits
  memory_limiter:
    check_interval: 500ms
    limit_percentage: 80
    spike_limit_percentage: 10

# Optimize receivers
receivers:
  prometheus:
    config:
      global:
        scrape_interval: 60s  # Reduce frequency
        scrape_timeout: 10s
        
  otlp:
    protocols:
      grpc:
        max_recv_msg_size_mib: 32
        max_concurrent_streams: 100
        keepalive:
          server_parameters:
            max_connection_idle: 60s
            max_connection_age: 90s

# Optimize exporters
exporters:
  newrelic:
    timeout: 30s
    sending_queue:
      enabled: true
      num_consumers: 10
      queue_size: 10000
    retry_on_failure:
      enabled: true
      initial_interval: 1s
      max_interval: 10s
      max_elapsed_time: 60s
```

### Low-Latency Configuration

```yaml
# Optimize for low latency
processors:
  batch:
    size: 100         # Smaller batches
    timeout: 10ms     # Very low timeout
    
  # Disable complex processors
  nrtransform:
    enabled: false
    
  # Simple cardinality limits
  nrcap:
    strategies:
      overflow_action: "drop"  # Fast dropping

exporters:
  newrelic:
    timeout: 5s
    sending_queue:
      enabled: false  # Direct sending
```

## Processor Optimization

### Security Processor (nrsecurity)

```yaml
processors:
  nrsecurity:
    # Use compiled regex for better performance
    redact_secrets:
      patterns:
        passwords: true
        api_keys: true
        # Disable expensive patterns
        credit_cards: false
        ssn: false
        
    # Limit custom patterns
    custom_patterns:
      - name: "api_key"
        pattern: 'X-API-Key:\s*\S+'  # Simple pattern
        
    # Disable if not needed
    redact_pii:
      enabled: false
```

**Performance Tips:**
- Pre-compile regex patterns
- Use simple patterns over complex ones
- Disable unused redaction types
- Consider pre-processing at source

### Enrichment Processor (nrenrich)

```yaml
processors:
  nrenrich:
    # Cache metadata aggressively
    host_metadata:
      enabled: true
      cache_duration: 24h
      
    # Disable expensive lookups
    cloud:
      enabled: true
      cache_duration: 1h
      timeout: 100ms  # Fast timeout
      
    # Optimize K8s metadata
    kubernetes:
      enabled: true
      cache_duration: 5m
      # Limit what to fetch
      pod:
        labels: true
        annotations: false  # Often large
      filters:
        include_labels:
          - "app"
          - "version"
```

**Performance Tips:**
- Use long cache durations for static data
- Disable unnecessary metadata collection
- Filter labels/annotations aggressively
- Set fast timeouts for external calls

### Transform Processor (nrtransform)

```yaml
processors:
  nrtransform:
    # Limit expensive calculations
    metrics:
      calculations:
        # Pre-calculate at fixed intervals
        - name: "error_rate"
          expression: "errors / requests * 100"
          cache_duration: 60s
          
    # Disable unused transformations
    traces:
      enabled: false
      
    # Optimize conversions
    metrics:
      conversions:
        # Batch similar conversions
        - metrics: ["*.bytes"]
          from: "bytes"
          to: "megabytes"
          batch: true
```

**Performance Tips:**
- Cache calculation results
- Batch similar operations
- Avoid complex expressions
- Pre-calculate when possible

### Cardinality Processor (nrcap)

```yaml
processors:
  nrcap:
    # Optimize tracking
    cache:
      size: 50000        # Smaller cache
      ttl: 15m          # Shorter TTL
      cleanup_interval: 5m
      
    # Fast overflow handling
    strategies:
      overflow_action: "drop"  # Fastest
      
    # Efficient dimension reduction
    strategies:
      dimension_reduction:
        rules:
          - dimension: "request_id"
            action: "drop"  # Don't track
```

**Performance Tips:**
- Use smaller caches for better CPU cache usage
- Drop instead of aggregate for speed
- Remove high-cardinality dimensions early
- Use probabilistic data structures

## Memory Management

### Memory Configuration

```yaml
# Memory optimization
extensions:
  memory_ballast:
    size_mib: 512  # 25% of available memory

processors:
  memory_limiter:
    check_interval: 1s
    limit_percentage: 75
    spike_limit_percentage: 20

service:
  telemetry:
    metrics:
      level: none  # Reduce memory for self-telemetry
```

### Garbage Collection Tuning

```bash
# Environment variables for GC tuning
export GOGC=200           # Less frequent GC
export GOMEMLIMIT=1GiB    # Hard memory limit
export GOMAXPROCS=4       # Match CPU cores

# For low latency
export GOGC=50            # More frequent GC
export GODEBUG=gctrace=1  # GC debugging
```

### Memory Profiling

```bash
# Enable memory profiling
nrdot-ctl profile memory --output mem.prof

# Analyze allocation
go tool pprof -alloc_space mem.prof

# Find memory leaks
go tool pprof -inuse_space mem.prof
```

## CPU Optimization

### CPU Affinity

```bash
# Pin to specific CPUs
taskset -c 0-3 nrdot-supervisor

# Or in systemd
[Service]
CPUAffinity=0-3
```

### Concurrency Tuning

```yaml
# Optimize concurrency
receivers:
  otlp:
    protocols:
      grpc:
        max_concurrent_streams: 100
        
processors:
  batch:
    send_batch_max_size: 5000
    
exporters:
  newrelic:
    sending_queue:
      num_consumers: 10  # Match CPU cores
```

### CPU Profiling

```bash
# Profile CPU usage
nrdot-ctl profile cpu --duration 60s --output cpu.prof

# Analyze hot paths
go tool pprof -http=:8080 cpu.prof

# Top functions
go tool pprof -top cpu.prof
```

## Network Optimization

### Compression

```yaml
# Enable compression
exporters:
  newrelic:
    compression: gzip
    
receivers:
  otlp:
    protocols:
      grpc:
        compression: gzip
```

### Batching

```yaml
# Optimize batch sizes for network
processors:
  batch:
    size: 1000            # ~100KB per batch
    timeout: 200ms
    send_batch_size: 1000
```

### Connection Pooling

```yaml
exporters:
  newrelic:
    connection:
      max_idle_conns: 10
      max_conns_per_host: 10
      idle_conn_timeout: 90s
```

### TCP Tuning

```bash
# System-level TCP tuning
sysctl -w net.core.rmem_max=134217728
sysctl -w net.core.wmem_max=134217728
sysctl -w net.ipv4.tcp_rmem="4096 87380 134217728"
sysctl -w net.ipv4.tcp_wmem="4096 65536 134217728"
```

## Monitoring Performance

### Key Metrics to Monitor

```yaml
# Prometheus metrics
# Throughput
rate(otelcol_processor_processed_total[5m])

# Latency
histogram_quantile(0.99, otelcol_processor_latency_bucket)

# Memory usage
process_resident_memory_bytes

# CPU usage
rate(process_cpu_seconds_total[5m])

# Queue size
otelcol_exporter_queue_size

# Dropped data
rate(otelcol_processor_dropped_total[5m])
```

### Performance Dashboard

```json
{
  "dashboard": {
    "title": "NRDOT Performance",
    "panels": [
      {
        "title": "Throughput",
        "targets": [
          {
            "expr": "rate(otelcol_receiver_accepted_total[5m])"
          }
        ]
      },
      {
        "title": "Processing Latency",
        "targets": [
          {
            "expr": "histogram_quantile(0.99, otelcol_processor_latency_bucket)"
          }
        ]
      },
      {
        "title": "Resource Usage",
        "targets": [
          {
            "expr": "process_resident_memory_bytes"
          },
          {
            "expr": "rate(process_cpu_seconds_total[5m])"
          }
        ]
      }
    ]
  }
}
```

### Alerting Rules

```yaml
groups:
  - name: nrdot_performance
    rules:
      - alert: HighProcessingLatency
        expr: histogram_quantile(0.99, otelcol_processor_latency_bucket) > 0.005
        for: 5m
        annotations:
          summary: "High processing latency detected"
          
      - alert: HighMemoryUsage
        expr: process_resident_memory_bytes > 1073741824
        for: 5m
        annotations:
          summary: "Memory usage exceeds 1GB"
          
      - alert: HighDropRate
        expr: rate(otelcol_processor_dropped_total[5m]) > 100
        for: 5m
        annotations:
          summary: "High data drop rate"
```

## Troubleshooting Performance Issues

### High CPU Usage

1. **Check Processing Rate**
   ```bash
   nrdot-ctl metrics throughput
   ```

2. **Profile CPU**
   ```bash
   nrdot-ctl profile cpu --duration 60s
   ```

3. **Common Causes**:
   - Complex regex patterns in nrsecurity
   - High cardinality in metrics
   - Inefficient custom patterns
   - Too many small batches

### High Memory Usage

1. **Check Memory Stats**
   ```bash
   nrdot-ctl metrics memory
   ```

2. **Common Causes**:
   - Large queue sizes
   - Memory leaks in processors
   - Cardinality explosion
   - Large batch sizes

3. **Solutions**:
   ```yaml
   # Reduce memory usage
   processors:
     memory_limiter:
       limit_mib: 512
       
     nrcap:
       cache:
         size: 10000  # Smaller cache
   ```

### High Latency

1. **Check Pipeline Latency**
   ```bash
   nrdot-ctl metrics latency --per-processor
   ```

2. **Common Causes**:
   - Slow external calls (enrichment)
   - Large batch timeout
   - Network latency
   - Queue backpressure

### Data Loss

1. **Check Drop Metrics**
   ```bash
   nrdot-ctl metrics drops
   ```

2. **Common Causes**:
   - Cardinality limits hit
   - Queue overflow
   - Memory pressure
   - Network timeouts

## Best Practices

### General Guidelines

1. **Start Simple**
   - Enable only needed processors
   - Add features incrementally
   - Monitor impact of changes

2. **Profile Before Optimizing**
   - Measure baseline performance
   - Identify actual bottlenecks
   - Validate improvements

3. **Resource Planning**
   ```yaml
   # Plan for peak load + 20%
   Peak metrics/sec: 100,000
   Peak CPU usage: 2 cores
   Allocated CPU: 2.4 cores (20% buffer)
   ```

### Configuration Best Practices

1. **Batch Optimization**
   ```yaml
   # Balance latency vs throughput
   processors:
     batch:
       size: 1000         # Start here
       timeout: 200ms     # Adjust based on SLA
   ```

2. **Memory Management**
   ```yaml
   # Always use memory limiter
   processors:
     memory_limiter:
       check_interval: 1s
       limit_percentage: 80
   ```

3. **Queue Sizing**
   ```yaml
   # Size for burst handling
   exporters:
     newrelic:
       sending_queue:
         queue_size: 10000  # 10s of data at 1K/s
   ```

### Scaling Strategy

1. **Vertical Scaling**
   - Add CPU cores
   - Increase memory
   - Upgrade network
   - Use faster disks

2. **Horizontal Scaling**
   - Deploy multiple collectors
   - Use load balancers
   - Partition by data type
   - Regional deployment

3. **Data Reduction**
   - Drop unnecessary metrics
   - Reduce cardinality
   - Increase aggregation intervals
   - Sample traces

### Monitoring Checklist

- [ ] Set up performance dashboards
- [ ] Configure alerts for key metrics
- [ ] Enable profiling endpoints
- [ ] Regular performance reviews
- [ ] Capacity planning updates

## Performance Testing

### Load Testing

```bash
# Generate load for testing
nrdot-ctl loadtest \
  --metrics-per-second 100000 \
  --duration 5m \
  --cardinality 10000
```

### Benchmark Suite

```bash
# Run performance benchmarks
cd otel-processor-nrsecurity
go test -bench=. -benchmem -benchtime=10s

# Compare results
benchstat baseline.txt optimized.txt
```

### Stress Testing

```bash
# Stress test configuration
nrdot-ctl stress \
  --config stress-test.yaml \
  --ramp-time 30s \
  --sustained-time 5m \
  --metrics-output stress-results.json
```

## Reference Configurations

### Low Volume (<10K metrics/sec)

```yaml
processors:
  batch:
    size: 500
    timeout: 500ms
  memory_limiter:
    limit_mib: 256
```

### Medium Volume (10K-100K metrics/sec)

```yaml
processors:
  batch:
    size: 1000
    timeout: 200ms
  memory_limiter:
    limit_mib: 512
```

### High Volume (>100K metrics/sec)

```yaml
processors:
  batch:
    size: 5000
    timeout: 100ms
  memory_limiter:
    limit_mib: 2048
```

## Additional Resources

- [Processor Benchmarks](./benchmarks/)
- [Performance Test Results](./performance-tests/)
- [Scaling Case Studies](./case-studies/)
- [OpenTelemetry Performance Docs](https://opentelemetry.io/docs/reference/specification/performance/)