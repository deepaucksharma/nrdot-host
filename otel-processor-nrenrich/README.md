# NR Enrich Processor

The NR Enrich processor is an OpenTelemetry processor that enriches telemetry data (traces, metrics, and logs) with additional metadata from various sources.

## Features

- **Environment Detection**: Automatically detects and adds metadata from cloud providers (AWS, GCP, Azure) and Kubernetes
- **Process Metadata**: Enriches telemetry with process information via the privileged helper
- **Static Attributes**: Adds user-defined static attributes to all telemetry
- **Dynamic Enrichment**: Computes attributes based on telemetry content
- **Conditional Rules**: Applies enrichment based on existing attributes
- **Resource Enrichment**: Enhances resource attributes with system information

## Configuration

```yaml
processors:
  nrenrich:
    # Static attributes to add to all telemetry
    static_attributes:
      environment: production
      team: platform
    
    # Environment metadata collection
    environment:
      enabled: true
      hostname: true
      cloud_provider: true
      kubernetes: true
    
    # Process metadata via privileged helper
    process:
      enabled: true
      helper_endpoint: "unix:///var/run/nrdot/helper.sock"
    
    # Conditional enrichment rules
    rules:
      - condition: 'attributes["http.method"] == "POST"'
        attributes:
          priority: high
      - condition: 'resource.attributes["service.name"] == "api"'
        attributes:
          tier: frontend
    
    # Dynamic attribute computation
    dynamic:
      - target: "request.size_category"
        source: "http.request.body.size"
        transform: |
          if value < 1024:
            return "small"
          elif value < 1048576:
            return "medium"
          else:
            return "large"
```

## Metadata Sources

### System Information
- Hostname
- Operating System
- Architecture
- Number of CPUs
- Total Memory

### Cloud Provider Metadata
- **AWS**: Instance ID, Region, Availability Zone, Instance Type
- **GCP**: Instance ID, Project ID, Zone, Machine Type
- **Azure**: VM ID, Resource Group, Location, VM Size

### Kubernetes
- Pod Name, Namespace
- Node Name
- Cluster Name
- Container Runtime

### Process Information (via Privileged Helper)
- Process ID, Parent Process ID
- User/Group IDs
- Command Line
- Environment Variables
- Working Directory

## Usage

```yaml
service:
  pipelines:
    traces:
      processors: [nrenrich]
    metrics:
      processors: [nrenrich]
    logs:
      processors: [nrenrich]
```