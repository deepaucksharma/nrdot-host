#!/bin/bash
# User data script for NRDOT-HOST installation on EC2

set -e

# Variables from Terraform
NEW_RELIC_LICENSE_KEY="${new_relic_license_key}"
NEW_RELIC_API_KEY="${new_relic_api_key}"
ENVIRONMENT="${environment}"
NRDOT_VERSION="${nrdot_version}"
INSTANCE_ID=$(ec2-metadata --instance-id | cut -d " " -f 2)
REGION=$(ec2-metadata --availability-zone | cut -d " " -f 2 | sed 's/[a-z]$//')

# Update system
yum update -y

# Install dependencies
yum install -y \
  wget \
  curl \
  jq \
  systemd \
  procps-ng \
  net-tools \
  lsof

# Create NRDOT user
useradd -r -s /bin/false -d /var/lib/nrdot nrdot

# Create directories
mkdir -p /etc/nrdot /var/lib/nrdot /var/log/nrdot
chown -R nrdot:nrdot /var/lib/nrdot /var/log/nrdot

# Download NRDOT-HOST
if [ "$NRDOT_VERSION" = "latest" ]; then
  DOWNLOAD_URL=$(curl -s https://api.github.com/repos/newrelic/nrdot-host/releases/latest \
    | jq -r '.assets[] | select(.name | contains("linux_amd64")) | .browser_download_url')
else
  DOWNLOAD_URL="https://github.com/newrelic/nrdot-host/releases/download/$NRDOT_VERSION/nrdot-host_linux_amd64"
fi

wget -q -O /usr/local/bin/nrdot-host "$DOWNLOAD_URL"
chmod +x /usr/local/bin/nrdot-host

# Download helper
HELPER_URL=$(echo "$DOWNLOAD_URL" | sed 's/nrdot-host/nrdot-helper/')
wget -q -O /usr/local/bin/nrdot-helper "$HELPER_URL"
chown root:nrdot /usr/local/bin/nrdot-helper
chmod 4750 /usr/local/bin/nrdot-helper

# Create configuration
cat > /etc/nrdot/config.yaml <<EOF
license_key: "\${NEW_RELIC_LICENSE_KEY}"

service:
  name: "$INSTANCE_ID"
  environment: "$ENVIRONMENT"
  attributes:
    cloud.provider: "aws"
    cloud.region: "$REGION"
    cloud.instance.id: "$INSTANCE_ID"
    
auto_config:
  enabled: true
  scan_interval: 5m
  remote_config:
    enabled: true
    api_key: "\${NEW_RELIC_API_KEY}"
    
processes:
  enabled: true
  top_n: 50
  interval: 60s
  
data_dir: /var/lib/nrdot
log_dir: /var/log/nrdot

logging:
  level: info
  format: json
  
api:
  enabled: true
  listen_addr: "0.0.0.0:8080"
  
# AWS-specific receivers
receivers:
  hostmetrics:
    collection_interval: 60s
    scrapers:
      cpu:
        metrics:
          system.cpu.utilization:
            enabled: true
      memory:
        metrics:
          system.memory.utilization:
            enabled: true
      disk:
        metrics:
          system.disk.io:
            enabled: true
      filesystem:
        metrics:
          system.filesystem.utilization:
            enabled: true
      network:
        metrics:
          system.network.io:
            enabled: true
      load:
        metrics:
          system.cpu.load_average.1m:
            enabled: true
            
processors:
  nrsecurity:
    enabled: true
  nrenrich:
    host_metadata: true
    cloud_detection: true
    aws_metadata:
      enabled: true
      imds_version: 2
  batch:
    timeout: 10s
    send_batch_size: 1000
  memory_limiter:
    check_interval: 1s
    limit_mib: 1024
    spike_limit_mib: 256
    
exporters:
  otlp/newrelic:
    endpoint: "otlp.nr-data.net:4317"
    headers:
      api-key: "\${NEW_RELIC_LICENSE_KEY}"
    compression: gzip
    
service:
  pipelines:
    metrics:
      receivers: [hostmetrics]
      processors: [nrsecurity, nrenrich, batch, memory_limiter]
      exporters: [otlp/newrelic]
EOF

# Set permissions
chown nrdot:nrdot /etc/nrdot/config.yaml
chmod 640 /etc/nrdot/config.yaml

# Create environment file
cat > /etc/nrdot/nrdot.env <<EOF
NEW_RELIC_LICENSE_KEY=$NEW_RELIC_LICENSE_KEY
NEW_RELIC_API_KEY=$NEW_RELIC_API_KEY
EOF

chown nrdot:nrdot /etc/nrdot/nrdot.env
chmod 600 /etc/nrdot/nrdot.env

# Create systemd service
cat > /etc/systemd/system/nrdot-host.service <<'EOF'
[Unit]
Description=NRDOT-HOST - New Relic OpenTelemetry Distribution
After=network-online.target
Wants=network-online.target

[Service]
Type=notify
User=nrdot
Group=nrdot
EnvironmentFile=/etc/nrdot/nrdot.env
WorkingDirectory=/var/lib/nrdot
ExecStart=/usr/local/bin/nrdot-host run --config=/etc/nrdot/config.yaml
Restart=always
RestartSec=10

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/nrdot /var/log/nrdot
AmbientCapabilities=CAP_SYS_PTRACE CAP_DAC_READ_SEARCH

# Resource limits
LimitNOFILE=65536
CPUQuota=200%
MemoryMax=2G

[Install]
WantedBy=multi-user.target
EOF

# Enable and start service
systemctl daemon-reload
systemctl enable nrdot-host
systemctl start nrdot-host

# Wait for service to be ready
sleep 10

# Run initial discovery
/usr/local/bin/nrdot-host discover --save

# Set up CloudWatch logs (optional)
if command -v amazon-cloudwatch-agent-ctl &> /dev/null; then
  cat > /opt/aws/amazon-cloudwatch-agent/etc/nrdot-logs.json <<EOF
{
  "logs": {
    "logs_collected": {
      "files": {
        "collect_list": [
          {
            "file_path": "/var/log/nrdot/nrdot.log",
            "log_group_name": "/aws/ec2/nrdot-host",
            "log_stream_name": "{instance_id}",
            "timezone": "UTC"
          }
        ]
      }
    }
  }
}
EOF
  
  amazon-cloudwatch-agent-ctl \
    -a append-config \
    -m ec2 \
    -c file:/opt/aws/amazon-cloudwatch-agent/etc/nrdot-logs.json \
    -s
fi

# Create completion marker
touch /var/lib/nrdot/.installation_complete

echo "NRDOT-HOST installation complete!"