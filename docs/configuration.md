# Configuration

Nightcrier uses explicit configuration with no hardcoded defaults. All required parameters must be provided via configuration file, environment variables, or command-line flags.

## Configuration Files

- **`configs/config.yaml`** - Main configuration file (copy from `configs/config.example.yaml`)
- **`configs/tuning.yaml`** - Optional tuning parameters for operational adjustments (rarely changed)
- **`kubeconfigs/`** - Directory containing cluster kubeconfig files for triage agent access

## Configuration Precedence

Configuration values are loaded in the following order (highest to lowest priority):
1. Command-line flags (e.g., `--mcp-endpoint`)
2. Environment variables (e.g., `K8S_CLUSTER_MCP_ENDPOINT`)
3. Configuration file (`config.yaml`)
4. Tuning file (`tuning.yaml`, optional)

## Required Configuration

The following parameters **must** be provided. The application will fail fast on startup if any are missing:

### Core Settings
- `WORKSPACE_ROOT` - Directory for incident artifacts (e.g., `./workspaces`)
- `SEVERITY_THRESHOLD` - Minimum event severity: `DEBUG`, `INFO`, `WARNING`, `ERROR`, `CRITICAL`

### Agent Configuration
- `AGENT_MODEL` - LLM model to use (e.g., `claude-sonnet-4-5-20250929`, `gpt-4o`, `haiku`)
- `AGENT_TIMEOUT` - Agent timeout in seconds (e.g., `600`)
- `AGENT_CLI` - AI CLI tool to use: `claude`, `codex`, `goose`, or `gemini`
- At least one LLM API key: `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, or `GEMINI_API_KEY`

### K8s Executor Configuration
- `K8S_NAMESPACE` - Namespace for Jobs and ConfigMaps (default: `nightcrier`)
- `K8S_IMAGE` - Container image for agent Jobs (e.g., `nc-agent-runner:latest`)
- `K8S_TIMEOUT` - Job timeout in seconds (default: `600`)
- `K8S_MEMORY_LIMIT` - Container memory limit (default: `2Gi`)
- `K8S_CPU_LIMIT` - Container CPU limit (default: `1`)
- `KUBECONFIG_PATH` - Path to kubeconfig for executor cluster access
- `KUBERNETES_CONTEXT` - Kubernetes context to use for executor

### Event Processing
- `MAX_CONCURRENT_AGENTS` - Maximum concurrent agent sessions
- `GLOBAL_QUEUE_SIZE` - Global event queue size
- `CLUSTER_QUEUE_SIZE` - Per-cluster queue size
- `DEDUP_WINDOW_SECONDS` - Event deduplication window (0 to disable)
- `QUEUE_OVERFLOW_POLICY` - Queue overflow policy: `drop` or `reject`
- `SHUTDOWN_TIMEOUT` - Graceful shutdown timeout in seconds

### SSE Connection Settings
- `SSE_RECONNECT_INITIAL_BACKOFF` - Initial SSE reconnect backoff in seconds
- `SSE_RECONNECT_MAX_BACKOFF` - Maximum SSE reconnect backoff in seconds
- `SSE_READ_TIMEOUT` - SSE read timeout in seconds

### Circuit Breaker
- `FAILURE_THRESHOLD_FOR_ALERT` - Failures before system degraded alert

## Optional Configuration

- `LOG_LEVEL` - Logging level: `debug`, `info`, `warn`, `error`
- `AGENT_SYSTEM_PROMPT_FILE` - Path to system prompt file
- `AGENT_ALLOWED_TOOLS` - Comma-separated list of allowed tools
- `NOTIFY_ON_AGENT_FAILURE` - Send system degraded alerts (default: true)
- `UPLOAD_FAILED_INVESTIGATIONS` - Upload failed investigations (default: false)

### Slack Notifications

- `SLACK_WEBHOOK_URL` - Slack webhook URL for notifications (if not set, Slack notifications are disabled)
- `NOTIFY_ON_AGENT_FAILURE` - Send system degraded alerts when agent failures occur (default: `true`)
- `FAILURE_THRESHOLD_FOR_ALERT` - Number of consecutive failures before sending alert (default: `3`)
- `UPLOAD_FAILED_INVESTIGATIONS` - Upload failed investigation attempts to storage (default: `false`)

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

## Tuning Configuration

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
