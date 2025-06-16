# nrdot-privileged-helper

Secure privileged helper for non-root process information collection.

## Overview
Setuid binary that enables NRDOT to collect process information without running the main collector as root, enhancing security.

## Features
- Minimal privileged operations
- Secure IPC protocol
- Process info collection
- Strict input validation
- Audit logging

## Security Model
```
nrdot-ctl (non-root) ← IPC → privileged-helper (setuid) → /proc/*
```

## API
```c
// Request/Response protocol
struct ProcessInfoRequest {
    uint32_t pid;
    uint32_t flags;
};

struct ProcessInfoResponse {
    char cmdline[4096];
    uint64_t memory_rss;
    // ...
};
```

## Integration
- Called by `otel-processor-nrsecurity`
- Minimal attack surface
