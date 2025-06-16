# nrdot-ctl

Main control binary for NRDOT-Host - The New Relic Distribution of OpenTelemetry for Host Monitoring.

## Overview
`nrdot-ctl` is the core control plane that manages the lifecycle of the OpenTelemetry Collector, provides configuration management, and ensures reliable operation of the NRDOT-Host monitoring solution.

## Features
- Process lifecycle management for OTel Collector
- Configuration generation from simple YAML
- Health monitoring and automatic recovery
- Self-instrumentation with New Relic Go Agent
- CLI interface for management operations

## Dependencies
- `nrdot-config-engine`: Configuration processing
- `nrdot-supervisor`: Process supervision
- `nrdot-telemetry-client`: Self-monitoring
- `nrdot-template-lib`: OTel config templates

## API
```go
// Main commands
nrdot-ctl start    // Start the monitoring system
nrdot-ctl stop     // Stop the monitoring system  
nrdot-ctl status   // Check current status
nrdot-ctl validate // Validate configuration
nrdot-ctl reload   // Reload configuration
```

## Configuration
```yaml
# /etc/nrdot/nrdot-host.yml
license_key: YOUR_NR_LICENSE_KEY
custom_attributes:
  environment: production
  role: webserver
```

## Integration Points
- Integrates with `nrdot-api-server` for REST API
- Uses `nrdot-remote-config` for feature flags
- Leverages `nrdot-schema` for validation

## Build
```bash
make build
```

## Testing
```bash
make test
```
