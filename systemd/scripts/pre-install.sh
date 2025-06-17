#!/bin/bash
# NRDOT Pre-installation Script

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
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

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

# Detect OS and version
detect_os() {
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS=$ID
        VER=$VERSION_ID
    else
        log_error "Cannot detect OS version"
        exit 1
    fi
    
    log_info "Detected OS: $OS $VER"
}

# Check system requirements
check_requirements() {
    log_info "Checking system requirements..."
    
    # Check CPU cores
    CPU_CORES=$(nproc)
    if [[ $CPU_CORES -lt 2 ]]; then
        log_warn "System has only $CPU_CORES CPU cores. Recommended: 4+"
    else
        log_info "CPU cores: $CPU_CORES"
    fi
    
    # Check memory
    MEM_KB=$(grep MemTotal /proc/meminfo | awk '{print $2}')
    MEM_GB=$((MEM_KB / 1024 / 1024))
    if [[ $MEM_GB -lt 4 ]]; then
        log_warn "System has only ${MEM_GB}GB RAM. Recommended: 8GB+"
    else
        log_info "Memory: ${MEM_GB}GB"
    fi
    
    # Check disk space
    DISK_AVAILABLE=$(df /opt 2>/dev/null | tail -1 | awk '{print $4}')
    DISK_GB=$((DISK_AVAILABLE / 1024 / 1024))
    if [[ $DISK_GB -lt 10 ]]; then
        log_error "Insufficient disk space: ${DISK_GB}GB available. Required: 10GB+"
        exit 1
    else
        log_info "Disk space: ${DISK_GB}GB available"
    fi
}

# Check kernel version
check_kernel() {
    log_info "Checking kernel version..."
    
    KERNEL_VERSION=$(uname -r)
    KERNEL_MAJOR=$(echo "$KERNEL_VERSION" | cut -d. -f1)
    KERNEL_MINOR=$(echo "$KERNEL_VERSION" | cut -d. -f2)
    
    if [[ $KERNEL_MAJOR -lt 4 ]] || [[ $KERNEL_MAJOR -eq 4 && $KERNEL_MINOR -lt 14 ]]; then
        log_warn "Kernel version $KERNEL_VERSION is old. Recommended: 4.14+"
        log_warn "Some eBPF features may not be available"
    else
        log_info "Kernel version: $KERNEL_VERSION"
    fi
    
    # Check for eBPF support
    if [[ ! -d /sys/fs/bpf ]]; then
        log_warn "BPF filesystem not mounted. eBPF features may be limited"
    fi
}

# Check required commands
check_dependencies() {
    log_info "Checking dependencies..."
    
    REQUIRED_CMDS=(
        "systemctl"
        "ip"
        "ss"
        "tc"
        "ethtool"
    )
    
    MISSING_CMDS=()
    
    for cmd in "${REQUIRED_CMDS[@]}"; do
        if ! command -v "$cmd" &> /dev/null; then
            MISSING_CMDS+=("$cmd")
        fi
    done
    
    if [[ ${#MISSING_CMDS[@]} -gt 0 ]]; then
        log_error "Missing required commands: ${MISSING_CMDS[*]}"
        log_info "Please install missing dependencies"
        exit 1
    fi
    
    log_info "All required commands found"
}

# Check for conflicting services
check_conflicts() {
    log_info "Checking for conflicting services..."
    
    CONFLICTING_SERVICES=(
        "netdata"
        "collectd"
        "telegraf"
    )
    
    for service in "${CONFLICTING_SERVICES[@]}"; do
        if systemctl is-active --quiet "$service" 2>/dev/null; then
            log_warn "Found active service: $service"
            log_warn "This may conflict with NRDOT. Consider disabling it."
        fi
    done
}

# Create backup directory
create_backup_dir() {
    BACKUP_DIR="/var/backups/nrdot/pre-install-$(date +%Y%m%d-%H%M%S)"
    log_info "Creating backup directory: $BACKUP_DIR"
    mkdir -p "$BACKUP_DIR"
    
    # Backup existing configs if any
    if [[ -d /etc/nrdot ]]; then
        log_info "Backing up existing configuration..."
        cp -a /etc/nrdot "$BACKUP_DIR/"
    fi
    
    # Save system info
    {
        echo "System Information Backup"
        echo "========================"
        echo "Date: $(date)"
        echo "Hostname: $(hostname)"
        echo "OS: $OS $VER"
        echo "Kernel: $(uname -r)"
        echo "CPU: $CPU_CORES cores"
        echo "Memory: ${MEM_GB}GB"
        echo ""
        echo "Network Interfaces:"
        ip -brief link
        echo ""
        echo "Routing Table:"
        ip route
    } > "$BACKUP_DIR/system-info.txt"
}

# Main execution
main() {
    log_info "Starting NRDOT pre-installation checks..."
    
    check_root
    detect_os
    check_requirements
    check_kernel
    check_dependencies
    check_conflicts
    create_backup_dir
    
    log_info "Pre-installation checks completed successfully"
    log_info "System is ready for NRDOT installation"
    
    # Create flag file for installer
    touch /tmp/.nrdot-preinstall-complete
}

# Run main function
main "$@"