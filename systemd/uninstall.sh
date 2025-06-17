#!/bin/bash
# NRDOT Uninstallation Script

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

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
   Uninstallation Script v1.0.0
EOF
    echo
}

# Check prerequisites
check_prerequisites() {
    log_step "Checking prerequisites..."
    
    # Check if running as root
    if [[ $EUID -ne 0 ]]; then
        log_error "This uninstaller must be run as root"
        exit 1
    fi
    
    # Check if NRDOT is installed
    if [[ ! -d "$INSTALL_PREFIX" ]] && [[ ! -d "$CONFIG_DIR" ]]; then
        log_error "NRDOT does not appear to be installed"
        exit 1
    fi
    
    # Run pre-uninstall script
    if [[ -x "$SCRIPT_DIR/scripts/pre-uninstall.sh" ]]; then
        "$SCRIPT_DIR/scripts/pre-uninstall.sh"
    else
        log_warn "Pre-uninstall script not found, continuing..."
    fi
    
    # Check for pre-uninstall completion flag
    if [[ ! -f /tmp/.nrdot-preuninstall-complete ]]; then
        log_warn "Pre-uninstall checks may have failed"
    else
        rm -f /tmp/.nrdot-preuninstall-complete
    fi
}

# Remove systemd services
remove_systemd() {
    log_step "Removing systemd services..."
    
    # List of service files
    SERVICES=(
        "nrdot.target"
        "nrdot-api-server.service"
        "nrdot-config-engine.service"
        "nrdot-collector.service"
        "nrdot-supervisor.service"
        "nrdot-privileged-helper.service"
        "nrdot-api.socket"
        "nrdot-privileged.socket"
    )
    
    # Remove service files
    for service in "${SERVICES[@]}"; do
        if [[ -f "$SYSTEMD_DIR/$service" ]]; then
            rm -f "$SYSTEMD_DIR/$service"
            log_info "Removed: $service"
        fi
    done
    
    # Reload systemd
    systemctl daemon-reload
    systemctl reset-failed
}

# Remove configuration files
remove_configs() {
    log_step "Removing configuration files..."
    
    # Ask about config removal
    read -p "Remove configuration files? (y/N): " -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        # Remove main config directory
        if [[ -d "$CONFIG_DIR" ]]; then
            rm -rf "$CONFIG_DIR"
            log_info "Removed: $CONFIG_DIR"
        fi
        
        # Remove sysctl config
        rm -f /etc/sysctl.d/99-nrdot.conf
        
        # Remove limits config
        rm -f /etc/security/limits.d/nrdot.conf
        
        # Remove logrotate config
        rm -f /etc/logrotate.d/nrdot
    else
        log_info "Configuration files preserved"
    fi
}

# Remove data files
remove_data() {
    log_step "Removing data files..."
    
    # Ask about data removal
    read -p "Remove all NRDOT data? (y/N): " -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        # Remove data directory
        if [[ -d "$DATA_DIR" ]]; then
            rm -rf "$DATA_DIR"
            log_info "Removed: $DATA_DIR"
        fi
        
        # Remove cache
        rm -rf /var/cache/nrdot
        
        # Remove runtime files
        rm -rf /run/nrdot
    else
        log_info "Data files preserved"
    fi
}

# Remove log files
remove_logs() {
    log_step "Removing log files..."
    
    # Ask about log removal
    read -p "Remove all NRDOT logs? (y/N): " -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        if [[ -d "$LOG_DIR" ]]; then
            rm -rf "$LOG_DIR"
            log_info "Removed: $LOG_DIR"
        fi
    else
        log_info "Log files preserved"
    fi
}

# Remove binaries
remove_binaries() {
    log_step "Removing binaries..."
    
    if [[ -d "$INSTALL_PREFIX" ]]; then
        rm -rf "$INSTALL_PREFIX"
        log_info "Removed: $INSTALL_PREFIX"
    fi
}

# Remove user and group
remove_user() {
    log_step "Removing NRDOT user and group..."
    
    # Remove user
    if getent passwd nrdot &>/dev/null; then
        userdel nrdot
        log_info "Removed user: nrdot"
    fi
    
    # Remove group
    if getent group nrdot &>/dev/null; then
        groupdel nrdot 2>/dev/null || true
        log_info "Removed group: nrdot"
    fi
}

# Clean up system changes
cleanup_system() {
    log_step "Cleaning up system changes..."
    
    # Remove BPF mount from fstab
    if grep -q "bpf /sys/fs/bpf" /etc/fstab; then
        sed -i '/bpf \/sys\/fs\/bpf/d' /etc/fstab
        log_info "Removed BPF mount from fstab"
    fi
    
    # Note: We don't revert sysctl changes as they might be used by other applications
    log_info "System cleanup completed"
}

# Purge all NRDOT files
purge_all() {
    log_warn "Purging ALL NRDOT files and settings..."
    
    # Remove everything without asking
    rm -rf "$INSTALL_PREFIX"
    rm -rf "$CONFIG_DIR"
    rm -rf "$DATA_DIR"
    rm -rf "$LOG_DIR"
    rm -rf /var/cache/nrdot
    rm -rf /run/nrdot
    rm -f /etc/sysctl.d/99-nrdot.conf
    rm -f /etc/security/limits.d/nrdot.conf
    rm -f /etc/logrotate.d/nrdot
    
    # Remove systemd files
    rm -f "$SYSTEMD_DIR"/nrdot*
    
    # Remove backups
    if [[ -d /var/backups/nrdot ]]; then
        log_info "Found backups in /var/backups/nrdot"
        read -p "Remove all backups? (y/N): " -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            rm -rf /var/backups/nrdot
        fi
    fi
    
    log_info "Purge completed"
}

# Uninstallation summary
show_summary() {
    log_step "Uninstallation Summary"
    
    echo
    echo "NRDOT uninstallation completed!"
    echo
    
    # Check what remains
    if [[ -d "$CONFIG_DIR" ]]; then
        echo "Configuration preserved in: $CONFIG_DIR"
    fi
    
    if [[ -d "$DATA_DIR" ]]; then
        echo "Data preserved in: $DATA_DIR"
    fi
    
    if [[ -d "$LOG_DIR" ]]; then
        echo "Logs preserved in: $LOG_DIR"
    fi
    
    if [[ -d /var/backups/nrdot ]]; then
        echo "Backups available in: /var/backups/nrdot"
    fi
    
    echo
}

# Main uninstallation function
main() {
    show_banner
    
    # Parse arguments
    PURGE=false
    while [[ $# -gt 0 ]]; do
        case $1 in
            --purge)
                PURGE=true
                shift
                ;;
            --help|-h)
                echo "Usage: $0 [OPTIONS]"
                echo "Options:"
                echo "  --purge    Remove all files without prompting"
                echo "  --help     Show this help message"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    check_prerequisites
    
    if [[ "$PURGE" == "true" ]]; then
        purge_all
        remove_user
        cleanup_system
    else
        # Interactive uninstall
        remove_systemd
        remove_binaries
        remove_configs
        remove_data
        remove_logs
        
        # Ask about user removal
        read -p "Remove NRDOT user and group? (y/N): " -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            remove_user
        fi
        
        cleanup_system
    fi
    
    show_summary
    log_info "Uninstallation completed"
}

# Run main function
main "$@"