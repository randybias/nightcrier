# Design: NATS-based Triage Agent Progress Tracking

## Context

Nightcrier spawns AI agents as Kubernetes Jobs to triage incidents. Currently, visibility into agent state is limited to:
- Job start (when K8s Job is created)
- Job completion (when Job terminates with exit code)

There is no intermediate visibility. If an agent wedges during a `kubectl logs` command or times out waiting for Claude's response, operators only discover this when the Job times out after 10 minutes.

**Stakeholders:**
- Platform operators monitoring triage agent health
- SREs debugging failed investigations
- Future UI dashboards showing real-time agent progress

## Goals / Non-Goals

**Goals:**
- Track triage agent progress with semantic events (not raw logs)
- Detect wedged agents before they timeout
- Provide flexible subscription patterns for consumers
- Maintain agent functionality when NATS is unavailable
- Support phased rollout starting with entrypoint.sh events

**Non-Goals:**
- Durability/replay (no JetStream for now)
- Complete command-level logging (that's what agent.log is for)
- Real-time UI (future work, builds on this foundation)
- Support for non-Claude agents' hooks in Phase 1

## Decisions

### Decision 1: Core NATS over JetStream

**What:** Use Core NATS pub/sub without JetStream streams.

**Why:**
- Fire-and-forget is acceptable for progress tracking
- Simpler operational model (no stream management)
- Lower latency for real-time monitoring
- Progress events are ephemeral; persistence is in the database
- JetStream can be added later if replay is needed

**Alternatives considered:**
- JetStream: Adds durability but complexity; not needed for initial use case
- Redis pub/sub: Another dependency to manage; NATS is lighter weight
- gRPC streaming: More complex client implementation in bash

### Decision 2: Hierarchical Subject Naming

**What:** Use dot-separated hierarchical subjects: `triage.<run-id>.<category>.<event>`

**Why:**
- Enables flexible wildcard subscriptions (`triage.*.error`, `triage.inc-123.*`)
- Self-documenting subject structure
- Matches NATS best practices for subject naming
- Allows consumers to subscribe at appropriate granularity

**Subject taxonomy:**
```
triage.
├── <incident-id>.
│   ├── run.
│   │   ├── started
│   │   └── completed
│   ├── executing      (activity info in payload)
│   └── error
```

### Decision 3: nats-cli in Container (Not Go Library)

**What:** Use `nats-cli` binary in the agent container rather than a Go library.

**Why:**
- entrypoint.sh is bash; no Go code available
- Claude hooks execute shell commands
- nats-cli is small (~15MB) and well-maintained
- Simple one-liner publishes: `nats pub "triage.$ID.lifecycle.started" "$JSON"`
- No compilation required for hook scripts

**Alternatives considered:**
- Go sidecar container: Adds complexity, resource overhead
- Custom Go binary for publishing: More to maintain, overkill for simple pub
- curl to NATS HTTP gateway: Requires additional NATS configuration

### Decision 4: Authentication via Token

**What:** Use NATS token authentication passed via environment variable.

**Why:**
- Simple to implement and rotate
- Works with K8s Secrets
- Sufficient for internal cluster communication
- Can upgrade to nkeys/JWT later if needed

**Configuration:**
```yaml
nats:
  enabled: true
  server: "nats://nats.nightcrier.svc:14222"
  token: "${NATS_AUTH_TOKEN}"  # From K8s Secret
```

### Decision 5: Listener Spawns at Startup

**What:** Start NATS listener goroutine at nightcrier startup, before any agents run.

**Why:**
- Ensures listener is ready before first event
- Single connection for all subscriptions
- Can process events even during agent spawn
- Graceful shutdown with context cancellation

**Implementation:**
```go
// In main.go startup sequence
if cfg.NATS.Enabled {
    natsClient, err := nats.Connect(cfg.NATS.Server, cfg.NATS.Token)
    listener := progress.NewListener(natsClient, stateStore)
    go listener.Start(ctx)  // Background goroutine
}
```

### Decision 6: Claude Hooks for Bash Tool Only

**What:** In Phase 2, only hook into PreToolUse for the `Bash` tool to emit "executing" events.

**Why:**
- Bash commands are the observable actions (kubectl, etc.)
- Read/Write/Grep are internal to Claude, less meaningful for progress
- We only need to know what the agent is currently doing, not track start/end of each command
- Most diagnostic value comes from knowing the last activity when an agent wedges

**Hook configuration (in container):**
```json
{
  "hooks": {
    "PreToolUse": [{
      "matcher": "Bash",
      "hooks": [{
        "type": "command",
        "command": "/home/agent/hooks/nats-executing.sh"
      }]
    }]
  }
}
```

### Decision 7: Phased Implementation

**What:** Implement in two phases with clear validation gates.

**Phase 1: Entrypoint Events**
- NATS config and listener
- entrypoint.sh emits run.started/completed
- Database columns for last activity tracking
- Validates end-to-end NATS flow

**Note:** `run.started` should be emitted AFTER preflight validation completes (see `add-entrypoint-preflight-checks`). If preflight fails, no run.started is emitted - the container exits before the triage run truly "starts".

**Phase 2: Claude Hooks**
- Hook script in container
- Claude settings.json configuration
- "executing" events from Bash tool usage
- Validates hook integration pattern

**Why:**
- Reduces risk; can validate NATS plumbing before adding hooks
- Phase 1 alone provides significant value (detect container crashes)
- Phase 2 can be deferred if hooks prove problematic

## Architecture

### Component Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│                           Kubernetes                                  │
│                                                                       │
│  ┌─────────────────────┐         ┌─────────────────────────────┐    │
│  │    nightcrier       │         │      nc-agent-runner        │    │
│  │                     │         │         (K8s Job)           │    │
│  │  ┌───────────────┐  │         │  ┌───────────────────────┐  │    │
│  │  │ NATS Listener │◄─┼─────────┼──│ entrypoint.sh         │  │    │
│  │  │  (goroutine)  │  │   NATS  │  │  nats pub run.*       │  │    │
│  │  └───────┬───────┘  │         │  └───────────────────────┘  │    │
│  │          │          │         │              │               │    │
│  │          ▼          │         │              ▼               │    │
│  │  ┌───────────────┐  │         │  ┌───────────────────────┐  │    │
│  │  │  StateStore   │  │         │  │   Claude Code         │  │    │
│  │  │  (PostgreSQL) │  │         │  │   PreToolUse hook     │──┼────┤
│  │  └───────────────┘  │         │  │   nats pub executing  │  │    │
│  │                     │         │  └───────────────────────┘  │    │
│  └─────────────────────┘         └─────────────────────────────┘    │
│                                                                       │
│  ┌─────────────────────┐                                             │
│  │    NATS Server      │ (port 14222, token auth)                    │
│  │    (Deployment)     │                                             │
│  └─────────────────────┘                                             │
└─────────────────────────────────────────────────────────────────────┘
```

### Database Schema

### Lifecycle Tracking (Migration 000003, 000004)

The schema distinguishes between **Job lifecycle** (Go orchestration) and **Run lifecycle** (container execution):

```sql
-- agent_executions table
-- Job lifecycle (Go orchestration - when K8s Job is created/cleaned up)
job_started_at TIMESTAMP NOT NULL     -- When Go creates K8s Job
job_completed_at TIMESTAMP            -- When Go finishes cleanup/uploads

-- Run lifecycle (container - when entrypoint.sh executes)
run_started_at TIMESTAMP              -- When container publishes run.started (after preflight)
run_completed_at TIMESTAMP            -- When container publishes run.completed
run_exit_code INTEGER                 -- Exit code from container execution

-- incidents table also uses job_started_at, job_completed_at
```

This separation allows tracking:
1. Time spent in K8s Job creation/scheduling
2. Time spent in container execution
3. Time spent in post-execution cleanup/uploads

**Timing Requirement:** The agent_execution record MUST be created at Job creation time
(in k8s_executor.go) so that when the NATS listener receives `run.started`, the record
exists and can be updated. The processor's later `RecordAgentExecution` call uses UPSERT
to update the existing record with completion data.

**SQLite Compatibility:** SQLite does not support `UPDATE ... ORDER BY ... LIMIT` syntax.
All update methods (UpdateRunStarted, UpdateRunCompleted, UpdateExecutionActivity) use
subqueries to select the target execution_id first, then update by that ID.

### Activity Tracking (Migration 000002)

```sql
-- Add activity tracking to agent_executions
-- Tracks current activity (what it's doing now) and last activity (what it just finished)
ALTER TABLE agent_executions ADD COLUMN current_activity TEXT;
ALTER TABLE agent_executions ADD COLUMN current_activity_started_at TIMESTAMP;
ALTER TABLE agent_executions ADD COLUMN last_activity TEXT;
ALTER TABLE agent_executions ADD COLUMN last_activity_finished_at TIMESTAMP;
```

No separate table needed - we only care about current state and the previous activity.

When a new "executing" event arrives, the listener:
1. Copies `current_activity` → `last_activity`
2. Copies `current_activity_started_at` → `last_activity_finished_at`
3. Sets `current_activity` = new command
4. Sets `current_activity_started_at` = event timestamp

This gives operators both "what is it stuck on?" and "what did it just finish?"

### Event Payload Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["incident_id", "timestamp", "event_type"],
  "properties": {
    "incident_id": {
      "type": "string",
      "description": "Incident identifier"
    },
    "cluster": {
      "type": "string",
      "description": "Target cluster name"
    },
    "timestamp": {
      "type": "string",
      "format": "date-time",
      "description": "ISO8601 timestamp when event occurred"
    },
    "event_type": {
      "type": "string",
      "enum": [
        "run.started",
        "run.completed",
        "executing",
        "error"
      ]
    },
    "agent_cli": {
      "type": "string",
      "enum": ["claude", "codex", "gemini", "goose"]
    },
    "model": {
      "type": "string"
    },
    "activity": {
      "type": "string",
      "description": "What the agent is currently executing (first 100 chars of command)",
      "maxLength": 100
    },
    "exit_code": {
      "type": "integer",
      "description": "Exit code (for run.completed)"
    },
    "error_message": {
      "type": "string",
      "description": "Error details (for error events)"
    }
  }
}
```

## Risks / Trade-offs

### Risk: NATS unavailability during agent run

**Impact:** Progress events lost, no tracking for that execution.
**Mitigation:**
- nats-cli publish is non-blocking with timeout
- Agent continues regardless of NATS publish success
- Log warning on publish failure for debugging

### Risk: High event volume overwhelming listener

**Impact:** Listener falls behind, memory growth.
**Mitigation:**
- Use buffered subscription (1000 messages)
- Drop events if buffer full (log warning)
- Limit to Bash tool hooks only (not Read/Write)

### Risk: Claude hooks slow down agent execution

**Impact:** Longer triage times.
**Mitigation:**
- Hooks execute in parallel (non-blocking)
- Set hook timeout to 5 seconds
- If hook fails, continue without blocking

### Risk: nats-cli binary size increases container

**Impact:** Larger image, longer pull times.
**Mitigation:**
- nats-cli is ~15MB; acceptable
- Use multi-stage build to minimize
- Consider static binary

## Migration Plan

### Steps

1. **Deploy NATS server** (can use existing if available)
   - Configure non-standard port (14222)
   - Enable token authentication
   - Create K8s Secret with auth token

2. **Deploy nightcrier with NATS listener** (Phase 1)
   - Add NATS config to config.yaml
   - Run database migration to add columns to agent_executions
   - Listener starts but no events yet

3. **Deploy updated nc-agent-runner** (Phase 1)
   - New image with nats-cli
   - entrypoint.sh emits run.started/completed events
   - Test run events flow

4. **Validate Phase 1**
   - Trigger test incident
   - Verify events received and last_activity updated
   - Check no impact on agent execution time

5. **Add Claude hooks** (Phase 2)
   - Add hook script to container
   - Update Claude settings.json
   - Deploy updated container image

6. **Validate Phase 2**
   - Trigger test incident with Claude agent
   - Verify "executing" events from Bash tool usage
   - Monitor for performance impact

### Rollback

- **Disable via config:** Set `nats.enabled: false` to disable listener
- **Container rollback:** Previous image without nats-cli works fine
- **No data migration needed:** Columns are nullable and additive

## Open Questions

1. **NATS deployment model:** Should nightcrier documentation include NATS deployment manifests, or assume operators have existing NATS infrastructure?

2. **Other agent hooks:** Research needed to determine if Gemini or Goose support hooks. Codex does NOT currently support hooks. For now, only Claude gets "executing" events; other agents only get run.started/completed from entrypoint.sh.
