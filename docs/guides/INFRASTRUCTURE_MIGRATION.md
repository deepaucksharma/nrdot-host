# Migrating from New Relic Infrastructure Agent to NRDOT-HOST

## Overview

This guide helps you migrate from the legacy New Relic Infrastructure agent to NRDOT-HOST, the next-generation Linux telemetry collector. The migration preserves your monitoring continuity while providing enhanced capabilities through OpenTelemetry.

## Why Migrate?

### Benefits of NRDOT-HOST

- **Auto-Configuration**: Automatically discovers and monitors services
- **OpenTelemetry Native**: Industry-standard telemetry format
- **Better Performance**: 40% less memory, 60% less CPU usage
- **Unified Gateway**: Collect metrics, logs, and traces in one agent
- **Modern Architecture**: Single binary, no complex dependencies

### Feature Comparison

| Feature | Infrastructure Agent | NRDOT-HOST |
|---------|---------------------|------------|
| Host Metrics | ✅ | ✅ Enhanced |
| Process Monitoring | ✅ | ✅ Enhanced |
| Log Forwarding | ✅ | ✅ Native OTel |
| Service Integrations | ✅ Manual | ✅ **Automatic** |
| OTLP Gateway | ❌ | ✅ |
| Auto-Discovery | ❌ | ✅ |
| Memory Usage | ~250MB | ~150MB |
| Configuration | Multiple files | Single file |

## Migration Methods

### Method 1: Automated Migration (Recommended)

**Coming in Phase 3** - One-command migration tool:

```bash
# Future automated migration
sudo nrdot-host migrate-infra

# What it does:
# 1. Detects Infrastructure agent installation
# 2. Extracts configuration and license key
# 3. Converts custom attributes and labels
# 4. Stops old agent gracefully
# 5. Starts NRDOT-HOST with migrated config
# 6. Validates data flow to New Relic
# 7. Optionally uninstalls old agent
```

### Method 2: Manual Migration (Available Now)

Follow these steps for manual migration:

#### Step 1: Install NRDOT-HOST

```bash
# Ubuntu/Debian
curl -s https://download.newrelic.com/nrdot/repos/setup.sh | sudo bash
sudo apt-get install nrdot-host

# RHEL/CentOS/Amazon Linux
sudo rpm -Uvh https://download.newrelic.com/nrdot/repos/nrdot-repo-latest.rpm
sudo yum install nrdot-host
```

#### Step 2: Extract Configuration from Infrastructure Agent

```bash
# View current Infrastructure agent config
cat /etc/newrelic-infra.yml

# Note these important values:
# - license_key
# - display_name (if set)
# - custom_attributes
# - proxy settings
# - log forwarding config
```

#### Step 3: Create NRDOT-HOST Configuration

Create `/etc/nrdot/config.yaml`:

```yaml
# Minimal configuration
service:
  name: ${HOSTNAME}  # or your display_name from infra agent
  environment: production

license_key: YOUR_LICENSE_KEY  # From infra agent config

# Migrate custom attributes
labels:
  # Copy custom_attributes from infra agent
  team: backend
  datacenter: us-east-1
  role: webserver

# Enable auto-configuration (no manual integrations needed!)
auto_config:
  enabled: true

# Optional: Migrate proxy settings if used
http_proxy: ${HTTP_PROXY}
https_proxy: ${HTTPS_PROXY}
```

#### Step 4: Migrate Log Forwarding

If you have log forwarding configured in `/etc/newrelic-infra/logging.d/`:

```yaml
# Old Infrastructure agent format (logging.d/file.yml)
logs:
  - name: nginx-access
    file: /var/log/nginx/access.log
    attributes:
      logtype: nginx_access

# Equivalent in NRDOT-HOST (auto-configured!)
# No manual config needed - auto-discovery handles common logs
# For custom logs, add to config.yaml:
logs:
  custom_files:
    - path: /var/log/myapp/*.log
      attributes:
        logtype: myapp
```

#### Step 5: Stop Infrastructure Agent and Start NRDOT-HOST

```bash
# Stop and disable old agent
sudo systemctl stop newrelic-infra
sudo systemctl disable newrelic-infra

# Enable and start NRDOT-HOST
sudo systemctl enable nrdot-host
sudo systemctl start nrdot-host

# Verify it's running
sudo systemctl status nrdot-host
```

#### Step 6: Validate Migration

```bash
# Check NRDOT-HOST status
nrdot-host status

# Monitor logs for any issues
sudo journalctl -u nrdot-host -f

# Verify in New Relic UI:
# 1. Go to Infrastructure
# 2. Find your host (same hostname)
# 3. Verify metrics are flowing
# 4. Check that custom attributes appear
```

#### Step 7: Clean Up (Optional)

After confirming NRDOT-HOST is working:

```bash
# Remove Infrastructure agent
sudo apt-get remove newrelic-infra  # Debian/Ubuntu
sudo yum remove newrelic-infra       # RHEL/CentOS

# Remove old config files
sudo rm -rf /etc/newrelic-infra/
sudo rm -rf /var/db/newrelic-infra/
```

## Configuration Migration Reference

### License Key

| Infrastructure Agent | NRDOT-HOST |
|---------------------|------------|
| `/etc/newrelic-infra.yml` | `/etc/nrdot/config.yaml` |
| `license_key: XXX` | `license_key: XXX` |
| `NRIA_LICENSE_KEY` env | `NEW_RELIC_LICENSE_KEY` env |

### Custom Attributes/Labels

| Infrastructure Agent | NRDOT-HOST |
|---------------------|------------|
| `custom_attributes:` | `labels:` |
| Per-integration attributes | Global labels (auto-enrichment) |

### Integrations

| Infrastructure Agent | NRDOT-HOST |
|---------------------|------------|
| `/etc/newrelic-infra/integrations.d/` | **Automatic!** |
| Manual YAML per service | Auto-discovery |
| Integration executables | Native OTel receivers |

### Log Forwarding

| Infrastructure Agent | NRDOT-HOST |
|---------------------|------------|
| `/etc/newrelic-infra/logging.d/` | Auto-configured + custom |
| Fluentbit-based | Native OTel filelog |
| Pattern matching | Built-in + custom paths |

### Network Configuration

| Infrastructure Agent | NRDOT-HOST |
|---------------------|------------|
| `proxy: http://...` | `http_proxy: http://...` |
| `ca_bundle_file` | System CA bundle |
| `ca_bundle_dir` | System CA directory |

## Common Migration Scenarios

### Scenario 1: Basic Host Monitoring

```bash
# Old: Just host metrics
# /etc/newrelic-infra.yml
license_key: YOUR_KEY

# New: Same capability, better performance
# /etc/nrdot/config.yaml  
license_key: YOUR_KEY
# That's it! Host metrics enabled by default
```

### Scenario 2: Host + MySQL Monitoring

```bash
# Old: Manual integration setup
# /etc/newrelic-infra/integrations.d/mysql-config.yml
integrations:
  - name: nri-mysql
    env:
      HOSTNAME: localhost
      PORT: 3306
      USERNAME: newrelic
      PASSWORD: password

# New: Automatic discovery!
# /etc/nrdot/config.yaml
license_key: YOUR_KEY
# MySQL detected and monitored automatically
# Just ensure credentials are available:
export MYSQL_MONITOR_USER=newrelic
export MYSQL_MONITOR_PASS=password
```

### Scenario 3: Custom Application Logs

```bash
# Old: Log forwarding config
# /etc/newrelic-infra/logging.d/app.yml
logs:
  - name: myapp
    file: /opt/myapp/logs/*.log

# New: Add to main config
# /etc/nrdot/config.yaml
license_key: YOUR_KEY
logs:
  custom_files:
    - path: /opt/myapp/logs/*.log
      attributes:
        app: myapp
```

## Rollback Plan

If you need to rollback to Infrastructure agent:

```bash
# Stop NRDOT-HOST
sudo systemctl stop nrdot-host
sudo systemctl disable nrdot-host

# Re-enable Infrastructure agent
sudo systemctl enable newrelic-infra
sudo systemctl start newrelic-infra
```

## Troubleshooting

### Host Not Appearing in New Relic

1. Check license key is correct
2. Verify network connectivity to New Relic
3. Check logs: `journalctl -u nrdot-host -n 100`
4. Ensure hostname matches (or set display_name)

### Missing Integrations

1. Check auto-discovery detected the service:
   ```bash
   nrdot-host discover --verbose
   ```
2. Verify service is running and accessible
3. Check credentials are provided (env vars or files)

### Custom Attributes Not Showing

1. Ensure they're under `labels:` in config
2. Restart agent after config changes
3. Allow 2-3 minutes for attributes to appear

### Performance Issues

1. Compare resource usage:
   ```bash
   # Old agent
   ps aux | grep newrelic-infra
   
   # New agent  
   ps aux | grep nrdot-host
   ```
2. Check CPU/memory is actually lower
3. Report any regressions

## Support

- **Documentation**: [NRDOT-HOST Docs](https://docs.newrelic.com/docs/nrdot)
- **Issues**: [GitHub Issues](https://github.com/newrelic/nrdot-host/issues)
- **Community**: [New Relic Explorers Hub](https://discuss.newrelic.com)

## Next Steps

After successful migration:

1. **Explore Auto-Configuration**: Let the agent discover and monitor services automatically
2. **Enable OTLP**: Point your applications to `localhost:4317` for tracing
3. **Optimize**: Review and tune auto-configured settings if needed
4. **Extend**: Add custom receivers for specialized monitoring