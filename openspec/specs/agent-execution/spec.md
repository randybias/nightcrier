# agent-execution Specification

## Purpose
Defines requirements for concurrent agent execution with per-cluster serialization, event TTL handling, queue overflow management, and graceful shutdown behavior.
## Requirements
### Requirement: Concurrent Agent Execution

The system SHALL execute agents concurrently up to a configured global limit while strictly serializing execution per cluster.

#### Scenario: Additional prompt appended to triage prompt
- **GIVEN** `agent.additional_prompt` is configured with operator text
- **WHEN** the triage prompt is built for agent execution
- **THEN** the additional prompt SHALL be appended after all other prompt components
- **AND** the additional prompt SHALL appear under a `## Additional Operator Instructions` heading
- **AND** the complete prompt (including additional prompt) SHALL be captured in `prompt-sent.md`

#### Scenario: Empty additional prompt omitted
- **GIVEN** `agent.additional_prompt` is empty or not configured
- **WHEN** the triage prompt is built for agent execution
- **THEN** no additional prompt section SHALL be appended
- **AND** the base triage prompt SHALL drive investigation methodology

#### Scenario: Additional prompt mounted in container
- **GIVEN** `agent.additional_prompt` is configured
- **WHEN** the agent Job is created
- **THEN** the ConfigMap SHALL include `additional-prompt.md` with the operator text
- **AND** the file SHALL be mounted at `/home/agent/additional-prompt.md`

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

