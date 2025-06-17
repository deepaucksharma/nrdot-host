# ADR-001: Platform Architecture Decisions

**Status**: Accepted  
**Date**: 2024-01-15  
**Authors**: Platform Team  

## Context

The Clean Platform Implementation requires a robust, scalable architecture that aligns with New Relic's infrastructure standards while providing flexibility for future growth. Key requirements include:

- High availability across multiple regions
- Support for 100k+ requests per second
- Sub-100ms P99 latency
- Zero direct AWS access
- Comprehensive security and compliance
- Cost-effective resource utilization

## Decision

We will implement a microservices architecture deployed on Container Fabric (Kubernetes) with the following key architectural decisions:

### 1. Deployment Platform: Grand Central + Container Fabric

**Choice**: Grand Central for orchestration, Container Fabric for runtime

**Rationale**:
- Grand Central provides standardized deployment workflows
- Container Fabric offers managed Kubernetes with security policies
- Automatic integration with platform services
- No direct AWS access required

**Alternatives Considered**:
- Direct EKS deployment - Rejected due to AWS access requirement
- VM-based deployment - Rejected due to scaling limitations

### 2. Service Architecture: Microservices with API Gateway

**Choice**: Three core services behind NGINX API Gateway
- API Gateway: Traffic routing and authentication
- Data Collector: High-throughput data ingestion
- Data Processor: Async processing with Kafka

**Rationale**:
- Clear separation of concerns
- Independent scaling of components
- Fault isolation between services
- Standard pattern for platform teams

**Alternatives Considered**:
- Monolithic application - Rejected due to scaling constraints
- Serverless functions - Rejected due to cold start latency

### 3. Data Flow: Event-Driven with Kafka

**Choice**: Apache Kafka for async messaging

**Rationale**:
- Platform standard for event streaming
- High throughput (10k+ msgs/sec per partition)
- Built-in durability and replay capability
- Existing platform Kafka clusters

**Configuration**:
```yaml
kafka:
  producer:
    acks: 1
    compression: lz4
    batch_size: 65536
  consumer:
    fetch_min_bytes: 65536
    max_poll_records: 500
```

### 4. Storage: PostgreSQL + Redis

**Choice**: 
- PostgreSQL (Aurora) for persistent storage
- Redis (ElastiCache) for caching and queues

**Rationale**:
- Managed through Tim Allen tool
- Automatic backups and failover
- Platform standard databases
- No direct AWS access needed

### 5. Security Model: Zero Trust

**Choice**: Multiple layers of security
- FIPS-compliant base images
- Kyverno policy enforcement
- Network policies for pod isolation
- Vault for secrets management
- Non-root containers (UID > 10000)

**Rationale**:
- Meets SOC2 compliance requirements
- Platform security standards
- Defense in depth approach
- Automated policy enforcement

### 6. Monitoring: New Relic APM + Prometheus

**Choice**: Dual monitoring approach
- New Relic APM for application monitoring
- Prometheus for infrastructure metrics
- OpenTelemetry for distributed tracing

**Rationale**:
- Comprehensive observability
- Platform standard tools
- No additional licensing costs
- Rich ecosystem of exporters

### 7. Multi-Cell Strategy

**Choice**: Active-active across two cells
- Primary: us-core-ops (60% traffic)
- Secondary: us-alt-mule (40% traffic)

**Rationale**:
- High availability
- Geographic distribution
- Automatic failover capability
- Cell isolation for fault tolerance

### 8. CI/CD: Jenkins + ArgoCD

**Choice**: 
- Jenkins-in-Docker for CI
- ArgoCD for GitOps CD

**Rationale**:
- Platform standard CI/CD tools
- GitOps enables declarative deployments
- Automatic rollback capability
- Integration with Grand Central

## Consequences

### Positive
- Aligns with all platform standards
- Highly scalable and resilient
- Comprehensive security posture
- Rich monitoring and observability
- Cost-effective resource usage
- Easy to maintain and operate

### Negative
- Initial complexity in setup
- Multiple moving parts to coordinate
- Requires team training on all tools
- Dependency on platform services

### Risks
- Kafka cluster capacity limits
- Cell failover latency
- Grand Central API availability
- Learning curve for new team members

## Mitigation Strategies

1. **Capacity Planning**: Reserve Kafka capacity early with capacity council
2. **Failover Testing**: Monthly disaster recovery drills
3. **Documentation**: Comprehensive runbooks and training materials
4. **Monitoring**: Proactive alerts on all dependencies

## References

- [Container Fabric Documentation](../container-fabric_combined.md)
- [Grand Central API](../grand_central_combined.md)
- [Platform Security Standards](../container_security_combined.md)
- [Tim Allen Database Guide](../database-engineering_combined.md)