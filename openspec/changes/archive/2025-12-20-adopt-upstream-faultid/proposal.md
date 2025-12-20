# Change: Adopt upstream FaultID from kubernetes-mcp-server

## Why

kubernetes-mcp-server is adding a stable, deterministic `faultId` field to fault event notifications. This enables proper deduplication and incident lifecycle management in nightcrier.

**Current Problem:**
- nightcrier generates a random UUID as FaultID on receipt (line 122 in client.go)
- Each re-emission of the same fault condition gets a different FaultID
- nightcrier has no way to know if two fault events represent the same underlying problem
- The `DeduplicationKey()` method is a local workaround that doesn't account for re-emissions

**Upstream Solution:**
- kubernetes-mcp-server will provide a stable `faultId` in each fault event
- Same fault condition = same FaultID, always (deterministic hash)
- Re-emissions after TTL expiry retain the same FaultID
- nightcrier can use this for proper deduplication and incident lifecycle decisions

## Semantic Model

This change clarifies the conceptual boundary between systems:

| Concept     | Owner                  | Purpose                                         |
|-------------|------------------------|-------------------------------------------------|
| Fault Event | kubernetes-mcp-server  | The notification/message emitted to subscribers |
| Fault ID    | kubernetes-mcp-server  | Stable identifier for the fault condition       |
| Incident    | nightcrier             | Tracks triage/response for a fault              |
| Incident ID | nightcrier             | nightcrier's internal tracking identifier       |

**Key insight:** Fault ID identifies the *fault condition*, not the emission. Multiple re-emissions of the same fault condition have the same Fault ID.

## What Changes

### 1. Accept FaultID from upstream (don't generate locally)

**Before:**
```go
// Generate FaultID and set ReceivedAt on receipt
faultEvent.FaultID = uuid.New().String()  // Random, not stable
```

**After:**
```go
// FaultID comes from kubernetes-mcp-server (stable, deterministic)
// Only set ReceivedAt locally
faultEvent.ReceivedAt = time.Now()
```

### 2. Add UID field to ResourceInfo

kubernetes-mcp-server now includes `uid` in the resource reference (used in FaultID hash):

```go
type ResourceInfo struct {
    APIVersion string `json:"apiVersion"`
    Kind       string `json:"kind"`
    Name       string `json:"name"`
    Namespace  string `json:"namespace,omitempty"`
    UID        string `json:"uid,omitempty"`           // NEW: Kubernetes resource UID
}
```

### 3. Remove local DeduplicationKey() method

With stable FaultID from upstream, the local `DeduplicationKey()` method becomes redundant:
- Current: `"{cluster}/{namespace}/{kind}/{name}/{reason}"` (generated locally)
- New: Use `FaultID` directly (provided by upstream, more reliable)

### 4. Update deduplication logic to use FaultID

The fault deduplication should use the upstream FaultID instead of computing a local key.

### 5. Update FaultID field comment

```go
// Before
FaultID string `json:"faultId,omitempty"` // UUID for tracing the raw fault notification

// After
FaultID string `json:"faultId"` // Stable identifier from kubernetes-mcp-server for the fault condition
```

Note: Remove `omitempty` since FaultID should always be present from upstream.

## FaultID Characteristics (from upstream)

- **Format:** 16 hex characters (e.g., `"f7a3b2c1d4e5f6a7"`)
- **Generation:** `hex(sha256(cluster + ":" + faultType + ":" + resourceUID + ":" + containerName)[:8])`
- **Stable:** Same fault condition = same Fault ID, always
- **Deterministic:** Computed from fault characteristics, not random
- **No timestamp:** Recurring faults have the same ID (intentional)
- **Multi-cluster safe:** Includes cluster name to avoid cross-cluster collisions

## Impact

**Benefits:**
- Proper deduplication across re-emissions
- nightcrier controls incident lifecycle policy (reopen vs new incident)
- Cleaner separation of concerns between systems
- Simpler code (remove local UUID generation and DeduplicationKey)

**Breaking Changes:**
- FaultID format changes from UUID (`550e8400-e29b-41d4-a716-446655440000`) to hex hash (`f7a3b2c1d4e5f6a7`)
- FaultID is now required (no `omitempty`)
- DeduplicationKey() method removed

**Coordination Required:**
- kubernetes-mcp-server must be updated first to emit `faultId`
- nightcrier changes should be applied after upstream is deployed

## Affected specs

- `walking-skeleton` - Event subscription and parsing requirements

## Affected code

- `internal/events/event.go` - FaultEvent struct and ResourceInfo struct
- `internal/events/client.go` - Remove local FaultID generation
- Any code using `DeduplicationKey()` method
