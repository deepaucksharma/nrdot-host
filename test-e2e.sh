#\!/bin/bash
# Simple e2e test for NRDOT-HOST

echo "=== NRDOT-HOST End-to-End Test ==="
echo

# Test 1: Version output
echo "Test 1: Version output"
./build/bin/nrdot-host --mode=version
echo

# Test 2: Help output
echo "Test 2: Help output (truncated)"
./build/bin/nrdot-host --help 2>&1  < /dev/null |  head -5
echo

# Test 3: API mode dry run (should fail without collector)
echo "Test 3: API mode dry run"
timeout 2s ./build/bin/nrdot-host --mode=api --log-format=json 2>&1 | head -5 || true
echo

# Test 4: Config validation
echo "Test 4: Config validation"
./build/bin/nrdot-host --mode=agent --config=test-config.yaml --log-format=json --collector=/bin/false 2>&1 | head -5 || true
echo

echo "=== Tests Complete ==="

