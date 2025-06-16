# guardian-fleet-infra

Infrastructure as Code for the Guardian Fleet continuous validation platform.

## Overview
Terraform modules and Ansible playbooks for provisioning and managing the Guardian Fleet - a 24/7 testing environment for NRDOT-Host.

## Components
- AWS/GCP/Azure fleet provisioning
- Heterogeneous OS distribution
- Workload simulator deployment
- Monitoring setup
- Cost optimization

## Fleet Composition
```hcl
# 70+ instances across:
- 20x baseline_nria (New Relic Agent)
- 20x baseline_otel (Vanilla OTel)
- 20x nrdot_stable
- 10x nrdot_canary
```

## Features
- Multi-cloud support
- Automated provisioning
- A/B testing infrastructure
- Performance baselines

## Integration
- Deploys `nrdot-workload-simulators`
- Monitored by `nrdot-health-analyzer`
