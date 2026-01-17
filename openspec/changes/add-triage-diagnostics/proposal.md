# Proposal: Add Triage Diagnostics

## Why

When triage runs fail or behave unexpectedly, operators lack visibility into what happened at the Kubernetes level. The agent container uploads its own logs, but these don't include:

1. **Container startup failures** - If the container fails to start (image pull errors, resource limits, etc.)
2. **Entrypoint preflight output** - Messages before the agent logging begins
3. **K8s-level events** - Pod scheduling, node issues, OOM kills

This proposal adds K8s pod log capture to nightcrier, providing complete diagnostic visibility for every triage run.

**User Value**: Operators can diagnose any triage failure without needing kubectl access to the execution cluster.

**Technical Value**: Complete observability chain from nightcrier → K8s Job → container logs, all accessible via the admin UI.

## Problem Statement

Recent investigation of incident `f106c582-3185-4183-908a-ac376cbe843c` revealed that NATS connectivity failed silently. The failure occurred during container startup (in `nats-publish.sh` connectivity check), but this output was not captured in the uploaded agent logs. The only way to see this output would be to run `kubectl logs` against the execution cluster - which requires cluster access and the pod must still exist.

## Proposed Solution

When a triage Job completes (success or failure), nightcrier should:

1. **Fetch K8s pod logs** from the agent container using the Kubernetes API
2. **Upload pod logs** to object storage as `{incidentID}/logs/pod-logs.txt`
3. **Store the URL** in the `agent_executions` table for easy retrieval

### Scope

- Agent container logs only (not init containers)
- Captured at job completion (not during execution)
- Always captured (not just debug mode) - storage is cheap, diagnostics are valuable

### Integration Points

- **K8s Executor**: After detecting Job completion, fetch pod logs before cleanup
- **Object Storage**: Upload alongside existing artifacts
- **Database**: Add `pod_logs_url` column to `agent_executions`
- **Admin UI**: (Future) Link to pod logs from incident detail view

## Success Criteria

1. Pod logs are captured for every triage run (success or failure)
2. Logs are available in object storage within 30 seconds of job completion
3. URL is stored in database and retrievable via admin UI
4. Logs include container startup output (entrypoint messages, NATS checks)

## Impact Assessment

### User Experience
- **Positive**: Complete diagnostic visibility without cluster access
- **Positive**: Pod logs persist even after TTL cleanup deletes the Job

### System Behavior
- **Minor overhead**: One additional K8s API call per triage run
- **Storage cost**: ~10-100KB per run (negligible)

### Implementation Risk
- **Low**: Uses existing K8s client and object storage infrastructure
- **Low**: Additive change, doesn't modify existing flows

## Alternatives Considered

### Alternative 1: Stream logs during execution
- **Pros**: Real-time visibility
- **Cons**: Complex (requires log streaming infrastructure)
- **Decision**: Defer - completion-time capture is sufficient for diagnostics

### Alternative 2: Only capture on failure
- **Pros**: Less storage
- **Cons**: Loses valuable context for investigating unexpected successes
- **Decision**: Capture always - storage is cheap

## Dependencies

- Existing K8s client infrastructure
- Existing object storage integration
- Database migration for new column
