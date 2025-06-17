# Disaster Recovery Runbook

## Overview

This runbook provides step-by-step procedures for recovering the platform in various disaster scenarios.

## Table of Contents

1. [Pre-requisites](#pre-requisites)
2. [Recovery Scenarios](#recovery-scenarios)
3. [RTO and RPO Targets](#rto-and-rpo-targets)
4. [Recovery Procedures](#recovery-procedures)
5. [Validation Steps](#validation-steps)
6. [Contact Information](#contact-information)

## Pre-requisites

### Access Requirements
- AWS Console access with appropriate IAM permissions
- kubectl access to Kubernetes clusters
- Access to backup storage (S3 buckets)
- Database admin credentials
- Monitoring system access

### Tools Required
- AWS CLI configured
- kubectl and helm installed
- Velero CLI installed
- PostgreSQL client tools
- jq for JSON processing

## RTO and RPO Targets

| Component | RPO (Recovery Point Objective) | RTO (Recovery Time Objective) |
|-----------|-------------------------------|-------------------------------|
| Database | 1 hour | 2 hours |
| Application State | 1 hour | 1 hour |
| Kubernetes Config | 24 hours | 30 minutes |
| Persistent Volumes | 1 hour | 2 hours |
| Complete Platform | 1 hour | 4 hours |

## Recovery Scenarios

### Scenario 1: Database Failure

**Symptoms:**
- Application errors related to database connectivity
- Database health checks failing
- No response from RDS endpoint

**Recovery Steps:**

1. **Assess the situation**
   ```bash
   # Check RDS status
   aws rds describe-db-instances --db-instance-identifier platform-prod
   
   # Check CloudWatch metrics
   aws cloudwatch get-metric-statistics \
     --namespace AWS/RDS \
     --metric-name DatabaseConnections \
     --dimensions Name=DBInstanceIdentifier,Value=platform-prod \
     --start-time $(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%S) \
     --end-time $(date -u +%Y-%m-%dT%H:%M:%S) \
     --period 300 \
     --statistics Average
   ```

2. **Attempt automatic failover (if Multi-AZ)**
   ```bash
   aws rds reboot-db-instance \
     --db-instance-identifier platform-prod \
     --force-failover
   ```

3. **Restore from snapshot if needed**
   ```bash
   # List available snapshots
   aws rds describe-db-snapshots \
     --db-instance-identifier platform-prod \
     --query 'DBSnapshots[*].[DBSnapshotIdentifier,SnapshotCreateTime]' \
     --output table
   
   # Restore from snapshot
   aws rds restore-db-instance-from-db-snapshot \
     --db-instance-identifier platform-prod-restored \
     --db-snapshot-identifier <snapshot-id>
   ```

4. **Point-in-time recovery**
   ```bash
   aws rds restore-db-instance-to-point-in-time \
     --source-db-instance-identifier platform-prod \
     --target-db-instance-identifier platform-prod-pitr \
     --restore-time 2024-01-15T03:30:00.000Z
   ```

5. **Update application configuration**
   ```bash
   # Update Kubernetes secret with new endpoint
   kubectl create secret generic db-connection \
     --from-literal=host=platform-prod-restored.xyz.rds.amazonaws.com \
     --from-literal=port=5432 \
     --from-literal=database=platform \
     --dry-run=client -o yaml | kubectl apply -f -
   
   # Restart applications
   kubectl rollout restart deployment -n platform-prod
   ```

### Scenario 2: Kubernetes Cluster Failure

**Symptoms:**
- Unable to access Kubernetes API
- Nodes showing NotReady status
- Control plane components failing

**Recovery Steps:**

1. **Assess cluster health**
   ```bash
   # Check cluster status
   kubectl get nodes
   kubectl get pods -n kube-system
   
   # Check EKS cluster status
   aws eks describe-cluster --name platform-prod
   ```

2. **Restore from Velero backup**
   ```bash
   # List available backups
   velero backup get
   
   # Restore latest backup
   velero restore create --from-backup daily-platform-backup-20240115
   
   # Monitor restore progress
   velero restore describe daily-platform-backup-20240115-restore
   ```

3. **Recreate cluster if needed**
   ```bash
   # Use Terraform to recreate cluster
   cd infrastructure/terraform/environments/prod
   terraform plan
   terraform apply
   
   # Wait for cluster to be ready
   aws eks wait cluster-active --name platform-prod
   
   # Update kubeconfig
   aws eks update-kubeconfig --name platform-prod
   ```

4. **Restore applications**
   ```bash
   # Apply base configurations
   kubectl apply -k infrastructure/kubernetes/base
   
   # Apply production overlays
   kubectl apply -k infrastructure/kubernetes/overlays/prod
   
   # Restore from Velero
   velero restore create full-restore --from-backup weekly-full-backup-latest
   ```

### Scenario 3: Region-wide Outage

**Symptoms:**
- Multiple AWS services unavailable in primary region
- Unable to access any resources in the region

**Recovery Steps:**

1. **Activate DR region**
   ```bash
   export DR_REGION=us-west-2
   export PRIMARY_REGION=us-east-1
   
   # Switch to DR region
   aws configure set region $DR_REGION
   ```

2. **Promote read replica to primary**
   ```bash
   # Promote RDS read replica
   aws rds promote-read-replica \
     --db-instance-identifier platform-prod-replica-dr \
     --region $DR_REGION
   ```

3. **Deploy applications to DR region**
   ```bash
   # Update kubectl context to DR cluster
   aws eks update-kubeconfig \
     --name platform-prod-dr \
     --region $DR_REGION
   
   # Deploy applications
   helm install platform ./helm-charts/platform \
     --namespace platform-prod \
     --values ./helm-charts/platform/values-dr.yaml
   ```

4. **Update DNS to point to DR region**
   ```bash
   # Update Route53 records
   aws route53 change-resource-record-sets \
     --hosted-zone-id Z1234567890ABC \
     --change-batch file://dr-dns-failover.json
   ```

### Scenario 4: Data Corruption

**Symptoms:**
- Inconsistent data in application
- Validation errors
- Reports showing incorrect information

**Recovery Steps:**

1. **Identify corruption scope**
   ```bash
   # Connect to database
   kubectl run -it --rm psql --image=postgres:15 --restart=Never -- \
     psql -h $DB_HOST -U $DB_USER -d $DB_NAME
   
   # Run integrity checks
   SELECT COUNT(*) FROM data_points WHERE created_at > NOW() - INTERVAL '1 day';
   SELECT COUNT(*) FROM processed_results WHERE status = 'failed';
   ```

2. **Restore specific tables**
   ```bash
   # Restore from logical backup
   pg_restore -h $DB_HOST -U $DB_USER -d $DB_NAME \
     --table=data_points \
     --clean \
     /backups/platform-prod-20240115.dump
   ```

3. **Replay data from source**
   ```bash
   # Trigger data replay job
   kubectl create job data-replay-$(date +%s) \
     --from=cronjob/data-replay \
     -n platform-prod
   ```

## Validation Steps

### Post-Recovery Validation

1. **System Health Checks**
   ```bash
   # Check all pods are running
   kubectl get pods -n platform-prod | grep -v Running
   
   # Check endpoints
   kubectl get endpoints -n platform-prod
   
   # Test API endpoints
   curl -f https://api.platform.example.com/health || echo "API Gateway unhealthy"
   ```

2. **Data Integrity Validation**
   ```sql
   -- Check record counts
   SELECT 
     (SELECT COUNT(*) FROM data_points) as data_points_count,
     (SELECT COUNT(*) FROM processed_results) as processed_results_count,
     (SELECT COUNT(*) FROM audit_logs) as audit_logs_count;
   
   -- Check for gaps in data
   SELECT DATE(created_at), COUNT(*) 
   FROM data_points 
   WHERE created_at > NOW() - INTERVAL '7 days'
   GROUP BY DATE(created_at)
   ORDER BY DATE(created_at);
   ```

3. **Performance Validation**
   ```bash
   # Run smoke tests
   ./scripts/smoke-tests.sh
   
   # Check response times
   for i in {1..10}; do
     time curl -s https://api.platform.example.com/api/v1/health > /dev/null
   done
   ```

4. **Monitoring Validation**
   ```bash
   # Check Prometheus targets
   curl http://prometheus.monitoring.svc.cluster.local:9090/api/v1/targets | jq '.data.activeTargets[] | {job: .job, health: .health}'
   
   # Verify metrics collection
   curl http://prometheus.monitoring.svc.cluster.local:9090/api/v1/query?query=up | jq '.data.result[] | {job: .metric.job, value: .value[1]}'
   ```

## Recovery Automation Scripts

### Script: quick-recovery.sh
```bash
#!/bin/bash
set -euo pipefail

ENVIRONMENT=${1:-prod}
SCENARIO=${2:-unknown}

echo "Starting recovery for environment: $ENVIRONMENT"
echo "Scenario: $SCENARIO"

case $SCENARIO in
  "database")
    ./scripts/recover-database.sh $ENVIRONMENT
    ;;
  "kubernetes")
    ./scripts/recover-kubernetes.sh $ENVIRONMENT
    ;;
  "full")
    ./scripts/full-platform-recovery.sh $ENVIRONMENT
    ;;
  *)
    echo "Unknown scenario. Please specify: database, kubernetes, or full"
    exit 1
    ;;
esac

# Run validation
./scripts/validate-recovery.sh $ENVIRONMENT
```

## Communication Plan

### During Incident

1. **Initial Alert** (Within 5 minutes)
   - Send to: #platform-incidents Slack channel
   - Email: platform-oncall@example.com
   - Page: On-call engineer via PagerDuty

2. **Status Updates** (Every 30 minutes)
   - Update status page
   - Post in #platform-incidents
   - Email stakeholders if critical

3. **Resolution Notice**
   - All-clear in all communication channels
   - Post-mortem scheduled within 48 hours

### Escalation Path

1. L1: On-call Engineer (0-15 mins)
2. L2: Platform Team Lead (15-30 mins)
3. L3: Engineering Manager (30-60 mins)
4. L4: VP of Engineering (60+ mins)

## Contact Information

| Role | Name | Phone | Email |
|------|------|-------|-------|
| Platform Lead | John Doe | +1-555-0123 | john.doe@example.com |
| Database Admin | Jane Smith | +1-555-0124 | jane.smith@example.com |
| Security Lead | Bob Johnson | +1-555-0125 | bob.johnson@example.com |
| AWS TAM | Alice Brown | +1-555-0126 | alice@aws.com |

## Testing Schedule

- **Monthly**: Table-top exercises
- **Quarterly**: Partial failover test
- **Annually**: Full DR test with region failover

## Lessons Learned

Document lessons learned from each incident:

| Date | Incident | Root Cause | Improvements |
|------|----------|------------|--------------|
| 2024-01-01 | DB Failover | Network partition | Implemented connection retry logic |
| 2023-12-15 | K8s Upgrade | Version mismatch | Added pre-upgrade validation |

---

**Last Updated**: 2024-01-15
**Next Review**: 2024-04-15
**Owner**: Platform Team