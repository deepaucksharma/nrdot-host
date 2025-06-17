# How to Run NRDOT-HOST

A comprehensive guide to running the NRDOT-HOST enterprise OpenTelemetry distribution.

## Prerequisites

1. **Go 1.21+** - Required for building from source
2. **Docker** - For containerized deployment
3. **New Relic License Key** - For exporting telemetry data
4. **Root or sudo access** - For system monitoring features

## Quick Start (Development)

### 1. Initial Setup
```bash
# Clone the repository
git clone https://github.com/deepaucksharma/nrdot-host.git
cd nrdot-host

# Run the quickstart script
./quickstart.sh

# Install dependencies
make setup
```

### 2. Configure Your License Key
Edit the example configuration:
```bash
# Edit the generated config
vim examples/local-config.yml

# Replace YOUR_NEW_RELIC_LICENSE_KEY_HERE with your actual license key
```

### 3. Build All Components
```bash
# Build everything
make all

# Or build specific components
make build-core       # Core components only
make build-processors # OTel processors only
make build-tools      # Tools only
```

### 4. Run Locally
```bash
# Start NRDOT with the example config
make run-local

# This runs: ./nrdot-ctl/bin/nrdot-ctl start --config=./examples/local-config.yml
```

## Production Deployment

### Option 1: Systemd Service (Linux)

1. **Build and Package**
```bash
# Create RPM/DEB packages
make package

# Install the package (example for DEB)
sudo dpkg -i nrdot-packaging/dist/nrdot-host_1.0.0_amd64.deb
```

2. **Configure**
```bash
# Edit the main configuration
sudo vim /etc/nrdot/config.yaml

# Set your license key and customize settings
```

3. **Start the Service**
```bash
# Enable and start the supervisor service
sudo systemctl enable nrdot-supervisor
sudo systemctl start nrdot-supervisor

# Check status
sudo systemctl status nrdot-supervisor
```

### Option 2: Docker Compose

1. **Build Docker Images**
```bash
# Build all container images
make docker
```

2. **Run with Docker Compose**
```bash
cd otelcol-builder/docker

# Edit the config if needed
vim ../test-config.yaml

# Start all services
docker-compose up -d

# View logs
docker-compose logs -f nrdot-collector
```

### Option 3: Kubernetes

1. **Deploy with Helm**
```bash
# Add the NRDOT Helm repository
helm repo add nrdot ./nrdot-helm-chart

# Install with custom values
helm install nrdot nrdot/nrdot-host \
  --set licenseKey=YOUR_LICENSE_KEY \
  --set environment=production
```

2. **Or use the Operator**
```bash
# Deploy the operator
kubectl apply -f nrdot-k8s-operator/deploy/

# Create a custom resource
kubectl apply -f nrdot-k8s-operator/examples/nrdot-instance.yaml
```

## Component Architecture

The system runs in this order:

1. **nrdot-ctl** (CLI) → User interface for management
2. **nrdot-config-engine** → Processes user YAML into OTel config
3. **nrdot-supervisor** → Manages OTel Collector lifecycle
4. **OTel Collector** → Runs with NRDOT processors:
   - **nrsecurity** → Secret redaction
   - **nrenrich** → Metadata enrichment
   - **nrtransform** → Metric calculations
   - **nrcap** → Cardinality protection
5. **nrdot-telemetry-client** → Self-monitoring

## Configuration Flow

```
User YAML Config → Config Engine → Template Library → Full OTel Config → Supervisor → Collector
```

## Essential Commands

### Service Management
```bash
# Start NRDOT
nrdot-ctl start --config=/etc/nrdot/config.yaml

# Stop NRDOT
nrdot-ctl stop

# Restart NRDOT
nrdot-ctl restart

# Check status
nrdot-ctl status
```

### Configuration Management
```bash
# Validate configuration
nrdot-ctl config validate --file=config.yaml

# Generate OTel config from NRDOT config
nrdot-ctl generate --input=config.yaml --output=otel-config.yaml

# Apply configuration changes
nrdot-ctl config apply --file=new-config.yaml
```

### Monitoring & Debugging
```bash
# View collector status
nrdot-ctl collector status

# Check health
curl http://localhost:13133/health

# View metrics
curl http://localhost:8888/metrics

# Debug mode
nrdot-ctl start --debug --config=config.yaml
```

## Testing

### Unit Tests
```bash
make test
```

### Integration Tests
```bash
make test-integration
```

### Run with Test Workloads
```bash
# Start workload simulators
cd nrdot-workload-simulators
./run-workloads.sh

# Monitor with Guardian Fleet
cd guardian-fleet-infra
make deploy-local
```

## Environment Variables

Key environment variables:
```bash
export NRDOT_CONFIG_PATH=/etc/nrdot/config.yaml
export NRDOT_LOG_LEVEL=info
export NRDOT_TELEMETRY_ENDPOINT=http://localhost:4318
export NEW_RELIC_LICENSE_KEY=your-license-key
```

## Troubleshooting

### Check Logs
```bash
# Supervisor logs
journalctl -u nrdot-supervisor -f

# Collector logs
docker logs nrdot-collector

# Debug with verbose logging
nrdot-ctl start --config=config.yaml --log-level=debug
```

### Common Issues

1. **Permission Denied**
   - Run with sudo or configure non-root mode with privileged-helper

2. **Port Already in Use**
   - Check ports 4317, 4318, 8888, 13133
   - `lsof -i :4317`

3. **High Memory Usage**
   - Adjust memory limits in config
   - Enable memory_limiter processor

4. **No Metrics Received**
   - Verify license key is correct
   - Check network connectivity
   - Validate exporters configuration

### Health Checks

```bash
# API Server health
curl http://localhost:8080/health

# Collector health  
curl http://localhost:13133/health

# Prometheus metrics
curl http://localhost:8888/metrics
```

## Advanced Usage

### Custom Processors
```go
// Use the SDK to create custom processors
import "github.com/newrelic/nrdot-host/nrdot-sdk-go"

// See nrdot-sdk-go/examples/
```

### Fleet Management
```bash
# Deploy fleet protocol for managing multiple instances
nrdot-fleet-protocol deploy --config=fleet-config.yaml
```

### Migration from Other Solutions
```bash
# Migrate from New Relic Infrastructure agent
nrdot-migrate from-infra --config=/etc/newrelic-infra.yml

# Migrate from vanilla OTel
nrdot-migrate from-otel --config=otel-config.yaml
```

## Performance Tuning

1. **Batch Processing**
   - Adjust batch size and timeout in config
   - Default: 1000 items, 10s timeout

2. **Memory Limits**
   - Set appropriate memory_limit_mib
   - Monitor with telemetry-client

3. **Cardinality Control**
   - Configure nrcap processor limits
   - Use cost-calculator to optimize

## Security Considerations

1. **Non-Root Operation**
   - Uses privileged-helper for process monitoring
   - Configure appropriate Linux capabilities

2. **Secret Redaction**
   - nrsecurity processor automatically redacts secrets
   - Configure additional patterns as needed

3. **TLS Configuration**
   - Enable TLS for all endpoints in production
   - Use proper certificate management

## Getting Help

- Check component logs first
- Use `nrdot-debug-tools` for diagnostics
- Review `/var/log/nrdot/` for detailed logs
- Enable debug mode for troubleshooting

For more details, see individual component READMEs in their respective directories.