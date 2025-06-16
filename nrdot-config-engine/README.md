# nrdot-config-engine

Configuration management engine for NRDOT-Host with layered configuration support.

## Overview
Handles configuration parsing, validation, merging, and rendering for the NRDOT system. Implements a layered configuration model with defaults, user config, remote config, and environment variables.

## Features
- Layered configuration (defaults → user → remote → env)
- YAML parsing and validation
- Configuration hot-reload support
- Template variable substitution
- Schema-based validation

## API
```go
type ConfigEngine interface {
    Load(path string) error
    Merge(configs ...Config) Config
    Render() ([]byte, error)
    Validate() error
    Watch() <-chan ConfigUpdate
}
```

## Dependencies
- `nrdot-schema`: Configuration schemas
- `nrdot-template-lib`: Template processing

## Integration
Used by `nrdot-ctl` for configuration management and `nrdot-remote-config` for dynamic updates.
