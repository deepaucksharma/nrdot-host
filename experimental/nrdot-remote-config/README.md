# nrdot-remote-config

Remote configuration and feature flag management for NRDOT-Host.

## Overview
Enables dynamic configuration updates and A/B testing through centralized feature flag management.

## Features
- Feature flag synchronization
- Gradual rollout support
- Configuration hot-reload
- Offline fallback
- Rollout percentage control

## Feature Flags
```yaml
# Example flags
process_monitoring_enabled: true
transform_processor_enabled: false
non_root_process_enabled: false
rollout_percentages:
  new_feature: 10  # 10% rollout
```

## API
```go
type RemoteConfig interface {
    GetFlags(hostID string) (*FeatureFlags, error)
    Subscribe(updates chan<- ConfigUpdate)
}
```

## Integration
- Used by `nrdot-ctl` for dynamic config
- Enables Guardian Fleet A/B testing
