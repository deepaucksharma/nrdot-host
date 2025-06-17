#!/bin/bash
# NRDOT Post-installation Script

set -euo pipefail

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

log_debug() {
    echo -e "${BLUE}[DEBUG]${NC} $1"
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

# Set up log rotation
setup_logrotate() {
    log_info "Setting up log rotation..."
    
    cat > /etc/logrotate.d/nrdot <<EOF
/var/log/nrdot/*.log {
    daily
    rotate 14
    maxsize 100M
    missingok
    notifempty
    compress
    delaycompress
    sharedscripts
    create 0640 nrdot nrdot
    postrotate
        # Send SIGHUP to all NRDOT services to reopen log files
        systemctl reload 'nrdot-*.service' 2>/dev/null || true
    endscript
}
EOF
    
    log_info "Log rotation configured"
}

# Configure firewall rules
setup_firewall() {
    log_info "Configuring firewall rules..."
    
    # Check if firewalld is active
    if systemctl is-active --quiet firewalld 2>/dev/null; then
        log_info "Configuring firewalld..."
        
        # Create NRDOT service definition
        firewall-cmd --permanent --new-service=nrdot 2>/dev/null || true
        firewall-cmd --permanent --service=nrdot --add-port=8080/tcp
        firewall-cmd --permanent --service=nrdot --add-port=9090/tcp
        firewall-cmd --permanent --service=nrdot --set-description="NRDOT Network Resource Discovery"
        firewall-cmd --permanent --add-service=nrdot
        firewall-cmd --reload
        
        log_info "Firewalld rules added"
    
    # Check if ufw is active
    elif command -v ufw &> /dev/null && ufw status | grep -q "Status: active"; then
        log_info "Configuring UFW..."
        
        ufw allow 8080/tcp comment 'NRDOT API'
        ufw allow 9090/tcp comment 'NRDOT Metrics'
        
        log_info "UFW rules added"
    
    # Check if iptables is being used
    elif iptables -L -n &> /dev/null; then
        log_warn "iptables detected but no automatic rules added"
        log_warn "Please manually add rules for ports 8080 and 9090"
    else
        log_info "No active firewall detected"
    fi
}

# Set up SELinux contexts (if applicable)
setup_selinux() {
    if command -v getenforce &> /dev/null && [[ $(getenforce) != "Disabled" ]]; then
        log_info "Configuring SELinux contexts..."
        
        # Set contexts for NRDOT directories
        semanage fcontext -a -t bin_t "/opt/nrdot/bin(/.*)?" 2>/dev/null || true
        semanage fcontext -a -t etc_t "/etc/nrdot(/.*)?" 2>/dev/null || true
        semanage fcontext -a -t var_lib_t "/var/lib/nrdot(/.*)?" 2>/dev/null || true
        semanage fcontext -a -t var_log_t "/var/log/nrdot(/.*)?" 2>/dev/null || true
        
        # Apply contexts
        restorecon -Rv /opt/nrdot /etc/nrdot /var/lib/nrdot /var/log/nrdot 2>/dev/null || true
        
        # Allow NRDOT to bind to ports
        semanage port -a -t http_port_t -p tcp 8080 2>/dev/null || true
        semanage port -a -t http_port_t -p tcp 9090 2>/dev/null || true
        
        log_info "SELinux contexts configured"
    fi
}

# Set up AppArmor profiles (if applicable)
setup_apparmor() {
    if command -v aa-status &> /dev/null && systemctl is-active --quiet apparmor 2>/dev/null; then
        log_info "Setting up AppArmor profiles..."
        
        # Create basic AppArmor profile
        cat > /etc/apparmor.d/opt.nrdot.bin.nrdot-collector <<EOF
#include <tunables/global>

/opt/nrdot/bin/nrdot-collector {
  #include <abstractions/base>
  #include <abstractions/nameservice>
  
  capability net_raw,
  capability net_admin,
  capability net_bind_service,
  
  network inet stream,
  network inet dgram,
  network inet6 stream,
  network inet6 dgram,
  network packet raw,
  
  /opt/nrdot/bin/nrdot-collector mr,
  /etc/nrdot/** r,
  /var/lib/nrdot/collector/** rw,
  /var/log/nrdot/** rw,
  /run/nrdot/collector/** rw,
  /proc/sys/net/** r,
  /sys/class/net/** r,
  /sys/devices/** r,
}
EOF
        
        # Load profile
        apparmor_parser -r /etc/apparmor.d/opt.nrdot.bin.nrdot-collector 2>/dev/null || true
        
        log_info "AppArmor profiles created"
    fi
}

# Initialize database
init_database() {
    log_info "Initializing NRDOT database..."
    
    # Create database directory
    mkdir -p /var/lib/nrdot/db
    chown nrdot:nrdot /var/lib/nrdot/db
    chmod 750 /var/lib/nrdot/db
    
    # Initialize if binary exists
    if [[ -x /opt/nrdot/bin/nrdot-db-init ]]; then
        sudo -u nrdot /opt/nrdot/bin/nrdot-db-init \
            --config /etc/nrdot/database.yaml \
            --init-schema
        log_info "Database initialized"
    else
        log_warn "Database initialization binary not found"
    fi
}

# Generate initial configuration
generate_config() {
    log_info "Generating initial configuration..."
    
    # Generate API token
    API_TOKEN=$(openssl rand -hex 32)
    echo "$API_TOKEN" > /etc/nrdot/auth-token
    chmod 600 /etc/nrdot/auth-token
    chown nrdot:nrdot /etc/nrdot/auth-token
    
    # Generate self-signed certificate for API
    if [[ ! -f /etc/nrdot/certs/server.crt ]]; then
        log_info "Generating self-signed certificate..."
        mkdir -p /etc/nrdot/certs
        
        openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
            -keyout /etc/nrdot/certs/server.key \
            -out /etc/nrdot/certs/server.crt \
            -subj "/C=US/ST=State/L=City/O=NRDOT/CN=localhost" \
            2>/dev/null
        
        chmod 600 /etc/nrdot/certs/server.key
        chmod 644 /etc/nrdot/certs/server.crt
        chown -R nrdot:nrdot /etc/nrdot/certs
    fi
    
    log_info "Configuration generated"
}

# Enable and start services
start_services() {
    log_info "Enabling NRDOT services..."
    
    # Enable services
    systemctl daemon-reload
    systemctl enable nrdot-privileged-helper.service
    systemctl enable nrdot-supervisor.service
    systemctl enable nrdot-collector.service
    systemctl enable nrdot-config-engine.service
    systemctl enable nrdot-api-server.service
    systemctl enable nrdot.target
    
    log_info "Starting NRDOT services..."
    
    # Start services in order
    systemctl start nrdot-privileged-helper.service
    sleep 2
    systemctl start nrdot-supervisor.service
    sleep 2
    systemctl start nrdot.target
    
    # Wait for services to stabilize
    sleep 5
    
    # Check service status
    log_info "Checking service status..."
    systemctl --no-pager status nrdot.target || true
}

# Run health checks
run_health_checks() {
    log_info "Running health checks..."
    
    # Check if services are running
    SERVICES=(
        "nrdot-privileged-helper"
        "nrdot-supervisor"
        "nrdot-collector"
        "nrdot-config-engine"
        "nrdot-api-server"
    )
    
    ALL_HEALTHY=true
    
    for service in "${SERVICES[@]}"; do
        if systemctl is-active --quiet "$service"; then
            log_info "$service is running"
        else
            log_error "$service is not running"
            ALL_HEALTHY=false
        fi
    done
    
    # Check API endpoint
    if curl -sk https://localhost:8080/health &> /dev/null; then
        log_info "API server is responding"
    else
        log_warn "API server is not responding (this may be normal during initial startup)"
    fi
    
    if [[ "$ALL_HEALTHY" == "true" ]]; then
        log_info "All services are healthy"
    else
        log_warn "Some services are not running. Check logs in /var/log/nrdot/"
    fi
}

# Print summary
print_summary() {
    echo
    echo "========================================"
    echo "NRDOT Installation Complete!"
    echo "========================================"
    echo
    echo "API Server: https://localhost:8080"
    echo "API Token: $(cat /etc/nrdot/auth-token 2>/dev/null || echo 'Not found')"
    echo
    echo "Service Status:"
    systemctl list-units 'nrdot-*' --no-legend --no-pager
    echo
    echo "Logs: /var/log/nrdot/"
    echo "Config: /etc/nrdot/"
    echo "Data: /var/lib/nrdot/"
    echo
    echo "To view logs: journalctl -u nrdot-supervisor -f"
    echo "To stop all services: systemctl stop nrdot.target"
    echo "To start all services: systemctl start nrdot.target"
    echo
}

# Main execution
main() {
    log_info "Starting NRDOT post-installation configuration..."
    
    check_root
    setup_logrotate
    setup_firewall
    setup_selinux
    setup_apparmor
    init_database
    generate_config
    start_services
    run_health_checks
    print_summary
    
    log_info "Post-installation completed"
}

# Run main function
main "$@"