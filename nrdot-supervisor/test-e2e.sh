#\!/bin/bash
# End-to-end test script for NRDOT-HOST

set -e

echo "=== NRDOT-HOST End-to-End Test ==="
echo

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
pass() {
    echo "✓ $1"
    ((TESTS_PASSED++))
}

fail() {
    echo "✗ $1"
    ((TESTS_FAILED++))
}

# Test 1: Check directory structure
echo "Test 1: Verify directory structure"
if [ -d "processors" ] && [ -d "deployments" ] && [ -d "tests" ] && [ -d "docs" ]; then
    pass "Directory structure is correct"
else
    fail "Directory structure is incorrect"
fi

# Test 2: Check processor modules
echo -e "\nTest 2: Verify processor modules"
for proc in nrsecurity nrenrich nrtransform nrcap common; do
    if [ -f "processors/$proc/go.mod" ]; then
        pass "Found go.mod for processor $proc"
    else
        fail "Missing go.mod for processor $proc"
    fi
done

# Test 3: Check deployment files
echo -e "\nTest 3: Verify deployment configurations"
if [ -f "deployments/docker/unified/Dockerfile" ]; then
    pass "Found unified Dockerfile"
else
    fail "Missing unified Dockerfile"
fi

# Test 4: Check documentation
echo -e "\nTest 4: Verify documentation"
if [ -f "docs/DIRECTORY_STRUCTURE.md" ]; then
    pass "Found directory structure documentation"
else
    fail "Missing directory structure documentation"
fi

# Summary
echo -e "\n=== Test Summary ==="
echo "Tests passed: $TESTS_PASSED"
echo "Tests failed: $TESTS_FAILED"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "\nAll tests passed\!"
    exit 0
else
    echo -e "\nSome tests failed\!"
    exit 1
fi
EOF < /dev/null
