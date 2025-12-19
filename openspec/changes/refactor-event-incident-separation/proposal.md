# Change: Refactor Event/Incident Separation

## Why

The current codebase conflates two distinct concepts:

1. **Event**: A signal from Kubernetes via MCP (input, immutable, from source)
2. **Incident**: Our response to an event requiring investigation (output, our lifecycle)

This conflation causes:
- `FaultEvent` struct polluted with `IncidentID` (response metadata on input data)
- Agent receives `event.json` but should only care about the incident it's investigating
- Separate `result.json` file when this metadata belongs on the incident
- Difficult to later implement: many-events-to-one-incident correlation, incident database tracking

## What Changes

### 1. Clean Domain Model Separation

**Event** (internal, for intake processing):
- Immutable data from MCP
- Has `eventId` (generated on receipt for tracing)
- Has `deduplicationKey` (computed for filtering)
- NOT exposed to agent

**Incident** (the primary entity):
- Created when we decide to investigate
- Has `incidentId` (our UUID)
- References triggering event data (flattened, not nested)
- Has lifecycle fields: `status`, `createdAt`, `startedAt`, `completedAt`
- Has result fields: `exitCode`, `failureReason`
- Single `incident.json` file - no separate `result.json`

### 2. File Structure Change

**Before:**
```
incidents/{uuid}/
  event.json      # Raw event + incidentId (polluted)
  result.json     # Separate execution results
  output/
    investigation.md
```

**After:**
```
incidents/{uuid}/
  incident.json   # Complete incident record (what agent sees + results)
  output/
    investigation.md
```

### 3. Agent Interface Simplification

Agent receives `incident.json` containing:
```json
{
  "incidentId": "abc-123-...",
  "status": "investigating",
  "createdAt": "2025-12-19T17:30:00Z",
  "cluster": "kind-events-test",
  "namespace": "default",
  "resource": {
    "kind": "Pod",
    "name": "crashloop-test"
  },
  "faultType": "CrashLoop",
  "severity": "critical",
  "context": "Container crashed with exit code 1...",
  "timestamp": "2025-12-19T17:29:55Z"
}
```

After agent completes, `incident.json` is updated:
```json
{
  "incidentId": "abc-123-...",
  "status": "resolved",
  "createdAt": "2025-12-19T17:30:00Z",
  "startedAt": "2025-12-19T17:30:01Z",
  "completedAt": "2025-12-19T17:32:15Z",
  "exitCode": 0,
  "cluster": "kind-events-test",
  ...
}
```

### 4. Eliminated Files

- `result.json` - absorbed into `incident.json`
- `event.json` - replaced by `incident.json` (agent doesn't need raw event)

## Impact

- Affected specs: `walking-skeleton`
- Affected code:
  - `internal/events/event.go` - Remove `IncidentID`, keep as pure input
  - NEW `internal/incident/incident.go` - New `Incident` struct
  - `internal/agent/context.go` - Write `incident.json` instead of `event.json`
  - `internal/reporting/result.go` - Remove (absorbed into incident)
  - `cmd/runner/main.go` - Create Incident from Event, update after agent runs
  - `internal/storage/` - Update artifact names
  - `agent-container/` - Update AGENTS.md/CLAUDE.md to reference incident.json

## Benefits

1. **Clean separation**: Events are input, Incidents are our response
2. **Single source of truth**: One `incident.json` with full lifecycle
3. **Agent simplicity**: Agent only knows about incidents, not events
4. **Database-ready**: `incident.json` structure maps directly to incident DB schema
5. **Future-proof**: Easy to add many-events-to-one-incident correlation
