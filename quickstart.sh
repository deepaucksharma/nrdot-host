#!/bin/bash
# NRDOT-Host Quick Start Script
# Demonstrates component integration

set -e

echo "🚀 NRDOT-Host Quick Start"
echo "========================="

# Check prerequisites
echo "📋 Checking prerequisites..."
command -v go >/dev/null 2>&1 || { echo "❌ Go is required but not installed."; exit 1; }
command -v docker >/dev/null 2>&1 || { echo "❌ Docker is required but not installed."; exit 1; }

echo "✅ Prerequisites satisfied"

# Create example configuration
echo "📝 Creating example configuration..."
mkdir -p examples
cat > examples/local-config.yml <<EOF
# Example NRDOT-Host Configuration
license_key: YOUR_NEW_RELIC_LICENSE_KEY_HERE

# Custom attributes for this host
custom_attributes:
  environment: development
  team: platform
  role: demo

# Enable process monitoring
process_monitoring:
  enabled: true

# Logging configuration  
logging:
  level: info
EOF

echo "✅ Configuration created at examples/local-config.yml"

# Show component structure
echo ""
echo "📦 Component Structure:"
echo "Core Control Plane:"
ls -d nrdot-ctl nrdot-config-engine nrdot-supervisor nrdot-telemetry-client nrdot-template-lib 2>/dev/null | sed 's/^/  - /'

echo ""
echo "OTel Processors:"
ls -d otel-processor-* nrdot-privileged-helper 2>/dev/null | sed 's/^/  - /'

echo ""
echo "🔗 Integration Flow:"
echo "1. User Config → nrdot-config-engine → OTel Config"
echo "2. nrdot-ctl → nrdot-supervisor → OTel Collector"
echo "3. Processors: Security → Enrichment → Transform → Export"
echo "4. Self-monitoring via nrdot-telemetry-client"

echo ""
echo "🏗️ To build all components:"
echo "  make all"

echo ""
echo "🧪 To run tests:"
echo "  make test"

echo ""
echo "📦 To create packages:"
echo "  make package"

echo ""
echo "🐳 To build Docker images:"
echo "  make docker"

echo ""
echo "✨ Ready to start developing NRDOT-Host!"
echo "   Edit examples/local-config.yml with your license key"
echo "   Then run: make run-local"