# nrdot-fleet-protocol

Fleet management protocol and coordination for NRDOT-Host deployments.

## Overview
Defines the protocol for managing large-scale NRDOT deployments, including coordinated updates, health reporting, and configuration distribution.

## Protocol Features
- Fleet-wide health aggregation
- Coordinated rolling updates
- Configuration broadcast
- Version management
- Canary deployments

## Message Types
```protobuf
message FleetCommand {
  enum Type {
    UPDATE_CONFIG = 0;
    UPGRADE_VERSION = 1;
    HEALTH_CHECK = 2;
  }
}
```

## Integration
- Implemented by fleet management console
- Used by `nrdot-ctl` for fleet operations
- Enables `guardian-fleet-infra` coordination
