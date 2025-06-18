# NRDOT-HOST Quick Start Guide

Get up and running with NRDOT-HOST in 5 minutes!

## 🚀 One-Line Install

### Linux/macOS
```bash
curl -sSL https://raw.githubusercontent.com/deepaucksharma/nrdot-host/main/install.sh | sudo bash
```

### Docker
```bash
docker run -d \
  -e NRDOT_LICENSE_KEY=YOUR_LICENSE_KEY \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  ghcr.io/deepaucksharma/nrdot-host/nrdot-collector:latest
```

## 📝 Minimal Configuration

Create `/etc/nrdot/config.yaml`:

```yaml
# This is all you need!
service:
  name: my-service

license_key: YOUR_NEW_RELIC_LICENSE_KEY
```

## ▶️ Start Collecting

```bash
# Start the service
sudo systemctl start nrdot-host

# Check status
nrdot-ctl status

# View logs
nrdot-ctl logs --tail 20
```

## 📊 What's Being Collected?

By default, NRDOT-HOST collects:
- ✅ **System Metrics**: CPU, memory, disk, network
- ✅ **Process Metrics**: Top processes by CPU/memory
- ✅ **Container Metrics**: Docker/containerd (if available)
- ✅ **Kubernetes Metrics**: Pod, node, cluster (if in K8s)

All with:
- 🔒 **Automatic Secret Redaction**
- 🏷️ **Smart Metadata Enrichment**
- 📈 **Built-in Cardinality Protection**
- 🎯 **Zero Configuration Required**

## 🎯 Common Use Cases

### Monitor a Web Application
```yaml
service:
  name: web-app
  environment: production

license_key: YOUR_LICENSE_KEY

# That's it! Automatic discovery handles the rest
```

### Add Custom Metrics
```yaml
service:
  name: my-service

license_key: YOUR_LICENSE_KEY

# Scrape Prometheus metrics
receivers:
  prometheus:
    config:
      scrape_configs:
        - job_name: 'my-app'
          static_configs:
            - targets: ['localhost:9090']
```

### Enable Trace Collection
```yaml
service:
  name: my-service

license_key: YOUR_LICENSE_KEY

traces:
  enabled: true

# OTLP receiver is automatically configured on port 4317
```

## 🛠️ Useful Commands

```bash
# Configuration
nrdot-ctl config validate    # Validate configuration
nrdot-ctl config show        # Show running config
nrdot-ctl config reload      # Reload without restart

# Monitoring
nrdot-ctl health            # Health check
nrdot-ctl metrics           # View metrics
nrdot-ctl status           # Detailed status

# Troubleshooting
nrdot-ctl test connection   # Test New Relic connection
nrdot-ctl debug enable      # Enable debug logging
nrdot-ctl logs --tail 50    # View recent logs
```

## 🔍 Verify in New Relic

1. Log into [New Relic One](https://one.newrelic.com)
2. Navigate to **Infrastructure** or **APM & Services**
3. Look for your service name
4. Data should appear within 1-2 minutes

## 📚 Next Steps

- **[Configuration Guide](./docs/configuration.md)** - Customize your setup
- **[Processor Documentation](./docs/processors.md)** - Understand data processing
- **[Troubleshooting Guide](./docs/troubleshooting.md)** - Solve common issues
- **[Performance Tuning](./docs/performance.md)** - Optimize for scale

## 💡 Tips

1. **Start Simple**: The defaults work great for most use cases
2. **Monitor the Monitor**: Check `nrdot-ctl metrics` to see self-telemetry
3. **Use the CLI**: `nrdot-ctl` has built-in help for every command
4. **Enable Only What You Need**: Each feature has a small performance cost

## 🆘 Need Help?

- **Documentation**: [Full Docs](./docs/)
- **Issues**: [GitHub Issues](https://github.com/deepaucksharma/nrdot-host/issues)
- **Community**: [Discussions](https://github.com/deepaucksharma/nrdot-host/discussions)

---

**Ready to scale?** Check out our [Production Deployment Guide](./docs/deployment.md) for best practices on running NRDOT-HOST at scale.