# Admin UI Specification

## Purpose

Provides a minimal developer dashboard for viewing triage run status and incidents during development and operations.

## ADDED Requirements

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
