# NRDOT-HOST Roadmap: Linux Telemetry Collector Evolution

## Vision Statement

Transform NRDOT-HOST from a generic OpenTelemetry distribution into New Relic's canonical Linux telemetry collector with intelligent auto-configuration capabilities.

## Timeline Overview

- **Phase 0**: Foundation & Cleanup (0.5 month) - *In Progress*
- **Phase 1**: Top-N Process Telemetry (1 month)
- **Phase 2**: Intelligent Auto-Configuration (1.5 months)
- **Phase 3**: GA & Migration Tools (1 month)

Total Timeline: **4 months** (not 3-6 months)

## Phase 0: Foundation & Cleanup (2 weeks)

**Status**: In Progress  
**Target**: Foundation for Linux-only telemetry collector

### Goals
- Complete Linux-only focus transition
- Remove cross-platform code
- Establish baseline telemetry

### Deliverables
- [x] Unified binary architecture (v2.0)
- [ ] Remove Windows/macOS build targets
- [ ] Clean up Kubernetes-specific code
- [ ] Integrate privileged helper into main flow
- [ ] Document actual v2.0 capabilities

### Success Metrics
- Single Linux binary < 50MB
- Zero cross-platform dependencies
- Clean CI/CD pipeline for Linux only

## Phase 1: Top-N Process Telemetry (4 weeks)

**Status**: Planned  
**Target**: Enhanced process monitoring via /proc

### Goals
- Rich process telemetry collection
- Top-N process tracking
- Service pattern detection

### Deliverables
- [ ] Process metrics collector using /proc
- [ ] Top-N CPU/memory process tracking
- [ ] Process relationship mapping (parent/child)
- [ ] Service detection by process patterns
- [ ] Privileged helper for elevated operations

### Technical Implementation

```go
// Enhanced process telemetry collector
type ProcessCollector struct {
    ProcPath     string              // /proc filesystem
    TopN         int                 // Track top N processes
    Interval     time.Duration       // Collection interval
    Cache        *ProcessCache       // Reduce syscall overhead
    Metrics      *MetricsEmitter     // Convert to OTel format
}

// Process data structure
type ProcessInfo struct {
    PID         int32
    PPID        int32               // Parent PID
    Name        string              // Process name
    Cmdline     string              // Full command line
    CPUPercent  float64             // CPU usage percentage
    MemoryRSS   uint64              // Resident memory in bytes
    MemoryVMS   uint64              // Virtual memory in bytes
    OpenFiles   int                 // File descriptor count
    ThreadCount int32               // Number of threads
    CreateTime  int64               // Process start time
    State       string              // R, S, D, Z, etc.
}

// /proc parsing methods
func (c *ProcessCollector) parseProc(pid int32) (*ProcessInfo, error) {
    // Read /proc/[pid]/stat for CPU and memory
    // Read /proc/[pid]/status for detailed info
    // Read /proc/[pid]/cmdline for full command
    // Calculate CPU percentage from jiffies
}
```

### Success Metrics
- < 5% CPU overhead for process monitoring
- 100% compatibility with Infrastructure agent data
- Sub-second process discovery

## Phase 2: Intelligent Auto-Configuration (6 weeks)

**Status**: Planned  
**Target**: Zero-touch service monitoring with enterprise-grade reliability

### Goals
- Automatic service discovery via multiple methods
- Cryptographically signed remote configurations
- Template-based dynamic pipeline generation
- Zero-downtime blue-green deployments

### Technical Architecture

#### Week 1-2: Service Discovery Engine
- [ ] **ProcessScanner**: /proc enumeration with pattern matching
- [ ] **PortScanner**: /proc/net parsing for listening services
- [ ] **ConfigLocator**: Standard path checking for configs
- [ ] **PackageDetector**: dpkg/rpm queries for installed services
- [ ] **Discovery Correlation**: Multi-signal confidence scoring

#### Week 3: Baseline Reporting & Remote Config
- [ ] **BaselineReporter**: Send discovered services to New Relic
- [ ] **ConfigFetcher**: Retrieve signed configurations
- [ ] **Signature Verification**: ECDSA P-256 validation
- [ ] **Retry Logic**: Exponential backoff, circuit breaker

#### Week 4: Template System & Generation
- [ ] **Template Library**: Embedded service templates
- [ ] **Variable Substitution**: Discovery data injection
- [ ] **Config Merger**: Combine templates into unified config
- [ ] **Validation Framework**: Schema and policy checks

#### Week 5: Integration & Apply
- [ ] **ConfigApplier**: Blue-green deployment orchestration
- [ ] **Health Checks**: Process, port, and metric validation
- [ ] **RollbackManager**: Automatic failure recovery
- [ ] **Version Management**: Config history and rollback

#### Week 6: Production Hardening
- [ ] **Performance Optimization**: <5% CPU, <1s discovery
- [ ] **Observability**: Metrics and structured logging
- [ ] **CLI Interface**: discover, preview, status commands
- [ ] **Security Audit**: Privilege model validation

### Discovery Implementation Details

```go
// Service patterns for process matching
servicePatterns := map[string][]string{
    "mysql":      {"mysqld", "mariadbd"},
    "postgresql": {"postgres", "postmaster"},
    "redis":      {"redis-server"},
    "nginx":      {"nginx"},
    "apache":     {"httpd", "apache2"},
}

// Well-known port mappings
wellKnownPorts := map[int]string{
    3306: "mysql",
    5432: "postgresql",
    6379: "redis",
    80:   "http",
    443:  "https",
}
```

### Template System Design

```yaml
# templates/integrations/mysql.yaml
receivers:
  mysql:
    endpoint: ${ENDPOINT:localhost:3306}
    collection_interval: ${INTERVAL:30s}
    username: ${MYSQL_MONITOR_USER}
    password: ${MYSQL_MONITOR_PASS}
    metrics:
      mysql.buffer_pool_pages:
        enabled: true
      mysql.locks:
        enabled: ${ENABLE_LOCKS:true}
```

### Security Architecture

1. **Config Signing**: ECDSA signatures on all configs
2. **TLS 1.3**: Encrypted communication to config service
3. **Privilege Separation**: Minimal capabilities via helper
4. **Credential Safety**: No secrets in configs or reports

### Supported Services (Phase 2.0)
- MySQL/MariaDB (metrics + error/slow logs)
- PostgreSQL (database stats + logs)
- Redis (operations + persistence metrics)
- Nginx (request rates + access/error logs)
- Apache (worker stats + logs)

### Success Metrics
- **Discovery Performance**: <1 second full scan
- **Config Generation**: <100ms template rendering
- **Service Detection**: <30 seconds for new services
- **Config Reduction**: 90% less manual YAML
- **Reload Safety**: Zero failed deployments

## Phase 3: GA & Migration Tools (4 weeks)

**Status**: Planned  
**Target**: Production-ready replacement

### Goals
- Seamless Infrastructure agent migration
- Enterprise packaging
- Production hardening

### Deliverables
- [ ] `nrdot-host migrate-infra` command
- [ ] Configuration conversion tools
- [ ] Official .deb/.rpm packages
- [ ] APT/YUM repository hosting
- [ ] Air-gapped deployment support

### Migration Strategy
```bash
# Automated migration
sudo nrdot-host migrate-infra

# What it does:
# 1. Detect Infrastructure agent
# 2. Convert configuration
# 3. Migrate license key
# 4. Preserve custom attributes
# 5. Validate metrics parity
# 6. Optional: Remove old agent
```

### Enterprise Features
- [ ] **Package Signing**: GPG signatures on .deb/.rpm
- [ ] **Proxy Support**: HTTP/HTTPS/SOCKS5 with auth
- [ ] **Air-Gapped Mode**: Offline installation packages
- [ ] **Compliance**:
  - FIPS 140-2 crypto modules
  - Common Criteria alignment
  - Audit logging for all operations
- [ ] **Multi-Tenant**: 
  - Multiple license keys
  - Namespace isolation
  - Per-service credentials

### Success Metrics
- 95% successful automated migrations
- < 1 hour migration for complex setups
- Feature parity + auto-config benefits

## Post-GA: Future Enhancements

### Stretch Goals (Not Committed)
- Container monitoring (via cgroups)
- eBPF-based network monitoring
- Custom integration SDK
- AI-powered anomaly detection

### Long-term Vision
- Become the standard Linux monitoring agent
- Replace need for manual integration setup
- Enable instant observability for any Linux service

## Risk Mitigation

### Technical Risks
1. **Process monitoring overhead**
   - Mitigation: Efficient /proc parsing, caching
   
2. **Auto-config complexity**
   - Mitigation: Start with 5 services, expand gradually
   
3. **Migration compatibility**
   - Mitigation: Extensive testing, gradual rollout

### Timeline Risks
1. **Scope creep**
   - Mitigation: Fixed feature set per phase
   
2. **Integration delays**
   - Mitigation: Early backend API development

## Success Criteria

### Overall Project Success
- [ ] 100% feature parity with Infrastructure agent
- [ ] 50% reduction in setup time
- [ ] 90% of services auto-configured
- [ ] < 150MB memory footprint
- [ ] < 5% CPU usage

### Phase Gate Criteria
Each phase must meet criteria before proceeding:
- All deliverables complete
- Success metrics achieved
- No critical bugs
- Documentation updated
- Customer validation (beta)

## Communication Plan

### Internal Updates
- Weekly status to stakeholders
- Bi-weekly demos
- Phase completion announcements

### External Communication
- Beta program for Phase 2
- GA announcement for Phase 3
- Migration webinars
- Documentation updates

---

*Last Updated: 2025-06-18*  
*Version: 1.0*  
*Status: Phase 0 In Progress*