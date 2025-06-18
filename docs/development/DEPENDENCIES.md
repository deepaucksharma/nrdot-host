# NRDOT-Host Component Dependencies

## Dependency Graph

```mermaid
graph TD
    %% User Interface Layer
    UI[nrdot-ctl CLI] --> CE[nrdot-config-engine]
    UI --> SUP[nrdot-supervisor]
    UI --> API[nrdot-api-server]
    
    %% Configuration Layer
    CE --> SCH[nrdot-schema]
    CE --> TL[nrdot-template-lib]
    RC[nrdot-remote-config] --> CE
    
    %% Execution Layer
    SUP --> TC[nrdot-telemetry-client]
    SUP --> OC[OTel Collector]
    
    %% Processor Layer
    OC --> SEC[otel-processor-nrsecurity]
    OC --> ENR[otel-processor-nrenrich]
    OC --> TRN[otel-processor-nrtransform]  
    OC --> CAP[otel-processor-nrcap]
    
    SEC --> PH[nrdot-privileged-helper]
    SEC --> COM[otel-processor-common]
    ENR --> COM
    TRN --> COM
    CAP --> COM
    
    %% Testing Layer
    TH[nrdot-test-harness] -.-> UI
    GF[guardian-fleet-infra] --> WS[nrdot-workload-simulators]
    CV[nrdot-compliance-validator] -.-> SEC
    BS[nrdot-benchmark-suite] -.-> GF
    
    %% Deployment Layer
    PKG[nrdot-packaging] --> UI
    PKG --> SUP
    CI[nrdot-container-images] --> PKG
    K8S[nrdot-k8s-operator] --> CI
    HELM[nrdot-helm-chart] --> CI
    ANS[nrdot-ansible-role] --> PKG
    
    %% Tools Layer
    MIG[nrdot-migrate] --> CE
    DBG[nrdot-debug-tools] --> API
    SDK[nrdot-sdk-go] --> COM
    HA[nrdot-health-analyzer] --> TC
    CC[nrdot-cost-calculator] --> CAP
    
    %% Fleet Management
    FP[nrdot-fleet-protocol] --> UI
    FP --> RC
```

## Integration Points

### Core Dependencies
- **nrdot-ctl** depends on: config-engine, supervisor, telemetry-client, api-server
- **config-engine** depends on: schema, template-lib
- **supervisor** depends on: telemetry-client

### Processor Dependencies  
- All processors depend on: otel-processor-common
- nrsecurity processor depends on: privileged-helper (for non-root mode)

### Testing Dependencies
- test-harness validates all components
- guardian-fleet-infra deploys workload-simulators
- compliance-validator specifically tests security processors

### Deployment Dependencies
- packaging bundles core components
- container-images packages for Docker/K8s
- k8s-operator and helm-chart deploy container-images
- ansible-role deploys packages

### Tool Dependencies
- migrate reads/writes config-engine formats
- debug-tools connects to api-server
- sdk-go extends processor-common
- health-analyzer analyzes telemetry-client data
- cost-calculator works with nrcap processor data