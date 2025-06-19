# NRDOT OpenTelemetry Processors

This directory contains custom OpenTelemetry processors for NRDOT-HOST.

## Processors

### nrsecurity
Security processor that adds security context and validates telemetry data for compliance.

### nrenrich  
Enrichment processor that adds host metadata, cloud provider information, and custom attributes.

### nrtransform
Transform processor that performs data transformations, unit conversions, and field mappings.

### nrcap
CAP (Collection and Processing) processor that handles data sampling, filtering, and aggregation.

### common
Shared code and utilities used by all processors.

## Building

Each processor can be built individually:

```bash
cd nrsecurity
go build ./...
```

Or build all processors from the root:

```bash
make build
```

## Testing

Run tests for a specific processor:

```bash
cd nrenrich
go test -v ./...
```

Or test all processors:

```bash
make test
```

## Integration

These processors are integrated into the custom OpenTelemetry Collector build using the OpenTelemetry Collector Builder (otelcol-builder).

## Configuration

Each processor has its own configuration schema. See individual processor directories for configuration examples.