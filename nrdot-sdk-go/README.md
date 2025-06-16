# nrdot-sdk-go

Go SDK for extending NRDOT-Host with custom processors and plugins.

## Overview
Software Development Kit for building custom extensions, processors, and integrations for NRDOT-Host.

## Features
- Processor development framework
- Plugin interfaces
- Testing utilities
- Code generation tools
- Example implementations

## API
```go
import "github.com/newrelic/nrdot-sdk-go"

// Create custom processor
type MyProcessor struct {
    nrdot.BaseProcessor
}

func (p *MyProcessor) ProcessMetrics(ctx context.Context, metrics pmetric.Metrics) (pmetric.Metrics, error) {
    // Custom logic
}
```

## Components
- Base processor interfaces
- Helper utilities
- Testing framework
- Documentation generator

## Integration
- Extends `otel-processor-common`
- Compatible with OTel Collector
