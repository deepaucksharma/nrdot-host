# Documentation Alignment Report & Gap Analysis

## Executive Summary

This report identifies gaps between NRDOT-HOST documentation and implementation, providing a task list to align docs with current reality while maintaining the future vision.

## Current State Analysis

### Implementation Reality (What Actually Exists)
- ✅ **Unified binary architecture** (v2.0 complete)
- ✅ **Basic Linux host monitoring** via OpenTelemetry
- ✅ **Custom processors** (nrsecurity, nrenrich, nrtransform, nrcap)
- ✅ **Blue-green configuration reload**
- ✅ **Privileged helper** (basic implementation, not integrated)
- ✅ **API server** for health/status

### Documentation Claims (Not Yet Implemented)
- ❌ **Auto-configuration** system
- ❌ **Service discovery** engine
- ❌ **Remote configuration** client
- ❌ **Migration tools** (nrdot-migrate)
- ❌ **Process telemetry** integration
- ❌ **Baseline reporting** to New Relic

## Critical Documentation Alignment Tasks

### 1. Repository Layout Reorganization

```bash
# Current structure (needs reorganization)
/
├── README.md
├── ARCHITECTURE_V2.md
├── AUTO_CONFIGURATION.md
├── INFRASTRUCTURE_MIGRATION.md
├── PROJECT_STATUS.md
└── docs/

# Target structure
/
├── README.md
├── CONTRIBUTING.md
├── SECURITY.md
├── LICENSE
└── docs/
    ├── ARCHITECTURE.md
    ├── roadmap/
    │   ├── ROADMAP.md
    │   ├── PROJECT_STATUS.md
    │   └── PHASE_CHECKLISTS.md
    ├── auto-config/
    │   ├── AUTO_CONFIGURATION.md
    │   └── baseline_schema.json
    ├── migration/
    │   └── INFRASTRUCTURE_MIGRATION.md
    └── config-schema/
        └── config.yaml.tmpl
```

### 2. README.md Precise Alignment

| Section | Current Issue | Required Change |
|---------|---------------|-----------------|
| Header badges | Shows 40% memory reduction | Remove - this is v1→v2 artifact |
| Vision & Roadmap | Vague "Enhanced Process Monitoring" | Change to "Top-N Process Telemetry (Phase 1)" |
| Installation | Points to non-existent URL | Update to `download.newrelic.com/nrtc/install.sh` with note |
| Configuration | Shows `scan_interval` | Remove - fixed at 5m in v1 |
| Performance | Shows absolute numbers | Replace with "Targets (Phase 3 GA)" |
| Roadmap timeline | Generic "3-6 months" | Specify: Phase 1=4 weeks, Phase 2=6 weeks, Phase 3=4 weeks |

### 3. ARCHITECTURE.md Updates

- Rename from `ARCHITECTURE_V2.md` → `ARCHITECTURE.md`
- Remove all Kubernetes and eBPF references
- Clarify Blue-Green is required strategy
- Add privileged helper limitations
- Update data flow diagrams with correct endpoints
- Sync roadmap timeline with README

### 4. AUTO_CONFIGURATION.md Reality Check

| Section | Required Update |
|---------|----------------|
| Security | State "env vars and secrets.yaml only in Phase 2" |
| Supported Services | Add header: "Lists describe future auto-discovery coverage" |
| Config examples | Remove MySQL per-metric toggles (not supported) |
| Baseline JSON | Add `schema_version: "1.0"` field |
| Phase labels | Rename "Phase 3 (6 months)" to "Post-GA (stretch)" |

### 5. Implementation Gaps to Document

Create new file `docs/roadmap/IMPLEMENTATION_STATUS.md`:

```markdown
# Implementation Status

## Currently Implemented (v2.0)
- Unified binary with modes (all, agent, api)
- OpenTelemetry-based collection
- Custom processors pipeline
- Configuration engine (manual)
- Blue-green reload strategy
- Basic privileged helper

## Not Yet Implemented (Roadmap Items)
- Auto-configuration engine
- Service discovery mechanisms
- Remote configuration client
- Migration CLI tools
- Process telemetry integration
- Baseline reporting API
```

### 6. Build System Alignment

The Makefile still builds for multiple platforms:
```makefile
# Current (line 163-170)
GOOS=darwin GOARCH=amd64 $(GO) build...
GOOS=windows GOARCH=amd64 $(GO) build...

# Should be Linux-only
GOOS=linux GOARCH=amd64 $(GO) build...
GOOS=linux GOARCH=arm64 $(GO) build...
```

### 7. Missing Code vs Documentation

| Documented Feature | Code Location | Status |
|--------------------|---------------|---------|
| `nrdot-host discover` | cmd/nrdot-host/main.go | ❌ Missing |
| Service discovery | nrdot-discovery/ | ❌ No directory |
| Remote config client | nrdot-remote-config/ | ❌ No directory |
| Migration tool | nrdot-migrate/ | ❌ No directory |
| Auto-config flow | nrdot-supervisor/ | ❌ No integration |

## Recommended Actions

### Phase 0: Documentation Honesty (Week 1)
1. **Update all docs** to clearly mark features as "Planned" vs "Implemented"
2. **Add implementation status** badges to each major feature
3. **Create roadmap directory** with honest status tracking
4. **Update README** to reflect current v2.0 state

### Phase 1: Foundation Alignment (Week 2-3)
1. **Remove multi-platform builds** from Makefile
2. **Create stub commands** for documented features with "not yet implemented" messages
3. **Integrate privileged helper** into main telemetry flow
4. **Document actual v2.0 capabilities** clearly

### Phase 2: Roadmap Clarity (Week 4)
1. **Create PHASE_CHECKLISTS.md** with engineering deliverables
2. **Add baseline_schema.json** for future compatibility
3. **Update all timelines** to match engineering plan
4. **Create CI checks** for documentation drift

## Version Strategy

Recommend version numbering:
- **v2.0.x** - Current unified architecture (manual config)
- **v2.1.x** - Phase 1: Enhanced process monitoring
- **v3.0.0-beta** - Phase 2: Auto-configuration preview
- **v3.0.0** - Phase 3: GA with migration tools

## Conclusion

The documentation currently describes an aspirational v3.0 feature set while the code is at v2.0. We need to:
1. Clearly separate "current" from "planned" features
2. Add implementation tracking documents
3. Create stubs for documented but unimplemented commands
4. Align build system with Linux-only focus
5. Set realistic timelines based on engineering capacity

This alignment will prevent user confusion and provide clear tracking for the 3-phase implementation plan.