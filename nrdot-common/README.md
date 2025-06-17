# nrdot-common

Common data structures, interfaces, and utilities shared across all NRDOT-HOST components.

## Overview

This module provides the foundation for inter-component communication and consistency across the NRDOT-HOST system. It defines:

- Core interfaces (StatusProvider, ConfigProvider, HealthProvider)
- Shared data models (CollectorStatus, ConfigUpdate, ErrorInfo)
- Common utilities (logging, metrics, errors)
- Serialization/deserialization helpers

## Purpose

Previously, each component defined its own data structures, leading to:
- Duplication and drift between components
- Difficult inter-component communication
- Increased maintenance burden

This module solves these issues by providing a single source of truth for all shared types.

## Package Structure

```
nrdot-common/
├── pkg/
│   ├── interfaces/      # Core provider interfaces
│   ├── models/          # Shared data structures
│   ├── errors/          # Common error types
│   └── utils/           # Shared utilities
├── internal/            # Internal implementation details
└── cmd/                 # Example usage (if needed)
```

## Usage

```go
import (
    "github.com/newrelic/nrdot-host/nrdot-common/pkg/interfaces"
    "github.com/newrelic/nrdot-host/nrdot-common/pkg/models"
)

// Implement a status provider
type MySupervisor struct {
    status models.CollectorStatus
}

func (s *MySupervisor) GetStatus() (*models.CollectorStatus, error) {
    return &s.status, nil
}
```

## Design Principles

1. **Minimal Dependencies**: Only essential external dependencies
2. **Backward Compatibility**: Changes must not break existing components
3. **Clear Ownership**: Each type has a clear purpose and owner
4. **Testability**: All types are easily testable with no side effects

## Migration Guide

When migrating existing components to use nrdot-common:

1. Replace local type definitions with common types
2. Implement required interfaces
3. Update serialization to use common helpers
4. Run integration tests to verify compatibility