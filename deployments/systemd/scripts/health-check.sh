#!/bin/bash
# NRDOT Health Check Script

set -euo pipefail

# Default values
SERVICE=""
CHECK_TYPE="status"
TIMEOUT=10

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        pre-start)
            CHECK_TYPE="pre-start"
            shift
            ;;
        --timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        *)
            SERVICE="$1"
            shift
            ;;
    esac
done

# Validate service name
if [[ -z "$SERVICE" ]]; then
    echo "Error: Service name required"
    exit 1
fi

# Health check functions
check_collector() {
    case "$CHECK_TYPE" in
        pre-start)
            # Check network interfaces
            if ! ip link show &>/dev/null; then
                echo "Error: Cannot access network interfaces"
                exit 1
            fi
            
            # Check BPF mount
            if [[ ! -d /sys/fs/bpf ]]; then
                echo "Warning: BPF filesystem not mounted"
            fi
            ;;
        status)
            # Check if collector is responding
            if [[ -S /run/nrdot/collector/control.sock ]]; then
                timeout "$TIMEOUT" nc -U /run/nrdot/collector/control.sock <<< "PING" | grep -q "PONG" || exit 1
            else
                # Check process
                pgrep -f "nrdot-collector" &>/dev/null || exit 1
            fi
            
            # Check metrics endpoint
            if curl -sf "http://localhost:9090/metrics" | grep -q "nrdot_collector_up 1"; then
                exit 0
            else
                exit 1
            fi
            ;;
    esac
}

check_supervisor() {
    case "$CHECK_TYPE" in
        pre-start)
            # Check systemd access
            if ! systemctl --version &>/dev/null; then
                echo "Error: systemd not available"
                exit 1
            fi
            ;;
        status)
            # Check supervisor socket
            if [[ -S /run/nrdot/supervisor/control.sock ]]; then
                timeout "$TIMEOUT" nc -U /run/nrdot/supervisor/control.sock <<< "STATUS" | grep -q "OK" || exit 1
            else
                pgrep -f "nrdot-supervisor" &>/dev/null || exit 1
            fi
            ;;
    esac
}

check_config_engine() {
    case "$CHECK_TYPE" in
        pre-start)
            # Check config directory
            if [[ ! -d /etc/nrdot ]]; then
                echo "Error: Config directory not found"
                exit 1
            fi
            ;;
        status)
            # Check if engine is processing
            if [[ -f /run/nrdot/config-engine/status ]]; then
                STATUS=$(cat /run/nrdot/config-engine/status)
                [[ "$STATUS" == "healthy" ]] || exit 1
            else
                pgrep -f "nrdot-config-engine" &>/dev/null || exit 1
            fi
            ;;
    esac
}

check_api_server() {
    case "$CHECK_TYPE" in
        pre-start)
            # Check port availability
            if ss -tln | grep -q ":8080 "; then
                echo "Warning: Port 8080 already in use"
            fi
            
            # Check certificates
            if [[ ! -f /etc/nrdot/certs/server.crt ]]; then
                echo "Warning: TLS certificate not found"
            fi
            ;;
        status)
            # Check API health endpoint
            RESPONSE=$(curl -sk -w "\n%{http_code}" https://localhost:8080/health 2>/dev/null || echo "000")
            HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
            
            if [[ "$HTTP_CODE" == "200" ]]; then
                exit 0
            else
                # Try HTTP as fallback
                RESPONSE=$(curl -s -w "\n%{http_code}" http://localhost:8080/health 2>/dev/null || echo "000")
                HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
                [[ "$HTTP_CODE" == "200" ]] || exit 1
            fi
            ;;
    esac
}

check_privileged_helper() {
    case "$CHECK_TYPE" in
        pre-start)
            # Check if running as root
            if [[ $EUID -ne 0 ]]; then
                echo "Error: Privileged helper must run as root"
                exit 1
            fi
            
            # Check capabilities
            if ! command -v capsh &>/dev/null; then
                echo "Warning: capsh not found, cannot verify capabilities"
            fi
            ;;
        status)
            # Check helper socket
            if [[ -S /run/nrdot/privileged.sock ]]; then
                # Socket exists, check permissions
                SOCKET_OWNER=$(stat -c '%U' /run/nrdot/privileged.sock)
                [[ "$SOCKET_OWNER" == "root" ]] || exit 1
            else
                # Check process running as root
                pgrep -f "nrdot-privileged-helper" -u 0 &>/dev/null || exit 1
            fi
            ;;
    esac
}

# Main health check logic
case "$SERVICE" in
    collector)
        check_collector
        ;;
    supervisor)
        check_supervisor
        ;;
    config-engine)
        check_config_engine
        ;;
    api-server)
        check_api_server
        ;;
    privileged-helper)
        check_privileged_helper
        ;;
    *)
        echo "Error: Unknown service: $SERVICE"
        exit 1
        ;;
esac

exit 0