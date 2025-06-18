#!/bin/bash
# NRDOT Pre-uninstallation Script

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

# Stop all NRDOT services
stop_services() {
    log_info "Stopping NRDOT services..."
    
    # Stop target and all dependent services
    systemctl stop nrdot.target 2>/dev/null || true
    
    # Stop individual services
    SERVICES=(
        "nrdot-api-server"
        "nrdot-config-engine"
        "nrdot-collector"
        "nrdot-supervisor"
        "nrdot-privileged-helper"
    )
    
    for service in "${SERVICES[@]}"; do
        if systemctl is-active --quiet "$service" 2>/dev/null; then
            log_info "Stopping $service..."
            systemctl stop "$service"
        fi
    done
    
    # Stop sockets
    systemctl stop nrdot-api.socket 2>/dev/null || true
    systemctl stop nrdot-privileged.socket 2>/dev/null || true
    
    # Wait for services to fully stop
    sleep 5
}

# Export data and configurations
export_data() {
    log_info "Exporting data and configurations..."
    
    BACKUP_DIR="/var/backups/nrdot/uninstall-$(date +%Y%m%d-%H%M%S)"
    mkdir -p "$BACKUP_DIR"
    
    # Backup configurations
    if [[ -d /etc/nrdot ]]; then
        log_info "Backing up configurations..."
        tar -czf "$BACKUP_DIR/etc-nrdot.tar.gz" -C / etc/nrdot
    fi
    
    # Backup data
    if [[ -d /var/lib/nrdot ]]; then
        log_info "Backing up data..."
        tar -czf "$BACKUP_DIR/var-lib-nrdot.tar.gz" -C / var/lib/nrdot
    fi
    
    # Backup logs
    if [[ -d /var/log/nrdot ]]; then
        log_info "Backing up logs..."
        tar -czf "$BACKUP_DIR/var-log-nrdot.tar.gz" -C / var/log/nrdot
    fi
    
    # Export database if applicable
    if [[ -x /opt/nrdot/bin/nrdot-db-export ]]; then
        log_info "Exporting database..."
        /opt/nrdot/bin/nrdot-db-export \
            --config /etc/nrdot/database.yaml \
            --output "$BACKUP_DIR/database-export.sql" 2>/dev/null || true
    fi
    
    # Save uninstall report
    {
        echo "NRDOT Uninstall Report"
        echo "======================"
        echo "Date: $(date)"
        echo "Hostname: $(hostname)"
        echo "Backup Location: $BACKUP_DIR"
        echo ""
        echo "Services Status Before Uninstall:"
        systemctl status 'nrdot-*' --no-pager 2>/dev/null || true
        echo ""
        echo "Disk Usage:"
        du -sh /opt/nrdot /etc/nrdot /var/lib/nrdot /var/log/nrdot 2>/dev/null || true
    } > "$BACKUP_DIR/uninstall-report.txt"
    
    log_info "Data exported to: $BACKUP_DIR"
}

# Disable services
disable_services() {
    log_info "Disabling NRDOT services..."
    
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
    
    for service in "${SERVICES[@]}"; do
        systemctl disable "$service" 2>/dev/null || true
    done
    
    # Reload systemd
    systemctl daemon-reload
}

# Remove firewall rules
remove_firewall_rules() {
    log_info "Removing firewall rules..."
    
    # Remove firewalld rules
    if systemctl is-active --quiet firewalld 2>/dev/null; then
        firewall-cmd --permanent --remove-service=nrdot 2>/dev/null || true
        firewall-cmd --permanent --delete-service=nrdot 2>/dev/null || true
        firewall-cmd --reload
        log_info "Firewalld rules removed"
    fi
    
    # Remove UFW rules
    if command -v ufw &> /dev/null && ufw status | grep -q "Status: active"; then
        ufw delete allow 8080/tcp 2>/dev/null || true
        ufw delete allow 9090/tcp 2>/dev/null || true
        log_info "UFW rules removed"
    fi
}

# Clean up runtime files
cleanup_runtime() {
    log_info "Cleaning up runtime files..."
    
    # Remove PID files
    rm -rf /run/nrdot
    
    # Remove temporary files
    rm -f /tmp/.nrdot-*
    
    # Clean systemd runtime
    systemctl reset-failed 'nrdot-*' 2>/dev/null || true
}

# Confirmation prompt
confirm_uninstall() {
    echo
    log_warn "This will uninstall NRDOT and stop all services."
    log_warn "Data will be backed up to: /var/backups/nrdot/"
    echo
    read -p "Are you sure you want to continue? (yes/no): " -r
    echo
    
    if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
        log_info "Uninstall cancelled"
        exit 0
    fi
}

# Main execution
main() {
    log_info "Starting NRDOT pre-uninstallation process..."
    
    check_root
    confirm_uninstall
    stop_services
    export_data
    disable_services
    remove_firewall_rules
    cleanup_runtime
    
    log_info "Pre-uninstallation completed"
    log_info "Backup saved to: /var/backups/nrdot/"
    
    # Create flag file for uninstaller
    touch /tmp/.nrdot-preuninstall-complete
}

# Run main function
main "$@"