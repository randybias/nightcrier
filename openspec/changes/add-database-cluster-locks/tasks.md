# Tasks: Add Database-Based Cluster Locks

## Implementation Order

### Phase 1: Database Schema and StateStore

- [ ] Create migration `000005_add_cluster_locks.up.sql` with `cluster_locks` table
- [ ] Create corresponding down migration
- [ ] Add `ClusterLock` struct to `internal/storage/types.go`
- [ ] Extend `StateStore` interface with lock methods in `internal/storage/store.go`
- [ ] Implement PostgreSQL lock methods in `internal/storage/postgres/`
- [ ] Implement SQLite lock methods in `internal/storage/sqlite/`
- [ ] Add unit tests for lock acquisition, release, refresh, and query
- [ ] Test lock expiration and cleanup behavior

### Phase 2: Instance Identifier

- [ ] Add `instanceID` generation at startup in `cmd/nightcrier/main.go`
- [ ] Pass `instanceID` to dispatcher during initialization
- [ ] Log instance ID at startup for debugging

### Phase 3: Dispatcher Integration

- [ ] Update `Dispatcher` to accept `StateStore` and `instanceID`
- [ ] Replace in-memory `cs.running` check with `TryAcquireClusterLock`
- [ ] Add `defer ReleaseClusterLock` after successful acquisition
- [ ] Configure lock TTL from agent timeout + buffer
- [ ] Add lock refresh for long-running agents (optional, can defer)

### Phase 4: Stranded Agent Recovery

- [ ] Add `ListClusterLocks` method to StateStore interface
- [ ] Add `GetStrandedIncidents` method to StateStore interface
- [ ] Implement `RecoverStrandedAgents` in dispatcher
- [ ] On startup: query all active locks from database
- [ ] For each lock: check if Kubernetes Job still exists
  - Job running: leave lock alone
  - Job completed: release lock, finalize incident
  - Job missing: release lock, mark incident as failed/stranded
- [ ] Find incidents in "investigating" without locks, mark as stranded
- [ ] Add `stranded` or update incident status to reflect recovery
- [ ] Call `RecoverStrandedAgents` during nightcrier startup (after StateStore init)

### Phase 5: Testing and Validation

- [ ] Add integration tests for lock contention scenarios
- [ ] Test graceful shutdown releases locks
- [ ] Test lock expiration when agent crashes
- [ ] Test stranded agent recovery on startup
- [ ] Test recovery when Job exists but completed
- [ ] Test recovery when Job doesn't exist
- [ ] Verify single-instance behavior unchanged
- [ ] Manual test with two nightcrier instances

### Phase 6: Observability

- [ ] Add DEBUG logging for lock acquisition/release
- [ ] Add INFO logging for lock contention (cluster busy)
- [ ] Add WARN logging for stranded agent recovery
- [ ] Add INFO logging for stale lock cleanup
- [ ] Expose lock status via health endpoint (optional, can defer)
- [ ] Add metric: `nightcrier_stranded_agents_recovered_total`

## Notes

- Keep in-memory locks as secondary defense during migration
- Remove in-memory per-cluster locks in follow-up change after validation
- Global semaphore (max concurrent agents) remains in-memory
- Stranded agent recovery runs once at startup, not continuously
