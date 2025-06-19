# NRDOT-HOST API Reference

This document describes the REST API exposed by the NRDOT API Server.

## Table of Contents

- [Overview](#overview)
- [Authentication](#authentication)
- [API Endpoints](#api-endpoints)
  - [Health & Status](#health--status)
  - [Configuration](#configuration)
  - [Metrics](#metrics)
  - [Control](#control)
- [Error Handling](#error-handling)
- [Examples](#examples)

## Overview

The NRDOT API Server provides a REST API for monitoring and managing NRDOT-HOST. By default, it listens on `localhost:8080` and is restricted to local access only.

### Base URL

```
http://localhost:8080/v1
```

### Content Types

- Request: `application/json`
- Response: `application/json`

### API Versioning

The API uses URL versioning. Current version: `v1`

## Authentication

By default, the API server only accepts connections from localhost. Additional authentication can be configured:

### Token Authentication

```yaml
security:
  api_auth:
    enabled: true
    methods:
      token:
        enabled: true
        tokens:
          - name: "monitoring"
            token: "${API_TOKEN}"
            permissions: ["read"]
```

Usage:
```bash
curl -H "X-API-Token: your-token" http://localhost:8080/v1/status
```

### mTLS Authentication

```yaml
security:
  api_auth:
    methods:
      mtls:
        enabled: true
        ca_file: "/etc/nrdot/certs/ca.crt"
```

## API Endpoints

### Health & Status

#### GET /v1/health

Health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0",
  "checks": {
    "collector": "healthy",
    "config": "healthy",
    "exporters": "healthy"
  }
}
```

**Status Codes:**
- `200 OK`: All components healthy
- `503 Service Unavailable`: One or more components unhealthy

#### GET /v1/health/live

Kubernetes liveness probe endpoint.

**Response:**
```json
{
  "status": "alive",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

#### GET /v1/health/ready

Kubernetes readiness probe endpoint.

**Response:**
```json
{
  "status": "ready",
  "timestamp": "2024-01-15T10:30:00Z",
  "components": {
    "collector": true,
    "exporters": true,
    "pipelines": true
  }
}
```

#### GET /v1/status

Detailed system status.

**Response:**
```json
{
  "service": {
    "name": "my-service",
    "version": "1.0.0",
    "uptime": 3600,
    "start_time": "2024-01-15T09:30:00Z"
  },
  "collector": {
    "status": "running",
    "pid": 12345,
    "memory_mb": 256,
    "cpu_percent": 2.5
  },
  "pipelines": {
    "metrics": {
      "status": "running",
      "processors": ["nrsecurity", "nrenrich", "nrtransform", "nrcap"],
      "items_processed": 150000,
      "errors": 0
    },
    "traces": {
      "status": "running",
      "processors": ["nrsecurity", "nrenrich"],
      "items_processed": 50000,
      "errors": 0
    }
  },
  "exporters": {
    "newrelic": {
      "status": "connected",
      "endpoint": "otlp.nr-data.net",
      "last_success": "2024-01-15T10:29:55Z",
      "items_sent": 200000,
      "errors": 0
    }
  }
}
```

### Configuration

#### GET /v1/config

Get current configuration (sanitized).

**Response:**
```json
{
  "service": {
    "name": "my-service",
    "environment": "production"
  },
  "license_key": "[REDACTED]",
  "metrics": {
    "enabled": true,
    "interval": "60s"
  },
  "processors": {
    "nrsecurity": {
      "enabled": true,
      "redact_secrets": true
    },
    "nrcap": {
      "limits": {
        "global": 100000
      }
    }
  }
}
```

#### GET /v1/config/raw

Get raw configuration (requires admin permission).

**Response:**
```yaml
service:
  name: my-service
license_key: eu01xx...NRAL
# Full configuration in YAML
```

#### POST /v1/config/validate

Validate configuration without applying.

**Request:**
```json
{
  "config": {
    "service": {
      "name": "test-service"
    },
    "license_key": "eu01xx...NRAL"
  }
}
```

**Response:**
```json
{
  "valid": true,
  "errors": [],
  "warnings": [
    "metrics.interval is not specified, using default: 60s"
  ]
}
```

#### POST /v1/config/reload

Reload configuration from disk.

**Response:**
```json
{
  "success": true,
  "message": "Configuration reloaded successfully",
  "version": "2024-01-15T10:30:00Z"
}
```

#### PATCH /v1/config

Update specific configuration values.

**Request:**
```json
{
  "path": "/processors/nrcap/limits/global",
  "value": 50000
}
```

**Response:**
```json
{
  "success": true,
  "message": "Configuration updated",
  "restart_required": false
}
```

### Metrics

#### GET /v1/metrics

Get internal metrics about NRDOT operation.

**Response:**
```json
{
  "collector": {
    "uptime_seconds": 3600,
    "memory_bytes": 268435456,
    "cpu_percent": 2.5,
    "goroutines": 42
  },
  "pipelines": {
    "metrics": {
      "received": 150000,
      "processed": 150000,
      "dropped": 0,
      "errors": 0,
      "processing_time_ms": {
        "p50": 0.5,
        "p95": 2.0,
        "p99": 5.0
      }
    }
  },
  "processors": {
    "nrsecurity": {
      "items_processed": 150000,
      "secrets_redacted": 1250,
      "processing_time_ms": 0.1
    },
    "nrcap": {
      "items_processed": 150000,
      "series_dropped": 500,
      "cardinality_current": 45000,
      "cardinality_limit": 100000
    }
  },
  "exporters": {
    "newrelic": {
      "items_sent": 200000,
      "bytes_sent": 10485760,
      "errors": 0,
      "retry_count": 2,
      "queue_size": 0
    }
  }
}
```

#### GET /v1/metrics/cardinality

Get cardinality information.

**Query Parameters:**
- `top`: Number of top metrics to return (default: 10)
- `sort`: Sort by "cardinality" or "name" (default: "cardinality")

**Response:**
```json
{
  "total_series": 45000,
  "limit": 100000,
  "utilization": 0.45,
  "top_metrics": [
    {
      "name": "http.request.duration",
      "cardinality": 15000,
      "dimensions": {
        "endpoint": 100,
        "method": 4,
        "status_code": 5,
        "user_id": 750
      }
    }
  ],
  "dropped_series": 500,
  "drop_rate": 0.01
}
```

#### GET /v1/metrics/throughput

Get throughput metrics.

**Response:**
```json
{
  "metrics": {
    "rate_per_second": 2500,
    "bytes_per_second": 102400,
    "points_per_metric": 1.2
  },
  "traces": {
    "rate_per_second": 100,
    "spans_per_trace": 15.5,
    "bytes_per_second": 51200
  },
  "logs": {
    "rate_per_second": 500,
    "bytes_per_second": 204800
  }
}
```

### Auto-Configuration (Phase 2 - Coming Soon)

#### GET /v1/discovery

Run service discovery scan.

**Response:**
```json
{
  "discovery_id": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2024-01-15T10:30:00Z",
  "discovered_services": [
    {
      "type": "mysql",
      "version": "8.0.32",
      "endpoints": [{"address": "localhost", "port": 3306}],
      "confidence": "HIGH",
      "discovered_by": ["process", "port", "config_file"]
    },
    {
      "type": "nginx",
      "version": "1.22.1",
      "endpoints": [{"address": "0.0.0.0", "port": 80}],
      "confidence": "HIGH",
      "discovered_by": ["process", "port"]
    }
  ],
  "scan_duration_ms": 450,
  "errors": []
}
```

#### GET /v1/discovery/status

Get auto-configuration status.

**Response:**
```json
{
  "enabled": true,
  "last_scan": "2024-01-15T10:30:00Z",
  "next_scan": "2024-01-15T10:35:00Z",
  "active_services": ["mysql", "nginx"],
  "config_version": "2024-01-15-001",
  "config_source": "remote"
}
```

#### POST /v1/discovery/preview

Preview what configuration would be generated.

**Request:**
```json
{
  "services": ["mysql", "nginx"]
}
```

**Response:**
```json
{
  "preview": {
    "receivers": {
      "mysql": {...},
      "nginx": {...}
    },
    "pipelines": {
      "metrics": {
        "receivers": ["mysql", "nginx", "hostmetrics"]
      }
    }
  },
  "template_version": "1.0",
  "variables_required": ["MYSQL_MONITOR_USER", "MYSQL_MONITOR_PASS"]
}
```

### Control

#### POST /v1/control/restart

Restart the OpenTelemetry Collector.

**Request:**
```json
{
  "graceful": true,
  "timeout_seconds": 30
}
```

**Response:**
```json
{
  "success": true,
  "message": "Collector restart initiated",
  "pid": 12346
}
```

#### POST /v1/control/stop

Stop the collector.

**Request:**
```json
{
  "graceful": true,
  "timeout_seconds": 30
}
```

**Response:**
```json
{
  "success": true,
  "message": "Collector stop initiated"
}
```

#### POST /v1/control/debug

Enable debug mode.

**Request:**
```json
{
  "enabled": true,
  "duration_seconds": 300,
  "components": ["nrsecurity", "exporters"]
}
```

**Response:**
```json
{
  "success": true,
  "message": "Debug mode enabled for 300 seconds",
  "expires_at": "2024-01-15T10:35:00Z"
}
```

#### GET /v1/control/pipelines

Get pipeline information.

**Response:**
```json
{
  "pipelines": {
    "metrics": {
      "receivers": ["prometheus", "hostmetrics"],
      "processors": ["nrsecurity", "nrenrich", "nrtransform", "nrcap"],
      "exporters": ["newrelic"],
      "status": "running"
    },
    "traces": {
      "receivers": ["otlp"],
      "processors": ["nrsecurity", "nrenrich"],
      "exporters": ["newrelic"],
      "status": "running"
    }
  }
}
```

#### POST /v1/control/pipelines/{name}/pause

Pause a specific pipeline.

**Response:**
```json
{
  "success": true,
  "pipeline": "metrics",
  "status": "paused"
}
```

#### POST /v1/control/pipelines/{name}/resume

Resume a paused pipeline.

**Response:**
```json
{
  "success": true,
  "pipeline": "metrics",
  "status": "running"
}
```

## Error Handling

### Error Response Format

```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "The request body is invalid",
    "details": {
      "field": "config.service.name",
      "reason": "required field missing"
    }
  },
  "request_id": "12345-67890",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INVALID_REQUEST` | 400 | Invalid request format or parameters |
| `UNAUTHORIZED` | 401 | Missing or invalid authentication |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Resource not found |
| `CONFLICT` | 409 | Resource conflict (e.g., already exists) |
| `RATE_LIMITED` | 429 | Too many requests |
| `INTERNAL_ERROR` | 500 | Internal server error |
| `SERVICE_UNAVAILABLE` | 503 | Service temporarily unavailable |

### Common Error Scenarios

1. **Invalid Configuration**
   ```json
   {
     "error": {
       "code": "INVALID_REQUEST",
       "message": "Configuration validation failed",
       "details": {
         "errors": [
           "license_key is required",
           "processors.nrcap.limits.global must be positive"
         ]
       }
     }
   }
   ```

2. **Rate Limiting**
   ```json
   {
     "error": {
       "code": "RATE_LIMITED",
       "message": "Rate limit exceeded",
       "details": {
         "limit": 100,
         "window": "1m",
         "retry_after": 45
       }
     },
     "headers": {
       "X-RateLimit-Limit": "100",
       "X-RateLimit-Remaining": "0",
       "X-RateLimit-Reset": "1705317045"
     }
   }
   ```

## Examples

### Using cURL

```bash
# Check health
curl http://localhost:8080/v1/health

# Get status
curl http://localhost:8080/v1/status | jq

# Get metrics
curl http://localhost:8080/v1/metrics | jq

# Reload configuration
curl -X POST http://localhost:8080/v1/config/reload

# Enable debug mode
curl -X POST http://localhost:8080/v1/control/debug \
  -H "Content-Type: application/json" \
  -d '{"enabled": true, "duration_seconds": 300}'

# With authentication
curl -H "X-API-Token: your-token" http://localhost:8080/v1/config
```

### Using nrdot-ctl

```bash
# nrdot-ctl uses the API internally
nrdot-ctl status        # GET /v1/status
nrdot-ctl health        # GET /v1/health
nrdot-ctl config show   # GET /v1/config
nrdot-ctl config reload # POST /v1/config/reload
```

### Python Example

```python
import requests
import json

class NRDOTClient:
    def __init__(self, base_url="http://localhost:8080/v1", token=None):
        self.base_url = base_url
        self.headers = {"Content-Type": "application/json"}
        if token:
            self.headers["X-API-Token"] = token
    
    def get_status(self):
        response = requests.get(f"{self.base_url}/status", headers=self.headers)
        response.raise_for_status()
        return response.json()
    
    def reload_config(self):
        response = requests.post(f"{self.base_url}/config/reload", headers=self.headers)
        response.raise_for_status()
        return response.json()
    
    def get_metrics(self):
        response = requests.get(f"{self.base_url}/metrics", headers=self.headers)
        response.raise_for_status()
        return response.json()

# Usage
client = NRDOTClient(token="your-token")
status = client.get_status()
print(f"Collector status: {status['collector']['status']}")

metrics = client.get_metrics()
print(f"Current cardinality: {metrics['processors']['nrcap']['cardinality_current']}")
```

### Go Example

```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
)

type Client struct {
    baseURL string
    token   string
    client  *http.Client
}

func NewClient(baseURL, token string) *Client {
    return &Client{
        baseURL: baseURL,
        token:   token,
        client:  &http.Client{},
    }
}

func (c *Client) GetStatus() (map[string]interface{}, error) {
    req, err := http.NewRequest("GET", c.baseURL+"/v1/status", nil)
    if err != nil {
        return nil, err
    }
    
    if c.token != "" {
        req.Header.Set("X-API-Token", c.token)
    }
    
    resp, err := c.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return result, nil
}

// Usage
func main() {
    client := NewClient("http://localhost:8080", "your-token")
    status, err := client.GetStatus()
    if err != nil {
        panic(err)
    }
    fmt.Printf("Status: %+v\n", status)
}
```

## Rate Limiting

The API implements rate limiting to prevent abuse:

- Default: 100 requests per minute per IP
- Configurable per endpoint
- Headers indicate rate limit status

```yaml
security:
  rate_limiting:
    enabled: true
    global:
      requests_per_second: 100
    endpoints:
      "/v1/config/reload":
        requests_per_second: 1
```

## Monitoring the API

The API server exports Prometheus metrics:

```
# API request count
nrdot_api_requests_total{method="GET",endpoint="/v1/status",status="200"}

# API request duration
nrdot_api_request_duration_seconds{method="GET",endpoint="/v1/status"}

# Active connections
nrdot_api_connections_active
```

## WebSocket Support

For real-time updates, WebSocket connections are supported:

```javascript
const ws = new WebSocket('ws://localhost:8080/v1/events');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Event:', data.type, data.payload);
};

// Subscribe to specific events
ws.send(JSON.stringify({
  action: 'subscribe',
  events: ['config.changed', 'pipeline.error']
}));
```

## API Deprecation Policy

- APIs are versioned (v1, v2, etc.)
- Deprecated endpoints return `Deprecation` header
- Minimum 6-month deprecation period
- Migration guides provided

Example deprecation header:
```
Deprecation: true
Sunset: Sat, 1 Jul 2024 00:00:00 GMT
Link: <https://docs.nrdot.com/migration/v2>; rel="alternate"
```