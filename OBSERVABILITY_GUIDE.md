# NRDOT-HOST Observability Guide

## Overview

This guide provides comprehensive information on monitoring NRDOT-HOST itself, understanding the telemetry it produces, and building effective dashboards and alerts.

## NRDOT-HOST Self-Monitoring

### Internal Metrics

NRDOT-HOST exposes its own operational metrics:

```yaml
# Enable self-monitoring in config
telemetry:
  enabled: true
  level: detailed
  metrics:
    address: ":8888"  # Prometheus metrics endpoint
```

#### Key Internal Metrics

**Performance Metrics:**
- `nrdot_discovery_duration_seconds` - Time taken for service discovery
- `nrdot_config_generation_duration_seconds` - Config generation time
- `nrdot_process_collection_duration_seconds` - Process scanning time
- `nrdot_api_request_duration_seconds` - API request latency
- `nrdot_memory_usage_bytes` - Memory consumption
- `nrdot_cpu_usage_percent` - CPU utilization

**Operational Metrics:**
- `nrdot_discovered_services_total` - Number of discovered services
- `nrdot_config_reloads_total` - Configuration reload count
- `nrdot_export_success_total` - Successful exports to New Relic
- `nrdot_export_failure_total` - Failed exports
- `nrdot_process_monitored_total` - Number of monitored processes

**Error Metrics:**
- `nrdot_discovery_errors_total` - Discovery failures
- `nrdot_config_validation_errors_total` - Config validation errors
- `nrdot_export_retries_total` - Export retry attempts

### Health Endpoints

```bash
# Basic health check
curl http://localhost:8080/health

# Response
{
  "status": "healthy",
  "version": "3.0.0",
  "uptime_seconds": 3600,
  "last_discovery": "2024-01-15T10:30:00Z"
}

# Detailed health with subsystems
curl http://localhost:8080/health?detailed=true

# Response
{
  "status": "healthy",
  "version": "3.0.0",
  "subsystems": {
    "discovery": {
      "status": "healthy",
      "last_run": "2024-01-15T10:30:00Z",
      "services_found": 5
    },
    "exporter": {
      "status": "healthy",
      "success_rate": 0.99,
      "queue_size": 150
    },
    "config": {
      "status": "healthy",
      "last_reload": "2024-01-15T10:00:00Z",
      "validation_errors": 0
    }
  }
}
```

### Status API

```bash
# Comprehensive status
curl http://localhost:8080/v1/status

# Response
{
  "instance": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "hostname": "prod-server-01",
    "version": "3.0.0",
    "start_time": "2024-01-15T09:00:00Z"
  },
  "configuration": {
    "auto_config_enabled": true,
    "last_scan": "2024-01-15T10:30:00Z",
    "next_scan": "2024-01-15T10:35:00Z",
    "remote_config_enabled": true,
    "signing_enabled": true
  },
  "services": {
    "discovered": 5,
    "monitored": 5,
    "failed": 0,
    "list": [
      {
        "type": "mysql",
        "version": "8.0.32",
        "status": "healthy",
        "metrics_collected": 45
      }
    ]
  },
  "telemetry": {
    "metrics_exported": 150000,
    "export_errors": 2,
    "last_export": "2024-01-15T10:34:30Z",
    "queue_size": 150
  }
}
```

## Telemetry Data Model

### Metric Naming Convention

NRDOT-HOST follows OpenTelemetry semantic conventions:

```
{namespace}.{component}.{measurement}_{unit}

Examples:
- system.cpu.utilization_ratio
- system.memory.usage_bytes
- mysql.buffer_pool.pages_total
- process.cpu.time_seconds
```

### Resource Attributes

All telemetry includes standard resource attributes:

```json
{
  "resource": {
    "host.name": "prod-server-01",
    "host.id": "550e8400-e29b-41d4-a716-446655440000",
    "os.type": "linux",
    "os.description": "Ubuntu 22.04.3 LTS",
    "cloud.provider": "aws",
    "cloud.region": "us-east-1",
    "cloud.instance.id": "i-1234567890abcdef0",
    "service.name": "nrdot-host",
    "service.version": "3.0.0",
    "deployment.environment": "production"
  }
}
```

### Metric Categories

#### 1. Host Metrics
```yaml
# CPU Metrics
system.cpu.utilization       # CPU usage percentage
system.cpu.load_average.1m   # 1-minute load average
system.cpu.load_average.5m   # 5-minute load average
system.cpu.load_average.15m  # 15-minute load average

# Memory Metrics  
system.memory.usage          # Memory usage in bytes
system.memory.utilization    # Memory usage percentage
system.memory.available      # Available memory
system.memory.swap.usage     # Swap usage

# Disk Metrics
system.disk.io.read          # Disk read bytes/sec
system.disk.io.write         # Disk write bytes/sec
system.disk.operations.read  # Disk read ops/sec
system.disk.operations.write # Disk write ops/sec

# Network Metrics
system.network.io.receive    # Network receive bytes/sec
system.network.io.transmit   # Network transmit bytes/sec
system.network.packets.receive  # Packets received/sec
system.network.packets.transmit # Packets transmitted/sec
system.network.errors        # Network errors/sec
```

#### 2. Process Metrics
```yaml
# Per-process metrics (with process.name label)
process.cpu.utilization      # Process CPU usage
process.cpu.time            # CPU time consumed
process.memory.rss          # Resident memory
process.memory.vms          # Virtual memory
process.threads             # Thread count
process.open_files          # Open file descriptors
process.io.read_bytes       # Bytes read
process.io.write_bytes      # Bytes written

# Aggregate metrics
nrdot.processes.total       # Total process count
nrdot.processes.top_cpu     # Top N by CPU
nrdot.processes.top_memory  # Top N by memory
```

#### 3. Service-Specific Metrics

**MySQL:**
```yaml
mysql.buffer_pool.pages     # Buffer pool pages
mysql.buffer_pool.usage     # Buffer pool usage
mysql.connections.active    # Active connections
mysql.connections.max       # Max connections
mysql.queries.slow          # Slow queries/sec
mysql.operations.select     # SELECT ops/sec
mysql.operations.insert     # INSERT ops/sec
mysql.operations.update     # UPDATE ops/sec
mysql.operations.delete     # DELETE ops/sec
mysql.replication.lag       # Replication lag
```

**PostgreSQL:**
```yaml
postgresql.connections.active     # Active connections
postgresql.connections.idle       # Idle connections
postgresql.database.size         # Database size
postgresql.table.size           # Table sizes
postgresql.operations.commits    # Commits/sec
postgresql.operations.rollbacks  # Rollbacks/sec
postgresql.cache.hit_ratio      # Cache hit ratio
postgresql.replication.lag      # Replication lag
```

**Redis:**
```yaml
redis.connections.clients       # Connected clients
redis.memory.used              # Memory usage
redis.memory.fragmentation     # Fragmentation ratio
redis.operations.commands      # Commands/sec
redis.keys.total              # Total keys
redis.keys.expired            # Expired keys/sec
redis.persistence.rdb_saves   # RDB saves
redis.replication.lag        # Replication lag
```

## Dashboard Examples

### 1. NRDOT-HOST Operations Dashboard

```sql
-- NRDOT Health Score
SELECT 
  percentage(
    count(*), 
    WHERE status = 'healthy'
  ) as 'Health Score'
FROM NrdotStatus
SINCE 1 hour ago

-- Discovery Performance
SELECT 
  average(nrdot_discovery_duration_seconds) as 'Avg Discovery Time',
  max(nrdot_discovery_duration_seconds) as 'Max Discovery Time'
FROM Metric
WHERE metricName = 'nrdot_discovery_duration_seconds'
TIMESERIES AUTO

-- Export Success Rate
SELECT 
  rate(sum(nrdot_export_success_total), 1 minute) / 
  rate(sum(nrdot_export_success_total + nrdot_export_failure_total), 1 minute) * 100
  as 'Export Success Rate %'
FROM Metric
TIMESERIES AUTO

-- Memory Usage Trend
SELECT 
  average(nrdot_memory_usage_bytes) / 1024 / 1024 as 'Memory Usage (MB)'
FROM Metric
FACET hostname
TIMESERIES AUTO
```

### 2. Host Performance Dashboard

```sql
-- CPU Utilization Heatmap
SELECT 
  average(system.cpu.utilization) 
FROM Metric
FACET hostname
TIMESERIES 1 minute

-- Memory Pressure Indicators
SELECT 
  average(system.memory.utilization) as 'Memory %',
  average(system.memory.swap.usage) / 1024 / 1024 as 'Swap MB'
FROM Metric
WHERE system.memory.utilization > 80
FACET hostname
TIMESERIES AUTO

-- Disk I/O Patterns
SELECT 
  rate(sum(system.disk.io.read), 1 minute) as 'Read MB/s',
  rate(sum(system.disk.io.write), 1 minute) as 'Write MB/s'
FROM Metric
FACET device
TIMESERIES AUTO

-- Network Traffic
SELECT 
  rate(sum(system.network.io.receive), 1 minute) / 1024 / 1024 as 'Receive Mbps',
  rate(sum(system.network.io.transmit), 1 minute) / 1024 / 1024 as 'Transmit Mbps'
FROM Metric
FACET interface
TIMESERIES AUTO
```

### 3. Process Monitoring Dashboard

```sql
-- Top CPU Consumers
SELECT 
  average(process.cpu.utilization) as 'CPU %'
FROM Metric
WHERE process.name IS NOT NULL
FACET process.name
LIMIT 10

-- Top Memory Consumers  
SELECT 
  latest(process.memory.rss) / 1024 / 1024 as 'Memory MB'
FROM Metric
WHERE process.name IS NOT NULL
FACET process.name
LIMIT 10

-- Process Count Trends
SELECT 
  uniqueCount(process.pid) as 'Process Count'
FROM Metric
FACET process.name
WHERE process.name IN ('mysql', 'nginx', 'redis', 'postgres')
TIMESERIES AUTO

-- Thread Count Analysis
SELECT 
  sum(process.threads) as 'Total Threads',
  average(process.threads) as 'Avg Threads per Process'
FROM Metric
FACET process.name
TIMESERIES AUTO
```

### 4. Service Health Dashboard

```sql
-- MySQL Performance
SELECT 
  average(mysql.connections.active) as 'Active Connections',
  average(mysql.queries.slow) as 'Slow Queries/sec',
  average(mysql.buffer_pool.usage) as 'Buffer Pool Usage %'
FROM Metric
WHERE service.type = 'mysql'
TIMESERIES AUTO

-- PostgreSQL Performance
SELECT 
  average(postgresql.connections.active) as 'Active Connections',
  average(postgresql.cache.hit_ratio) * 100 as 'Cache Hit %',
  average(postgresql.operations.commits) as 'Commits/sec'
FROM Metric
WHERE service.type = 'postgresql'
TIMESERIES AUTO

-- Redis Performance
SELECT 
  average(redis.memory.used) / 1024 / 1024 as 'Memory MB',
  average(redis.operations.commands) as 'Commands/sec',
  average(redis.connections.clients) as 'Connected Clients'
FROM Metric
WHERE service.type = 'redis'
TIMESERIES AUTO
```

## Alert Conditions

### 1. NRDOT-HOST Health Alerts

```yaml
# NRDOT-HOST Down
- name: "NRDOT-HOST Service Down"
  query: |
    SELECT count(*)
    FROM NrdotHeartbeat
    WHERE hostname = '${hostname}'
  condition:
    threshold: 0
    duration: 5
    operator: "equals"
  priority: CRITICAL

# High Memory Usage
- name: "NRDOT-HOST High Memory"
  query: |
    SELECT average(nrdot_memory_usage_bytes) / 1024 / 1024
    FROM Metric
    WHERE hostname = '${hostname}'
  condition:
    threshold: 1500  # MB
    duration: 10
    operator: "above"
  priority: WARNING

# Export Failures
- name: "NRDOT-HOST Export Failures"
  query: |
    SELECT rate(sum(nrdot_export_failure_total), 5 minute)
    FROM Metric
  condition:
    threshold: 10
    duration: 5
    operator: "above"
  priority: WARNING
```

### 2. Host Resource Alerts

```yaml
# High CPU Usage
- name: "High CPU Utilization"
  query: |
    SELECT average(system.cpu.utilization)
    FROM Metric
    FACET hostname
  condition:
    threshold: 90
    duration: 10
    operator: "above"
  priority: WARNING

# Memory Exhaustion
- name: "Memory Exhaustion Warning"
  query: |
    SELECT average(system.memory.utilization)
    FROM Metric
    FACET hostname
  condition:
    threshold: 95
    duration: 5
    operator: "above"
  priority: CRITICAL

# Disk Space Low
- name: "Low Disk Space"
  query: |
    SELECT average(system.filesystem.utilization)
    FROM Metric
    FACET hostname, device
    WHERE device NOT LIKE '/dev/loop%'
  condition:
    threshold: 90
    duration: 5
    operator: "above"
  priority: WARNING
```

### 3. Service-Specific Alerts

```yaml
# MySQL Connection Exhaustion
- name: "MySQL Max Connections"
  query: |
    SELECT 
      average(mysql.connections.active) / average(mysql.connections.max) * 100
    FROM Metric
    FACET hostname
  condition:
    threshold: 90
    duration: 5
    operator: "above"
  priority: WARNING

# PostgreSQL Replication Lag
- name: "PostgreSQL Replication Lag"
  query: |
    SELECT max(postgresql.replication.lag)
    FROM Metric
    FACET hostname
  condition:
    threshold: 60  # seconds
    duration: 5
    operator: "above"
  priority: WARNING

# Redis Memory Limit
- name: "Redis Memory Critical"
  query: |
    SELECT average(redis.memory.used) / average(redis.memory.max) * 100
    FROM Metric
    FACET hostname
  condition:
    threshold: 90
    duration: 5
    operator: "above"
  priority: WARNING
```

## Troubleshooting with Telemetry

### 1. No Data Issues

```sql
-- Check NRDOT-HOST reporting
SELECT count(*)
FROM NrdotHeartbeat
WHERE hostname = 'problematic-host'
SINCE 1 hour ago

-- Check export errors
SELECT 
  sum(nrdot_export_failure_total) as 'Export Failures',
  latest(error_message) as 'Last Error'
FROM Metric
WHERE hostname = 'problematic-host'
SINCE 1 hour ago
```

### 2. Performance Issues

```sql
-- Identify slow discovery
SELECT 
  max(nrdot_discovery_duration_seconds) as 'Max Duration',
  average(nrdot_discovery_duration_seconds) as 'Avg Duration'
FROM Metric
FACET discovery_method
WHERE nrdot_discovery_duration_seconds > 5
SINCE 1 hour ago

-- Find memory leaks
SELECT 
  average(nrdot_memory_usage_bytes) / 1024 / 1024 as 'Memory MB'
FROM Metric
FACET hostname
TIMESERIES 5 minutes
SINCE 24 hours ago
```

### 3. Service Discovery Issues

```sql
-- Services not discovered
SELECT 
  uniqueCount(service.type) as 'Services Found'
FROM Metric
FACET hostname
COMPARE WITH 1 day ago

-- Discovery method effectiveness
SELECT 
  count(*) as 'Discoveries'
FROM NrdotDiscoveryEvent
FACET discovery_method, service_type
SINCE 24 hours ago
```

## Advanced Observability

### 1. Custom Metrics

Add custom metrics to your services:

```yaml
# In service configuration
receivers:
  prometheus:
    config:
      scrape_configs:
        - job_name: 'custom_app'
          static_configs:
            - targets: ['localhost:9090']
```

### 2. Distributed Tracing

Enable trace collection:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318
        
exporters:
  otlp/newrelic:
    endpoint: otlp.nr-data.net:4317
    headers:
      api-key: ${NEW_RELIC_LICENSE_KEY}
      
service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlp/newrelic]
```

### 3. Log Correlation

Correlate logs with metrics:

```yaml
# Log processing configuration
filelog:
  include: [/var/log/mysql/*.log]
  start_at: beginning
  operators:
    - type: regex_parser
      regex: '^(?P<time>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d+Z) \[(?P<level>\w+)\] (?P<message>.*)'
    - type: add_attributes
      attributes:
        service.type: mysql
        hostname: ${HOSTNAME}
```

## Performance Optimization

### 1. Metric Collection Tuning

```yaml
# Reduce collection frequency for stable metrics
receivers:
  hostmetrics:
    collection_interval: 300s  # 5 minutes for stable hosts
    scrapers:
      filesystem:
        collection_interval: 600s  # 10 minutes for disk metrics
```

### 2. Batching Configuration

```yaml
processors:
  batch:
    timeout: 30s           # Larger batches
    send_batch_size: 5000  # More metrics per batch
    send_batch_max_size: 10000
```

### 3. Filtering Unnecessary Metrics

```yaml
processors:
  filter:
    metrics:
      exclude:
        match_type: regexp
        metric_names:
          - system.disk.pending_operations
          - system.network.connections
```

## Compliance and Auditing

### 1. Audit Trail

```sql
-- Configuration changes
SELECT 
  timestamp,
  user,
  action,
  details
FROM NrdotAuditLog
WHERE action IN ('config_update', 'service_restart', 'discovery_manual')
SINCE 7 days ago
```

### 2. Compliance Metrics

```sql
-- Security compliance
SELECT 
  percentage(count(*), WHERE encryption_enabled = true) as 'Encrypted Connections',
  percentage(count(*), WHERE signature_valid = true) as 'Valid Signatures'
FROM NrdotServiceStatus
SINCE 1 day ago
```

## Conclusion

Effective observability of NRDOT-HOST involves:
1. Monitoring its internal health and performance
2. Understanding the telemetry data model
3. Building comprehensive dashboards
4. Setting up proactive alerts
5. Using telemetry for troubleshooting
6. Optimizing for performance
7. Maintaining compliance

Regular review and tuning of your observability setup ensures optimal performance and reliability of your monitoring infrastructure.