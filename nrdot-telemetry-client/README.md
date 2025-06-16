# nrdot-telemetry-client

Self-instrumentation client for monitoring NRDOT-Host's own health and performance.

## Overview
Provides self-monitoring capabilities by instrumenting NRDOT components with the New Relic Go Agent and collecting health metrics about the monitoring system itself.

## Features
- New Relic Go Agent integration
- Custom event collection (NrdotHealthSample)
- Resource usage tracking
- Error and restart tracking
- Feature flag telemetry

## Metrics Collected
- Agent version and configuration
- Collector process CPU/memory usage
- Restart counts and uptime
- Configuration changes
- Feature flag states

## API
```go
type TelemetryClient interface {
    RecordHealth(sample HealthSample) error
    RecordRestart(reason string) error
    RecordConfigChange(old, new string) error
}
```

## Integration
- Used by `nrdot-ctl` and `nrdot-supervisor`
- Sends data to configured New Relic account
