# nrdot-schema

Configuration schemas and validation rules for NRDOT-Host.

## Overview
Defines and validates all configuration schemas using JSON Schema, ensuring configuration correctness and providing IDE support.

## Schemas
- `nrdot-host.yml` user configuration
- OTel Collector configuration
- API request/response formats
- Feature flag definitions

## Example Schema
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["license_key"],
  "properties": {
    "license_key": {
      "type": "string",
      "pattern": "^[a-f0-9]{40}$"
    }
  }
}
```

## Integration
- Used by `nrdot-config-engine` for validation
- Referenced by `nrdot-api-server` for API validation
- Enables IDE autocomplete
