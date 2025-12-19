## MODIFIED Requirements

### Requirement: Incident Workspace Creation

The system SHALL create a unique workspace directory for each incident with proper file structure.

#### Scenario: Workspace directory created
- **GIVEN** a received fault event passes filtering
- **WHEN** processing begins
- **THEN** an Incident is created with a unique incidentId (UUID)
- **AND** a directory is created at `<WORKSPACE_ROOT>/<incident-uuid>/` with 0700 permissions

#### Scenario: Incident context written
- **GIVEN** a created workspace
- **WHEN** context is prepared for the agent
- **THEN** an `incident.json` file is written containing incidentId, status, cluster, namespace, resource, faultType, severity, context, and timestamp
- **AND** the status is set to "investigating"

#### Scenario: Incident updated after completion
- **GIVEN** agent execution completes
- **WHEN** results are recorded
- **THEN** the `incident.json` file is updated in place with completedAt, exitCode, and updated status
- **AND** no separate result.json file is created

### Requirement: Containerized Agent Execution

The system SHALL execute a containerized agent for incident triage using incident context.

#### Scenario: Agent invocation
- **GIVEN** a workspace with incident.json
- **WHEN** the executor runs
- **THEN** the agent container is launched with access to incident.json
- **AND** the agent reads incident context from incident.json (not event.json)

#### Scenario: Investigation report
- **GIVEN** agent execution completes successfully
- **WHEN** the agent finishes triage
- **THEN** an investigation report is written to `<workspace>/output/investigation.md`

## REMOVED Requirements

### Requirement: Result Recording

**Reason**: Result metadata (exitCode, status, completedAt) is now stored directly in incident.json rather than a separate result.json file.

**Migration**: The Incident struct includes all fields previously in Result. The incident.json file is updated in place after agent execution.

## ADDED Requirements

### Requirement: Event-Incident Separation

The system SHALL maintain clean separation between events (input) and incidents (response).

#### Scenario: Event remains immutable
- **GIVEN** a fault event is received from MCP
- **WHEN** the event is parsed
- **THEN** an eventId is generated for tracing
- **AND** the event data is not modified with incident metadata

#### Scenario: Incident created from event
- **GIVEN** an event passes severity filtering and deduplication
- **WHEN** processing begins
- **THEN** an Incident is created with relevant event data flattened into it
- **AND** the Incident references the triggeringEventId for traceability

### Requirement: Incident Lifecycle Status

The system SHALL track incident status through its lifecycle.

#### Scenario: Status transitions
- **GIVEN** an incident is created
- **THEN** status progresses through: investigating → resolved|failed|agent_failed
- **AND** timestamps are recorded for each transition (createdAt, startedAt, completedAt)

#### Scenario: Resolved status
- **GIVEN** agent exits with code 0 and produces valid investigation.md
- **WHEN** incident is updated
- **THEN** status is set to "resolved"

#### Scenario: Failed status
- **GIVEN** agent exits with non-zero code or crashes
- **WHEN** incident is updated
- **THEN** status is set to "failed"
- **AND** failureReason describes the failure

#### Scenario: Agent failed status
- **GIVEN** agent exits successfully but produces no valid output
- **WHEN** incident is updated
- **THEN** status is set to "agent_failed"
- **AND** failureReason indicates missing or invalid investigation report
