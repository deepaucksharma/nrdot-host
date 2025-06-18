# NRDOT Kubernetes Deployment

This directory contains Kubernetes manifests and Helm charts for deploying NRDOT-HOST in Kubernetes environments.

## Directory Structure

```
kubernetes/
├── manifests/          # Raw Kubernetes YAML manifests
├── helm/              # Helm chart for NRDOT
├── kustomize/         # Kustomize configurations
├── argocd/           # ArgoCD application examples
├── flux/             # Flux v2 configurations
└── openshift/        # OpenShift specific templates
```

## Quick Start

### Using Raw Manifests

1. **Create namespace and apply manifests:**
```bash
kubectl apply -f manifests/namespace.yaml
kubectl apply -f manifests/
```

2. **Configure secrets:**
```bash
# Replace with your actual New Relic license key
kubectl -n nrdot-system create secret generic nrdot-secrets \
  --from-literal=NEW_RELIC_LICENSE_KEY=your-license-key \
  --from-literal=API_AUTH_TOKEN=your-api-token
```

3. **Create TLS certificates:**
```bash
# Using cert-manager (recommended)
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.12.0/cert-manager.yaml

# Or create manually
kubectl -n nrdot-system create secret tls nrdot-tls \
  --cert=path/to/tls.crt \
  --key=path/to/tls.key
```

### Using Helm

1. **Install the Helm chart:**
```bash
helm install nrdot ./helm/nrdot \
  --namespace nrdot-system \
  --create-namespace \
  --set newrelic.licenseKey=YOUR_LICENSE_KEY
```

2. **Upgrade with custom values:**
```bash
helm upgrade nrdot ./helm/nrdot \
  --namespace nrdot-system \
  --values helm/nrdot/values-prod.yaml
```

### Using Kustomize

```bash
# Development environment
kubectl apply -k kustomize/overlays/dev

# Production environment
kubectl apply -k kustomize/overlays/prod
```

### Using ArgoCD

```bash
kubectl apply -f argocd/nrdot-application.yaml
```

### Using Flux v2

```bash
kubectl apply -f flux/nrdot-source.yaml
kubectl apply -f flux/nrdot-kustomization.yaml
```

## Components

### Core Components

1. **Config Engine** - Manages and validates configurations
2. **Supervisor** - Orchestrates and monitors NRDOT processes
3. **Collector** - OpenTelemetry collector with NRDOT processors
4. **API Server** - REST API for management and monitoring
5. **Privileged Helper** - DaemonSet for privileged operations

### Security Features

- **Pod Security Standards** - Enforces restricted security context
- **Network Policies** - Fine-grained network segmentation
- **RBAC** - Least privilege access controls
- **TLS** - End-to-end encryption
- **Secrets Management** - Secure credential handling

### High Availability

- Multiple replicas for critical components
- Pod Disruption Budgets
- Anti-affinity rules
- Zone-aware scheduling
- Automatic scaling (HPA/VPA)

## Configuration

### Environment Variables

Key environment variables for configuration:

- `NEW_RELIC_LICENSE_KEY` - Your New Relic license key
- `NRDOT_LOG_LEVEL` - Logging level (debug, info, warn, error)
- `NRDOT_CONFIG_PATH` - Path to configuration file
- `NRDOT_API_AUTH_TOKEN` - API authentication token

### Resource Requirements

Minimum resource requirements:

| Component | CPU Request | Memory Request | CPU Limit | Memory Limit |
|-----------|-------------|----------------|-----------|--------------|
| Collector | 1000m | 2Gi | 4000m | 4Gi |
| API Server | 250m | 256Mi | 1000m | 1Gi |
| Supervisor | 250m | 256Mi | 1000m | 1Gi |
| Config Engine | 100m | 128Mi | 500m | 512Mi |
| Privileged Helper | 50m | 64Mi | 250m | 256Mi |

### Storage

- ConfigMaps for configuration
- EmptyDir volumes for temporary data
- No persistent volumes required by default

## Monitoring

### Prometheus Metrics

All components expose Prometheus metrics:

- Collector: `:8888/metrics`
- API Server: `:8082/metrics`
- Supervisor: `:8082/metrics`
- Config Engine: `:8082/metrics`
- Privileged Helper: `:8084/metrics`

### Health Checks

All components provide health endpoints:

- Liveness probe: `/health`
- Readiness probe: `/ready`

## Troubleshooting

### Check Component Status

```bash
# Check all pods
kubectl -n nrdot-system get pods

# Check logs
kubectl -n nrdot-system logs -l app.kubernetes.io/name=nrdot

# Describe pod for events
kubectl -n nrdot-system describe pod <pod-name>
```

### Common Issues

1. **Pods not starting:**
   - Check secrets are created
   - Verify RBAC permissions
   - Check resource limits

2. **Network connectivity:**
   - Verify network policies
   - Check service endpoints
   - Test DNS resolution

3. **High memory usage:**
   - Review cardinality limits
   - Check batch processor settings
   - Adjust memory limits

## Production Considerations

1. **Security:**
   - Enable Pod Security Standards
   - Use network policies
   - Implement admission controllers
   - Regular security scanning

2. **Performance:**
   - Tune resource limits based on load
   - Configure appropriate HPA settings
   - Monitor cardinality metrics
   - Use node selectors for dedicated nodes

3. **Reliability:**
   - Configure PodDisruptionBudgets
   - Implement proper health checks
   - Use multiple availability zones
   - Regular backup configurations

4. **Compliance:**
   - Enable audit logging
   - Implement data retention policies
   - Configure encryption at rest
   - Document access controls

## Upgrading

### Rolling Updates

The manifests support zero-downtime rolling updates:

```bash
# Update image version
kubectl -n nrdot-system set image deployment/nrdot-collector \
  collector=docker.io/newrelic/nrdot-collector:v2.0.0

# Monitor rollout
kubectl -n nrdot-system rollout status deployment/nrdot-collector
```

### Helm Upgrades

```bash
# Test upgrade
helm upgrade nrdot ./helm/nrdot --namespace nrdot-system --dry-run

# Perform upgrade
helm upgrade nrdot ./helm/nrdot --namespace nrdot-system
```

## Integration

### GitOps

The manifests are designed for GitOps workflows:

- Declarative configuration
- Version controlled
- Automated deployment
- Drift detection

### CI/CD Pipeline

Example pipeline stages:

1. Lint manifests
2. Security scanning
3. Deploy to staging
4. Run integration tests
5. Deploy to production

## Support

For issues and questions:

1. Check component logs
2. Review Kubernetes events
3. Consult troubleshooting guide
4. Contact support team