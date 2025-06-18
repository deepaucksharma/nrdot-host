# NRDOT-HOST Architecture: Linux Telemetry Collector

## Overview

NRDOT-HOST is evolving from a unified OpenTelemetry distribution into New Relic's canonical Linux telemetry collector with intelligent auto-configuration capabilities. This architecture document describes both the current state and the roadmap for becoming a self-configuring, Linux-optimized host monitoring agent.

## Architectural Evolution

### Current State (v2.0) - Unified Architecture
- **Single binary** deployment with embedded components
- **Direct function calls** replacing complex IPC
- **40% memory reduction** through process consolidation
- **Linux-optimized** with platform-specific enhancements

### Future State (3-6 months) - Intelligent Auto-Configuration
- **Zero-touch setup** with automatic service discovery
- **Remote configuration** management from New Relic
- **Dynamic pipelines** that adapt to detected services
- **Seamless migration** from legacy Infrastructure agent

## Architecture Components

### 1. Core Binary (`nrdot-host`)

The Linux-optimized binary that serves as both the current unified agent and future auto-configuring collector:

```bash
# Default: Full Linux host monitoring
nrdot-host --mode=all

# Agent only: Minimal footprint
nrdot-host --mode=agent

# Future: Auto-configuration enabled (default in next release)
nrdot-host --auto-config=true
```

### 2. Core Components

#### Unified Supervisor
- Embeds API server, config engine, and collector management
- Direct function calls between components
- Shared memory state
- Single telemetry client

```go
type UnifiedSupervisor struct {
    configEngine  *configengine.EngineV2    // Embedded
    apiServer     *http.Server              // Embedded
    collector     *CollectorProcess         // Managed
    telemetry     *telemetryclient.Client   // Shared
}
```

#### Config Engine v2
- Consolidated schema validation and template generation
- In-memory config generation (no file dependencies)
- Version history and rollback support
- Direct integration with supervisor

```go
type EngineV2 struct {
    validator *schema.Validator      // Internal package
    generator *templates.Generator   // Internal package
    versions  []models.ConfigVersion // In-memory history
}
```

#### Common Module (`nrdot-common`)
- Shared data structures across all components
- Provider interfaces for clean contracts
- Centralized error handling
- Consistent serialization

### 3. Reload Strategies

The new architecture supports multiple reload strategies:

1. **Blue-Green (Default)**
   - Start new collector with new config
   - Verify health
   - Switch traffic
   - Stop old collector
   - Zero downtime

2. **Graceful**
   - Stop collector
   - Start with new config
   - Brief downtime but clean restart

3. **In-Place (Legacy)**
   - SIGHUP signal
   - Not recommended
   - Limited platform support

## Data Flow

### Current Configuration Flow
```
User YAML → ConfigEngine.Validate() → ConfigEngine.Generate() → Supervisor.Apply()
```

### Future Auto-Configuration Flow
```
Host Scan → Service Discovery → Baseline Upload → Remote Config Fetch → 
  → Template Selection → Pipeline Generation → Automatic Apply
```

### Status/Health Flow
```
Collector → Supervisor.GetStatus() → API.HandleStatus() → HTTP Response
```

### Dynamic Reload Flow
```
Config Change Detected → Validation → Blue-Green Deploy → Health Check → Commit/Rollback
```

## Linux Deployment Scenarios

### 1. Standard Linux Host (Primary Use Case)
```bash
# Install via package manager
sudo apt install nrdot-host
# or
sudo yum install nrdot-host

# Systemd service (auto-enabled)
[Service]
ExecStart=/usr/bin/nrdot-host --mode=all
User=nrdot
PrivilegesRequired=CAP_SYS_PTRACE,CAP_DAC_READ_SEARCH
```

### 2. Container Deployment (Docker)
```bash
docker run -d \
  --name nrdot-host \
  --network host \
  --pid host \
  --privileged \
  -v /proc:/host/proc:ro \
  -v /sys:/host/sys:ro \
  -v /etc:/host/etc:ro \
  newrelic/nrdot-host:latest
```

### 3. Migration from Infrastructure Agent
```bash
# Future: Automated migration
nrdot-host migrate-infra
# - Detects existing newrelic-infra
# - Migrates config and license key  
# - Preserves custom attributes
# - Seamless switchover
```

## Benefits Realized

### 1. Reduced Complexity
- From 5 processes to 1
- No IPC coordination issues
- Simplified debugging

### 2. Better Performance
- 30-40% memory reduction
- Lower CPU overhead
- Faster startup time

### 3. Improved Reliability
- No version mismatch between components
- Atomic updates
- Simplified error handling

### 4. Easier Operations
- Single binary to deploy
- One configuration file
- Unified logging and telemetry

## Configuration Evolution

### Current: Manual Configuration
```yaml
# /etc/nrdot/config.yaml
service:
  name: my-host
  environment: production
  
license_key: YOUR_LICENSE_KEY

# Manual integration setup required
receivers:
  mysql:
    endpoint: localhost:3306
    username: monitoring
    # ... more config
```

### Future: Auto-Configuration (Default)
```yaml
# /etc/nrdot/config.yaml
service:
  name: my-host
  environment: production
  
license_key: YOUR_LICENSE_KEY

# That's it! Auto-config handles:
# - Service discovery
# - Integration setup  
# - Optimal settings
# - Dynamic updates
```

## Roadmap Integration

### Phase 0: Foundation (Complete)
- ✅ Unified architecture implementation
- ✅ Core Linux telemetry collection
- ✅ Custom processors (security, enrichment)

### Phase 1: Enhanced Monitoring (1 month)
- 🎯 Process-level metrics and metadata
- 🎯 Service detection framework
- 🎯 Privileged helper integration

### Phase 2: Auto-Configuration (1.5 months)  
- 🎯 Automatic service discovery
- 🎯 Remote configuration API
- 🎯 Dynamic pipeline management
- 🎯 Template-based integrations

### Phase 3: Production Release (1 month)
- 🎯 Infrastructure agent migration tools
- 🎯 Enterprise packaging and signing
- 🎯 Performance optimization
- 🎯 Documentation and training

## Auto-Configuration Architecture (Coming Soon)

### Service Discovery Engine
```go
type ServiceDiscovery struct {
    ProcessScanner  *ProcessScanner  // Detect running services
    PortScanner     *PortScanner     // Find listening ports
    ConfigLocator   *ConfigLocator   // Locate service configs
    PrivilegedHelper *Helper         // Elevated access when needed
}
```

### Remote Configuration Client
```go
type RemoteConfigClient struct {
    BaselineReporter *BaselineReporter // Send discovered services
    ConfigFetcher    *ConfigFetcher    // Retrieve optimal config
    ConfigApplier    *ConfigApplier    // Apply without restart
    RollbackManager  *RollbackManager  // Safety mechanism
}
```

### Template-Based Integration System
```yaml
# Auto-generated for detected MySQL
receivers:
  mysql:
    endpoint: localhost:3306
    collection_interval: 30s
    metrics:
      mysql.buffer_pool.usage: enabled
      mysql.threads: enabled
```

## Linux-Specific Optimizations

### 1. Privileged Helper
- Minimal setuid binary for elevated operations
- Allows main process to run as non-root
- Secure communication via Unix sockets
- Operations: process details, network connections, file access

### 2. Efficient Resource Collection
- Direct `/proc` and `/sys` filesystem parsing
- Inotify for log file monitoring
- Minimal syscall overhead
- No eBPF dependency (compatibility first)

### 3. Process Monitoring
```go
type ProcessMonitor struct {
    TopProcesses    int           // Track top N by CPU/memory
    ServiceMatcher  *Matcher      // Identify known services
    MetricsEmitter  *Emitter      // Convert to OTel metrics
}
```

## Security Architecture

### Defense in Depth
1. **Process Isolation**: Runs as dedicated user
2. **Capability-based**: Only required Linux capabilities
3. **Secret Redaction**: Automatic PII/credential scrubbing
4. **TLS Only**: All external communication encrypted
5. **Local Socket**: Privileged helper communication

## Performance Targets

- **Memory**: <150MB base footprint
- **CPU**: <2% idle, <5% under load
- **Startup**: <3 seconds to first metric
- **Discovery**: <10 seconds initial scan
- **Config Apply**: <100ms with blue-green