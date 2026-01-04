# Proposal: Add Database-Based Cluster Locks

## Problem Statement

Currently, per-cluster serialization (ensuring only one triage agent runs per cluster at a time) is implemented using in-memory Go mutexes in the dispatcher:

```go
type ClusterState struct {
    mu      sync.Mutex
    running bool
    // ...
}
```

This approach fails in multi-instance deployments. If two nightcrier instances run simultaneously (e.g., for high availability or during rolling deployments), each maintains its own lock state. Two agents could triage the same cluster concurrently, violating the serialization invariant.

## Proposed Solution

Move the per-cluster lock to the database layer using row-level locking. When an agent starts for a cluster, acquire an exclusive lock on a `cluster_locks` row. Release the lock when the agent completes.

**Approach**: Use a dedicated `cluster_locks` table with `SELECT FOR UPDATE SKIP LOCKED` semantics:
- PostgreSQL: Native row-level locking
- SQLite: Table-level locking via transactions (acceptable for single-node deployments)

## Scope

1. Add `cluster_locks` table to track active triage sessions per cluster
2. Add lock acquisition/release methods to StateStore interface
3. Update dispatcher to use database locks instead of in-memory mutexes
4. Maintain deduplication and TTL logic in memory (these are optimization, not correctness-critical)

## Additional Problem: Stale Locks and Stranded Triage Agents

### Stale Locks

Locks become stale when:
- Nightcrier crashes or is killed without releasing locks
- Network partition prevents lock release
- Process hangs indefinitely

**Solution**: Every lock has an `expires_at` timestamp. Lock acquisition automatically cleans up expired locks before attempting to insert. No background goroutine needed.

### Stranded Triage Agents

When nightcrier restarts while agents are running:
1. The in-memory lock state is lost
2. The incident remains in `investigating` status in the database
3. The Kubernetes Job may still be running (or completed without nightcrier knowing)
4. New events for that cluster may trigger new agents, causing parallel execution

**Solution**: Startup recovery process:
1. Query all active locks from `cluster_locks`
2. For locks held by the current instance ID (impossible after restart with new UUID), release them
3. For locks held by other instance IDs, check if the Kubernetes Job still exists:
   - If Job exists and running: leave lock alone (another instance is handling it)
   - If Job exists and completed: release lock, mark incident based on Job status
   - If Job doesn't exist: release lock, mark incident as `failed` (stranded)
4. Query incidents in `investigating` status without corresponding locks:
   - Mark as `failed` with reason "stranded after restart"

The database lock table enables this by:
- Persisting lock state across restarts
- Using expiration timestamps to auto-release stale locks
- Storing incident_id to correlate locks with incidents
- Allowing startup recovery to detect and clean up orphaned state

## Out of Scope

- Distributed global concurrency limiting (global semaphore remains in-memory)
- Event queuing in database (remains in-memory per instance)
- Leader election or cluster-wide coordination beyond per-cluster serialization

## Requirements Changed

- **state-persistence**: Add cluster lock acquisition and release requirements
- **cluster-registry**: Add distributed lock semantics for multi-instance deployments

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Lock not released on crash | Use `expires_at` timestamp; stale locks auto-expire after configurable TTL |
| Database contention | Use `SKIP LOCKED` to allow fast rejection without blocking |
| SQLite limitations | Document that SQLite provides weaker guarantees in rare edge cases |

## Success Criteria

1. Two nightcrier instances cannot run agents for the same cluster simultaneously
2. Single-instance behavior unchanged (no performance regression)
3. Lock automatically released on agent completion or expiration
4. Existing tests continue to pass
