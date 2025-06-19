# NRDOT-HOST Deployment Configurations

This directory contains all deployment configurations for NRDOT-HOST across different platforms.

## Directory Structure

### docker/
Docker images and Docker Compose configurations:
- `unified/`: Unified binary Docker image (v2.0)
- `legacy/`: Legacy microservices images
- `configs/`: Configuration files for Docker deployments
- `docker-compose.v2.yaml`: Docker Compose for unified deployment
- `docker-compose.yaml`: Legacy microservices deployment

### kubernetes/
Kubernetes deployment manifests:
- `helm/`: Helm chart for NRDOT-HOST
  - `nrdot/`: Main Helm chart
- `kustomize/`: Kustomize bases and overlays
  - `base/`: Base configuration
  - `overlays/`: Environment-specific overlays (dev, staging, prod)
- `manifests/`: Raw Kubernetes YAML files

### systemd/
SystemD service configurations for Linux systems:
- Service files for different components
- Installation scripts
- System configuration examples

## Quick Start

### Docker Deployment

```bash
# Build and run unified binary
cd docker
docker-compose -f docker-compose.v2.yaml up -d

# Or use the Makefile from root
make docker-unified
```

### Kubernetes Deployment

Using Helm:
```bash
cd kubernetes/helm
helm install nrdot ./nrdot --namespace nrdot-system --create-namespace
```

Using Kustomize:
```bash
cd kubernetes/kustomize
kubectl apply -k overlays/prod
```

### SystemD Installation

```bash
cd systemd
sudo ./install.sh
sudo systemctl start nrdot-supervisor
```

## Configuration

Each deployment method supports different configuration approaches:

- **Docker**: Environment variables and mounted config files
- **Kubernetes**: ConfigMaps, Secrets, and Helm values
- **SystemD**: Configuration files in `/etc/nrdot/`

## Security Considerations

- All deployments bind API to localhost only by default
- TLS/mTLS configuration examples provided
- Secret management best practices documented
- RBAC configurations for Kubernetes

## Monitoring

Each deployment includes:
- Health check endpoints
- Prometheus metrics exposure
- Log aggregation setup
- Example Grafana dashboards

## Troubleshooting

See deployment-specific README files for troubleshooting guides:
- `docker/README.md`
- `kubernetes/README.md`
- `systemd/README.md`