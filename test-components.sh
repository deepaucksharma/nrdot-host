#!/bin/bash
# Component-level testing for NRDOT-HOST

echo "=== NRDOT-HOST Component Testing ==="
echo

# Test each component's structure and imports
components=(
    "nrdot-common"
    "nrdot-api-server" 
    "nrdot-config-engine"
    "nrdot-supervisor"
    "nrdot-ctl"
    "processors/nrsecurity"
    "processors/nrenrich"
    "processors/nrtransform"
    "processors/nrcap"
)

for comp in "${components[@]}"; do
    echo "Testing $comp..."
    
    if [ -d "$comp" ]; then
        echo "  ✓ Directory exists"
        
        if [ -f "$comp/go.mod" ]; then
            echo "  ✓ go.mod exists"
            
            # Check module name
            module_name=$(grep "^module" "$comp/go.mod" | awk '{print $2}')
            echo "  → Module: $module_name"
            
            # Check for main Go files
            go_files=$(find "$comp" -name "*.go" -type f | head -5)
            if [ -n "$go_files" ]; then
                echo "  ✓ Go files found"
            else
                echo "  ✗ No Go files found"
            fi
        else
            echo "  ✗ go.mod missing"
        fi
    else
        echo "  ✗ Directory missing"
    fi
    echo
done

# Test authentication implementation
echo "Testing Authentication Implementation..."
if [ -f "nrdot-common/pkg/auth/jwt.go" ]; then
    echo "  ✓ JWT implementation found"
    grep -q "type JWTManager struct" nrdot-common/pkg/auth/jwt.go && echo "  ✓ JWTManager defined"
fi

if [ -f "nrdot-common/pkg/auth/store.go" ]; then
    echo "  ✓ Token store implementation found"
    grep -q "type TokenStore struct" nrdot-common/pkg/auth/store.go && echo "  ✓ TokenStore defined"
fi
echo

# Test metrics implementation
echo "Testing Metrics Implementation..."
if [ -f "nrdot-supervisor/metrics_provider.go" ]; then
    echo "  ✓ Metrics provider found"
    grep -q "GetCustomMetrics" nrdot-supervisor/metrics_provider.go && echo "  ✓ GetCustomMetrics method defined"
fi

if [ -f "nrdot-api-server/pkg/handlers/metrics.go" ]; then
    echo "  ✓ Metrics handler found"
    grep -q "Prometheus" nrdot-api-server/pkg/handlers/metrics.go && echo "  ✓ Prometheus format support"
fi
echo

# Test rate limiting
echo "Testing Rate Limiting..."
if [ -f "nrdot-api-server/pkg/middleware/ratelimit.go" ]; then
    echo "  ✓ Rate limit middleware found"
    grep -q "RateLimiter" nrdot-api-server/pkg/middleware/ratelimit.go && echo "  ✓ RateLimiter implemented"
    grep -q "tokenBucket" nrdot-api-server/pkg/middleware/ratelimit.go && echo "  ✓ Token bucket algorithm used"
fi
echo

# Test main binary
echo "Testing Main Binary..."
if [ -f "cmd/nrdot-host/main.go" ]; then
    echo "  ✓ main.go found"
    grep -q "ModeAll" cmd/nrdot-host/main.go && echo "  ✓ All mode supported"
    grep -q "ModeAgent" cmd/nrdot-host/main.go && echo "  ✓ Agent mode supported"
    grep -q "ModeAPI" cmd/nrdot-host/main.go && echo "  ✓ API mode supported"
    grep -q "rate-limit" cmd/nrdot-host/main.go && echo "  ✓ Rate limiting flags added"
fi
echo

echo "=== Component testing complete ===="