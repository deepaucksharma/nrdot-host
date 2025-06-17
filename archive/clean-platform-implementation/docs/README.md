# Platform Clean Implementation

## Overview

This is a clean, cloud-native implementation of a team-level platform that provides:
- Container orchestration with Kubernetes (EKS)
- CI/CD with GitHub Actions
- Data collection and processing services
- Monitoring with Prometheus and Grafana
- Infrastructure as Code with Terraform
- Security-first design with network policies and pod security standards

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   API Gateway   │────▶│ Data Collector  │────▶│     Redis       │
│    (Nginx)      │     │   (Python)      │     │    (Cache)      │
└─────────────────┘     └─────────────────┘     └─────────────────┘
         │                       │                         │
         │              ┌─────────────────┐               │
         └─────────────▶│ Data Processor  │───────────────┘
                        │  (Python/Celery) │
                        └─────────────────┘
                                 │
                        ┌─────────────────┐
                        │   PostgreSQL    │
                        │      (RDS)      │
                        └─────────────────┘
```

## Project Structure

```
clean-platform-implementation/
├── services/                    # Microservices
│   ├── api-gateway/            # Nginx-based API gateway
│   ├── data-collector/         # Data collection service
│   └── data-processor/         # Data processing service
├── infrastructure/             # Infrastructure configuration
│   ├── terraform/              # Terraform modules and environments
│   ├── kubernetes/             # Kubernetes manifests
│   ├── monitoring/             # Prometheus/Grafana configuration
│   └── security/               # Security policies
├── database/                   # Database schemas and migrations
├── scripts/                    # Utility scripts
├── tests/                      # Test suites
└── docs/                       # Documentation
```

## Quick Start

### Prerequisites

- AWS Account with appropriate permissions
- Terraform >= 1.0
- kubectl >= 1.21
- Docker >= 20.10
- Python >= 3.9
- Make

### Initial Setup

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd clean-platform-implementation
   ```

2. **Configure AWS credentials**
   ```bash
   aws configure
   # Or use environment variables
   export AWS_ACCESS_KEY_ID=<your-key>
   export AWS_SECRET_ACCESS_KEY=<your-secret>
   export AWS_REGION=us-east-1
   ```

3. **Run setup script**
   ```bash
   ./scripts/setup.sh
   ```

4. **Deploy infrastructure**
   ```bash
   cd infrastructure/terraform/environments/dev
   terraform init
   terraform plan
   terraform apply
   ```

5. **Deploy services**
   ```bash
   make deploy-dev
   ```

## Development

### Local Development

1. **Start local dependencies**
   ```bash
   docker-compose -f docker-compose.local.yml up -d
   ```

2. **Activate Python environment**
   ```bash
   source venv/bin/activate
   ```

3. **Run services locally**
   ```bash
   # Terminal 1: API Gateway
   cd services/api-gateway
   docker build -t api-gateway:local .
   docker run -p 8080:8080 api-gateway:local

   # Terminal 2: Data Collector
   cd services/data-collector
   python app.py

   # Terminal 3: Data Processor
   cd services/data-processor
   python app.py
   ```

### Running Tests

```bash
# Unit tests
make test-unit

# Integration tests
make test-integration

# End-to-end tests
make test-e2e

# All tests
make test
```

### Code Quality

```bash
# Linting
make lint

# Type checking
make typecheck

# Security scanning
make security-scan
```

## Deployment

### Environment Structure

- **Development** (`dev`) - For active development and testing
- **Staging** (`staging`) - Pre-production environment
- **Production** (`prod`) - Live production environment

### Deployment Process

1. **Development deployment** (automatic on push to `develop`)
   ```bash
   git push origin develop
   ```

2. **Staging deployment** (automatic on push to `main`)
   ```bash
   git push origin main
   ```

3. **Production deployment** (manual approval required)
   - Push to `main` branch
   - Wait for staging deployment
   - Approve production deployment in GitHub Actions

### Manual Deployment

```bash
# Deploy specific service to specific environment
make deploy-service SERVICE=data-collector ENV=staging

# Deploy all services
make deploy-all ENV=prod
```

## Configuration

### Environment Variables

Each service uses environment variables for configuration. See individual service READMEs for details.

Common variables:
- `ENVIRONMENT` - Environment name (dev/staging/prod)
- `LOG_LEVEL` - Logging level (debug/info/warning/error)
- `DATABASE_URL` - PostgreSQL connection string
- `REDIS_URL` - Redis connection string

### Secrets Management

Secrets are managed through AWS Secrets Manager and injected at runtime.

```bash
# Create secret
aws secretsmanager create-secret \
  --name platform/prod/api-keys \
  --secret-string '{"key": "value"}'

# Update secret
aws secretsmanager update-secret \
  --secret-id platform/prod/api-keys \
  --secret-string '{"key": "new-value"}'
```

## Monitoring

### Metrics

All services expose Prometheus metrics at `/metrics` endpoint.

Key metrics:
- Request rate and latency
- Error rates
- Resource utilization
- Business-specific metrics

### Dashboards

Grafana dashboards are available at: `https://grafana.<environment>.platform.example.com`

Pre-configured dashboards:
- Platform Overview
- Service Health
- Infrastructure Metrics
- Business Metrics

### Alerts

Alerts are configured in Prometheus and sent to:
- Email (all alerts)
- Slack (warnings and above)
- PagerDuty (critical only)

## Security

### Network Security

- All traffic encrypted with TLS
- Network policies restrict pod-to-pod communication
- Private subnets for services
- WAF rules on load balancers

### Pod Security

- Non-root containers
- Read-only root filesystems
- Resource limits enforced
- Security contexts applied

### Secret Management

- Secrets stored in AWS Secrets Manager
- Encrypted at rest with KMS
- Rotated regularly
- Never committed to git

### Compliance

- SOC2 compliant infrastructure
- GDPR-ready data handling
- Audit logging enabled
- Regular security scanning

## Troubleshooting

### Common Issues

1. **Pod not starting**
   ```bash
   kubectl describe pod <pod-name> -n platform-<env>
   kubectl logs <pod-name> -n platform-<env>
   ```

2. **Service unavailable**
   ```bash
   # Check service endpoints
   kubectl get endpoints -n platform-<env>
   
   # Check ingress
   kubectl describe ingress -n platform-<env>
   ```

3. **Database connection issues**
   ```bash
   # Check security groups
   aws ec2 describe-security-groups --group-ids <sg-id>
   
   # Test connection
   kubectl run -it --rm debug --image=postgres:15 --restart=Never -- psql <connection-string>
   ```

### Debug Mode

Enable debug mode for verbose logging:
```bash
kubectl set env deployment/<service> LOG_LEVEL=debug -n platform-<env>
```

### Support

- Documentation: `/docs`
- Issues: GitHub Issues
- Slack: #platform-support

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Run `make pre-commit`
6. Create a pull request

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

## License

Copyright (c) 2024 Platform Team. All rights reserved.