## ADDED Requirements

### Requirement: NATS Progress Event Emission

The system SHALL emit progress events to NATS when progress tracking is enabled.

#### Scenario: Run started event
- **WHEN** `nats.enabled` is true
- **AND** the agent container starts execution
- **THEN** the entrypoint SHALL publish to `triage.<incident-id>.run.started`
- **AND** the payload SHALL include `incident_id`, `agent_cli`, `model`, `cluster`, and `timestamp`

#### Scenario: Run completed event
- **WHEN** `nats.enabled` is true
- **AND** the agent container finishes execution
- **THEN** the entrypoint SHALL publish to `triage.<incident-id>.run.completed`
- **AND** the payload SHALL include `exit_code` and `timestamp`

#### Scenario: Executing event from Claude hooks
- **WHEN** `nats.enabled` is true
- **AND** `AGENT_CLI=claude`
- **AND** Claude invokes the Bash tool
- **THEN** a PreToolUse hook SHALL publish to `triage.<incident-id>.executing`
- **AND** the `activity` field SHALL contain the first 100 characters of the command

#### Scenario: Error event emission
- **WHEN** `nats.enabled` is true
- **AND** an error occurs during agent execution
- **THEN** the entrypoint or hook MAY publish to `triage.<incident-id>.error`
- **AND** the payload SHALL include `error_message`

#### Scenario: NATS publish failure does not block execution
- **WHEN** `nats.enabled` is true
- **AND** a NATS publish operation fails
- **THEN** the agent execution SHALL continue normally
- **AND** a warning SHALL be logged to agent.log

### Requirement: NATS Progress Listener

The nightcrier service SHALL subscribe to progress events when NATS is enabled.

#### Scenario: Listener starts at startup
- **WHEN** nightcrier starts
- **AND** `nats.enabled` is true
- **THEN** a NATS listener goroutine SHALL be started before any agents are spawned
- **AND** the listener SHALL subscribe to `triage.>`

#### Scenario: Activity state updated in database
- **WHEN** the listener receives an "executing" event
- **THEN** the current `current_activity` SHALL be copied to `last_activity`
- **AND** the current `current_activity_started_at` SHALL be copied to `last_activity_finished_at`
- **AND** `current_activity` SHALL be set to the new activity
- **AND** `current_activity_started_at` SHALL be set to the event timestamp

#### Scenario: Listener handles reconnection
- **WHEN** the NATS connection is lost
- **THEN** the listener SHALL attempt to reconnect with exponential backoff
- **AND** events during disconnection SHALL be lost (fire-and-forget)

#### Scenario: Listener graceful shutdown
- **WHEN** nightcrier receives SIGTERM
- **THEN** the listener SHALL drain pending messages
- **AND** the listener SHALL close the NATS connection

### Requirement: Progress Event Payload Schema

Progress events SHALL conform to a defined JSON schema.

#### Scenario: Required fields present
- **WHEN** a progress event is published
- **THEN** the payload SHALL include `incident_id`, `timestamp`, and `event_type`
- **AND** `timestamp` SHALL be in ISO8601 format

#### Scenario: Run event fields
- **WHEN** `event_type` is `run.started` or `run.completed`
- **THEN** the payload SHALL include `agent_cli` and `model`
- **AND** `run.completed` SHALL include `exit_code`

#### Scenario: Executing event fields
- **WHEN** `event_type` is `executing`
- **THEN** the payload SHALL include `activity`
- **AND** `activity` SHALL be the first 100 characters of the command being executed

#### Scenario: Error event fields
- **WHEN** `event_type` is `error`
- **THEN** the payload SHALL include `error_message`

### Requirement: Activity Tracking Schema

The system SHALL track current and last activity in the database.

#### Scenario: agent_executions columns added
- **WHEN** the database migration runs
- **THEN** `agent_executions` SHALL have new columns:
  - `current_activity` (TEXT) - what the agent is doing now
  - `current_activity_started_at` (TIMESTAMP) - when it started
  - `last_activity` (TEXT) - what the agent just finished
  - `last_activity_finished_at` (TIMESTAMP) - when it finished
- **AND** all columns SHALL be nullable for backward compatibility
