# Security Policy

## Reporting Security Vulnerabilities

The NRDOT team takes security seriously. We appreciate your efforts to responsibly disclose your findings.

### Where to Report

**DO NOT** report security vulnerabilities through public GitHub issues.

Instead, please report them via one of these methods:

1. **Email**: security@newrelic.com
2. **New Relic Security Portal**: https://newrelic.com/security
3. **GitHub Security Advisories**: [Report a vulnerability](https://github.com/deepaucksharma/nrdot-host/security/advisories/new)

### What to Include

Please include:

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fixes (if any)
- Your contact information

### Response Timeline

- **Initial Response**: Within 48 hours
- **Status Update**: Within 5 business days
- **Resolution Target**: Based on severity

## Security Features

### Built-in Security

NRDOT-HOST includes several security features by default:

#### 1. Secret Redaction
Automatically detects and redacts:
- Passwords and API keys
- Credit card numbers
- Social Security Numbers
- Private keys and certificates
- Connection strings

#### 2. Secure Defaults
- TLS 1.2+ required for all connections
- Local-only API server binding
- Minimal privilege execution
- No default passwords

#### 3. Process Isolation
- Privileged operations use separate helper binary
- Collector runs with minimal permissions
- Configuration validation before execution

### Security Best Practices

#### Installation
```bash
# Verify package signatures
gpg --verify nrdot-host.rpm.sig nrdot-host.rpm

# Set secure permissions
chmod 600 /etc/nrdot/config.yaml
chown nrdot:nrdot /etc/nrdot/config.yaml
```

#### Configuration
```yaml
# Enable all security features
security:
  redact_secrets: true
  redaction_rules:
    - pattern: "(?i)api[_-]?key"
      replacement: "[REDACTED]"
  
  # Restrict API access
  api:
    bind_address: "127.0.0.1"
    enable_tls: true
    tls_cert: "/etc/nrdot/certs/server.crt"
    tls_key: "/etc/nrdot/certs/server.key"
```

#### Runtime
- Run as non-root user
- Use systemd security features
- Enable SELinux/AppArmor policies
- Monitor security logs

### Compliance

NRDOT-HOST helps meet compliance requirements:

#### PCI-DSS
- Credit card number detection and masking
- Secure transmission of cardholder data
- Access logging and monitoring

#### HIPAA
- PHI detection patterns
- Encryption in transit
- Audit trails

#### SOC2
- Security controls documentation
- Change management processes
- Incident response procedures

## Security Updates

### Automatic Updates
```bash
# Enable automatic security updates
systemctl enable nrdot-updater
```

### Manual Updates
```bash
# Check for updates
nrdot-ctl check-updates

# Apply updates
sudo yum update nrdot-host
```

### Security Bulletins

Subscribe to security updates:
- GitHub Watch (Security Advisories)
- New Relic Security Bulletins
- Email notifications

## Security Checklist

### Pre-Production
- [ ] Review configuration for secrets
- [ ] Enable all security features
- [ ] Set up TLS certificates
- [ ] Configure firewall rules
- [ ] Test secret redaction

### Production
- [ ] Monitor security logs
- [ ] Regular security updates
- [ ] Audit configuration changes
- [ ] Review access logs
- [ ] Incident response plan

## Hardening Guide

### Linux
```bash
# Create dedicated user
useradd -r -s /bin/false nrdot

# Set up directories
mkdir -p /etc/nrdot /var/lib/nrdot /var/log/nrdot
chown -R nrdot:nrdot /etc/nrdot /var/lib/nrdot /var/log/nrdot
chmod 750 /etc/nrdot /var/lib/nrdot
chmod 755 /var/log/nrdot

# SELinux context
semanage fcontext -a -t nrdot_conf_t '/etc/nrdot(/.*)?'
restorecon -Rv /etc/nrdot
```

### Container
```dockerfile
# Run as non-root
USER nrdot:nrdot

# Read-only filesystem
RUN chmod -R a-w /app

# No new privileges
RUN setcap -r /app/nrdot-collector
```

### Kubernetes
```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  fsGroup: 1000
  seccompProfile:
    type: RuntimeDefault
  capabilities:
    drop:
    - ALL
```

## Contact

For security questions not related to vulnerabilities:
- Documentation: [Security Guide](./docs/security.md)
- Community: [Discussions](https://github.com/deepaucksharma/nrdot-host/discussions)

---

Thank you for helping keep NRDOT-HOST secure!