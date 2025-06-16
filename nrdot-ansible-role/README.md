# nrdot-ansible-role

Ansible role for deploying and managing NRDOT-Host across infrastructure.

## Overview
Production-ready Ansible role for automated deployment, configuration, and maintenance of NRDOT-Host.

## Features
- Multi-OS support
- Idempotent operations
- Rolling deployments
- Configuration management
- Health verification

## Variables
```yaml
# Example playbook
- hosts: all
  roles:
    - role: nrdot-ansible-role
      vars:
        nrdot_version: "1.0.0"
        nrdot_license_key: "{{ vault_nr_license }}"
        nrdot_process_monitoring: true
```

## Platforms
- RHEL/CentOS 7+
- Ubuntu 20.04+
- Windows Server 2019+

## Integration
- Installs from `nrdot-packaging`
- Used by `guardian-fleet-infra`
