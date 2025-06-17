#!/bin/bash
# Image security scanning integration with PIE scan tool
# Required by container_security_combined.md

set -euo pipefail

# Configuration
IMAGE="${1:-}"
OUTPUT_DIR="${2:-./scan-results}"
SCAN_TIMEOUT="${SCAN_TIMEOUT:-600}"
PIE_SCAN_URL="${PIE_SCAN_URL:-https://pie-scan.nr-ops.net}"

# Validation
if [ -z "$IMAGE" ]; then
    echo "ERROR: Image name required"
    echo "Usage: $0 <image-name> [output-dir]"
    exit 1
fi

# Ensure output directory exists
mkdir -p "$OUTPUT_DIR"

# Generate scan ID
SCAN_ID="scan-$(date +%Y%m%d-%H%M%S)-$$"
RESULT_FILE="$OUTPUT_DIR/${SCAN_ID}.json"

echo "Starting security scan for image: $IMAGE"
echo "Scan ID: $SCAN_ID"

# Function to check FIPS compliance
check_fips_compliance() {
    local image="$1"
    
    # Check if image uses FIPS-compliant base
    if [[ "$image" =~ cf-registry\.nr-ops\.net/newrelic/.*-fips:.* ]] || \
       [[ "$image" =~ cf-registry\.nr-ops\.net/.*/fips-.* ]]; then
        echo "✓ FIPS-compliant base image detected"
        return 0
    else
        echo "✗ Non-FIPS base image detected"
        return 1
    fi
}

# Function to run PIE scan
run_pie_scan() {
    local image="$1"
    local output="$2"
    
    # Call PIE scan API
    curl -s -X POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${PIE_SCAN_TOKEN:-}" \
        -d "{\"image\": \"$image\", \"scanType\": [\"trivy\", \"dockle\"]}" \
        "${PIE_SCAN_URL}/api/v1/scan" \
        -o "$output.tmp"
    
    # Parse and format results
    jq '{
        scanId: .scanId,
        image: .image,
        timestamp: .timestamp,
        vulnerabilities: .results.trivy.vulnerabilities,
        compliance: .results.dockle.compliance,
        summary: {
            critical: (.results.trivy.vulnerabilities | map(select(.severity == "CRITICAL")) | length),
            high: (.results.trivy.vulnerabilities | map(select(.severity == "HIGH")) | length),
            medium: (.results.trivy.vulnerabilities | map(select(.severity == "MEDIUM")) | length),
            low: (.results.trivy.vulnerabilities | map(select(.severity == "LOW")) | length),
            cisFailures: (.results.dockle.compliance | map(select(.pass == false)) | length)
        }
    }' "$output.tmp" > "$output"
    
    rm -f "$output.tmp"
}

# Function to check CIS benchmarks
check_cis_benchmarks() {
    local result_file="$1"
    local failures=0
    
    echo "Checking CIS Docker benchmarks..."
    
    # CIS-DI-0001: Create a user for the container
    if jq -e '.compliance[] | select(.checkID == "CIS-DI-0001" and .pass == false)' "$result_file" > /dev/null; then
        echo "✗ CIS-DI-0001: Container must run as non-root user"
        ((failures++))
    else
        echo "✓ CIS-DI-0001: Non-root user check passed"
    fi
    
    # CIS-DI-0007: Do not use update instructions alone in Dockerfile
    if jq -e '.compliance[] | select(.checkID == "CIS-DI-0007" and .pass == false)' "$result_file" > /dev/null; then
        echo "✗ CIS-DI-0007: Update instructions must be combined with install"
        ((failures++))
    else
        echo "✓ CIS-DI-0007: Update instruction check passed"
    fi
    
    # CIS-DI-0008: Remove setuid and setgid permissions
    if jq -e '.compliance[] | select(.checkID == "CIS-DI-0008" and .pass == false)' "$result_file" > /dev/null; then
        echo "✗ CIS-DI-0008: Setuid/setgid permissions found"
        ((failures++))
    else
        echo "✓ CIS-DI-0008: Setuid/setgid check passed"
    fi
    
    # CIS-DI-0009: Use COPY instead of ADD
    if jq -e '.compliance[] | select(.checkID == "CIS-DI-0009" and .pass == false)' "$result_file" > /dev/null; then
        echo "✗ CIS-DI-0009: ADD instruction should be replaced with COPY"
        ((failures++))
    else
        echo "✓ CIS-DI-0009: COPY instruction check passed"
    fi
    
    # CIS-DI-0010: Do not store secrets in images
    if jq -e '.compliance[] | select(.checkID == "CIS-DI-0010" and .pass == false)' "$result_file" > /dev/null; then
        echo "✗ CIS-DI-0010: Secrets detected in image"
        ((failures++))
    else
        echo "✓ CIS-DI-0010: No secrets detected"
    fi
    
    return $failures
}

# Main scanning process
echo "Running security scans..."

# Check FIPS compliance
FIPS_COMPLIANT=true
if ! check_fips_compliance "$IMAGE"; then
    FIPS_COMPLIANT=false
fi

# Run PIE scan
echo "Running PIE security scan..."
if ! run_pie_scan "$IMAGE" "$RESULT_FILE"; then
    echo "ERROR: PIE scan failed"
    exit 1
fi

# Check for critical vulnerabilities
CRITICAL_COUNT=$(jq -r '.summary.critical' "$RESULT_FILE")
HIGH_COUNT=$(jq -r '.summary.high' "$RESULT_FILE")

echo ""
echo "Vulnerability Summary:"
echo "- Critical: $CRITICAL_COUNT"
echo "- High: $HIGH_COUNT"
echo "- Medium: $(jq -r '.summary.medium' "$RESULT_FILE")"
echo "- Low: $(jq -r '.summary.low' "$RESULT_FILE")"
echo ""

# Check CIS benchmarks
CIS_FAILURES=0
check_cis_benchmarks "$RESULT_FILE" || CIS_FAILURES=$?

echo ""
echo "CIS Benchmark Failures: $CIS_FAILURES"

# Generate final report
SCAN_PASSED=true
SCAN_STATUS="passed"
FAILURE_REASONS=()

if [ "$CRITICAL_COUNT" -gt 0 ]; then
    SCAN_PASSED=false
    FAILURE_REASONS+=("$CRITICAL_COUNT critical vulnerabilities found")
fi

if [ "$FIPS_COMPLIANT" = "false" ]; then
    SCAN_PASSED=false
    FAILURE_REASONS+=("Non-FIPS compliant base image")
fi

if [ "$CIS_FAILURES" -gt 0 ]; then
    SCAN_PASSED=false
    FAILURE_REASONS+=("$CIS_FAILURES CIS benchmark failures")
fi

if [ "$SCAN_PASSED" = "false" ]; then
    SCAN_STATUS="failed"
fi

# Create attestation file
ATTESTATION_FILE="$OUTPUT_DIR/${SCAN_ID}-attestation.json"
cat > "$ATTESTATION_FILE" << EOF
{
  "scanId": "$SCAN_ID",
  "image": "$IMAGE",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "status": "$SCAN_STATUS",
  "fipsCompliant": $FIPS_COMPLIANT,
  "vulnerabilities": {
    "critical": $CRITICAL_COUNT,
    "high": $HIGH_COUNT
  },
  "cisFailures": $CIS_FAILURES,
  "attestation": {
    "type": "security-scan",
    "predicateType": "https://nr-ops.net/attestation/security-scan/v1",
    "subject": {
      "name": "$IMAGE",
      "digest": "$(docker inspect --format='{{index .RepoDigests 0}}' "$IMAGE" 2>/dev/null || echo 'unknown')"
    }
  }
}
EOF

echo ""
echo "Scan Results:"
echo "- Status: $SCAN_STATUS"
echo "- Report: $RESULT_FILE"
echo "- Attestation: $ATTESTATION_FILE"

if [ "$SCAN_PASSED" = "false" ]; then
    echo ""
    echo "ERROR: Security scan failed!"
    for reason in "${FAILURE_REASONS[@]}"; do
        echo "  - $reason"
    done
    exit 1
fi

echo ""
echo "✓ Security scan passed!"

# Label the image with scan status (if we have docker access)
if command -v docker &> /dev/null; then
    docker label "$IMAGE" \
        security.scan.status="$SCAN_STATUS" \
        security.scan.timestamp="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        security.scan.id="$SCAN_ID" \
        security.fips.compliant="$FIPS_COMPLIANT" \
        2>/dev/null || true
fi

exit 0