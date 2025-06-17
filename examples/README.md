# NRDOT-HOST Examples

This directory contains example configurations and deployment scenarios for NRDOT-HOST. Each example is designed to demonstrate specific use cases and best practices.

## Available Examples

### 🚀 Basic Setup
Simple single-instance deployment perfect for getting started or development environments.
- **Path**: `basic/`
- **Use Case**: Development, testing, small-scale deployments
- **Key Features**: Minimal configuration, Docker Compose setup

### ☸️ Kubernetes Deployment
Production-ready Kubernetes deployment with horizontal scaling capabilities.
- **Path**: `kubernetes/`
- **Use Case**: Container orchestration, auto-scaling, production workloads
- **Key Features**: ConfigMaps, Deployments, Service definitions

### ☁️ Cloud Native Deployments
Platform-specific configurations optimized for major cloud providers.
- **AWS**: `cloud-native/aws/` - ECS/EKS ready with S3 integration
- **GCP**: `cloud-native/gcp/` - GKE optimized with Cloud Storage
- **Azure**: `cloud-native/azure/` - AKS ready with Blob Storage

### ⚡ High Performance
Optimized configurations for maximum throughput and minimal latency.
- **Path**: `high-performance/`
- **Use Case**: High-volume data processing, real-time analytics
- **Key Features**: Performance tuning, resource optimization

### 🔐 Security Focused
Security-hardened configurations with compliance considerations.
- **Path**: `security-focused/`
- **Use Case**: Regulated industries, sensitive data processing
- **Key Features**: Encryption, access controls, audit logging

### 🏢 Multi-Tenant
Configurations for serving multiple isolated tenants from a single deployment.
- **Path**: `multi-tenant/`
- **Use Case**: SaaS platforms, enterprise deployments
- **Key Features**: Tenant isolation, resource quotas, routing

## Quick Start

1. Choose an example that matches your use case
2. Copy the configuration files to your deployment directory
3. Customize the settings according to your requirements
4. Follow the deployment instructions in each example's README

## Configuration Hierarchy

All examples follow this configuration precedence:
1. Environment variables (highest priority)
2. Configuration file (`nrdot-config.yaml`)
3. Default values (lowest priority)

## Common Customizations

### Data Sources
```yaml
sources:
  - type: kafka
    config:
      brokers: ["localhost:9092"]
      topics: ["events"]
```

### Processors
```yaml
processors:
  - type: filter
    config:
      expression: "status == 'active'"
  - type: transform
    config:
      mapping:
        user_id: $.userId
        timestamp: $.eventTime
```

### Outputs
```yaml
outputs:
  - type: elasticsearch
    config:
      hosts: ["localhost:9200"]
      index: "events"
```

## Performance Considerations

- **Memory**: Allocate at least 2GB for basic deployments
- **CPU**: 2+ cores recommended for production
- **Storage**: Fast SSD storage for write-heavy workloads
- **Network**: Low-latency connections between components

## Support

For questions or issues:
- Check the example-specific README files
- Review the main documentation
- Open an issue on GitHub
- Contact support for enterprise deployments