# otel-processor-nrenrich

Enrichment processor that adds New Relic-specific metadata to telemetry data.

## Overview
Enriches all telemetry with New Relic entity information, agent metadata, and cloud provider context for seamless platform integration.

## Features
- Entity GUID generation
- Agent name/version injection
- Cloud provider metadata detection
- Host identity enrichment
- Service correlation attributes

## Enrichment Fields
```yaml
# Added to all telemetry
entity.guid: <generated-guid>
agent.name: nrdot-host
agent.version: <version>
instrumentation.provider: newrelic
```

## Configuration
```yaml
processors:
  nrenrich:
    entity_synthesis: true
    cloud_detection: auto
```

## Integration
- Works with New Relic's entity synthesis
- Enables APM/Infra correlation
