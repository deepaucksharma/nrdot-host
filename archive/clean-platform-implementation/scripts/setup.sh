#!/bin/bash
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Project configuration
PROJECT_NAME="${PROJECT_NAME:-platform}"
ENVIRONMENT="${ENVIRONMENT:-dev}"
AWS_REGION="${AWS_REGION:-us-east-1}"

echo -e "${GREEN}Setting up ${PROJECT_NAME} environment...${NC}"

# Check prerequisites
echo -e "${YELLOW}Checking prerequisites...${NC}"

check_command() {
    if ! command -v $1 &> /dev/null; then
        echo -e "${RED}$1 is required but not installed.${NC}"
        exit 1
    else
        echo -e "${GREEN}✓ $1 found${NC}"
    fi
}

check_command terraform
check_command kubectl
check_command aws
check_command docker
check_command python3

# Check AWS credentials
if ! aws sts get-caller-identity &> /dev/null; then
    echo -e "${RED}AWS credentials not configured. Run 'aws configure'${NC}"
    exit 1
else
    echo -e "${GREEN}✓ AWS credentials configured${NC}"
fi

# Initialize Terraform
echo -e "${YELLOW}Initializing Terraform...${NC}"
cd infrastructure/terraform/environments/${ENVIRONMENT}
terraform init

# Create namespace
echo -e "${YELLOW}Creating Kubernetes namespace...${NC}"
kubectl create namespace ${PROJECT_NAME}-${ENVIRONMENT} --dry-run=client -o yaml | kubectl apply -f -

# Create secrets
echo -e "${YELLOW}Creating secrets...${NC}"

# Generate random passwords if not exists
if ! aws secretsmanager describe-secret --secret-id ${PROJECT_NAME}/${ENVIRONMENT}/db-password &> /dev/null; then
    DB_PASSWORD=$(openssl rand -base64 32)
    aws secretsmanager create-secret \
        --name ${PROJECT_NAME}/${ENVIRONMENT}/db-password \
        --secret-string "${DB_PASSWORD}" \
        --region ${AWS_REGION}
    echo -e "${GREEN}✓ Database password created${NC}"
else
    echo -e "${GREEN}✓ Database password already exists${NC}"
fi

if ! aws secretsmanager describe-secret --secret-id ${PROJECT_NAME}/${ENVIRONMENT}/redis-password &> /dev/null; then
    REDIS_PASSWORD=$(openssl rand -base64 32)
    aws secretsmanager create-secret \
        --name ${PROJECT_NAME}/${ENVIRONMENT}/redis-password \
        --secret-string "${REDIS_PASSWORD}" \
        --region ${AWS_REGION}
    echo -e "${GREEN}✓ Redis password created${NC}"
else
    echo -e "${GREEN}✓ Redis password already exists${NC}"
fi

# Create ECR repositories
echo -e "${YELLOW}Creating ECR repositories...${NC}"
SERVICES=("api-gateway" "data-collector" "data-processor")

for service in "${SERVICES[@]}"; do
    if ! aws ecr describe-repositories --repository-names ${PROJECT_NAME}/${service} &> /dev/null; then
        aws ecr create-repository \
            --repository-name ${PROJECT_NAME}/${service} \
            --region ${AWS_REGION} \
            --image-scanning-configuration scanOnPush=true
        echo -e "${GREEN}✓ ECR repository ${service} created${NC}"
    else
        echo -e "${GREEN}✓ ECR repository ${service} already exists${NC}"
    fi
done

# Setup Python virtual environment
echo -e "${YELLOW}Setting up Python environment...${NC}"
python3 -m venv venv
source venv/bin/activate
pip install -r requirements-dev.txt

# Create local development files
echo -e "${YELLOW}Creating local development configuration...${NC}"
cat > .env.local << EOF
# Local development environment variables
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/${PROJECT_NAME}
REDIS_URL=redis://localhost:6379
AWS_REGION=${AWS_REGION}
ENVIRONMENT=local
LOG_LEVEL=debug
EOF

# Create docker-compose for local development
cat > docker-compose.local.yml << EOF
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: ${PROJECT_NAME}
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

volumes:
  postgres_data:
  redis_data:
EOF

echo -e "${GREEN}✓ Local development files created${NC}"

# Summary
echo -e "\n${GREEN}Setup completed successfully!${NC}"
echo -e "\nNext steps:"
echo -e "1. Review and apply Terraform configuration:"
echo -e "   ${YELLOW}cd infrastructure/terraform/environments/${ENVIRONMENT} && terraform plan${NC}"
echo -e "2. Start local development environment:"
echo -e "   ${YELLOW}docker-compose -f docker-compose.local.yml up -d${NC}"
echo -e "3. Build and deploy services:"
echo -e "   ${YELLOW}make build && make deploy-${ENVIRONMENT}${NC}"
echo -e "\nFor more information, see docs/README.md"