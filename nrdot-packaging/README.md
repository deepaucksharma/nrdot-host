# nrdot-packaging

Multi-platform packaging specifications and build scripts for NRDOT-Host.

## Overview
Handles creation of distribution packages for all supported platforms including RPM, DEB, MSI, and archives.

## Package Types
- RPM (RHEL, CentOS, Fedora)
- DEB (Ubuntu, Debian)
- MSI (Windows)
- TAR.GZ (Generic Linux)
- PKG (macOS)

## Features
- Package signing
- Dependency management
- Post-install scripts
- Service registration
- Upgrade handling

## Build Commands
```bash
# Build all packages
make packages

# Build specific type
make rpm
make deb
```

## Integration
- Uses binaries from all components
- Signed with New Relic certificates
