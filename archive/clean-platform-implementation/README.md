# Clean Platform Implementation

A streamlined microservices platform built on AWS EKS with Infrastructure as Code.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                   Load Balancer (ALB)                   │
└─────────────────────────┬───────────────────────────────┘
                          │
┌─────────────────────────┴───────────────────────────────┐
│                    API Gateway                          │
│                 (NGINX / Kong)                          │
└─────────┬────────────────────────────┬─────────────────┘
          │                            │
┌─────────┴──────────┐      ┌─────────┴──────────┐
│  Data Collector    │      │  Data Processor    │
│   (Python/Go)      │      │   (Python/Go)      │
└─────────┬──────────┘      └─────────┬──────────┘
          │                            │
          └────────────┬───────────────┘
                       │
              ┌────────┴────────┐
              │   PostgreSQL    │
              │   (RDS Aurora)  │
              └─────────────────┘
```

## Quick Start

### Prerequisites
- AWS CLI configured
- Terraform >= 1.5.0
- kubectl >= 1.27
- Docker
- Python 3.9+

### Local Development
```bash
# Start local services
./scripts/local-dev.sh

# Run tests
make test

# Build services
make build
```

### Deploy to AWS
```bash
# Deploy infrastructure
cd infrastructure/terraform/environments/dev
terraform init
terraform apply

# Deploy services
kubectl apply -k infrastructure/kubernetes/overlays/dev
```

## Project Structure

```
.
├── services/                # Microservices
│   ├── api-gateway/        # API routing and authentication
│   ├── data-collector/     # Data ingestion service
│   └── data-processor/     # Stream processing service
├── infrastructure/         # Infrastructure as Code
│   ├── terraform/          # AWS infrastructure
│   └── kubernetes/         # K8s manifests
├── scripts/               # Automation scripts
├── docs/                  # Documentation
└── tests/                 # Integration tests
```

## Services

### API Gateway
- Routes requests to backend services
- Handles authentication and rate limiting
- Provides API documentation

### Data Collector
- RESTful API for data ingestion
- Validates and queues data for processing
- Scales horizontally based on load

### Data Processor
- Processes queued data asynchronously
- Writes to PostgreSQL database
- Handles retries and error recovery

## Infrastructure

### AWS Resources
- **VPC**: Multi-AZ with public/private subnets
- **EKS**: Managed Kubernetes cluster
- **RDS**: Aurora PostgreSQL
- **ElastiCache**: Redis for caching
- **S3**: Object storage
- **CloudWatch**: Logging and monitoring

### Kubernetes Resources
- **Deployments**: Service deployments with HPA
- **Services**: Load balancing and service discovery
- **Ingress**: External traffic routing
- **ConfigMaps**: Configuration management
- **Secrets**: Sensitive data management

## Development Workflow

1. **Feature Development**
   ```bash
   git checkout -b feature/your-feature
   # Make changes
   make test
   git commit -m "Add feature"
   git push origin feature/your-feature
   ```

2. **CI/CD Pipeline**
   - GitHub Actions runs tests
   - Builds and pushes Docker images
   - Deploys to dev environment
   - Manual approval for staging/prod

3. **Monitoring**
   - Prometheus metrics
   - Grafana dashboards
   - CloudWatch logs
   - Alerts via SNS

## Configuration

### Environment Variables
```bash
# Database
DATABASE_URL=postgresql://user:pass@host:5432/db

# Redis
REDIS_URL=redis://host:6379

# AWS
AWS_REGION=us-east-1
AWS_ACCOUNT_ID=123456789012

# Application
LOG_LEVEL=info
PORT=8080
```

### Secrets Management
Using AWS Secrets Manager:
```bash
aws secretsmanager create-secret \
  --name platform/dev/db-password \
  --secret-string "your-password"
```

## Deployment

### Development
```bash
make deploy-dev
```

### Staging
```bash
make deploy-staging
```

### Production
```bash
make deploy-prod
```

## Monitoring and Alerts

### Metrics
- Request rate, error rate, duration (RED)
- CPU, memory, network usage
- Database connections and query performance
- Queue depth and processing rate

### Dashboards
Access Grafana at: http://grafana.your-domain.com

### Alerts
- Service availability < 99.9%
- Error rate > 1%
- Response time > 500ms (p99)
- CPU/Memory > 80%

## Security

- All traffic encrypted with TLS
- Network policies for pod-to-pod communication
- IAM roles for service accounts
- Regular security scanning
- Secrets rotation every 90 days

## Cost Management

- Auto-scaling based on metrics
- Spot instances for non-critical workloads
- Reserved instances for baseline capacity
- S3 lifecycle policies
- Regular cost reviews

## Support

- Documentation: [docs/](docs/)
- Issues: GitHub Issues
- Slack: #platform-support

## License

MIT License - see LICENSE file for details