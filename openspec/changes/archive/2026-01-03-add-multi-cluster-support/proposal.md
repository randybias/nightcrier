# Proposal: Multi-Cluster MCP Server Support

## Summary

Enable Nightcrier to connect to multiple kubernetes-mcp-server instances simultaneously, allowing centralized incident triage across many Kubernetes clusters. Each cluster has two credential sets: MCP server endpoint (with future API key auth) and kubeconfig for direct cluster access during triage. Kubernetes permissions for the AI agent performing triage are intentionally restricted to primarily read-only access.

## Motivation

Production environments typically span multiple Kubernetes clusters:
- Development, staging, production environments
- Regional deployments (us-east, eu-west, etc.)
- Multi-tenant clusters
- Specialized clusters (GPU, high-memory, etc.)

Currently, Nightcrier only connects to a single MCP endpoint. Operators must run separate instances to monitor multiple clusters, leading to:
- Operational overhead managing many instances
- No unified view of incidents
- Wasted resources (each instance has its own agent pool)

Additionally, the triage agents currently cannot perform meaningful investigation because they lack proper cluster credentials. This proposal addresses both multi-cluster support and the credential model needed for real triage.

## Design Goals

1. **Centralized Management**: Single Nightcrier instance monitors all clusters
2. **Working Triage**: Agents can actually connect to clusters and investigate
3. **Explicit Credentials**: Never guess credentials; fail-closed when absent
4. **Scalability**: Support 10 servers initially, 100+ at scale
5. **Resilience**: Individual server failures don't affect others

## Core Rule

**Nightcrier never guesses credentials.**

If a cluster entry does not explicitly provide a kubeconfig with `triage.enabled: true`, triage is disabled for that cluster. Events are still received via MCP, but no agent is spawned.

## Architecture

### Credential Model

Each cluster Nightcrier monitors has two distinct credential sets:

| Credential | Purpose | Location |
|------------|---------|----------|
| MCP API Key | Authenticate to kubernetes-mcp-server | Config file (placeholder for now) |
| Kubeconfig | Direct cluster API access for triage agents | `./kubeconfigs/<cluster>.yaml` |

The kubeconfig should be bound to a dedicated `kubernetes-triage-readonly` ServiceAccount with:
- `view` ClusterRole (get, list, watch)
- Helm read-only access (get secrets in release namespaces)
- NO mutation verbs

### Configuration

Clusters are defined in the main configuration file:

```yaml
# config.yaml
clusters:
  - name: prod-us-east-1
    environment: production

    mcp:
      endpoint: http://kubernetes-mcp-server.mcp-system.svc.cluster.local:8080/mcp
      api_key: THIS_IS_A_PLACEHOLDER_TO_REMIND_US_TO_MAKE_AUTH_WORK_ON_THE_MCP_SERVER

    triage:
      enabled: true
      kubeconfig: ./kubeconfigs/prod-us-east-1-readonly.yaml
      # Allow agent to read secrets/configmaps for Helm debugging
      # Default: false (disabled for security - secrets may contain credentials)
      # When enabled, agent can run helm_release_debug.sh and access Helm release data
      allow_secrets_access: false

  - name: staging-eu-west-1
    environment: staging

    mcp:
      endpoint: http://10.42.0.23:8080/mcp
      api_key: THIS_IS_A_PLACEHOLDER_TO_REMIND_US_TO_MAKE_AUTH_WORK_ON_THE_MCP_SERVER

    triage:
      enabled: false
      # No kubeconfig - events received but not triaged
```

### Secrets Access Trade-off

The `allow_secrets_access` option is a conscious trade-off:

**When disabled (default):**
- Agent cannot access Helm release data or other secrets
- Safer, but Helm debugging capabilities are limited
- Agent will see a warning in `incident_cluster_permissions.json`

**When enabled:**
- Agent can read all secrets/configmaps (cluster-wide or namespace-scoped depending on RBAC)
- Enables full Helm debugging with `helm_release_debug.sh`
- Risk: LLM sees actual secret contents (credentials, keys, etc.)

**Future improvements (out of scope):**
- kubernetes-mcp-server could expose restricted Helm queries that return metadata without secret values
- Dynamic permission escalation with operator approval workflow

### Kubeconfig Location

Kubeconfigs are stored in `./kubeconfigs/` directory (parallel to `./configs/` and `./incidents/`):

```
nightcrier/
├── configs/
│   └── config.yaml
├── kubeconfigs/
│   ├── prod-us-east-1-readonly.yaml
│   └── staging-eu-west-1-readonly.yaml
├── incidents/
│   └── <incident-id>/
│       ├── incident.json
│       ├── incident_cluster_permissions.json   # NEW
│       └── output/
└── ...
```

### Preflight Validation

On startup, for each cluster with `triage.enabled: true`, Nightcrier:

1. Validates kubeconfig file exists and is readable
2. Runs permission check: `kubectl auth can-i --list`
3. Records available permissions in memory
4. Logs warning if minimum permissions are missing

When spawning a triage agent, the recorded permissions are written to `incident_cluster_permissions.json` in the workspace, giving the agent context about what it can and cannot do.

### Connection Manager

A `ConnectionManager` component manages the lifecycle of multiple MCP connections:

```
┌─────────────────────────────────────────────────────────────────┐
│                        Nightcrier                                │
│                                                                  │
│  ┌────────────────────────────────────────────────────────┐     │
│  │              Connection Manager                         │     │
│  │                                                         │     │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐      ┌─────────┐│     │
│  │  │ MCP     │  │ MCP     │  │ MCP     │ ...  │ MCP     ││     │
│  │  │ Client  │  │ Client  │  │ Client  │      │ Client  ││     │
│  │  │ (prod)  │  │ (stage) │  │ (dev)   │      │ (N)     ││     │
│  │  └────┬────┘  └────┬────┘  └────┬────┘      └────┬────┘│     │
│  └───────┼────────────┼───────────┼────────────────┼─────┘     │
│          │            │           │                │            │
│          └────────────┴─────┬─────┴────────────────┘            │
│                             ▼                                    │
│                    ┌────────────────┐                           │
│                    │  Event Router  │                           │
│                    │  (fan-in)      │                           │
│                    └───────┬────────┘                           │
│                            ▼                                     │
│            ┌───────────────────────────────┐                    │
│            │     Global Agent Pool         │                    │
│            │  (max_concurrent_agents=20)   │                    │
│            └───────────────────────────────┘                    │
└─────────────────────────────────────────────────────────────────┘
```

### Event Flow

```
Event from cluster: prod-us-east-1
↓
Is triage.enabled?
  ├─ no → log "triage disabled for cluster" + skip
  └─ yes
       ↓
Create workspace with:
  - incident.json
  - incident_cluster_permissions.json
       ↓
Spawn triage agent with:
  - KUBECONFIG pointing to cluster-specific kubeconfig
       ↓
Run diagnostics
```

### Connection Lifecycle

Each cluster connection follows this lifecycle:

```
DISCONNECTED → CONNECTING → CONNECTED → SUBSCRIBING → ACTIVE
      ↑            │             │            │          │
      └────────────┴─────────────┴────────────┴──────────┘
                        (on error)
```

With exponential backoff on reconnection (1s initial, 60s max).

## Implementation Phases

### Phase 1: Array Configuration (Start Here)

Add `clusters` array to configuration, initially with single cluster:

- Define `ClusterConfig` struct with nested `mcp` and `triage` sections
- Add `clusters []ClusterConfig` field to main Config
- Validate cluster config (name uniqueness, required fields)
- Create `./kubeconfigs/` directory convention
- Update example config with new structure

At end of Phase 1: Config parses correctly but behavior unchanged.

### Phase 2: Refactor to Use Array

Convert existing single-endpoint code to use clusters array:

- Create `ConnectionManager` to manage cluster connections
- Refactor `events.Client` usage to iterate over clusters
- Add cluster name to event metadata and logging
- Update agent executor to receive cluster-specific kubeconfig
- Ensure single cluster still works identically

At end of Phase 2: Works with array of 1, behavior unchanged.

### Phase 3: Enable Real Triage

Make triage agents actually work:

- Implement preflight permission validation
- Create `incident_cluster_permissions.json` writer
- Pass cluster kubeconfig to `run-agent.sh`
- Update workspace creation to include permissions file
- Test with real cluster to verify agent can run kubectl commands

At end of Phase 3: Triage agents produce meaningful output.

### Phase 4: Multi-Cluster Validation

Expand to multiple clusters:

- Test with 2-3 clusters simultaneously
- Verify independent connection lifecycles
- Verify event fan-in works correctly
- Verify correct kubeconfig used per cluster
- Add health monitoring endpoint

At end of Phase 4: Production-ready multi-cluster support.

### Future Phases (Out of Scope)

- Config hot-reload (SIGHUP)
- Per-cluster metrics (Prometheus)
- Per-cluster overrides (severity thresholds, agent model)
- MCP API key authentication (when kubernetes-mcp-server supports it)
- Scale testing (100+ clusters)

## Open Questions (Resolved)

1. **Kubeconfig Distribution**: Store in `./kubeconfigs/` directory, managed outside Nightcrier.

2. **RBAC Profile**: Deferred. Not needed for proof of concept.

3. **MCP Authentication**: Placeholder field in config. Implementation when kubernetes-mcp-server adds auth.

4. **Single-cluster backwards compatibility**: Not maintained. Migrate to clusters array.

## Decision

Proceed with clusters array configuration, phased implementation starting with single-cluster refactor, then enabling real triage, then expanding to multiple clusters.
