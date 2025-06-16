# NRDOT-Host: New Relic Distribution of OpenTelemetry for Host Monitoring

Enterprise-grade, secure, and manageable OpenTelemetry distribution for infrastructure monitoring.

## 🎯 Overview

NRDOT-Host solves the enterprise OpenTelemetry paradox by providing a hardened, opinionated distribution that combines OTel's flexibility with New Relic's operational excellence. This monorepo contains 30 modular components that together create a complete observability solution.

## 📦 Repository Structure

### Core Control Plane (5 repos)
- **[nrdot-ctl](./nrdot-ctl)** - Main control binary and CLI
- **[nrdot-config-engine](./nrdot-config-engine)** - Configuration management
- **[nrdot-supervisor](./nrdot-supervisor)** - Process lifecycle management
- **[nrdot-telemetry-client](./nrdot-telemetry-client)** - Self-instrumentation
- **[nrdot-template-lib](./nrdot-template-lib)** - OTel config templates

### OTel Processors (6 repos)
- **[otel-processor-nrsecurity](./otel-processor-nrsecurity)** - Secret redaction & security
- **[otel-processor-nrenrich](./otel-processor-nrenrich)** - Metadata enrichment
- **[otel-processor-nrtransform](./otel-processor-nrtransform)** - Metric calculations
- **[otel-processor-nrcap](./otel-processor-nrcap)** - Cardinality protection
- **[nrdot-privileged-helper](./nrdot-privileged-helper)** - Non-root process collection
- **[otel-processor-common](./otel-processor-common)** - Shared utilities

### Configuration & Management (4 repos)
- **[nrdot-schema](./nrdot-schema)** - Configuration schemas
- **[nrdot-remote-config](./nrdot-remote-config)** - Feature flags & A/B testing
- **[nrdot-api-server](./nrdot-api-server)** - Local REST API
- **[nrdot-fleet-protocol](./nrdot-fleet-protocol)** - Fleet coordination
### Testing & Validation (5 repos)
- **[nrdot-test-harness](./nrdot-test-harness)** - Integration testing framework
- **[guardian-fleet-infra](./guardian-fleet-infra)** - 24/7 validation platform
- **[nrdot-workload-simulators](./nrdot-workload-simulators)** - Load generation
- **[nrdot-compliance-validator](./nrdot-compliance-validator)** - Security compliance
- **[nrdot-benchmark-suite](./nrdot-benchmark-suite)** - Performance comparison

### Deployment & Packaging (5 repos)
- **[nrdot-packaging](./nrdot-packaging)** - RPM/DEB/MSI packages
- **[nrdot-container-images](./nrdot-container-images)** - Docker images
- **[nrdot-k8s-operator](./nrdot-k8s-operator)** - Kubernetes operator
- **[nrdot-ansible-role](./nrdot-ansible-role)** - Ansible automation
- **[nrdot-helm-chart](./nrdot-helm-chart)** - Helm deployment

### Utilities & Tools (5 repos)
- **[nrdot-migrate](./nrdot-migrate)** - Migration utilities
- **[nrdot-debug-tools](./nrdot-debug-tools)** - Diagnostic tools
- **[nrdot-sdk-go](./nrdot-sdk-go)** - Extension SDK
- **[nrdot-health-analyzer](./nrdot-health-analyzer)** - KPI analysis
- **[nrdot-cost-calculator](./nrdot-cost-calculator)** - Cost optimization

## 🏗️ Architecture

```
User Config (Simple YAML)
    ↓
[nrdot-ctl] → [config-engine] → [template-lib]
    ↓                               ↓
[supervisor] ← [telemetry-client]   Generated OTel Config
    ↓
[OTel Collector with Custom Processors]
    ↓
New Relic Platform
```## 🔄 Integration Flow

1. **Configuration**: User provides simple `nrdot-host.yml` → `config-engine` merges with defaults → `template-lib` generates OTel config
2. **Execution**: `nrdot-ctl` starts → `supervisor` manages OTel Collector → Custom processors enhance data
3. **Security**: `nrsecurity` redacts secrets → `nrenrich` adds metadata → `nrtransform` calculates metrics
4. **Monitoring**: `telemetry-client` self-reports → `health-analyzer` calculates KPIs → Dashboards update

## 🚀 Quick Start

```bash
# Install NRDOT-Host
sudo rpm -i nrdot-host-1.0.0.x86_64.rpm

# Configure
cat > /etc/nrdot/nrdot-host.yml <<EOF
license_key: YOUR_NEW_RELIC_LICENSE_KEY
process_monitoring:
  enabled: true
EOF

# Start
sudo systemctl start nrdot-host

# Check status
nrdot-ctl status
```

## 🛠️ Development

### Prerequisites
- Go 1.21+
- Docker 20.10+
- Make 4.0+

### Build All Components
```bash
make all
```

### Run Tests
```bash
make test           # Unit tests
make test-integration # Integration tests
make test-security   # Security validation
```## 🎯 Key Features

### Security First
- ✅ Automatic secret redaction
- ✅ Non-root process monitoring (Phase 2)
- ✅ Secure-by-default configuration
- ✅ Compliance validation (PCI-DSS, HIPAA, SOC2)

### Operational Excellence
- ✅ Zero-config deployment
- ✅ Self-healing supervision
- ✅ Automatic cardinality protection
- ✅ A/B testing & gradual rollouts

### Enterprise Ready
- ✅ Multi-platform support (Linux, Windows, containers)
- ✅ Fleet management at scale
- ✅ Cost optimization built-in
- ✅ 24/7 Guardian Fleet validation

## 📊 Project Goals

1. **Phase 1** (Months 1-6): Core security & usability
2. **Phase 2** (Months 7-12): Intelligence & optimization
3. **Phase 3** (Months 13-18): Scale & automation

Target: 10,000+ production hosts in Year 1

## 🤝 Contributing

Each repository is independently versioned and can be developed in parallel:

1. Pick a repository from the structure above
2. Follow its README for setup
3. Make changes with tests
4. Submit PR to that specific repo

## 📄 License

Dual-licensed:
- Open source components: Apache 2.0
- Enterprise features: Commercial

## 🔗 Links

- [Documentation](https://docs.newrelic.com/nrdot)
- [Guardian Fleet Dashboard](https://one.newrelic.com/nrdot-fleet)
- [Community Forum](https://discuss.newrelic.com/c/nrdot)

---
Built with ❤️ by New Relic