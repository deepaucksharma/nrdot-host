# nrdot-supervisor

Process supervision and health management for the OpenTelemetry Collector.

## Overview
Implements a robust process supervisor with health checking, automatic restart policies, and graceful shutdown handling for the OTel Collector process.

## Features
- State machine-based process management
- Exponential backoff restart (1s → 2s → 4s → ... → 5m cap)
- Health endpoint monitoring
- Resource limit enforcement
- Graceful shutdown coordination

## API
```go
type ProcessSupervisor interface {
    Start(ctx context.Context) error
    Stop() error
    Health() HealthStatus
    Restart() error
    GetState() SupervisorState
}
```

## Integration
- Used by `nrdot-ctl` for collector lifecycle
- Reports health to `nrdot-telemetry-client`
- Reads configs from `nrdot-config-engine`
