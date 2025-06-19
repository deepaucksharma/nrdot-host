#!/bin/bash
# Build test script for NRDOT-HOST

echo "=== NRDOT-HOST Build Test ==="
echo

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Test building individual components
echo "Testing component builds..."

components=(
    "nrdot-common"
    "nrdot-api-server"
    "nrdot-config-engine"
    "nrdot-ctl"
    "nrdot-telemetry-client"
)

for comp in "${components[@]}"; do
    echo -n "Building $comp... "
    if (cd "$comp" && go build ./... 2>/dev/null); then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗${NC}"
        echo "  Error: Failed to build $comp"
    fi
done

# Test the main binary
echo -e "\nTesting main binary build..."
echo -n "Building cmd/nrdot-host... "
if (cd cmd/nrdot-host && go build -o ../../build/bin/nrdot-host . 2>/dev/null); then
    echo -e "${GREEN}✓${NC}"
    echo "  Binary created at: build/bin/nrdot-host"
else
    echo -e "${RED}✗${NC}"
    echo "  Error: Failed to build main binary"
fi

# Check if binary was created
if [ -f "build/bin/nrdot-host" ]; then
    echo -e "\n${GREEN}Build successful!${NC}"
    echo "Binary location: $(pwd)/build/bin/nrdot-host"
    
    # Show help
    echo -e "\nTesting binary help output:"
    ./build/bin/nrdot-host -h 2>/dev/null || echo "  (Binary execution test failed)"
else
    echo -e "\n${RED}Build failed!${NC}"
fi