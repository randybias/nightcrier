# Change: Remove backwards compatibility for deprecated "faults" mode

## Why

The kubernetes-mcp-server is removing the deprecated "faults" subscription mode in favor of the canonical "resource-faults" mode. Nightcrier currently maintains backwards compatibility by supporting both modes with dual-mode event structures and fallback logic. This backwards compatibility creates cognitive load and contaminates the codebase with unused code paths.

Since nightcrier is a greenfield project that has never been in production, there is no need to maintain backwards compatibility. The codebase should be continuously refined and simplified before reaching production.

## What Changes

Remove all backwards compatibility for the deprecated "faults" subscription mode:

1. **Event Structure Cleanup** (`internal/events/event.go`):
   - Rename `EventID` field to `FaultID` (we only handle faults, not general events)
   - Remove `Event *EventData` field from `FaultEvent` (nested "faults" mode structure)
   - Remove `Logs []ContainerLog` field from `FaultEvent` (replaced by `Context` field in resource-faults mode)
   - Remove `EventData`, `InvolvedObject`, and `ContainerLog` type definitions
   - Keep only resource-faults mode fields: `FaultID`, `Resource`, `Context`, `FaultType`, `Severity`, `Timestamp`

2. **Helper Method Simplification** (`internal/events/event.go`):
   - Remove all fallback logic checking `f.Event != nil`
   - Simplify all `Get*()` methods to only access resource-faults fields
   - Remove `IsResourceFaultsMode()` method (only one mode exists)
   - Keep helper methods but simplify to single code path

3. **Incident Creation** (`internal/incident/incident.go`):
   - Remove fallback logic in `extractResourceInfo()` checking `event.Event != nil`
   - Simplify to only extract from `event.Resource`
   - Remove comments about "both modes"

4. **Specification Updates**:
   - Remove "faults mode" scenarios from walking-skeleton spec
   - Remove dual-mode support language
   - Update to reflect single canonical mode

5. **Documentation**:
   - Update comments removing references to "faults" mode
   - Clarify that only resource-faults mode is supported

## Impact

- **Breaking Change**: Nightcrier will no longer work with kubernetes-mcp-server versions that only support "faults" mode
- **No Production Impact**: Nightcrier has never been in production
- **Timing**: This change should be applied after kubernetes-mcp-server removes "faults" mode support

**Affected specs:**
- `walking-skeleton` - Event subscription and parsing requirements

**Affected code:**
- `internal/events/event.go` - FaultEvent struct and helper methods, EventID → FaultID rename
- `internal/events/client.go` - FaultID generation (EventID → FaultID)
- `internal/incident/incident.go` - Resource extraction logic
- Any code referencing `event.EventID` needs update to `event.FaultID`
- Archived specs and proposals (documentation only)

## Dependencies

**PREREQUISITE**: kubernetes-mcp-server must remove "faults" mode support before this change is applied.

## Migration Plan

This is a coordinated breaking change:
1. kubernetes-mcp-server removes "faults" mode
2. Apply this change to nightcrier immediately after
3. No migration path needed (greenfield project)
