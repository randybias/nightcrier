# agent-execution Spec Delta

## ADDED Requirements

### Requirement: Event Drop During Active Triage

The system SHALL drop inbound fault events for a cluster while triage is actively running when `DropEventsWhileBusy` is enabled.

#### Scenario: Event dropped while cluster busy (drop mode enabled)
- **GIVEN** `DropEventsWhileBusy` is set to true (default)
- **AND** an agent is currently running for Cluster A
- **WHEN** a new fault event arrives for Cluster A
- **THEN** the event is dropped immediately
- **AND** an INFO log entry is recorded with fault_id, cluster, and reason "active triage in progress"
- **AND** the dropped event count for Cluster A is incremented

#### Scenario: Event queued while cluster busy (drop mode disabled)
- **GIVEN** `DropEventsWhileBusy` is set to false
- **AND** an agent is currently running for Cluster A
- **WHEN** a new fault event arrives for Cluster A
- **THEN** the event is queued using existing overflow policy
- **AND** the behavior matches the existing "Queue Overflow Handling" requirement

#### Scenario: First event always processed
- **GIVEN** no agent is running for Cluster A
- **WHEN** a fault event arrives for Cluster A
- **THEN** the event is processed normally regardless of `DropEventsWhileBusy` setting

## MODIFIED Requirements

### Requirement: Queue Overflow Handling

The system SHALL implement bounded per-cluster queues with drop-oldest overflow policy when queuing is enabled.

#### Scenario: Queue full, drop oldest
- **GIVEN** `DropEventsWhileBusy` is set to false
- **AND** the queue for Cluster A is at capacity (`ClusterFailureEventQueueSize`)
- **WHEN** a new event arrives for Cluster A
- **THEN** the oldest event in the queue is dropped
- **AND** a warning log entry is recorded with the dropped event ID
- **AND** the new event is added to the queue

#### Scenario: Queue not full
- **GIVEN** `DropEventsWhileBusy` is set to false
- **AND** the queue for Cluster A has fewer than `ClusterFailureEventQueueSize` events
- **WHEN** a new event arrives for Cluster A
- **AND** the cluster is busy
- **THEN** the event is added to the queue
- **AND** a debug log entry is recorded

_Note: Configuration field renamed from `ClusterQueueSize` to `ClusterFailureEventQueueSize`. Default changed from 10 to 3._
