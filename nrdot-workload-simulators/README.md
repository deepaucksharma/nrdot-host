# nrdot-workload-simulators

Collection of workload simulators for testing NRDOT-Host under various conditions.

## Overview
Provides realistic workload simulation tools to test monitoring capabilities, performance impact, and edge cases.

## Simulators
- **process-spawner**: Creates process churn
- **secret-emitter**: Tests secret redaction
- **cpu-stress**: CPU load generation
- **memory-hog**: Memory pressure testing
- **disk-io**: I/O pattern simulation

## Usage
```python
# Example: Process churn simulation
simulator = ProcessSpawner(
    min_processes=100,
    max_processes=1000,
    lifetime_seconds=(1, 60)
)
simulator.run()
```

## Integration
- Deployed by `guardian-fleet-infra`
- Results analyzed by `nrdot-benchmark-suite`
- Validates processor functionality
