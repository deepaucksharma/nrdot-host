# NRDOT Processors Documentation

This guide covers the custom OpenTelemetry processors included in NRDOT-HOST.

## Table of Contents

- [Overview](#overview)
- [Processor Pipeline](#processor-pipeline)
- [Security Processor (nrsecurity)](#security-processor-nrsecurity)
- [Enrichment Processor (nrenrich)](#enrichment-processor-nrenrich)
- [Transform Processor (nrtransform)](#transform-processor-nrtransform)
- [Cardinality Processor (nrcap)](#cardinality-processor-nrcap)
- [Configuration Examples](#configuration-examples)
- [Performance Considerations](#performance-considerations)
- [Troubleshooting](#troubleshooting)

## Overview

NRDOT includes four custom processors that enhance OpenTelemetry's capabilities:

| Processor | Purpose | Position in Pipeline |
|-----------|---------|---------------------|
| **nrsecurity** | Secret redaction, PII protection | First (security) |
| **nrenrich** | Metadata addition, context enrichment | Second (context) |
| **nrtransform** | Metric calculations, unit conversions | Third (computation) |
| **nrcap** | Cardinality limiting, cost control | Last (protection) |

### Processing Order

```
Data In → nrsecurity → nrenrich → nrtransform → nrcap → Exporters
```

This order ensures:
1. Sensitive data is redacted first
2. Context is added before transformations
3. Calculations happen on enriched data
4. Cardinality limits are enforced last

## Security Processor (nrsecurity)

### Purpose

The security processor protects sensitive information by:
- Redacting secrets and credentials
- Masking PII (Personally Identifiable Information)
- Enforcing compliance requirements
- Auditing data access

### Configuration

```yaml
processors:
  nrsecurity:
    # Enable/disable processor
    enabled: true
    
    # Secret redaction
    redact_secrets:
      enabled: true
      
      # Built-in patterns
      patterns:
        passwords: true      # Password patterns
        api_keys: true       # API key patterns
        tokens: true         # Token patterns
        credit_cards: true   # Credit card numbers
        ssn: true           # Social Security Numbers
        private_keys: true   # Private key content
        
      # Custom redaction patterns
      custom_patterns:
        - name: "internal_api_key"
          pattern: 'X-Internal-Key:\s*([A-Za-z0-9]{32})'
          replacement: 'X-Internal-Key: [REDACTED]'
          
        - name: "database_url"
          pattern: 'postgres://[^@]+@[^/]+/\w+'
          replacement: 'postgres://[REDACTED]@[REDACTED]/[REDACTED]'
    
    # PII redaction
    redact_pii:
      enabled: true
      patterns:
        email: true          # Email addresses
        phone: true          # Phone numbers
        ip_address: false    # IP addresses (keep for debugging)
        names: true          # Common name patterns
        
    # Compliance modes
    compliance:
      pci_dss:
        enabled: true
        # Mask all but last 4 digits of credit cards
        credit_card_mask: "****-****-****-####"
        
      hipaa:
        enabled: true
        # Remove medical record numbers
        patterns:
          - 'MRN:\s*\d+'
          - 'Patient ID:\s*\d+'
          
      gdpr:
        enabled: true
        # Right to be forgotten support
        user_id_hashing: true
        
    # Audit trail
    audit:
      enabled: true
      # Log redaction events
      log_redactions: true
      # Include redaction count in metrics
      emit_metrics: true
```

### Examples

#### Input Data
```json
{
  "message": "User login failed for john@example.com with password=secret123",
  "api_key": "sk-1234567890abcdef",
  "credit_card": "4111-1111-1111-1111",
  "ssn": "123-45-6789",
  "database_url": "postgres://admin:pass123@db.example.com/myapp"
}
```

#### Output Data
```json
{
  "message": "User login failed for [EMAIL] with password=[REDACTED]",
  "api_key": "[REDACTED]",
  "credit_card": "****-****-****-1111",
  "ssn": "[REDACTED]",
  "database_url": "postgres://[REDACTED]@[REDACTED]/[REDACTED]"
}
```

### Built-in Patterns

| Pattern Type | Examples | Default Action |
|--------------|----------|----------------|
| Passwords | `password=`, `pwd:`, `pass:` | Full redaction |
| API Keys | `api_key:`, `apikey=`, `X-API-Key:` | Full redaction |
| Tokens | `token:`, `auth:`, `Bearer ` | Full redaction |
| Credit Cards | Visa, MasterCard, Amex patterns | Last 4 digits |
| SSN | `###-##-####` format | Full redaction |
| Email | Standard email format | Full redaction or domain only |
| Phone | International formats | Full redaction |

### Performance Impact

- **CPU**: ~2-5% overhead for typical workloads
- **Memory**: Minimal (pattern cache ~10MB)
- **Latency**: <1ms per span/metric

## Enrichment Processor (nrenrich)

### Purpose

The enrichment processor adds contextual metadata:
- Host and cloud provider information
- Kubernetes annotations and labels
- Environment variables
- Custom static/dynamic attributes

### Configuration

```yaml
processors:
  nrenrich:
    enabled: true
    
    # Host metadata enrichment
    host_metadata:
      enabled: true
      
      # Basic host info
      hostname: true
      fqdn: true
      os: true
      arch: true
      
      # Network information
      network:
        interfaces: true
        primary_ip: true
        
      # System information
      system:
        boot_time: true
        timezone: true
        
    # Cloud provider metadata
    cloud:
      enabled: true
      
      # Auto-detect provider
      auto_detect: true
      
      # Provider-specific settings
      aws:
        enabled: true
        metadata:
          - instance_id
          - instance_type
          - availability_zone
          - region
          - account_id
          - vpc_id
          - subnet_id
          - security_groups
          - tags
          
      gcp:
        enabled: true
        metadata:
          - instance_id
          - machine_type
          - zone
          - project_id
          - tags
          
      azure:
        enabled: true
        metadata:
          - vm_id
          - vm_size
          - location
          - resource_group
          - subscription_id
          - tags
          
    # Kubernetes metadata
    kubernetes:
      enabled: true
      
      # API server connection
      api_endpoint: ""  # Auto-detected in-cluster
      
      # Metadata to collect
      pod:
        annotations: true
        labels: true
        owner: true
        node_name: true
        service_account: true
        
      node:
        annotations: true
        labels: true
        allocatable: true
        capacity: true
        
      namespace:
        annotations: true
        labels: true
        
      # Label/annotation filters
      filters:
        # Include only matching labels
        include_labels:
          - "app.*"
          - "version"
          - "team"
          
        # Exclude sensitive labels
        exclude_labels:
          - ".*secret.*"
          - ".*password.*"
          
    # Container runtime metadata
    container:
      enabled: true
      
      docker:
        enabled: true
        socket: "/var/run/docker.sock"
        
      containerd:
        enabled: true
        socket: "/run/containerd/containerd.sock"
        
    # Static attributes (always added)
    static_attributes:
      - key: "environment"
        value: "${ENVIRONMENT:-production}"
        
      - key: "datacenter"
        value: "us-east-1"
        
      - key: "team"
        value: "platform"
        
      - key: "cost_center"
        value: "engineering"
        
    # Dynamic attributes (computed)
    dynamic_attributes:
      - key: "hour_of_day"
        source: "time.Now().Hour()"
        
      - key: "day_of_week"
        source: "time.Now().Weekday().String()"
        
      - key: "is_business_hours"
        source: "time.Now().Hour() >= 9 && time.Now().Hour() < 17"
        
      - key: "deployment_id"
        source: "os.Getenv('DEPLOYMENT_ID')"
        
    # Environment variable mapping
    env_attributes:
      - env: "SERVICE_VERSION"
        key: "service.version"
        
      - env: "GIT_COMMIT"
        key: "git.commit"
        
      - env: "BUILD_NUMBER"
        key: "build.number"
```

### Examples

#### Before Enrichment
```json
{
  "name": "http.request.duration",
  "value": 125.5,
  "attributes": {
    "http.method": "GET",
    "http.route": "/api/users"
  }
}
```

#### After Enrichment
```json
{
  "name": "http.request.duration",
  "value": 125.5,
  "attributes": {
    "http.method": "GET",
    "http.route": "/api/users",
    "host.name": "web-server-1",
    "host.ip": "10.0.1.50",
    "cloud.provider": "aws",
    "cloud.region": "us-east-1",
    "cloud.availability_zone": "us-east-1a",
    "cloud.instance.type": "t3.medium",
    "k8s.pod.name": "web-server-1-abc123",
    "k8s.namespace.name": "production",
    "k8s.deployment.name": "web-server",
    "k8s.node.name": "ip-10-0-1-50.ec2.internal",
    "environment": "production",
    "datacenter": "us-east-1",
    "team": "platform",
    "hour_of_day": 14,
    "is_business_hours": true
  }
}
```

### Metadata Sources

| Source | Update Frequency | Cache Duration |
|--------|------------------|----------------|
| Host | Once at startup | Forever |
| Cloud | Every 5 minutes | 5 minutes |
| Kubernetes | Real-time via watch | No cache |
| Container | Every 30 seconds | 30 seconds |
| Environment | Once at startup | Forever |

## Transform Processor (nrtransform)

### Purpose

The transform processor performs:
- Unit conversions
- Metric calculations and aggregations
- Field renaming and restructuring
- Data type conversions

### Configuration

```yaml
processors:
  nrtransform:
    enabled: true
    
    # Metric transformations
    metrics:
      # Unit conversions
      conversions:
        - metric: "system.memory.usage"
          from: "bytes"
          to: "gigabytes"
          
        - metric: "system.disk.io"
          from: "bytes/sec"
          to: "megabytes/sec"
          
        - metric: "network.io.*.bytes"
          from: "bytes"
          to: "megabits"
          scale: 8  # bytes to bits
          
        - metric: "temperature.*"
          from: "celsius"
          to: "fahrenheit"
          formula: "value * 9/5 + 32"
          
      # Aggregations
      aggregations:
        - name: "http.request.rate"
          type: "rate"
          metric: "http.request.count"
          interval: "1m"
          unit: "requests/sec"
          
        - name: "error.rate"
          type: "rate"
          metric: "http.request.errors"
          interval: "5m"
          unit: "errors/min"
          
        - name: "cpu.usage.average"
          type: "average"
          metrics: ["cpu.usage.core*"]
          interval: "1m"
          
        - name: "memory.usage.total"
          type: "sum"
          metrics: ["container.*.memory.usage"]
          group_by: ["k8s.namespace.name"]
          
      # Calculations
      calculations:
        - name: "error_percentage"
          expression: "(http.errors / http.requests) * 100"
          unit: "percent"
          precision: 2
          
        - name: "memory_percentage"
          expression: "(memory.used / memory.total) * 100"
          unit: "percent"
          
        - name: "disk_free_percentage"
          expression: "(disk.free / disk.total) * 100"
          unit: "percent"
          
        - name: "request_success_rate"
          expression: "((http.requests - http.errors) / http.requests) * 100"
          unit: "percent"
          default: 100  # When no requests
          
      # Histogram transformations
      histograms:
        - metric: "http.request.duration"
          buckets: [0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10]
          unit: "seconds"
          
        - metric: "response.size"
          buckets: [100, 1000, 10000, 100000, 1000000]
          unit: "bytes"
          
    # Trace transformations
    traces:
      # Span name normalization
      span_names:
        # Normalize HTTP routes
        - pattern: "GET /api/users/[0-9]+"
          replacement: "GET /api/users/{id}"
          
        - pattern: "POST /api/orders/[a-f0-9-]{36}"
          replacement: "POST /api/orders/{uuid}"
          
        # Normalize database queries
        - pattern: "SELECT .* FROM users WHERE id = \\d+"
          replacement: "SELECT * FROM users WHERE id = ?"
          
      # Span attribute transformations
      attributes:
        # Rename attributes
        rename:
          - from: "http.status_code"
            to: "http.response.status_code"
            
          - from: "db.statement"
            to: "db.query"
            
        # Add computed attributes
        compute:
          - key: "http.duration_ms"
            expression: "duration * 1000"
            
          - key: "error"
            expression: "http.status_code >= 400"
            
          - key: "cache_hit"
            expression: "http.response.headers['X-Cache'] == 'HIT'"
            
      # Error detection rules
      error_detection:
        - condition: "http.status_code >= 500"
          set_error: true
          error_type: "server_error"
          
        - condition: "http.status_code >= 400 && http.status_code < 500"
          set_error: true
          error_type: "client_error"
          
        - condition: "rpc.grpc.status_code != 0"
          set_error: true
          error_type: "grpc_error"
          
    # Log transformations
    logs:
      # Field mapping
      field_mapping:
        - from: "msg"
          to: "message"
          
        - from: "lvl"
          to: "severity"
          
        - from: "ts"
          to: "timestamp"
          
      # Severity normalization
      severity_mapping:
        - from: ["TRACE", "FINEST"]
          to: "trace"
          
        - from: ["DEBUG", "FINE"]
          to: "debug"
          
        - from: ["INFO", "INFORMATION"]
          to: "info"
          
        - from: ["WARN", "WARNING"]
          to: "warning"
          
        - from: ["ERROR", "ERR", "SEVERE"]
          to: "error"
          
        - from: ["FATAL", "CRITICAL", "PANIC"]
          to: "fatal"
          
      # Parse structured fields
      parsers:
        - field: "message"
          type: "json"
          target: "parsed"
          
        - field: "stack_trace"
          type: "multiline"
          pattern: '^\\s+at'
          
      # Extract fields
      extractors:
        - field: "message"
          pattern: 'user_id=(?P<user_id>\\d+)'
          
        - field: "message"
          pattern: 'duration=(?P<duration>\\d+)ms'
          type: "int"
```

### Transformation Examples

#### Unit Conversion
```yaml
# Input
- name: "system.memory.usage"
  value: 8589934592  # bytes
  
# Output  
- name: "system.memory.usage"
  value: 8  # gigabytes
  unit: "GB"
```

#### Metric Calculation
```yaml
# Input metrics
- name: "http.requests"
  value: 1000
- name: "http.errors"
  value: 50

# Calculated output
- name: "error_percentage"
  value: 5.0
  unit: "percent"
```

#### Span Normalization
```yaml
# Input
- name: "GET /api/users/12345/orders/67890"
  
# Output
- name: "GET /api/users/{id}/orders/{order_id}"
```

### Supported Unit Conversions

| From | To | Category |
|------|-----|----------|
| bytes | KB, MB, GB, TB | Storage |
| bytes/sec | Kbps, Mbps, Gbps | Bandwidth |
| milliseconds | seconds, minutes | Time |
| celsius | fahrenheit, kelvin | Temperature |
| percent | ratio | Percentage |

## Cardinality Processor (nrcap)

### Purpose

The cardinality processor protects against metric explosion by:
- Limiting unique time series
- Aggregating high-cardinality dimensions
- Dropping low-value metrics
- Providing cardinality analytics

### Configuration

```yaml
processors:
  nrcap:
    enabled: true
    
    # Global cardinality limit
    limits:
      # Maximum total unique series
      global: 1000000
      
      # Per-metric limits
      metrics:
        # Specific metric limits
        "http.request.duration": 50000
        "custom.business.metric": 10000
        
        # Pattern-based limits
        "trace.*": 100000
        "log.*": 200000
        "system.*": 20000
        
      # Per-dimension limits
      dimensions:
        "user_id": 10000
        "session_id": 50000
        "request_id": 100000
        "url": 1000
        
    # Cardinality reduction strategies
    strategies:
      # What to do when limit exceeded
      overflow_action: "aggregate"  # drop, aggregate, sample
      
      # Dimension reduction
      dimension_reduction:
        enabled: true
        
        # High-cardinality dimension handling
        rules:
          - dimension: "user_id"
            action: "hash"
            buckets: 1000
            
          - dimension: "session_id"
            action: "drop"
            
          - dimension: "url"
            action: "normalize"
            patterns:
              - pattern: "/api/users/\\d+"
                replacement: "/api/users/{id}"
                
          - dimension: "error_message"
            action: "truncate"
            max_length: 100
            
      # Aggregation rules
      aggregation:
        # Group similar metrics
        - pattern: "http.request.duration"
          group_by: ["service", "method", "status_class"]
          drop: ["user_id", "session_id"]
          
        - pattern: "database.query.duration"
          group_by: ["service", "operation", "table"]
          drop: ["query_id", "connection_id"]
          
      # Sampling rules (when dropping)
      sampling:
        # Keep percentage of dropped metrics
        rate: 0.01  # 1%
        
        # Always keep certain metrics
        always_keep:
          - pattern: "error.*"
          - pattern: "*.p99"
          - attributes:
              severity: "critical"
              
    # Priority system
    priority:
      # Higher priority metrics are kept
      rules:
        - pattern: "*.p99"
          priority: 100
          
        - pattern: "*.p95"
          priority: 90
          
        - pattern: "error.*"
          priority: 80
          
        - pattern: "http.request.duration"
          priority: 70
          
        - pattern: "custom.*"
          priority: 30
          
        - pattern: "debug.*"
          priority: 10
          
    # Monitoring and alerting
    monitoring:
      # Track cardinality metrics
      enabled: true
      
      # Emit cardinality metrics
      emit_metrics: true
      metric_prefix: "nrdot.cardinality"
      
      # Alert thresholds
      alerts:
        - level: "warning"
          threshold: 0.8  # 80% of limit
          
        - level: "critical"
          threshold: 0.95  # 95% of limit
          
      # Detailed tracking
      track_by:
        - "metric_name"
        - "service"
        - "dimension"
        
    # Adaptive limits
    adaptive:
      enabled: true
      
      # Adjust limits based on usage
      adjustment:
        # Increase limits for frequently accessed metrics
        promote_threshold: 1000  # accesses per minute
        promote_factor: 1.5
        
        # Decrease limits for rarely used metrics
        demote_threshold: 10  # accesses per minute
        demote_factor: 0.5
        
      # Learning period
      learning_duration: "24h"
      
    # Cache configuration
    cache:
      # Size of cardinality tracking cache
      size: 100000
      
      # TTL for cached entries
      ttl: "1h"
      
      # Cleanup interval
      cleanup_interval: "5m"
```

### Cardinality Examples

#### Before Limiting
```yaml
# 1M unique user_ids × 100 endpoints = 100M series
metrics:
  - name: "http.request.duration"
    labels:
      user_id: "user_123456"  # 1M unique values
      endpoint: "/api/v1/data" # 100 unique values
      method: "GET"
```

#### After Limiting
```yaml
# 1000 user buckets × 100 endpoints = 100K series
metrics:
  - name: "http.request.duration"
    labels:
      user_id_bucket: "bucket_123"  # 1000 buckets
      endpoint: "/api/v1/data"
      method: "GET"
```

### Cardinality Reduction Techniques

| Technique | Use Case | Example |
|-----------|----------|---------|
| **Hashing** | High-cardinality IDs | user_id → user_bucket |
| **Dropping** | Unnecessary dimensions | Remove session_id |
| **Normalization** | URL paths | /users/123 → /users/{id} |
| **Truncation** | Long strings | Limit to 100 chars |
| **Aggregation** | Similar metrics | Group by status class |
| **Sampling** | Overflow handling | Keep 1% sample |

### Monitoring Cardinality

The processor emits these metrics:

```yaml
# Current cardinality
nrdot.cardinality.current{metric="http.request.duration"} 45000

# Limit utilization
nrdot.cardinality.utilization{metric="http.request.duration"} 0.9

# Dropped series count
nrdot.cardinality.dropped{metric="http.request.duration"} 5000

# Top cardinality dimensions
nrdot.cardinality.dimension{metric="http.request.duration",dimension="user_id"} 35000
```

## Configuration Examples

### Minimal Configuration

```yaml
processors:
  nrsecurity:
    redact_secrets: true
    
  nrenrich:
    host_metadata: true
    
  nrtransform:
    # Use defaults
    
  nrcap:
    limits:
      global: 100000
```

### Production Configuration

```yaml
processors:
  nrsecurity:
    redact_secrets:
      enabled: true
      patterns:
        passwords: true
        api_keys: true
        credit_cards: true
    redact_pii:
      enabled: true
    compliance:
      pci_dss: true
      
  nrenrich:
    host_metadata:
      enabled: true
    cloud:
      enabled: true
      auto_detect: true
    kubernetes:
      enabled: true
    static_attributes:
      - key: "environment"
        value: "production"
      - key: "region"
        value: "us-east-1"
        
  nrtransform:
    metrics:
      conversions:
        - metric: "system.memory.usage"
          from: "bytes"
          to: "gigabytes"
      calculations:
        - name: "error_rate"
          expression: "(http.errors / http.requests) * 100"
          unit: "percent"
          
  nrcap:
    limits:
      global: 500000
      metrics:
        "http.request.duration": 100000
    strategies:
      overflow_action: "aggregate"
    priority:
      rules:
        - pattern: "error.*"
          priority: 100
```

### High-Security Configuration

```yaml
processors:
  nrsecurity:
    redact_secrets:
      enabled: true
      patterns:
        passwords: true
        api_keys: true
        tokens: true
        credit_cards: true
        ssn: true
        private_keys: true
      custom_patterns:
        - name: "internal_keys"
          pattern: 'X-Internal-.*:\s*[^\s]+'
          replacement: 'X-Internal-*: [REDACTED]'
    redact_pii:
      enabled: true
      patterns:
        email: true
        phone: true
        ip_address: true
    compliance:
      pci_dss: true
      hipaa: true
      gdpr: true
    audit:
      enabled: true
      
  # Other processors...
```

### High-Volume Configuration

```yaml
processors:
  # Minimal security for performance
  nrsecurity:
    redact_secrets:
      enabled: true
      patterns:
        passwords: true
        api_keys: true
        
  # Skip expensive enrichments
  nrenrich:
    host_metadata:
      enabled: true
    cloud:
      enabled: false
    kubernetes:
      enabled: false
      
  # No complex transformations
  nrtransform:
    enabled: false
    
  # Aggressive cardinality limits
  nrcap:
    limits:
      global: 50000
    strategies:
      overflow_action: "drop"
      dimension_reduction:
        enabled: true
        rules:
          - dimension: "user_id"
            action: "drop"
          - dimension: "request_id"
            action: "drop"
```

## Performance Considerations

### Processor Overhead

| Processor | CPU Impact | Memory Impact | Latency |
|-----------|------------|---------------|---------|
| nrsecurity | 2-5% | 10MB | <1ms |
| nrenrich | 1-3% | 50-100MB | <2ms |
| nrtransform | 3-7% | 20MB | <1ms |
| nrcap | 1-2% | 100-500MB | <1ms |

### Optimization Tips

1. **Order matters**: Place expensive processors last
2. **Disable unused features**: Each feature adds overhead
3. **Use sampling**: For high-volume, low-value data
4. **Cache metadata**: Reduce API calls for enrichment
5. **Batch operations**: Process multiple items together

### Benchmarks

```bash
# Run processor benchmarks
cd otel-processor-nrsecurity && go test -bench=.
cd otel-processor-nrenrich && go test -bench=.
cd otel-processor-nrtransform && go test -bench=.
cd otel-processor-nrcap && go test -bench=.
```

## Troubleshooting

### Debug Logging

Enable debug logging for processors:

```yaml
processors:
  nrsecurity:
    debug: true
    
service:
  telemetry:
    logs:
      level: debug
```

### Common Issues

1. **Missing Enrichment Data**
   ```bash
   # Check metadata sources
   nrdot-ctl test enrichment
   
   # Verify permissions
   kubectl auth can-i get pods --as=system:serviceaccount:nrdot:nrdot
   ```

2. **Incorrect Transformations**
   ```bash
   # Test transformation rules
   nrdot-ctl test transform --rule="error_rate"
   
   # Validate expressions
   nrdot-ctl validate expression "(http.errors / http.requests) * 100"
   ```

3. **Cardinality Limit Exceeded**
   ```bash
   # Check current cardinality
   nrdot-ctl metrics cardinality
   
   # Find high-cardinality dimensions
   nrdot-ctl metrics dimensions --top=10
   ```

4. **Performance Issues**
   ```bash
   # Profile processors
   nrdot-ctl profile processors --duration=60s
   
   # Check processor metrics
   curl localhost:9090/metrics | grep processor_
   ```

### Validation Tools

```bash
# Validate processor configuration
nrdot-ctl config validate-processors

# Test processor chain
nrdot-ctl test pipeline --input=test-data.json

# Simulate processing
nrdot-ctl simulate --config=config.yaml --data=sample.json
```

## Best Practices

1. **Security First**
   - Always enable secret redaction
   - Review custom patterns regularly
   - Audit redaction effectiveness

2. **Enrichment Strategy**
   - Cache static metadata
   - Use filters to limit data collection
   - Balance enrichment vs. performance

3. **Transformation Rules**
   - Test expressions thoroughly
   - Handle edge cases (division by zero)
   - Document custom calculations

4. **Cardinality Management**
   - Monitor cardinality trends
   - Set alerts for limit approach
   - Review and adjust limits monthly

5. **Testing**
   - Test with production-like data
   - Validate compliance requirements
   - Benchmark performance impact

## Next Steps

- [Configuration Reference](./configuration.md) - Detailed configuration options
- [Troubleshooting Guide](./troubleshooting.md) - Common issues and solutions
- [Performance Tuning](./performance.md) - Optimization strategies