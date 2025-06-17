# Platform Compliance Enforcement Guide

## Overview

This guide ensures all developers follow platform guidelines documented in the .md files. It provides automated enforcement mechanisms, validation tools, and clear procedures for maintaining compliance.

## Enforcement Architecture

```
┌─────────────────────┐
│   Pre-commit Hooks  │ ← Local validation before commit
└──────────┬──────────┘
           │
┌──────────▼──────────┐
│   CI/CD Pipeline    │ ← Automated validation in Jenkins
└──────────┬──────────┘
           │
┌──────────▼──────────┐
│  Runtime Monitors   │ ← Production compliance checks
└─────────────────────┘
```

## 1. Local Development Enforcement

### Setup Pre-commit Hooks

```bash
# Install pre-commit
pip install pre-commit

# Install hooks
pre-commit install

# Run manually
pre-commit run --all-files
```

### Key Validation Hooks

| Hook | Purpose | Blocks |
|------|---------|--------|
| `check-grand-central-config` | Validates grandcentral.yml | Invalid config |
| `check-fips-compliance` | Ensures FIPS base images | Non-FIPS images |
| `check-vault-paths` | Validates secret paths | Hardcoded secrets |
| `check-required-integrations` | Verifies platform integrations | Missing integrations |
| `compliance-validation` | Full platform compliance | Any violation |

## 2. Required Platform Integrations

Every service MUST integrate these components:

### Change Management Client

```python
# services/common/change_management_client.py
from typing import Dict
import requests

class ChangeManagementClient:
    """Required for all deployments"""
    
    def __init__(self, api_endpoint: str, api_key: str):
        self.api_endpoint = api_endpoint
        self.headers = {"Authorization": f"Bearer {api_key}"}
    
    def create_change_request(self, deployment_info: Dict) -> str:
        """Create change request before deployment"""
        # Implementation required
        pass
```

### Grand Central Client

```python
# services/common/grand_central_client.py
class GrandCentralClient:
    """Required for deployment orchestration"""
    
    def __init__(self, api_token: str):
        self.base_url = "https://grand-central.nr-ops.net/api/v1"
        self.headers = {"X-Grand-Central-Auth": api_token}
    
    def trigger_deployment(self, project: str, env: str, version: str):
        """Trigger Grand Central deployment"""
        # Implementation required
        pass
```

### Feature Flags Integration

```python
# In every service's app.py
from services.feature_flags.feature_flags_client import FeatureFlagsClient

# Initialize
ff_client = FeatureFlagsClient(
    service_url=os.environ['FEATURE_FLAGS_SERVICE_URL'],
    api_key=os.environ['FEATURE_FLAGS_API_KEY'],
    environment=os.environ['ENVIRONMENT']
)

# Use in endpoints
@app.route('/api/v1/feature')
def feature_endpoint():
    if ff_client.is_enabled('new_feature'):
        return new_implementation()
    return old_implementation()
```

## 3. Configuration Requirements

### Grand Central Configuration

**Required Fields:**
```yaml
# grandcentral.yml
project_name: <project>
team_name: <team>
deploy_mechanism: kubernetes  # or terraform
slack_channel: "#team-channel"

# Required for production
change_management:
  enabled: true
  
# Required integrations
monitoring:
  prometheus:
    enabled: true
    
entity_synthesis:
  domain: PLATFORM
  type: SERVICE
```

### Dockerfile Requirements

```dockerfile
# Must use FIPS-compliant base
FROM cf-registry.nr-ops.net/newrelic/python-3.11-fips:latest

# Must run as non-root with UID >= 10000
RUN groupadd -r -g 10001 appuser && \
    useradd -r -u 10001 -g appuser appuser
USER 10001

# Must have security labels
LABEL security.scan="required" \
      fips.compliant="true"
```

### Kubernetes Manifests

```yaml
# Required security context
securityContext:
  runAsNonRoot: true
  runAsUser: 10001
  runAsGroup: 10001
  fsGroup: 10001
  seccompProfile:
    type: RuntimeDefault

# Required for containers
containers:
- name: app
  securityContext:
    allowPrivilegeEscalation: false
    readOnlyRootFilesystem: true
    capabilities:
      drop:
      - ALL
  
  # Required probes
  livenessProbe:
    httpGet:
      path: /healthz
  readinessProbe:
    httpGet:
      path: /readyz
  
  # Required resource limits
  resources:
    requests:
      cpu: 100m
      memory: 256Mi
    limits:
      memory: 512Mi
      # Note: No CPU limits per platform guidance
```

## 4. CI/CD Enforcement

### Jenkins Pipeline Stages

```groovy
// Required validation stage
stage('Platform Compliance') {
    steps {
        sh '''
            # Run compliance checker
            python scripts/compliance-checker.py
            
            # Validate Grand Central config
            python scripts/validate-grand-central.py grandcentral.yml
            
            # Check FIPS compliance
            find . -name Dockerfile -exec scripts/check-fips-compliance.sh {} +
            
            # Validate Kubernetes manifests
            kubectl --dry-run=client apply -f k8s/
        '''
    }
}

// Required security scanning
stage('Security Scan') {
    steps {
        sh '''
            # Scan for secrets
            trufflehog --regex --entropy=False .
            
            # Scan Docker images
            trivy image --severity HIGH,CRITICAL ${IMAGE}
            
            # Check dependencies
            safety check -r requirements.txt
        '''
    }
}
```

## 5. Common Violations and Fixes

### ❌ Violation: Non-FIPS Base Image

```dockerfile
# WRONG
FROM python:3.11-slim

# CORRECT
FROM cf-registry.nr-ops.net/newrelic/python-3.11-fips:latest
```

### ❌ Violation: Hardcoded Secrets

```yaml
# WRONG
env:
- name: DATABASE_PASSWORD
  value: "mysecretpassword"

# CORRECT
env:
- name: DATABASE_PASSWORD
  valueFrom:
    secretKeyRef:
      name: db-credentials
      key: password
```

### ❌ Violation: Missing Resource Limits

```yaml
# WRONG
containers:
- name: app
  image: myimage

# CORRECT
containers:
- name: app
  image: myimage
  resources:
    requests:
      cpu: 100m
      memory: 256Mi
    limits:
      memory: 512Mi
```

### ❌ Violation: No Health Checks

```python
# WRONG - No health endpoints

# CORRECT
@app.route('/healthz')
def healthz():
    return jsonify({"status": "healthy"}), 200

@app.route('/readyz')
def readyz():
    # Check dependencies
    try:
        redis_client.ping()
        return jsonify({"status": "ready"}), 200
    except:
        return jsonify({"status": "not ready"}), 503
```

## 6. Monitoring Compliance

### Runtime Compliance Checks

```python
# monitoring/compliance_monitor.py
class ComplianceMonitor:
    """Monitors runtime compliance"""
    
    def check_deployments(self):
        """Verify all deployments meet requirements"""
        violations = []
        
        # Check running containers
        for pod in get_running_pods():
            # Verify non-root
            if pod.security_context.run_as_user < 10000:
                violations.append(f"Pod {pod.name} running as privileged user")
            
            # Verify resource limits
            if not pod.resources.limits:
                violations.append(f"Pod {pod.name} missing resource limits")
        
        return violations
```

### Compliance Dashboard

```yaml
# monitoring/compliance-dashboard.json
{
  "dashboard": {
    "title": "Platform Compliance",
    "panels": [
      {
        "title": "Non-Compliant Deployments",
        "query": "FROM K8sContainerSample SELECT count(*) WHERE runAsUser < 10000"
      },
      {
        "title": "Missing Resource Limits",
        "query": "FROM K8sContainerSample SELECT count(*) WHERE memoryLimitBytes IS NULL"
      },
      {
        "title": "FIPS Compliance",
        "query": "FROM K8sContainerSample SELECT count(*) WHERE image NOT LIKE '%fips%'"
      }
    ]
  }
}
```

## 7. Automated Remediation

### Auto-fix Script

```bash
#!/bin/bash
# scripts/auto-fix-compliance.sh

echo "🔧 Auto-fixing common compliance issues..."

# Fix Dockerfile USER statements
find . -name Dockerfile -exec sed -i 's/USER [0-9]\{1,4\}$/USER 10001/g' {} +

# Add missing resource limits
for f in k8s/**/*.yaml; do
    if grep -q "kind: Deployment" "$f" && ! grep -q "resources:" "$f"; then
        echo "Adding resource limits to $f"
        # Add resources section
    fi
done

# Update base images to FIPS
find . -name Dockerfile -exec sed -i 's|FROM python:|FROM cf-registry.nr-ops.net/newrelic/python-3.11-fips:|g' {} +
```

## 8. Compliance Checklist

Before any deployment, ensure:

- [ ] All Dockerfiles use FIPS-compliant base images
- [ ] All containers run as non-root (UID >= 10000)
- [ ] Resource limits defined for all containers
- [ ] Health check endpoints implemented
- [ ] Network policies are restrictive (not 0.0.0.0/0)
- [ ] Secrets use Vault references, not hardcoded
- [ ] Change management integration configured
- [ ] Monitoring and alerting configured
- [ ] Entity platform integration enabled
- [ ] All pre-commit hooks passing
- [ ] Compliance score >= 90%

## 9. Getting Help

### Compliance Issues
- Run: `python scripts/compliance-checker.py --verbose`
- Check: `COMPLIANCE-VALIDATION-REPORT.md`
- Slack: #platform-compliance

### Platform Standards
- Grand Central: See `grand_central_combined.md`
- Security: See `container_security_combined.md`
- Monitoring: See `monitoring_combined.md`

## 10. Enforcement Metrics

Track compliance with these KPIs:

| Metric | Target | Alert Threshold |
|--------|--------|-----------------|
| Pre-commit Hook Success Rate | 100% | < 95% |
| CI/CD Compliance Pass Rate | 100% | < 98% |
| Production Non-Compliance | 0 | > 0 |
| FIPS Image Coverage | 100% | < 100% |
| Secret Hardcoding | 0 | > 0 |

## Conclusion

Platform compliance is not optional. These enforcement mechanisms ensure all code meets New Relic's platform standards before reaching production. Regular compliance checks and automated validation prevent technical debt and security vulnerabilities.

**Remember**: It's easier to build compliant from the start than to fix violations later.