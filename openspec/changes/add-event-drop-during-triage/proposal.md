# Proposal: Add Event Drop During Triage

## Summary

Add a `drop_events_while_busy` configuration flag (default: true) that drops inbound fault events for a cluster while triage is already running. When disabled, fall back to the existing queue behavior with a renamed and smaller default queue size.

## Motivation

The current dispatcher queues fault events when a cluster is busy with triage. This creates unnecessary backlog because:

1. **Triage agents examine the entire cluster** - The agent should discover any ongoing issue, not just the one that triggered the run
2. **Queued events create redundant work** - Running triage → process queued event → triage again wastes resources
3. **Event chasing** - In a degraded cluster, queued events can cause a triage loop that never catches up
4. **Eventual consistency** - If an issue persists, monitoring will emit another fault event after triage completes

This follows the "debounce/coalesce" pattern common in event-driven systems (Kubernetes controllers, React state batching, file watchers).

## Changes

### Configuration

1. **Add `drop_events_while_busy`** (bool, default: `true`)
   - When true: drop all events for a cluster while triage is running
   - When false: queue events using existing overflow policy

2. **Rename `cluster_queue_size` → `cluster_failure_event_queue_size`**
   - More descriptive name
   - Change default from 10 to 3

### Dispatcher Behavior

When `drop_events_while_busy: true` and a cluster is busy:
- Log at INFO level: event acknowledged but dropped due to active triage
- Include fault_id, cluster, and reason in log
- Track dropped event count in ClusterState for observability

When `drop_events_while_busy: false`:
- Use existing queue behavior with renamed config field

## Scope

- `internal/dispatcher/dispatcher.go` - Add drop logic
- `internal/dispatcher/queue.go` - No changes needed
- `internal/config/config.go` - Add new field, rename existing field
- `configs/config.example.yaml` - Document new options
- Unit tests for new behavior

## Out of Scope

- Metrics/Prometheus integration (future observability work)
- ~~Per-FaultID deduplication~~ **IMPLEMENTED** - See note below
- "Keep latest only" mode

## Implementation Note: Fault-ID Deduplication

Per-FaultID deduplication was implemented separately to fix a critical bug where duplicate MCP events caused duplicate K8s Jobs to be created for the same fault.

**Implementation (in `internal/dispatcher/dispatcher.go`):**
- Added `seenFaults map[string]time.Time` to track seen fault_ids per cluster
- Uses existing `dedup_window_seconds` config (was defined but unused)
- Key format: `cluster:fault_id` to allow same fault on different clusters
- Cleanup goroutine removes expired entries periodically

**Behavior:**
- First event for a fault_id is processed normally
- Subsequent events for the same fault_id within the dedup window are logged and dropped
- Default window: 300 seconds (5 minutes)

This is complementary to `drop_events_while_busy` - dedup prevents duplicate processing of the same fault, while drop_events_while_busy prevents queueing of different faults during triage.
