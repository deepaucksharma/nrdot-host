# nrdot-compliance-validator

Security and compliance validation framework for NRDOT-Host.

## Overview
Ensures NRDOT meets security standards and compliance requirements through automated validation and audit report generation.

## Compliance Checks
- Secret redaction verification
- Non-root execution validation
- TLS configuration audit
- Data residency compliance
- FIPS mode validation

## Standards Supported
- PCI-DSS
- HIPAA
- SOC 2
- FedRAMP
- GDPR

## Reports
```bash
# Generate compliance report
nrdot-compliance-validator audit --standard=pci-dss
```

## Integration
- Validates `otel-processor-nrsecurity`
- Part of CI/CD pipeline
- Used for customer attestation
