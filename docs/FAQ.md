# NRDOT-HOST Frequently Asked Questions

## General Questions

### What is NRDOT-HOST?
NRDOT-HOST is an enterprise-grade distribution of the OpenTelemetry Collector specifically designed for host monitoring. It combines OpenTelemetry's flexibility with New Relic's operational expertise, providing automatic security, enrichment, and cardinality management.

### How is NRDOT-HOST different from vanilla OpenTelemetry?
NRDOT-HOST adds:
- **Automatic secret redaction** - No leaked credentials in your telemetry
- **Built-in PII protection** - Compliance out of the box
- **Smart cardinality limiting** - Prevent cost explosions
- **Zero-config operation** - Works with minimal configuration
- **Cloud/K8s enrichment** - Automatic metadata addition
- **Self-monitoring** - Know when something's wrong

### Is NRDOT-HOST open source?
Yes! NRDOT-HOST is licensed under Apache 2.0. You can use, modify, and distribute it freely.

### What platforms are supported?
- Linux (x86_64, ARM64): Ubuntu, Debian, RHEL, CentOS, Amazon Linux
- macOS (Intel, Apple Silicon)
- Windows Server 2016+
- Docker containers
- Kubernetes

## Installation & Setup

### How do I install NRDOT-HOST?
The easiest way is our one-line installer:
```bash
curl -sSL https://raw.githubusercontent.com/deepaucksharma/nrdot-host/main/install.sh | sudo bash
```

Or use package managers, Docker, or Kubernetes. See the [Installation Guide](./installation.md).

### What's the minimum configuration needed?
Just two lines:
```yaml
service:
  name: "my-service"
license_key: "YOUR_NEW_RELIC_LICENSE_KEY"
```

### Where do I get a New Relic license key?
1. Log into [New Relic](https://one.newrelic.com)
2. Click on your profile (bottom left)
3. Select "API keys"
4. Copy your license key (starts with region code like 'eu01')

### Can I use NRDOT-HOST without New Relic?
Yes! You can configure any OpenTelemetry-compatible backend:
```yaml
exporters:
  otlp:
    endpoint: "your-backend:4317"
```

## Configuration

### How do I enable trace collection?
Add to your config:
```yaml
traces:
  enabled: true
```
The OTLP receiver will automatically listen on port 4317.

### How do I scrape Prometheus metrics?
```yaml
receivers:
  prometheus:
    config:
      scrape_configs:
        - job_name: 'my-app'
          static_configs:
            - targets: ['localhost:9090']
```

### How do I disable secret redaction for debugging?
```yaml
processors:
  nrsecurity:
    enabled: false  # Disables the entire processor
    # OR just disable specific features:
    redact_secrets: false
    redact_pii: false
```

### How do I set cardinality limits?
```yaml
processors:
  nrcap:
    limits:
      global: 50000  # Total series limit
      metrics:
        "http.request.duration": 10000  # Per-metric limit
```

## Performance & Scaling

### How much data can NRDOT-HOST handle?
A single instance can handle:
- 1M+ metrics/second
- 50K+ traces/second
- 200K+ logs/second

Actual performance depends on enabled processors and hardware.

### How much memory does NRDOT-HOST use?
Typical usage:
- Minimum: 256MB
- Recommended: 512MB-1GB
- High volume: 2-4GB

Configure limits:
```yaml
processors:
  memory_limiter:
    limit_mib: 512
```

### How do I run multiple collectors?
1. Deploy multiple instances (different hosts/containers)
2. Use a load balancer for receivers
3. Partition data by type or source
4. See [Scaling Guidelines](./performance.md#scaling-strategy)

### How do I optimize for high volume?
```yaml
processors:
  batch:
    size: 5000        # Larger batches
    timeout: 100ms    # Lower timeout
  nrsecurity:
    enabled: false    # If data is pre-sanitized
  nrcap:
    strategies:
      overflow_action: "drop"  # Fast dropping
```

## Troubleshooting

### NRDOT-HOST won't start
1. Check logs: `sudo journalctl -u nrdot-host -n 100`
2. Validate config: `nrdot-ctl config validate`
3. Check permissions on `/etc/nrdot/config.yaml`
4. Ensure license key is valid

### No data in New Relic
1. Test connection: `nrdot-ctl test connection`
2. Check firewall allows outbound HTTPS (443)
3. Verify license key starts with correct region (US/EU)
4. Look for errors: `nrdot-ctl logs --grep ERROR`

### High memory usage
1. Check cardinality: `nrdot-ctl metrics cardinality`
2. Enable memory limiter (see above)
3. Reduce batch sizes
4. Disable unused processors

### How do I debug issues?
```bash
# Enable debug mode
nrdot-ctl debug enable --duration 5m

# Get diagnostics bundle
nrdot-ctl diagnostics --output debug.tar.gz

# Check component status
nrdot-ctl status --verbose
```

## Security

### Is my data secure?
Yes! NRDOT-HOST:
- Automatically redacts secrets and PII
- Uses TLS for all external connections
- Runs as non-root user
- Supports authentication and authorization

### What gets redacted by default?
- Passwords (password=, pwd:, pass:)
- API keys (api_key=, apikey:, X-API-Key:)
- Tokens (token=, auth:, Bearer)
- Credit card numbers
- Social Security Numbers
- Email addresses (configurable)

### Can I add custom redaction patterns?
```yaml
processors:
  nrsecurity:
    custom_patterns:
      - name: "internal_id"
        pattern: 'employee_id:\s*\d+'
        replacement: 'employee_id: [REDACTED]'
```

### How do I enable compliance modes?
```yaml
processors:
  nrsecurity:
    compliance:
      pci_dss: true
      hipaa: true
      gdpr: true
```

## Operations

### How do I update NRDOT-HOST?
```bash
# Package manager
sudo apt-get update && sudo apt-get upgrade nrdot-host

# Or download new version and restart
sudo systemctl stop nrdot-host
# ... install new version ...
sudo systemctl start nrdot-host
```

### How do I backup configuration?
```bash
# Backup
sudo tar -czf nrdot-backup.tar.gz /etc/nrdot/

# Restore
sudo tar -xzf nrdot-backup.tar.gz -C /
```

### Can I reload config without restart?
Yes! 
```bash
nrdot-ctl config reload
```
Note: Some changes may still require restart.

### How do I monitor NRDOT-HOST itself?
NRDOT-HOST self-reports metrics:
- View locally: `nrdot-ctl metrics`
- In New Relic: Look for service "nrdot-telemetry"
- Prometheus endpoint: `http://localhost:9090/metrics`

## Integration

### Does NRDOT-HOST work with Kubernetes?
Yes! Full support including:
- DaemonSet deployment
- Pod/node metadata enrichment
- Service discovery
- Helm chart included

### Can I use NRDOT-HOST with Prometheus?
Yes! NRDOT-HOST can:
- Scrape Prometheus endpoints
- Export Prometheus metrics
- Use Prometheus service discovery

### Does it work with distributed tracing?
Yes! NRDOT-HOST supports:
- OTLP trace receiver
- Jaeger receiver
- Zipkin receiver
- W3C trace context propagation

### Can I send data to multiple backends?
```yaml
exporters:
  newrelic:
    license_key: "${NEW_RELIC_LICENSE_KEY}"
  prometheus:
    endpoint: "0.0.0.0:8889"
  otlp/backup:
    endpoint: "backup.example.com:4317"

service:
  pipelines:
    metrics:
      exporters: [newrelic, prometheus, otlp/backup]
```

## Common Errors

### "License key format invalid"
- Ensure key is 40 characters
- Check it starts with region code (US keys: no prefix, EU keys: 'eu01')
- No extra spaces or quotes

### "Connection refused" to API
- API only listens on localhost by default
- Use `nrdot-ctl` locally or configure remote access
- Check firewall rules

### "Cardinality limit exceeded"
- Too many unique metric series
- Check dimensions: `nrdot-ctl metrics cardinality --top 10`
- Increase limits or reduce dimensions

### "Memory limit exceeded"
- Collector using too much memory
- Enable memory_limiter processor
- Reduce batch sizes
- Check for cardinality issues

## Getting Help

### Where can I get help?
1. Check the [Troubleshooting Guide](./troubleshooting.md)
2. Search [GitHub Issues](https://github.com/deepaucksharma/nrdot-host/issues)
3. Ask in [Discussions](https://github.com/deepaucksharma/nrdot-host/discussions)
4. Review [Examples](../examples/)

### How do I report bugs?
1. Check if already reported in Issues
2. Create new issue with:
   - Version: `nrdot-ctl version`
   - Config (sanitized): `nrdot-ctl config show --sanitize`
   - Logs: Recent errors
   - Steps to reproduce

### Can I contribute?
Absolutely! See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines. We welcome:
- Bug fixes
- New features
- Documentation improvements
- Examples and tutorials

### Is commercial support available?
For commercial support options, contact New Relic sales or support teams.