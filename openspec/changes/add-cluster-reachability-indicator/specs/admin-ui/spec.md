# admin-ui Spec Delta

## ADDED Requirements

### Requirement: Cluster Reachability Indicator

The Monitored Clusters pane SHALL display a live reachability status for each cluster.

#### Scenario: Green indicator for active connection

- **GIVEN** a cluster with ConnectionStatus = "active"
- **WHEN** the dashboard is loaded or refreshed
- **THEN** a green indicator dot appears in the STATUS column
- **AND** hovering shows tooltip "active"

#### Scenario: Amber indicator for transient states

- **GIVEN** a cluster with ConnectionStatus in ["connecting", "connected", "subscribing", "disconnected"]
- **WHEN** the dashboard is loaded or refreshed
- **THEN** an amber indicator dot appears in the STATUS column
- **AND** hovering shows the actual status string

#### Scenario: Red indicator for failed connection

- **GIVEN** a cluster with ConnectionStatus = "failed"
- **WHEN** the dashboard is loaded or refreshed
- **THEN** a red indicator dot appears in the STATUS column
- **AND** hovering shows tooltip "failed"

#### Scenario: Status column in table

- **WHEN** viewing the Monitored Clusters pane
- **THEN** a STATUS column appears as the first column
- **AND** each row shows a colored indicator dot
