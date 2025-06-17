#!/bin/bash
set -euo pipefail

# Setup script for ELK stack logging infrastructure
# Usage: ./setup-logging.sh [environment]

ENVIRONMENT="${1:-prod}"
NAMESPACE="logging"
ELASTIC_VERSION="8.11.0"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}Setting up ELK stack for environment: ${ENVIRONMENT}${NC}"

# Create namespace
echo -e "${YELLOW}Creating namespace...${NC}"
kubectl create namespace ${NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -

# Generate certificates
echo -e "${YELLOW}Generating certificates...${NC}"
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: elastic-certificates
  namespace: ${NAMESPACE}
type: Opaque
data:
  elastic-certificates.p12: $(docker run --rm -i \
    docker.elastic.co/elasticsearch/elasticsearch:${ELASTIC_VERSION} \
    bin/elasticsearch-certutil cert --silent --pem -out - | \
    base64 -w0)
EOF

# Generate passwords
echo -e "${YELLOW}Generating passwords...${NC}"
ELASTIC_PASSWORD=$(openssl rand -base64 32)
KIBANA_PASSWORD=$(openssl rand -base64 32)
KIBANA_ENCRYPTION_KEY=$(openssl rand -base64 32)

kubectl create secret generic elastic-credentials \
  --namespace=${NAMESPACE} \
  --from-literal=password="${ELASTIC_PASSWORD}" \
  --from-literal=kibana-password="${KIBANA_PASSWORD}" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl create secret generic kibana-encryption-key \
  --namespace=${NAMESPACE} \
  --from-literal=key="${KIBANA_ENCRYPTION_KEY}" \
  --dry-run=client -o yaml | kubectl apply -f -

# Install Elasticsearch
echo -e "${YELLOW}Installing Elasticsearch...${NC}"
helm repo add elastic https://helm.elastic.co
helm repo update

helm upgrade --install elasticsearch elastic/elasticsearch \
  --namespace ${NAMESPACE} \
  --version 8.5.1 \
  --values infrastructure/logging/elasticsearch-values.yaml \
  --set imageTag=${ELASTIC_VERSION} \
  --wait

# Wait for Elasticsearch to be ready
echo -e "${YELLOW}Waiting for Elasticsearch to be ready...${NC}"
kubectl wait --namespace=${NAMESPACE} \
  --for=condition=ready pod \
  --selector=app=elasticsearch-master \
  --timeout=600s

# Create ILM policies
echo -e "${YELLOW}Creating ILM policies...${NC}"
kubectl exec -n ${NAMESPACE} elasticsearch-master-0 -- curl -s -k \
  -u elastic:"${ELASTIC_PASSWORD}" \
  -X PUT "https://localhost:9200/_ilm/policy/platform-ilm-policy" \
  -H "Content-Type: application/json" \
  -d '{
    "policy": {
      "phases": {
        "hot": {
          "min_age": "0ms",
          "actions": {
            "rollover": {
              "max_age": "1d",
              "max_size": "50GB"
            },
            "set_priority": {
              "priority": 100
            }
          }
        },
        "warm": {
          "min_age": "7d",
          "actions": {
            "forcemerge": {
              "max_num_segments": 1
            },
            "shrink": {
              "number_of_shards": 1
            },
            "set_priority": {
              "priority": 50
            }
          }
        },
        "cold": {
          "min_age": "30d",
          "actions": {
            "set_priority": {
              "priority": 0
            }
          }
        },
        "delete": {
          "min_age": "90d",
          "actions": {
            "delete": {}
          }
        }
      }
    }
  }'

# Install Logstash
echo -e "${YELLOW}Installing Logstash...${NC}"
kubectl apply -f infrastructure/logging/logstash-config.yaml

helm upgrade --install logstash elastic/logstash \
  --namespace ${NAMESPACE} \
  --version 8.5.1 \
  --set imageTag=${ELASTIC_VERSION} \
  --set replicas=3 \
  --set resources.requests.cpu=1000m \
  --set resources.requests.memory=2Gi \
  --set resources.limits.cpu=2000m \
  --set resources.limits.memory=4Gi \
  --wait

# Install Kibana
echo -e "${YELLOW}Installing Kibana...${NC}"
helm upgrade --install kibana elastic/kibana \
  --namespace ${NAMESPACE} \
  --version 8.5.1 \
  --values infrastructure/logging/kibana-values.yaml \
  --set imageTag=${ELASTIC_VERSION} \
  --wait

# Create Kibana user
echo -e "${YELLOW}Creating Kibana system user...${NC}"
kubectl exec -n ${NAMESPACE} elasticsearch-master-0 -- curl -s -k \
  -u elastic:"${ELASTIC_PASSWORD}" \
  -X POST "https://localhost:9200/_security/user/kibana_system/_password" \
  -H "Content-Type: application/json" \
  -d "{\"password\": \"${KIBANA_PASSWORD}\"}"

# Deploy Filebeat
echo -e "${YELLOW}Deploying Filebeat...${NC}"
kubectl apply -f infrastructure/logging/filebeat-daemonset.yaml

# Create index patterns in Kibana
echo -e "${YELLOW}Creating Kibana index patterns...${NC}"
sleep 30  # Wait for Kibana to be fully ready

KIBANA_URL="https://$(kubectl get ingress -n ${NAMESPACE} kibana-kibana -o jsonpath='{.spec.rules[0].host}')"

# Create index patterns
for pattern in "platform-k8s-*" "platform-app-*" "platform-metrics-*"; do
  curl -k -u elastic:"${ELASTIC_PASSWORD}" \
    -X POST "${KIBANA_URL}/api/saved_objects/index-pattern" \
    -H "kbn-xsrf: true" \
    -H "Content-Type: application/json" \
    -d "{
      \"attributes\": {
        \"title\": \"${pattern}\",
        \"timeFieldName\": \"@timestamp\"
      }
    }"
done

# Create sample dashboards
echo -e "${YELLOW}Creating sample dashboards...${NC}"
cat > /tmp/platform-dashboard.json <<EOF
{
  "version": "8.11.0",
  "objects": [
    {
      "id": "platform-overview",
      "type": "dashboard",
      "attributes": {
        "title": "Platform Overview",
        "hits": 0,
        "description": "Main platform monitoring dashboard",
        "panelsJSON": "[{\"gridData\":{\"x\":0,\"y\":0,\"w\":24,\"h\":15},\"type\":\"visualization\",\"id\":\"log-levels\"}]",
        "timeRestore": false,
        "kibanaSavedObjectMeta": {
          "searchSourceJSON": "{\"query\":{\"query\":\"\",\"language\":\"kuery\"},\"filter\":[]}"
        }
      }
    }
  ]
}
EOF

curl -k -u elastic:"${ELASTIC_PASSWORD}" \
  -X POST "${KIBANA_URL}/api/saved_objects/_import" \
  -H "kbn-xsrf: true" \
  -F file=@/tmp/platform-dashboard.json

# Output summary
echo -e "\n${GREEN}ELK Stack setup completed!${NC}"
echo -e "\nAccess URLs:"
echo -e "Kibana: ${KIBANA_URL}"
echo -e "Elasticsearch: https://elasticsearch-master.${NAMESPACE}.svc.cluster.local:9200"
echo -e "\nCredentials:"
echo -e "Username: elastic"
echo -e "Password: ${ELASTIC_PASSWORD}"
echo -e "\nTo access Kibana locally:"
echo -e "kubectl port-forward -n ${NAMESPACE} svc/kibana-kibana 5601:5601"

# Save credentials securely
echo -e "\n${YELLOW}Saving credentials to AWS Secrets Manager...${NC}"
aws secretsmanager create-secret \
  --name "platform/${ENVIRONMENT}/elk-credentials" \
  --secret-string "{
    \"elastic_password\": \"${ELASTIC_PASSWORD}\",
    \"kibana_password\": \"${KIBANA_PASSWORD}\",
    \"kibana_encryption_key\": \"${KIBANA_ENCRYPTION_KEY}\",
    \"kibana_url\": \"${KIBANA_URL}\"
  }" || echo "Secret already exists"

echo -e "\n${GREEN}Setup complete!${NC}"