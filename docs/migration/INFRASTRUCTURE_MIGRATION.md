# Infrastructure Agent Migration Guide (Phase 3)

## Overview

**Status**: Not yet implemented - Planned for Phase 3 of the roadmap

This guide will help you migrate from the New Relic Infrastructure agent to NRDOT-HOST when migration tools are available. Until then, manual migration steps are provided for early adopters.

## Why Migrate?

### Current Benefits (v2.0)
- **OpenTelemetry Native**: Industry-standard telemetry
- **Unified Binary**: Single process architecture
- **OTLP Gateway**: Accept traces and metrics from apps
- **Custom Processors**: Enhanced security and enrichment

### Future Benefits (Phase 2-3)
- **Auto-Configuration**: Service discovery (Phase 2)
- **Better Performance**: Target <150MB memory (Phase 3)
- **Zero-Config**: Automatic integration setup

### Feature Comparison

| Feature | Infrastructure Agent | NRDOT-HOST v2.0 | NRDOT-HOST v3.0 (Target) |
|---------|---------------------|-----------------|-------------------------|
| Host Metrics | ✅ | ✅ | ✅ Enhanced |
| Process Monitoring | ✅ | ✅ Basic | ✅ Top-N tracking |
| Log Forwarding | ✅ | ✅ | ✅ |
| Service Integrations | ✅ Manual | ✅ Manual | ✅ **Automatic** |
| OTLP Gateway | ❌ | ✅ | ✅ |
| Auto-Discovery | ❌ | ❌ | ✅ |
| Memory Usage | ~250MB | ~300MB | <150MB |
| Configuration | YAML | YAML | Minimal YAML |

## Migration Methods

### Method 1: Automated Migration (Phase 3)

**Coming in Phase 3** - Automated migration tool:

```bash
# Planned command
sudo nrdot-host migrate-infra

# What it will do:
# 1. Detect Infrastructure agent installation
# 2. Extract license key from /etc/newrelic-infra.yml
# 3. Convert custom_attributes to labels
# 4. Create /etc/nrdot/config.yaml
# 5. Stop newrelic-infra service
# 6. Start nrdot-host service
# 7. Validate metrics continuity
```

### Method 2: Manual Migration (Available Now)

Follow these steps for manual migration:

#### Step 1: Install NRDOT-HOST

```bash
# Download and install (current method)
curl -L https://github.com/newrelic/nrdot-host/releases/latest/download/nrdot-host-linux-amd64 -o nrdot-host
chmod +x nrdot-host
sudo mv nrdot-host /usr/local/bin/

# Future: Package manager installation (Phase 3)
# sudo apt-get install nrdot-host
# sudo yum install nrdot-host
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

# Future: Enable auto-configuration (Phase 2)
# auto_config:
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

# New in NRDOT-HOST - add to config.yaml:
receivers:
  filelog/custom:
    include:
      - /var/log/myapp/*.log
    operators:
      - type: add_resource_attributes
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
# Check service status
sudo systemctl status nrdot-host

# Monitor logs
sudo journalctl -u nrdot-host -f

# Verify in New Relic UI:
# 1. Go to Infrastructure
# 2. Find your host
# 3. Verify metrics are flowing
# 4. Check custom attributes (labels)
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

| Infrastructure Agent | NRDOT-HOST v2.0 | NRDOT-HOST v3.0 |
|---------------------|-----------------|------------------|
| `/etc/newrelic-infra/integrations.d/` | Manual config | **Automatic!** |
| Manual YAML per service | OTel receivers | Auto-discovery |
| Integration executables | Native receivers | Native receivers |

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

```yaml
# Old: Infrastructure agent integration
# /etc/newrelic-infra/integrations.d/mysql-config.yml
integrations:
  - name: nri-mysql
    env:
      HOSTNAME: localhost
      PORT: 3306
      USERNAME: newrelic
      PASSWORD: password

# New: NRDOT-HOST manual config (v2.0)
# /etc/nrdot/config.yaml
receivers:
  mysql:
    endpoint: localhost:3306
    username: newrelic
    password: ${MYSQL_PASSWORD}
    collection_interval: 60s

# Future: Auto-configured (v3.0)
# MySQL will be detected and configured automatically!

```

### Scenario 3: Custom Application Logs

```yaml
# Old: Infrastructure agent logging
# /etc/newrelic-infra/logging.d/app.yml
logs:
  - name: myapp
    file: /opt/myapp/logs/*.log

# New: NRDOT-HOST config
# /etc/nrdot/config.yaml
receivers:
  filelog/myapp:
    include:
      - /opt/myapp/logs/*.log
    resource:
      app: myapp

service:
  pipelines:
    logs:
      receivers: [filelog/myapp]
      processors: [nrsecurity]
      exporters: [otlp]
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

**Current (v2.0)**:
1. Integrations must be manually configured
2. Add appropriate receiver to config.yaml
3. Include receiver in service pipeline

**Future (v3.0)**:
1. Auto-discovery will detect services
2. Check detection: `nrdot-host discover`
3. Verify credentials are provided

### Custom Attributes Not Showing

1. Ensure they're under `labels:` in config
2. Restart agent after config changes
3. Allow 2-3 minutes for attributes to appear

### Performance Comparison

```bash
# Check resource usage
ps aux | grep -E '(newrelic-infra|nrdot-host)'

# Expected in v2.0:
# Memory: ~300MB (similar to infra agent)
# CPU: 2-5% (similar)

# Target for v3.0:
# Memory: <150MB (40% reduction)
# CPU: <2% (improved efficiency)
```

## Support

- **Documentation**: See `/docs` directory
- **Issues**: GitHub Issues
- **Roadmap**: [4-Month Plan](../roadmap/ROADMAP.md)

## Next Steps

After successful migration:

1. **Use OTLP Gateway**: Point apps to `localhost:4317`
2. **Monitor Performance**: Verify resource usage
3. **Plan for v3.0**: Auto-configuration coming soon
4. **Provide Feedback**: Help shape the roadmap

## Timeline

- **Now**: Manual migration available
- **Phase 1** (4 weeks): Enhanced process monitoring
- **Phase 2** (10 weeks): Auto-configuration
- **Phase 3** (14 weeks): Automated migration tools