# NRDOT-HOST: New Relic's OpenTelemetry Distribution for Linux

<p align="center">
  <img src="https://img.shields.io/badge/OpenTelemetry-Native-blue" alt="OpenTelemetry">
  <img src="https://img.shields.io/badge/License-Apache%202.0-green" alt="License">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/Platform-Linux-FCC624?logo=linux" alt="Linux">
</p>

<p align="center">
  <b>New Relic's canonical Linux telemetry collector with intelligent auto-configuration</b>
</p>

## 🎯 Overview

NRDOT-HOST is New Relic's next-generation Linux telemetry collector built on OpenTelemetry. Currently in v2.0 with unified architecture, it's evolving to provide intelligent auto-configuration for zero-touch monitoring.

### Current Features (v2.0)
- **Unified Binary**: All components in a single process
- **OpenTelemetry Native**: Built on OTel Collector, not a fork
- **Linux Optimized**: Designed specifically for Linux hosts
- **Custom Processors**: Security, enrichment, transformation, and capping
- **Blue-Green Reload**: Configuration updates without downtime

### Coming Soon (4-Month Roadmap)
- **Auto-Configuration**: Automatic service discovery and setup
- **Enhanced Process Monitoring**: Top-N processes via /proc
- **Remote Configuration**: Centralized config management
- **Migration Tools**: Automated Infrastructure agent transition

## 🚀 Roadmap

### Phase 1: Top-N Process Telemetry (4 weeks)
- Enhanced process monitoring using /proc
- Top CPU/memory process tracking
- Service detection by process patterns

### Phase 2: Intelligent Auto-Configuration (6 weeks)
- Automatic service discovery (MySQL, Redis, Nginx, etc.)
- Remote configuration from New Relic
- Dynamic pipeline generation

### Phase 3: GA & Migration (4 weeks)
- Infrastructure agent migration tools
- Production packaging (.deb/.rpm)
- Enterprise features

See [Full Roadmap](docs/roadmap/ROADMAP.md) for detailed timeline and deliverables.

## 📦 Installation

### Quick Install (Linux)

```bash
# Download latest release
curl -LO https://github.com/newrelic/nrdot-host/releases/latest/download/nrdot-host-linux-amd64
chmod +x nrdot-host-linux-amd64
sudo mv nrdot-host-linux-amd64 /usr/local/bin/nrdot-host

# Or use package manager
sudo apt install nrdot-host     # Ubuntu/Debian  
sudo yum install nrdot-host     # RHEL/CentOS/Amazon Linux
```

### Docker

```bash
docker run -d \
  --name nrdot-host \
  --network host \
  --pid host \
  --privileged \
  -v /etc/nrdot:/etc/nrdot \
  -v /var/log:/var/log:ro \
  -e NEW_RELIC_LICENSE_KEY=YOUR_KEY \
  newrelic/nrdot-host:latest
```

## ⚡ Configuration

Create `/etc/nrdot/config.yaml`:

```yaml
# Minimal configuration - auto-config handles the rest!
service:
  name: my-host
  environment: production

license_key: YOUR_NEW_RELIC_LICENSE_KEY

# Future: Auto-configuration (Phase 2)
# auto_config:
#   enabled: true          # Will be default in v3.0
#   scan_interval: 5m      # Service detection interval
```

### Future: Auto-Configuration (Phase 2)

When auto-configuration is released, NRDOT-HOST will:
1. **Scan** for running services (MySQL, Redis, Nginx, etc.)
2. **Report** discovered services to New Relic
3. **Receive** optimized configuration
4. **Apply** monitoring automatically
5. **Update** as services change

Until then, services must be configured manually in the YAML file.

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────┐
│           NRDOT-HOST v2.0 (Unified Binary)          │
├─────────────────────────────────────────────────────┤
│                                                     │
│  ┌─────────────────────────────────────────────┐   │
│  │          Supervisor (Main Process)           │   │
│  │  - Collector lifecycle management            │   │
│  │  - Configuration validation                  │   │
│  │  - API server (health, status)               │   │
│  │  - Blue-green reload orchestration           │   │
│  └───────────────────┬─────────────────────────┘   │
│                      │                              │
│  ┌───────────────────▼─────────────────────────┐   │
│  │         OpenTelemetry Collector Core        │   │
│  │                                             │   │
│  │  Receivers:        Processors:  Exporters: │   │
│  │  - hostmetrics    - nrsecurity  - otlp     │   │
│  │  - filelog        - nrenrich               │   │
│  │  - otlp           - nrtransform            │   │
│  │                   - nrcap                  │   │
│  └─────────────────────────────────────────────┘   │
│                                                     │
│  Future Components (Phase 2):                       │
│  [ ] Auto-Configuration Engine                      │
│  [ ] Service Discovery                              │
│  [ ] Remote Config Client                           │
└─────────────────────────────────────────────────────┘
```

## 📊 What Gets Monitored

### Always Enabled (Base Telemetry)
- **System Metrics**: CPU, memory, disk, network, load
- **System Logs**: syslog, auth logs, kernel logs
- **Process Info**: Process count, top processes by CPU/memory
- **Host Metadata**: Cloud attributes, OS info, hostname

### Future: Auto-Discovered Services (Phase 2)
When auto-configuration is released, these services will be automatically monitored:
- **MySQL/MariaDB**: Performance metrics, slow query logs
- **PostgreSQL**: Database statistics, connection metrics
- **Redis**: Operations, memory, persistence metrics
- **Nginx**: Request rates, connection stats
- **Apache**: Worker stats, request performance
- *More services added in Phase 2.5*

### Custom Applications
- **OTLP Gateway**: Applications can send traces/metrics to localhost:4317
- **Automatic Enrichment**: App telemetry is enriched with host context

## 🔄 Migrating from Infrastructure Agent

### Future: Automated Migration (Phase 3)

```bash
# Coming in Phase 3
sudo nrdot-host migrate-infra

# What it will do:
# 1. Detect existing Infrastructure agent
# 2. Convert configuration format
# 3. Migrate license key
# 4. Preserve custom attributes
# 5. Validate metrics continuity
```

### Manual Migration

1. Install NRDOT-HOST (see Installation above)
2. Copy your license key from `/etc/newrelic-infra.yml`
3. Create `/etc/nrdot/config.yaml` with license key
4. Stop Infrastructure agent: `sudo systemctl stop newrelic-infra`
5. Start NRDOT-HOST: `sudo systemctl start nrdot-host`
6. Verify in New Relic UI that host data is flowing

## 🛠️ Operations

### Status & Health

```bash
# Check agent status
nrdot-host status

# View live metrics
curl http://localhost:8090/metrics

# Health check
curl http://localhost:8090/health
```

### Troubleshooting

```bash
# View logs
journalctl -u nrdot-host -f

# Debug mode
sudo nrdot-host --mode=all --log-level=debug

# Test configuration (manual validation)
nrdot-host --mode=all --config=/etc/nrdot/config.yaml --dry-run
```

### Manual Configuration (Current)

Until auto-configuration is available, configure services manually:

```yaml
# Example: Add MySQL monitoring
receivers:
  mysql:
    endpoint: localhost:3306
    username: monitor
    password: ${env:MYSQL_PASSWORD}
    collection_interval: 60s

# Add receiver to pipeline
service:
  pipelines:
    metrics:
      receivers: [hostmetrics, mysql]
```

## 📈 Performance

### Current (v2.0)
- **Memory**: ~300MB (unified binary)
- **CPU**: 2-5% idle usage
- **Startup**: ~3 seconds
- **Config Reload**: <100ms (blue-green)

### Target (Phase 3 GA)
- **Memory**: <150MB
- **CPU**: <2% idle usage
- **Process Discovery**: <5 seconds
- **Service Detection**: <30 seconds

## 🔒 Security

- **Automatic Secret Redaction**: nrsecurity processor prevents credential leaks
- **Privileged Helper**: Secure elevation for specific operations (setuid)
- **TLS 1.3**: Encrypted communication to New Relic
- **No eBPF**: Uses traditional /proc methods for compatibility

## 🤝 Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

Key areas for contribution:
- Additional service auto-discovery modules
- OpenTelemetry receiver integrations
- Performance optimizations
- Documentation improvements

## 📚 Documentation

- [Architecture Overview](docs/architecture/ARCHITECTURE.md)
- [Roadmap Details](docs/roadmap/ROADMAP.md)
- [Configuration Reference](docs/configuration.md)
- [Troubleshooting](docs/troubleshooting.md)

### Future Documentation
- [Auto-Configuration Guide](docs/auto-config/AUTO_CONFIGURATION.md) (Phase 2)
- [Migration Guide](docs/migration/INFRASTRUCTURE_MIGRATION.md) (Phase 3)

## 🛠️ Development Status

### Implemented (v2.0)
- ✅ Unified binary architecture
- ✅ Host metrics and logs collection
- ✅ Custom processors (security, enrichment, transform, cap)
- ✅ Blue-green configuration reload
- ✅ Basic privileged helper

### In Progress
- 🔄 Linux-only optimization (removing cross-platform code)
- 🔄 Enhanced process telemetry planning

### Not Yet Implemented
- ❌ Auto-configuration engine
- ❌ Service discovery
- ❌ Remote configuration
- ❌ Migration tools
- ❌ Process telemetry integration

See [Roadmap](docs/roadmap/ROADMAP.md) for implementation timeline.

## 📄 License

Apache License 2.0 - see [LICENSE](LICENSE) for details.

---

<p align="center">
  <b>NRDOT-HOST: Building the future of Linux host monitoring</b><br>
  OpenTelemetry-Native • Auto-Configuring • Enterprise-Ready
</p>