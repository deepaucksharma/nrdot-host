# NRDOT-HOST Configuration Reference

Complete configuration reference for NRDOT-HOST.

## Table of Contents

- [Configuration Overview](#configuration-overview)
- [Configuration Files](#configuration-files)
- [Environment Variables](#environment-variables)
- [Configuration Schema](#configuration-schema)
- [Core Settings](#core-settings)
- [Telemetry Configuration](#telemetry-configuration)
- [Processor Configuration](#processor-configuration)
- [Security Configuration](#security-configuration)
- [Advanced Configuration](#advanced-configuration)
- [Examples](#examples)

## Configuration Overview

NRDOT-HOST uses a layered configuration system:

1. **Built-in defaults** - Sensible defaults for all settings
2. **Configuration file** - YAML-based configuration
3. **Environment variables** - Override file settings
4. **Command-line flags** - Highest priority

### Configuration Priority

```
Command-line flags > Environment variables > Config file > Defaults
```

## Configuration Files

### Default Locations

- Linux: `/etc/nrdot/config.yaml`
- macOS: `/usr/local/etc/nrdot/config.yaml`
- Windows: `C:\ProgramData\nrdot\config.yaml`
- Container: `/etc/nrdot/config.yaml`

### File Format

Configuration files use YAML format:

```yaml
# config.yaml
service:
  name: my-service
  environment: production

license_key: YOUR_LICENSE_KEY

# ... additional configuration
```

### Loading Custom Config

```bash
# Using CLI
nrdot-ctl --config /path/to/config.yaml

# Using environment variable
export NRDOT_CONFIG_FILE=/path/to/config.yaml

# Using systemd
systemctl edit nrdot-host
# Add: Environment="NRDOT_CONFIG_FILE=/path/to/config.yaml"
```

## Environment Variables

All configuration options can be set via environment variables using the pattern:
`NRDOT_<SECTION>_<KEY>`

### Examples

```bash
# Core settings
export NRDOT_LICENSE_KEY="your-license-key"
export NRDOT_SERVICE_NAME="my-service"
export NRDOT_SERVICE_ENVIRONMENT="production"

# Telemetry settings
export NRDOT_METRICS_ENABLED="true"
export NRDOT_METRICS_INTERVAL="60s"
export NRDOT_TRACES_ENABLED="true"
export NRDOT_TRACES_SAMPLE_RATE="0.1"

# Security settings
export NRDOT_SECURITY_REDACT_SECRETS="true"
export NRDOT_SECURITY_REDACT_PII="true"

# API settings
export NRDOT_API_BIND_ADDRESS="127.0.0.1:8080"
export NRDOT_API_ENABLE_TLS="true"
```

## Configuration Schema

### Top-Level Structure

```yaml
# Service identification
service:
  name: string           # Service name (required)
  namespace: string      # Service namespace
  environment: string    # Environment (dev/staging/prod)
  version: string        # Service version
  
# New Relic settings
license_key: string      # New Relic license key (required)
api_endpoint: string     # API endpoint (optional)
region: string           # Region (US/EU)

# Telemetry collection
metrics:                 # Metrics configuration
traces:                  # Traces configuration  
logs:                    # Logs configuration

# Processing
processors:              # Processor configuration
pipelines:               # Pipeline configuration

# Security
security:                # Security settings

# Operational
api:                     # API server settings
health:                  # Health check settings
telemetry:               # Self-telemetry settings
```

## Core Settings

### Service Configuration

```yaml
service:
  # Required: Identifies your service in New Relic
  name: "checkout-service"
  
  # Optional: Namespace for multi-tenant environments
  namespace: "e-commerce"
  
  # Optional: Environment identifier
  environment: "production"  # dev, staging, production
  
  # Optional: Service version
  version: "1.2.3"
  
  # Optional: Additional metadata
  metadata:
    team: "payments"
    region: "us-east-1"
    datacenter: "dc1"
```

### New Relic Configuration

```yaml
# Required: Your New Relic license key
license_key: "eu01xx...NRAL"

# Optional: API endpoint (auto-detected from license key)
api_endpoint: "https://otlp.eu01.nr-data.net"

# Optional: Region (auto-detected from license key)
# Values: US, EU, FedRAMP
region: "EU"

# Optional: Request timeout
timeout: "30s"

# Optional: Retry configuration
retry:
  enabled: true
  initial_interval: "5s"
  max_interval: "30s"
  max_elapsed_time: "5m"
```

### Logging Configuration

```yaml
# Log level: debug, info, warn, error
log_level: "info"

# Log format: text, json
log_format: "text"

# Log output: stdout, stderr, file
log_output: "stdout"

# File logging
log_file:
  path: "/var/log/nrdot/collector.log"
  max_size: 100  # MB
  max_backups: 5
  max_age: 30    # days
  compress: true
```

## Telemetry Configuration

### Metrics Configuration

```yaml
metrics:
  # Enable/disable metrics collection
  enabled: true
  
  # Collection interval
  interval: "60s"
  
  # Metric sources
  sources:
    # Host metrics
    host:
      enabled: true
      cpu: true
      memory: true
      disk: true
      network: true
      filesystem: true
      
    # Process metrics
    process:
      enabled: true
      include_patterns:
        - "nginx.*"
        - "java.*"
      exclude_patterns:
        - ".*test.*"
        
    # Container metrics
    container:
      enabled: true
      docker: true
      containerd: true
      
    # Kubernetes metrics
    kubernetes:
      enabled: true
      node: true
      pod: true
      container: true
      
  # Aggregations
  aggregations:
    - type: "histogram"
      metrics: ["http.request.duration"]
      buckets: [0.1, 0.5, 1, 2, 5, 10]
```

### Traces Configuration

```yaml
traces:
  # Enable/disable traces collection
  enabled: true
  
  # Sampling configuration
  sampling:
    # Sampling rate (0.0-1.0)
    rate: 0.1
    
    # Adaptive sampling
    adaptive:
      enabled: true
      min_rate: 0.01
      max_rate: 1.0
      target_tps: 100  # traces per second
      
  # Trace processors
  processors:
    # Span attributes
    attributes:
      - key: "environment"
        value: "production"
        action: "upsert"
        
    # Span filtering
    filter:
      exclude:
        - attributes["http.route"] == "/health"
        - duration < 1ms
        
  # Trace propagation
  propagation:
    formats:
      - "w3c"
      - "b3"
      - "jaeger"
```

### Logs Configuration

```yaml
logs:
  # Enable/disable logs collection
  enabled: true
  
  # Log sources
  sources:
    # File logs
    files:
      - path: "/var/log/nginx/access.log"
        parser: "nginx"
        multiline:
          pattern: '^\d{4}-\d{2}-\d{2}'
          
      - path: "/var/log/app/*.log"
        parser: "json"
        exclude_patterns:
          - "*.gz"
          - "*.tmp"
          
    # Syslog
    syslog:
      enabled: true
      protocol: "rfc5424"
      listen_address: "0.0.0.0:514"
      
    # Journal (systemd)
    journal:
      enabled: true
      units:
        - "nginx.service"
        - "app.service"
        
  # Log parsing
  parsers:
    - name: "nginx"
      type: "regex"
      pattern: '^(?P<remote_addr>\S+) .* \[(?P<time_local>.+)\] "(?P<request>.+)" (?P<status>\d+)'
      
    - name: "json"
      type: "json"
      timestamp_key: "timestamp"
      timestamp_format: "RFC3339"
      
  # Log enrichment
  enrichment:
    # Add hostname
    add_hostname: true
    
    # Add environment
    add_environment: true
    
    # Custom attributes
    attributes:
      - key: "service"
        value: "nginx"
        
  # Log filtering
  filters:
    # Severity filter
    severity: "info"  # debug, info, warn, error
    
    # Include patterns
    include:
      - attributes["log.level"] >= "WARN"
      
    # Exclude patterns  
    exclude:
      - body =~ "health check"
```

## Processor Configuration

### Security Processor

```yaml
processors:
  security:
    # Enable security processor
    enabled: true
    
    # Secret redaction
    redact_secrets:
      enabled: true
      
      # Built-in patterns
      patterns:
        passwords: true
        api_keys: true
        tokens: true
        credit_cards: true
        ssn: true
        
      # Custom patterns
      custom_patterns:
        - name: "internal_token"
          pattern: 'X-Internal-Token:\s*([^\s]+)'
          replacement: 'X-Internal-Token: [REDACTED]'
          
    # PII redaction
    redact_pii:
      enabled: true
      
      patterns:
        email: true
        phone: true
        ip_address: false  # Keep IPs for debugging
        
    # Compliance
    compliance:
      pci_dss: true
      hipaa: true
      gdpr: true
```

### Enrichment Processor

```yaml
processors:
  enrichment:
    # Enable enrichment processor
    enabled: true
    
    # Host metadata
    host_metadata:
      enabled: true
      
      # Standard metadata
      hostname: true
      os: true
      arch: true
      
      # Cloud metadata
      cloud:
        enabled: true
        providers:
          - aws
          - gcp
          - azure
        timeout: "5s"
        
    # Kubernetes metadata
    kubernetes:
      enabled: true
      
      # Metadata to collect
      pod:
        annotations: true
        labels: true
        owner: true
        
      node:
        annotations: true
        labels: true
        
    # Custom attributes
    static_attributes:
      - key: "datacenter"
        value: "us-east-1a"
        
      - key: "team"
        value: "platform"
        
    # Dynamic attributes
    dynamic_attributes:
      - key: "hour_of_day"
        source: "time.Hour()"
        
      - key: "is_business_hours"
        source: "time.Hour() >= 9 && time.Hour() < 17"
```

### Transform Processor

```yaml
processors:
  transform:
    # Enable transform processor
    enabled: true
    
    # Metric transformations
    metrics:
      # Unit conversions
      conversions:
        - metric: "disk.usage"
          from: "bytes"
          to: "gigabytes"
          
        - metric: "network.throughput"
          from: "bytes/s"
          to: "megabits/s"
          
      # Aggregations
      aggregations:
        - type: "sum"
          metrics: ["api.requests"]
          dimensions: ["endpoint", "method"]
          interval: "1m"
          
        - type: "average"
          metrics: ["response.time"]
          dimensions: ["service"]
          interval: "5m"
          
      # Calculations
      calculations:
        - name: "error_rate"
          expression: "errors / requests * 100"
          unit: "percent"
          
        - name: "saturation"
          expression: "used / total * 100"
          unit: "percent"
          
    # Trace transformations
    traces:
      # Span name mapping
      span_names:
        - from: "GET /api/v1/users/*"
          to: "GET /api/v1/users/{id}"
          
      # Error detection
      error_detection:
        - condition: 'attributes["http.status_code"] >= 400'
          set_error: true
          
    # Log transformations
    logs:
      # Field mapping
      field_mapping:
        - from: "msg"
          to: "message"
          
        - from: "lvl"
          to: "severity"
          
      # Severity mapping
      severity_mapping:
        - from: "FATAL"
          to: "critical"
```

### Cardinality Processor

```yaml
processors:
  cardinality:
    # Enable cardinality limiter
    enabled: true
    
    # Global limits
    limits:
      # Maximum unique metric series
      global: 100000
      
      # Per-metric limits
      metrics:
        "http.request.duration": 10000
        "custom.*": 5000
        
    # Limiting strategies
    strategy:
      # Action when limit exceeded
      action: "drop"  # drop, aggregate, sample
      
      # Priority rules
      priority:
        - pattern: "*.p99"
          weight: 10
          
        - pattern: "error.*"
          weight: 8
          
        - pattern: "custom.*"
          weight: 1
          
    # Monitoring
    monitoring:
      # Track cardinality metrics
      enabled: true
      
      # Alert thresholds
      thresholds:
        warning: 0.8   # 80% of limit
        critical: 0.95 # 95% of limit
```

## Security Configuration

### Authentication & Authorization

```yaml
security:
  # API authentication
  api_auth:
    enabled: true
    
    # Authentication methods
    methods:
      # Token authentication
      token:
        enabled: true
        header: "X-API-Token"
        tokens:
          - name: "ci-pipeline"
            token: "${API_TOKEN_CI}"
            permissions: ["read"]
            
          - name: "admin"
            token: "${API_TOKEN_ADMIN}"
            permissions: ["read", "write", "admin"]
            
      # mTLS authentication
      mtls:
        enabled: true
        ca_file: "/etc/nrdot/certs/ca.crt"
        require_client_cert: true
        
  # TLS configuration
  tls:
    # Server TLS
    server:
      enabled: true
      cert_file: "/etc/nrdot/certs/server.crt"
      key_file: "/etc/nrdot/certs/server.key"
      min_version: "1.2"
      cipher_suites:
        - "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
        - "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
        
    # Client TLS (for exporters)
    client:
      insecure_skip_verify: false
      ca_file: "/etc/nrdot/certs/ca.crt"
```

### Access Control

```yaml
security:
  # IP allowlist
  ip_allowlist:
    enabled: true
    ranges:
      - "10.0.0.0/8"
      - "172.16.0.0/12"
      - "192.168.0.0/16"
      
  # Rate limiting
  rate_limiting:
    enabled: true
    
    # Global limits
    global:
      requests_per_second: 1000
      burst: 2000
      
    # Per-endpoint limits
    endpoints:
      "/v1/metrics":
        requests_per_second: 100
        burst: 200
        
  # Audit logging
  audit:
    enabled: true
    
    # What to log
    events:
      - "config.change"
      - "auth.failure"
      - "api.access"
      
    # Where to log
    output:
      file: "/var/log/nrdot/audit.log"
      syslog: true
```

## Advanced Configuration

### Pipeline Configuration

```yaml
pipelines:
  # Metrics pipeline
  metrics:
    receivers:
      - prometheus
      - hostmetrics
      
    processors:
      - nrsecurity
      - nrenrich
      - nrtransform
      - nrcap
      
    exporters:
      - newrelic
      
  # Traces pipeline
  traces:
    receivers:
      - otlp
      - jaeger
      
    processors:
      - nrsecurity
      - nrenrich
      - attributes
      - batch
      
    exporters:
      - newrelic
      
  # Logs pipeline
  logs:
    receivers:
      - filelog
      - syslog
      
    processors:
      - nrsecurity
      - nrenrich
      - attributes
      
    exporters:
      - newrelic
```

### Receiver Configuration

```yaml
receivers:
  # OTLP receiver
  otlp:
    protocols:
      grpc:
        endpoint: "0.0.0.0:4317"
        
      http:
        endpoint: "0.0.0.0:4318"
        
  # Prometheus receiver
  prometheus:
    config:
      scrape_configs:
        - job_name: "node-exporter"
          static_configs:
            - targets: ["localhost:9100"]
              
  # Host metrics
  hostmetrics:
    collection_interval: "30s"
    scrapers:
      cpu: {}
      memory: {}
      disk: {}
      network: {}
      
  # File log receiver
  filelog:
    include:
      - "/var/log/**/*.log"
    exclude:
      - "/var/log/**/*.gz"
```

### Exporter Configuration

```yaml
exporters:
  # New Relic exporter
  newrelic:
    api_key: "${NRDOT_LICENSE_KEY}"
    
    # Compression
    compression: "gzip"
    
    # Batching
    timeout: "30s"
    
    # Retry
    retry_on_failure:
      enabled: true
      initial_interval: "5s"
      max_interval: "30s"
      
  # Debug exporter (development)
  debug:
    verbosity: "detailed"
    sampling_initial: 10
    sampling_thereafter: 100
```

### Resource Detection

```yaml
resource_detection:
  # Detectors to run
  detectors:
    - system
    - env
    - ec2
    - gcp
    - azure
    - kubernetes
    
  # Timeout for detection
  timeout: "5s"
  
  # Override detected values
  override: true
  
  # Additional attributes
  attributes:
    - key: "deployment.environment"
      value: "${ENVIRONMENT}"
      
    - key: "service.version"
      value: "${VERSION}"
```

### Health Checks

```yaml
health:
  # Startup checks
  startup:
    # Configuration validation
    validate_config: true
    
    # Connection tests
    test_exporters: true
    
    # Timeout
    timeout: "30s"
    
  # Liveness check
  liveness:
    enabled: true
    endpoint: "/health/live"
    interval: "30s"
    
  # Readiness check
  readiness:
    enabled: true
    endpoint: "/health/ready"
    
    # Readiness conditions
    conditions:
      - pipeline_running: true
      - exporter_connected: true
      - error_rate_low: true
```

### Performance Tuning

```yaml
performance:
  # Memory limits
  memory:
    # Soft limit (triggers GC)
    soft_limit: "400MiB"
    
    # Hard limit (OOM killer)
    hard_limit: "512MiB"
    
    # Ballast size
    ballast_size: "200MiB"
    
  # Batch processing
  batch:
    # Batch size
    size: 1000
    
    # Timeout
    timeout: "10s"
    
  # Queue settings
  queue:
    # Queue size
    size: 5000
    
    # Number of consumers
    consumers: 10
    
  # Concurrency
  concurrency:
    # Max concurrent requests
    limit: 100
    
    # Worker pool size
    workers: 10
```

## Examples

### Minimal Configuration

```yaml
# Minimal required configuration
service:
  name: "my-app"
  
license_key: "YOUR_LICENSE_KEY"

# Everything else uses defaults
```

### Production Configuration

```yaml
# Production-ready configuration
service:
  name: "payment-service"
  namespace: "finance"
  environment: "production"
  version: "${VERSION}"
  
license_key: "${NEW_RELIC_LICENSE_KEY}"
region: "US"

# Telemetry settings
metrics:
  enabled: true
  interval: "30s"
  
traces:
  enabled: true
  sampling:
    rate: 0.1
    adaptive:
      enabled: true
      target_tps: 100
      
logs:
  enabled: true
  sources:
    files:
      - path: "/var/log/app/*.log"
        parser: "json"
        
# Security
security:
  redact_secrets: true
  redact_pii: true
  
processors:
  cardinality:
    enabled: true
    limits:
      global: 50000
      
# Performance
performance:
  memory:
    soft_limit: "800MiB"
    hard_limit: "1GiB"
  batch:
    size: 2000
    timeout: "5s"
```

### Kubernetes Configuration

```yaml
# Kubernetes-optimized configuration
service:
  name: "${POD_NAME}"
  namespace: "${POD_NAMESPACE}"
  environment: "${CLUSTER_NAME}"
  
license_key: "${NEW_RELIC_LICENSE_KEY}"

# Kubernetes-specific settings
metrics:
  sources:
    kubernetes:
      enabled: true
      node: true
      pod: true
      
processors:
  enrichment:
    kubernetes:
      enabled: true
      pod:
        annotations: true
        labels: true
        
# Resource detection
resource_detection:
  detectors:
    - kubernetes
    - env
    
  attributes:
    - key: "k8s.cluster.name"
      value: "${CLUSTER_NAME}"
```

### High-Security Configuration

```yaml
# High-security configuration
service:
  name: "secure-app"
  environment: "production"
  
license_key: "${NEW_RELIC_LICENSE_KEY}"

# Maximum security
security:
  redact_secrets:
    enabled: true
    patterns:
      passwords: true
      api_keys: true
      tokens: true
      credit_cards: true
      ssn: true
      
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
    
  api_auth:
    enabled: true
    methods:
      mtls:
        enabled: true
        require_client_cert: true
        
  tls:
    server:
      enabled: true
      min_version: "1.3"
      
  audit:
    enabled: true
    events:
      - "*"
```

## Configuration Validation

### CLI Validation

```bash
# Validate configuration file
nrdot-ctl config validate

# Validate with specific file
nrdot-ctl config validate -f /path/to/config.yaml

# Verbose validation
nrdot-ctl config validate --verbose
```

### Pre-deployment Validation

```bash
# Test configuration without starting
nrdot-supervisor --dry-run

# Check specific components
nrdot-ctl test config
nrdot-ctl test connection
nrdot-ctl test processors
```

### Common Validation Errors

1. **Missing Required Fields**
   ```
   Error: service.name is required
   ```

2. **Invalid License Key**
   ```
   Error: license_key format invalid
   ```

3. **Type Mismatches**
   ```
   Error: metrics.interval must be a duration (e.g., "30s")
   ```

4. **Invalid Patterns**
   ```
   Error: security.custom_patterns[0].pattern is not a valid regex
   ```

## Getting Help

- Configuration examples: `/usr/share/nrdot/examples/`
- Online documentation: https://github.com/deepaucksharma/nrdot-host/docs
- Configuration generator: `nrdot-ctl config generate`
- Community support: https://github.com/deepaucksharma/nrdot-host/discussions