# NRDOT-HOST Production Deployment Guide

## Overview

This guide provides comprehensive instructions for deploying NRDOT-HOST v3.0 in production environments. It covers deployment scenarios, security hardening, performance tuning, and operational best practices.

## Pre-Deployment Checklist

### System Requirements

**Minimum Requirements:**
- CPU: 2 cores (x86_64 or ARM64)
- Memory: 2GB RAM
- Storage: 10GB available disk space
- OS: Linux kernel 4.14+ (RHEL 7+, Ubuntu 18.04+, Debian 10+)

**Recommended Requirements:**
- CPU: 4+ cores
- Memory: 4GB+ RAM
- Storage: 20GB+ SSD
- Network: 100+ Mbps bandwidth

### Prerequisites

1. **New Relic Account**
   - Valid license key
   - OTLP endpoint access
   - API key for remote configuration (optional)

2. **System Access**
   - Root or sudo privileges for initial setup
   - Service account for runtime (created automatically)

3. **Network Requirements**
   - Outbound HTTPS (443) to New Relic endpoints
   - Local port access for service discovery
   - API port (8080) for management (optional)

## Installation Methods

### 1. Package Manager Installation

#### RPM-based Systems (RHEL, CentOS, Fedora)
```bash
# Add NRDOT repository
sudo tee /etc/yum.repos.d/nrdot.repo <<EOF
[nrdot]
name=NRDOT Repository
baseurl=https://download.newrelic.com/nrdot/linux/\$basearch
enabled=1
gpgcheck=1
gpgkey=https://download.newrelic.com/nrdot/nrdot.gpg
EOF

# Install NRDOT-HOST
sudo yum install -y nrdot-host

# Configure license key
sudo tee /etc/nrdot/nrdot.env <<EOF
NEW_RELIC_LICENSE_KEY=YOUR_LICENSE_KEY_HERE
EOF

# Enable and start service
sudo systemctl enable --now nrdot-host
```

#### DEB-based Systems (Ubuntu, Debian)
```bash
# Add NRDOT repository
curl -fsSL https://download.newrelic.com/nrdot/nrdot.gpg | sudo apt-key add -
echo "deb https://download.newrelic.com/nrdot/linux/ stable main" | \
  sudo tee /etc/apt/sources.list.d/nrdot.list

# Update and install
sudo apt-get update
sudo apt-get install -y nrdot-host

# Configure license key
sudo tee /etc/nrdot/nrdot.env <<EOF
NEW_RELIC_LICENSE_KEY=YOUR_LICENSE_KEY_HERE
EOF

# Enable and start service
sudo systemctl enable --now nrdot-host
```

### 2. Manual Installation

```bash
# Download latest release
ARCH=$(uname -m)
VERSION=$(curl -s https://api.github.com/repos/newrelic/nrdot-host/releases/latest | grep tag_name | cut -d '"' -f 4)
wget https://github.com/newrelic/nrdot-host/releases/download/${VERSION}/nrdot-host_${VERSION}_linux_${ARCH}.tar.gz

# Extract and install
sudo tar -xzf nrdot-host_${VERSION}_linux_${ARCH}.tar.gz -C /usr/local/bin/
sudo chmod +x /usr/local/bin/nrdot-host
sudo chmod +x /usr/local/bin/nrdot-helper

# Create directories
sudo mkdir -p /etc/nrdot /var/lib/nrdot /var/log/nrdot

# Create service user
sudo useradd -r -s /bin/false -d /var/lib/nrdot nrdot
sudo chown -R nrdot:nrdot /var/lib/nrdot /var/log/nrdot

# Install systemd service
sudo cp contrib/systemd/nrdot-host.service /etc/systemd/system/
sudo systemctl daemon-reload
```

### 3. Container Deployment

#### Docker
```bash
# Create configuration
mkdir -p /opt/nrdot/config
cat > /opt/nrdot/config/config.yaml <<EOF
license_key: "\${NEW_RELIC_LICENSE_KEY}"
auto_config:
  enabled: true
  scan_interval: 5m
EOF

# Run with host monitoring
docker run -d \
  --name nrdot-host \
  --restart unless-stopped \
  --privileged \
  --network host \
  --pid host \
  -v /proc:/host/proc:ro \
  -v /sys:/host/sys:ro \
  -v /opt/nrdot/config:/etc/nrdot:ro \
  -v /opt/nrdot/data:/var/lib/nrdot \
  -e NEW_RELIC_LICENSE_KEY=YOUR_LICENSE_KEY_HERE \
  newrelic/nrdot-host:latest
```

#### Docker Compose
See `examples/docker-compose/production.yml` for a complete example.

### 4. Kubernetes Deployment

```bash
# Create namespace and secret
kubectl create namespace nrdot-system
kubectl create secret generic nrdot-license \
  --from-literal=license-key=YOUR_LICENSE_KEY_HERE \
  -n nrdot-system

# Deploy DaemonSet
kubectl apply -f https://raw.githubusercontent.com/newrelic/nrdot-host/main/examples/kubernetes/daemonset.yaml

# Verify deployment
kubectl get pods -n nrdot-system
kubectl logs -n nrdot-system -l app=nrdot-host
```

## Configuration

### Basic Configuration

```yaml
# /etc/nrdot/config.yaml
license_key: "${NEW_RELIC_LICENSE_KEY}"

service:
  name: "${HOSTNAME}"
  environment: "production"
  
auto_config:
  enabled: true
  scan_interval: 5m
  remote_config:
    enabled: true
    api_key: "${NEW_RELIC_API_KEY}"
    
processes:
  enabled: true
  top_n: 50
  interval: 60s
  
data_dir: /var/lib/nrdot
log_dir: /var/log/nrdot

logging:
  level: info
  format: json
  
api:
  enabled: false  # Enable only if needed
  listen_addr: "127.0.0.1:8080"
  auth_token: "${NRDOT_API_TOKEN}"
```

### Advanced Configuration

```yaml
# Advanced security and performance settings
security:
  signing:
    enabled: true
    key_file: /etc/nrdot/signing.key
    verify_remote: true
  
  redaction:
    enabled: true
    patterns:
      - "password"
      - "secret"
      - "token"
      - "key"
      
performance:
  discovery:
    parallel_workers: 4
    timeout: 30s
    
  collection:
    batch_size: 1000
    timeout: 10s
    
  memory:
    limit_mib: 1024
    spike_limit_mib: 256
    
receivers:
  # Host metrics with custom settings
  hostmetrics:
    collection_interval: 60s
    root_path: /host
    scrapers:
      cpu:
        metrics:
          system.cpu.utilization:
            enabled: true
      memory:
        metrics:
          system.memory.utilization:
            enabled: true
            
processors:
  # Security processor first
  nrsecurity:
    enabled: true
    drop_sensitive: true
    
  # Enrichment
  nrenrich:
    host_metadata: true
    cloud_detection: true
    
  # Resource attributes
  resource:
    attributes:
      - key: deployment.environment
        value: "production"
        action: insert
        
  # Batching for efficiency
  batch:
    timeout: 10s
    send_batch_size: 1000
    
exporters:
  otlp/newrelic:
    endpoint: "${OTLP_ENDPOINT:otlp.nr-data.net:4317}"
    headers:
      api-key: "${NEW_RELIC_LICENSE_KEY}"
    compression: gzip
    retry_on_failure:
      enabled: true
      initial_interval: 5s
      max_interval: 30s
```

## Security Hardening

### 1. File Permissions

```bash
# Set proper ownership and permissions
sudo chown -R nrdot:nrdot /etc/nrdot /var/lib/nrdot /var/log/nrdot
sudo chmod 750 /etc/nrdot
sudo chmod 640 /etc/nrdot/*
sudo chmod 755 /var/lib/nrdot
sudo chmod 755 /var/log/nrdot

# Secure the helper binary
sudo chown root:nrdot /usr/local/bin/nrdot-helper
sudo chmod 4750 /usr/local/bin/nrdot-helper
```

### 2. AppArmor Profile

```bash
# Install AppArmor profile
sudo cp contrib/apparmor/nrdot-host /etc/apparmor.d/
sudo apparmor_parser -r /etc/apparmor.d/nrdot-host
```

### 3. SELinux Policy

```bash
# Install SELinux policy module
sudo semodule -i contrib/selinux/nrdot-host.pp

# Set context
sudo semanage fcontext -a -t nrdot_exec_t '/usr/local/bin/nrdot-host'
sudo restorecon -v /usr/local/bin/nrdot-host
```

### 4. Network Security

```bash
# Firewall rules (iptables)
# Allow outbound HTTPS
sudo iptables -A OUTPUT -p tcp --dport 443 -m state --state NEW,ESTABLISHED -j ACCEPT

# Restrict API access to localhost only
sudo iptables -A INPUT -p tcp --dport 8080 -s 127.0.0.1 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 8080 -j DROP
```

### 5. Secrets Management

```bash
# Use environment file with restricted permissions
sudo touch /etc/nrdot/nrdot.env
sudo chmod 600 /etc/nrdot/nrdot.env
sudo chown nrdot:nrdot /etc/nrdot/nrdot.env

# Store secrets
cat <<EOF | sudo tee /etc/nrdot/nrdot.env
NEW_RELIC_LICENSE_KEY=your-license-key
NEW_RELIC_API_KEY=your-api-key
MYSQL_MONITOR_PASS=mysql-password
POSTGRES_MONITOR_PASS=postgres-password
EOF
```

## Performance Tuning

### 1. CPU Optimization

```yaml
# Adjust worker counts based on CPU cores
performance:
  discovery:
    parallel_workers: 8  # For 8+ core systems
    
  collection:
    worker_pool_size: 4
```

### 2. Memory Optimization

```yaml
# Memory limits for different workloads
# Low memory (2GB systems)
performance:
  memory:
    limit_mib: 512
    spike_limit_mib: 128
    
# Standard (4GB systems)
performance:
  memory:
    limit_mib: 1024
    spike_limit_mib: 256
    
# High memory (8GB+ systems)
performance:
  memory:
    limit_mib: 2048
    spike_limit_mib: 512
```

### 3. Collection Intervals

```yaml
# Adjust based on monitoring needs
# Real-time monitoring
receivers:
  hostmetrics:
    collection_interval: 15s
    
processes:
  interval: 30s
  
# Standard monitoring
receivers:
  hostmetrics:
    collection_interval: 60s
    
processes:
  interval: 60s
  
# Low-frequency monitoring
receivers:
  hostmetrics:
    collection_interval: 300s
    
processes:
  interval: 300s
```

## Monitoring and Alerting

### 1. Health Checks

```bash
# Command line health check
nrdot-host status

# HTTP health endpoint
curl -s http://localhost:8080/health

# Systemd status
systemctl status nrdot-host
```

### 2. Metrics to Monitor

Key metrics to track:
- CPU usage: < 5% average
- Memory usage: < configured limit
- Discovery duration: < 1 second
- Config generation time: < 100ms
- Export success rate: > 99%
- API response time: < 50ms p95

### 3. Log Analysis

```bash
# View recent logs
sudo journalctl -u nrdot-host -n 100

# Follow logs
sudo journalctl -u nrdot-host -f

# Check for errors
sudo journalctl -u nrdot-host -p err

# Export logs for analysis
sudo journalctl -u nrdot-host --since "1 hour ago" -o json > nrdot-logs.json
```

### 4. Alerting Rules

Example New Relic alert conditions:
```yaml
# High CPU usage
- name: "NRDOT High CPU"
  query: "SELECT average(cpu.usage) FROM SystemSample WHERE hostname LIKE 'nrdot-%' FACET hostname"
  threshold: 80
  duration: 5
  
# Memory limit approaching
- name: "NRDOT Memory Warning"
  query: "SELECT average(memory.usage) FROM SystemSample WHERE hostname LIKE 'nrdot-%' FACET hostname"
  threshold: 90
  duration: 5
  
# Export failures
- name: "NRDOT Export Failures"
  query: "SELECT count(*) FROM NrdotExportError FACET error_type"
  threshold: 10
  duration: 5
```

## Troubleshooting

### Common Issues

1. **Service won't start**
   ```bash
   # Check logs
   sudo journalctl -u nrdot-host -n 50
   
   # Verify configuration
   sudo nrdot-host validate --config /etc/nrdot/config.yaml
   
   # Check permissions
   ls -la /etc/nrdot /var/lib/nrdot /var/log/nrdot
   ```

2. **No data in New Relic**
   ```bash
   # Check connectivity
   curl -v https://otlp.nr-data.net:443
   
   # Verify license key
   grep license_key /etc/nrdot/config.yaml
   
   # Check export metrics
   nrdot-host status --verbose
   ```

3. **High memory usage**
   ```bash
   # Check current usage
   ps aux | grep nrdot-host
   
   # Review configuration
   grep -A5 memory /etc/nrdot/config.yaml
   
   # Analyze memory profile
   nrdot-host debug profile --type=heap
   ```

4. **Auto-configuration not working**
   ```bash
   # Run discovery manually
   sudo nrdot-host discover --verbose
   
   # Check helper permissions
   ls -la /usr/local/bin/nrdot-helper
   
   # Review discovery logs
   grep discovery /var/log/nrdot/nrdot.log
   ```

### Debug Mode

```bash
# Run in debug mode
sudo nrdot-host run --log-level=debug --config=/etc/nrdot/config.yaml

# Enable debug logging in config
logging:
  level: debug
  
# Capture debug bundle
sudo nrdot-host debug bundle --output=/tmp/nrdot-debug.tar.gz
```

## Backup and Recovery

### Configuration Backup

```bash
# Backup script
#!/bin/bash
BACKUP_DIR="/backup/nrdot/$(date +%Y%m%d)"
mkdir -p "$BACKUP_DIR"

# Backup configuration
cp -r /etc/nrdot "$BACKUP_DIR/"

# Backup data
tar -czf "$BACKUP_DIR/nrdot-data.tar.gz" /var/lib/nrdot

# Backup logs (last 7 days)
journalctl -u nrdot-host --since "7 days ago" > "$BACKUP_DIR/nrdot-logs.txt"

# Create metadata
cat > "$BACKUP_DIR/metadata.json" <<EOF
{
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "version": "$(nrdot-host version)",
  "hostname": "$(hostname)"
}
EOF
```

### Recovery Procedure

```bash
# Stop service
sudo systemctl stop nrdot-host

# Restore configuration
sudo cp -r /backup/nrdot/20240115/nrdot/* /etc/nrdot/

# Restore data (optional)
sudo tar -xzf /backup/nrdot/20240115/nrdot-data.tar.gz -C /

# Verify configuration
sudo nrdot-host validate --config /etc/nrdot/config.yaml

# Start service
sudo systemctl start nrdot-host
```

## Migration from Infrastructure Agent

### Pre-Migration Assessment

```bash
# Check current Infrastructure Agent
sudo newrelic-infra --version
sudo newrelic-infra --show_config

# List custom integrations
ls -la /etc/newrelic-infra/integrations.d/
ls -la /var/db/newrelic-infra/custom-integrations/

# Backup current configuration
sudo cp -r /etc/newrelic-infra /backup/
```

### Migration Process

```bash
# 1. Install NRDOT-HOST (don't start yet)
sudo yum install -y nrdot-host
sudo systemctl disable nrdot-host

# 2. Run migration tool
sudo nrdot-host migrate-infra --dry-run
sudo nrdot-host migrate-infra --preserve

# 3. Review migrated configuration
cat /etc/nrdot/config.yaml
cat /etc/nrdot/migrated-integrations.yaml

# 4. Stop Infrastructure Agent
sudo systemctl stop newrelic-infra
sudo systemctl disable newrelic-infra

# 5. Start NRDOT-HOST
sudo systemctl enable --now nrdot-host

# 6. Verify data flow
nrdot-host status
# Check New Relic UI for data
```

### Post-Migration Validation

```bash
# Compare metrics
# Old: Infrastructure Agent query
# New: NRDOT-HOST query

# Verify all integrations working
nrdot-host discover
grep -i error /var/log/nrdot/nrdot.log

# Monitor for 24 hours before removing Infrastructure Agent
```

## Best Practices

### 1. Deployment Strategy

- **Staged Rollout**: Deploy to non-production first
- **Canary Deployment**: Start with 5-10% of hosts
- **Blue-Green**: Use for configuration updates
- **Monitoring**: Watch metrics during rollout

### 2. Configuration Management

- Use configuration management tools (Ansible, Puppet, Chef)
- Version control configuration files
- Use environment variables for secrets
- Implement configuration validation in CI/CD

### 3. Operational Excellence

- Regular updates (monthly)
- Automated health checks
- Log aggregation and analysis
- Performance baseline establishment
- Disaster recovery testing

### 4. Security Practices

- Regular security updates
- Credential rotation (quarterly)
- Audit log review
- Principle of least privilege
- Network segmentation

## Support and Resources

### Documentation
- Official Docs: https://docs.newrelic.com/nrdot
- API Reference: https://docs.newrelic.com/nrdot/api
- Troubleshooting: https://docs.newrelic.com/nrdot/troubleshooting

### Community
- GitHub: https://github.com/newrelic/nrdot-host
- Forum: https://discuss.newrelic.com/c/nrdot
- Slack: #nrdot-host

### Support
- New Relic Support: https://support.newrelic.com
- Enterprise Support: Available 24/7
- Community Support: Best effort

## Appendix

### A. Environment Variables

```bash
# Core settings
NEW_RELIC_LICENSE_KEY     # Required: New Relic license key
NEW_RELIC_API_KEY        # Optional: API key for remote config
OTLP_ENDPOINT           # Optional: Custom OTLP endpoint

# Service settings
NRDOT_SERVICE_NAME      # Service name (default: hostname)
NRDOT_ENVIRONMENT       # Environment name
NRDOT_LOG_LEVEL        # Log level (debug, info, warn, error)

# Auto-config settings
NRDOT_AUTO_CONFIG       # Enable auto-configuration (true/false)
NRDOT_SCAN_INTERVAL     # Discovery interval (default: 5m)

# Service credentials
MYSQL_MONITOR_USER      # MySQL monitoring user
MYSQL_MONITOR_PASS      # MySQL monitoring password
POSTGRES_MONITOR_USER   # PostgreSQL monitoring user
POSTGRES_MONITOR_PASS   # PostgreSQL monitoring password
REDIS_PASSWORD          # Redis password
```

### B. Configuration Schema

See `/schemas/config-schema.json` for complete JSON schema.

### C. Metrics Reference

See `METRICS_REFERENCE.md` for complete list of collected metrics.

### D. API Reference

See `API_REFERENCE.md` for complete API documentation.