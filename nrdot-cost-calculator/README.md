# nrdot-cost-calculator

Cost analysis and optimization tools for NRDOT-Host deployments.

## Overview
Analyzes metric cardinality and data volume to estimate costs and provide optimization recommendations.

## Features
- Cardinality analysis
- Data volume calculation
- Cost estimation
- Optimization recommendations
- What-if scenarios

## Metrics
```yaml
# Cost factors analyzed
- Metric cardinality
- Data points per minute
- Attribute dimensions
- Collection frequency
- Retention period
```

## Recommendations
- Process filtering strategies
- Collection interval optimization
- Dimension reduction
- Sampling strategies

## Usage
```bash
# Analyze current costs
nrdot-cost analyze --account=prod

# Optimization report
nrdot-cost optimize --target-reduction=40%
```

## Integration
- Works with `otel-processor-nrcap`
- Analyzes Guardian Fleet data
