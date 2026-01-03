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

## Implementation Tasks (drop_events_while_busy - NOT YET STARTED)

- [ ] Add `DropEventsWhileBusy` field to config struct (bool, default true)
- [ ] Rename `ClusterQueueSize` to `ClusterFailureEventQueueSize` in config struct
- [ ] Update config field tags (mapstructure, env var names)
- [ ] Change default queue size from 10 to 3
- [ ] Update config validation (if any)
- [ ] Add `droppedCount` field to ClusterState for observability
- [ ] Update Dispatcher to check `DropEventsWhileBusy` flag in Dispatch()
- [ ] Add INFO log when dropping event due to active triage
- [ ] Update config.example.yaml with new field names and defaults
- [ ] Update any other example configs (codex, gemini, claude, goose)

## Test Tasks

- [ ] Add unit test: events dropped when busy and flag is true
- [ ] Add unit test: events queued when busy and flag is false
- [ ] Add unit test: dropped count tracked correctly
- [ ] Update existing queue tests to use renamed config field
- [ ] Verify config parsing with new field names

## Documentation Tasks

- [ ] Update docs/configuration.md with new options
