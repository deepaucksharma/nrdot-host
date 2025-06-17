# NRDOT-HOST Installation Guide

This guide walks you through installing NRDOT-HOST on various platforms.

## Table of Contents

- [Quick Install](#quick-install)
- [System Requirements](#system-requirements)
- [Installation Methods](#installation-methods)
  - [Package Managers](#package-managers)
  - [Binary Installation](#binary-installation)
  - [Container Installation](#container-installation)
  - [Source Installation](#source-installation)
- [Post-Installation](#post-installation)
- [Upgrading](#upgrading)
- [Uninstalling](#uninstalling)

## Quick Install

### One-Line Install (Linux)

```bash
curl -sSL https://raw.githubusercontent.com/deepaucksharma/nrdot-host/main/install.sh | sudo bash
```

### Package Manager Install

```bash
# Ubuntu/Debian
sudo apt-get update && sudo apt-get install -y nrdot-host

# RHEL/CentOS/Fedora
sudo yum install -y nrdot-host

# macOS
brew install nrdot-host
```

## System Requirements

### Minimum Requirements

- **CPU**: 1 core (2+ recommended)
- **Memory**: 256MB (512MB+ recommended)
- **Disk**: 500MB for binaries, 1GB+ for data
- **Network**: Outbound HTTPS (443) to New Relic

### Supported Platforms

| Platform | Versions | Architecture |
|----------|----------|--------------|
| RHEL/CentOS | 7, 8, 9 | x86_64, arm64 |
| Ubuntu | 18.04, 20.04, 22.04 | x86_64, arm64 |
| Debian | 10, 11, 12 | x86_64, arm64 |
| Amazon Linux | 2, 2023 | x86_64, arm64 |
| SUSE Linux | 12, 15 | x86_64 |
| macOS | 11, 12, 13 | x86_64, arm64 |
| Windows Server | 2016, 2019, 2022 | x86_64 |

### Prerequisites

- systemd 219+ (Linux)
- .NET Framework 4.7.2+ (Windows)
- Administrative/root access for installation

## Installation Methods

### Package Managers

#### APT (Ubuntu/Debian)

```bash
# Add repository
curl -fsSL https://packages.newrelic.com/nrdot/apt/gpg.key | sudo apt-key add -
echo "deb https://packages.newrelic.com/nrdot/apt stable main" | sudo tee /etc/apt/sources.list.d/nrdot.list

# Update and install
sudo apt-get update
sudo apt-get install -y nrdot-host

# Configure license key
sudo nrdot-ctl config set license_key YOUR_LICENSE_KEY
```

#### YUM/DNF (RHEL/CentOS/Fedora)

```bash
# Add repository
cat <<EOF | sudo tee /etc/yum.repos.d/nrdot.repo
[nrdot]
name=NRDOT-HOST Repository
baseurl=https://packages.newrelic.com/nrdot/rpm/stable/\$basearch
enabled=1
gpgcheck=1
gpgkey=https://packages.newrelic.com/nrdot/rpm/gpg.key
EOF

# Install
sudo yum install -y nrdot-host

# Configure license key
sudo nrdot-ctl config set license_key YOUR_LICENSE_KEY
```

#### Homebrew (macOS)

```bash
# Add tap
brew tap newrelic/nrdot

# Install
brew install nrdot-host

# Configure
nrdot-ctl config set license_key YOUR_LICENSE_KEY

# Start service
brew services start nrdot-host
```

#### Chocolatey (Windows)

```powershell
# Install
choco install nrdot-host

# Configure
nrdot-ctl config set license_key YOUR_LICENSE_KEY

# Start service
Start-Service nrdot-host
```

### Binary Installation

#### Linux/macOS

```bash
# Set variables
VERSION="v1.0.0"
PLATFORM="linux"  # or "darwin" for macOS
ARCH="amd64"      # or "arm64"

# Download
wget https://github.com/deepaucksharma/nrdot-host/releases/download/${VERSION}/nrdot-${VERSION}-${PLATFORM}-${ARCH}.tar.gz

# Extract
sudo tar -xzf nrdot-${VERSION}-${PLATFORM}-${ARCH}.tar.gz -C /usr/local/bin/

# Create user and directories
sudo useradd -r -s /bin/false nrdot
sudo mkdir -p /etc/nrdot /var/lib/nrdot /var/log/nrdot
sudo chown -R nrdot:nrdot /var/lib/nrdot /var/log/nrdot

# Install service (Linux)
sudo cp /usr/local/bin/systemd/nrdot-host.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable nrdot-host

# Configure
sudo cp /usr/local/bin/examples/config.yaml /etc/nrdot/
sudo vi /etc/nrdot/config.yaml  # Add your license key

# Start
sudo systemctl start nrdot-host
```

#### Windows

```powershell
# Set variables
$version = "v1.0.0"
$url = "https://github.com/deepaucksharma/nrdot-host/releases/download/$version/nrdot-$version-windows-amd64.zip"

# Download
Invoke-WebRequest -Uri $url -OutFile nrdot.zip

# Extract
Expand-Archive -Path nrdot.zip -DestinationPath C:\nrdot

# Install service
C:\nrdot\nrdot-supervisor.exe install

# Configure
Copy-Item C:\nrdot\examples\config.yaml C:\nrdot\config.yaml
notepad C:\nrdot\config.yaml  # Add your license key

# Start service
Start-Service nrdot-host
```

### Container Installation

#### Docker

```bash
# Basic run
docker run -d \
  --name nrdot-host \
  -e NRDOT_LICENSE_KEY=YOUR_LICENSE_KEY \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  --restart unless-stopped \
  ghcr.io/deepaucksharma/nrdot-host/nrdot-collector:latest

# With custom config
docker run -d \
  --name nrdot-host \
  -v $(pwd)/config.yaml:/etc/nrdot/config.yaml:ro \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  --restart unless-stopped \
  ghcr.io/deepaucksharma/nrdot-host/nrdot-collector:latest
```

#### Docker Compose

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  nrdot:
    image: ghcr.io/deepaucksharma/nrdot-host/nrdot-collector:latest
    container_name: nrdot-host
    environment:
      - NRDOT_LICENSE_KEY=${NRDOT_LICENSE_KEY}
    volumes:
      - ./config.yaml:/etc/nrdot/config.yaml:ro
      - /var/run/docker.sock:/var/run/docker.sock:ro
    restart: unless-stopped
    network_mode: host
```

Run:

```bash
export NRDOT_LICENSE_KEY=YOUR_LICENSE_KEY
docker-compose up -d
```

#### Kubernetes

See [Kubernetes Deployment Guide](./deployment.md#kubernetes) for detailed instructions.

### Source Installation

#### Prerequisites

- Go 1.21+
- Make
- Git

#### Build from Source

```bash
# Clone repository
git clone https://github.com/deepaucksharma/nrdot-host.git
cd nrdot-host

# Build all components
make all

# Install binaries
sudo make install

# Configure
sudo cp examples/config.yaml /etc/nrdot/
sudo vi /etc/nrdot/config.yaml

# Install and start service
sudo make install-systemd
sudo systemctl start nrdot-host
```

## Post-Installation

### Initial Configuration

1. **Set License Key**
   ```bash
   sudo nrdot-ctl config set license_key YOUR_LICENSE_KEY
   ```

2. **Verify Installation**
   ```bash
   # Check version
   nrdot-ctl version
   
   # Test configuration
   nrdot-ctl config validate
   
   # Check service status
   sudo systemctl status nrdot-host  # Linux
   Get-Service nrdot-host            # Windows
   ```

3. **Test Connection**
   ```bash
   nrdot-ctl test connection
   ```

### Security Configuration

```bash
# Set secure permissions
sudo chmod 600 /etc/nrdot/config.yaml
sudo chown nrdot:nrdot /etc/nrdot/config.yaml

# Enable SELinux context (if applicable)
sudo semanage fcontext -a -t nrdot_conf_t '/etc/nrdot(/.*)?'
sudo restorecon -Rv /etc/nrdot
```

### Firewall Configuration

NRDOT only requires outbound HTTPS:

```bash
# No inbound rules needed by default
# API server binds to localhost only

# If using custom exporters, allow outbound to those endpoints
sudo firewall-cmd --permanent --add-port=443/tcp --zone=public
sudo firewall-cmd --reload
```

### Log Configuration

```bash
# Set log level
nrdot-ctl config set log_level info

# Configure log rotation
cat <<EOF | sudo tee /etc/logrotate.d/nrdot
/var/log/nrdot/*.log {
    daily
    rotate 7
    compress
    missingok
    notifempty
    create 0640 nrdot nrdot
    sharedscripts
    postrotate
        systemctl reload nrdot-host
    endscript
}
EOF
```

## Upgrading

### Package Manager Upgrade

```bash
# Ubuntu/Debian
sudo apt-get update && sudo apt-get upgrade nrdot-host

# RHEL/CentOS
sudo yum update nrdot-host

# macOS
brew upgrade nrdot-host
```

### Binary Upgrade

```bash
# Stop service
sudo systemctl stop nrdot-host

# Backup current installation
sudo cp -r /usr/local/bin/nrdot* /backup/

# Download and extract new version
wget https://github.com/deepaucksharma/nrdot-host/releases/latest/download/nrdot-linux-amd64.tar.gz
sudo tar -xzf nrdot-linux-amd64.tar.gz -C /usr/local/bin/

# Start service
sudo systemctl start nrdot-host
```

### Zero-Downtime Upgrade

For critical environments:

```bash
# Install new version alongside old
sudo tar -xzf nrdot-${NEW_VERSION}.tar.gz -C /usr/local/bin/nrdot-new/

# Test new version
/usr/local/bin/nrdot-new/nrdot-ctl config validate

# Swap binaries
sudo systemctl stop nrdot-host
sudo mv /usr/local/bin/nrdot* /usr/local/bin/nrdot-old/
sudo mv /usr/local/bin/nrdot-new/* /usr/local/bin/
sudo systemctl start nrdot-host

# Verify
nrdot-ctl health
```

## Uninstalling

### Package Manager Uninstall

```bash
# Ubuntu/Debian
sudo apt-get remove --purge nrdot-host

# RHEL/CentOS
sudo yum remove nrdot-host

# macOS
brew uninstall nrdot-host
```

### Manual Uninstall

```bash
# Stop and disable service
sudo systemctl stop nrdot-host
sudo systemctl disable nrdot-host

# Remove service files
sudo rm /etc/systemd/system/nrdot-host.service
sudo systemctl daemon-reload

# Remove binaries
sudo rm -rf /usr/local/bin/nrdot*

# Remove configuration and data (optional)
sudo rm -rf /etc/nrdot
sudo rm -rf /var/lib/nrdot
sudo rm -rf /var/log/nrdot

# Remove user
sudo userdel nrdot
```

### Container Uninstall

```bash
# Docker
docker stop nrdot-host
docker rm nrdot-host
docker rmi ghcr.io/deepaucksharma/nrdot-host/nrdot-collector:latest

# Kubernetes
kubectl delete namespace nrdot-system
```

## Troubleshooting Installation

### Common Issues

1. **Permission Denied**
   ```bash
   # Ensure running as root/sudo
   sudo nrdot-ctl config validate
   ```

2. **Service Won't Start**
   ```bash
   # Check logs
   sudo journalctl -u nrdot-host -n 50
   
   # Validate config
   sudo nrdot-ctl config validate
   ```

3. **Missing Dependencies**
   ```bash
   # Install required packages
   sudo apt-get install -y ca-certificates curl  # Debian/Ubuntu
   sudo yum install -y ca-certificates curl       # RHEL/CentOS
   ```

4. **License Key Invalid**
   ```bash
   # Verify key format
   nrdot-ctl config get license_key
   
   # Test connection
   nrdot-ctl test connection --verbose
   ```

### Getting Help

- Documentation: https://github.com/deepaucksharma/nrdot-host/docs
- Issues: https://github.com/deepaucksharma/nrdot-host/issues
- Community: https://github.com/deepaucksharma/nrdot-host/discussions

## Next Steps

- [Configuration Guide](./configuration.md) - Detailed configuration options
- [Deployment Guide](./deployment.md) - Production deployment strategies
- [Security Guide](./security.md) - Security best practices