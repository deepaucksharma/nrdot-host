#!/bin/bash
# Check if Dockerfiles use FIPS-compliant base images

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

ERRORS=0
WARNINGS=0

# Required FIPS base image patterns
FIPS_PATTERNS=(
    "cf-registry.nr-ops.net/newrelic/.*-fips"
    "cf-registry.nr-ops.net/newrelic/infrastructure-k8s-fips"
    "cf-registry.nr-ops.net/newrelic/python-.*-fips"
    "cf-registry.nr-ops.net/newrelic/java-.*-fips"
    "cf-registry.nr-ops.net/newrelic/node-.*-fips"
    "cf-registry.nr-ops.net/newrelic/nginx-fips"
    "cf-registry.nr-ops.net/newrelic/ubuntu-.*-fips"
)

# Check if a base image is FIPS-compliant
is_fips_compliant() {
    local image="$1"
    
    for pattern in "${FIPS_PATTERNS[@]}"; do
        if [[ "$image" =~ $pattern ]]; then
            return 0
        fi
    done
    
    # Special case for platform-team images (must check if they're based on FIPS)
    if [[ "$image" =~ cf-registry.nr-ops.net/platform-team/.* ]]; then
        echo -e "${YELLOW}⚠️  Warning: Platform team image detected. Ensure it's based on FIPS image${NC}"
        ((WARNINGS++))
        return 0
    fi
    
    return 1
}

# Check each Dockerfile
for dockerfile in "$@"; do
    echo "Checking: $dockerfile"
    
    # Extract FROM statements
    from_statements=$(grep -E "^FROM\s+" "$dockerfile" || true)
    
    if [[ -z "$from_statements" ]]; then
        echo -e "${RED}❌ No FROM statement found${NC}"
        ((ERRORS++))
        continue
    fi
    
    # Check each FROM statement
    while IFS= read -r from_line; do
        # Extract the image name
        image=$(echo "$from_line" | sed -E 's/^FROM\s+([^ ]+).*/\1/')
        
        # Skip build stages (FROM ... AS ...)
        if [[ "$from_line" =~ "AS " ]]; then
            stage_name=$(echo "$from_line" | sed -E 's/.*AS\s+([^ ]+).*/\1/')
            # Check if this is referencing a previous stage
            if grep -qE "^FROM.*AS\s+$image" "$dockerfile"; then
                continue
            fi
        fi
        
        # Check if image is FIPS-compliant
        if ! is_fips_compliant "$image"; then
            echo -e "${RED}❌ Non-FIPS base image: $image${NC}"
            echo "   Required: Use FIPS-compliant image from cf-registry.nr-ops.net/newrelic/*-fips"
            ((ERRORS++))
        else
            echo -e "${GREEN}✅ FIPS-compliant: $image${NC}"
        fi
        
        # Additional checks
        if [[ "$image" == *":latest" ]]; then
            echo -e "${YELLOW}⚠️  Warning: Using :latest tag is discouraged${NC}"
            ((WARNINGS++))
        fi
        
    done <<< "$from_statements"
    
    # Check for required security practices
    if ! grep -q "USER " "$dockerfile"; then
        echo -e "${RED}❌ No USER statement found - must run as non-root${NC}"
        ((ERRORS++))
    else
        # Check if USER ID is >= 10000
        user_id=$(grep -E "^USER\s+[0-9]+" "$dockerfile" | sed -E 's/^USER\s+([0-9]+).*/\1/' | head -1)
        if [[ -n "$user_id" ]] && [[ "$user_id" -lt 10000 ]]; then
            echo -e "${RED}❌ USER ID must be >= 10000, found: $user_id${NC}"
            ((ERRORS++))
        fi
    fi
    
    echo ""
done

# Summary
if [[ $ERRORS -gt 0 ]]; then
    echo -e "${RED}❌ FIPS compliance check failed with $ERRORS errors${NC}"
    exit 1
elif [[ $WARNINGS -gt 0 ]]; then
    echo -e "${YELLOW}⚠️  FIPS compliance check passed with $WARNINGS warnings${NC}"
    exit 0
else
    echo -e "${GREEN}✅ All Dockerfiles are FIPS-compliant${NC}"
    exit 0
fi