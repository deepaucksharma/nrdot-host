# otel-processor-nrtransform

Transformation processor that creates convenience metrics and calculations.

## Overview
Generates additional derived metrics (prefixed with `nr.`) that make data more queryable and actionable in the New Relic platform.

## Features
- CPU percentage calculations
- Memory percentage conversion
- Rate calculations for counters
- Process CPU% from CPU time
- Disk usage percentages

## Transformations
```yaml
# Examples
system.cpu.time → nr.cpu.percent
system.memory.usage → nr.memory.percent
process.cpu.time → nr.process.cpu.percent
```

## Configuration
```yaml
processors:
  nrtransform:
    calculations:
      - cpu_percentage
      - memory_percentage
      - disk_percentage
```

## Integration
- Preserves original OTel metrics
- Adds convenience metrics for New Relic UI
