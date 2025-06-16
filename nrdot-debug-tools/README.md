# nrdot-debug-tools

Diagnostic and debugging utilities for NRDOT-Host troubleshooting.

## Overview
Collection of tools for diagnosing issues, analyzing performance, and debugging NRDOT deployments.

## Tools
- **nrdot-diag**: System diagnostics
- **nrdot-trace**: Request tracing
- **nrdot-metrics**: Metric validation
- **nrdot-logs**: Log analysis
- **nrdot-config-test**: Config testing

## Features
- Health check validation
- Configuration analysis
- Performance profiling
- Log correlation
- Metric verification

## Usage
```bash
# Run full diagnostics
nrdot-diag --full

# Trace metric flow
nrdot-trace --metric="system.cpu.utilization"

# Analyze logs
nrdot-logs analyze --since=1h
```

## Integration
- Connects to `nrdot-api-server`
- Analyzes `nrdot-telemetry-client` data
