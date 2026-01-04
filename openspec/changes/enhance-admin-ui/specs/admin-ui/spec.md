# admin-ui Spec Delta

## ADDED Requirements

### Requirement: Monitored Clusters Display

The dashboard SHALL display the list of monitored clusters from configuration.

#### Scenario: Show clusters pane

- **WHEN** the dashboard is loaded
- **THEN** a "Monitored Clusters" pane appears above Running Triages
- **AND** each row displays: name, environment, MCP endpoint, triage enabled status

#### Scenario: Empty clusters list

- **WHEN** no clusters are configured
- **THEN** the pane shows "No clusters configured"

### Requirement: Delete Incident Action

The dashboard SHALL allow deleting individual incidents.

#### Scenario: Delete button present

- **WHEN** viewing the incidents list
- **THEN** each incident row has a Delete button

#### Scenario: Delete with confirmation

- **WHEN** user clicks Delete on an incident
- **THEN** a confirmation dialog appears
- **AND** if confirmed, a POST request is sent to /admin/incidents/{id}/delete
- **AND** the incident is removed from the database
- **AND** the page refreshes to show updated list

#### Scenario: Delete preserves related data

- **WHEN** an incident is deleted
- **THEN** associated fault records are NOT deleted
- **AND** associated agent_execution records are NOT deleted

### Requirement: Cancel Triage Action

The dashboard SHALL allow cancelling running triage executions.

#### Scenario: Cancel button present

- **WHEN** viewing the running triages list
- **THEN** each triage row has a Cancel button

#### Scenario: Cancel with confirmation

- **WHEN** user clicks Cancel on a running triage
- **THEN** a confirmation dialog appears
- **AND** if confirmed, a POST request is sent to /admin/triages/{id}/cancel
- **AND** the Kubernetes Job is deleted/cancelled
- **AND** the execution record is marked as cancelled (job_completed_at set, error_message = "cancelled by user")
- **AND** the triage disappears from Running Triages on next refresh
