## ADDED Requirements

### Requirement: Minimal Event Intake

The system SHALL connect to a kubernetes-mcp-server SSE endpoint and receive fault events.

#### Scenario: Successful SSE connection
- **GIVEN** a valid SSE endpoint URL
- **WHEN** the runner starts
- **THEN** it connects to the endpoint and begins receiving events

#### Scenario: Event parsing
- **GIVEN** an SSE event with JSON data
- **WHEN** the event is received
- **THEN** the JSON is parsed into a FaultEvent struct with cluster_id, namespace, resource_type, resource_name, severity, message, and timestamp fields

### Requirement: Incident Workspace Creation

The system SHALL create a unique workspace directory for each incident.

#### Scenario: Workspace directory created
- **GIVEN** a received FaultEvent
- **WHEN** processing begins
- **THEN** a directory is created at `<WORKSPACE_ROOT>/<incident-uuid>/` with 0700 permissions

#### Scenario: Event context written
- **GIVEN** a created workspace
- **WHEN** context is prepared
- **THEN** the FaultEvent is written as JSON to `<workspace>/event.json`

### Requirement: Stub Agent Execution

The system SHALL execute a stub script to simulate agent processing.

#### Scenario: Script execution
- **GIVEN** a workspace with event.json
- **WHEN** the executor runs
- **THEN** the stub script executes with INCIDENT_ID environment variable set and working directory set to the workspace

#### Scenario: Exit code capture
- **GIVEN** the stub script completes
- **WHEN** execution finishes
- **THEN** the exit code is captured and logged

### Requirement: Result Recording

The system SHALL record execution results to the workspace.

#### Scenario: Result file written
- **GIVEN** script execution completes
- **WHEN** reporting runs
- **THEN** a `result.json` file is written containing incident_id, exit_code, started_at, completed_at, and status fields
