# Phase 1-3 Integration Workflow Diagram

## Complete End-to-End Workflow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     INTEGRATION TEST WORKFLOW                               │
└─────────────────────────────────────────────────────────────────────────────┘

PHASE 1: INPUT SETUP & CONFIGURATION
════════════════════════════════════════════════════════════════════════════════

┌─────────────────────────────┐
│ K8s Client Creation         │
│ ├─ Fake Clientset           │
│ ├─ Client Wrapper           │
│ └─ Context Setup            │
└────────────┬────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────┐
│ Generate Presigned URLs                         │
│ ├─ report.md → https://storage/report.md      │
│ ├─ agent.log → https://storage/agent.log      │
│ ├─ session.tar.gz → https://storage/session   │
│ ├─ result.json → https://storage/result.json  │
│ └─ commands → https://storage/commands.log    │
└────────────┬────────────────────────────────────┘
             │
             ▼
┌──────────────────────────────────────────────────────────┐
│ Create ConfigMap with Incident Data                      │
│ ├─ incident.json (Incident struct)                       │
│ │  └─ IncidentID, Fault, Severity, Cluster, Resource  │
│ ├─ permissions.json (ClusterPermissions)                 │
│ │  └─ Can Get Pods, Logs, Events, Nodes, etc.           │
│ └─ system-prompt.md                                      │
│    └─ Kubernetes troubleshooting instructions            │
└────────────┬──────────────────────────────────────────────┘
             │
             ▼
         ConfigMap Created
         incident-integration-test-123


PHASE 2: JOB ORCHESTRATION & EXECUTION
════════════════════════════════════════════════════════════════════════════════

┌──────────────────────────────────┐
│ Create Job with Configuration    │
│                                  │
│ JobConfig:                       │
│ ├─ Namespace: nightcrier         │
│ ├─ Image: nc-agent-runner:test   │
│ ├─ AgentCLI: claude              │
│ ├─ LLMModel: claude-opus-4-5     │
│ ├─ ConfigMapName: incident-...   │
│ │                                │
│ ├─ Volumes:                      │
│ │  ├─ incident-data (ConfigMap)  │
│ │  └─ kubeconfig (Secret)        │
│ │                                │
│ ├─ Environment Variables:        │
│ │  ├─ AGENT_CLI=claude           │
│ │  ├─ INCIDENT_ID=...            │
│ │  ├─ OUTPUT_URL_REPORT=...      │
│ │  ├─ OUTPUT_URL_LOG=...         │
│ │  └─ (plus other URLs)          │
│ │                                │
│ └─ Resources:                    │
│    ├─ Limits: 2Gi RAM, 1 CPU     │
│    └─ Requests: 512Mi, 250m      │
└────────────┬─────────────────────┘
             │
             ▼
         Job Created
         triage-integration-test-123
             │
             ▼
┌──────────────────────────────────┐
│ Watch Job for Completion         │
│                                  │
│ Job Status Transitions:          │
│ ├─ Initial: Pending/Running      │
│ ├─ Active: 1 pod running         │
│ └─ Terminal: Succeeded/Failed    │
│                                  │
│ Watch Configuration:             │
│ ├─ Namespace: nightcrier         │
│ ├─ JobName: triage-...           │
│ ├─ Timeout: 5 seconds            │
│ └─ LogFunc: Progress logging     │
└────────────┬─────────────────────┘
             │
             ▼
         Job Status = Succeeded


PHASE 3: RESULTS RETRIEVAL & CLEANUP
════════════════════════════════════════════════════════════════════════════════

┌─────────────────────────────────────────────────┐
│ Retrieve Results from Object Store              │
│                                                 │
│ Storage Paths:                                  │
│ incidents/integration-test-123/results/         │
│ ├─ result.json                                  │
│ │  └─ {"exit_code": 0, "message": "Success"}   │
│ ├─ report.md                                    │
│ │  └─ # Investigation Report                    │
│ ├─ agent.log                                    │
│ │  └─ Starting agent execution...               │
│ ├─ commands-executed.log                        │
│ │  └─ $ kubectl get pods                        │
│ └─ session.tar.gz (optional)                    │
│    └─ Full session archive                      │
└────────────┬────────────────────────────────────┘
             │
             ▼
┌────────────────────────────────────┐
│ Build JobResults Structure         │
│                                    │
│ Results:                           │
│ ├─ ResultJSON                      │
│ │  └─ ExitCode: 0                  │
│ ├─ ReportMD: []byte                │
│ ├─ AgentLog: []byte                │
│ ├─ CommandsExecuted: []byte        │
│ ├─ SessionArchive: []byte          │
│ └─ Missing: []string (empty)       │
└────────────┬───────────────────────┘
             │
             ▼
┌──────────────────────────────────┐
│ Verify All Artifacts Retrieved   │
│                                  │
│ Assertions:                      │
│ ├─ ResultJSON != nil             │
│ ├─ ExitCode == 0                 │
│ ├─ ReportMD length > 0           │
│ ├─ AgentLog length > 0           │
│ ├─ CommandsExecuted length > 0   │
│ ├─ SessionArchive length > 0     │
│ └─ Missing array is empty        │
└────────────┬──────────────────────┘
             │
             ▼
┌──────────────────────────────────┐
│ Cleanup ConfigMap                │
│                                  │
│ ├─ Delete incident-...-123       │
│ └─ Verify deletion successful    │
└────────────┬──────────────────────┘
             │
             ▼
         WORKFLOW COMPLETE


═══════════════════════════════════════════════════════════════════════════════

SCENARIO: Partial Artifacts (Failure Scenario)
═══════════════════════════════════════════════════════════════════════════════

If Job fails before completing all uploads:

    Phase 1: ConfigMap created ✓
            │
    Phase 2: Job created ✓
            │
            ├─ Executes commands...
            │
            ├─ Uploads result.json ✓
            │
            ├─ Timeout occurs! ✗
            │
            └─ Incomplete uploads

    Phase 3: Results retrieval handles partial success

    Retrieved:
    ├─ ResultJSON: Present (exit code indicates failure)
    │
    Missing:
    ├─ report.md (not uploaded)
    ├─ agent.log (not uploaded)
    ├─ commands-executed.log (not uploaded)
    └─ session.tar.gz (not uploaded)

    Application can:
    ├─ Read exit code from result.json
    ├─ Track missing artifacts
    └─ Handle gracefully with partial data


═══════════════════════════════════════════════════════════════════════════════

SCENARIO: Complete Failure (No Artifacts)
═══════════════════════════════════════════════════════════════════════════════

If Job fails immediately:

    Phase 1: ConfigMap created ✓
            │
    Phase 2: Job created ✓
            │
            ├─ ImagePullBackOff or other early failure
            │
            └─ No uploads attempted

    Phase 3: Results retrieval handles complete failure

    Retrieved: (nothing)

    Missing:
    ├─ result.json
    ├─ report.md
    ├─ agent.log
    ├─ commands-executed.log
    └─ session.tar.gz

    Application:
    ├─ Knows Job failed (no results)
    ├─ Can retry or escalate
    └─ Still cleans up ConfigMap


═══════════════════════════════════════════════════════════════════════════════

DATA FLOW SUMMARY
═══════════════════════════════════════════════════════════════════════════════

Input Data
    │
    ├─→ Phase 1: Marshal to JSON strings
    │       │
    │       └─→ Store in ConfigMap
    │            │
    │            └─→ Mount in Pod volumes
    │
    ├─→ Phase 2: Pass presigned URLs to Job
    │       │
    │       └─→ Container uses URLs to upload results
    │            │
    │            └─→ Results → Object Store
    │
    └─→ Phase 3: Retrieve from Object Store
            │
            └─→ Parse and deliver to application
                    │
                    └─→ Delete ConfigMap


═══════════════════════════════════════════════════════════════════════════════

COMPONENT INTERACTIONS
═══════════════════════════════════════════════════════════════════════════════

K8s Client           │     Object Store        │     Application
─────────────────────┼──────────────────────────┼─────────────────
                     │                          │
CreateIncidentConfig │                          │
Map()                │                          │
        ├──→ Stores  │                          │
        │            │                          │
CreateJob()          │                          │
        ├──→ Injects │ GeneratePresignedURLs    │
        │   URLs     │     ←──────────────────→ │ Passed to Job
        │            │                          │
WatchJob()           │                          │
        ├──→ Monitors│                          │
        │   progress │                          │
        │            │                          │
        │            │ (Job container uploads  │
        │            │  results using URLs)    │
        │            │                          │
RetrieveResults()    │←───────────────────────→ │ Downloaded
        ├──→ Reads   │                          │
        │            │                          │
DeleteConfigMap()    │                          │
        └──→ Cleans  │                          │
             up      │                          │
