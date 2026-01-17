# admin-ui Specification

## Purpose
TBD - created by archiving change add-admin-ui. Update Purpose after archive.
## Requirements
### Requirement: Admin Dashboard Endpoint

The system SHALL serve a single-page admin dashboard when the `--admin-listen` flag is provided.

#### Scenario: Dashboard served on configured address

- **WHEN** nightcrier is started with `--admin-listen 127.0.0.1:8847`
- **THEN** a GET request to `http://127.0.0.1:8847/admin` returns the dashboard HTML

#### Scenario: No admin server when flag omitted

- **WHEN** nightcrier is started without the `--admin-listen` flag
- **THEN** no admin HTTP server is started

### Requirement: Running Triages Display

The dashboard SHALL display a list of currently running triage agent executions.

#### Scenario: Show active executions

- **WHEN** the dashboard is loaded
- **THEN** the top pane shows agent executions where `job_completed_at IS NULL`
- **AND** each row displays: cluster, incident_id, run state, age

#### Scenario: Run state derivation

- **WHEN** displaying a running triage
- **THEN** the run state is derived from timestamps:
  - `queued`: job_started_at set, run_started_at NULL
  - `running`: run_started_at set, run_completed_at NULL
  - `finishing`: run_completed_at set, job_completed_at NULL

### Requirement: Incidents Display

The dashboard SHALL display all incidents ordered by creation time.

#### Scenario: Show incidents list

- **WHEN** the dashboard is loaded
- **THEN** the bottom pane shows incidents ordered by `created_at DESC`
- **AND** each row displays: created, cluster, severity, fault_type, status, triage indicator, view button

#### Scenario: Triage indicator colors

- **WHEN** displaying an incident
- **THEN** the triage indicator shows:
  - Gray: no execution exists
  - Blue: execution exists and running
  - Green: run completed successfully (run_exit_code == 0)
  - Red: failed (run_exit_code != 0 OR error_message set OR status in failed/agent_failed)

### Requirement: View Artifacts Link

Each incident SHALL have a view button linking to its artifacts in object storage.

#### Scenario: View button links to index.html

- **WHEN** user clicks the View button for an incident
- **THEN** the browser navigates to the object storage URL for that incident's index.html

### Requirement: Database Backend Detection

The admin UI SHALL auto-detect and use the configured database backend.

#### Scenario: PostgreSQL backend

- **WHEN** `state_storage.type` is "postgres" in the config
- **THEN** the admin UI queries the PostgreSQL database

#### Scenario: SQLite backend

- **WHEN** `state_storage.type` is "sqlite" in the config
- **THEN** the admin UI queries the SQLite database

### Requirement: Auto-Refresh

The dashboard SHALL auto-refresh to show updated data.

#### Scenario: Periodic refresh

- **WHEN** the dashboard is displayed
- **THEN** the page auto-refreshes every 5-10 seconds

### Requirement: Early Startup

The admin UI SHALL start as early as possible during Nightcrier initialization.

#### Scenario: Available before permission validation

- **WHEN** Nightcrier starts with `--admin-listen` configured
- **THEN** the admin UI SHALL be available before cluster permission validation completes
- **AND** operators can view status while clusters are being validated

#### Scenario: Requires only database connection

- **GIVEN** the admin UI is starting
- **WHEN** the database connection is available
- **THEN** the admin UI SHALL start serving requests
- **AND** the admin UI SHALL NOT wait for MCP connections or permission validation

### Requirement: System Status Display

The admin UI SHALL display overall system health including bootstrap and credential status.

#### Scenario: Display global status
- **WHEN** the admin UI is loaded
- **THEN** the UI SHALL display a "System Status" section
- **AND** the status SHALL show "Ready", "Degraded", or "Initializing".

#### Scenario: Display API key status
- **GIVEN** some API keys are missing
- **WHEN** the admin UI is loaded
- **THEN** the UI SHALL indicate which API key providers are missing
- **AND** the display SHALL list provider names (anthropic, openai, gemini).

#### Scenario: Display cluster summary
- **GIVEN** 100 monitored clusters with 5 degraded
- **WHEN** the admin UI is loaded
- **THEN** the UI SHALL display "Clusters: 95/100 ready".

### Requirement: Cluster Bootstrap Status Column

The admin UI SHALL display per-cluster bootstrap status in the Clusters pane.

#### Scenario: Bootstrap status column
- **WHEN** the Monitored Clusters table is displayed
- **THEN** a "Bootstrap" column SHALL be present
- **AND** each row SHALL show "Ready", "Degraded", or "Retrying".

#### Scenario: Visual indicator for degraded clusters
- **GIVEN** a cluster is in degraded state
- **WHEN** the Clusters table is displayed
- **THEN** the cluster row SHALL have a visual indicator (color or icon)
- **AND** the indicator SHALL distinguish degraded from ready clusters.

#### Scenario: Last error display
- **GIVEN** a cluster failed bootstrap with an error
- **WHEN** the Clusters table is displayed
- **THEN** the error message SHALL be visible (column or tooltip)
- **AND** the error SHALL be human-readable (e.g., "kubeconfig not found").

#### Scenario: Auto-refresh status
- **GIVEN** the admin UI is displaying cluster status
- **WHEN** a degraded cluster recovers
- **THEN** the UI SHALL update to show the cluster as "Ready"
- **AND** the update SHALL occur within the normal refresh interval.

