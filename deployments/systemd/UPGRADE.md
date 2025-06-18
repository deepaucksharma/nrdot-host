# NRDOT Upgrade Guide

This guide covers upgrading NRDOT installations to newer versions.

## Before You Begin

### Prerequisites

1. **Backup your data**:
   ```bash
   sudo tar -czf nrdot-backup-$(date +%Y%m%d).tar.gz \
       /etc/nrdot \
       /var/lib/nrdot \
       /var/log/nrdot
   ```

2. **Check current version**:
   ```bash
   nrdot-collector --version
   ```

3. **Review release notes** for breaking changes

### Compatibility Matrix

| From Version | To Version | Direct Upgrade | Notes |
|--------------|------------|----------------|-------|
| 0.9.x        | 1.0.x      | Yes            | Config migration required |
| 0.8.x        | 1.0.x      | No             | Upgrade to 0.9.x first |
| 1.0.x        | 1.1.x      | Yes            | Backward compatible |

## Upgrade Methods

### Method 1: Using Package Manager

#### RPM-based Systems (RHEL, CentOS, Fedora)

```bash
# Download new RPM
wget https://github.com/NRDOT/nrdot-host/releases/download/v1.1.0/nrdot-1.1.0-1.el8.x86_64.rpm

# Upgrade package
sudo yum upgrade nrdot-1.1.0-1.el8.x86_64.rpm
# or
sudo dnf upgrade nrdot-1.1.0-1.el8.x86_64.rpm
```

#### DEB-based Systems (Ubuntu, Debian)

```bash
# Download new DEB
wget https://github.com/NRDOT/nrdot-host/releases/download/v1.1.0/nrdot_1.1.0-1_amd64.deb

# Upgrade package
sudo apt install ./nrdot_1.1.0-1_amd64.deb
```

### Method 2: Manual Upgrade

1. **Stop services**:
   ```bash
   sudo systemctl stop nrdot.target
   ```

2. **Backup current installation**:
   ```bash
   sudo cp -a /opt/nrdot /opt/nrdot.backup
   ```

3. **Install new version**:
   ```bash
   cd /path/to/new/nrdot
   sudo ./install.sh
   ```

4. **Migrate configuration** (if needed):
   ```bash
   sudo /opt/nrdot/bin/nrdot-migrate-config \
       --from /etc/nrdot.backup \
       --to /etc/nrdot
   ```

5. **Start services**:
   ```bash
   sudo systemctl start nrdot.target
   ```

## Version-Specific Upgrade Instructions

### Upgrading to 1.1.0

#### Configuration Changes

1. **API Configuration**:
   ```yaml
   # Old format (1.0.x)
   api:
     port: 8080
     
   # New format (1.1.x)
   api:
     listen: "0.0.0.0:8080"
     tls:
       enabled: true
       cert: /etc/nrdot/certs/server.crt
       key: /etc/nrdot/certs/server.key
   ```

2. **Collector Configuration**:
   ```yaml
   # New fields in 1.1.x
   collector:
     buffer_size: 65536
     workers: auto
     ebpf:
       enabled: true
       programs:
         - tcp_monitor
         - udp_tracker
   ```

#### Database Schema Updates

The upgrade process will automatically migrate the database schema:

```bash
# Manual migration (if auto-migration fails)
sudo -u nrdot /opt/nrdot/bin/nrdot-db-migrate \
    --config /etc/nrdot/database.yaml \
    --from-version 1.0.0 \
    --to-version 1.1.0
```

#### New Features Configuration

1. **Enable new eBPF programs**:
   ```bash
   sudo vi /etc/nrdot/collector.yaml
   # Add new eBPF programs to configuration
   ```

2. **Configure new metrics**:
   ```bash
   sudo vi /etc/nrdot/nrdot.conf
   # Set NRDOT_FEATURE_ADVANCED_METRICS=true
   ```

### Upgrading to 1.0.0

#### Breaking Changes

1. **Service Names**: All services renamed from `nrd-*` to `nrdot-*`
2. **Config Format**: YAML format changed significantly
3. **API Endpoints**: `/api/v1/*` prefix added to all endpoints

#### Migration Steps

1. **Stop old services**:
   ```bash
   sudo systemctl stop 'nrd-*'
   ```

2. **Migrate configuration**:
   ```bash
   # Use provided migration tool
   sudo /opt/nrdot/tools/migrate-0.9-to-1.0.sh
   ```

3. **Update API clients**:
   - Add `/api/v1` prefix to all API calls
   - Update authentication headers

## Rolling Upgrade (High Availability)

For HA deployments with multiple nodes:

1. **Upgrade passive nodes first**:
   ```bash
   # On passive node
   sudo systemctl stop nrdot-collector
   sudo yum upgrade nrdot
   sudo systemctl start nrdot-collector
   ```

2. **Verify passive node health**:
   ```bash
   curl -k https://node2:8080/health
   ```

3. **Failover to upgraded node**:
   ```bash
   # Update load balancer to use upgraded nodes
   ```

4. **Upgrade remaining nodes**

## Post-Upgrade Tasks

### 1. Verify Installation

```bash
# Check service status
sudo systemctl status nrdot.target

# Verify versions
nrdot-collector --version
nrdot-api-server --version

# Check logs for errors
sudo journalctl -u nrdot-supervisor -n 100
```

### 2. Test Functionality

```bash
# Test API
curl -k https://localhost:8080/api/v1/status

# Test metrics
curl http://localhost:9090/metrics

# Run health checks
sudo /opt/nrdot/scripts/health-check.sh all
```

### 3. Update Monitoring

- Update Prometheus scrape configs
- Update Grafana dashboards
- Update alerting rules

### 4. Clean Up

```bash
# Remove backup after confirming upgrade success
sudo rm -rf /opt/nrdot.backup
sudo rm -rf /etc/nrdot.backup

# Clean old logs
sudo journalctl --vacuum-time=7d
```

## Rollback Procedure

If upgrade fails:

### 1. Stop Services

```bash
sudo systemctl stop nrdot.target
```

### 2. Restore Backup

```bash
# Restore configuration
sudo rm -rf /etc/nrdot
sudo tar -xzf nrdot-backup-*.tar.gz -C / etc/nrdot

# Restore binaries (manual install)
sudo rm -rf /opt/nrdot
sudo mv /opt/nrdot.backup /opt/nrdot
```

### 3. Downgrade Package

```bash
# RPM
sudo yum downgrade nrdot-0.9.5

# DEB
sudo apt install nrdot=0.9.5-1
```

### 4. Start Services

```bash
sudo systemctl start nrdot.target
```

## Troubleshooting

### Service Won't Start After Upgrade

1. **Check logs**:
   ```bash
   sudo journalctl -xe -u nrdot-supervisor
   ```

2. **Validate configuration**:
   ```bash
   sudo -u nrdot /opt/nrdot/bin/nrdot-config-validator \
       --config /etc/nrdot/
   ```

3. **Check permissions**:
   ```bash
   sudo chown -R nrdot:nrdot /var/lib/nrdot
   sudo chown -R nrdot:nrdot /etc/nrdot
   ```

### API Compatibility Issues

1. **Check API version**:
   ```bash
   curl -k https://localhost:8080/version
   ```

2. **Update clients to use new endpoints**

3. **Enable compatibility mode** (if available):
   ```yaml
   # In api-server.yaml
   api:
     compatibility_mode: true
     legacy_endpoints: true
   ```

### Database Migration Failures

1. **Check migration logs**:
   ```bash
   sudo tail -f /var/log/nrdot/db-migration.log
   ```

2. **Run manual migration**:
   ```bash
   sudo -u nrdot /opt/nrdot/bin/nrdot-db-migrate \
       --config /etc/nrdot/database.yaml \
       --force
   ```

3. **Restore database backup** if migration fails

## Best Practices

1. **Always backup before upgrading**
2. **Test upgrades in staging environment**
3. **Read release notes carefully**
4. **Plan for downtime (if not using HA)**
5. **Monitor closely after upgrade**
6. **Keep previous version packages available**
7. **Document any custom configurations**

## Support

- Documentation: https://docs.nrdot.io/upgrade
- Issues: https://github.com/NRDOT/nrdot-host/issues
- Community: https://nrdot.community/upgrade-help