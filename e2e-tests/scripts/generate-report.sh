#!/bin/bash

REPORT_DIR="reports"
SUMMARY_FILE="$REPORT_DIR/summary.json"

echo "Generating E2E test report..."

# Initialize summary
cat > "$SUMMARY_FILE" <<EOF
{
    "test_run": {
        "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
        "duration": "$SECONDS seconds"
    },
    "scenarios": {}
}
EOF

# Aggregate scenario results
for scenario in microservices kubernetes host-monitoring security-compliance high-cardinality; do
    if [ -f "$REPORT_DIR/$scenario/report.json" ]; then
        jq --arg scenario "$scenario" \
           '.scenarios[$scenario] = input' \
           "$SUMMARY_FILE" "$REPORT_DIR/$scenario/report.json" > "$SUMMARY_FILE.tmp" && \
           mv "$SUMMARY_FILE.tmp" "$SUMMARY_FILE"
    fi
done

# Generate HTML report
cat > "$REPORT_DIR/index.html" <<'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>NRDOT E2E Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .passed { color: green; }
        .failed { color: red; }
        .scenario { margin: 20px 0; padding: 10px; border: 1px solid #ddd; }
        h1, h2 { color: #333; }
        pre { background: #f4f4f4; padding: 10px; overflow-x: auto; }
    </style>
</head>
<body>
    <h1>NRDOT E2E Test Report</h1>
    <div id="content"></div>
    <script>
        fetch('summary.json')
            .then(response => response.json())
            .then(data => {
                const content = document.getElementById('content');
                content.innerHTML = `
                    <p>Test Run: ${data.test_run.timestamp}</p>
                    <p>Duration: ${data.test_run.duration}</p>
                    ${Object.entries(data.scenarios).map(([name, scenario]) => `
                        <div class="scenario">
                            <h2>${name}</h2>
                            <p class="${scenario.status}">${scenario.status.toUpperCase()}</p>
                            <pre>${JSON.stringify(scenario.tests, null, 2)}</pre>
                        </div>
                    `).join('')}
                `;
            });
    </script>
</body>
</html>
EOF

echo "Test report generated: $REPORT_DIR/index.html"
echo "Summary saved to: $SUMMARY_FILE"