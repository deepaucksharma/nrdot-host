# NRDOT-HOST Deployment Guide

This guide covers deployment options for NRDOT-HOST across different environments.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Deployment Options](#deployment-options)
- [Linux (systemd)](#linux-systemd)
- [Docker](#docker)
- [Kubernetes](#kubernetes)
- [Configuration Management](#configuration-management)
- [High Availability](#high-availability)
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### System Requirements

- **CPU**: 2+ cores recommended
- **Memory**: 512MB minimum, 2GB recommended
- **Disk**: 1GB for installation, 10GB+ for data
- **OS**: Linux (RHEL 7+, Ubuntu 18.04+), Windows Server 2016+
- **Network**: Outbound HTTPS to New Relic endpoints

### Software Dependencies

- systemd 219+ (for Linux deployments)
- Docker 20.10+ (for container deployments)
- Kubernetes 1.21+ (for K8s deployments)

## Deployment Options

| Method | Best For | Pros | Cons |
|--------|----------|------|------|
| systemd | Production Linux servers | Native integration, resource efficient | Linux only |
| Docker | Containerized environments | Portable, easy updates | Container overhead |
| Kubernetes | Cloud-native at scale | Auto-scaling, orchestration | Complexity |
| Binary | Testing/development | Simple, no dependencies | Manual management |

## Linux (systemd)

### RPM Installation (RHEL/CentOS)

```bash
# Download latest release
wget https://github.com/deepaucksharma/nrdot-host/releases/latest/download/nrdot-host.rpm

# Install package
sudo rpm -i nrdot-host.rpm

# Configure
sudo vi /etc/nrdot/config.yaml

# Enable and start
sudo systemctl enable nrdot-host
sudo systemctl start nrdot-host
```

### DEB Installation (Ubuntu/Debian)

```bash
# Download latest release
wget https://github.com/deepaucksharma/nrdot-host/releases/latest/download/nrdot-host.deb

# Install package
sudo dpkg -i nrdot-host.deb

# Configure
sudo vi /etc/nrdot/config.yaml

# Enable and start
sudo systemctl enable nrdot-host
sudo systemctl start nrdot-host
```

### Manual Installation

```bash
# Create user
sudo useradd -r -s /bin/false nrdot

# Create directories
sudo mkdir -p /etc/nrdot /var/lib/nrdot /var/log/nrdot
sudo chown -R nrdot:nrdot /var/lib/nrdot /var/log/nrdot

# Download binaries
wget https://github.com/deepaucksharma/nrdot-host/releases/latest/download/nrdot-linux-amd64.tar.gz
sudo tar -xzf nrdot-linux-amd64.tar.gz -C /usr/local/bin/

# Install systemd service
sudo cp /usr/local/bin/nrdot-supervisor.service /etc/systemd/system/
sudo systemctl daemon-reload

# Configure
sudo cp /usr/local/bin/example-config.yaml /etc/nrdot/config.yaml
sudo vi /etc/nrdot/config.yaml

# Start service
sudo systemctl enable nrdot-host
sudo systemctl start nrdot-host
```

### Systemd Configuration

The systemd service includes:

```ini
[Unit]
Description=NRDOT-HOST OpenTelemetry Distribution
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=nrdot
Group=nrdot
ExecStart=/usr/bin/nrdot-supervisor
Restart=always
RestartSec=10

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/nrdot /var/log/nrdot

[Install]
WantedBy=multi-user.target
```

## Docker

### Quick Start

```bash
# Run with config file
docker run -d \
  --name nrdot-host \
  -v /etc/nrdot:/etc/nrdot:ro \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  --restart unless-stopped \
  ghcr.io/deepaucksharma/nrdot-host/nrdot-collector:latest
```

### Docker Compose

```yaml
version: '3.8'

services:
  nrdot:
    image: ghcr.io/deepaucksharma/nrdot-host/nrdot-collector:latest
    container_name: nrdot-host
    restart: unless-stopped
    volumes:
      - ./config.yaml:/etc/nrdot/config.yaml:ro
      - /var/run/docker.sock:/var/run/docker.sock:ro
    environment:
      - NRDOT_LICENSE_KEY=${NEW_RELIC_LICENSE_KEY}
    network_mode: host
    cap_drop:
      - ALL
    cap_add:
      - CAP_NET_RAW  # For network monitoring
    security_opt:
      - no-new-privileges:true
    read_only: true
    tmpfs:
      - /tmp
```

### Custom Image

```dockerfile
FROM ghcr.io/deepaucksharma/nrdot-host/nrdot-collector:latest

# Add custom configuration
COPY config.yaml /etc/nrdot/config.yaml

# Add certificates if needed
COPY certs/ /etc/nrdot/certs/

# Set environment
ENV NRDOT_ENV=production
```

## Kubernetes

### Helm Installation

```bash
# Add repository
helm repo add nrdot https://deepaucksharma.github.io/nrdot-host
helm repo update

# Install with custom values
helm install nrdot nrdot/nrdot-host \
  --namespace nrdot-system \
  --create-namespace \
  --set config.licenseKey=$NEW_RELIC_LICENSE_KEY \
  --set config.clusterName=my-cluster
```

### Manual Installation

```bash
# Create namespace
kubectl create namespace nrdot-system

# Create secret for license key
kubectl create secret generic nrdot-license \
  --from-literal=license-key=$NEW_RELIC_LICENSE_KEY \
  -n nrdot-system

# Apply manifests
kubectl apply -f https://raw.githubusercontent.com/deepaucksharma/nrdot-host/main/kubernetes/manifests/
```

### DaemonSet Configuration

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: nrdot-host
  namespace: nrdot-system
spec:
  selector:
    matchLabels:
      app: nrdot-host
  template:
    metadata:
      labels:
        app: nrdot-host
    spec:
      serviceAccountName: nrdot-host
      containers:
      - name: nrdot
        image: ghcr.io/deepaucksharma/nrdot-host/nrdot-collector:latest
        resources:
          limits:
            memory: 512Mi
            cpu: 500m
          requests:
            memory: 256Mi
            cpu: 200m
        volumeMounts:
        - name: config
          mountPath: /etc/nrdot
        - name: varlog
          mountPath: /var/log
          readOnly: true
        - name: varlibdockercontainers
          mountPath: /var/lib/docker/containers
          readOnly: true
        env:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: NRDOT_LICENSE_KEY
          valueFrom:
            secretKeyRef:
              name: nrdot-license
              key: license-key
      volumes:
      - name: config
        configMap:
          name: nrdot-config
      - name: varlog
        hostPath:
          path: /var/log
      - name: varlibdockercontainers
        hostPath:
          path: /var/lib/docker/containers
```

### Deployment (Control Plane)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nrdot-control-plane
  namespace: nrdot-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nrdot-control-plane
  template:
    metadata:
      labels:
        app: nrdot-control-plane
    spec:
      serviceAccountName: nrdot-control-plane
      containers:
      - name: api-server
        image: ghcr.io/deepaucksharma/nrdot-host/nrdot-api-server:latest
        ports:
        - containerPort: 8080
          name: api
        - containerPort: 9090
          name: metrics
      - name: config-engine
        image: ghcr.io/deepaucksharma/nrdot-host/nrdot-config-engine:latest
        volumeMounts:
        - name: config-cache
          mountPath: /var/cache/nrdot
      volumes:
      - name: config-cache
        emptyDir: {}
```

## Configuration Management

### Environment Variables

All configuration options can be set via environment variables:

```bash
export NRDOT_LICENSE_KEY="your-license-key"
export NRDOT_API_ENDPOINT="https://otlp.nr-data.net"
export NRDOT_LOG_LEVEL="info"
export NRDOT_METRICS_INTERVAL="60s"
```

### Configuration Hierarchy

1. Default values (built-in)
2. Configuration file (`/etc/nrdot/config.yaml`)
3. Environment variables
4. Command-line flags

### Dynamic Configuration

For dynamic configuration updates without restart:

```bash
# Update configuration
sudo vi /etc/nrdot/config.yaml

# Reload configuration
sudo nrdot-ctl config reload

# Verify new configuration
sudo nrdot-ctl config validate
```

### Configuration Templates

Use templates for environment-specific configs:

```yaml
# config-template.yaml
service:
  name: ${SERVICE_NAME}
  environment: ${ENVIRONMENT}

license_key: ${NRDOT_LICENSE_KEY}

processors:
  cardinality:
    enabled: true
    limits:
      global: ${CARDINALITY_LIMIT:-100000}
```

Process template:

```bash
envsubst < config-template.yaml > config.yaml
```

## High Availability

### Active-Passive Setup

```yaml
# Primary node
nrdot:
  role: primary
  cluster:
    enabled: true
    bind_address: 0.0.0.0:7777
    peers:
      - secondary.example.com:7777

# Secondary node
nrdot:
  role: secondary
  cluster:
    enabled: true
    bind_address: 0.0.0.0:7777
    peers:
      - primary.example.com:7777
```

### Load Balancing

For scaled deployments, use a load balancer:

```nginx
upstream nrdot_api {
    least_conn;
    server nrdot-1.internal:8080;
    server nrdot-2.internal:8080;
    server nrdot-3.internal:8080;
}

server {
    listen 80;
    location /v1/ {
        proxy_pass http://nrdot_api;
        proxy_set_header Host $host;
    }
}
```

### Disaster Recovery

1. **Configuration Backup**
   ```bash
   # Backup configuration
   tar -czf nrdot-config-backup.tar.gz /etc/nrdot/
   
   # Store in version control
   git add nrdot-config-backup.tar.gz
   git commit -m "Backup NRDOT configuration"
   ```

2. **State Backup**
   ```bash
   # Backup state
   nrdot-ctl backup create --output=/backup/nrdot-state.tar.gz
   
   # Restore state
   nrdot-ctl backup restore --input=/backup/nrdot-state.tar.gz
   ```

## Monitoring

### Health Checks

```bash
# CLI health check
nrdot-ctl health

# HTTP health endpoint
curl -f http://localhost:8080/v1/health

# Systemd status
systemctl status nrdot-host
```

### Metrics

NRDOT exposes Prometheus metrics on port 9090:

```bash
# View metrics
curl http://localhost:9090/metrics

# Key metrics to monitor
nrdot_collector_uptime_seconds
nrdot_collector_pipeline_metrics_processed_total
nrdot_collector_pipeline_errors_total
nrdot_processor_cardinality_dropped_total
```

### Logging

```bash
# View logs
journalctl -u nrdot-host -f

# Change log level
nrdot-ctl log-level set debug

# Log locations
/var/log/nrdot/collector.log
/var/log/nrdot/supervisor.log
/var/log/nrdot/api-server.log
```

## Troubleshooting

### Common Issues

1. **Service Won't Start**
   ```bash
   # Check logs
   journalctl -u nrdot-host -n 50
   
   # Validate configuration
   nrdot-ctl config validate
   
   # Check permissions
   ls -la /etc/nrdot/
   ls -la /var/lib/nrdot/
   ```

2. **No Data in New Relic**
   ```bash
   # Check connectivity
   nrdot-ctl test connection
   
   # Verify license key
   nrdot-ctl config get license_key
   
   # Check pipeline status
   nrdot-ctl status pipelines
   ```

3. **High Memory Usage**
   ```bash
   # Check cardinality
   nrdot-ctl metrics cardinality
   
   # Adjust limits
   nrdot-ctl config set processors.cardinality.limits.global=50000
   ```

### Debug Mode

Enable debug mode for detailed troubleshooting:

```bash
# Temporary debug mode
nrdot-ctl debug enable --duration=5m

# Persistent debug mode
echo "debug: true" >> /etc/nrdot/config.yaml
systemctl restart nrdot-host
```

### Support

For additional help:

- Documentation: https://github.com/deepaucksharma/nrdot-host/docs
- Issues: https://github.com/deepaucksharma/nrdot-host/issues
- Community: https://github.com/deepaucksharma/nrdot-host/discussions