#!/bin/bash
# Simple test runner for NRDOT-HOST

echo "================================="
echo "NRDOT-HOST Test Suite"
echo "================================="
echo

# Test 1: Check project structure
echo "Test 1: Project Structure"
echo "------------------------"
dirs=(
    "otel-processor-common"
    "otel-processor-nrsecurity" 
    "otel-processor-nrenrich"
    "otel-processor-nrtransform"
    "otel-processor-nrcap"
    "nrdot-ctl"
    "nrdot-config-engine"
    "nrdot-supervisor"
    "nrdot-api-server"
    "nrdot-telemetry-client"
    "nrdot-privileged-helper"
    "nrdot-schema"
    "nrdot-template-lib"
    "docs"
    "examples"
    "kubernetes"
    "docker"
    "systemd"
)

found=0
missing=0
for dir in "${dirs[@]}"; do
    if [ -d "$dir" ]; then
        echo "  ✓ Found: $dir"
        ((found++))
    else
        echo "  ✗ Missing: $dir"
        ((missing++))
    fi
done
echo "  Total: $found found, $missing missing"
echo

# Test 2: Check documentation
echo "Test 2: Documentation"
echo "--------------------"
docs=(
    "README.md"
    "CONTRIBUTING.md"
    "SECURITY.md"
    "LICENSE"
    "CLAUDE.md"
    "docs/installation.md"
    "docs/configuration.md"
    "docs/deployment.md"
    "docs/troubleshooting.md"
    "docs/processors.md"
    "docs/api.md"
    "docs/development.md"
    "docs/performance.md"
)

doc_found=0
doc_missing=0
for doc in "${docs[@]}"; do
    if [ -f "$doc" ]; then
        ((doc_found++))
    else
        ((doc_missing++))
    fi
done
echo "  Documentation: $doc_found found, $doc_missing missing"
echo

# Test 3: Check Go modules
echo "Test 3: Go Modules"
echo "-----------------"
modules=(
    "otel-processor-common"
    "otel-processor-nrsecurity"
    "otel-processor-nrenrich"
    "otel-processor-nrtransform"
    "otel-processor-nrcap"
    "nrdot-ctl"
    "nrdot-config-engine"
    "nrdot-supervisor"
    "nrdot-api-server"
    "nrdot-telemetry-client"
    "nrdot-privileged-helper"
    "nrdot-schema"
    "nrdot-template-lib"
)

mod_found=0
mod_missing=0
for module in "${modules[@]}"; do
    if [ -f "$module/go.mod" ]; then
        ((mod_found++))
    else
        ((mod_missing++))
    fi
done
echo "  Go modules: $mod_found found, $mod_missing missing"
echo

# Test 4: Run processor tests (if Go is available)
echo "Test 4: Component Tests"
echo "----------------------"
if command -v go &> /dev/null; then
    # Test otel-processor-common
    if [ -d "otel-processor-common" ]; then
        echo "  Testing otel-processor-common..."
        cd otel-processor-common
        if go test -short ./... &> /dev/null; then
            echo "    ✓ Tests passed"
        else
            echo "    ✗ Tests failed"
        fi
        cd ..
    fi
else
    echo "  Go not available - skipping component tests"
fi
echo

# Test 5: Demo scripts
echo "Test 5: Demo Scripts"
echo "-------------------"
if [ -f "demo-simple.sh" ]; then
    echo "  ✓ demo-simple.sh exists"
    if [ -x "demo-simple.sh" ]; then
        echo "  ✓ demo-simple.sh is executable"
    else
        echo "  ✗ demo-simple.sh is not executable"
    fi
else
    echo "  ✗ demo-simple.sh not found"
fi
echo

# Test 6: Configuration examples
echo "Test 6: Configuration Examples"
echo "-----------------------------"
configs=(
    "examples/basic/nrdot-config.yaml"
    "examples/kubernetes/nrdot-config.yaml"
    "examples/high-performance/nrdot-config.yaml"
    "examples/security-focused/nrdot-config.yaml"
)

config_found=0
for config in "${configs[@]}"; do
    if [ -f "$config" ]; then
        ((config_found++))
    fi
done
echo "  Example configs: $config_found/${#configs[@]} found"
echo

# Summary
echo "================================="
echo "Test Summary"
echo "================================="
echo "Project structure: $found/${#dirs[@]} components found"
echo "Documentation: $doc_found/${#docs[@]} files found"
echo "Go modules: $mod_found/${#modules[@]} configured"
echo "Example configs: $config_found/${#configs[@]} available"
echo

if [ $missing -eq 0 ] && [ $doc_missing -eq 0 ] && [ $mod_missing -eq 0 ]; then
    echo "Status: ✓ All core components present"
    exit 0
else
    echo "Status: ✗ Some components missing"
    exit 1
fi