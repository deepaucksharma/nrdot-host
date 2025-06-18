# NRDOT-HOST Auto-Configuration System

## Overview

The auto-configuration system transforms NRDOT-HOST from a manually configured agent into an intelligent, self-configuring telemetry collector. This system automatically discovers services running on Linux hosts and configures appropriate monitoring without user intervention.

## How It Works

### 1. Service Discovery

On startup and periodically thereafter, NRDOT-HOST scans the Linux host to detect running services:

```go
// Discovery runs every 5 minutes by default
type ServiceDiscovery struct {
    ProcessScanner   // Scans /proc for running processes
    PortScanner      // Checks listening ports (via netstat/ss)
    ConfigLocator    // Finds service config files
    PackageDetector  // Queries dpkg/rpm for installed services
}
```

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

### 2. Baseline Reporting

Discovered services are reported to New Relic as a baseline:

```json
{
  "host_id": "i-1234567890abcdef0",
  "hostname": "web-server-01",
  "discovered_services": [
    {
      "type": "mysql",
      "version": "8.0.32",
      "port": 3306,
      "config_path": "/etc/mysql/my.cnf"
    },
    {
      "type": "nginx", 
      "version": "1.22.1",
      "ports": [80, 443],
      "config_path": "/etc/nginx/nginx.conf"
    }
  ],
  "host_metadata": {
    "os": "Ubuntu 22.04",
    "cpu_count": 4,
    "memory_gb": 16,
    "cloud_provider": "aws",
    "instance_type": "t3.large"
  }
}
```

### 3. Configuration Retrieval

The agent fetches optimized configuration from New Relic's configuration service:

```bash
GET https://config.nr-data.net/v1/hosts/{host_id}/config
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

### 4. Dynamic Configuration Application

The configuration engine processes the remote config and generates appropriate OpenTelemetry pipelines:

```yaml
# Auto-generated configuration
receivers:
  # Base host metrics (always enabled)
  hostmetrics:
    collection_interval: 30s
    scrapers:
      cpu: {}
      memory: {}
      disk: {}
      network: {}
      process: {}
  
  # Auto-enabled MySQL receiver
  mysql:
    endpoint: localhost:3306
    collection_interval: 30s
    username: "${MYSQL_MONITOR_USER}"
    password: "${MYSQL_MONITOR_PASS}"
    metrics:
      mysql.buffer_pool_pages: { enabled: true }
      mysql.buffer_pool_data_pages: { enabled: true }
      mysql.buffer_pool_limit: { enabled: true }
  
  # Auto-enabled log collection
  filelog/mysql:
    include:
      - /var/log/mysql/error.log
      - /var/log/mysql/slow.log
    start_at: end
    
processors:
  # Always-on security
  nrsecurity:
    redact_patterns:
      - password
      - api_key
      - secret
      
  # Auto-enrichment
  nrenrich:
    host_metadata: true
    service_detection: true
    
exporters:
  otlp:
    endpoint: https://otlp.nr-data.net
    headers:
      api-key: ${NEW_RELIC_LICENSE_KEY}
```

### 5. Blue-Green Reload

Configuration changes are applied using blue-green deployment:

1. New configuration is validated
2. New collector instance starts with new config
3. Health checks verify new instance
4. Traffic switches to new instance
5. Old instance gracefully shuts down
6. Rollback on failure

## Configuration Options

### Enabling/Disabling Auto-Configuration

```yaml
# /etc/nrdot/config.yaml
auto_config:
  enabled: true              # Default: true
  scan_interval: 5m          # How often to scan for changes
  report_interval: 1h        # How often to check for config updates
  
  # Exclude specific services from auto-config
  exclude_services:
    - redis    # Don't auto-configure Redis
    
  # Override auto-detected settings
  service_overrides:
    mysql:
      username: custom_monitor
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

## Supported Services

### Phase 1 (Launch)
- **MySQL/MariaDB**: Metrics and logs
- **PostgreSQL**: Database statistics  
- **Redis**: Operations and memory metrics
- **Nginx**: Request and connection metrics
- **Apache**: Worker and request stats

### Phase 2 (3 months)
- **MongoDB**: Database and collection metrics
- **Elasticsearch**: Cluster and node health
- **RabbitMQ**: Queue and message metrics
- **Kafka**: Broker and topic metrics
- **Docker**: Container metrics and logs

### Phase 3 (6 months)
- **Kubernetes**: Node metrics (without full K8s monitoring)
- **Custom Applications**: Auto-detect apps with Prometheus endpoints
- **JVM Applications**: Auto-attach JMX monitoring
- **Database Clusters**: Multi-node awareness

## Security Considerations

### Credential Management

Auto-configuration never transmits credentials. Service credentials must be provided locally via:

1. **Environment Variables**
   ```bash
   export MYSQL_MONITOR_USER=monitoring
   export MYSQL_MONITOR_PASS=secure_password
   ```

2. **Secure Files**
   ```yaml
   # /etc/nrdot/secrets.yaml (mode 0600)
   mysql:
     username: monitoring
     password: secure_password
   ```

3. **Integration with Secret Managers** (future)
   - HashiCorp Vault
   - AWS Secrets Manager
   - Kubernetes Secrets

### Network Security

- All communication uses TLS 1.3
- Certificate pinning for nr-data.net
- No sensitive data in baseline reports
- Fail-closed on communication errors

## Troubleshooting

### View Discovery Results

```bash
# See what services were detected
nrdot-host discover --verbose

Scanning for services...
Found: MySQL 8.0.32 on port 3306
Found: Nginx 1.22.1 on ports 80, 443
Found: Redis 7.0.5 on port 6379
```

### Check Auto-Config Status

```bash
# View current auto-config state
nrdot-host status --auto-config

Auto-Configuration: ENABLED
Last Scan: 2024-01-15 10:30:00
Detected Services: mysql, nginx, redis
Last Config Update: 2024-01-15 10:31:00
Active Integrations: mysql, nginx
```

### Debug Auto-Configuration

```bash
# Run with debug logging
nrdot-host --log-level=debug --auto-config-debug

# Check logs
journalctl -u nrdot-host | grep auto-config
```

### Force Configuration Refresh

```bash
# Manually trigger discovery and config fetch
nrdot-host auto-config --refresh
```

## Best Practices

1. **Let Auto-Config Work**: Avoid manual configuration unless necessary
2. **Provide Credentials Securely**: Use environment variables or secure files
3. **Monitor Config Changes**: Review logs when services are added/removed
4. **Test in Staging**: Validate auto-config behavior before production
5. **Gradual Rollout**: Enable on a subset of hosts first

## Architecture Details

See [ARCHITECTURE_V2.md](ARCHITECTURE_V2.md#auto-configuration-architecture-coming-soon) for implementation details.

## Roadmap

- **Current**: Manual configuration with templates
- **Phase 1**: Basic service discovery and static config fetch
- **Phase 2**: Dynamic config with hot reload
- **Phase 3**: AI-powered optimization and anomaly-based config