# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

NRDOT-HOST is an enterprise-grade OpenTelemetry distribution for host monitoring. Starting with v2.0, it uses a **unified architecture** where all components run in a single binary, replacing the previous microservices design.

## Architecture v2.0 (Current)

### Unified Binary Design
- **Single Process**: All components (supervisor, config engine, API server) run in one process
- **Direct Communication**: Components use direct function calls, no IPC
- **Mode Selection**: Can run as `--mode=all` (default), `--mode=agent`, or `--mode=api`
- **Resource Efficient**: 40% less memory, 60% less CPU than v1.0

### Core Components
1. **nrdot-common**: Shared interfaces and data models
2. **nrdot-supervisor**: Unified supervisor with embedded API server
3. **nrdot-config-engine**: Consolidated config validation and generation
4. **Custom Processors**: nrsecurity, nrenrich, nrtransform, nrcap

## Essential Commands

### Building
```bash
# Build unified binary
cd cmd/nrdot-host
make build

# Build all components (legacy)
make all
```

### Running
```bash
# Run unified binary (all components)
./bin/nrdot-host --mode=all

# Run minimal agent (no API)
./bin/nrdot-host --mode=agent

# Run with debug logging
./bin/nrdot-host --mode=all --log-level=debug
```

### Testing
```bash
make test             # Run all unit tests
make test-integration # Run integration tests
make lint            # Run linters
```

## Key Design Principles

1. **Simplicity First**: Single binary deployment over complex orchestration
2. **Direct Integration**: Function calls over IPC
3. **Zero-Config**: Works with minimal configuration
4. **Enterprise Ready**: Security, reliability, performance built-in

## Component Integration

### Configuration Flow
```
User YAML → ConfigEngine.Validate() → ConfigEngine.Generate() → Supervisor.Apply()
```

### Reload Strategy
The system uses blue-green reload by default:
1. Start new collector with new config
2. Verify health
3. Switch traffic
4. Stop old collector

## Important Files

- `cmd/nrdot-host/main.go`: Unified binary entry point
- `nrdot-supervisor/supervisor_v2.go`: Unified supervisor implementation
- `nrdot-config-engine/engine_v2.go`: Consolidated config engine
- `nrdot-common/`: Shared types and interfaces
- `ARCHITECTURE_V2.md`: Detailed architecture documentation

## Development Best Practices

1. **Use Common Types**: Always use types from `nrdot-common` for inter-component data
2. **Direct Calls**: Prefer direct function calls over any form of IPC
3. **Single Telemetry**: Use the unified telemetry client in supervisor
4. **Mode Awareness**: Code should work in all operating modes

## Testing Strategy

- Unit tests: In each component's `*_test.go` files
- Integration tests: Test unified binary with all modes
- Performance tests: Verify memory/CPU improvements

## Migration Notes

This is a first release, so no migration is needed. The v2.0 architecture is the recommended approach for all new deployments.