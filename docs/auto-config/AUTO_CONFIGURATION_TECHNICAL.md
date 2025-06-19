# NRDOT-HOST Auto-Configuration Engine - Technical Specification

## Executive Summary

The auto-configuration engine transforms NRDOT-HOST into an intelligent, self-configuring telemetry collector for Linux hosts. It delivers **zero-touch setup** by automatically discovering services and configuring optimal monitoring without manual intervention. This specification details the production-grade implementation focusing on immediate customer value, reliability, and extensibility.

**Status**: Planned for Phase 2 (6 weeks) of the 4-month roadmap

## Core Goals and Value Delivered

### Zero-Touch Setup
Automatically discover running services (MySQL, PostgreSQL, Redis, Nginx, Apache) and configure their monitoring without manual intervention. Users install the agent and get telemetry **out-of-the-box** within minutes.

### Optimal Default Configurations  
Leverage New Relic's best-practice templates ensuring each discovered service is monitored with recommended metrics, logs, and safe settings. This guarantees **correctness** and consistent coverage across deployments.

### Platform Leverage & Remote Control
Allow New Relic's platform to deliver updated or improved configs to agents remotely. By reporting a "baseline" of detected services, the agent can fetch tailored configurations, enabling central management and future extensibility.

### Reliability & Safety
All configs are validated before use, and updates are applied via **blue-green deployment** with rollback, ensuring no downtime or broken monitoring. Cryptographic signing guarantees integrity and trust.

### Focus on Essentials
The design excludes non-essential features (no GUI, no ML heuristics, no GitOps) to minimize complexity. Initial feature set concentrates on **immediate customer value** and a robust foundation.

## Architecture Overview

### High-Level Architecture

The auto-configuration engine runs within NRDOT-HOST (built on OpenTelemetry Collector) as modular components handling discovery, config generation, validation, signing, and application. The design is **modular and embeddable**, allowing invocation via CLI or as a library.

### Configuration Data Flow

1. **Service Discovery**: Periodic scans (every 5 minutes) detect running services via multiple methods
2. **Baseline Report**: Send discovered services inventory to New Relic's configuration service
3. **Remote Config Fetch**: Retrieve signed configuration manifest optimized for the host
4. **Template Rendering**: Generate OpenTelemetry Collector pipeline configuration
5. **Validation**: Schema and policy validation before applying
6. **Dynamic Apply**: Blue-green deployment with health checks and rollback
7. **Continuous Operation**: Repeat cycle periodically for adaptation

## Modular Components

### Service Discovery Engine

```go
type ServiceDiscovery struct {
    ProcessScanner   *ProcessScanner  // Detect running processes
    PortScanner      *PortScanner     // Find listening ports
    ConfigLocator    *ConfigLocator   // Locate service configs on disk
    PackageDetector  *PackageDetector // Query installed packages (dpkg/rpm)
    PrivilegedHelper *Helper          // For elevated access if needed
}
```

#### ProcessScanner
- Enumerates running processes via `/proc` filesystem
- Pattern matching for known service executables:
  - MySQL: `mysqld`, `mariadbd`
  - PostgreSQL: `postgres`, `postmaster`
  - Redis: `redis-server`
  - Nginx: `nginx` (master/worker)
  - Apache: `httpd`, `apache2`
- Captures command-line arguments for multi-instance differentiation

#### PortScanner
- Checks listening ports via `/proc/net/*` or `ss` utility
- Maps well-known ports to services:
  - TCP 3306 → MySQL/MariaDB
  - TCP 5432 → PostgreSQL
  - TCP 6379 → Redis
  - TCP 80/443 → Web servers
  - TCP 11211 → Memcached
- Provides additional validation for process matches

#### ConfigLocator
- Probes filesystem for service configuration indicators:
  - `/etc/mysql/` or `/etc/my.cnf` → MySQL
  - `/etc/postgresql/` → PostgreSQL
  - `/etc/nginx/nginx.conf` → Nginx
  - `/etc/httpd/` or `/etc/apache2/` → Apache
- Extracts configuration details (custom ports, data directories)

#### PackageDetector
- Queries system package manager:
  - Debian/Ubuntu: `dpkg -l | grep service-name`
  - RHEL/CentOS: `rpm -q service-name`
- Identifies installed but not running services
- Provides accurate version information

#### PrivilegedHelper
- Minimal setuid binary for elevated operations
- Required capabilities:
  - `CAP_SYS_PTRACE`: Inspect all processes
  - `CAP_DAC_READ_SEARCH`: Read protected configs
- Secure local communication with main process
- Allows non-root agent operation

#### Discovery Integration
- Correlates findings from all methods for high confidence
- Aggregates into structured service list with:
  - Service type and version
  - Endpoints (host:port)
  - Configuration file paths
  - Discovery confidence score
- Caches results briefly for efficiency

### Template Renderer & Config Generator

The config generator transforms discovered services and remote instructions into concrete OpenTelemetry Collector configurations.

#### Template Management
```yaml
# Example MySQL template structure
receivers:
  mysql:
    endpoint: ${ENDPOINT}
    collection_interval: ${INTERVAL:30s}
    username: ${MYSQL_MONITOR_USER}
    password: ${MYSQL_MONITOR_PASS}
    metrics:
      mysql.buffer_pool_pages:
        enabled: true
      mysql.locks:
        enabled: ${METRICS_LOCKS:true}
        
  filelog/mysql_error:
    include:
      - ${MYSQL_ERROR_LOG:/var/log/mysql/error.log}
    start_at: end
    operators:
      - type: regex_parser
        regex: '^\d{4}-\d{2}-\d{2}'
```

#### Template Organization
- `templates/integrations/`: Service-specific configs
  - `mysql.yaml`, `postgres.yaml`, `nginx.yaml`, etc.
- `templates/common/`: Shared components
  - `hostmetrics.yaml`: Base host monitoring
  - `processors.yaml`: Security, enrichment
  - `exporters.yaml`: New Relic OTLP endpoint
- Templates embedded in binary for reliability

#### Template Processing
1. Load templates based on discovered services
2. Substitute variables from:
   - Discovery data (ports, paths)
   - Remote manifest (intervals, features)
   - Environment variables (credentials)
3. Merge components into unified config
4. Handle aggregation (multiple log paths, etc.)

#### Extensibility
- New services added via new templates
- No core logic changes required
- Templates updateable via agent releases

### Configuration Validator

Ensures generated configurations are safe and correct before application.

#### Schema Validation
- Validates against OpenTelemetry Collector schema
- Checks New Relic extension schemas
- Verifies:
  - Required fields present
  - Correct data types
  - No unknown fields
  - Valid references between sections

#### Policy Enforcement
- Resource limits (prevent excessive scraping)
- Path accessibility checks
- User override compliance
- Security policy adherence

#### Validation Process
```go
type ConfigValidator struct {
    schemaValidator  *schema.Validator
    policyEngine     *PolicyEngine
    collectorLoader  *otelcol.ConfigLoader
}

func (v *ConfigValidator) Validate(config CollectorConfig) error {
    // 1. Schema validation
    if err := v.schemaValidator.Validate(config); err != nil {
        return fmt.Errorf("schema validation failed: %w", err)
    }
    
    // 2. Policy checks
    if err := v.policyEngine.Check(config); err != nil {
        return fmt.Errorf("policy violation: %w", err)
    }
    
    // 3. Dry-run load test
    if err := v.collectorLoader.DryRun(config); err != nil {
        return fmt.Errorf("config load test failed: %w", err)
    }
    
    return nil
}
```

### Cryptographic Signing Service

Ensures authenticity and integrity of configurations delivered to agents.

#### Signing Architecture
- **Algorithm**: ECDSA with P-256 curve
- **Key Management**: 
  - Private key secured in New Relic backend (HSM)
  - Public key embedded in agent binary
- **Signature Format**: Attached to config response

#### Signing Process (Backend)
```go
// New Relic Configuration Service
func (s *ConfigService) SignConfig(config ConfigManifest) (SignedConfig, error) {
    // 1. Serialize config deterministically
    data, _ := json.Marshal(config)
    
    // 2. Compute hash
    hash := sha256.Sum256(data)
    
    // 3. Sign with private key
    r, s, _ := ecdsa.Sign(rand.Reader, s.privateKey, hash[:])
    signature := append(r.Bytes(), s.Bytes()...)
    
    return SignedConfig{
        Manifest:  config,
        Signature: base64.StdEncoding.EncodeToString(signature),
    }, nil
}
```

#### Verification Process (Agent)
```go
func (a *Agent) VerifyConfig(signed SignedConfig) error {
    // 1. Decode signature
    sigBytes, _ := base64.StdEncoding.DecodeString(signed.Signature)
    
    // 2. Recompute hash
    data, _ := json.Marshal(signed.Manifest)
    hash := sha256.Sum256(data)
    
    // 3. Verify with public key
    r := new(big.Int).SetBytes(sigBytes[:32])
    s := new(big.Int).SetBytes(sigBytes[32:])
    
    if !ecdsa.Verify(a.publicKey, hash[:], r, s) {
        return errors.New("signature verification failed")
    }
    
    return nil
}
```

#### Future Cosign Integration
Optional enhancement using Sigstore Cosign for transparency logs and OCI registry storage.

### Config Delivery Engine

Orchestrates the fetch, validation, and application of configurations.

```go
type RemoteConfigClient struct {
    BaselineReporter *BaselineReporter // Send discovered services
    ConfigFetcher    *ConfigFetcher    // Retrieve optimal config
    ConfigApplier    *ConfigApplier    // Apply without restart
    RollbackManager  *RollbackManager  // Handle failures
}
```

#### BaselineReporter
```go
type BaselinePayload struct {
    SchemaVersion      string           `json:"schema_version"`
    HostID            string           `json:"host_id"`
    Hostname          string           `json:"hostname"`
    Timestamp         time.Time        `json:"timestamp"`
    DiscoveredServices []ServiceInfo    `json:"discovered_services"`
    HostMetadata      HostMetadata     `json:"host_metadata"`
}

func (r *BaselineReporter) Report(ctx context.Context, services []ServiceInfo) error {
    payload := BaselinePayload{
        SchemaVersion:      "1.0",
        HostID:            r.getHostID(),
        Hostname:          r.getHostname(),
        Timestamp:         time.Now(),
        DiscoveredServices: services,
        HostMetadata:      r.collectHostMetadata(),
    }
    
    resp, err := r.httpClient.Post(
        "https://config.nr-data.net/v1/hosts/baseline",
        "application/json",
        payload,
    )
    // Handle response...
}
```

#### ConfigFetcher
- Retrieves signed configuration manifest
- Handles authentication (license key)
- Implements retry with exponential backoff
- Caches last successful config

#### ConfigApplier
- Coordinates blue-green deployment
- Interfaces with supervisor for process management
- Monitors health checks
- Triggers rollback if needed

## Blue-Green Reload Model

The agent employs zero-downtime configuration updates via blue-green deployment.

### Deployment Process

1. **Spawn New Instance**: Start new collector process with new config
2. **Health Validation**: 
   - Process startup check
   - Port binding verification
   - Metric emission validation
   - 5-10 second grace period
3. **Traffic Switchover**: Direct telemetry to new instance
4. **Graceful Shutdown**: Stop old instance with data flush
5. **Rollback on Failure**: Keep old instance if new fails

### Implementation Details
```go
func (s *Supervisor) BlueGreenReload(newConfig string) error {
    // 1. Start new collector
    newCollector := s.startCollector(newConfig)
    
    // 2. Health check with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    if err := s.healthCheck(ctx, newCollector); err != nil {
        // Rollback: kill new, keep old
        newCollector.Stop()
        return fmt.Errorf("health check failed: %w", err)
    }
    
    // 3. Switch traffic
    oldCollector := s.activeCollector
    s.activeCollector = newCollector
    
    // 4. Graceful shutdown of old
    ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    oldCollector.GracefulStop(ctx)
    
    return nil
}
```

### Resource Considerations
- Temporary double memory usage (2x ~150MB)
- Acceptable on modern servers
- Brief overlap minimizes impact

## Service Discovery Methods Detail

### Process Matching Patterns
```go
var servicePatterns = map[string][]string{
    "mysql":      {"mysqld", "mariadbd"},
    "postgresql": {"postgres", "postmaster"},
    "redis":      {"redis-server"},
    "nginx":      {"nginx"},
    "apache":     {"httpd", "apache2"},
    "mongodb":    {"mongod"},  // Phase 2.5
}
```

### Port Mapping
```go
var wellKnownPorts = map[int]string{
    3306:  "mysql",
    5432:  "postgresql",
    6379:  "redis",
    80:    "http",
    443:   "https",
    11211: "memcached",
    27017: "mongodb",  // Phase 2.5
}
```

### Config Path Detection
```go
var configPaths = map[string][]string{
    "mysql": {
        "/etc/mysql/my.cnf",
        "/etc/my.cnf",
        "/etc/mysql/",
    },
    "postgresql": {
        "/etc/postgresql/",
        "/var/lib/postgresql/data/postgresql.conf",
    },
    // ... more services
}
```

### Discovery Correlation
- Multiple signals increase confidence
- Process + Port + Config = High confidence
- Single signal = Medium confidence
- Report all findings, let backend decide

## Template Model Examples

### MySQL Integration Template
```yaml
# templates/integrations/mysql.yaml
receivers:
  mysql:
    endpoint: ${ENDPOINT:localhost:3306}
    collection_interval: ${COLLECTION_INTERVAL:30s}
    username: ${MYSQL_MONITOR_USER}
    password: ${MYSQL_MONITOR_PASS}
    
    # Metrics selection
    metrics:
      mysql.buffer_pool_pages:
        enabled: true
      mysql.buffer_pool_data_pages:
        enabled: true
      mysql.buffer_pool_limit:
        enabled: true
      mysql.locks:
        enabled: ${ENABLE_LOCK_METRICS:true}
      mysql.questions:
        enabled: true
      mysql.slow_queries:
        enabled: true
      mysql.threads:
        enabled: true
      mysql.commands:
        enabled: ${ENABLE_COMMAND_METRICS:false}
        
  filelog/mysql_error:
    include:
      - ${MYSQL_ERROR_LOG:/var/log/mysql/error.log}
    start_at: end
    include_file_path: true
    operators:
      - type: regex_parser
        regex: '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z\s+(?P<level>\w+)'
        
  filelog/mysql_slow:
    include:
      - ${MYSQL_SLOW_LOG:/var/log/mysql/slow.log}
    start_at: end
    multiline:
      line_start_pattern: '^# Time:'
```

### Host Metrics Template (Always Included)
```yaml
# templates/common/hostmetrics.yaml
receivers:
  hostmetrics:
    collection_interval: ${HOST_COLLECTION_INTERVAL:60s}
    scrapers:
      cpu:
        metrics:
          system.cpu.utilization:
            enabled: true
          system.cpu.load_average.1m:
            enabled: true
          system.cpu.load_average.5m:
            enabled: true
          system.cpu.load_average.15m:
            enabled: true
            
      memory:
        metrics:
          system.memory.utilization:
            enabled: true
          system.memory.usage:
            enabled: true
            
      disk:
        metrics:
          system.disk.operations:
            enabled: true
          system.disk.io:
            enabled: true
          system.disk.time:
            enabled: true
            
      filesystem:
        metrics:
          system.filesystem.utilization:
            enabled: true
          system.filesystem.usage:
            enabled: true
            
      network:
        metrics:
          system.network.packets:
            enabled: true
          system.network.errors:
            enabled: true
          system.network.io:
            enabled: true
            
      load:
        metrics:
          system.cpu.load_average.1m:
            enabled: true
            
      processes:
        # Enhanced in Phase 1 for Top-N tracking
```

### Common Processors Template
```yaml
# templates/common/processors.yaml
processors:
  # Security - must be first in pipeline
  nrsecurity:
    # Automatic secret redaction
    # No configuration needed
    
  # Enrichment
  nrenrich:
    host_metadata: true
    cloud_detection: true
    service_detection: true
    
  # Resource attributes
  resource:
    attributes:
      - key: service.name
        value: ${SERVICE_NAME:nrdot-host}
        action: upsert
      - key: service.environment
        value: ${ENVIRONMENT:production}
        action: upsert
      - key: host.name
        value: ${HOSTNAME}
        action: upsert
        
  # Performance optimization
  batch:
    timeout: 10s
    send_batch_size: 1000
    send_batch_max_size: 1500
    
  # Memory limiting
  memory_limiter:
    check_interval: 1s
    limit_mib: ${MEMORY_LIMIT:512}
    spike_limit_mib: ${MEMORY_SPIKE_LIMIT:128}
```

## Observability and Telemetry

The auto-configuration engine provides comprehensive self-monitoring.

### Internal Metrics
```go
// Discovery metrics
nrdot_discovery_duration_seconds{phase="scan"} histogram
nrdot_discovery_services_found{type="mysql"} gauge
nrdot_discovery_errors_total{reason="permission"} counter
nrdot_discovery_last_run_timestamp gauge

// Configuration metrics
nrdot_config_apply_total{status="success|failure"} counter
nrdot_config_version{version="hash"} gauge
nrdot_config_fetch_duration_seconds histogram
nrdot_config_validation_errors_total counter

// Reload metrics
nrdot_reload_duration_seconds histogram
nrdot_reload_rollback_total counter
nrdot_active_integrations{service="mysql"} gauge
```

### Structured Logging
```json
{
  "level": "info",
  "timestamp": "2024-01-15T10:30:00Z",
  "component": "discovery",
  "message": "Service discovery completed",
  "services_found": 3,
  "services": ["mysql", "nginx", "redis"],
  "duration_ms": 245
}

{
  "level": "info",
  "timestamp": "2024-01-15T10:31:00Z",
  "component": "config",
  "message": "Configuration applied successfully",
  "version": "2024-01-15-001",
  "integrations_enabled": ["mysql", "nginx"],
  "reload_method": "blue_green"
}
```

### Health Status Endpoint
```http
GET /health/autoconfig
{
  "status": "healthy",
  "discovery": {
    "last_run": "2024-01-15T10:30:00Z",
    "services_found": 3,
    "next_run": "2024-01-15T10:35:00Z"
  },
  "configuration": {
    "version": "2024-01-15-001",
    "applied_at": "2024-01-15T10:31:00Z",
    "integrations": ["mysql", "nginx", "redis"]
  }
}
```

## CLI and SDK Interface

### CLI Commands
```bash
# Discovery operations
nrdot-host discover [--json]                    # Run discovery and show results
nrdot-host discover --export baseline.json      # Export baseline payload

# Configuration operations  
nrdot-host config generate                      # Generate config from discovery
nrdot-host config preview                       # Show what would be applied
nrdot-host config validate <file>               # Validate a config file

# Auto-config control
nrdot-host autoconfig enable                    # Enable auto-configuration
nrdot-host autoconfig disable                   # Disable auto-configuration
nrdot-host autoconfig status                    # Show current status

# Manual operations
nrdot-host reload                               # Force config reload
nrdot-host rollback                             # Rollback to previous config
```

### SDK Interface
```go
// Discovery API
package autoconfig

type DiscoveryAPI interface {
    DiscoverServices(ctx context.Context) ([]ServiceInfo, error)
    DiscoverService(ctx context.Context, serviceType string) (*ServiceInfo, error)
}

// Config Generation API
type ConfigAPI interface {
    GenerateConfig(services []ServiceInfo, manifest ConfigManifest) (*CollectorConfig, error)
    ValidateConfig(config *CollectorConfig) error
    PreviewConfig(services []ServiceInfo) (string, error)
}

// Runtime API
type RuntimeAPI interface {
    EnableAutoConfig() error
    DisableAutoConfig() error
    GetStatus() (*AutoConfigStatus, error)
    ForceReload() error
}
```

### HTTP API Endpoints
```http
# Discovery
GET /api/v1/discovery/services
GET /api/v1/discovery/baseline

# Configuration
GET /api/v1/config/current
GET /api/v1/config/preview
POST /api/v1/config/reload

# Status
GET /api/v1/autoconfig/status
PUT /api/v1/autoconfig/enable
PUT /api/v1/autoconfig/disable
```

## Security Considerations

### Privilege Model
- Agent runs as `nrdot` user (non-root)
- Privileged helper for specific operations:
  - Process inspection (`CAP_SYS_PTRACE`)
  - Config file reading (`CAP_DAC_READ_SEARCH`)
- Minimal privilege principle throughout

### Credential Management
- Never store credentials in configs
- Environment variables for secrets
- Future: Integration with secret stores
- Credentials never sent to New Relic

### Communication Security
- TLS 1.3 for all external communication
- Certificate pinning for config service
- Signed configurations (ECDSA)
- No sensitive data in baseline reports

### Configuration Safety
- All configs validated before use
- Resource limits enforced
- Rollback on any failure
- Audit trail of all changes

## Performance Targets

### Resource Usage
- **Discovery Overhead**: <5% CPU during scan
- **Memory Impact**: <10MB for auto-config components
- **Scan Duration**: <1 second for typical host
- **Config Generation**: <100ms

### Operational Metrics
- **Discovery Interval**: 5 minutes (configurable)
- **Config Check Interval**: 1 hour (configurable)
- **Service Detection Time**: <30 seconds for new services
- **Reload Time**: <5 seconds (including health check)

### Scalability
- Supports 100+ services per host
- Efficient caching reduces repeated work
- Parallel discovery operations where possible

## Implementation Phases

### Phase 2.0: Core Engine (Weeks 1-3)
- Discovery engine with all methods
- Template system and config generation
- Validation framework
- Basic CLI commands

### Phase 2.1: Remote Integration (Weeks 4-5)
- Baseline reporting
- Config fetch with signing
- Blue-green integration
- Rollback mechanisms

### Phase 2.2: Production Hardening (Week 6)
- Performance optimization
- Comprehensive testing
- Security audit
- Documentation

## Future Enhancements

### Phase 2.5 Additions
- MongoDB, Elasticsearch, RabbitMQ support
- Secrets file support
- Enhanced CLI features
- Config history and diff

### Post-GA Considerations
- Cosign integration
- Container awareness
- Custom template support
- A/B testing for configs

## Summary

The auto-configuration engine delivers immediate value by eliminating manual configuration overhead while maintaining enterprise-grade reliability and security. Its modular design ensures extensibility without compromising the core focus on automatic, zero-touch service monitoring for Linux hosts.