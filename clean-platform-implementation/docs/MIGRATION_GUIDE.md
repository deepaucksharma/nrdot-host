# Migration Guide: From New Relic Platform to Clean Implementation

## Overview

This guide helps you migrate from the New Relic-specific platform components to a clean, cloud-native implementation.

## Component Mapping

### Old → New Component Mapping

| New Relic Component | Clean Implementation | Notes |
|---------------------|---------------------|--------|
| Grand Central | GitHub Actions / GitLab CI | Standard CI/CD pipelines |
| Jenkins-in-Docker | GitHub Actions | Cloud-native CI/CD |
| Container Fabric | Standard EKS | Managed Kubernetes |
| Vault | AWS Secrets Manager | Native AWS service |
| cf-registry | Amazon ECR | Managed container registry |
| Change Management Service | GitHub PR Reviews | Built-in approval process |
| NR-Prod Okta | AWS IAM + SSO | Native AWS authentication |
| Monitoring | Prometheus + Grafana | Open-source monitoring |

## Migration Steps

### Phase 1: Assessment (Week 1)

1. **Inventory Current Services**
   ```bash
   # List all running services
   kubectl get deployments --all-namespaces
   
   # Export current configurations
   kubectl get all --all-namespaces -o yaml > current-state.yaml
   ```

2. **Document Dependencies**
   - External service dependencies
   - Database schemas
   - API contracts
   - Configuration requirements

3. **Identify Custom Components**
   - Platform-specific integrations
   - Custom controllers
   - Proprietary libraries

### Phase 2: Infrastructure Setup (Week 2)

1. **Create AWS Infrastructure**
   ```bash
   cd clean-platform-implementation/infrastructure/terraform/environments/dev
   terraform init
   terraform plan
   terraform apply
   ```

2. **Migrate Secrets**
   ```bash
   # Export from Vault
   vault kv get -format=json secret/team/service > secrets.json
   
   # Import to AWS Secrets Manager
   aws secretsmanager create-secret \
     --name platform/dev/service \
     --secret-string file://secrets.json
   ```

3. **Setup Networking**
   - Create VPC peering if needed
   - Configure security groups
   - Update DNS records

### Phase 3: Service Migration (Week 3-4)

1. **Containerize Services**
   ```dockerfile
   # Standard Dockerfile template
   FROM python:3.9-slim
   WORKDIR /app
   COPY requirements.txt .
   RUN pip install -r requirements.txt
   COPY . .
   USER 1000
   CMD ["python", "app.py"]
   ```

2. **Update Configuration**
   ```yaml
   # Old (Grand Central)
   environments:
     - name: staging
       datacenter: chicago
       
   # New (Kubernetes)
   metadata:
     namespace: platform-staging
   spec:
     replicas: 3
   ```

3. **Migrate Databases**
   ```bash
   # Backup existing database
   pg_dump -h old-host -U user -d dbname > backup.sql
   
   # Restore to new RDS instance
   psql -h new-rds-host -U admin -d dbname < backup.sql
   ```

### Phase 4: Cutover (Week 5)

1. **Parallel Run**
   - Deploy new services alongside old
   - Use feature flags for gradual migration
   - Monitor both systems

2. **Traffic Migration**
   ```bash
   # Update load balancer weights
   aws elbv2 modify-target-group-attributes \
     --target-group-arn arn:aws:elasticloadbalancing:... \
     --attributes Key=stickiness.enabled,Value=true
   ```

3. **Validation**
   - Compare metrics between systems
   - Run integration tests
   - Verify data consistency

## Code Changes Required

### Authentication
```python
# Old (NR-specific)
from nr_auth import authenticate
user = authenticate(token)

# New (Standard)
from functools import wraps
import jwt

def authenticate(f):
    @wraps(f)
    def decorated(*args, **kwargs):
        token = request.headers.get('Authorization')
        try:
            payload = jwt.decode(token, SECRET_KEY, algorithms=['HS256'])
            request.user = payload['user']
        except jwt.InvalidTokenError:
            return jsonify({'error': 'Invalid token'}), 401
        return f(*args, **kwargs)
    return decorated
```

### Monitoring
```python
# Old (New Relic APM)
import newrelic.agent
@newrelic.agent.function_trace()
def process_data():
    pass

# New (Prometheus)
from prometheus_client import Counter, Histogram
import time

request_count = Counter('requests_total', 'Total requests')
request_duration = Histogram('request_duration_seconds', 'Request duration')

def process_data():
    start = time.time()
    request_count.inc()
    # Process data
    request_duration.observe(time.time() - start)
```

### Configuration
```python
# Old (Vault)
import vault_client
config = vault_client.get_secret('service/config')

# New (Environment variables + AWS Secrets)
import os
import boto3
import json

def get_config():
    # Local config from environment
    config = {
        'port': int(os.getenv('PORT', 8080)),
        'log_level': os.getenv('LOG_LEVEL', 'info')
    }
    
    # Sensitive config from Secrets Manager
    if os.getenv('AWS_REGION'):
        sm = boto3.client('secretsmanager')
        secret = sm.get_secret_value(SecretId='platform/prod/api')
        config.update(json.loads(secret['SecretString']))
    
    return config
```

## Common Pitfalls

### 1. Hard-coded Platform Dependencies
```python
# Bad: Platform-specific
gc_client = GrandCentralClient()
gc_client.deploy()

# Good: Standard K8s
subprocess.run(['kubectl', 'apply', '-f', 'deployment.yaml'])
```

### 2. Missing Health Checks
```python
# Add standard health endpoints
@app.route('/health')
def health():
    return jsonify({'status': 'healthy'}), 200

@app.route('/ready')
def ready():
    # Check dependencies
    try:
        db.ping()
        redis.ping()
        return jsonify({'status': 'ready'}), 200
    except:
        return jsonify({'status': 'not ready'}), 503
```

### 3. Improper Secret Handling
```bash
# Bad: Secrets in code
DATABASE_URL = "postgresql://user:password@host/db"

# Good: Environment injection
DATABASE_URL = os.environ['DATABASE_URL']
```

## Rollback Plan

### Quick Rollback
```bash
# 1. Update DNS to point back to old system
aws route53 change-resource-record-sets \
  --hosted-zone-id Z123456 \
  --change-batch file://rollback-dns.json

# 2. Scale down new services
kubectl scale deployment --all --replicas=0 -n platform-prod

# 3. Restore traffic to old system
```

### Data Rollback
```bash
# If data changes were made
pg_dump new_database > rollback_backup.sql
psql old_database < original_backup.sql
```

## Validation Checklist

- [ ] All services deployed and healthy
- [ ] Monitoring dashboards showing data
- [ ] Alerts configured and tested
- [ ] Backup procedures verified
- [ ] Performance metrics comparable
- [ ] Security scans passed
- [ ] Documentation updated
- [ ] Team trained on new systems

## Support During Migration

### Monitoring Both Systems
```yaml
# Grafana dashboard for comparison
panels:
  - title: "Request Rate Comparison"
    targets:
      - expr: "rate(old_system_requests[5m])"
        legend: "Old System"
      - expr: "rate(new_system_requests[5m])"
        legend: "New System"
```

### Gradual Migration with Feature Flags
```python
if feature_flag('use_new_system'):
    return new_system_handler(request)
else:
    return old_system_handler(request)
```

This migration guide provides a structured approach to moving from New Relic-specific components to standard cloud-native alternatives while maintaining service reliability.