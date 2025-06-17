#!/bin/bash
# NRDOT-HOST Installation Script
# Usage: curl -sSL https://raw.githubusercontent.com/deepaucksharma/nrdot-host/main/install.sh | sudo bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
REPO="deepaucksharma/nrdot-host"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/nrdot"
SERVICE_USER="nrdot"
LATEST_RELEASE_URL="https://api.github.com/repos/$REPO/releases/latest"

# Functions
print_banner() {
    echo -e "${GREEN}"
    echo "  _   _ ____  ____   ___ _____   _   _  ___  ____ _____ "
    echo " | \ | |  _ \|  _ \ / _ \_   _| | | | |/ _ \/ ___|_   _|"
    echo " |  \| | |_) | | | | | | || |   | |_| | | | \___ \ | |  "
    echo " | |\  |  _ <| |_| | |_| || |   |  _  | |_| |___) || |  "
    echo " |_| \_|_| \_\____/ \___/ |_|   |_| |_|\___/|____/ |_|  "
    echo -e "${NC}"
    echo "Enterprise OpenTelemetry Distribution for Host Monitoring"
    echo "========================================================="
    echo
}

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

detect_os() {
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS=$ID
        VER=$VERSION_ID
    else
        log_error "Cannot detect OS"
        exit 1
    fi
}

detect_arch() {
    ARCH=$(uname -m)
    case $ARCH in
        x86_64)
            ARCH="amd64"
            ;;
        aarch64)
            ARCH="arm64"
            ;;
        *)
            log_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
}

check_dependencies() {
    local deps=("curl" "tar" "systemctl")
    for dep in "${deps[@]}"; do
        if ! command -v $dep &> /dev/null; then
            log_error "Missing dependency: $dep"
            exit 1
        fi
    done
}

create_user() {
    if ! id "$SERVICE_USER" &>/dev/null; then
        log_info "Creating service user: $SERVICE_USER"
        useradd -r -s /bin/false $SERVICE_USER
    fi
}

create_directories() {
    log_info "Creating directories..."
    mkdir -p $CONFIG_DIR
    mkdir -p /var/lib/nrdot
    mkdir -p /var/log/nrdot
    
    chown -R $SERVICE_USER:$SERVICE_USER /var/lib/nrdot
    chown -R $SERVICE_USER:$SERVICE_USER /var/log/nrdot
    chmod 750 $CONFIG_DIR
}

download_release() {
    log_info "Fetching latest release..."
    
    # Get latest release info
    RELEASE_INFO=$(curl -s $LATEST_RELEASE_URL)
    VERSION=$(echo $RELEASE_INFO | grep -oP '"tag_name": "\K[^"]+')
    
    if [[ -z "$VERSION" ]]; then
        log_error "Failed to fetch latest release version"
        exit 1
    fi
    
    log_info "Latest version: $VERSION"
    
    # Construct download URL
    BINARY_NAME="nrdot-host-${VERSION}-linux-${ARCH}.tar.gz"
    DOWNLOAD_URL="https://github.com/$REPO/releases/download/${VERSION}/${BINARY_NAME}"
    
    log_info "Downloading from: $DOWNLOAD_URL"
    
    # Download and extract
    TMP_DIR=$(mktemp -d)
    cd $TMP_DIR
    
    if ! curl -L -o $BINARY_NAME $DOWNLOAD_URL; then
        log_error "Failed to download release"
        exit 1
    fi
    
    log_info "Extracting binaries..."
    tar -xzf $BINARY_NAME
    
    # Install binaries
    log_info "Installing binaries to $INSTALL_DIR..."
    install -m 755 nrdot-ctl $INSTALL_DIR/
    install -m 755 nrdot-supervisor $INSTALL_DIR/
    install -m 755 nrdot-collector $INSTALL_DIR/
    install -m 4755 -o root -g $SERVICE_USER nrdot-privileged-helper $INSTALL_DIR/
    
    # Install systemd service
    if [[ -f nrdot-host.service ]]; then
        log_info "Installing systemd service..."
        install -m 644 nrdot-host.service /etc/systemd/system/
        systemctl daemon-reload
    fi
    
    # Cleanup
    cd /
    rm -rf $TMP_DIR
}

create_config() {
    if [[ ! -f $CONFIG_DIR/config.yaml ]]; then
        log_info "Creating default configuration..."
        cat > $CONFIG_DIR/config.yaml <<EOF
# NRDOT-HOST Configuration
# For full options see: https://github.com/$REPO/docs/configuration.md

service:
  name: "$(hostname)"
  environment: "production"

# Add your New Relic license key here
license_key: "YOUR_NEW_RELIC_LICENSE_KEY"

# Metrics collection enabled by default
metrics:
  enabled: true
  interval: "60s"

# Security processor enabled by default
processors:
  nrsecurity:
    redact_secrets: true
    redact_pii: true

# Cardinality protection enabled by default
processors:
  nrcap:
    limits:
      global: 100000
EOF
        chmod 600 $CONFIG_DIR/config.yaml
        log_warn "Please edit $CONFIG_DIR/config.yaml and add your New Relic license key"
    fi
}

setup_service() {
    log_info "Setting up systemd service..."
    systemctl enable nrdot-host
    
    echo
    log_info "Installation complete!"
    echo
    echo "Next steps:"
    echo "1. Edit configuration: sudo vi $CONFIG_DIR/config.yaml"
    echo "2. Add your New Relic license key"
    echo "3. Start the service: sudo systemctl start nrdot-host"
    echo "4. Check status: nrdot-ctl status"
    echo
    echo "For more information, visit: https://github.com/$REPO"
}

verify_installation() {
    log_info "Verifying installation..."
    
    if command -v nrdot-ctl &> /dev/null; then
        log_info "✓ nrdot-ctl installed successfully"
    else
        log_error "✗ nrdot-ctl not found in PATH"
        exit 1
    fi
    
    if [[ -f /etc/systemd/system/nrdot-host.service ]]; then
        log_info "✓ systemd service installed"
    else
        log_error "✗ systemd service not installed"
        exit 1
    fi
    
    if [[ -d $CONFIG_DIR ]]; then
        log_info "✓ Configuration directory created"
    else
        log_error "✗ Configuration directory missing"
        exit 1
    fi
}

main() {
    print_banner
    check_root
    detect_os
    detect_arch
    check_dependencies
    
    log_info "Installing NRDOT-HOST on $OS $VER ($ARCH)..."
    
    create_user
    create_directories
    download_release
    create_config
    verify_installation
    setup_service
}

# Run main function
main "$@"