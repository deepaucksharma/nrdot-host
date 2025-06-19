# Security Policy

## Reporting Security Vulnerabilities

The NRDOT-HOST team takes security seriously. We appreciate your efforts to responsibly disclose your findings.

### Where to Report

**DO NOT** report security vulnerabilities through public GitHub issues.

Instead, please report them via one of these methods:

1. **Email**: security@newrelic.com
2. **New Relic Security Portal**: https://newrelic.com/security
3. **GitHub Security Advisories**: [Report a vulnerability](https://github.com/newrelic/nrdot-host/security/advisories/new)

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

#### Installation (Linux Only)
```bash
# Download and verify binary
curl -L https://github.com/newrelic/nrdot-host/releases/latest/download/nrdot-host-linux-amd64 -o nrdot-host
curl -L https://github.com/newrelic/nrdot-host/releases/latest/download/checksums.txt -o checksums.txt
sha256sum -c checksums.txt

# Set secure permissions
chmod 755 nrdot-host
sudo mv nrdot-host /usr/local/bin/

# Secure configuration
sudo mkdir -p /etc/nrdot
sudo chmod 600 /etc/nrdot/config.yaml
sudo chown nrdot:nrdot /etc/nrdot/config.yaml
```

#### Configuration
```yaml
# Security is built-in via nrsecurity processor
processors:
  nrsecurity:
    # Automatic secret detection and redaction
    # No configuration needed - secure by default

# API server security (when implemented)
# api:
#   bind_address: "127.0.0.1:8090"  # Local only
#   auth_enabled: true              # Phase 3
```

#### Runtime Security (Linux)
- Run as dedicated nrdot user
- Use systemd security features
- Privileged helper for elevated operations
- Monitor logs: `journalctl -u nrdot-host`

#### Systemd Hardening
```ini
[Service]
User=nrdot
Group=nrdot
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
NoNewPrivileges=true
ReadWritePaths=/var/lib/nrdot
```

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

### Updates

#### Current Process
```bash
# Check for new releases
curl -s https://api.github.com/repos/newrelic/nrdot-host/releases/latest | grep tag_name

# Download and install new version
curl -L https://github.com/newrelic/nrdot-host/releases/latest/download/nrdot-host-linux-amd64 -o nrdot-host
# Verify and install as above
```

#### Future (Phase 3)
```bash
# Package manager updates
sudo apt update && sudo apt upgrade nrdot-host
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

## Linux Hardening Guide

### System Setup
```bash
# Create dedicated user
sudo useradd -r -s /bin/false nrdot

# Set up directories
sudo mkdir -p /etc/nrdot /var/lib/nrdot
sudo chown -R nrdot:nrdot /etc/nrdot /var/lib/nrdot
sudo chmod 750 /etc/nrdot /var/lib/nrdot

# Privileged helper (for process monitoring)
sudo chown root:nrdot /usr/local/bin/nrdot-privileged-helper
sudo chmod 4750 /usr/local/bin/nrdot-privileged-helper
```

### SELinux (Optional)
```bash
# Create custom policy for NRDOT-HOST
ausearch -c 'nrdot-host' --raw | audit2allow -M nrdot-host
semodule -i nrdot-host.pp
```

### Container Security
```dockerfile
# Minimal Linux base
FROM alpine:latest

# Non-root user
RUN adduser -D -H -s /sbin/nologin nrdot
USER nrdot

# Read-only root filesystem
WORKDIR /app
COPY --chown=nrdot:nrdot nrdot-host /app/
```

**Note**: Container deployments require host PID namespace for process monitoring:
```bash
docker run --pid=host --network=host -v /proc:/host/proc:ro nrdot-host
```

## Contact

For security questions not related to vulnerabilities:
- Documentation: [Security Guide](./docs/security.md)
- Community: GitHub Issues

---

Thank you for helping keep NRDOT-HOST secure!