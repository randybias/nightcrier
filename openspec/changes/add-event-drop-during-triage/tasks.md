# Tasks: Add Event Drop During Triage

## Pre-requisite: Fault-ID Deduplication ✅ COMPLETE

Deduplication by fault_id was implemented as an emergency fix for duplicate Jobs.
See proposal.md "Implementation Note" section for details.

- [x] Add `seenFaults map[string]time.Time` to Dispatcher struct
- [x] Add `seenFaultsMu sync.RWMutex` for thread-safe access
- [x] Add `dedupWindow time.Duration` from DedupWindowSeconds config
- [x] Implement `isDuplicate(faultID, cluster)` method in dispatcher.go
- [x] Add dedup check in `Dispatch()` before processing
- [x] Implement `cleanupSeenFaults()` goroutine for expired entries
- [x] Start cleanup goroutine in `NewDispatcher()` if dedup enabled
- [x] Add INFO log when dropping duplicate event
- [x] Update config.example.yaml with improved dedup documentation
- [x] All existing dispatcher tests pass

## Implementation Tasks ✅ COMPLETE

- [x] Add `DropEventsWhileBusy` field to config struct (bool pointer, default true)
- [x] Rename `ClusterQueueSize` to `ClusterFailureEventQueueSize` in config struct
- [x] Update config field tags (mapstructure, env var names)
- [x] Change default queue size from 10 to 3
- [x] Update config validation (if any)
- [x] Add `droppedCount` field to ClusterState for observability
- [x] Update Dispatcher to check `DropEventsWhileBusy` flag in Dispatch()
- [x] Add INFO log when dropping event due to active triage
- [x] Update config.example.yaml with new field names and defaults

## Test Tasks ✅ COMPLETE

- [x] Add unit test: events dropped when busy and flag is true (TestDispatcher_DropEventsWhileBusyEnabled)
- [x] Add unit test: events queued when busy and flag is false (TestDispatcher_DropEventsWhileBusyDisabled)
- [x] Add unit test: default behavior when flag is nil (TestDispatcher_DropEventsWhileBusyDefaultTrue)
- [x] Update existing queue tests to use renamed config field
- [x] Verify config parsing with new field names
- [x] All dispatcher tests pass

## Documentation Tasks

- [ ] Update docs/configuration.md with new options (if file exists)
