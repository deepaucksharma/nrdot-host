# Emergency Mitigation Plan - Clean Platform Implementation

## Immediate Actions Required (24 Hours)

### 1. Prevent Kafka Data Loss

**Current State**: Losing 350 messages per pod restart

**Emergency Fix** (2 hours):
```yaml
# grandcentral-enhanced.yml
env_vars:
  KAFKA_PRODUCER_ACKS: "all"  # Changed from "1"
  KAFKA_PRODUCER_ENABLE_IDEMPOTENCE: "true"
  KAFKA_PRODUCER_MAX_IN_FLIGHT_REQUESTS: "5"
  KAFKA_PRODUCER_RETRIES: "2147483647"  # Max retries
```

**Code Fix** (4 hours):
```python
# services/kafka-consumer/optimized_consumer.py
def _commit_batch_offsets(self, batch: List[Dict]):
    """Commit offsets ONLY after successful processing"""
    try:
        # Store batch for recovery
        self._persist_batch(batch)
        
        # Process with retry
        self._process_with_retry(batch)
        
        # Only commit after success
        offsets = self._calculate_offsets(batch)
        self.consumer.commit(offsets)
        
        # Clean up persisted batch
        self._cleanup_batch(batch)
    except Exception as e:
        # Batch remains persisted for retry
        logger.error(f"Batch processing failed: {e}")
        raise
```

### 2. Fix Database Connection Exhaustion

**Emergency Fix** (1 hour):
```yaml
# Add to all service deployments
env:
- name: DATABASE_POOL_SIZE
  value: "5"  # Max 5 connections per pod
- name: DATABASE_POOL_TIMEOUT
  value: "30"
```

**Infrastructure Fix** (4 hours):
```yaml
# k8s/base/database/pgbouncer.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pgbouncer
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: pgbouncer
        image: pgbouncer/pgbouncer:1.20.1
        env:
        - name: DATABASES_HOST
          value: "aurora-endpoint.cluster-xxxxx.us-east-1.rds.amazonaws.com"
        - name: DATABASES_PORT
          value: "5432"
        - name: POOL_MODE
          value: "transaction"
        - name: MAX_CLIENT_CONN
          value: "1000"
        - name: DEFAULT_POOL_SIZE
          value: "25"
        - name: MAX_DB_CONNECTIONS
          value: "100"
```

### 3. Secure Vault Tokens

**Emergency Fix** (2 hours):
```yaml
# Remove from all deployments
# DELETE THESE LINES:
- name: VAULT_TOKEN
  valueFrom:
    secretKeyRef:
      name: vault-credentials
      key: token
```

**Proper Fix** (8 hours):
```yaml
# Add Vault Agent sidecar
annotations:
  vault.hashicorp.com/agent-inject: "true"
  vault.hashicorp.com/role: "clean-platform"
  vault.hashicorp.com/agent-inject-secret-db: "database/creds/clean-platform"
  vault.hashicorp.com/agent-inject-template-db: |
    {{- with secret "database/creds/clean-platform" -}}
    export DATABASE_URL="postgresql://{{ .Data.username }}:{{ .Data.password }}@pgbouncer:5432/cleanplatform"
    {{- end }}
```

### 4. Fix Alert Delays

**Immediate Fix** (30 minutes):
```terraform
# terraform/monitoring/alerts.tf
# Change ALL alert conditions:
aggregation_delay = 0  # Was 120
```

## Critical Configuration Changes (48 Hours)

### 5. Fix Kafka Worker Alignment

```yaml
# grandcentral-enhanced.yml
kafka:
  topics:
    - name: "platform-events"
      partitions: 12
  performance:
    worker_threads: 12  # MUST equal partitions
```

### 6. Remove CPU Limits

```yaml
# ALL deployment files
resources:
  requests:
    memory: "1Gi"
    cpu: "500m"
  limits:
    memory: "2Gi"
    # cpu: REMOVED - no CPU limits
```

### 7. Fix Network Policies

```yaml
# k8s/base/security/network-policies-fixed.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: clean-platform-egress
spec:
  podSelector:
    matchLabels:
      app: data-collector
  policyTypes:
  - Egress
  egress:
  # Explicit allow list ONLY
  - to:
    - namespaceSelector:
        matchLabels:
          name: clean-platform
    ports:
    - protocol: TCP
      port: 5432  # PostgreSQL
  - to:
    - namespaceSelector:
        matchLabels:
          name: clean-platform
    ports:
    - protocol: TCP
      port: 6379  # Redis
  # Block metadata service
  - to:
    - ipBlock:
        cidr: 0.0.0.0/0
        except:
        - 169.254.169.254/32
        - 169.254.170.2/32  # EKS metadata
```

### 8. Add Circuit Breakers

```python
# services/common/resilience.py
from circuit_breaker import CircuitBreaker

# Global breakers for critical services
DB_BREAKER = CircuitBreaker(
    name="database",
    failure_threshold=5,
    recovery_timeout=60,
    expected_exception=DatabaseError
)

KAFKA_BREAKER = CircuitBreaker(
    name="kafka",
    failure_threshold=10,
    recovery_timeout=30,
    expected_exception=KafkaError
)

# Use in code:
@DB_BREAKER
def query_database(sql):
    return db.execute(sql)
```

## Monitoring for Success

### Add These Alerts Immediately:

```yaml
# terraform/monitoring/emergency-alerts.tf
resource "newrelic_nrql_alert_condition" "connection_pool_usage" {
  name = "Database Connection Pool Critical"
  nrql {
    query = "SELECT average(postgres.connections.used) / average(postgres.connections.max) * 100 FROM PostgresqlDatabaseSample"
  }
  critical {
    operator = "above"
    threshold = 80
    threshold_duration = 60
  }
}

resource "newrelic_nrql_alert_condition" "kafka_data_loss" {
  name = "Kafka Producer Failures"
  nrql {
    query = "SELECT count(*) FROM Log WHERE message LIKE '%Failed to produce message%'"
  }
  critical {
    operator = "above"
    threshold = 10
    threshold_duration = 60
  }
}

resource "newrelic_nrql_alert_condition" "vault_token_exposure" {
  name = "Vault Token in Environment"
  nrql {
    query = "SELECT count(*) FROM K8sContainerSample WHERE environment LIKE '%VAULT_TOKEN%'"
  }
  critical {
    operator = "above"
    threshold = 0
    threshold_duration = 60
  }
}
```

## Validation Checklist

After implementing fixes, verify:

### Data Loss Prevention
- [ ] Kafka producer uses `acks=all`
- [ ] Consumer only commits after successful processing
- [ ] DLQ configured with same guarantees
- [ ] No messages lost during pod restart test

### Connection Pool
- [ ] PgBouncer deployed and healthy
- [ ] Connection count < 100 per database
- [ ] No connection refused errors
- [ ] Applications use connection string with pooler

### Security
- [ ] No VAULT_TOKEN in pod environment
- [ ] Vault Agent injecting secrets
- [ ] Network policies blocking unnecessary egress
- [ ] All secrets in files, not environment

### Stability
- [ ] Alerts fire within 1 minute
- [ ] Circuit breakers prevent cascades
- [ ] CPU throttling eliminated
- [ ] Worker count matches partitions

## Rollback Plan

If any fix causes issues:

1. **Kafka Changes**: Revert producer config, drain topics
2. **PgBouncer**: Update services to direct database connection
3. **Vault Agent**: Temporarily use k8s secrets
4. **Network Policies**: Delete policies (permissive mode)

## Communication Plan

### Stakeholder Notification
```
Subject: Critical Platform Fixes In Progress

Team,

We've identified critical issues that could cause data loss and outages:
1. Kafka configuration causing message loss
2. Database connection limits being exceeded  
3. Security vulnerabilities in secret management
4. Alert delays hiding problems

We're implementing emergency fixes over the next 48 hours.

Impact: Minimal if executed correctly
Risk: High if not fixed immediately

Will update every 4 hours.
```

### Success Criteria

Platform is safe when:
- Zero data loss during normal operations
- Can scale to 50 pods without database issues
- Secrets not visible in pod specs
- Alerts fire within 60 seconds
- All critical risks mitigated

**Target: All CRITICAL fixes complete in 48 hours**