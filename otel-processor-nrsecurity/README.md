# NR Security Processor

The NR Security Processor is an OpenTelemetry processor that redacts sensitive information from telemetry data (traces, metrics, and logs).

## Features

- **Pattern-based redaction**: Uses regex patterns to detect and redact secrets
- **Keyword-based redaction**: Redacts values based on attribute names containing keywords
- **Nested attribute support**: Handles nested attributes in telemetry data
- **Configurable replacement**: Customize the redacted text replacement
- **Performance optimized**: Caches compiled regex patterns
- **Allow/Deny lists**: Control which attributes to process

## Default Redaction Patterns

The processor includes built-in patterns for common sensitive data:

- API keys (various formats)
- Passwords in URLs and connection strings
- JWT tokens
- Credit card numbers
- Social security numbers
- Email addresses (optional)
- IP addresses (optional)

## Configuration

```yaml
processors:
  nrsecurity:
    # Enable/disable the processor
    enabled: true
    
    # Replacement text for redacted values
    replacement_text: "[REDACTED]"
    
    # Redact email addresses
    redact_emails: false
    
    # Redact IP addresses
    redact_ips: false
    
    # Additional regex patterns for redaction
    patterns:
      - name: "custom_api_key"
        regex: "custom-key-[a-zA-Z0-9]{32}"
        
    # Keywords in attribute names that trigger redaction
    keywords:
      - password
      - secret
      - token
      - key
      - credential
      - auth
      
    # Attributes to always allow (never redact)
    allow_list:
      - "service.name"
      - "span.kind"
      
    # Attributes to always deny (always redact)
    deny_list:
      - "http.request.header.authorization"
      - "db.connection_string"
```

## Usage

Add the processor to your OpenTelemetry Collector pipeline:

```yaml
service:
  pipelines:
    traces:
      processors: [nrsecurity]
    metrics:
      processors: [nrsecurity]
    logs:
      processors: [nrsecurity]
```

## Performance Considerations

- Regex patterns are compiled once and cached
- Processing is done in-place when possible
- Only attribute values are processed, not keys or telemetry structure

## Building

```bash
make build
```

## Testing

```bash
make test
```

## Benchmarks

```bash
make bench
```