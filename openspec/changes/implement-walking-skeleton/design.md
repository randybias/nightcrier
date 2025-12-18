# Walking Skeleton Design

## Status: IMPLEMENTED (2025-12-18)

## Context

This design intentionally favors simplicity over completeness. The goal is a working end-to-end flow, not production readiness.

## Architecture (As Implemented)

```
┌─────────────────────┐     ┌─────────────────────┐     ┌─────────────────────┐
│  kubernetes-mcp-    │     │    Event Runner     │     │   Stub Agent        │
│  server             │────>│    (walking         │────>│   (bash script)     │
│  (/mcp endpoint)    │     │     skeleton)       │     │                     │
└─────────────────────┘     └─────────────────────┘     └─────────────────────┘
         │                           │                           │
         │ MCP Protocol              │ Create workspace          │ Write artifacts
         │ events_subscribe          │ Write event.json          │
         │ logging/message notifs    │ Exec stub script          │
         │                           │ Write result.json         │
         ▼                           ▼                           ▼
```

### Key Change from Original Plan

**Original**: Raw SSE connection to `/sse` endpoint
**Implemented**: MCP client with StreamableHTTP transport to `/mcp` endpoint

This change was necessary because:
1. kubernetes-mcp-server requires proper MCP protocol for session tracking
2. Event subscriptions need session ID for notification delivery
3. Faults are delivered via `logging/message` notifications, not raw SSE events

## Directory Structure (As Implemented)

```
cmd/
  runner/
    main.go              # CLI entry point with Cobra
internal/
  config/
    config.go            # Configuration struct + loading (MCP_ENDPOINT)
  events/
    client.go            # MCP client with StreamableHTTP transport
    event.go             # FaultEvent struct (matches kubernetes-mcp-server format)
  agent/
    workspace.go         # Create incident directory
    context.go           # Write event.json
    executor.go          # exec.Command wrapper (with absolute path handling)
  reporting/
    result.go            # Write result.json
scripts/
  stub-agent.sh          # Test script that reads event.json and writes investigation.md
```

## Decisions

### D1: No Reconnection Logic
- **Decision**: Skip exponential backoff reconnection
- **Rationale**: Manual restart acceptable for skeleton; full logic in Phase 4

### D2: Faults-Only Filtering
- **Decision**: Subscribe with `mode: "faults"` to only receive fault events
- **Rationale**: Server-side filtering is more efficient; matches intended use case
- **Implementation**: `session.CallTool("events_subscribe", {mode: "faults"})`

### D3: Stub Script Instead of Real Agent
- **Decision**: Use bash script that logs incident details and exits 0
- **Rationale**: Proves execution flow without Claude API dependency
- **Implementation**: Script reads `event.json`, writes `output/investigation.md`

### D4: Synchronous Processing
- **Decision**: Process events synchronously (block on each)
- **Rationale**: Simplifies debugging; concurrency in Phase 4

### D5: MCP Protocol Configuration
- **Decision**: Use `MCP_ENDPOINT` instead of `SSE_ENDPOINT`
- **Rationale**: Reflects actual protocol being used (MCP over StreamableHTTP)

### D6: Session Lifecycle Management
- **Decision**: Call `session.Wait()` in goroutine to keep connection alive
- **Rationale**: Required for receiving server-initiated notifications
- **Implementation**: Goroutine calls `session.Wait()`, triggers cleanup on close

## Data Flow (As Implemented)

1. Runner connects to `MCP_ENDPOINT/mcp` via StreamableHTTP
2. MCP initialization and session established (session ID assigned)
3. Set logging level to "info" for notifications
4. Subscribe to faults: `events_subscribe(mode="faults")`
5. Wait for `logging/message` notifications with `logger="kubernetes/faults"`
6. On notification: parse FaultEvent from `params.Data`
7. Generate incident ID (UUID)
8. Create `WORKSPACE_ROOT/<incident-id>/`
9. Write `event.json` with FaultEvent data
10. Execute `stub-agent.sh` with working dir = workspace
11. Write `result.json` with exit code and timestamp
12. Log completion, loop for next event

## Configuration (As Implemented)

```bash
# Required
MCP_ENDPOINT=http://localhost:8383

# Optional (with defaults)
WORKSPACE_ROOT=./incidents
LOG_LEVEL=info
```

CLI flags override environment variables:
```bash
./runner --mcp-endpoint http://localhost:8383 --workspace-root ./incidents --log-level debug
```

## Testing Results

### Test Environment
- kubernetes-mcp-server: localhost:8383
- Cluster: kind-events-test
- Test pod: `kubectl run failing-test --image=busybox --restart=Never -- /nonexistent`

### Verified Flow
1. Runner connects and subscribes
2. Failing pod created (StartError status)
3. kubernetes-mcp-server detects Warning event on Pod
4. Fault notification sent via MCP `logging/message`
5. Runner receives and parses FaultEvent
6. Workspace created: `incidents/<uuid>/`
7. `event.json` written with full event data including container logs
8. Stub agent executed successfully
9. `result.json` shows `status: "success"`, `exit_code: 0`

### Sample Output
```
time=2025-12-18T07:54:58.737+01:00 level=INFO msg="received fault event" cluster=kind-events-test namespace=default resource=Pod/final-test-1766040896 reason=Failed
time=2025-12-18T07:54:58.737+01:00 level=INFO msg="processing fault event" incident_id=5cc31391-ca70-4550-87a2-b97e50b37031
time=2025-12-18T07:54:58.737+01:00 level=INFO msg="created workspace" path=incidents/5cc31391-ca70-4550-87a2-b97e50b37031
time=2025-12-18T07:54:58.773+01:00 level=INFO msg="event processed" status=success exit_code=0 duration=35.705583ms
```

## FaultEvent Structure

The FaultEvent matches kubernetes-mcp-server's `pkg/events/faults.go`:

```go
type FaultEvent struct {
    SubscriptionID string         `json:"subscriptionId"`
    Cluster        string         `json:"cluster"`
    Event          EventData      `json:"event"`
    Logs           []ContainerLog `json:"logs,omitempty"`
}

type EventData struct {
    Namespace      string            `json:"namespace"`
    Timestamp      string            `json:"timestamp"`
    Type           string            `json:"type"`      // "Warning"
    Reason         string            `json:"reason"`    // "Failed", "BackOff", etc.
    Message        string            `json:"message"`
    InvolvedObject *InvolvedObject   `json:"involvedObject"`
    Labels         map[string]string `json:"labels,omitempty"`
    Count          int32             `json:"count"`
}
```

## Known Limitations

1. **No reconnection**: If MCP connection drops, runner exits
2. **No concurrent processing**: Events processed sequentially
3. **No workspace cleanup**: Old workspaces accumulate
4. **Stub agent only**: Real AI agent integration in Phase 2
