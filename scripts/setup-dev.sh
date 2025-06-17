#!/bin/bash
# Development environment setup script for NRDOT-HOST

set -e

echo "Setting up NRDOT-HOST development environment..."

# Check for required tools
check_tool() {
    if ! command -v "$1" &> /dev/null; then
        echo "✗ $1 is not installed. Please install it first."
        exit 1
    else
        echo "✓ $1 is installed"
    fi
}

echo "Checking prerequisites..."
check_tool go
check_tool docker
check_tool docker-compose
check_tool make
check_tool git

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
REQUIRED_GO_VERSION="1.21"
if [ "$(printf '%s\n' "$REQUIRED_GO_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_GO_VERSION" ]; then
    echo "✗ Go version $REQUIRED_GO_VERSION or higher is required (found $GO_VERSION)"
    exit 1
fi
echo "✓ Go version $GO_VERSION"

# Create necessary directories
echo "Creating directories..."
mkdir -p bin dist logs tmp

# Install Go tools
echo "Installing Go development tools..."
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install golang.org/x/tools/cmd/godoc@latest
go install github.com/go-delve/delve/cmd/dlv@latest
go install github.com/cosmtrek/air@latest

# Install OpenTelemetry Collector builder
echo "Installing OpenTelemetry Collector builder..."
go install go.opentelemetry.io/collector/cmd/builder@latest

# Download Go dependencies for all components
echo "Downloading Go dependencies..."
for dir in */; do
    if [ -f "$dir/go.mod" ]; then
        echo "  Processing $dir..."
        (cd "$dir" && go mod download && go mod tidy)
    fi
done

# Setup git hooks
if [ -d .git ]; then
    echo "Setting up git hooks..."
    cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
# Pre-commit hook for NRDOT-HOST

# Run linting
echo "Running linters..."
make lint || exit 1

# Run tests for changed components
for component in $(git diff --cached --name-only | grep -E "^[^/]+/" | cut -d/ -f1 | sort -u); do
    if [ -f "$component/go.mod" ]; then
        echo "Testing $component..."
        make test-$component || exit 1
    fi
done

echo "Pre-commit checks passed!"
EOF
    chmod +x .git/hooks/pre-commit
fi

# Create local configuration
if [ ! -f .env ]; then
    echo "Creating local environment configuration..."
    cat > .env << EOF
# NRDOT-HOST Development Environment
NRDOT_ENV=development
NRDOT_LOG_LEVEL=debug
NRDOT_API_PORT=8089
NRDOT_COLLECTOR_PORT=4317
NRDOT_PROMETHEUS_PORT=9090
EOF
fi

# Pull required Docker images
echo "Pulling Docker images..."
docker pull otel/opentelemetry-collector-contrib:latest
docker pull prom/prometheus:latest
docker pull grafana/grafana:latest
docker pull jaegertracing/all-in-one:latest

echo ""
echo "Development environment setup complete!"
echo ""
echo "Quick start:"
echo "  make build         # Build all components"
echo "  make test          # Run all tests"
echo "  make dev           # Start development environment"
echo "  make help          # Show all available commands"
echo ""
echo "Happy coding! 🚀"