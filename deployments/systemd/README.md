# NRDOT Systemd Installation

This directory contains systemd service files and installation scripts for NRDOT (Network Resource Discovery and Optimization Toolkit).

## Quick Start

```bash
# Install NRDOT
sudo ./install.sh

# Complete post-installation setup
sudo ./scripts/post-install.sh

# Start services
sudo systemctl start nrdot.target

# Check status
sudo systemctl status nrdot.target
```

## Directory Structure

```
systemd/
├── README.md                   # This file
├── install.sh                  # Main installer script
├── uninstall.sh               # Uninstaller script
├── services/                  # Systemd service files
│   ├── nrdot-collector.service
│   ├── nrdot-supervisor.service
│   ├── nrdot-config-engine.service
│   ├── nrdot-api-server.service
│   ├── nrdot-privileged-helper.service
│   ├── nrdot-api.socket
│   ├── nrdot-privileged.socket
│   └── nrdot.target
├── configs/                   # Configuration files
│   ├── nrdot.conf            # Environment variables
│   ├── sysctl.d/             # Kernel parameters
│   └── limits.d/             # Resource limits
└── scripts/                  # Helper scripts
    ├── pre-install.sh        # Pre-installation checks
    ├── post-install.sh       # Post-installation setup
    ├── pre-uninstall.sh      # Pre-uninstall backup
    └── health-check.sh       # Service health checks
```

## Installation

### Prerequisites

- Linux system with systemd (v240+)
- Root access
- 4+ CPU cores (recommended)
- 8GB+ RAM (recommended)
- 10GB+ free disk space

### Installation Steps

1. **Run the installer:**
   ```bash
   sudo ./install.sh
   ```

2. **Review configuration:**
   ```bash
   # Edit service configuration
   sudo vi /etc/nrdot/collector.yaml
   sudo vi /etc/nrdot/api-server.yaml
   ```

3. **Complete setup:**
   ```bash
   sudo ./scripts/post-install.sh
   ```

4. **Start services:**
   ```bash
   sudo systemctl start nrdot.target
   ```

### Installation Options

```bash
# Install with custom prefix
sudo ./install.sh --prefix /usr/local/nrdot

# Build from source during installation
sudo ./install.sh --build
```

## Service Management

### Starting Services

```bash
# Start all services
sudo systemctl start nrdot.target

# Start individual service
sudo systemctl start nrdot-collector
```

### Stopping Services

```bash
# Stop all services
sudo systemctl stop nrdot.target

# Stop individual service
sudo systemctl stop nrdot-collector
```

### Service Status

```bash
# Check all services
sudo systemctl status 'nrdot-*'

# Check specific service
sudo systemctl status nrdot-api-server

# View logs
sudo journalctl -u nrdot-collector -f
```

### Enable Auto-start

```bash
# Enable all services
sudo systemctl enable nrdot.target

# Disable auto-start
sudo systemctl disable nrdot.target
```

## Configuration

### Environment Variables

Edit `/etc/nrdot/nrdot.conf` to configure system-wide settings:

```bash
NRDOT_LOG_LEVEL=info          # Log level: debug, info, warn, error
NRDOT_API_PORT=8080           # API server port
NRDOT_METRICS_PORT=9090       # Metrics port
```

### Service Configuration

Each service has its own YAML configuration file in `/etc/nrdot/`:

- `collector.yaml` - Data collector settings
- `supervisor.yaml` - Process supervisor settings
- `config-engine.yaml` - Configuration engine settings
- `api-server.yaml` - API server settings
- `privileged-helper.yaml` - Privileged helper settings

### Security

Services run with security hardening:

- Unprivileged user (except privileged-helper)
- Filesystem protections
- Network restrictions
- Resource limits
- Capability restrictions

## Health Checks

Services include built-in health checks:

```bash
# Manual health check
/opt/nrdot/scripts/health-check.sh collector

# View health status
curl -k https://localhost:8080/health

# Metrics endpoint
curl http://localhost:9090/metrics
```

## Troubleshooting

### Service Won't Start

1. Check logs:
   ```bash
   sudo journalctl -u nrdot-supervisor -n 50
   ```

2. Verify permissions:
   ```bash
   ls -la /var/lib/nrdot/
   ls -la /etc/nrdot/
   ```

3. Run health check:
   ```bash
   sudo /opt/nrdot/scripts/health-check.sh pre-start collector
   ```

### Port Conflicts

Check for port usage:
```bash
sudo ss -tlnp | grep -E '8080|9090'
```

### Performance Issues

1. Check resource usage:
   ```bash
   systemctl status nrdot-collector
   ```

2. Adjust limits in service files:
   ```bash
   sudo systemctl edit nrdot-collector
   ```

## Uninstallation

### Interactive Uninstall

```bash
sudo ./uninstall.sh
```

This will:
- Stop all services
- Create backups
- Ask before removing files
- Preserve data by default

### Complete Removal

```bash
sudo ./uninstall.sh --purge
```

This removes everything without prompting.

## Backup and Recovery

### Backup Locations

- Pre-uninstall: `/var/backups/nrdot/uninstall-*`
- Configuration: `/etc/nrdot/`
- Data: `/var/lib/nrdot/`
- Logs: `/var/log/nrdot/`

### Manual Backup

```bash
# Backup configuration and data
tar -czf nrdot-backup-$(date +%Y%m%d).tar.gz \
    /etc/nrdot \
    /var/lib/nrdot
```

### Restore

```bash
# Stop services
sudo systemctl stop nrdot.target

# Restore files
sudo tar -xzf nrdot-backup-*.tar.gz -C /

# Start services
sudo systemctl start nrdot.target
```

## Advanced Configuration

### Custom Service Overrides

```bash
# Create override directory
sudo systemctl edit nrdot-collector

# Add custom settings
[Service]
MemoryMax=4G
CPUWeight=200
```

### Socket Activation

API server supports socket activation:

```bash
# Enable socket
sudo systemctl enable --now nrdot-api.socket

# Service starts on demand
curl https://localhost:8080/health
```

### High Availability

For HA deployments:

1. Use shared storage for `/var/lib/nrdot`
2. Configure load balancer for API
3. Run multiple collector instances
4. Use external database

## Integration

### Monitoring

NRDOT exposes Prometheus metrics:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'nrdot'
    static_configs:
      - targets: ['localhost:9090']
```

### Logging

Configure log shipping:

```bash
# /etc/nrdot/nrdot.conf
NRDOT_LOG_FORMAT=json
NRDOT_LOG_OUTPUT=stdout
```

### API Access

```bash
# Get API token
API_TOKEN=$(sudo cat /etc/nrdot/auth-token)

# Make API request
curl -H "Authorization: Bearer $API_TOKEN" \
     https://localhost:8080/api/v1/status
```

## Support

- Documentation: https://github.com/NRDOT/nrdot-host
- Issues: https://github.com/NRDOT/nrdot-host/issues
- Community: https://nrdot.community

## License

NRDOT is licensed under the Apache License 2.0. See LICENSE file for details.