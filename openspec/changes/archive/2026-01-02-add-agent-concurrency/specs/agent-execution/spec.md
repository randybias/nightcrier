## ADDED Requirements

### Requirement: Concurrent Agent Execution

The system SHALL execute agents concurrently up to a configured global limit while strictly serializing execution per cluster.

#### Scenario: Global concurrency limit enforced
- **GIVEN** `MaxConcurrentAgents` is set to N
- **WHEN** N+1 events arrive for N+1 different clusters
- **THEN** N agents start execution immediately
- **AND** the (N+1)th agent waits until one of the running agents completes

#### Scenario: Per-cluster serialization enforced
- **GIVEN** an agent is currently running for Cluster A
- **WHEN** a new event arrives for Cluster A
- **THEN** the new event is queued in the cluster-specific queue
- **AND** the system does NOT run two agents simultaneously on the same cluster

#### Scenario: Non-blocking ingestion
- **GIVEN** the system is at max capacity
- **WHEN** a new event arrives
- **THEN** the event ingestion loop DOES NOT block
- **AND** the event is accepted and queued (or dropped if expired/queue full)

#### Scenario: Isolated failures
- **GIVEN** an agent panics or crashes during execution
- **WHEN** the dispatcher handles the result
- **THEN** the global semaphore slot is released
- **AND** the cluster lock for that cluster is released
- **AND** the system continues processing other events

### Requirement: Event TTL

The system SHALL drop events that have exceeded a configurable time-to-live (TTL).

#### Scenario: Event expires before dispatch
- **GIVEN** `EventTTLSeconds` is set to 300
- **WHEN** an event arrives that was generated more than 300 seconds ago
- **THEN** the event is dropped immediately
- **AND** a debug log entry is recorded with the event age

#### Scenario: Event expires while queued
- **GIVEN** an event is queued for Cluster A
- **AND** the event's age exceeds `EventTTLSeconds`
- **WHEN** the cluster becomes available
- **THEN** the expired event is pruned from the queue
- **AND** a debug log entry is recorded

#### Scenario: Event processed before expiry
- **GIVEN** an event is queued for Cluster A
- **AND** the event's age is less than `EventTTLSeconds`
- **WHEN** the cluster becomes available
- **THEN** the event is processed normally

### Requirement: Queue Overflow Handling

The system SHALL implement bounded per-cluster queues with drop-oldest overflow policy.

#### Scenario: Queue full, drop oldest
- **GIVEN** the queue for Cluster A is at capacity (`ClusterQueueSize`)
- **WHEN** a new event arrives for Cluster A
- **THEN** the oldest event in the queue is dropped
- **AND** a warning log entry is recorded with the dropped event ID
- **AND** the new event is added to the queue

#### Scenario: Queue not full
- **GIVEN** the queue for Cluster A has fewer than `ClusterQueueSize` events
- **WHEN** a new event arrives for Cluster A
- **AND** the cluster is busy
- **THEN** the event is added to the queue
- **AND** a debug log entry is recorded

### Requirement: Graceful Shutdown

The system SHALL complete in-flight agents during shutdown with a configurable timeout.

#### Scenario: Shutdown with in-flight agents
- **GIVEN** agents are currently executing
- **WHEN** SIGTERM is received
- **THEN** no new events are dispatched
- **AND** the system waits for in-flight agents to complete (up to shutdown timeout)
- **AND** queue contents are logged before exit

#### Scenario: Shutdown timeout exceeded
- **GIVEN** agents are currently executing
- **WHEN** SIGTERM is received
- **AND** agents do not complete within the shutdown timeout
- **THEN** the system logs a warning with the count of incomplete agents
- **AND** the system exits

#### Scenario: Clean shutdown
- **GIVEN** no agents are currently executing
- **WHEN** SIGTERM is received
- **THEN** the system exits cleanly
- **AND** an info log entry confirms clean shutdown
