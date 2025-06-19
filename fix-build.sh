#!/bin/bash
# Fix build issues for NRDOT-HOST

echo "=== Fixing NRDOT-HOST Build Issues ==="
echo

# Remove go.work to avoid conflicts
if [ -f go.work ]; then
    echo "Removing go.work file..."
    rm go.work
fi

# Clean module cache
echo "Cleaning module cache..."
go clean -modcache 2>/dev/null || true

# Update each component's go.mod with proper replace directives
echo "Updating go.mod files with replace directives..."

# Function to add replace directives
add_replaces() {
    local module=$1
    local go_mod="$module/go.mod"
    
    if [ ! -f "$go_mod" ]; then
        return
    fi
    
    echo "  Updating $module..."
    
    # Check if replace block exists
    if ! grep -q "^replace" "$go_mod"; then
        echo "" >> "$go_mod"
        echo "replace (" >> "$go_mod"
        echo "    github.com/newrelic/nrdot-host/nrdot-api-server => ../nrdot-api-server" >> "$go_mod"
        echo "    github.com/newrelic/nrdot-host/nrdot-common => ../nrdot-common" >> "$go_mod"
        echo "    github.com/newrelic/nrdot-host/nrdot-config-engine => ../nrdot-config-engine" >> "$go_mod"
        echo "    github.com/newrelic/nrdot-host/nrdot-schema => ../nrdot-schema" >> "$go_mod"
        echo "    github.com/newrelic/nrdot-host/nrdot-telemetry-client => ../nrdot-telemetry-client" >> "$go_mod"
        echo "    github.com/newrelic/nrdot-host/nrdot-template-lib => ../nrdot-template-lib" >> "$go_mod"
        echo ")" >> "$go_mod"
    fi
}

# Add replace directives to key modules
for module in nrdot-api-server nrdot-config-engine nrdot-ctl; do
    add_replaces "$module"
done

# Download dependencies for each module
echo -e "\nDownloading dependencies..."
for module in nrdot-common nrdot-api-server nrdot-config-engine nrdot-supervisor nrdot-telemetry-client; do
    if [ -d "$module" ]; then
        echo "  $module..."
        (cd "$module" && go mod download 2>/dev/null) || true
    fi
done

echo -e "\nBuild fixes applied. Try running ./test-build.sh again."