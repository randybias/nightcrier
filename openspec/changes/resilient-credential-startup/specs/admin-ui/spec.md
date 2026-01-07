## ADDED Requirements

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
