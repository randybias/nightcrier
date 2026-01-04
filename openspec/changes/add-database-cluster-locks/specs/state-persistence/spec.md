## ADDED Requirements

### Requirement: Cluster Lock Acquisition

The StateStore MUST support acquiring exclusive per-cluster locks for triage serialization.

#### Scenario: Successful lock acquisition
- **GIVEN** no active lock exists for cluster "prod-east"
- **WHEN** TryAcquireClusterLock is called with cluster="prod-east", lockedBy="instance-1", ttl=10m
- **THEN** the method SHALL return (true, nil)
- **AND** a row SHALL be inserted into cluster_locks with expires_at = now + 10m

#### Scenario: Lock acquisition when cluster busy
- **GIVEN** an active lock exists for cluster "prod-east" held by "instance-1"
- **WHEN** TryAcquireClusterLock is called with cluster="prod-east", lockedBy="instance-2"
- **THEN** the method SHALL return (false, nil)
- **AND** no changes SHALL be made to cluster_locks

#### Scenario: Lock acquisition with expired lock
- **GIVEN** an expired lock exists for cluster "prod-east" (expires_at < now)
- **WHEN** TryAcquireClusterLock is called with cluster="prod-east", lockedBy="instance-2"
- **THEN** the expired lock SHALL be deleted
- **AND** the method SHALL return (true, nil)
- **AND** a new lock SHALL be created for "instance-2"

### Requirement: Cluster Lock Release

The StateStore MUST support releasing previously acquired cluster locks.

#### Scenario: Release own lock
- **GIVEN** an active lock for cluster "prod-east" held by "instance-1"
- **WHEN** ReleaseClusterLock is called with cluster="prod-east", lockedBy="instance-1"
- **THEN** the lock row SHALL be deleted
- **AND** the method SHALL return nil

#### Scenario: Release lock held by another instance
- **GIVEN** an active lock for cluster "prod-east" held by "instance-1"
- **WHEN** ReleaseClusterLock is called with cluster="prod-east", lockedBy="instance-2"
- **THEN** the lock row SHALL NOT be deleted
- **AND** the method SHALL return nil (no error, but no effect)

#### Scenario: Release non-existent lock
- **GIVEN** no lock exists for cluster "prod-east"
- **WHEN** ReleaseClusterLock is called with cluster="prod-east"
- **THEN** the method SHALL return nil (idempotent)

### Requirement: Cluster Lock Refresh

The StateStore MUST support extending the expiration of an active lock.

#### Scenario: Refresh own lock
- **GIVEN** an active lock for cluster "prod-east" held by "instance-1" expiring in 5 minutes
- **WHEN** RefreshClusterLock is called with cluster="prod-east", lockedBy="instance-1", ttl=10m
- **THEN** the expires_at timestamp SHALL be updated to now + 10m
- **AND** the method SHALL return nil

#### Scenario: Refresh lock held by another instance
- **GIVEN** an active lock for cluster "prod-east" held by "instance-1"
- **WHEN** RefreshClusterLock is called with cluster="prod-east", lockedBy="instance-2"
- **THEN** the lock SHALL NOT be modified
- **AND** the method SHALL return an error indicating lock ownership mismatch

### Requirement: Cluster Lock Query

The StateStore MUST support querying the current lock status for observability.

#### Scenario: Get active lock
- **GIVEN** an active lock for cluster "prod-east"
- **WHEN** GetClusterLock is called with cluster="prod-east"
- **THEN** the method SHALL return the ClusterLock struct with all fields populated

#### Scenario: Get lock for unlocked cluster
- **GIVEN** no lock exists for cluster "prod-east"
- **WHEN** GetClusterLock is called with cluster="prod-east"
- **THEN** the method SHALL return (nil, nil)

### Requirement: Startup Lock Recovery

The StateStore MUST support identifying orphaned locks on startup for recovery.

#### Scenario: List all active locks
- **GIVEN** locks exist for multiple clusters and instances
- **WHEN** ListClusterLocks is called
- **THEN** the method SHALL return all active (non-expired) locks
- **AND** each lock SHALL include the locked_by identifier for instance detection
- **AND** each lock SHALL include the incident_id for correlation

#### Scenario: List locks includes expired locks for defensive cleanup
- **GIVEN** an expired lock exists for cluster "prod-east"
- **WHEN** ListClusterLocks is called
- **THEN** the method SHALL include expired locks in the result
- **AND** the caller can identify expired locks via expires_at < now

### Requirement: Stranded Incident Detection

The StateStore MUST support finding incidents stuck in investigating status without corresponding locks.

#### Scenario: Incident investigating without lock
- **GIVEN** an incident "inc-123" in "investigating" status
- **AND** no lock exists in cluster_locks with incident_id="inc-123"
- **WHEN** GetStrandedIncidents is called
- **THEN** the method SHALL return incident "inc-123" in the result

#### Scenario: Incident investigating with valid lock
- **GIVEN** an incident "inc-456" in "investigating" status
- **AND** a lock exists in cluster_locks with incident_id="inc-456"
- **WHEN** GetStrandedIncidents is called
- **THEN** the method SHALL NOT return incident "inc-456"

#### Scenario: Resolved incidents not returned
- **GIVEN** an incident "inc-789" in "resolved" status without a lock
- **WHEN** GetStrandedIncidents is called
- **THEN** the method SHALL NOT return incident "inc-789"

### Requirement: Stale Lock Cleanup

Lock acquisition MUST clean up expired locks atomically to prevent deadlock.

#### Scenario: Cleanup during acquisition
- **GIVEN** expired locks exist for clusters "cluster-a" and "cluster-b"
- **WHEN** TryAcquireClusterLock is called for cluster "cluster-c"
- **THEN** the expired locks for "cluster-a" and "cluster-b" SHALL be deleted
- **AND** the lock for "cluster-c" SHALL be acquired normally

#### Scenario: Acquire previously expired lock
- **GIVEN** an expired lock exists for cluster "prod-east" held by "instance-1"
- **WHEN** TryAcquireClusterLock is called with cluster="prod-east", lockedBy="instance-2"
- **THEN** the expired lock SHALL be deleted
- **AND** the method SHALL return (true, nil)
- **AND** the new lock SHALL be held by "instance-2"
