# Proposal: Resilient Credential Startup

## Problem Statement

Currently, nightcrier fails fast at startup if credentials or dependent resources are unavailable:

1. **API keys**: If no API keys are configured, bootstrap fails immediately
2. **Kubeconfig files**: If triage kubeconfig files are missing, bootstrap fails
3. **K8s API unavailable**: If the execution cluster is unreachable, startup fails

This causes problems in dynamic environments where:
- Secrets may be injected asynchronously (Vault, external-secrets, etc.)
- Kubeconfig files may be mounted after container start
- K8s API may be temporarily unavailable during cluster operations
- Rolling deployments may have brief windows of unavailability

## Proposed Solution

Instead of failing fast, nightcrier should:
1. **Warn** about missing credentials at startup
2. **Retry** with exponential backoff until credentials become available
3. **Start in degraded mode** - run core functionality but disable features that require missing credentials
4. **Recover automatically** when credentials become available

## Behavior Changes

### Current Behavior
```
startup -> check credentials -> FAIL if missing -> exit(1)
```

### Proposed Behavior
```
startup -> attempt bootstrap -> WARN on failures -> START IMMEDIATELY (degraded if needed)
                                                 -> retry failures in background
                                                 -> recover automatically
                                                 -> update admin UI status
```

Key principle: **Never block startup.** Start immediately, retry failures in background.

## Degraded Mode Capabilities

When running in degraded mode due to missing credentials:

| Component | Missing Credential | Degraded Behavior |
|-----------|-------------------|-------------------|
| Bootstrap | API keys | Skip secret creation, warn, retry in background |
| Bootstrap | Kubeconfig file | Skip that cluster's secret, warn, retry in background |
| Bootstrap | K8s API unreachable | Retry with backoff, start without bootstrap after timeout |
| Dispatcher | No API keys | Accept events but cannot dispatch agents |
| MCP Client | N/A | Connect normally (no credentials needed) |

## Configuration

```yaml
startup:
  # Backoff settings for background credential retry
  credential_retry_initial: 5s
  credential_retry_max: 300s  # 5 minutes max between retries
  credential_retry_multiplier: 2.0
```

Note: No `credential_wait_timeout` or `allow_degraded_start` - we always start immediately and retry in background.

## Parallel Bootstrap for Many Clusters

With 100+ monitored clusters, sequential bootstrap is unacceptable. The bootstrap process must:

1. **Bootstrap global resources first** (namespace, RBAC, API keys secret)
2. **Bootstrap per-cluster resources in parallel** (triage kubeconfig secrets)
3. **Track per-cluster status independently** - one failing cluster doesn't block others
4. **Report aggregate status** - "95/100 clusters ready, 5 degraded"

```go
type ClusterBootstrapStatus struct {
    Name      string
    Ready     bool
    Error     error
    LastRetry time.Time
    Retries   int
}

type BootstrapStatus struct {
    GlobalReady     bool
    ClusterStatuses map[string]*ClusterBootstrapStatus
}
```

## Admin UI Integration

The admin UI must display bootstrap/credential status:

### New "System Status" Section
- **Global Status**: Ready / Degraded / Initializing
- **API Keys**: Available / Missing (with which providers)
- **Clusters Table** (in existing Clusters pane):
  - Add "Bootstrap Status" column: Ready / Degraded / Retrying
  - Add "Last Error" column for failed clusters
  - Visual indicator (color/icon) for degraded clusters

### Example Admin UI Display
```
System Status: DEGRADED
├── API Keys: Missing (anthropic, openai)
├── Namespace: Ready
├── RBAC: Ready
└── Clusters: 95/100 ready

Monitored Clusters:
| Name          | Environment | Bootstrap | Last Error              |
|---------------|-------------|-----------|-------------------------|
| prod-east     | production  | Ready     | -                       |
| prod-west     | production  | Ready     | -                       |
| staging-eu    | staging     | Degraded  | kubeconfig not found    |
| dev-local     | development | Retrying  | connection refused      |
```

## Scope

1. Never block startup - start immediately, retry in background
2. Parallel bootstrap for per-cluster resources
3. Per-cluster degraded status tracking
4. Background retry goroutine with exponential backoff (max 300s)
5. Admin UI system status section
6. Admin UI cluster bootstrap status column
7. Health endpoint degraded status
8. Periodic warning logs when degraded

## Out of Scope

- Hot-reloading of credentials after initial availability (future enhancement)
- Credential rotation handling
- Multiple credential sources (Vault integration, etc.)

## Requirements Changed

- **configuration**: Add resilient startup configuration requirements
- **admin-ui**: Add system status and cluster bootstrap status display

## Success Criteria

1. Nightcrier starts even when API keys are not yet available
2. Bootstrap retries automatically when K8s API becomes available
3. Clear warnings indicate degraded state
4. Automatic recovery when credentials become available
5. Health endpoint reflects degraded status
