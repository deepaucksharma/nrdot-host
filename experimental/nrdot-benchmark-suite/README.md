# nrdot-benchmark-suite

Performance benchmarking and comparison tools for NRDOT-Host.

## Overview
Comprehensive benchmarking suite for measuring NRDOT performance against vanilla OTel and New Relic Infrastructure agent.

## Benchmarks
- CPU overhead comparison
- Memory usage analysis
- Startup time measurement
- Data fidelity validation
- Cardinality handling

## Metrics
```yaml
# Key performance indicators
- Agent CPU usage %
- Memory RSS (MB)
- Metric collection latency
- Data completeness score
- Cost per metric
```

## Usage
```bash
# Run full benchmark
nrdot-benchmark compare --baseline=nria --challenger=nrdot
```

## Integration
- Uses `guardian-fleet-infra` data
- Publishes to KPI dashboard
