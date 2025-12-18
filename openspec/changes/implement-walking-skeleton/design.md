# Walking Skeleton Design

## Context

This design intentionally favors simplicity over completeness. The goal is a working end-to-end flow, not production readiness.

## Architecture

```
┌─────────────────────┐     ┌─────────────────────┐     ┌─────────────────────┐
│  kubernetes-mcp-    │     │    Event Runner     │     │   Stub Agent        │
│  server             │────>│    (walking         │────>│   (bash script)     │
│  (SSE endpoint)     │     │     skeleton)       │     │                     │
└─────────────────────┘     └─────────────────────┘     └─────────────────────┘
         │                           │                           │
         │ FaultEvent JSON           │ Create workspace          │ Write artifacts
         │                           │ Write event.json          │
         │                           │ Exec stub script          │
         │                           │ Write result.json         │
         ▼                           ▼                           ▼
```

## Directory Structure

```
cmd/
  runner/
    main.go              # CLI entry point
internal/
  config/
    config.go            # Configuration struct + loading
  events/
    client.go            # SSE client (minimal)
    event.go             # FaultEvent struct
  agent/
    workspace.go         # Create incident directory
    context.go           # Write event.json
    executor.go          # exec.Command wrapper
  reporting/
    result.go            # Write result.json
scripts/
  stub-agent.sh          # Echo script for testing
```

## Decisions

### D1: No Reconnection Logic
- **Decision**: Skip exponential backoff reconnection
- **Rationale**: Manual restart acceptable for skeleton; full logic in Phase 1

### D2: No Event Filtering
- **Decision**: Process ALL events received
- **Rationale**: Simplifies testing; filtering logic in Phase 1

### D3: Stub Script Instead of Real Agent
- **Decision**: Use bash script that logs incident details and exits 0
- **Rationale**: Proves execution flow without Claude API dependency

### D4: Synchronous Processing
- **Decision**: Process events synchronously (block on each)
- **Rationale**: Simplifies debugging; concurrency in Phase 1

### D5: Minimal Config
- **Decision**: Only require SSE_ENDPOINT and WORKSPACE_ROOT
- **Rationale**: Other config deferred to full phases

## Data Flow

1. Runner connects to `SSE_ENDPOINT`
2. On event: parse JSON to FaultEvent
3. Generate incident ID (UUID)
4. Create `WORKSPACE_ROOT/<incident-id>/`
5. Write `event.json` with FaultEvent data
6. Execute `stub-agent.sh` with working dir = workspace
7. Write `result.json` with exit code and timestamp
8. Log completion, loop for next event

## Configuration

```bash
# Required
SSE_ENDPOINT=http://localhost:8080/events
WORKSPACE_ROOT=./incidents

# Optional (with defaults)
LOG_LEVEL=info
```

## Testing Approach

1. Start kubernetes-mcp-server (real or mock)
2. Start runner: `./runner --sse-endpoint $SSE_ENDPOINT`
3. Trigger fault event in cluster
4. Verify:
   - Console shows "Received event: ..."
   - Directory `./incidents/<uuid>/` exists
   - File `event.json` contains event data
   - File `result.json` shows exit_code: 0
