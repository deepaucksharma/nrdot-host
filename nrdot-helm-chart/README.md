# nrdot-helm-chart

Helm chart for deploying NRDOT-Host on Kubernetes.

## Overview
Production-grade Helm chart for easy deployment and configuration of NRDOT-Host in Kubernetes clusters.

## Features
- DaemonSet deployment
- ConfigMap management
- Secret handling
- Resource limits
- Prometheus ServiceMonitor

## Installation
```bash
helm repo add newrelic https://helm.newrelic.com
helm install nrdot-host newrelic/nrdot-host \
  --set licenseKey=YOUR_KEY \
  --set cluster.name=production
```

## Values
```yaml
# Key configuration options
image:
  repository: newrelic/nrdot-host
  tag: latest
resources:
  limits:
    memory: 200Mi
```

## Integration
- Uses `nrdot-container-images`
- Compatible with `nrdot-k8s-operator`
