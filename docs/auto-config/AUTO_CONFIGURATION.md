# NRDOT-HOST Auto-Configuration System (Phase 2)

## Overview

**Status**: Not yet implemented - Planned for Phase 2 of the roadmap  
**Timeline**: 6 weeks of development  
**Technical Specification**: See [detailed technical documentation](AUTO_CONFIGURATION_TECHNICAL.md)

The auto-configuration system will transform NRDOT-HOST from a manually configured agent into an intelligent, self-configuring telemetry collector. This enterprise-grade feature delivers **zero-touch setup** by automatically discovering services and configuring optimal monitoring without manual intervention.

### Core Value Proposition
- **Zero-Touch Setup**: Install agent, get instant telemetry for common services
- **Optimal Defaults**: New Relic's best-practice configurations automatically applied
- **Remote Control**: Centrally manage and update configurations across fleet
- **Enterprise Security**: Cryptographically signed configs with rollback safety

## How It Will Work

### 1. Service Discovery Engine

The discovery engine uses multiple methods to ensure reliable service detection:

```go
type ServiceDiscovery struct {
    ProcessScanner   *ProcessScanner  // Scan /proc for running processes
    PortScanner      *PortScanner     // Check listening ports via /proc/net
    ConfigLocator    *ConfigLocator   // Find service config files
    PackageDetector  *PackageDetector // Query dpkg/rpm databases
    PrivilegedHelper *Helper          // Elevated access via setuid helper
}
```

**Multi-Method Correlation**: The engine correlates findings from all methods for high-confidence detection. For example:
- Process "mysqld" + Port 3306 + /etc/mysql/ = MySQL (HIGH confidence)
- Port 3306 only = Possible MySQL (MEDIUM confidence)

#### Detection Methods

- **Process Matching**: Identifies services by process name patterns
  - MySQL: `mysqld`, `mariadbd`
  - PostgreSQL: `postgres`, `postmaster`
  - Redis: `redis-server`
  - Nginx: `nginx`
  - Apache: `httpd`, `apache2`

- **Port Scanning**: Maps well-known ports to services
  - 3306 → MySQL/MariaDB
  - 5432 → PostgreSQL
  - 6379 → Redis
  - 80/443 → Web servers
  - 11211 → Memcached

- **Configuration Detection**: Looks for service config directories
  - `/etc/mysql/` → MySQL installed
  - `/etc/postgresql/` → PostgreSQL installed
  - `/etc/nginx/` → Nginx installed

### 2. Baseline Reporting (Phase 2)

Discovered services will be reported to New Relic:

```json
{
  "schema_version": "1.0",
  "host_id": "i-1234567890abcdef0",
  "hostname": "web-server-01",
  "discovered_services": [
    {
      "type": "mysql",
      "version": "8.0.32",
      "endpoint": "localhost:3306",
      "discovered_by": ["process", "port"]
    },
    {
      "type": "nginx", 
      "version": "1.22.1",
      "endpoints": [":80", ":443"],
      "discovered_by": ["process", "port"]
    }
  ],
  "host_metadata": {
    "os": "Ubuntu 22.04 LTS",
    "kernel": "5.15.0-88-generic",
    "cpu_cores": 4,
    "memory_gb": 16
  }
}
```

### 3. Configuration Retrieval (Phase 2)

The agent will fetch optimized configuration from New Relic:

```bash
# Planned API endpoint
GET https://api.newrelic.com/v1/nrdot/hosts/{host_id}/config
Authorization: Bearer {license_key}
Content-Type: application/json

Response:
{
  "version": "2024-01-15-001",
  "integrations": [
    {
      "type": "mysql",
      "enabled": true,
      "config": {
        "collection_interval": "30s",
        "metrics": ["connections", "queries", "innodb"],
        "logs": {
          "error_log": "/var/log/mysql/error.log",
          "slow_query_log": "/var/log/mysql/slow.log"
        }
      }
    }
  ]
}
```

### 4. Template-Based Configuration Generation

The configuration engine uses a sophisticated template system to generate optimal OpenTelemetry pipelines:

#### Template Organization
- `templates/integrations/`: Service-specific configurations
- `templates/common/`: Shared components (host metrics, processors)
- Templates embedded in binary for reliability
- Variable substitution from discovery data and remote manifest

#### Generated Configuration
The engine produces a complete OpenTelemetry Collector configuration. For a full example of what gets generated when MySQL, PostgreSQL, and Nginx are discovered, see [example-generated-config.yaml](example-generated-config.yaml).

#### Key Features
- **Automatic metric selection**: Only essential metrics enabled by default
- **Log integration**: Service logs automatically configured
- **Security first**: nrsecurity processor always first in pipeline
- **Resource optimization**: Memory limits and batching configured

### 5. Blue-Green Reload (Already Implemented)

The blue-green reload strategy is already implemented in v2.0:

1. New configuration is validated
2. New collector instance starts with new config
3. Health checks verify new instance
4. Traffic switches to new instance
5. Old instance gracefully shuts down
6. Automatic rollback on failure

This mechanism will be reused for auto-configuration updates.

## Future Configuration Options (Phase 2)

### Enabling/Disabling Auto-Configuration

```yaml
# /etc/nrdot/config.yaml
auto_config:
  enabled: true              # Will be default in Phase 2
  scan_interval: 5m          # Service discovery frequency
  
  # Exclude specific services
  exclude_services:
    - redis    # Skip Redis auto-config
    
  # Override auto-detected settings
  service_overrides:
    mysql:
      collection_interval: 60s
```

### Manual Override

Auto-configuration can be completely disabled for air-gapped or high-security environments:

```yaml
auto_config:
  enabled: false

# Manual configuration required
receivers:
  mysql:
    endpoint: localhost:3306
    # ... full manual config
```

## Service Support Roadmap

### Phase 2.0: Core Services (Week 1-5)
Full integration support including metrics and logs:

| Service | Detection Methods | Key Metrics | Log Types |
|---------|------------------|-------------|------------|
| **MySQL/MariaDB** | Process: mysqld<br>Port: 3306<br>Config: /etc/mysql/ | Buffer pool, queries, InnoDB, replication | Error log, slow query log |
| **PostgreSQL** | Process: postgres<br>Port: 5432<br>Config: /etc/postgresql/ | Database size, backends, operations, replication | PostgreSQL log |
| **Redis** | Process: redis-server<br>Port: 6379<br>Config: /etc/redis/ | Operations, memory, persistence, clients | Redis log |
| **Nginx** | Process: nginx<br>Port: 80/443<br>Config: /etc/nginx/ | Connections, requests, status | Access log, error log |
| **Apache** | Process: httpd/apache2<br>Port: 80/443<br>Config: /etc/apache2/ | Workers, requests, connections | Access log, error log |

### Phase 2.5: Extended Services
- **MongoDB**: Collections, operations, replication
- **Elasticsearch**: Cluster health, indices, search rate
- **RabbitMQ**: Queue depth, message rates, connections

### Service Detection Confidence Levels
- **HIGH**: Process + Port + Config detected
- **MEDIUM**: Two of three signals present
- **LOW**: Single signal only (requires manual confirmation)

## Security Architecture

### Configuration Integrity
- **Cryptographic Signing**: All configs signed with ECDSA P-256
- **Signature Verification**: Agent verifies before applying any config
- **Public Key**: Embedded in agent binary
- **Fail-Closed**: Unsigned or invalid configs are rejected

### Credential Management
1. **Environment Variables** (Phase 2.0)
   ```bash
   export MYSQL_MONITOR_USER=monitoring
   export MYSQL_MONITOR_PASS=secure_password
   export POSTGRES_MONITOR_USER=monitoring
   export POSTGRES_MONITOR_PASS=secure_password
   ```

2. **Secrets File** (Phase 2.5)
   ```yaml
   # /etc/nrdot/secrets.yaml (mode 0600, owner: nrdot)
   mysql:
     username: monitoring
     password: !vault |  # Future: Vault reference
       $ANSIBLE_VAULT;1.1;AES256...
   ```

### Network Security
- **TLS 1.3**: All external communication encrypted
- **Certificate Validation**: Proper CA chain verification
- **No Secrets in Transit**: Baseline reports contain no credentials
- **Fail-Closed**: Network errors prevent config updates

### Privilege Separation
- **Main Process**: Runs as 'nrdot' user
- **Privileged Helper**: Minimal setuid binary with:
  - `CAP_SYS_PTRACE`: Process inspection
  - `CAP_DAC_READ_SEARCH`: Config file access
- **Defense in Depth**: Multiple security layers

## Future CLI Commands (Phase 2)

### Service Discovery

```bash
# Will show detected services
nrdot-host discover

# Example output:
# Discovered Services:
#   mysql (8.0.32) on localhost:3306
#   nginx (1.22.1) on :80, :443
#   redis (7.0.5) on localhost:6379
```

### Auto-Config Status

```bash
# Will show auto-config state
nrdot-host status

# Auto-Configuration: ENABLED
# Last Discovery: 2024-01-15 10:30:00
# Active Integrations: mysql, nginx
```

### Troubleshooting

```bash
# Debug mode
sudo nrdot-host --mode=all --log-level=debug

# View logs
journalctl -u nrdot-host -f
```

## Best Practices (When Available)

1. **Start Simple**: Enable on non-production hosts first
2. **Secure Credentials**: Use environment variables for service passwords
3. **Monitor Changes**: Review logs after service additions
4. **Override When Needed**: Use manual config for special cases

## Implementation Plan

### Week-by-Week Breakdown

#### Weeks 1-2: Service Discovery Engine
- ProcessScanner with /proc parsing
- PortScanner with /proc/net analysis  
- ConfigLocator for standard paths
- PackageDetector for dpkg/rpm
- Discovery correlation logic

#### Week 3: Baseline & Remote Config
- BaselineReporter implementation
- ConfigFetcher with retry logic
- ECDSA signature verification
- Config caching and versioning

#### Week 4: Template System
- Template library creation
- Variable substitution engine
- Config merger and validator
- Test templates for 5 services

#### Week 5: Integration
- ConfigApplier with blue-green
- Health check framework
- RollbackManager implementation
- CLI commands (discover, preview)

#### Week 6: Production Hardening
- Performance optimization
- Security audit
- Observability (metrics, logs)
- End-to-end testing
- Documentation

### Success Metrics
- **Discovery Performance**: <1 second full scan
- **Config Generation**: <100ms template rendering  
- **Service Detection**: <30 seconds for new services
- **Config Safety**: Zero failed deployments
- **User Experience**: 90% reduction in manual YAML

## Related Documentation

- [Technical Specification](AUTO_CONFIGURATION_TECHNICAL.md) - Complete implementation details
- [Example Generated Config](example-generated-config.yaml) - See actual output
- [Architecture](../architecture/ARCHITECTURE.md) - System design
- [Roadmap](../roadmap/ROADMAP.md) - Full project timeline
- [Baseline Schema](baseline_schema.json) - Discovery report format