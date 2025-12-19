## Context

The system receives fault events from kubernetes-mcp-server and triggers AI agent investigations. Currently, the concepts of "event" (input signal) and "incident" (our response) are conflated, making the codebase harder to reason about and extend.

## Goals

- Clean separation between Event (input) and Incident (response)
- Single `incident.json` file containing full incident lifecycle
- Agent only receives incident context, not raw event data
- Structure that maps cleanly to future incident database

## Non-Goals

- Many-events-to-one-incident correlation (future work, this enables it)
- Incident database implementation (future work, this enables it)
- Changes to MCP client or event subscription logic

## Decisions

### Decision: Create separate Incident struct

**What**: New `internal/incident/incident.go` with `Incident` struct distinct from `FaultEvent`

**Why**:
- Events are immutable input from MCP
- Incidents are mutable records of our response
- Mixing them violates single responsibility

**Alternatives considered**:
- Keep using FaultEvent with embedded fields → Rejected: perpetuates confusion
- Use generic map[string]any → Rejected: loses type safety

### Decision: Single incident.json file

**What**: Replace `event.json` + `result.json` with single `incident.json`

**Why**:
- Incident is the primary entity
- Result metadata (exitCode, completedAt, status) belongs on incident
- Simpler for agent - one file to read
- Maps directly to database record

**Alternatives considered**:
- Keep separate files → Rejected: artificial separation, harder to track
- Nested structure with event inside → Rejected: agent doesn't need event details

### Decision: Flatten event data into incident

**What**: Copy relevant fields from Event to Incident (cluster, namespace, resource, etc.)

**Why**:
- Agent needs context but not MCP metadata (subscriptionId, etc.)
- Decouples incident from event structure changes
- Simpler incident.json for agent consumption

**Alternatives considered**:
- Reference eventId and require lookup → Rejected: unnecessary complexity
- Embed full event → Rejected: exposes internal details to agent

## Data Model

### Event (internal only)

```go
// Event is a fault notification from MCP - immutable input
type Event struct {
    EventID          string         `json:"eventId"`          // Generated on receipt
    ReceivedAt       time.Time      `json:"receivedAt"`       // When we got it
    SubscriptionID   string         `json:"subscriptionId"`   // MCP subscription
    Cluster          string         `json:"cluster"`
    // ... rest of MCP data
}

func (e *Event) DeduplicationKey() string {
    // cluster + namespace + kind + name + reason
}
```

### Incident (primary entity)

```go
// Incident represents our investigation of a fault
type Incident struct {
    // Identity
    IncidentID string `json:"incidentId"`

    // Lifecycle
    Status      string     `json:"status"`      // pending, investigating, resolved, failed
    CreatedAt   time.Time  `json:"createdAt"`
    StartedAt   *time.Time `json:"startedAt,omitempty"`
    CompletedAt *time.Time `json:"completedAt,omitempty"`

    // Result (populated after agent runs)
    ExitCode      *int   `json:"exitCode,omitempty"`
    FailureReason string `json:"failureReason,omitempty"`

    // Context (flattened from triggering event)
    Cluster   string        `json:"cluster"`
    Namespace string        `json:"namespace"`
    Resource  *ResourceInfo `json:"resource"`
    FaultType string        `json:"faultType"`
    Severity  string        `json:"severity"`
    Context   string        `json:"context"`   // Human-readable description
    Timestamp string        `json:"timestamp"` // When fault occurred in K8s

    // Traceability (internal, not for agent)
    TriggeringEventID string `json:"triggeringEventId,omitempty"`
}
```

### Status Values

| Status | Meaning |
|--------|---------|
| `pending` | Incident created, waiting for agent slot |
| `investigating` | Agent is running |
| `resolved` | Agent completed successfully |
| `failed` | Agent failed (non-zero exit, crash, timeout) |
| `agent_failed` | Agent ran but produced no valid output |

## File Lifecycle

1. **Incident Created** (before agent starts):
```json
{
  "incidentId": "abc-123",
  "status": "investigating",
  "createdAt": "2025-12-19T17:30:00Z",
  "cluster": "kind-events-test",
  "namespace": "default",
  "resource": {"kind": "Pod", "name": "crashloop-test"},
  "faultType": "CrashLoop",
  "severity": "critical",
  "context": "Container crashed with exit code 1...",
  "timestamp": "2025-12-19T17:29:55Z"
}
```

2. **After Agent Completes** (updated in place):
```json
{
  "incidentId": "abc-123",
  "status": "resolved",
  "createdAt": "2025-12-19T17:30:00Z",
  "startedAt": "2025-12-19T17:30:01Z",
  "completedAt": "2025-12-19T17:32:15Z",
  "exitCode": 0,
  "cluster": "kind-events-test",
  ...
}
```

## Migration Path

1. Create new `internal/incident/` package
2. Update `processEvent()` to create Incident from Event
3. Write `incident.json` instead of `event.json`
4. Update incident after agent completes (no separate result.json)
5. Update storage to use new file names
6. Update agent container docs (AGENTS.md, CLAUDE.md)
7. Remove `IncidentID` from `FaultEvent` struct
8. Delete `internal/reporting/result.go`

## Risks / Trade-offs

**Risk**: Breaking change for any external tooling reading event.json
**Mitigation**: This is internal tooling, no external consumers

**Risk**: Agent prompts reference event.json
**Mitigation**: Update AGENTS.md, CLAUDE.md, triage-system-prompt.md

**Trade-off**: Slightly more code to create Incident from Event
**Benefit**: Much cleaner separation of concerns, easier future extension

## Open Questions

None - design is straightforward refactoring.
