# otel-processor-nrsecurity

Security processor for automatic secret redaction and data protection in OpenTelemetry pipelines.

## Overview
Custom OpenTelemetry processor that automatically redacts sensitive information from telemetry data, ensuring no secrets or PII leak through monitoring.

## Features
- Automatic secret pattern detection
- Command-line argument redaction
- API key/token removal
- Configurable redaction rules
- Audit logging
- Future: Non-root process collection support

## Redaction Patterns
- API Keys: 32-64 char hex strings
- Passwords: Various formats
- AWS credentials
- JWT tokens
- Custom patterns via config

## Configuration
```yaml
processors:
  nrsecurity:
    redact_command_lines: true
    audit_log: /var/log/nrdot/security.log
```

## Integration
- Plugs into OTel Collector pipelines
- First processor in pipeline for security
