# otel-processor-nrcap

Cardinality protection processor for preventing metric explosions.

## Overview
Monitors and controls metric cardinality to prevent cost overruns and performance issues from high-cardinality data.

## Features
- Real-time cardinality tracking
- Configurable limits per metric
- Automatic dimension reduction
- Alerting on threshold breach
- Adaptive sampling

## Configuration
```yaml
processors:
  nrcap:
    limits:
      process.cpu.time: 1000
      custom.metric: 5000
    actions:
      drop: false
      aggregate: true
      alert: true
```

## Protection Strategies
- Drop high-cardinality series
- Aggregate to lower dimensions
- Sample probabilistically

## Integration
- Works with `nrdot-cost-calculator`
- Alerts via `nrdot-telemetry-client`
