# NRDOT Installation Guide

## Quick Installation

### Using the Installer Script

```bash
# Clone the repository
git clone https://github.com/NRDOT/nrdot-host.git
cd nrdot-host/systemd

# Run installer
sudo ./install.sh

# Complete setup
sudo ./scripts/post-install.sh

# Start services
sudo systemctl start nrdot.target
```

## Package Installation

### RPM-based Systems

```bash
# Build RPM package
rpmbuild -ba nrdot.spec

# Install package
sudo yum install rpmbuild/RPMS/x86_64/nrdot-1.0.0-1.el8.x86_64.rpm
```

### Debian-based Systems

```bash
# Build DEB package
dpkg-buildpackage -b -uc -us

# Install package
sudo apt install ../nrdot_1.0.0-1_amd64.deb
```

## Manual Installation

### 1. Install Dependencies

```bash
# RHEL/CentOS
sudo yum install -y golang ethtool iproute kernel-devel

# Ubuntu/Debian
sudo apt install -y golang ethtool iproute2 linux-headers-$(uname -r)
```

### 2. Build from Source

```bash
# Build binaries
make build

# Or use Go directly
go build -o bin/ ./cmd/...
```

### 3. Run Installer

```bash
sudo ./install.sh --build
```

## Docker Installation

```bash
# Build Docker image
docker build -t nrdot:latest .

# Run container
docker run -d \
  --name nrdot \
  --network host \
  --privileged \
  -v /sys/fs/bpf:/sys/fs/bpf \
  -v /etc/nrdot:/etc/nrdot \
  -v /var/lib/nrdot:/var/lib/nrdot \
  nrdot:latest
```

## Kubernetes Installation

```bash
# Install using Helm
helm repo add nrdot https://charts.nrdot.io
helm install nrdot nrdot/nrdot-host

# Or using kubectl
kubectl apply -f https://raw.githubusercontent.com/NRDOT/nrdot-host/main/deploy/kubernetes/
```

## Configuration

### Basic Configuration

1. **Edit environment variables**:
   ```bash
   sudo vi /etc/nrdot/nrdot.conf
   ```

2. **Configure services**:
   ```bash
   sudo vi /etc/nrdot/collector.yaml
   sudo vi /etc/nrdot/api-server.yaml
   ```

3. **Set up authentication**:
   ```bash
   # View generated API token
   sudo cat /etc/nrdot/auth-token
   ```

### Network Configuration

1. **Configure interfaces to monitor**:
   ```yaml
   # In collector.yaml
   interfaces:
     - name: eth0
       enabled: true
     - name: eth1
       enabled: true
   ```

2. **Set up flow export**:
   ```yaml
   # In collector.yaml
   export:
     netflow:
       enabled: true
       collectors:
         - address: 192.168.1.100:2055
   ```

## Verification

### Check Services

```bash
# Check status
sudo systemctl status nrdot.target

# View logs
sudo journalctl -u nrdot-collector -f

# Test API
curl -k https://localhost:8080/api/v1/status
```

### Run Diagnostics

```bash
# Health check
sudo /opt/nrdot/scripts/health-check.sh all

# System info
nrdot-collector --system-info
```

## Next Steps

1. **Access Web UI**: https://localhost:8080
2. **Configure monitoring**: Set up Prometheus/Grafana
3. **Set up alerts**: Configure alerting rules
4. **Review security**: Check firewall rules and certificates

## Troubleshooting

See [Troubleshooting Guide](README.md#troubleshooting) for common issues.

## Support

- Documentation: https://docs.nrdot.io
- GitHub: https://github.com/NRDOT/nrdot-host
- Community: https://nrdot.community