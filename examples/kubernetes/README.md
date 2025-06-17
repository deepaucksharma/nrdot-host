# Kubernetes NRDOT-HOST Example

Production-ready Kubernetes deployment with horizontal scaling, high availability, and cloud-native integrations.

## Overview

This example provides:
- Horizontally scalable deployment with HPA
- High availability with pod anti-affinity
- Leader election for distributed coordination
- Integration with Kafka, Elasticsearch, and Redis
- Comprehensive monitoring and tracing
- Security best practices

## Prerequisites

- Kubernetes cluster (1.19+)
- kubectl configured
- Helm 3 (for dependencies)
- Minimum 3 nodes with 4GB RAM each

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Kafka Cluster │────▶│  NRDOT-HOST     │────▶│  Elasticsearch  │
└─────────────────┘     │  (3+ replicas)  │     └─────────────────┘
                        └─────────────────┘              │
                                │                        │
                                ▼                        ▼
                        ┌─────────────────┐     ┌─────────────────┐
                        │     Redis       │     │   S3 Archive    │
                        └─────────────────┘     └─────────────────┘
```

## Installation

### 1. Create Namespace

```bash
kubectl create namespace nrdot-system
```

### 2. Install Dependencies

```bash
# Redis
helm repo add bitnami https://charts.bitnami.com/bitnami
helm install redis bitnami/redis \
  --namespace nrdot-system \
  --set auth.password=redis-password-changeme \
  --set replica.replicaCount=3

# Elasticsearch
helm repo add elastic https://helm.elastic.co
helm install elasticsearch elastic/elasticsearch \
  --namespace nrdot-system \
  --set replicas=3 \
  --set minimumMasterNodes=2

# Kafka (optional, if not using existing)
helm repo add confluentinc https://confluentinc.github.io/cp-helm-charts/
helm install kafka confluentinc/cp-helm-charts \
  --namespace nrdot-system \
  --set cp-kafka.brokers=3
```

### 3. Deploy NRDOT-HOST

```bash
# Create ConfigMap from configuration file
kubectl create configmap nrdot-config \
  --from-file=config.yaml=nrdot-config.yaml \
  -n nrdot-system

# Apply Kubernetes manifests
kubectl apply -f configmap.yaml
kubectl apply -f deployment.yaml
```

### 4. Verify Deployment

```bash
# Check pod status
kubectl get pods -n nrdot-system -l app=nrdot-host

# Check services
kubectl get svc -n nrdot-system

# View logs
kubectl logs -n nrdot-system -l app=nrdot-host --tail=100

# Check HPA status
kubectl get hpa -n nrdot-system
```

## Configuration

### Environment Variables

Key environment variables to customize:

```yaml
env:
- name: LOG_LEVEL
  value: "info"  # debug, info, warn, error
- name: CLUSTER_NAME
  value: "production"  # your cluster identifier
- name: AWS_REGION
  value: "us-east-1"  # AWS region for S3
- name: S3_BUCKET
  value: "nrdot-events-archive"  # S3 bucket name
```

### Scaling Configuration

Adjust HPA settings in `deployment.yaml`:

```yaml
spec:
  minReplicas: 3      # Minimum pods
  maxReplicas: 10     # Maximum pods
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70  # Scale at 70% CPU
```

### Resource Limits

Modify resource requests/limits:

```yaml
resources:
  requests:
    memory: "2Gi"    # Guaranteed memory
    cpu: "1000m"     # Guaranteed CPU (1 core)
  limits:
    memory: "4Gi"    # Maximum memory
    cpu: "2000m"     # Maximum CPU (2 cores)
```

## Monitoring

### Prometheus Metrics

Metrics are exposed on port 9090:

```bash
# Port forward to access metrics
kubectl port-forward -n nrdot-system svc/nrdot-host 9090:9090

# View metrics
curl http://localhost:9090/metrics
```

Key metrics to monitor:
- `nrdot_events_processed_total` - Total events processed
- `nrdot_processing_duration_seconds` - Processing latency
- `nrdot_pipeline_errors_total` - Pipeline errors
- `nrdot_output_batch_size` - Output batch sizes

### Grafana Dashboard

Import the provided dashboard:

```bash
kubectl create configmap grafana-dashboards \
  --from-file=grafana/nrdot-dashboard.json \
  -n monitoring
```

### Distributed Tracing

View traces in Jaeger:

```bash
# Port forward Jaeger UI
kubectl port-forward -n nrdot-system svc/jaeger-query 16686:16686

# Open http://localhost:16686
```

## High Availability

### Leader Election

The deployment uses Kubernetes lease-based leader election for:
- Singleton processors
- Scheduled tasks
- Cache invalidation

### Pod Distribution

Anti-affinity rules ensure pods are distributed across nodes:

```yaml
affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
    - weight: 100
      podAffinityTerm:
        topologyKey: kubernetes.io/hostname
```

### Circuit Breaker

Automatic circuit breaking for downstream services:
- Opens after 5 consecutive failures
- Half-opens after 30 seconds
- Closes after 2 successful requests

## Security

### RBAC

Limited permissions for service account:
- ConfigMap access for configuration
- Lease access for leader election

### Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: nrdot-host
  namespace: nrdot-system
spec:
  podSelector:
    matchLabels:
      app: nrdot-host
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8080
```

### Pod Security

- Non-root user (UID 1000)
- Read-only root filesystem
- No capabilities
- Security context enforced

## Troubleshooting

### Pods not starting

```bash
# Check events
kubectl describe pod -n nrdot-system <pod-name>

# Check resource availability
kubectl top nodes
kubectl describe node <node-name>
```

### High memory usage

```bash
# Check memory usage
kubectl top pods -n nrdot-system -l app=nrdot-host

# Adjust GC settings
kubectl set env deployment/nrdot-host GOGC=50 -n nrdot-system
```

### Processing lag

```bash
# Check consumer lag
kubectl exec -n nrdot-system <pod-name> -- nrdot-cli lag

# Scale up replicas
kubectl scale deployment/nrdot-host --replicas=5 -n nrdot-system
```

## Production Checklist

- [ ] Configure proper resource limits
- [ ] Set up monitoring and alerting
- [ ] Enable audit logging
- [ ] Configure backup strategy
- [ ] Test disaster recovery
- [ ] Review security policies
- [ ] Set up TLS/SSL
- [ ] Configure external DNS
- [ ] Enable pod security policies
- [ ] Set up GitOps workflow

## Advanced Topics

### Multi-Region Deployment

For multi-region setup, see `cloud-native/` examples.

### Custom Processors

Add custom processors by mounting additional code:

```yaml
volumeMounts:
- name: custom-processors
  mountPath: /app/processors
volumes:
- name: custom-processors
  configMap:
    name: custom-processors
```

### Integration with Service Mesh

Compatible with Istio/Linkerd. Add appropriate annotations:

```yaml
metadata:
  annotations:
    sidecar.istio.io/inject: "true"
```