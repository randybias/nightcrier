# Design: Adopt Upstream FaultID

## Context

kubernetes-mcp-server and nightcrier need to agree on fault identity. Currently, each system has its own notion of "what fault is this":

- **kubernetes-mcp-server**: Uses internal deduplication based on fault condition
- **nightcrier**: Generates random UUID, uses `DeduplicationKey()` for local dedup

This creates a mismatch where nightcrier cannot properly track fault recurrence across re-emissions.

## Goals

1. Single source of truth for fault identity (kubernetes-mcp-server)
2. nightcrier controls incident lifecycle policy
3. Clean separation of concerns
4. Support for fault recurrence detection

## Non-Goals

1. Implementing incident lifecycle policy (future work)
2. Changing kubernetes-mcp-server behavior (done upstream)
3. Multi-cluster coordination (out of scope)

## Decisions

### Decision 1: Accept FaultID from upstream, don't generate locally

**What:** Remove local UUID generation for FaultID. The field will be populated by JSON unmarshaling from kubernetes-mcp-server's payload.

**Why:**
- kubernetes-mcp-server has the complete context to generate a stable ID
- Local generation loses stability across re-emissions
- Simpler code, single source of truth

**Alternative considered:** Keep generating local FaultID and add a separate `UpstreamFaultID` field.
- Rejected: Adds complexity, confusing to have two IDs

### Decision 2: Remove DeduplicationKey() method

**What:** Delete the `DeduplicationKey()` method from FaultEvent.

**Why:**
- The method computes `"{cluster}/{namespace}/{kind}/{name}/{reason}"`
- This is essentially what kubernetes-mcp-server now provides as FaultID (but hashed)
- Having both is redundant and confusing
- FaultID from upstream is more reliable (includes container name, uses resource UID)

**Alternative considered:** Keep DeduplicationKey() for backward compatibility.
- Rejected: No backward compatibility needed (greenfield project)

### Decision 3: Add UID to ResourceInfo

**What:** Add `UID string` field to ResourceInfo struct.

**Why:**
- kubernetes-mcp-server now includes resource UID in the payload
- UID is used in the FaultID hash for uniqueness
- Useful for nightcrier to have for incident context

**Alternative considered:** Ignore the UID field (don't add to struct).
- Rejected: Loses useful information, struct should match upstream payload

### Decision 4: Make FaultID required (no omitempty)

**What:** Remove `omitempty` from FaultID JSON tag.

**Why:**
- FaultID should always be present from upstream
- If missing, it indicates a protocol error that should be visible
- Helps catch integration issues early

## Data Flow

```
kubernetes-mcp-server                          nightcrier
┌─────────────────────────────┐               ┌─────────────────────────────┐
│                             │               │                             │
│  Detect fault condition     │               │                             │
│           │                 │               │                             │
│           ▼                 │               │                             │
│  Compute FaultID:           │               │                             │
│  sha256(cluster:faultType:  │               │                             │
│         resourceUID:        │               │                             │
│         containerName)[:8]  │               │                             │
│           │                 │               │                             │
│           ▼                 │               │                             │
│  Emit Fault Event with      │    MCP        │  Receive Fault Event        │
│  faultId field ─────────────┼──────────────▶│           │                 │
│                             │               │           ▼                 │
│                             │               │  JSON unmarshal populates   │
│                             │               │  FaultID field directly     │
│                             │               │           │                 │
│                             │               │           ▼                 │
│                             │               │  Use FaultID for:           │
│                             │               │  - Deduplication            │
│                             │               │  - Incident correlation     │
│                             │               │  - Recurrence detection     │
│                             │               │                             │
└─────────────────────────────┘               └─────────────────────────────┘
```

## Updated Payload Structure

**Before (current):**
```json
{
  "subscriptionId": "sub-abc12345",
  "cluster": "prod-us-east",
  "faultType": "CrashLoop",
  "severity": "critical",
  "resource": {
    "apiVersion": "v1",
    "kind": "Pod",
    "name": "api-server-7f8b9c",
    "namespace": "production"
  },
  "context": "Error: connection refused...",
  "timestamp": "2025-12-20T15:30:00Z"
}
```

**After (with upstream faultId):**
```json
{
  "subscriptionId": "sub-abc12345",
  "cluster": "prod-us-east",
  "faultId": "f7a3b2c1d4e5f6a7",
  "faultType": "CrashLoop",
  "severity": "critical",
  "resource": {
    "apiVersion": "v1",
    "kind": "Pod",
    "name": "api-server-7f8b9c",
    "namespace": "production",
    "uid": "abc123-def456-ghi789"
  },
  "context": "Error: connection refused...",
  "timestamp": "2025-12-20T15:30:00Z"
}
```

## Risks / Trade-offs

### Risk: Missing FaultID from old kubernetes-mcp-server versions

**Mitigation:**
- This is a coordinated change - kubernetes-mcp-server is updated first
- nightcrier should validate FaultID is present and log warning if missing
- Could generate local fallback FaultID if needed (but prefer to fail fast)

### Risk: FaultID format change breaks tooling

**Mitigation:**
- FaultID was only recently added (previous change)
- No external tooling depends on it yet
- Format change is intentional (UUID → hex hash)

### Trade-off: Losing ReceivedAt-based deduplication

**Accepted:**
- ReceivedAt is still set locally for internal timing
- Deduplication is now FaultID-based, which is more correct
- ReceivedAt can still be used for TTL-based cleanup of tracking state

## Migration Plan

1. **kubernetes-mcp-server updated first** (prerequisite)
2. **nightcrier changes applied:**
   - Update structs to match new payload
   - Remove local FaultID generation
   - Remove DeduplicationKey() method
   - Update any code using DeduplicationKey()
3. **Validation:**
   - Test with updated kubernetes-mcp-server
   - Verify FaultID populated from upstream
   - Verify same fault condition produces same FaultID

## Open Questions

None - the design is straightforward given the upstream changes.
