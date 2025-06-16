# nrdot-api-server

Local REST API server for NRDOT-Host management and monitoring.

## Overview
Provides a localhost-only REST API for status monitoring, configuration management, and operational control of NRDOT-Host.

## Endpoints
```yaml
GET  /v1/status          # Current system status
GET  /v1/config          # Active configuration
POST /v1/config          # Update configuration
POST /v1/reload          # Reload configuration
GET  /v1/metrics         # Prometheus metrics
GET  /v1/health          # Health check
```

## Security
- Localhost only (127.0.0.1:8089)
- No authentication (local only)
- Read-only by default

## Integration
- Started by `nrdot-ctl`
- Validates with `nrdot-schema`
- Used by `nrdot-debug-tools`
