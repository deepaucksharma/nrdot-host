# nrdot-migrate

Migration tools for transitioning to NRDOT-Host from other monitoring solutions.

## Overview
Provides automated migration paths from New Relic Infrastructure agent, vanilla OpenTelemetry, and other monitoring tools.

## Migration Paths
- New Relic Agent → NRDOT
- Vanilla OTel → NRDOT
- Prometheus Node Exporter → NRDOT
- Mixed environments → NRDOT

## Features
- Configuration conversion
- Compatibility validation
- Parallel running mode
- Data comparison tools
- Rollback support

## Usage
```bash
# Analyze current setup
nrdot-migrate analyze

# Convert configuration
nrdot-migrate convert --from=nri --to=nrdot

# Validate migration
nrdot-migrate validate
```

## Integration
- Reads `nrdot-schema` for validation
- Outputs `nrdot-config-engine` compatible configs
