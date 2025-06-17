# Multi-Tenant Isolation Guide for NRDOT-HOST

This guide explains how to implement and maintain strong isolation between tenants in a multi-tenant NRDOT-HOST deployment.

## Overview

Multi-tenant isolation is critical for:
- **Security**: Preventing data leaks between tenants
- **Performance**: Ensuring fair resource allocation
- **Compliance**: Meeting regulatory requirements
- **Operations**: Simplifying management and troubleshooting

## Isolation Strategies

### 1. Network Isolation

#### VPC Per Tenant (High Isolation)
```yaml
network:
  strategy: "vpc"
  vpc_per_tenant: true
  vpc_config:
    cidr_pattern: "10.${tenant_id}.0.0/16"
    enable_flow_logs: true
    enable_dns_hostnames: true
```

**Pros:**
- Complete network isolation
- Dedicated IP space
- Independent security policies

**Cons:**
- Higher infrastructure cost
- Complex cross-tenant communication
- VPC limits per account

#### Security Groups Per Tenant (Medium Isolation)
```yaml
network:
  strategy: "security_group"
  security_group_per_tenant: true
  rules:
    ingress:
      - from_port: 443
        to_port: 443
        protocol: "tcp"
        source: "${tenant_cidr}"
```

**Pros:**
- Good isolation within shared VPC
- Cost-effective
- Easier management

**Cons:**
- Shared network infrastructure
- Potential for misconfiguration

#### Namespace Isolation (Kubernetes)
```yaml
network:
  strategy: "namespace"
  namespace_pattern: "nrdot-${tenant_id}"
  network_policies:
    enabled: true
    default_deny: true
```

**Pros:**
- Native Kubernetes isolation
- Resource quotas per namespace
- RBAC integration

**Cons:**
- Requires Kubernetes
- Shared cluster resources

### 2. Data Isolation

#### Database Isolation

**Database Per Tenant**
```yaml
database:
  isolation_mode: "database"
  database_pattern: "nrdot_${tenant_id}"
  connection_string: "postgresql://host/${database_name}"
```

**Schema Per Tenant**
```yaml
database:
  isolation_mode: "schema"
  schema_pattern: "tenant_${tenant_id}"
  search_path: "${schema_name},public"
```

**Row-Level Security**
```yaml
database:
  isolation_mode: "row_level"
  tenant_column: "tenant_id"
  rls_policy: |
    CREATE POLICY tenant_isolation ON events
    FOR ALL
    USING (tenant_id = current_setting('app.tenant_id'));
```

#### Storage Isolation

**S3 Bucket Per Tenant**
```yaml
storage:
  s3:
    isolation: "bucket"
    bucket_pattern: "nrdot-${tenant_id}-data"
    iam_role_pattern: "arn:aws:iam::${account}:role/nrdot-${tenant_id}"
```

**S3 Prefix Isolation**
```yaml
storage:
  s3:
    isolation: "prefix"
    bucket: "nrdot-multi-tenant"
    prefix_pattern: "tenants/${tenant_id}/"
    bucket_policy:
      enforce_prefix_isolation: true
```

### 3. Compute Isolation

#### Container Isolation
```yaml
compute:
  container:
    cpu_shares: 1024
    memory_limit: "4Gi"
    pids_limit: 1000
    security_context:
      run_as_non_root: true
      read_only_root_filesystem: true
```

#### Process Isolation
```yaml
compute:
  process:
    cgroups:
      cpu_quota: 200000  # 2 CPUs
      memory_limit: "4G"
    namespaces:
      - "pid"
      - "net"
      - "ipc"
      - "uts"
      - "mount"
```

### 4. Resource Isolation

#### Quota Management
```yaml
quotas:
  events_per_second:
    default: 1000
    enforcement: "hard"
    burst_multiplier: 1.5
  storage_gb:
    default: 100
    enforcement: "soft"
    warning_threshold: 0.8
  api_requests_per_minute:
    default: 1000
    enforcement: "token_bucket"
```

#### Rate Limiting
```yaml
rate_limiting:
  strategy: "distributed"
  backend: "redis"
  limits:
    - resource: "api_requests"
      window: "1m"
      limit: "${tenant.api_quota}"
    - resource: "data_ingestion"
      window: "1s"
      limit: "${tenant.ingestion_quota}"
```

## Implementation Patterns

### 1. Tenant Context Propagation

```go
// Middleware to extract and propagate tenant context
func TenantMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        tenantID := extractTenantID(r)
        ctx := context.WithValue(r.Context(), "tenant_id", tenantID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Use tenant context in processing
func ProcessEvent(ctx context.Context, event Event) error {
    tenantID := ctx.Value("tenant_id").(string)
    // Ensure all operations are scoped to tenant
    return processWithTenant(tenantID, event)
}
```

### 2. Tenant-Aware Data Access

```yaml
# Configuration for tenant-aware data access
data_access:
  # Automatic tenant filtering
  auto_filter:
    enabled: true
    filter_field: "tenant_id"
    
  # Query rewriting
  query_rewrite:
    enabled: true
    rules:
      - pattern: "SELECT * FROM events"
        rewrite: "SELECT * FROM events WHERE tenant_id = :tenant_id"
```

### 3. Cross-Tenant Access Control

```yaml
cross_tenant_access:
  # Default deny all
  default_policy: "deny"
  
  # Explicit permissions
  permissions:
    - role: "super_admin"
      tenants: ["*"]
      actions: ["read", "write", "admin"]
    - role: "support"
      tenants: ["*"]
      actions: ["read"]
    - role: "tenant_admin"
      tenants: ["self"]
      actions: ["read", "write", "admin"]
```

## Security Best Practices

### 1. Authentication and Authorization

```yaml
auth:
  # Tenant-specific auth providers
  providers:
    - type: "oauth2"
      tenant_claim: "org_id"
      validation:
        issuer_pattern: "https://${tenant_id}.auth.example.com"
        audience: "nrdot-api"
        
  # Tenant validation
  validation:
    require_tenant_header: true
    validate_tenant_exists: true
    validate_tenant_active: true
```

### 2. Encryption Per Tenant

```yaml
encryption:
  # Unique key per tenant
  key_management:
    provider: "aws_kms"
    key_pattern: "alias/nrdot-${tenant_id}"
    
  # Field-level encryption
  field_encryption:
    enabled: true
    fields: ["pii", "sensitive_data"]
    algorithm: "AES-256-GCM"
```

### 3. Audit Logging

```yaml
audit:
  # Log all cross-tenant access
  cross_tenant_access:
    enabled: true
    log_level: "info"
    
  # Tenant-specific audit trails
  tenant_audit:
    enabled: true
    retention_days: 365
    fields:
      - "user_id"
      - "action"
      - "resource"
      - "timestamp"
      - "ip_address"
```

## Operational Considerations

### 1. Tenant Onboarding

```bash
# Automated tenant provisioning script
#!/bin/bash

TENANT_ID=$1
TIER=$2

# Create database schema
psql -c "CREATE SCHEMA tenant_${TENANT_ID};"

# Create Kafka topics
kafka-topics --create --topic "${TENANT_ID}-events" --partitions 10

# Create S3 bucket
aws s3 mb "s3://nrdot-${TENANT_ID}-data"

# Configure IAM role
aws iam create-role --role-name "nrdot-${TENANT_ID}" \
  --assume-role-policy-document file://trust-policy.json

# Update tenant registry
curl -X POST https://api.nrdot.com/tenants \
  -d "{\"id\": \"${TENANT_ID}\", \"tier\": \"${TIER}\"}"
```

### 2. Monitoring and Alerting

```yaml
monitoring:
  # Per-tenant dashboards
  dashboards:
    - name: "tenant-overview"
      widgets:
        - type: "timeseries"
          metric: "events_processed"
          group_by: "tenant_id"
        - type: "gauge"
          metric: "storage_used"
          group_by: "tenant_id"
          
  # Tenant-specific alerts
  alerts:
    - name: "tenant_quota_exceeded"
      condition: "usage > quota * 0.9"
      notification:
        channel: "tenant_contact"
        template: "quota_warning"
```

### 3. Backup and Recovery

```yaml
backup:
  # Tenant-specific backups
  strategy: "per_tenant"
  schedule:
    enterprise: "0 */4 * * *"  # Every 4 hours
    standard: "0 0 * * *"      # Daily
    
  # Isolated recovery
  recovery:
    isolation: true
    validation: true
    test_restore: "monthly"
```

## Testing Tenant Isolation

### 1. Security Testing

```bash
# Test cross-tenant data access
curl -H "X-Tenant-ID: tenant-a" \
     https://api.nrdot.com/data/tenant-b/events
# Expected: 403 Forbidden

# Test resource limits
for i in {1..10000}; do
  curl -H "X-Tenant-ID: tenant-a" \
       https://api.nrdot.com/events
done
# Expected: Rate limit after quota
```

### 2. Performance Testing

```yaml
performance_tests:
  - name: "noisy_neighbor"
    description: "Ensure one tenant cannot impact others"
    steps:
      - load_tenant_a: "100% capacity"
      - measure_tenant_b: "latency and throughput"
      - assert: "tenant_b.latency < 100ms"
```

### 3. Compliance Validation

```yaml
compliance_checks:
  - name: "data_isolation"
    checks:
      - "No shared database connections"
      - "Encryption keys are unique"
      - "Audit logs are separated"
      - "Backups are isolated"
```

## Troubleshooting

### Common Issues

1. **Tenant ID not propagating**
   - Check middleware ordering
   - Verify context passing
   - Enable debug logging

2. **Cross-tenant data leaks**
   - Review database queries
   - Check cache keys include tenant ID
   - Audit data access patterns

3. **Performance degradation**
   - Monitor resource quotas
   - Check for noisy neighbors
   - Review isolation boundaries

### Debug Commands

```bash
# Check tenant isolation
nrdot-cli tenant verify --tenant-id tenant-a

# View tenant resources
nrdot-cli tenant resources --tenant-id tenant-a

# Test tenant access
nrdot-cli tenant test-access --from tenant-a --to tenant-b

# Monitor tenant metrics
nrdot-cli tenant metrics --tenant-id tenant-a --period 1h
```

## Migration Guide

### Moving to Multi-Tenant

1. **Assessment Phase**
   - Identify tenant boundaries
   - Catalog shared resources
   - Plan isolation strategy

2. **Preparation Phase**
   - Update schemas for tenant ID
   - Implement tenant context
   - Add isolation controls

3. **Migration Phase**
   - Migrate data with tenant tags
   - Update access controls
   - Test isolation thoroughly

4. **Validation Phase**
   - Run isolation tests
   - Perform security audit
   - Monitor for issues

Remember: Strong tenant isolation is not just a technical requirement but a business imperative for multi-tenant systems. Regular audits and testing ensure isolation remains effective as the system evolves.