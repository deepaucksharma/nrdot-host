# nrdot-template-lib

OpenTelemetry configuration template library for NRDOT-Host.

## Overview
Provides versioned OTel configuration templates that transform simple user configurations into full OpenTelemetry Collector configurations.

## Features
- Version-specific OTel config templates
- Template variable substitution
- Pipeline generation logic
- Receiver/processor/exporter configuration
- Backwards compatibility support

## Template Structure
```yaml
# Templates follow OTel Collector schema
receivers:
  hostmetrics:
    collection_interval: {{ .Interval }}
processors:
  {{- if .Security.Enabled }}
  nrsecurity: {}
  {{- end }}
exporters:
  otlp:
    endpoint: {{ .Endpoint }}
```

## Integration
- Used by `nrdot-config-engine` for rendering
- Templates validated against `nrdot-schema`
