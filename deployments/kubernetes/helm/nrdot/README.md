# NRDOT Helm Chart

This Helm chart deploys NRDOT (New Relic Data Optimization Tool) on a Kubernetes cluster.

## Prerequisites

- Kubernetes 1.21+
- Helm 3.8+
- New Relic License Key
- PV provisioner support in the underlying infrastructure (optional)

## Installation

### Add the Helm repository (when published)

```bash
helm repo add nrdot https://newrelic.github.io/nrdot-helm-charts
helm repo update
```

### Install from local directory

```bash
# Install with default values
helm install nrdot ./nrdot \
  --namespace nrdot-system \
  --create-namespace \
  --set newrelic.licenseKey=YOUR_LICENSE_KEY

# Install with custom values file
helm install nrdot ./nrdot \
  --namespace nrdot-system \
  --create-namespace \
  --values values-prod.yaml
```

## Configuration

See [values.yaml](values.yaml) for the full list of configurable parameters.

### Key Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `newrelic.licenseKey` | New Relic license key | `""` |
| `newrelic.licenseKeySecretName` | Existing secret with license key | `""` |
| `newrelic.otlpEndpoint` | New Relic OTLP endpoint | `otlp.nr-data.net:4317` |
| `newrelic.euDatacenter` | Use EU datacenter | `false` |
| `collector.enabled` | Enable collector deployment | `true` |
| `collector.replicaCount` | Number of collector replicas | `3` |
| `collector.resources` | Collector resource limits/requests | See values.yaml |
| `collector.autoscaling.enabled` | Enable HPA for collector | `true` |
| `apiServer.enabled` | Enable API server | `true` |
| `apiServer.service.type` | API server service type | `LoadBalancer` |
| `privilegedHelper.enabled` | Enable privileged helper DaemonSet | `true` |
| `networkPolicy.enabled` | Enable network policies | `true` |
| `tls.enabled` | Enable TLS | `true` |

### Example Configurations

#### Minimal Installation

```yaml
newrelic:
  licenseKey: YOUR_LICENSE_KEY

collector:
  replicaCount: 1
  resources:
    requests:
      cpu: 500m
      memory: 1Gi

apiServer:
  enabled: false

privilegedHelper:
  enabled: false
```

#### Production Installation

```yaml
newrelic:
  licenseKeySecretName: nrdot-license
  
collector:
  replicaCount: 5
  autoscaling:
    enabled: true
    minReplicas: 5
    maxReplicas: 20
  resources:
    requests:
      cpu: 2
      memory: 4Gi
    limits:
      cpu: 4
      memory: 8Gi

apiServer:
  replicaCount: 3
  ingress:
    enabled: true
    className: nginx
    hosts:
      - host: nrdot-api.example.com
        paths:
          - path: /
            pathType: Prefix
    tls:
      - secretName: nrdot-api-tls
        hosts:
          - nrdot-api.example.com

networkPolicy:
  enabled: true

monitoring:
  serviceMonitor:
    enabled: true
```

## Upgrading

```bash
# Upgrade to a new version
helm upgrade nrdot ./nrdot \
  --namespace nrdot-system \
  --values values-prod.yaml

# Upgrade with rollback on failure
helm upgrade nrdot ./nrdot \
  --namespace nrdot-system \
  --values values-prod.yaml \
  --atomic \
  --cleanup-on-fail
```

## Uninstalling

```bash
helm uninstall nrdot --namespace nrdot-system
```

## Monitoring

The chart includes Prometheus metrics endpoints for all components:

- Collector: `:8888/metrics`
- API Server: `:8082/metrics`
- Supervisor: `:8082/metrics`
- Config Engine: `:8082/metrics`
- Privileged Helper: `:8084/metrics`

Enable ServiceMonitor creation for Prometheus Operator:

```yaml
monitoring:
  serviceMonitor:
    enabled: true
    namespace: prometheus
```

## Security Considerations

1. **Network Policies**: Enabled by default to restrict traffic
2. **Pod Security**: Runs with restricted security context
3. **RBAC**: Minimal required permissions
4. **TLS**: Enabled by default for API server
5. **Secrets**: Store sensitive data in Kubernetes secrets

## Troubleshooting

### Check pod status

```bash
kubectl get pods -n nrdot-system
```

### View logs

```bash
# Collector logs
kubectl logs -n nrdot-system -l app.kubernetes.io/component=collector

# API server logs
kubectl logs -n nrdot-system -l app.kubernetes.io/component=api-server
```

### Debug with increased verbosity

```yaml
global:
  logLevel: debug
```

### Common Issues

1. **Pods in CrashLoopBackOff**
   - Check license key is valid
   - Verify resource limits are sufficient
   - Review pod logs for errors

2. **No metrics in New Relic**
   - Verify network connectivity to New Relic
   - Check collector logs for export errors
   - Ensure license key is correct

3. **High memory usage**
   - Adjust memory limits
   - Configure cardinality limits
   - Review batch processor settings

## Development

### Testing chart changes

```bash
# Lint the chart
helm lint ./nrdot

# Dry run installation
helm install nrdot ./nrdot --dry-run --debug

# Template rendering
helm template nrdot ./nrdot --values values-dev.yaml
```

### Running tests

```bash
helm test nrdot --namespace nrdot-system
```

## Contributing

Please see the main NRDOT repository for contribution guidelines.

## License

Apache License 2.0