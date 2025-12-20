# Nightcrier

An AI-powered incident triage system that automatically investigates Kubernetes faults using Claude AI agents through the Model Context Protocol (MCP).

## Purpose and Architecture Philosophy

**Nightcrier is a specialized incident management system, NOT a general Kubernetes event processor.**

### What Nightcrier Does

Nightcrier subscribes to **high-quality fault events only** from 100s to 1000s of kubernetes-mcp-servers and treats each fault as a serious incident requiring investigation:

1. **One fault subscription per MCP server** - Each kubernetes-mcp-server connection maintains a single fault event subscription
2. **Treats faults as incidents** - Every fault event is a serious incident requiring AI-powered investigation
3. **Launches AI triage agents** - Spawns containerized AI agents for full root cause analysis
4. **Reports to operations teams** - Delivers investigation reports via Slack with storage links
5. **End-to-end incident tracking** - Tracks each incident from detection through investigation to reporting

### What Nightcrier is NOT

- **NOT a general event subscriber** - Would be overwhelmed by raw Kubernetes events at scale
- **NOT responsible for event filtering** - Signal-to-noise filtering happens upstream in kubernetes-mcp-server
- **NOT a high-volume event processing system** - Designed for pre-filtered, high-signal faults only

### Design Decisions

**Event Types:**
- **General Events**: Raw Kubernetes events (handled by kubernetes-mcp-server)
- **Fault Events**: Pre-filtered, high-signal events indicating problems (from kubernetes-mcp-server to nightcrier)
- **Incidents**: Fault events under active AI investigation (nightcrier's domain)

**Filtering Philosophy:**
- Signal-to-noise filtering happens in kubernetes-mcp-server, NOT in nightcrier
- kubernetes-mcp-server uses sophisticated logic to identify true faults worth investigating
- This design prevents nightcrier from being overwhelmed at scale
- Ensures AI agents only investigate genuine incidents, not noise

**Future Scale-Out (not implemented):**
- If needed: High-performance queue with worker pools for pre-qualification
- Current architecture keeps filtering in MCP servers to maintain simplicity

## Overview

This system listens for fault events from a Kubernetes MCP server and spawns AI agents to autonomously investigate and triage incidents. It provides automated root cause analysis with configurable storage backends (filesystem or Azure Blob Storage) and Slack notifications.

## Architecture

### High-Level Flow

```
kubernetes-mcp-server -> MCP Events -> Nightcrier -> AI Agent -> Investigation Report
                                            |
                                            v
                                    Storage (Azure/Filesystem)
                                            |
                                            v
                                    Slack Notification (with Report URL)
```

### Detailed Validation Flow

```
┌─────────────────┐
│  Fault Event    │
│  from MCP       │
└────────┬────────┘
         │
         v
┌─────────────────────────────────────────────────────────────┐
│  Nightcrier                                                  │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 1. Create Workspace                                   │  │
│  │    ./incidents/<incident-id>/                        │  │
│  │    - incident.json (event context)                   │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          v                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 2. Execute AI Agent                                   │  │
│  │    - Spawn container with LLM CLI                    │  │
│  │    - Agent investigates incident                     │  │
│  │    - Writes output/investigation.md                  │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          v                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 3. Validate Output                                    │  │
│  │    ✓ Exit code = 0?                                  │  │
│  │    ✓ investigation.md exists?                        │  │
│  │    ✓ File size > 100 bytes?                          │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│              ┌───────────┴───────────┐                      │
│              │                       │                       │
│              v                       v                       │
│      ┌──────────────┐        ┌──────────────┐              │
│      │   SUCCESS    │        │   FAILURE    │              │
│      └──────┬───────┘        └──────┬───────┘              │
│             │                       │                       │
│             │                       v                       │
│             │              ┌─────────────────────┐         │
│             │              │ Circuit Breaker     │         │
│             │              │ Track Failures      │         │
│             │              │ (count: 1/3, 2/3..) │         │
│             │              └─────────┬───────────┘         │
│             │                        │                      │
│             │                        v                      │
│             │              ┌─────────────────────┐         │
│             │              │ Threshold Reached?  │         │
│             │              │ (default: 3 failures)│         │
│             │              └─────────┬───────────┘         │
│             │                        │                      │
│             │              ┌─────────┴─────────┐           │
│             │              │                   │           │
│             │              v                   v           │
│             │         ┌────────┐      ┌──────────────┐   │
│             │         │  YES   │      │     NO       │   │
│             │         └───┬────┘      └──────────────┘   │
│             │             │                                │
│             │             v                                │
│             │    ┌─────────────────────┐                  │
│             │    │ Send System         │                  │
│             │    │ Degraded Alert      │                  │
│             │    │ (if configured)     │                  │
│             │    └─────────────────────┘                  │
│             │                                               │
│             │  Skip individual notification                │
│             │  Skip storage upload (by default)           │
│             │                                               │
│             v                                               │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 4. Upload to Storage (if validated)                  │  │
│  │    - Azure Blob Storage (with SAS URLs)              │  │
│  │    - OR Filesystem storage                           │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          v                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 5. Send Slack Notification (if validated)            │  │
│  │    - Incident details + root cause                   │  │
│  │    - Link to investigation report                    │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                               │
└───────────────────────────────────────────────────────────────┘

Circuit Breaker States:
┌──────────┐  threshold failures   ┌──────────┐
│  CLOSED  │ ─────────────────────> │   OPEN   │
│ (Normal) │                        │(Degraded)│
└──────────┘ <───────────────────── └──────────┘
              1 success + alert sent
```

## Features

- Automated incident detection via MCP server integration
- AI-powered root cause analysis using Claude agents
- Multi-backend storage support (filesystem and Azure Blob Storage)
- Slack notifications with investigation reports
- Secure artifact storage with SAS URL generation (Azure mode)
- Containerized agent execution environment
- Circuit breaker for agent failure handling
- Intelligent validation to prevent spurious notifications
- System health monitoring with degraded/recovered alerts

## Prerequisites

- Go 1.23 or later
- Docker (for containerized agent execution)
- Kubernetes cluster with MCP server (for production use)
- Azure Storage Account (optional, for cloud storage)
- Slack webhook (optional, for notifications)

## Installation

### Build from source

```bash
# Clone the repository
git clone https://github.com/rbias/nightcrier.git
cd nightcrier

# Build the runner
go build -o runner ./cmd/runner

# Build the agent container
cd agent-container
docker build -t k8s-triage-agent:latest .
```

## Configuration

Nightcrier uses explicit configuration with no hardcoded defaults. All required parameters must be provided via configuration file, environment variables, or command-line flags.

### Configuration Files

- **`configs/config.yaml`** - Main configuration file (copy from `configs/config.example.yaml`)
- **`configs/tuning.yaml`** - Optional tuning parameters for operational adjustments (rarely changed)
- **`kubeconfigs/`** - Directory containing cluster kubeconfig files for triage agent access

### Configuration Precedence

Configuration values are loaded in the following order (highest to lowest priority):
1. Command-line flags (e.g., `--mcp-endpoint`)
2. Environment variables (e.g., `K8S_CLUSTER_MCP_ENDPOINT`)
3. Configuration file (`config.yaml`)
4. Tuning file (`tuning.yaml`, optional)

## Multi-Cluster Configuration

Nightcrier supports monitoring multiple Kubernetes clusters simultaneously through a single instance. Each cluster requires two credential sets:

1. **MCP Endpoint** - Connection to kubernetes-mcp-server for receiving fault events
2. **Kubeconfig** - Direct cluster API access for triage agents (optional, for investigation)

### Clusters Array Structure

Define all clusters in `configs/config.yaml`:

```yaml
clusters:
  - name: prod-us-east-1
    environment: production
    labels:
      region: us-east
      tier: production

    mcp:
      endpoint: http://kubernetes-mcp-server.mcp-system.svc.cluster.local:8080/mcp
      api_key: PLACEHOLDER_FOR_FUTURE_AUTH

    triage:
      enabled: true
      kubeconfig: ./kubeconfigs/prod-us-east-1-readonly.yaml
      allow_secrets_access: false

  - name: staging-eu-west-1
    environment: staging
    labels:
      region: eu-west
      tier: staging

    mcp:
      endpoint: http://10.42.0.23:8080/mcp
      api_key: PLACEHOLDER_FOR_FUTURE_AUTH

    triage:
      enabled: true
      kubeconfig: ./kubeconfigs/staging-eu-west-1-readonly.yaml
      allow_secrets_access: false

  - name: dev-local
    environment: development

    mcp:
      endpoint: http://localhost:8080/mcp
      api_key: PLACEHOLDER_FOR_FUTURE_AUTH

    triage:
      enabled: false
      # No kubeconfig - events received but not investigated
```

### Configuration Fields

**Cluster-level**:
- `name` (required) - Unique cluster identifier, used in logs and incident metadata
- `environment` (optional) - Environment label (production, staging, development)
- `labels` (optional) - Custom key-value labels for organization

**MCP Configuration**:
- `mcp.endpoint` (required) - kubernetes-mcp-server URL with `/mcp` path
- `mcp.api_key` (optional) - Placeholder for future MCP authentication

**Triage Configuration**:
- `triage.enabled` (required) - Enable/disable AI triage for this cluster
- `triage.kubeconfig` (required if enabled) - Path to cluster kubeconfig file
- `triage.allow_secrets_access` (optional, default: false) - Allow agent to read secrets/configmaps

### Triage Enable/Disable Behavior

**When `triage.enabled: true`**:
1. Nightcrier validates the kubeconfig at startup
2. Runs `kubectl auth can-i` checks to verify RBAC permissions
3. Creates `incident_cluster_permissions.json` in each incident workspace
4. Spawns AI agent with cluster credentials for investigation
5. Agent can run kubectl commands to diagnose the issue
6. Investigation report uploaded to storage and sent to Slack

**When `triage.enabled: false`**:
1. Fault events are still received from the MCP server
2. Events are logged but NOT investigated
3. No workspace is created
4. No AI agent is spawned
5. No Slack notification is sent

This allows you to:
- Monitor events from clusters without investigation capabilities
- Disable expensive AI triage for low-priority environments
- Receive events while waiting for RBAC setup

### Kubeconfig Path Convention

Kubeconfig files should be stored in the `./kubeconfigs/` directory:

```
nightcrier/
├── configs/
│   └── config.yaml
├── kubeconfigs/
│   ├── prod-us-east-1-readonly.yaml
│   ├── prod-eu-west-1-readonly.yaml
│   ├── staging-us-east-1-readonly.yaml
│   └── dev-local-readonly.yaml
└── incidents/
    └── <incident-id>/
        ├── incident.json
        ├── incident_cluster_permissions.json
        └── output/
```

**Important**: The kubeconfig directory is NOT committed to git. Add to `.gitignore`:
```
kubeconfigs/*.yaml
```

## Kubeconfig Setup

Triage agents require read-only cluster access to investigate incidents. Follow these steps to create appropriate credentials.

### 1. Create Read-Only ServiceAccount

Create a dedicated ServiceAccount with minimal permissions:

```yaml
# kubernetes-triage-readonly-sa.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kubernetes-triage-readonly
  namespace: kube-system
---
apiVersion: v1
kind: Secret
metadata:
  name: kubernetes-triage-readonly-token
  namespace: kube-system
  annotations:
    kubernetes.io/service-account.name: kubernetes-triage-readonly
type: kubernetes.io/service-account-token
```

Apply:
```bash
kubectl apply -f kubernetes-triage-readonly-sa.yaml
```

### 2. Grant RBAC Permissions

**Minimum Permissions (Required)**:

Bind the built-in `view` ClusterRole for basic read access:

```yaml
# kubernetes-triage-readonly-rbac.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kubernetes-triage-readonly-view
subjects:
  - kind: ServiceAccount
    name: kubernetes-triage-readonly
    namespace: kube-system
roleRef:
  kind: ClusterRole
  name: view
  apiGroup: rbac.authorization.k8s.io
```

This provides access to:
- Pods (get, list, watch)
- Pod logs (pods/log subresource)
- Events (get, list, watch)
- Deployments, ReplicaSets, StatefulSets
- Services, Endpoints
- ConfigMaps (but NOT secrets)

**Optional: Node Access**:

For cluster-wide visibility (node resource usage, taints, etc.):

```yaml
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kubernetes-triage-nodes-readonly
rules:
  - apiGroups: [""]
    resources: ["nodes"]
    verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kubernetes-triage-nodes-readonly
subjects:
  - kind: ServiceAccount
    name: kubernetes-triage-readonly
    namespace: kube-system
roleRef:
  kind: ClusterRole
  name: kubernetes-triage-nodes-readonly
  apiGroup: rbac.authorization.k8s.io
```

**Optional: Helm Debugging Permissions**:

WARNING: This allows reading secrets, which may contain sensitive data.

```yaml
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kubernetes-triage-helm-readonly
rules:
  - apiGroups: [""]
    resources: ["secrets", "configmaps"]
    verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kubernetes-triage-helm-readonly
subjects:
  - kind: ServiceAccount
    name: kubernetes-triage-readonly
    namespace: kube-system
roleRef:
  kind: ClusterRole
  name: kubernetes-triage-helm-readonly
  apiGroup: rbac.authorization.k8s.io
```

If you grant secrets access, set `triage.allow_secrets_access: true` in config.

Apply RBAC:
```bash
kubectl apply -f kubernetes-triage-readonly-rbac.yaml
```

### 3. Extract Kubeconfig

Extract the ServiceAccount token and generate a kubeconfig:

```bash
#!/bin/bash
# extract-triage-kubeconfig.sh

CLUSTER_NAME="prod-us-east-1"
SA_NAME="kubernetes-triage-readonly"
SA_NAMESPACE="kube-system"
OUTPUT_FILE="./kubeconfigs/${CLUSTER_NAME}-readonly.yaml"

# Get cluster info
CLUSTER_SERVER=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')
CLUSTER_CA=$(kubectl config view --minify --raw -o jsonpath='{.clusters[0].cluster.certificate-authority-data}')

# Get ServiceAccount token
SA_TOKEN=$(kubectl get secret -n ${SA_NAMESPACE} kubernetes-triage-readonly-token -o jsonpath='{.data.token}' | base64 -d)

# Create kubeconfig
cat > ${OUTPUT_FILE} <<EOF
apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority-data: ${CLUSTER_CA}
    server: ${CLUSTER_SERVER}
  name: ${CLUSTER_NAME}
contexts:
- context:
    cluster: ${CLUSTER_NAME}
    user: ${SA_NAME}
  name: ${CLUSTER_NAME}
current-context: ${CLUSTER_NAME}
users:
- name: ${SA_NAME}
  user:
    token: ${SA_TOKEN}
EOF

echo "Kubeconfig written to ${OUTPUT_FILE}"

# Test the kubeconfig
kubectl --kubeconfig=${OUTPUT_FILE} auth can-i --list
```

Run:
```bash
chmod +x extract-triage-kubeconfig.sh
./extract-triage-kubeconfig.sh
```

### 4. Verify Permissions

Test the generated kubeconfig:

```bash
KUBECONFIG_FILE="./kubeconfigs/prod-us-east-1-readonly.yaml"

# Test basic access
kubectl --kubeconfig=${KUBECONFIG_FILE} get pods --all-namespaces

# Verify can-i permissions
kubectl --kubeconfig=${KUBECONFIG_FILE} auth can-i get pods
kubectl --kubeconfig=${KUBECONFIG_FILE} auth can-i get pods/log
kubectl --kubeconfig=${KUBECONFIG_FILE} auth can-i get events
kubectl --kubeconfig=${KUBECONFIG_FILE} auth can-i get nodes

# Verify cannot mutate
kubectl --kubeconfig=${KUBECONFIG_FILE} auth can-i delete pods    # Should return "no"
kubectl --kubeconfig=${KUBECONFIG_FILE} auth can-i create pods    # Should return "no"
```

### 5. Configure Nightcrier

Update `configs/config.yaml` to reference the kubeconfig:

```yaml
clusters:
  - name: prod-us-east-1
    mcp:
      endpoint: http://kubernetes-mcp-server:8080/mcp
    triage:
      enabled: true
      kubeconfig: ./kubeconfigs/prod-us-east-1-readonly.yaml
      allow_secrets_access: false  # or true if Helm debugging needed
```

### Startup Permission Validation

When Nightcrier starts, it automatically validates cluster permissions:

```
level=INFO msg="initializing connection manager - validating permissions"
level=INFO msg="validating cluster permissions" cluster=prod-us-east-1 kubeconfig=./kubeconfigs/prod-us-east-1-readonly.yaml
level=INFO msg="cluster permissions validated successfully" cluster=prod-us-east-1 minimum_met=true helm_access=false
```

If permissions are insufficient:
```
level=WARN msg="cluster has permission warnings" cluster=prod-us-east-1 warnings="cannot get nodes (cluster-wide visibility limited)"
```

Permission validation results are written to `incident_cluster_permissions.json` in each incident workspace, allowing the AI agent to understand what actions are available.

### Required Configuration

The following parameters **must** be provided. The application will fail fast on startup if any are missing:

- `K8S_CLUSTER_MCP_ENDPOINT` - MCP server endpoint URL (e.g., `http://localhost:8080/mcp`)
- `SUBSCRIBE_MODE` - Event subscription mode: `events` or `faults` (recommended: `faults`)
- `WORKSPACE_ROOT` - Directory for incident artifacts (e.g., `./incidents`)
- `AGENT_SCRIPT_PATH` - Path to agent execution script (e.g., `./agent-container/run-agent.sh`)
- `AGENT_MODEL` - LLM model to use (e.g., `sonnet`, `opus`, `haiku`, `gpt-4o`)
- `AGENT_TIMEOUT` - Agent timeout in seconds (e.g., `300`)
- `AGENT_CLI` - AI CLI tool to use: `claude`, `codex`, `goose`, or `gemini`
- `AGENT_IMAGE` - Docker image for agent container (e.g., `nightcrier-agent:latest`)
- `AGENT_PROMPT` - Prompt sent to agent for triage
- `SEVERITY_THRESHOLD` - Minimum event severity: `DEBUG`, `INFO`, `WARNING`, `ERROR`, `CRITICAL`
- `MAX_CONCURRENT_AGENTS` - Maximum concurrent agent sessions
- `GLOBAL_QUEUE_SIZE` - Global event queue size
- `CLUSTER_QUEUE_SIZE` - Per-cluster queue size
- `DEDUP_WINDOW_SECONDS` - Event deduplication window (0 to disable)
- `QUEUE_OVERFLOW_POLICY` - Queue overflow policy: `drop` or `reject`
- `SHUTDOWN_TIMEOUT` - Graceful shutdown timeout in seconds
- `SSE_RECONNECT_INITIAL_BACKOFF` - Initial SSE reconnect backoff in seconds
- `SSE_RECONNECT_MAX_BACKOFF` - Maximum SSE reconnect backoff in seconds
- `SSE_READ_TIMEOUT` - SSE read timeout in seconds
- `FAILURE_THRESHOLD_FOR_ALERT` - Failures before system degraded alert
- At least one LLM API key: `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, or `GEMINI_API_KEY`

### Optional Configuration

- `LOG_LEVEL` - Logging level: `debug`, `info`, `warn`, `error`
- `AGENT_SYSTEM_PROMPT_FILE` - Path to system prompt file
- `AGENT_ALLOWED_TOOLS` - Comma-separated list of allowed tools
- `NOTIFY_ON_AGENT_FAILURE` - Send system degraded alerts (default: true)
- `UPLOAD_FAILED_INVESTIGATIONS` - Upload failed investigations (default: false)

### Tuning Configuration

The `configs/tuning.yaml` file contains operational parameters that rarely need adjustment. This file is **optional** - if not present, the application uses sensible defaults.

Tunable parameters include:
- **HTTP timeouts** - Slack webhook timeout (default: 10s)
- **Agent behavior** - Timeout buffer, minimum investigation size
- **Reporting** - Root cause truncation length, failure display count
- **Event processing** - Channel buffer sizes
- **I/O** - stdout/stderr buffer sizes

See `configs/tuning.yaml` for full documentation and default values.

### Migration from Previous Versions

**Breaking Change:** Nightcrier now requires explicit configuration for all operational parameters.

If upgrading from a version with implicit defaults:

1. Copy `configs/config.example.yaml` to `configs/config.yaml`
2. Fill in all required fields (see Required Configuration above)
3. Optionally create `configs/tuning.yaml` if you need to adjust operational parameters
4. Review environment variables - many previously optional parameters are now required
5. The application will fail fast on startup with clear error messages for any missing required fields

**Agent-Agnostic Design:** Environment variables now use generic names (`LLM_MODEL`, `AGENT_ALLOWED_TOOLS`) instead of Claude-specific names. Legacy Claude-specific variables are supported for backward compatibility but should be migrated.

#### Optional - Slack Notifications

- `SLACK_WEBHOOK_URL` - Slack webhook URL for notifications (if not set, Slack notifications are disabled)
- `NOTIFY_ON_AGENT_FAILURE` - Send system degraded alerts when agent failures occur (default: `true`)
- `FAILURE_THRESHOLD_FOR_ALERT` - Number of consecutive failures before sending alert (default: `3`)
- `UPLOAD_FAILED_INVESTIGATIONS` - Upload failed investigation attempts to storage (default: `false`)

#### Optional - Azure Blob Storage

When Azure storage is configured, incident artifacts are automatically uploaded to Azure Blob Storage and SAS URLs are generated for secure access. If Azure is not configured, the system falls back to filesystem storage.

##### Option 1: Connection String (Recommended)

- `AZURE_STORAGE_CONNECTION_STRING` - Full Azure connection string
- `AZURE_STORAGE_CONTAINER` - Blob container name (required)
- `AZURE_SAS_EXPIRY` - SAS URL expiration duration (default: `168h` / 7 days)

Example:
```bash
export AZURE_STORAGE_CONNECTION_STRING="DefaultEndpointsProtocol=https;AccountName=myaccount;AccountKey=xxxxx;EndpointSuffix=core.windows.net"
export AZURE_STORAGE_CONTAINER="incident-reports"
export AZURE_SAS_EXPIRY="168h"
```

##### Option 2: Account + Key

- `AZURE_STORAGE_ACCOUNT` - Storage account name
- `AZURE_STORAGE_KEY` - Storage account access key
- `AZURE_STORAGE_CONTAINER` - Blob container name (required)
- `AZURE_SAS_EXPIRY` - SAS URL expiration duration (default: `168h` / 7 days)

Example:
```bash
export AZURE_STORAGE_ACCOUNT="myaccount"
export AZURE_STORAGE_KEY="xxxxx"
export AZURE_STORAGE_CONTAINER="incident-reports"
```

### Storage Mode Detection

The system automatically detects which storage backend to use:
- **Azure Storage**: Used when `AZURE_STORAGE_ACCOUNT` or `AZURE_STORAGE_CONNECTION_STRING` is set
- **Filesystem Storage**: Used as fallback when Azure is not configured

### Azure Blob Storage Setup

1. Create a storage account in Azure Portal
2. Create a blob container for incident reports
3. Get connection string or account keys from the Azure Portal
4. Set environment variables as shown above

The system will:
- Upload all artifacts to `<container>/<incident-id>/` structure
- Generate SAS URLs with read-only access
- Include URLs in Slack notifications and result.json
- Set URL expiration based on `AZURE_SAS_EXPIRY`

### Container Requirements

The container must have the following structure:
```
<container>/
  <incident-id>/
    event.json              # Original fault event
    result.json             # Execution result with URLs
    output/
      investigation.md      # AI-generated investigation report
```

### Circuit Breaker and Agent Failure Handling

The system includes intelligent agent failure handling to prevent spurious notifications and improve reliability.

#### How It Works

When an AI agent executes, the system validates the output to ensure the agent successfully completed its investigation:

1. **Validation Checks**: Each agent execution is validated for:
   - Exit code is 0 (no execution errors)
   - Output file exists (`output/investigation.md`)
   - Output file size is substantial (> 100 bytes)

2. **Circuit Breaker**: If validation fails, the system records the failure and tracks consecutive failures:
   - **Closed State** (Normal): Agent failures are tracked but no system-level alerts are sent
   - **Open State** (Degraded): After reaching the failure threshold, a system degraded alert is sent

3. **Automatic Recovery**: When an agent successfully completes an investigation after the circuit breaker opened:
   - Circuit breaker resets to closed state
   - A system recovered alert is sent
   - Failure counter resets to zero

#### Configuration Options

Three environment variables control circuit breaker behavior:

```bash
# Enable/disable system degraded alerts (default: true)
export NOTIFY_ON_AGENT_FAILURE=true

# Number of consecutive failures before sending alert (default: 3)
export FAILURE_THRESHOLD_FOR_ALERT=3

# Upload failed investigations to storage (default: false)
export UPLOAD_FAILED_INVESTIGATIONS=false
```

In `config.yaml`:
```yaml
# Circuit breaker and failure notification configuration
notify_on_agent_failure: true
failure_threshold_for_alert: 3
upload_failed_investigations: false
```

#### Notification Behavior

**Individual Incident Notifications (per-incident):**
- Sent for successful investigations only
- Skipped when agent validation fails
- Prevents spam from failed LLM API calls or agent issues

**System Degraded Alerts (aggregated):**
- Sent when `failure_threshold_for_alert` consecutive failures occur
- Only sent if `notify_on_agent_failure` is `true`
- Includes failure statistics and recent failure reasons
- Indicates the AI agent system may be experiencing issues

**System Recovered Alerts:**
- Sent when agent successfully completes after circuit opened
- Includes total downtime and failure count
- Indicates system returned to healthy state

#### Storage Upload Behavior

By default (`upload_failed_investigations: false`):
- Only successful investigations are uploaded to storage
- Failed investigations remain in local workspace for debugging
- Reduces storage costs and prevents uploading incomplete data

When `upload_failed_investigations: true`:
- All investigations are uploaded, even if validation failed
- Useful for debugging agent issues
- Allows inspection of partial output

#### Example Flow

```
Event → Agent Execution → Validation → Outcome
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Event 1 → Agent runs → ✓ Valid → Upload + Notify
Event 2 → API error  → ✗ Failed → Skip upload/notify (failure 1/3)
Event 3 → Agent runs → ✗ Empty  → Skip upload/notify (failure 2/3)
Event 4 → Timeout    → ✗ Failed → Skip upload/notify (failure 3/3)
                                    ⚠️  SEND SYSTEM DEGRADED ALERT
Event 5 → Agent runs → ✓ Valid → Upload + Notify
                                    ✅ SEND SYSTEM RECOVERED ALERT
```

## Usage

### Running the Runner

```bash
# With environment variables
export K8S_CLUSTER_MCP_ENDPOINT="http://localhost:8080"
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
./runner

# With command-line flags
./runner --mcp-endpoint http://localhost:8080 --workspace-root ./incidents --log-level debug
```

### Command-line Flags

All configuration can be overridden via CLI flags:
- `--mcp-endpoint` - MCP server endpoint URL
- `--workspace-root` - Workspace root directory
- `--script-path` - Path to agent script
- `--log-level` - Log level (debug, info, warn, error)

## Local Development with Azurite

For local development and testing without an Azure account, use Azurite (Azure Storage Emulator).

### Using Docker Compose

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  azurite:
    image: mcr.microsoft.com/azure-storage/azurite:latest
    ports:
      - "10000:10000"  # Blob service
      - "10001:10001"  # Queue service
      - "10002:10002"  # Table service
    volumes:
      - azurite-data:/data
    command: azurite-blob --blobHost 0.0.0.0 --blobPort 10000 --location /data --debug /data/debug.log

volumes:
  azurite-data:
```

Start Azurite:
```bash
docker-compose up -d
```

### Configure for Azurite

Use the default Azurite connection string:

```bash
export AZURE_STORAGE_CONNECTION_STRING="DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://127.0.0.1:10000/devstoreaccount1;"
export AZURE_STORAGE_CONTAINER="incident-reports"
```

Create the container (one-time setup):
```bash
# Install Azure CLI or use Azure Storage Explorer
az storage container create --name incident-reports --connection-string "$AZURE_STORAGE_CONNECTION_STRING"
```

Or using curl:
```bash
curl -X PUT "http://127.0.0.1:10000/devstoreaccount1/incident-reports?restype=container" \
  -H "x-ms-date: $(date -u '+%a, %d %b %Y %H:%M:%S GMT')" \
  -H "x-ms-version: 2021-08-06"
```

### Verify Azurite Setup

```bash
# List containers
az storage container list --connection-string "$AZURE_STORAGE_CONNECTION_STRING" --output table

# After running an incident, list blobs
az storage blob list --container-name incident-reports --connection-string "$AZURE_STORAGE_CONNECTION_STRING" --output table
```

## Testing

### Unit Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/storage/... -v
go test ./internal/config/... -v
```

### Integration Tests

```bash
# Test storage backends
go test ./internal/storage/... -v

# Test with Azurite (requires Azurite running)
export AZURE_STORAGE_CONNECTION_STRING="DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://127.0.0.1:10000/devstoreaccount1;"
export AZURE_STORAGE_CONTAINER="test-incidents"
go test ./internal/storage/... -tags=integration
```

### End-to-End Testing

Since we can't trigger real Kubernetes incidents in a test environment, here's the recommended E2E testing approach:

#### 1. Setup Test Environment

```bash
# Start Azurite
docker-compose up -d

# Configure for Azurite
export AZURE_STORAGE_CONNECTION_STRING="DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://127.0.0.1:10000/devstoreaccount1;"
export AZURE_STORAGE_CONTAINER="incident-reports"
export K8S_CLUSTER_MCP_ENDPOINT="http://localhost:8080"
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"

# Create container
az storage container create --name incident-reports --connection-string "$AZURE_STORAGE_CONNECTION_STRING"

# Build and run
go build -o runner ./cmd/runner
./runner --log-level debug
```

#### 2. Trigger Test Event

Use the MCP server to send a test fault event. The runner will:
1. Receive the event
2. Create a workspace
3. Execute the AI agent
4. Upload artifacts to Azure/filesystem
5. Send Slack notification with report URL

#### 3. Verify Results

**Check Logs:**
```bash
# Look for these log entries:
# - "storage backend initialized" (mode: azure or filesystem)
# - "incident artifacts saved to storage"
# - "slack notification sent"
```

**Check Azure Storage:**
```bash
# List uploaded artifacts
az storage blob list \
  --container-name incident-reports \
  --connection-string "$AZURE_STORAGE_CONNECTION_STRING" \
  --output table

# Download and view investigation report
az storage blob download \
  --container-name incident-reports \
  --name "<incident-id>/output/investigation.md" \
  --file investigation.md \
  --connection-string "$AZURE_STORAGE_CONNECTION_STRING"
```

**Check Slack:**
- Verify notification received
- Click "View Report" button
- Confirm SAS URL works and report is accessible

**Check Result JSON:**
```bash
# View result with presigned URLs
cat ./incidents/<incident-id>/result.json

# Should contain:
# - "presigned_urls": {"event.json": "...", "investigation.md": "..."}
# - "presigned_urls_expire_at": "2025-12-25T..."
```

#### 4. Test URL Expiration

Wait for SAS URL to expire (or set short expiry for testing):
```bash
export AZURE_SAS_EXPIRY="1m"
```

Then verify URL becomes inaccessible after expiration.

#### 5. Test Filesystem Fallback

```bash
# Unset Azure config
unset AZURE_STORAGE_CONNECTION_STRING
unset AZURE_STORAGE_ACCOUNT
unset AZURE_STORAGE_KEY

# Run again
./runner

# Verify:
# - Logs show "mode: filesystem"
# - Artifacts saved to local filesystem
# - Slack notification shows file path instead of URL
```

## Output Structure

### Filesystem Storage
```
./incidents/
  <incident-id>/
    event.json              # Original fault event
    result.json             # Execution result
    output/
      investigation.md      # AI-generated investigation report
```

### Azure Storage
```
<container>/
  <incident-id>/
    event.json
    result.json
    output/
      investigation.md
```

Plus SAS URLs in result.json and Slack notifications.

## Slack Notification Format

When Slack is configured, notifications include:
- Incident metadata (cluster, namespace, resource)
- Root cause analysis with confidence level
- Investigation duration
- **"View Report" button** (when Azure storage is enabled)
- File path (when filesystem storage is used)

## Troubleshooting

### Agent Failures

**Problem**: Receiving "AI Agent System Degraded" alerts in Slack
- This indicates the circuit breaker threshold has been reached (default: 3 consecutive failures)
- Common causes:
  - LLM API key issues (expired, missing, or incorrect)
  - LLM API rate limiting or service outages
  - Agent timeout (default 300 seconds)
  - Network connectivity issues
  - Resource constraints (CPU, memory)

**Diagnosis Steps**:

1. Check runner logs for failure details:
```bash
# Look for agent failure messages
./runner --log-level debug

# Sample log output:
# WARN agent execution failed validation incident_id=abc-123 reason="agent exited with non-zero code: 1"
# WARN circuit breaker threshold reached, system degraded failure_count=3
```

2. Check the workspace for investigation artifacts:
```bash
# List failed incidents
ls -la ./incidents/

# Check specific incident
cat ./incidents/<incident-id>/incident.json

# Check if output directory exists and has content
ls -la ./incidents/<incident-id>/output/
cat ./incidents/<incident-id>/output/investigation.md
```

3. Verify LLM API key configuration:
```bash
# Check if API key is set
echo $ANTHROPIC_API_KEY | wc -c  # Should be > 50 characters
echo $OPENAI_API_KEY | wc -c
echo $GEMINI_API_KEY | wc -c

# Test API key validity (for Anthropic)
curl https://api.anthropic.com/v1/messages \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "content-type: application/json" \
  -d '{"model":"claude-3-5-sonnet-20241022","max_tokens":10,"messages":[{"role":"user","content":"test"}]}'
```

4. Check agent execution logs:
```bash
# If using docker, check container logs
docker ps -a | grep triage-agent
docker logs <container-id>
```

**Common Solutions**:

1. **API Key Issues**: Verify correct API key is set and not expired
2. **Rate Limiting**: Increase delay between incidents or upgrade API tier
3. **Timeouts**: Increase `AGENT_TIMEOUT` (default: 300s)
```bash
export AGENT_TIMEOUT=600
```
4. **Network Issues**: Check firewall rules and proxy settings
5. **Resource Constraints**: Increase container memory/CPU limits

**Recovery**:
Once the underlying issue is fixed, the system will automatically detect the next successful investigation and send a "System Recovered" alert.

**Problem**: Agent failures not triggering system alerts
- Check `NOTIFY_ON_AGENT_FAILURE` is set to `true` (default)
- Verify `SLACK_WEBHOOK_URL` is configured
- Check failure count hasn't reached threshold yet (default: 3 consecutive failures)

**Problem**: Want to inspect failed investigation artifacts
- Set `UPLOAD_FAILED_INVESTIGATIONS=true` to upload failed attempts to storage
- Failed investigations remain in local workspace: `./incidents/<incident-id>/`

### Azure Storage Issues

**Problem**: "failed to initialize Azure storage"
- Check connection string format
- Verify account name and key are correct
- Ensure container exists
- Check network connectivity to Azure

**Problem**: "failed to upload blob"
- Verify container exists and has correct permissions
- Check storage account access keys
- Ensure container name is lowercase (Azure requirement)

**Problem**: SAS URL returns 403 Forbidden
- Check URL hasn't expired
- Verify SAS token permissions (should include read)
- Ensure blob exists at the path

### Azurite Issues

**Problem**: Connection refused to Azurite
- Ensure Azurite is running: `docker-compose ps`
- Check port 10000 is accessible
- Verify connection string uses `http://` not `https://`

**Problem**: Container not found
- Create container: `az storage container create --name incident-reports --connection-string "$AZURE_STORAGE_CONNECTION_STRING"`
- Verify with: `az storage container list --connection-string "$AZURE_STORAGE_CONNECTION_STRING"`

### Debug Mode

Enable debug logging to see detailed storage operations:
```bash
./runner --log-level debug
```

Look for these log entries:
- "storage backend initialized" - Shows which backend is active
- "incident artifacts saved to storage" - Shows successful upload
- "failed to save incident to storage" - Shows upload errors

## Multi-Cluster Troubleshooting

### Connection Issues

**Problem**: MCP server unreachable or connection timeouts

**Symptoms**:
```
level=ERROR msg="cluster connection failed" cluster=prod-us-east-1 error="dial tcp: connection refused"
level=INFO msg="cluster reconnection scheduled" cluster=prod-us-east-1 retry_in="2s"
```

**Diagnosis Steps**:

1. Test MCP endpoint from Nightcrier host:
```bash
curl -v http://kubernetes-mcp-server:8080/mcp
# Should return: 405 Method Not Allowed (MCP server only accepts SSE connections)
```

2. Check kubernetes-mcp-server is running:
```bash
kubectl get pods -n mcp-system
kubectl logs -n mcp-system deployment/kubernetes-mcp-server
```

3. Verify network connectivity:
```bash
# If MCP server is in-cluster
kubectl get svc -n mcp-system kubernetes-mcp-server

# If MCP server is external
ping <mcp-server-host>
telnet <mcp-server-host> 8080
```

4. Check firewall rules and network policies

**Solutions**:
- Verify endpoint URL in config (include `/mcp` path)
- Check DNS resolution for cluster service names
- Ensure network policies allow ingress to MCP server
- Verify MCP server logs show no errors
- Connection will automatically retry with exponential backoff

### Permission Issues

**Problem**: kubectl auth can-i failures during startup

**Symptoms**:
```
level=ERROR msg="permission validation failed" cluster=prod error="kubectl auth can-i failed: exit status 1"
```

**Diagnosis Steps**:

1. Test kubeconfig manually:
```bash
KUBECONFIG=./kubeconfigs/prod-us-east-1-readonly.yaml kubectl get pods
```

2. Check ServiceAccount exists:
```bash
kubectl get sa -n kube-system kubernetes-triage-readonly
kubectl get secret -n kube-system kubernetes-triage-readonly-token
```

3. Verify RBAC bindings:
```bash
kubectl get clusterrolebinding | grep kubernetes-triage
kubectl describe clusterrolebinding kubernetes-triage-readonly-view
```

4. Test specific permissions:
```bash
KUBECONFIG=./kubeconfigs/prod-readonly.yaml \
  kubectl auth can-i get pods
KUBECONFIG=./kubeconfigs/prod-readonly.yaml \
  kubectl auth can-i get pods/log
```

**Solutions**:
- Recreate ServiceAccount and token secret
- Reapply RBAC ClusterRoleBindings
- Regenerate kubeconfig using extraction script
- Check ServiceAccount token hasn't expired
- Verify kubectl is in PATH and accessible

**Problem**: Insufficient permissions warning

**Symptoms**:
```
level=WARN msg="cluster has permission warnings" cluster=prod warnings="cannot get nodes (cluster-wide visibility limited)"
```

**Impact**:
- Agent will still run but with limited investigation capabilities
- Some diagnostic commands may fail inside agent container
- Investigation report may be incomplete

**Solutions**:
- Grant additional RBAC permissions (nodes, secrets, etc.)
- Accept limited functionality for security-conscious deployments
- Review `incident_cluster_permissions.json` to see exact limitations

### Kubeconfig Problems

**Problem**: Kubeconfig file not found

**Symptoms**:
```
level=ERROR msg="failed to initialize connection manager" error="cluster prod-us-east-1: kubeconfig not found: ./kubeconfigs/prod-readonly.yaml"
```

**Solutions**:
- Verify file exists: `ls -la ./kubeconfigs/`
- Check file path in config matches actual filename
- Ensure kubeconfig was extracted using setup script
- Check file permissions (should be readable by Nightcrier process)

**Problem**: Kubeconfig authentication fails

**Symptoms**:
```
level=ERROR msg="permission validation failed" error="Unable to connect to the server: x509: certificate signed by unknown authority"
```

**Solutions**:
- Verify cluster CA certificate in kubeconfig
- Test with kubectl: `kubectl --kubeconfig=<file> cluster-info`
- Regenerate kubeconfig from ServiceAccount token
- Check cluster API server is accessible from Nightcrier host

### Triage Disabled Behavior

**Problem**: Events received but no investigations performed

**Symptoms**:
```
level=INFO msg="fault event received" cluster=staging namespace=default resource=pod/webapp
level=INFO msg="triage disabled for cluster - skipping agent execution" cluster=staging reason="triage.enabled=false or no kubeconfig"
```

**Expected Behavior**: This is intentional when `triage.enabled: false`

**When this happens**:
- Fault events are logged for visibility
- No workspace is created
- No AI agent is spawned
- No Slack notification is sent
- No storage upload occurs

**To enable triage**:
1. Create kubeconfig for cluster (see Kubeconfig Setup section)
2. Set `triage.enabled: true` in config
3. Restart Nightcrier
4. Verify permission validation succeeds

### Reading Logs for Cluster-Specific Issues

**Enable debug logging**:
```bash
./nightcrier --log-level debug
```

**Key log fields to monitor**:
- `cluster` - Which cluster the event came from
- `incident_id` - Unique incident identifier
- `kubeconfig` - Which kubeconfig is being used
- `minimum_met` - Whether minimum permissions are satisfied
- `triage_enabled` - Whether triage is enabled for cluster

**Example log analysis**:

```bash
# Filter logs for specific cluster
./nightcrier 2>&1 | grep 'cluster=prod-us-east-1'

# Check permission validation results
./nightcrier 2>&1 | grep 'permissions validated'

# Monitor connection status
./nightcrier 2>&1 | grep 'cluster connection'

# Track triage skip events
./nightcrier 2>&1 | grep 'triage disabled'
```

### Understanding Connection Status

Connection lifecycle states:

| State | Meaning | Next Steps |
|-------|---------|------------|
| `disconnected` | Initial state | Connecting... |
| `connecting` | TCP connection in progress | Wait for connected |
| `connected` | Connected to MCP server | Subscribing... |
| `subscribing` | Requesting fault event stream | Wait for active |
| `active` | Receiving events | Normal operation |
| `failed` | Connection error occurred | Auto-retry with backoff |

**Check connection status** (future feature in Phase 4):
```bash
curl http://localhost:9090/health/clusters
```

**Reconnection behavior**:
- Initial backoff: 1 second
- Maximum backoff: 60 seconds
- Multiplier: 2.0 (exponential)
- Jitter: 10% (randomization)
- Continues indefinitely until successful

### incident_cluster_permissions.json File

Each incident workspace contains cluster permission information:

**File location**:
```
./incidents/<incident-id>/incident_cluster_permissions.json
```

**Example contents**:
```json
{
  "cluster_name": "prod-us-east-1",
  "validated_at": "2025-12-20T10:30:00Z",
  "can_get_pods": true,
  "can_get_logs": true,
  "can_get_events": true,
  "can_get_deployments": true,
  "can_get_services": true,
  "secrets_access_allowed": false,
  "can_get_secrets": false,
  "can_get_configmaps": false,
  "can_get_nodes": true,
  "warnings": []
}
```

**Use cases**:
- AI agent reads this to understand available capabilities
- Operators can verify what permissions were active during incident
- Debugging permission-related investigation failures
- Audit trail of cluster access permissions

**Warnings examples**:
```json
"warnings": [
  "cannot get nodes (cluster-wide visibility limited)",
  "secrets access disabled by config (set triage.allow_secrets_access=true for Helm debugging)"
]
```

## Contributing

See [openspec/AGENTS.md](openspec/AGENTS.md) for development workflow and contribution guidelines.

## License

[Add your license here]

## Related Projects

- [kubernetes-mcp-server](https://github.com/rbias/kubernetes-mcp-server) - MCP server for Kubernetes fault events
- [Model Context Protocol](https://github.com/anthropics/mcp) - Protocol specification
- [Nightcrier](https://github.com/rbias/nightcrier) - This project
