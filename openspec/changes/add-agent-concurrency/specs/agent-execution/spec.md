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
- **THEN** the new investigation waits until the current agent for Cluster A completes
- **AND** the system does NOT run two agents simultaneously on the same cluster

#### Scenario: Non-blocking ingestion
- **GIVEN** the system is at max capacity
- **WHEN** a new event arrives
- **THEN** the event ingestion loop DOES NOT block
- **AND** the event is accepted and queued for dispatch (or dropped if queue full)

#### Scenario: Isolated Failures
- **GIVEN** an agent panics or crashes during execution
- **WHEN** the dispatcher handles the result
- **THEN** the global semaphore is released
- **AND** the cluster lock for that cluster is released
- **AND** the system continues processing other events
