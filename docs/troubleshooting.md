# NRDOT-HOST Troubleshooting Guide

This guide helps you diagnose and resolve common issues with NRDOT-HOST.

## Table of Contents

- [Quick Diagnostics](#quick-diagnostics)
- [Common Issues](#common-issues)
  - [Installation Issues](#installation-issues)
  - [Service Won't Start](#service-wont-start)
  - [No Data in New Relic](#no-data-in-new-relic)
  - [High Resource Usage](#high-resource-usage)
  - [Configuration Errors](#configuration-errors)
  - [Network Issues](#network-issues)
- [Component-Specific Issues](#component-specific-issues)
- [Debug Mode](#debug-mode)
- [Log Analysis](#log-analysis)
- [Performance Issues](#performance-issues)
- [Emergency Procedures](#emergency-procedures)
- [Getting Help](#getting-help)

## Quick Diagnostics

### Health Check Script

Run this script for a quick system check:

```bash
#!/bin/bash
# NRDOT Quick Diagnostics

echo "=== NRDOT System Check ==="

# Check service status
echo -n "Service Status: "
if systemctl is-active --quiet nrdot-host; then
    echo "✓ Running"
else
    echo "✗ Not Running"
fi

# Check configuration
echo -n "Configuration: "
if nrdot-ctl config validate &>/dev/null; then
    echo "✓ Valid"
else
    echo "✗ Invalid"
fi

# Check connectivity
echo -n "New Relic Connection: "
if nrdot-ctl test connection &>/dev/null; then
    echo "✓ Connected"
else
    echo "✗ Disconnected"
fi

# Check resources
echo -n "Resources: "
MEM=$(ps aux | grep nrdot | awk '{sum+=$6} END {print sum/1024}')
echo "Memory: ${MEM}MB"

# Check logs for errors
echo -n "Recent Errors: "
ERRORS=$(journalctl -u nrdot-host --since "5 minutes ago" | grep -c ERROR)
echo "$ERRORS"
```

### Quick Commands

```bash
# Check status
nrdot-ctl status

# View recent logs
nrdot-ctl logs --tail 50

# Test configuration
nrdot-ctl config validate

# Test connection
nrdot-ctl test all

# Get diagnostics bundle
nrdot-ctl diagnostics --output diagnostics.tar.gz
```

## Common Issues

### Installation Issues

#### Problem: Package installation fails

**Symptoms:**
- Error during `apt-get install` or `yum install`
- Missing dependencies
- GPG key errors

**Solutions:**

1. **Update package lists:**
   ```bash
   # Ubuntu/Debian
   sudo apt-get update
   
   # RHEL/CentOS
   sudo yum clean all && sudo yum update
   ```

2. **Install missing dependencies:**
   ```bash
   # Ubuntu/Debian
   sudo apt-get install -y ca-certificates curl gnupg
   
   # RHEL/CentOS
   sudo yum install -y ca-certificates curl
   ```

3. **Fix GPG key issues:**
   ```bash
   # Re-import GPG key
   curl -fsSL https://packages.newrelic.com/nrdot/gpg.key | sudo apt-key add -
   ```

4. **Manual installation:**
   ```bash
   # Download and install manually
   wget https://github.com/deepaucksharma/nrdot-host/releases/latest/download/nrdot-host.deb
   sudo dpkg -i nrdot-host.deb
   sudo apt-get install -f  # Fix dependencies
   ```

### Service Won't Start

#### Problem: NRDOT service fails to start

**Symptoms:**
- `systemctl start nrdot-host` fails
- Service exits immediately
- Status shows "failed" or "inactive"

**Diagnostics:**

```bash
# Check service status
sudo systemctl status nrdot-host -l

# View detailed logs
sudo journalctl -xeu nrdot-host -n 100

# Check for port conflicts
sudo netstat -tlnp | grep -E '8080|9090|4317|4318'

# Verify permissions
ls -la /etc/nrdot/
ls -la /var/lib/nrdot/
ls -la /var/log/nrdot/
```

**Common Causes and Solutions:**

1. **Invalid Configuration:**
   ```bash
   # Validate configuration
   sudo nrdot-ctl config validate
   
   # Common issues:
   # - Missing license key
   # - Invalid YAML syntax
   # - Incorrect field names
   ```

2. **Permission Issues:**
   ```bash
   # Fix ownership
   sudo chown -R nrdot:nrdot /etc/nrdot
   sudo chown -R nrdot:nrdot /var/lib/nrdot
   sudo chown -R nrdot:nrdot /var/log/nrdot
   
   # Fix permissions
   sudo chmod 750 /etc/nrdot
   sudo chmod 600 /etc/nrdot/config.yaml
   ```

3. **Port Already in Use:**
   ```bash
   # Find process using port
   sudo lsof -i :8080
   
   # Change port in config
   sudo vi /etc/nrdot/config.yaml
   # api:
   #   bind_address: "127.0.0.1:8081"
   ```

4. **Missing Dependencies:**
   ```bash
   # Check binary dependencies
   ldd /usr/bin/nrdot-collector
   
   # Install missing libraries
   sudo apt-get install -y libc6
   ```

### No Data in New Relic

#### Problem: NRDOT is running but no data appears in New Relic

**Diagnostics:**

```bash
# Test connection to New Relic
nrdot-ctl test connection --verbose

# Check exporter status
nrdot-ctl status exporters

# View exporter logs
nrdot-ctl logs --component exporter --tail 50

# Check metrics
curl -s localhost:9090/metrics | grep exporter_sent_
```

**Common Causes and Solutions:**

1. **Invalid License Key:**
   ```bash
   # Verify license key format
   nrdot-ctl config get license_key
   
   # Should be 40 characters, starting with region code
   # US: starts with "xxx...NRAL"
   # EU: starts with "eu01xx...NRAL"
   
   # Update license key
   sudo nrdot-ctl config set license_key YOUR_CORRECT_KEY
   sudo systemctl restart nrdot-host
   ```

2. **Network Connectivity:**
   ```bash
   # Test connectivity
   curl -I https://otlp.nr-data.net
   
   # Check firewall
   sudo iptables -L -n | grep 443
   
   # Test with proxy
   export HTTPS_PROXY=http://proxy:8080
   nrdot-ctl test connection
   ```

3. **Wrong Region/Endpoint:**
   ```bash
   # Check configured endpoint
   nrdot-ctl config get api_endpoint
   
   # Set correct endpoint
   # US: https://otlp.nr-data.net
   # EU: https://otlp.eu01.nr-data.net
   
   sudo nrdot-ctl config set api_endpoint https://otlp.eu01.nr-data.net
   ```

4. **Data Not Being Collected:**
   ```bash
   # Check receiver status
   nrdot-ctl status receivers
   
   # Verify data sources
   nrdot-ctl test receivers
   
   # Enable debug mode
   nrdot-ctl debug enable --duration 5m
   ```

### High Resource Usage

#### Problem: NRDOT consuming too much CPU/Memory

**Diagnostics:**

```bash
# Check resource usage
nrdot-ctl metrics resources

# Top processes
top -p $(pgrep -f nrdot)

# Memory breakdown
nrdot-ctl debug memory

# CPU profile
nrdot-ctl profile cpu --duration 30s
```

**Common Causes and Solutions:**

1. **High Cardinality:**
   ```bash
   # Check cardinality
   nrdot-ctl metrics cardinality --top 10
   
   # Add cardinality limits
   sudo vi /etc/nrdot/config.yaml
   ```
   
   ```yaml
   processors:
     nrcap:
       limits:
         global: 50000
         metrics:
           "http.request.duration": 10000
   ```

2. **Memory Leaks:**
   ```bash
   # Enable memory profiling
   nrdot-ctl profile memory --output mem.prof
   
   # Set memory limits
   sudo vi /etc/nrdot/config.yaml
   ```
   
   ```yaml
   performance:
     memory:
       soft_limit: "400MiB"
       hard_limit: "512MiB"
       ballast_size: "200MiB"
   ```

3. **Inefficient Configuration:**
   ```bash
   # Reduce collection frequency
   sudo vi /etc/nrdot/config.yaml
   ```
   
   ```yaml
   metrics:
     interval: "60s"  # Instead of 10s
   
   # Disable unused features
   logs:
     enabled: false
   
   traces:
     enabled: false
   ```

4. **Too Many Processors:**
   ```bash
   # Disable expensive processors
   sudo vi /etc/nrdot/config.yaml
   ```
   
   ```yaml
   processors:
     nrenrich:
       kubernetes:
         enabled: false  # If not in K8s
   ```

### Configuration Errors

#### Problem: Configuration validation fails

**Common Errors:**

1. **YAML Syntax Error:**
   ```bash
   Error: yaml: line 23: found character '\t' that cannot start any token
   
   # Fix: Use spaces, not tabs
   # Validate YAML
   yamllint /etc/nrdot/config.yaml
   ```

2. **Missing Required Fields:**
   ```bash
   Error: service.name is required
   
   # Fix: Add required fields
   service:
     name: "my-service"
   ```

3. **Type Mismatch:**
   ```bash
   Error: metrics.interval must be a duration string
   
   # Fix: Use proper format
   metrics:
     interval: "30s"  # Not just "30"
   ```

4. **Invalid License Key:**
   ```bash
   Error: license_key format is invalid
   
   # Fix: Check key format
   # Should be 40 characters
   license_key: "eu01xx0000000000000000000000000000NRAL"
   ```

### Network Issues

#### Problem: Network-related errors

**Diagnostics:**

```bash
# DNS resolution
nslookup otlp.nr-data.net

# Connection test
telnet otlp.nr-data.net 443

# SSL/TLS test
openssl s_client -connect otlp.nr-data.net:443

# Proxy test
curl -x $HTTPS_PROXY https://otlp.nr-data.net
```

**Solutions:**

1. **Proxy Configuration:**
   ```bash
   # Set proxy environment
   export HTTPS_PROXY=http://proxy.company.com:8080
   export NO_PROXY=localhost,127.0.0.1
   
   # Or in systemd
   sudo systemctl edit nrdot-host
   # Add:
   [Service]
   Environment="HTTPS_PROXY=http://proxy:8080"
   ```

2. **SSL/TLS Issues:**
   ```bash
   # Update CA certificates
   sudo update-ca-certificates
   
   # Disable verification (NOT for production)
   export NRDOT_INSECURE_SKIP_VERIFY=true
   ```

3. **DNS Issues:**
   ```bash
   # Add DNS servers
   echo "nameserver 8.8.8.8" | sudo tee -a /etc/resolv.conf
   
   # Or use IP directly
   sudo nrdot-ctl config set api_endpoint https://162.247.241.2
   ```

## Component-Specific Issues

### Supervisor Issues

```bash
# Supervisor won't start collector
nrdot-ctl logs --component supervisor

# Common issues:
# - Binary not found
# - Permission denied
# - Config generation failed

# Solutions:
# Check binary path
which nrdot-collector

# Fix permissions
sudo chmod +x /usr/bin/nrdot-collector
```

### Config Engine Issues

```bash
# Config generation fails
nrdot-ctl debug config-engine

# Common issues:
# - Template errors
# - Schema validation fails

# Solutions:
# Validate schema
nrdot-ctl schema validate

# Regenerate config
nrdot-ctl config regenerate
```

### API Server Issues

```bash
# API not responding
curl -f http://localhost:8080/v1/health

# Common issues:
# - Port binding failed
# - TLS certificate issues

# Solutions:
# Check port binding
sudo netstat -tlnp | grep 8080

# Regenerate certificates
nrdot-ctl certificates regenerate
```

### Processor Issues

```bash
# Processor errors
nrdot-ctl logs --processor nrsecurity

# Common issues:
# - Regex pattern errors
# - Memory exhaustion
# - Processing timeouts

# Solutions:
# Validate patterns
nrdot-ctl test processor nrsecurity

# Increase timeout
processors:
  nrsecurity:
    timeout: "10s"
```

## Debug Mode

### Enable Debug Logging

```bash
# Temporary debug mode (5 minutes)
nrdot-ctl debug enable --duration 5m

# Persistent debug mode
sudo nrdot-ctl config set log_level debug
sudo systemctl restart nrdot-host

# Component-specific debug
nrdot-ctl debug enable --component supervisor
```

### Debug Output

```yaml
# In configuration
service:
  telemetry:
    logs:
      level: debug
      development: true
      
# Debug specific processors
processors:
  nrsecurity:
    debug: true
    
  nrenrich:
    debug: true
```

### Collecting Debug Information

```bash
# Generate diagnostics bundle
nrdot-ctl diagnostics \
  --include-config \
  --include-logs \
  --include-metrics \
  --output diagnostics.tar.gz

# What's included:
# - Configuration (sanitized)
# - Logs (last 1000 lines)
# - Metrics snapshot
# - System information
# - Component status
```

## Log Analysis

### Log Locations

```bash
# System logs
/var/log/nrdot/collector.log
/var/log/nrdot/supervisor.log
/var/log/nrdot/api-server.log

# Systemd journal
journalctl -u nrdot-host

# Container logs
docker logs nrdot-host
kubectl logs -n nrdot-system nrdot-host-xxxxx
```

### Log Patterns

```bash
# Find errors
grep ERROR /var/log/nrdot/collector.log

# Find warnings
grep WARN /var/log/nrdot/collector.log

# Find specific component
grep "component=nrsecurity" /var/log/nrdot/collector.log

# Find by time range
journalctl -u nrdot-host --since "10 minutes ago"
```

### Common Log Messages

```
# Normal startup
INFO Starting NRDOT supervisor version=1.0.0
INFO Configuration loaded path=/etc/nrdot/config.yaml
INFO Starting OpenTelemetry Collector
INFO Everything is ready. Begin running and processing data.

# Connection established
INFO Exporter started endpoint=otlp.nr-data.net:443

# Processing data
DEBUG Metrics received count=150
DEBUG Traces received count=50
INFO Metrics sent successfully count=150

# Common warnings
WARN High cardinality detected metric=http.request.duration series=50000
WARN Memory usage high current=450MB limit=512MB
WARN Failed to enrich with K8s metadata error="connection refused"

# Common errors
ERROR Failed to export metrics error="context deadline exceeded"
ERROR Configuration error error="invalid license key format"
ERROR Processor panic processor=nrtransform error="division by zero"
```

## Performance Issues

### Slow Processing

```bash
# Check processing rate
nrdot-ctl metrics throughput

# Identify bottlenecks
nrdot-ctl profile pipeline

# Common solutions:
# 1. Increase batch size
processors:
  batch:
    size: 2000
    timeout: "5s"

# 2. Reduce processor complexity
processors:
  nrenrich:
    kubernetes:
      enabled: false

# 3. Add more workers
performance:
  concurrency:
    workers: 20
```

### Memory Growth

```bash
# Monitor memory
nrdot-ctl monitor memory --interval 10s

# Find leaks
nrdot-ctl profile memory --type heap

# Solutions:
# 1. Set memory limits
performance:
  memory:
    hard_limit: "1GiB"
    
# 2. Enable garbage collection
performance:
  gc:
    enabled: true
    target_percentage: 80
```

### High CPU Usage

```bash
# CPU profiling
nrdot-ctl profile cpu --duration 60s --output cpu.prof

# Analyze profile
go tool pprof cpu.prof

# Common causes:
# 1. Regex processing
# 2. Metric calculations
# 3. Serialization

# Solutions:
# 1. Optimize regex patterns
# 2. Reduce calculation frequency
# 3. Enable compression
```

## Emergency Procedures

### Service Crash Loop

```bash
# Stop the service
sudo systemctl stop nrdot-host

# Clear bad state
sudo rm -rf /var/lib/nrdot/state/*

# Reset to defaults
sudo cp /usr/share/nrdot/config.yaml.default /etc/nrdot/config.yaml

# Start with minimal config
sudo nrdot-supervisor --config /usr/share/nrdot/minimal.yaml

# Gradually add features back
```

### Data Loss Prevention

```bash
# Enable persistent queue
exporters:
  newrelic:
    sending_queue:
      enabled: true
      storage: file_storage
      
# Configure storage
extensions:
  file_storage:
    directory: /var/lib/nrdot/queue
    timeout: 10s
```

### Rollback Procedure

```bash
# List installed versions
dpkg -l | grep nrdot-host

# Downgrade to previous version
sudo apt-get install nrdot-host=1.0.0

# Or restore from backup
sudo tar -xzf /backup/nrdot-config-backup.tar.gz -C /
```

### Emergency Shutdown

```bash
# Graceful shutdown
nrdot-ctl shutdown --graceful --timeout 30s

# Force shutdown
sudo systemctl kill -s KILL nrdot-host

# Disable auto-restart
sudo systemctl disable nrdot-host
```

## Getting Help

### Self-Service Resources

1. **Built-in Help:**
   ```bash
   nrdot-ctl help
   nrdot-ctl <command> --help
   ```

2. **Documentation:**
   ```bash
   # Local docs
   man nrdot-ctl
   
   # Online docs
   https://github.com/deepaucksharma/nrdot-host/docs
   ```

3. **Diagnostics:**
   ```bash
   # Self-diagnostics
   nrdot-ctl doctor
   
   # Generate report
   nrdot-ctl report --output report.html
   ```

### Community Support

- **GitHub Issues**: https://github.com/deepaucksharma/nrdot-host/issues
- **Discussions**: https://github.com/deepaucksharma/nrdot-host/discussions
- **Stack Overflow**: Tag with `nrdot-host`

### When Reporting Issues

Include:

1. **Version Information:**
   ```bash
   nrdot-ctl version --verbose
   ```

2. **Configuration (sanitized):**
   ```bash
   nrdot-ctl config show --sanitize
   ```

3. **Error Messages:**
   ```bash
   journalctl -u nrdot-host --since "1 hour ago" > error.log
   ```

4. **System Information:**
   ```bash
   uname -a
   cat /etc/os-release
   free -h
   df -h
   ```

5. **Steps to Reproduce**

### Emergency Contacts

For critical production issues:

- **New Relic Support**: https://support.newrelic.com
- **Security Issues**: security@newrelic.com

## Quick Reference Card

```bash
# Status checks
nrdot-ctl status              # Overall status
nrdot-ctl health              # Health check
nrdot-ctl test connection     # Test New Relic connection

# Logs
nrdot-ctl logs --tail 50      # Recent logs
journalctl -u nrdot-host -f   # Follow logs

# Configuration
nrdot-ctl config validate     # Validate config
nrdot-ctl config show         # Show running config

# Debugging
nrdot-ctl debug enable        # Enable debug mode
nrdot-ctl diagnostics         # Generate diagnostics

# Metrics
nrdot-ctl metrics cardinality # Check cardinality
nrdot-ctl metrics throughput  # Check throughput

# Service control
sudo systemctl restart nrdot-host  # Restart service
sudo systemctl status nrdot-host   # Service status
```