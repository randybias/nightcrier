# Design: Database-Based Cluster Locks

## Overview

Replace in-memory per-cluster mutexes with database row locks to enable multi-instance coordination.

## Database Schema

### New Table: `cluster_locks`

```sql
CREATE TABLE IF NOT EXISTS cluster_locks (
    cluster TEXT PRIMARY KEY,
    locked_by TEXT NOT NULL,        -- Instance identifier (hostname + PID)
    incident_id TEXT,               -- Current incident being processed
    locked_at TIMESTAMP NOT NULL,
    expires_at TIMESTAMP NOT NULL,  -- Auto-expire stale locks

    CONSTRAINT chk_cluster_locks_cluster CHECK (cluster <> '')
);

CREATE INDEX IF NOT EXISTS idx_cluster_locks_expires_at ON cluster_locks(expires_at);
```

## Lock Acquisition Flow

```
Dispatch(event, cluster)
    │
    ├── Check event TTL/dedup (in-memory, unchanged)
    │
    ├── TryAcquireClusterLock(cluster, incidentID, lockTTL)
    │       │
    │       ├── DELETE FROM cluster_locks WHERE expires_at < NOW()  -- Cleanup stale
    │       │
    │       └── INSERT INTO cluster_locks ... ON CONFLICT DO NOTHING
    │               │
    │               ├── Success (1 row inserted) → Lock acquired
    │               │
    │               └── Failure (0 rows) → Cluster busy, drop/queue event
    │
    ├── Execute agent
    │
    └── ReleaseClusterLock(cluster)
            │
            └── DELETE FROM cluster_locks WHERE cluster = ? AND locked_by = ?
```

## StateStore Interface Extensions

```go
// ClusterLock represents an active lock on a cluster for triage serialization.
type ClusterLock struct {
    Cluster    string
    LockedBy   string    // Instance identifier
    IncidentID string
    LockedAt   time.Time
    ExpiresAt  time.Time
}

// Extended StateStore interface
type StateStore interface {
    // ... existing methods ...

    // TryAcquireClusterLock attempts to acquire an exclusive lock for a cluster.
    // Returns (true, nil) if lock acquired, (false, nil) if cluster busy,
    // or (false, error) on database error.
    TryAcquireClusterLock(ctx context.Context, cluster, lockedBy, incidentID string, ttl time.Duration) (bool, error)

    // ReleaseClusterLock releases a previously acquired lock.
    // Only releases if lockedBy matches (prevents releasing another instance's lock).
    ReleaseClusterLock(ctx context.Context, cluster, lockedBy string) error

    // RefreshClusterLock extends the expiration of an existing lock.
    // Used for long-running agents to prevent premature expiration.
    RefreshClusterLock(ctx context.Context, cluster, lockedBy string, ttl time.Duration) error

    // GetClusterLock returns the current lock status for a cluster (for observability).
    GetClusterLock(ctx context.Context, cluster string) (*ClusterLock, error)
}
```

## PostgreSQL Implementation

```sql
-- TryAcquireClusterLock
WITH cleanup AS (
    DELETE FROM cluster_locks WHERE expires_at < NOW()
)
INSERT INTO cluster_locks (cluster, locked_by, incident_id, locked_at, expires_at)
VALUES ($1, $2, $3, NOW(), NOW() + $4::interval)
ON CONFLICT (cluster) DO NOTHING
RETURNING cluster;
-- Returns 1 row if acquired, 0 if busy
```

## SQLite Implementation

SQLite lacks `FOR UPDATE SKIP LOCKED`, but exclusive transactions provide adequate serialization for single-node deployments:

```sql
BEGIN EXCLUSIVE;
DELETE FROM cluster_locks WHERE expires_at < datetime('now');
INSERT OR IGNORE INTO cluster_locks (cluster, locked_by, incident_id, locked_at, expires_at)
VALUES (?, ?, ?, datetime('now'), datetime('now', '+' || ? || ' seconds'));
-- Check if insert succeeded via changes()
COMMIT;
```

## Dispatcher Changes

### Before (in-memory)
```go
cs.mu.Lock()
if cs.running {
    // busy
}
cs.running = true
cs.mu.Unlock()
```

### After (database)
```go
acquired, err := d.stateStore.TryAcquireClusterLock(ctx, cluster, d.instanceID, incidentID, d.lockTTL)
if err != nil {
    slog.Error("failed to acquire cluster lock", "error", err)
    return
}
if !acquired {
    // Cluster busy - drop or queue
    return
}
defer d.stateStore.ReleaseClusterLock(ctx, cluster, d.instanceID)
```

## Instance Identifier

Each nightcrier instance generates a unique identifier at startup:

```go
instanceID := fmt.Sprintf("%s-%d-%s", hostname, os.Getpid(), uuid.New().String()[:8])
```

This allows:
- Identifying which instance holds a lock (debugging)
- Preventing accidental release of another instance's lock
- Distinguishing restarts of the same process

## Lock TTL and Refresh

- Default lock TTL: `agent_timeout + 60 seconds` (buffer for cleanup)
- For long agents, refresh lock periodically (every TTL/2)
- Stale locks auto-expire and are cleaned up on next acquisition attempt

## Fallback Behavior

If database is unavailable during lock acquisition:
1. Log error with stack trace
2. Drop the event (fail-safe: prefer missing one event over risking duplicate processing)
3. Rely on circuit breaker to disable cluster if database issues persist

## Migration Path

1. Deploy new code with database locking
2. In-memory locks remain as secondary defense (belt and suspenders)
3. After validation, remove in-memory per-cluster locks in follow-up change

## Startup Recovery

On startup, nightcrier must reconcile database state with reality:

```go
func (d *Dispatcher) RecoverStrandedAgents(ctx context.Context) error {
    // 1. Get all active locks
    locks, err := d.stateStore.ListClusterLocks(ctx)
    if err != nil {
        return fmt.Errorf("failed to list cluster locks: %w", err)
    }

    for _, lock := range locks {
        // 2. Check if lock is expired (should be cleaned up automatically, but be defensive)
        if lock.ExpiresAt.Before(time.Now()) {
            d.stateStore.ReleaseClusterLock(ctx, lock.Cluster, lock.LockedBy)
            d.markIncidentStranded(ctx, lock.IncidentID, "lock expired")
            continue
        }

        // 3. Check if the Kubernetes Job still exists
        job, err := d.k8sExecutor.GetJob(ctx, lock.IncidentID)
        if err != nil {
            // Job doesn't exist - stranded
            d.stateStore.ReleaseClusterLock(ctx, lock.Cluster, lock.LockedBy)
            d.markIncidentStranded(ctx, lock.IncidentID, "kubernetes job not found")
            continue
        }

        // 4. Check Job status
        if job.Status.Succeeded > 0 || job.Status.Failed > 0 {
            // Job completed but lock wasn't released
            d.stateStore.ReleaseClusterLock(ctx, lock.Cluster, lock.LockedBy)
            d.finalizeIncidentFromJob(ctx, lock.IncidentID, job)
        }
        // Job still running - leave lock alone, it will expire or complete
    }

    // 5. Find incidents stuck in "investigating" without locks
    strandedIncidents, err := d.stateStore.GetStrandedIncidents(ctx)
    if err != nil {
        return fmt.Errorf("failed to get stranded incidents: %w", err)
    }

    for _, incident := range strandedIncidents {
        d.markIncidentStranded(ctx, incident.ID, "no lock found after restart")
    }

    return nil
}
```

### StateStore Extensions for Recovery

```go
// ListClusterLocks returns all active (non-expired) locks for recovery
ListClusterLocks(ctx context.Context) ([]ClusterLock, error)

// GetStrandedIncidents returns incidents in "investigating" status
// that don't have a corresponding active lock
GetStrandedIncidents(ctx context.Context) ([]Incident, error)
```

### Recovery SQL

```sql
-- GetStrandedIncidents (PostgreSQL)
SELECT i.* FROM incidents i
LEFT JOIN cluster_locks cl ON i.id = cl.incident_id
WHERE i.status = 'investigating'
  AND cl.cluster IS NULL;
```

## Stale Lock Cleanup

Stale locks are cleaned up opportunistically during lock acquisition:

```sql
-- Part of TryAcquireClusterLock
DELETE FROM cluster_locks WHERE expires_at < NOW();
```

This approach:
- Requires no background goroutine
- Cleans up stale locks only when needed
- Keeps the lock table small

For clusters with very low event frequency, expired locks may persist until the next acquisition attempt. This is acceptable since:
- Expired locks don't block new acquisitions (they're deleted first)
- The incident is already marked based on TTL expiration
- No correctness issues, only minor storage overhead

## Observability

- Log lock acquisitions/releases at DEBUG level
- Log lock contention (busy cluster) at INFO level
- Log stale lock cleanup at INFO level with lock details
- Log stranded agent recovery at WARN level
- Expose lock status via `/health/clusters` endpoint
- Add metrics: `nightcrier_cluster_lock_acquired_total`, `nightcrier_cluster_lock_contention_total`, `nightcrier_stranded_agents_recovered_total`
