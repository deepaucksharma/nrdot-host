# NRDOT-HOST Phase Checklists

This document provides detailed engineering checklists for each phase of the NRDOT-HOST evolution to become New Relic's canonical Linux telemetry collector.

## Phase 0: Foundation & Cleanup (2 weeks)

**Goal**: Prepare codebase for Linux-only focus and establish baseline

### Week 1: Code Cleanup
- [ ] Remove Windows build targets from Makefile
- [ ] Remove macOS build targets from Makefile  
- [ ] Delete `process_metrics_other.go` files
- [ ] Remove cross-platform abstractions
- [ ] Update all build tags to `//go:build linux`
- [ ] Remove Kubernetes operator code
- [ ] Clean up deployment/kubernetes unnecessary files

### Week 2: CI/CD & Documentation
- [ ] Update GitHub Actions for Linux-only builds
- [ ] Remove Windows/macOS CI jobs
- [ ] Update all documentation to reflect Linux focus
- [ ] Create Linux distribution test matrix
- [ ] Update CONTRIBUTING.md with Linux-only policy
- [ ] Archive old cross-platform code

### Exit Criteria
- [ ] Builds only produce Linux binaries
- [ ] All tests pass on Linux
- [ ] Documentation reflects Linux-only focus
- [ ] CI/CD runs Linux targets only

## Phase 1: Top-N Process Telemetry (4 weeks)

**Goal**: Implement enhanced process monitoring compatible with Infrastructure agent

### Week 1: /proc Parser Design
- [ ] Design /proc parsing architecture
- [ ] Create process data models
- [ ] Define metrics schema
- [ ] Plan privileged helper integration
- [ ] Document performance targets

### Week 2: Core Implementation
- [ ] Implement /proc/[pid]/stat parser
- [ ] Implement /proc/[pid]/status parser
- [ ] Implement /proc/[pid]/cmdline parser
- [ ] Add memory metrics collection
- [ ] Add CPU metrics collection
- [ ] Create process relationship mapping

### Week 3: Top-N Tracking & Patterns
- [ ] Implement Top-N by CPU algorithm
- [ ] Implement Top-N by memory algorithm
- [ ] Add service pattern matching
- [ ] Create service detection rules
- [ ] Integrate with privileged helper
- [ ] Add caching for efficiency

### Week 4: Testing & Integration
- [ ] Unit tests for all parsers
- [ ] Benchmark /proc parsing performance
- [ ] Integration with OTel metrics
- [ ] Compatibility testing with Infra agent
- [ ] Load testing with many processes
- [ ] Documentation updates

### Exit Criteria
- [ ] Process metrics match Infrastructure agent format
- [ ] < 5% CPU overhead for monitoring
- [ ] Top-10 processes tracked accurately
- [ ] Service patterns detect common services
- [ ] All tests passing with >80% coverage

## Phase 2: Intelligent Auto-Configuration (6 weeks)

**Goal**: Implement zero-touch service discovery and configuration

### Week 1-2: Service Discovery Engine
- [ ] Design discovery architecture
- [ ] Implement process scanner
- [ ] Implement port scanner (netstat/ss)
- [ ] Create service matchers
- [ ] Add version detection
- [ ] Unit test all scanners

### Week 3: Baseline Reporting
- [ ] Design baseline schema
- [ ] Implement baseline collector
- [ ] Create reporting client
- [ ] Add retry/backoff logic
- [ ] Test with mock endpoints
- [ ] Document API contract

### Week 4: Remote Configuration
- [ ] Design config retrieval flow
- [ ] Implement config client
- [ ] Add config validation
- [ ] Create template engine
- [ ] Implement config merger
- [ ] Add rollback capability

### Week 5: Integration & Templates
- [ ] Create MySQL template
- [ ] Create PostgreSQL template
- [ ] Create Redis template
- [ ] Create Nginx template
- [ ] Create Apache template
- [ ] Test blue-green reload

### Week 6: Testing & Polish
- [ ] End-to-end discovery tests
- [ ] Config update testing
- [ ] Performance benchmarks
- [ ] Error handling scenarios
- [ ] Documentation
- [ ] Beta release preparation

### Exit Criteria
- [ ] 5 services auto-configured successfully
- [ ] < 30 second discovery time
- [ ] Zero-downtime config updates
- [ ] 90% reduction in manual config
- [ ] All integration tests passing

## Phase 3: GA & Migration (4 weeks)

**Goal**: Production-ready release with migration tools

### Week 1: Migration Tool
- [ ] Design migration architecture
- [ ] Implement infra agent detection
- [ ] Create config converter
- [ ] Add license key migration
- [ ] Implement validation checks
- [ ] Create rollback mechanism

### Week 2: Packaging
- [ ] Create .deb package spec
- [ ] Create .rpm package spec
- [ ] Set up package signing
- [ ] Create repository structure
- [ ] Test package installation
- [ ] Create uninstall scripts

### Week 3: Installation & Distribution
- [ ] Create install.sh script
- [ ] Set up download.newrelic.com
- [ ] Configure CDN distribution
- [ ] Create systemd service files
- [ ] Add logrotate configuration
- [ ] Test on major distributions

### Week 4: GA Preparation
- [ ] Performance optimization
- [ ] Security audit
- [ ] Documentation review
- [ ] Migration guide updates
- [ ] Release notes
- [ ] GA announcement prep

### Exit Criteria
- [ ] 95% successful automated migrations
- [ ] < 1 hour migration for complex setups
- [ ] Packages install cleanly on all distros
- [ ] < 150MB memory footprint
- [ ] All security scans passing
- [ ] Documentation complete

## Success Metrics Summary

### Overall Project Success
- [ ] 100% feature parity with Infrastructure agent
- [ ] 50% reduction in setup time
- [ ] 90% of services auto-configured
- [ ] < 150MB memory footprint
- [ ] < 2% CPU usage at idle

### Quality Gates
Each phase must pass before proceeding:
- [ ] All checklist items complete
- [ ] Exit criteria met
- [ ] No critical bugs
- [ ] Performance targets achieved
- [ ] Documentation updated

## Risk Tracking

### Phase 1 Risks
- /proc parsing performance
- Privileged helper security
- Process tracking accuracy

### Phase 2 Risks
- Service detection accuracy
- Config template complexity
- API availability

### Phase 3 Risks
- Migration compatibility
- Package conflicts
- Distribution coverage

---

*Last Updated: 2025-06-18*  
*Version: 1.0*