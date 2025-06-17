#!/bin/bash
# Comprehensive validation script for clean-platform-implementation
# Checks all components and reports compliance status

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0
WARNINGS=0

# Function to print status
print_status() {
    local status=$1
    local message=$2
    ((TOTAL_CHECKS++))
    
    case $status in
        "PASS")
            echo -e "${GREEN}✓${NC} $message"
            ((PASSED_CHECKS++))
            ;;
        "FAIL")
            echo -e "${RED}✗${NC} $message"
            ((FAILED_CHECKS++))
            ;;
        "WARN")
            echo -e "${YELLOW}⚠${NC} $message"
            ((WARNINGS++))
            ;;
    esac
}

# Function to check file exists
check_file() {
    local file=$1
    local description=$2
    
    if [ -f "$file" ]; then
        print_status "PASS" "$description exists"
        return 0
    else
        print_status "FAIL" "$description missing: $file"
        return 1
    fi
}

# Function to check directory exists
check_dir() {
    local dir=$1
    local description=$2
    
    if [ -d "$dir" ]; then
        print_status "PASS" "$description exists"
        return 0
    else
        print_status "FAIL" "$description missing: $dir"
        return 1
    fi
}

# Function to check for placeholders
check_no_placeholders() {
    local file=$1
    local description=$2
    
    if [ -f "$file" ]; then
        if grep -q '\${.*}' "$file" || grep -q 'YOUR_.*_ID' "$file" || grep -q 'TEAMSTORE_ID' "$file"; then
            print_status "WARN" "$description contains placeholders that need to be replaced"
            return 1
        else
            print_status "PASS" "$description has no placeholders"
            return 0
        fi
    fi
}

echo "=================================================="
echo "Clean Platform Implementation Validation"
echo "=================================================="
echo ""

# Change to script directory
cd "$(dirname "$0")/.."

echo "1. Checking Deployment Scripts..."
echo "---------------------------------"
for script in backup-check.sh health-check.sh smoke-tests.sh rollback.sh notify-failure.sh suppress-alerts.sh restore-alerts.sh; do
    check_file "scripts/deployment-hooks/$script" "Deployment hook: $script"
    if [ -f "scripts/deployment-hooks/$script" ]; then
        if [ -x "scripts/deployment-hooks/$script" ]; then
            print_status "PASS" "$script is executable"
        else
            print_status "FAIL" "$script is not executable"
        fi
    fi
done

echo ""
echo "2. Checking Configuration Files..."
echo "-----------------------------------"
check_file "grandcentral.yml" "Grand Central configuration"
check_file "jenkins.yml" "Jenkins configuration"
check_file "team-permissions.yml" "Team permissions"
check_file "Dockerfile" "Main Dockerfile"

# Check for placeholders
check_no_placeholders "jenkins.yml" "Jenkins configuration"
check_no_placeholders "team-permissions.yml" "Team permissions"

echo ""
echo "3. Checking Kubernetes Resources..."
echo "------------------------------------"
check_file "k8s/base/deployments/data-collector.yaml" "Data collector deployment"
check_file "k8s/base/services/data-collector-service.yaml" "Data collector service"
check_file "k8s/base/redis/redis-deployment.yaml" "Redis deployment"
check_file "k8s/base/security/kyverno-policies.yaml" "Kyverno security policies"
check_file "k8s/base/rbac/service-account-rbac.yaml" "RBAC configuration"

echo ""
echo "4. Checking Service Implementations..."
echo "---------------------------------------"
check_file "services/data-collector/app.py" "Data collector application"
check_file "services/data-collector/health_server.py" "Health check server"
check_file "services/data-collector/start.sh" "Startup script"
check_file "services/grand_central/gc_client.py" "Grand Central client"
check_file "services/team-access/okta_integration.py" "Okta integration"

echo ""
echo "5. Checking Security Components..."
echo "-----------------------------------"
check_file "scripts/scan-image.sh" "Image security scanner"
if [ -f "scripts/scan-image.sh" ] && [ -x "scripts/scan-image.sh" ]; then
    print_status "PASS" "Image scanner is executable"
else
    print_status "FAIL" "Image scanner is not executable"
fi

# Check Dockerfile for security
if grep -q "USER appuser" services/data-collector/Dockerfile && \
   grep -q "useradd -r -u 10001" services/data-collector/Dockerfile; then
    print_status "PASS" "Dockerfile uses non-root user with UID > 10000"
else
    print_status "FAIL" "Dockerfile security requirements not met"
fi

echo ""
echo "6. Checking Terraform Configuration..."
echo "---------------------------------------"
check_dir "terraform/environments/staging" "Staging environment"
check_dir "terraform/environments/production" "Production environment"
check_file "terraform/environments/staging/backend.tf" "Staging backend config"
check_file "terraform/environments/staging/variables.tf" "Staging variables"

echo ""
echo "7. Checking Test Coverage..."
echo "-----------------------------"
check_file "tests/unit/test_data_collector.py" "Data collector unit tests"
check_file "tests/unit/test_grand_central_client.py" "Grand Central client tests"
check_file "tests/unit/test_okta_integration.py" "Okta integration tests"
check_file "tests/integration/test_deployment_flow.py" "Deployment flow tests"
check_file "tests/e2e/test_platform_integration.py" "E2E platform tests"

echo ""
echo "8. Checking Documentation..."
echo "-----------------------------"
check_file "README.md" "Main README"
check_file "CLAUDE.md" "Claude instructions"
check_file "IMPLEMENTATION-STATUS-FINAL.md" "Implementation status"

echo ""
echo "9. Checking Critical Integrations..."
echo "-------------------------------------"
# Check Grand Central configuration
if grep -q "grand_central_api:" grandcentral.yml && \
   grep -q "apm_verification:" grandcentral.yml && \
   grep -q "entity_synthesis:" grandcentral.yml; then
    print_status "PASS" "Grand Central fully configured"
else
    print_status "FAIL" "Grand Central configuration incomplete"
fi

# Check Kyverno policies
if grep -q "require-fips-compliant-images" k8s/base/security/kyverno-policies.yaml && \
   grep -q "CIS-DI-0001" k8s/base/security/kyverno-policies.yaml; then
    print_status "PASS" "Kyverno policies include FIPS and CIS benchmarks"
else
    print_status "FAIL" "Kyverno policies missing requirements"
fi

echo ""
echo "10. Checking Port Configuration..."
echo "------------------------------------"
# Check health check port configuration
if grep -q "8081" services/data-collector/health_server.py && \
   grep -q "containerPort: 8081" k8s/base/deployments/data-collector.yaml; then
    print_status "PASS" "Health check port 8081 properly configured"
else
    print_status "FAIL" "Health check port mismatch"
fi

echo ""
echo "=================================================="
echo "Validation Summary"
echo "=================================================="
echo "Total Checks: $TOTAL_CHECKS"
echo -e "Passed: ${GREEN}$PASSED_CHECKS${NC}"
echo -e "Failed: ${RED}$FAILED_CHECKS${NC}"
echo -e "Warnings: ${YELLOW}$WARNINGS${NC}"
echo ""

# Calculate compliance percentage
COMPLIANCE=$(echo "scale=2; ($PASSED_CHECKS * 100) / $TOTAL_CHECKS" | bc)
echo "Compliance Score: ${COMPLIANCE}%"

# Determine overall status
if [ "$FAILED_CHECKS" -eq 0 ]; then
    echo -e "\n${GREEN}✓ All critical checks passed!${NC}"
    exit 0
elif [ "$FAILED_CHECKS" -lt 5 ]; then
    echo -e "\n${YELLOW}⚠ Some issues found but implementation is mostly complete${NC}"
    exit 1
else
    echo -e "\n${RED}✗ Multiple critical issues found${NC}"
    exit 2
fi