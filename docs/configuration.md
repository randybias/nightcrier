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

### Execution Cluster Configuration
At least one `execution_clusters[]` entry must be configured in the YAML config file.
The first execution cluster's kubeconfig is used for bootstrap and Job execution.

Default values can be set via `execution_defaults`:
- `EXECUTION_DEFAULTS_NAMESPACE` - Namespace for Jobs and ConfigMaps (default: `nightcrier`)
- `EXECUTION_DEFAULTS_RUNNER_IMAGE` - Container image for agent Jobs (default: `nc-agent-runner:latest`)
- `EXECUTION_DEFAULTS_TIMEOUT` - Job timeout in seconds (default: `600`)
- `EXECUTION_DEFAULTS_MEMORY_LIMIT` - Container memory limit (default: `2Gi`)
- `EXECUTION_DEFAULTS_CPU_LIMIT` - Container CPU limit (default: `1`)

### Event Processing and Concurrency
- `MAX_CONCURRENT_AGENTS` - Maximum number of concurrent agent executions globally (default: `10`)
- `CLUSTER_QUEUE_SIZE` - Maximum queued events per cluster before dropping oldest (default: `10`)
- `EVENT_TTL_SECONDS` - Events older than this are dropped as stale (default: `300`)
- `DEDUP_WINDOW_SECONDS` - Event deduplication window (0 to disable)
- `SHUTDOWN_TIMEOUT` - Graceful shutdown timeout in seconds

### MCP Connection Settings
- `MCP_RECONNECT_INITIAL_BACKOFF` - Initial MCP reconnect backoff in seconds
- `MCP_RECONNECT_MAX_BACKOFF` - Maximum MCP reconnect backoff in seconds
- `MCP_READ_TIMEOUT_SECONDS` - MCP read timeout in seconds

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

Nightcrier supports monitoring multiple Kubernetes clusters simultaneously through a single instance. The configuration separates **monitored clusters** (where faults are detected) from **execution clusters** (where triage agent Jobs run).

### Configuration Structure

Define clusters in `configs/config.yaml`:

```yaml
# Execution clusters - where triage agent Jobs run
execution_clusters:
  - name: triage-west
    kubeconfig_path: "./kubeconfigs/triage-executor.yaml"
    namespace: "nightcrier"
    runner_image: "nc-agent-runner:latest"
    max_concurrent_agents: 10

# Monitored clusters - where faults are detected
monitored_clusters:
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
      target_kubeconfig_path: ./kubeconfigs/prod-us-east-1-readonly.yaml
      allow_secrets_access: false
      execution_cluster: "triage-west"  # optional, uses default if omitted

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
      target_kubeconfig_path: ./kubeconfigs/staging-eu-west-1-readonly.yaml
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

### Execution Cluster Fields

- `name` (required) - Unique identifier for the execution cluster
- `kubeconfig_path` (required) - Path to kubeconfig for this execution cluster
- `namespace` (optional, default: "nightcrier") - Namespace for Jobs
- `runner_image` (optional, default: "nc-agent-runner:latest") - Container image
- `image_pull_policy` (optional, default: "IfNotPresent") - Image pull policy
- `timeout` (optional, default: 600) - Job timeout in seconds
- `memory_limit` (optional, default: "2Gi") - Container memory limit
- `cpu_limit` (optional, default: "1") - Container CPU limit
- `cleanup_ttl` (optional, default: 3600) - Job cleanup TTL in seconds
- `max_concurrent_agents` (optional, default: 10) - Max concurrent agents

### Monitored Cluster Fields

**Cluster-level**:
- `name` (required) - Unique cluster identifier, used in logs and incident metadata
- `environment` (optional) - Environment label (production, staging, development)
- `labels` (optional) - Custom key-value labels for organization

**MCP Configuration**:
- `mcp.endpoint` (required) - kubernetes-mcp-server URL with `/mcp` path
- `mcp.api_key` (optional) - Placeholder for future MCP authentication

**Triage Configuration**:
- `triage.enabled` (required) - Enable/disable AI triage for this cluster
- `triage.target_kubeconfig_path` (required if enabled) - Path to read-only kubeconfig for agent access to the target cluster
- `triage.allow_secrets_access` (optional, default: false) - Allow agent to read secrets/configmaps
- `triage.execution_cluster` (optional) - Pin triage to a specific execution cluster; uses first configured if omitted

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

## Agent Concurrency Configuration

Nightcrier uses a sophisticated concurrency model to balance parallelism with cluster-level ordering guarantees.

### Concurrency Settings

| Setting | Environment Variable | Default | Description |
|---------|---------------------|---------|-------------|
| `max_concurrent_agents` | `MAX_CONCURRENT_AGENTS` | `10` | Maximum number of concurrent agent executions globally |
| `cluster_queue_size` | `CLUSTER_QUEUE_SIZE` | `10` | Maximum queued events per cluster before dropping oldest |
| `event_ttl_seconds` | `EVENT_TTL_SECONDS` | `300` | Events older than this (in seconds) are dropped as stale |

### Example Configuration

```yaml
# In configs/config.yaml
max_concurrent_agents: 10
cluster_queue_size: 10
event_ttl_seconds: 300
```

Or via environment variables:

```bash
export MAX_CONCURRENT_AGENTS=10
export CLUSTER_QUEUE_SIZE=10
export EVENT_TTL_SECONDS=300
```

### Concurrency Behavior

The concurrency model provides these guarantees:

1. **Cross-cluster parallelism**: Events from different clusters run in parallel (up to `max_concurrent_agents`)
2. **Per-cluster serialization**: Events for the same cluster are strictly serialized (one at a time)
3. **Non-blocking ingestion**: Event ingestion never blocks the MCP event loop
4. **Stale event dropping**: Events older than `event_ttl_seconds` are dropped before execution
5. **Queue overflow handling**: When a cluster queue is full, the oldest event is dropped to make room for newer (more relevant) events

### Tuning Guidelines

- **High-volume environments**: Increase `max_concurrent_agents` for more parallelism across clusters
- **Resource-constrained environments**: Decrease `max_concurrent_agents` to limit resource usage
- **Bursty event patterns**: Increase `cluster_queue_size` to buffer more events per cluster
- **Slow investigations**: Increase `event_ttl_seconds` if agent investigations take longer than 5 minutes

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
