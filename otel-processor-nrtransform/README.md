# OpenTelemetry NRTransform Processor

The NRTransform processor performs metric calculations and transformations in the OpenTelemetry Collector pipeline.

## Features

- **Metric Aggregations**: Sum, average, min, max, count across dimensions
- **Rate Calculations**: Convert cumulative metrics to rates
- **Delta Calculations**: Convert cumulative metrics to deltas
- **Unit Conversions**: Convert between units (bytes to MB, ms to seconds, etc.)
- **Metric Combining**: Create new metrics from multiple existing ones
- **Filtering and Renaming**: Filter metrics by conditions and rename them
- **Label Manipulation**: Extract and manipulate metric labels
- **Histogram Adjustments**: Modify histogram bucket boundaries
- **Summary Calculations**: Calculate percentiles from summaries

## Configuration

```yaml
processors:
  nrtransform:
    transformations:
      - type: aggregate
        metric_name: "http.request.duration"
        aggregation: "avg"
        group_by: ["service", "endpoint"]
        output_metric: "http.request.duration.avg"
      
      - type: calculate_rate
        metric_name: "http.requests.total"
        output_metric: "http.requests.rate"
      
      - type: convert_unit
        metric_name: "memory.usage"
        from_unit: "bytes"
        to_unit: "megabytes"
        output_metric: "memory.usage.mb"
      
      - type: combine
        expression: "cpu.user + cpu.system"
        output_metric: "cpu.total"
        metrics:
          - "cpu.user"
          - "cpu.system"
```

## Transformation Types

### Aggregate
Aggregate metrics across dimensions using various functions.

### Calculate Rate
Convert cumulative metrics to per-second rates.

### Calculate Delta
Convert cumulative metrics to deltas between observations.

### Convert Unit
Convert metric values between different units.

### Combine
Create new metrics by combining existing ones using expressions.

### Rename
Rename metrics while preserving their data.

### Filter
Filter metrics based on conditions.

### Extract Label
Extract label values into new metrics.

## Building

```bash
make build
```

## Testing

```bash
make test
```

## License

Apache License 2.0
EOF < /dev/null
