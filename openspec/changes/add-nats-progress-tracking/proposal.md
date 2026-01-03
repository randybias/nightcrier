# Change: Add NATS-based Triage Agent Progress Tracking

## Why

Currently, nightcrier has no visibility into triage agent execution state between start and completion. When agents wedge, timeout, or fail at specific stages, operators cannot determine:

1. Where in the triage process the agent stopped
2. Whether the agent is making progress or stuck
3. What specific operations the agent is performing

This creates blind spots in observability, making debugging difficult and preventing proactive intervention when agents stall.

## What Changes

Add an **optional** NATS-based progress tracking system that emits semantic progress events from triage agents. This is implemented in phases:

### Phase 1: Entrypoint Progress Events
- Add NATS configuration options to nightcrier config
- Spawn NATS listener at nightcrier startup (before any agents run)
- Emit `lifecycle.started` and `lifecycle.completed` events from `entrypoint.sh`
- Store progress state in `agent_executions` table
- Update nc-agent-runner container with NATS client (nats-cli)

### Phase 2: Claude Hooks Integration
- Configure Claude Code hooks for PreToolUse/PostToolUse (Bash tool only)
- Emit `phase.<name>.started` and `phase.<name>.completed` events from hooks
- Provide hook scripts that publish to NATS

### Phase 3: Other Agent Support (Future)
- Extend to Codex, Gemini, Goose as they gain hook support
- Document hook integration patterns for each agent

## Message Model

Use hierarchical NATS subjects for flexible subscription:

**Subjects:**
- `triage.<incident-id>.run.started` - Triage run begins
- `triage.<incident-id>.run.completed` - Triage run ends
- `triage.<incident-id>.executing` - Agent is executing something (with activity info)
- `triage.<incident-id>.error` - Error occurred

**Payload (JSON):**
```json
{
  "incident_id": "inc-abc123",
  "cluster": "prod-us-east-1",
  "timestamp": "2025-01-03T12:00:00Z",
  "event_type": "run.started",
  "agent_cli": "claude",
  "model": "sonnet",
  "activity": "kubectl get pods -n kube-system",
  "exit_code": 0,
  "error_message": "timeout waiting for response"
}
```

**Subscription Patterns:**
- `triage.>` - All events from all runs
- `triage.<incident-id>.*` - All events for specific incident
- `triage.*.error` - All errors across runs
- `triage.*.run.*` - All run start/complete events

## Design Principles

1. **Fire-and-forget**: Core NATS only (no JetStream). Events are transient; nightcrier processes them immediately or they're lost.
2. **Graceful degradation**: If NATS is unavailable, agents continue normally; progress tracking is optional.
3. **Non-standard port**: NATS server runs on configurable port (not 4222) for security.
4. **Token authentication**: NATS connections require auth token passed via environment.
5. **Minimal payload**: Small JSON payloads with timestamps and essential metadata only.

## Impact

- **Affected specs**: `configuration`, `agent-container`, NEW: `agent-progress-tracking`
- **Affected code**:
  - `internal/config/config.go` - NATS configuration struct
  - `internal/nats/` - NEW: NATS client and listener
  - `nc-agent-runner/entrypoint.sh` - Progress event emission
  - `nc-agent-runner/Dockerfile` - Add nats-cli
  - `nc-agent-runner/hooks/` - NEW: Claude hook scripts
  - `internal/agent/k8s_executor.go` - Pass NATS env vars to Job
  - `migrations/` - Schema for progress tracking
- **Database changes**: Add 4 columns to `agent_executions`: `current_activity`, `current_activity_started_at`, `last_activity`, `last_activity_finished_at`
- **Container changes**: nc-agent-runner needs nats-cli binary and hook scripts
