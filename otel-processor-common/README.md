# otel-processor-common

Common utilities and base implementations for NRDOT OpenTelemetry processors.

## Overview
Shared library providing common functionality for all NRDOT custom processors, ensuring consistency and reducing code duplication.

## Features
- Base processor interfaces
- Common metric utilities
- Attribute manipulation helpers
- Testing frameworks
- Error handling patterns

## Utilities
```go
// Common interfaces
type BaseProcessor interface {
    ProcessMetrics(context.Context, pmetric.Metrics) (pmetric.Metrics, error)
    Shutdown(context.Context) error
}

// Helper functions
func RedactAttribute(attr pcommon.Value, pattern string) pcommon.Value
func CalculateRate(current, previous float64, interval time.Duration) float64
```

## Integration
- Used by all otel-processor-* repos
- Provides consistent behavior
- Shared testing utilities
