# nrdot-k8s-operator

Kubernetes operator for managing NRDOT-Host deployments.

## Overview
Provides automated deployment and lifecycle management of NRDOT-Host in Kubernetes environments through custom resources.

## Features
- Custom Resource Definitions (CRDs)
- Automatic DaemonSet management
- ConfigMap generation
- Rolling updates
- Multi-cluster support

## CRDs
```yaml
apiVersion: nrdot.newrelic.com/v1
kind: NRDOTHost
metadata:
  name: production
spec:
  licenseKey: <secret>
  processMonitoring: true
  nodeSelector:
    node-role: worker
```

## Integration
- Deploys `nrdot-container-images`
- Works with `nrdot-helm-chart`
- Manages fleet via `nrdot-fleet-protocol`
