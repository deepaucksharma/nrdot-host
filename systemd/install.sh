#!/bin/bash
# NRDOT Installation Script

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Installation directories
INSTALL_PREFIX="/opt/nrdot"
CONFIG_DIR="/etc/nrdot"
DATA_DIR="/var/lib/nrdot"
LOG_DIR="/var/log/nrdot"
SYSTEMD_DIR="/etc/systemd/system"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# Display banner
show_banner() {
    cat << 'EOF'
    _   _ ____  ____   ___ _____   _   _  ___  ____ _____ 
   | \ | |  _ \|  _ \ / _ \_   _| | | | |/ _ \/ ___|_   _|
   |  \| | |_) | | | | | | || |   | |_| | | | \___ \ | |  
   | |\  |  _ <| |_| | |_| || |   |  _  | |_| |___) || |  
   |_| \_|_| \_\____/ \___/ |_|   |_| |_|\___/|____/ |_|  
   
   Network Resource Discovery and Optimization Toolkit
   Installation Script v1.0.0
EOF
    echo
}

# Check prerequisites
check_prerequisites() {
    log_step "Checking prerequisites..."
    
    # Check if running as root
    if [[ $EUID -ne 0 ]]; then
        log_error "This installer must be run as root"
        exit 1
    fi
    
    # Run pre-installation checks
    if [[ -x "$SCRIPT_DIR/scripts/pre-install.sh" ]]; then
        "$SCRIPT_DIR/scripts/pre-install.sh"
    else
        log_error "Pre-installation script not found"
        exit 1
    fi
    
    # Check for pre-install completion flag
    if [[ ! -f /tmp/.nrdot-preinstall-complete ]]; then
        log_error "Pre-installation checks failed"
        exit 1
    fi
    
    rm -f /tmp/.nrdot-preinstall-complete
}

# Create user and group
create_user() {
    log_step "Creating NRDOT user and group..."
    
    # Create group
    if ! getent group nrdot &>/dev/null; then
        groupadd -r nrdot
        log_info "Created group: nrdot"
    else
        log_info "Group 'nrdot' already exists"
    fi
    
    # Create user
    if ! getent passwd nrdot &>/dev/null; then
        useradd -r -g nrdot -d "$DATA_DIR" -s /sbin/nologin -c "NRDOT Service Account" nrdot
        log_info "Created user: nrdot"
    else
        log_info "User 'nrdot' already exists"
    fi
}

# Create directory structure
create_directories() {
    log_step "Creating directory structure..."
    
    # Create directories with proper permissions
    directories=(
        "$INSTALL_PREFIX/bin"
        "$INSTALL_PREFIX/scripts"
        "$INSTALL_PREFIX/lib"
        "$INSTALL_PREFIX/share"
        "$CONFIG_DIR/certs"
        "$CONFIG_DIR/generated"
        "$DATA_DIR/collector"
        "$DATA_DIR/supervisor"
        "$DATA_DIR/config"
        "$DATA_DIR/api"
        "$DATA_DIR/privileged"
        "$DATA_DIR/db"
        "$LOG_DIR"
        "/var/cache/nrdot"
        "/run/nrdot"
    )
    
    for dir in "${directories[@]}"; do
        mkdir -p "$dir"
        log_info "Created: $dir"
    done
    
    # Set ownership and permissions
    chown -R nrdot:nrdot "$DATA_DIR" "$LOG_DIR" "/var/cache/nrdot"
    chown nrdot:nrdot "$CONFIG_DIR" "$CONFIG_DIR/generated"
    chown root:nrdot "$CONFIG_DIR/certs"
    chmod 750 "$CONFIG_DIR" "$DATA_DIR" "$LOG_DIR"
    chmod 770 "$CONFIG_DIR/generated"
    chmod 750 "$CONFIG_DIR/certs"
    chmod 700 "$DATA_DIR/privileged"
}

# Install binaries
install_binaries() {
    log_step "Installing binaries..."
    
    # Check if binaries exist in project
    if [[ -d "$PROJECT_ROOT/bin" ]]; then
        # Copy real binaries
        cp -a "$PROJECT_ROOT/bin/"* "$INSTALL_PREFIX/bin/" 2>/dev/null || true
        log_info "Installed binaries from $PROJECT_ROOT/bin"
    else
        # Create placeholder binaries for testing
        log_warn "No binaries found, creating placeholders..."
        
        for binary in nrdot-collector nrdot-supervisor nrdot-config-engine nrdot-api-server nrdot-privileged-helper; do
            cat > "$INSTALL_PREFIX/bin/$binary" << 'EOF'
#!/bin/bash
echo "Starting $0..."
# Placeholder binary - replace with actual implementation
while true; do
    sleep 60
done
EOF
            chmod 755 "$INSTALL_PREFIX/bin/$binary"
        done
    fi
    
    # Install helper scripts
    cp "$SCRIPT_DIR/scripts/health-check.sh" "$INSTALL_PREFIX/scripts/"
    chmod 755 "$INSTALL_PREFIX/scripts/health-check.sh"
    
    # Set ownership
    chown -R root:root "$INSTALL_PREFIX"
}

# Install configuration files
install_configs() {
    log_step "Installing configuration files..."
    
    # Install environment file
    cp "$SCRIPT_DIR/configs/nrdot.conf" "$CONFIG_DIR/"
    
    # Install sysctl settings
    cp "$SCRIPT_DIR/configs/sysctl.d/99-nrdot.conf" /etc/sysctl.d/
    sysctl -p /etc/sysctl.d/99-nrdot.conf &>/dev/null || true
    
    # Install limits
    cp "$SCRIPT_DIR/configs/limits.d/nrdot.conf" /etc/security/limits.d/
    
    # Create default config files if they don't exist
    for service in collector supervisor config-engine api-server privileged-helper database; do
        if [[ ! -f "$CONFIG_DIR/$service.yaml" ]]; then
            cat > "$CONFIG_DIR/$service.yaml" << EOF
# NRDOT $service Configuration
# Generated by installer

service:
  name: $service
  log_level: info
  
# Add service-specific configuration here
EOF
        fi
    done
    
    # Set permissions
    chown -R nrdot:nrdot "$CONFIG_DIR"
    chmod 640 "$CONFIG_DIR"/*.yaml
}

# Install systemd services
install_systemd() {
    log_step "Installing systemd services..."
    
    # Copy service files
    cp "$SCRIPT_DIR/services/"*.service "$SYSTEMD_DIR/"
    cp "$SCRIPT_DIR/services/"*.socket "$SYSTEMD_DIR/"
    cp "$SCRIPT_DIR/services/"*.target "$SYSTEMD_DIR/"
    
    # Reload systemd
    systemctl daemon-reload
    
    log_info "Systemd services installed"
}

# Configure system
configure_system() {
    log_step "Configuring system..."
    
    # Mount BPF filesystem if not mounted
    if [[ ! -d /sys/fs/bpf ]]; then
        mount -t bpf bpf /sys/fs/bpf 2>/dev/null || true
        
        # Make it permanent
        if ! grep -q "bpf /sys/fs/bpf" /etc/fstab; then
            echo "bpf /sys/fs/bpf bpf defaults 0 0" >> /etc/fstab
        fi
    fi
    
    # Enable IP forwarding
    sysctl -w net.ipv4.ip_forward=1 &>/dev/null || true
    sysctl -w net.ipv6.conf.all.forwarding=1 &>/dev/null || true
    
    # Load kernel modules
    modprobe nf_conntrack 2>/dev/null || true
    modprobe nf_conntrack_ipv4 2>/dev/null || true
    modprobe nf_conntrack_ipv6 2>/dev/null || true
}

# Build from source (optional)
build_from_source() {
    if [[ -f "$PROJECT_ROOT/Makefile" ]] || [[ -f "$PROJECT_ROOT/go.mod" ]]; then
        log_step "Building from source..."
        
        cd "$PROJECT_ROOT"
        
        # Go project
        if [[ -f "go.mod" ]]; then
            if command -v go &>/dev/null; then
                go build -o "$INSTALL_PREFIX/bin/" ./cmd/... 2>/dev/null || true
            else
                log_warn "Go not installed, skipping build"
            fi
        fi
        
        # Makefile project
        if [[ -f "Makefile" ]]; then
            if command -v make &>/dev/null; then
                make install PREFIX="$INSTALL_PREFIX" 2>/dev/null || true
            else
                log_warn "Make not installed, skipping build"
            fi
        fi
        
        cd - &>/dev/null
    fi
}

# Installation summary
show_summary() {
    log_step "Installation Summary"
    
    echo
    echo "NRDOT has been successfully installed!"
    echo
    echo "Installation directories:"
    echo "  Binaries: $INSTALL_PREFIX/bin"
    echo "  Config:   $CONFIG_DIR"
    echo "  Data:     $DATA_DIR"
    echo "  Logs:     $LOG_DIR"
    echo
    echo "Next steps:"
    echo "  1. Review and edit configuration files in $CONFIG_DIR"
    echo "  2. Run post-installation script: $SCRIPT_DIR/scripts/post-install.sh"
    echo "  3. Start services: systemctl start nrdot.target"
    echo
}

# Main installation function
main() {
    show_banner
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --prefix)
                INSTALL_PREFIX="$2"
                shift 2
                ;;
            --build)
                BUILD_FROM_SOURCE=true
                shift
                ;;
            --help|-h)
                echo "Usage: $0 [OPTIONS]"
                echo "Options:"
                echo "  --prefix PATH    Installation prefix (default: /opt/nrdot)"
                echo "  --build          Build from source if available"
                echo "  --help           Show this help message"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    # Run installation steps
    check_prerequisites
    create_user
    create_directories
    
    if [[ "${BUILD_FROM_SOURCE:-false}" == "true" ]]; then
        build_from_source
    fi
    
    install_binaries
    install_configs
    install_systemd
    configure_system
    show_summary
    
    log_info "Installation completed successfully!"
    log_info "Run $SCRIPT_DIR/scripts/post-install.sh to complete setup"
}

# Run main function
main "$@"