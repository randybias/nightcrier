## Context

Nightcrier currently uses a single `clusters:` array that combines monitoring and triage concepts. The `k8s:` section is a global executor configuration. This design lacks:
1. Clear separation between where faults are detected vs where agents run
2. Ability to pin monitored clusters to specific execution clusters
3. Runtime cluster management via database
4. Hot-reload capability for cluster changes

### Stakeholders
- Operators managing multiple Kubernetes clusters
- DevOps teams adding/removing clusters without restarts
- Platform teams running multi-tenant Nightcrier deployments

## Goals / Non-Goals

### Goals
- Clear, unambiguous configuration keys
- Separation of monitored clusters from execution clusters
- Database-backed cluster storage with YAML fallback
- SIGHUP-triggered full configuration reload
- Support starting with zero configured clusters

### Non-Goals
- UI for cluster management (API/DB only)
- Multi-executor load balancing (pin to single executor or use default)
- Encrypted kubeconfig storage in database (use DB-level encryption)
- Backwards compatibility with old config format

## Decisions

### Decision 1: Two-Section Configuration Structure

**New structure:**
```yaml
execution_clusters:
  - name: triage-west
    kubeconfig_path: "/path/to/executor.kubeconfig"
    namespace: "nightcrier"
    runner_image: "nc-agent-runner:latest"
    max_concurrent_agents: 10

monitored_clusters:
  - name: eastus-cluster1
    environment: testing
    mcp:
      endpoint: "http://.../mcp"
    triage:
      enabled: true
      target_kubeconfig_path: "/path/to/eastus-readonly.kubeconfig"
      allow_secrets_access: false
      execution_cluster: "triage-west"  # optional, uses default if omitted
```

**Rationale:**
- `execution_clusters` contains settings moved from current `k8s:` section
- `monitored_clusters` replaces `clusters:` with clearer naming
- `triage.target_kubeconfig_path` clarifies the kubeconfig is for agent access to target cluster
- `execution_cluster` reference allows pinning monitored clusters to specific executors

**Alternatives considered:**
- Keep single `clusters:` with embedded executor config - rejected: conflates concerns
- Nested execution config per monitored cluster - rejected: would duplicate executor settings

### Decision 2: Global K8s Section Becomes Defaults

The current `k8s:` section becomes `execution_defaults:` providing defaults for `execution_clusters[]`:

```yaml
execution_defaults:
  namespace: "nightcrier"
  runner_image: "nc-agent-runner:latest"
  image_pull_policy: "IfNotPresent"
  timeout: 600
  memory_limit: "2Gi"
  cpu_limit: "1"
  cleanup_ttl: 3600
```

Individual `execution_clusters[]` entries can override any of these.

### Decision 3: Database Storage with Full Inline Credentials

**Schema for monitored_clusters table:**
```sql
CREATE TABLE monitored_clusters (
    name TEXT PRIMARY KEY,
    environment TEXT,
    labels JSONB,
    mcp_endpoint TEXT NOT NULL,
    mcp_api_key TEXT,
    triage_enabled BOOLEAN NOT NULL DEFAULT false,
    target_kubeconfig TEXT,  -- Full kubeconfig YAML content
    allow_secrets_access BOOLEAN NOT NULL DEFAULT false,
    execution_cluster TEXT,  -- FK to execution_clusters.name
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    source TEXT NOT NULL DEFAULT 'database'  -- 'yaml' or 'database'
);

CREATE TABLE execution_clusters (
    name TEXT PRIMARY KEY,
    kubeconfig TEXT NOT NULL,  -- Full kubeconfig YAML content
    namespace TEXT NOT NULL DEFAULT 'nightcrier',
    runner_image TEXT NOT NULL DEFAULT 'nc-agent-runner:latest',
    image_pull_policy TEXT NOT NULL DEFAULT 'IfNotPresent',
    timeout INTEGER NOT NULL DEFAULT 600,
    memory_limit TEXT NOT NULL DEFAULT '2Gi',
    cpu_limit TEXT NOT NULL DEFAULT '1',
    cleanup_ttl INTEGER NOT NULL DEFAULT 3600,
    max_concurrent_agents INTEGER NOT NULL DEFAULT 10,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    source TEXT NOT NULL DEFAULT 'database'
);
```

**Rationale:**
- Inline kubeconfig storage simplifies runtime management (no external file dependencies)
- `source` column tracks whether cluster came from YAML or was added via database
- Database-sourced clusters take precedence over YAML on reload

### Decision 4: SIGHUP Full Configuration Reload

On SIGHUP:
1. Re-read YAML configuration file
2. Re-read all clusters from database
3. Database clusters override YAML clusters (by name)
4. Apply full configuration changes:
   - Stop connections to removed monitored clusters
   - Start connections to new monitored clusters
   - Update execution cluster configurations
   - Reload agent settings, storage settings, etc.
5. Log all changes at INFO level

**Rationale:**
- Full reload is cleaner than partial updates
- Database takes precedence to enable runtime management
- Graceful connection management prevents event loss

### Decision 5: Zero Clusters Startup Mode

System starts successfully even with zero configured clusters:
- Log warning: "No clusters configured - waiting for cluster configuration"
- Enter monitoring mode, checking database every 30 seconds for new clusters
- Process SIGHUP to reload from YAML/database

**Rationale:**
- Enables bootstrap scenarios where clusters are added after deployment
- Prevents crash loops when DB is temporarily empty

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| Kubeconfig in DB is sensitive | Use database-level encryption; consider future Vault integration |
| SIGHUP during active triage | Complete in-flight work before applying changes |
| Config validation failure on reload | Reject invalid config, keep running with previous valid config |
| Orphaned Jobs if execution cluster removed | Complete existing Jobs before removing executor |

## Migration Plan

1. **Create migration**: Add new tables, don't modify existing schema
2. **Update config structs**: New Go types for new structure
3. **Update config files**: All `configs/*.yaml` files use new format
4. **Add signal handling**: SIGHUP handler in cmd/runner
5. **Update cluster manager**: Reload logic, zero-cluster support
6. **Update documentation**: Config examples, migration guide

### Rollback
Not applicable - no backwards compatibility required per user request.

## Open Questions

None - all questions resolved during proposal discussion.
