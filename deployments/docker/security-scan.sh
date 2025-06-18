#!/bin/bash
# Security scanning for NRDOT Docker images

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
REGISTRY=${REGISTRY:-docker.io/newrelic}
TAG=${TAG:-latest}
SCANNER=${SCANNER:-trivy}
SEVERITY=${SEVERITY:-HIGH,CRITICAL}
EXIT_ON_VULN=${EXIT_ON_VULN:-true}
REPORT_DIR=${REPORT_DIR:-./security-reports}
FORMAT=${FORMAT:-table}

# Components to scan
COMPONENTS=(
    "base"
    "collector"
    "supervisor"
    "config-engine"
    "api-server"
    "privileged-helper"
    "ctl"
)

# Function to print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Function to check if scanner is installed
check_scanner() {
    case $SCANNER in
        trivy)
            if ! command -v trivy &> /dev/null; then
                print_error "Trivy not installed. Installing..."
                install_trivy
            fi
            ;;
        grype)
            if ! command -v grype &> /dev/null; then
                print_error "Grype not installed. Please install: https://github.com/anchore/grype"
                exit 1
            fi
            ;;
        *)
            print_error "Unknown scanner: $SCANNER"
            exit 1
            ;;
    esac
}

# Function to install Trivy
install_trivy() {
    print_info "Installing Trivy..."
    curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin
}

# Function to scan with Trivy
scan_with_trivy() {
    local image=$1
    local report_file=$2
    local format=$3
    
    trivy image \
        --severity "$SEVERITY" \
        --format "$format" \
        --output "$report_file" \
        --timeout 10m \
        "$image"
}

# Function to scan with Grype
scan_with_grype() {
    local image=$1
    local report_file=$2
    local format=$3
    
    grype "$image" \
        --fail-on "$SEVERITY" \
        --output "$format" \
        --file "$report_file"
}

# Function to scan image
scan_image() {
    local component=$1
    local image="${REGISTRY}/nrdot-${component}:${TAG}"
    
    print_info "Scanning $image..."
    
    # Create report directory
    mkdir -p "$REPORT_DIR"
    
    # Generate report filename
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local report_file="${REPORT_DIR}/${component}_${timestamp}.${FORMAT}"
    
    # Run scan
    local scan_exit_code=0
    case $SCANNER in
        trivy)
            scan_with_trivy "$image" "$report_file" "$FORMAT" || scan_exit_code=$?
            ;;
        grype)
            scan_with_grype "$image" "$report_file" "$FORMAT" || scan_exit_code=$?
            ;;
    esac
    
    # Check results
    if [ $scan_exit_code -eq 0 ]; then
        print_success "$component: No vulnerabilities found"
        return 0
    else
        print_warning "$component: Vulnerabilities found (see $report_file)"
        
        # Show summary for table format
        if [ "$FORMAT" = "table" ]; then
            echo "Summary:"
            cat "$report_file" | grep -E "Total:|HIGH:|CRITICAL:" || true
        fi
        
        return $scan_exit_code
    fi
}

# Function to generate HTML report
generate_html_report() {
    local html_file="${REPORT_DIR}/security-report-$(date +%Y%m%d_%H%M%S).html"
    
    cat > "$html_file" <<EOF
<!DOCTYPE html>
<html>
<head>
    <title>NRDOT Security Scan Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        h1 { color: #333; }
        .summary { background: #f0f0f0; padding: 10px; margin: 20px 0; }
        .component { margin: 20px 0; padding: 10px; border: 1px solid #ddd; }
        .success { color: green; }
        .warning { color: orange; }
        .error { color: red; }
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <h1>NRDOT Docker Security Scan Report</h1>
    <div class="summary">
        <p><strong>Date:</strong> $(date)</p>
        <p><strong>Scanner:</strong> $SCANNER</p>
        <p><strong>Severity Filter:</strong> $SEVERITY</p>
    </div>
EOF
    
    # Add component results
    for component in "${COMPONENTS[@]}"; do
        local latest_report=$(ls -t "${REPORT_DIR}/${component}_"*.json 2>/dev/null | head -1)
        if [ -f "$latest_report" ]; then
            echo "<div class='component'>" >> "$html_file"
            echo "<h2>$component</h2>" >> "$html_file"
            
            # Parse JSON report for summary (if using Trivy JSON format)
            if [ "$SCANNER" = "trivy" ] && command -v jq &> /dev/null; then
                local vuln_count=$(jq '.Results[0].Vulnerabilities | length' "$latest_report" 2>/dev/null || echo "0")
                echo "<p>Vulnerabilities found: $vuln_count</p>" >> "$html_file"
            fi
            
            echo "</div>" >> "$html_file"
        fi
    done
    
    echo "</body></html>" >> "$html_file"
    print_info "HTML report generated: $html_file"
}

# Function to scan all images
scan_all() {
    local failed=0
    local total=${#COMPONENTS[@]}
    
    print_info "Starting security scan of $total components"
    print_info "Scanner: $SCANNER"
    print_info "Severity: $SEVERITY"
    print_info "Report format: $FORMAT"
    
    # Scan each component
    for component in "${COMPONENTS[@]}"; do
        if scan_image "$component"; then
            :
        else
            failed=$((failed + 1))
        fi
    done
    
    # Generate summary
    echo
    print_info "Scan Summary:"
    print_info "  Total: $total"
    print_info "  Clean: $((total - failed))"
    print_info "  With vulnerabilities: $failed"
    
    # Generate HTML report if requested
    if [ "$FORMAT" = "json" ]; then
        generate_html_report
    fi
    
    # Exit based on results
    if [ $failed -gt 0 ] && [ "$EXIT_ON_VULN" = "true" ]; then
        print_error "Security scan failed: $failed components have vulnerabilities"
        exit 1
    else
        print_success "Security scan completed"
    fi
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --registry)
            REGISTRY="$2"
            shift 2
            ;;
        --tag)
            TAG="$2"
            shift 2
            ;;
        --scanner)
            SCANNER="$2"
            shift 2
            ;;
        --severity)
            SEVERITY="$2"
            shift 2
            ;;
        --format)
            FORMAT="$2"
            shift 2
            ;;
        --report-dir)
            REPORT_DIR="$2"
            shift 2
            ;;
        --no-fail)
            EXIT_ON_VULN=false
            shift
            ;;
        --component)
            # Scan only specific component
            check_scanner
            scan_image "$2"
            exit $?
            ;;
        --help)
            echo "Usage: $0 [options]"
            echo "Options:"
            echo "  --registry REGISTRY  Docker registry (default: docker.io/newrelic)"
            echo "  --tag TAG           Image tag (default: latest)"
            echo "  --scanner SCANNER   Scanner to use: trivy, grype (default: trivy)"
            echo "  --severity SEVERITY Severity levels (default: HIGH,CRITICAL)"
            echo "  --format FORMAT     Output format: table, json, sarif (default: table)"
            echo "  --report-dir DIR    Report directory (default: ./security-reports)"
            echo "  --no-fail           Don't exit with error on vulnerabilities"
            echo "  --component NAME    Scan only specific component"
            echo "  --help              Show this help message"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Check scanner availability
check_scanner

# Run scan
scan_all