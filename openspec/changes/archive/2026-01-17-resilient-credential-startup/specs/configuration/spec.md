## ADDED Requirements

### Requirement: Non-Blocking Startup

The system SHALL start immediately regardless of credential availability, retrying failures in the background.

#### Scenario: Start with missing API keys
- **GIVEN** no API keys are configured at startup
- **WHEN** the bootstrap process runs
- **THEN** the system SHALL log a warning indicating missing API keys
- **AND** the system SHALL start immediately in degraded mode
- **AND** the system SHALL begin background retry for API key secret creation.

#### Scenario: Start with missing kubeconfig
- **GIVEN** a monitored cluster's triage kubeconfig file does not exist
- **WHEN** the bootstrap process runs
- **THEN** the system SHALL log a warning for that cluster
- **AND** the system SHALL mark that cluster as degraded
- **AND** the system SHALL start immediately
- **AND** other clusters SHALL bootstrap normally in parallel.

#### Scenario: Start with K8s API unavailable
- **GIVEN** the execution cluster's K8s API is unreachable
- **WHEN** the bootstrap process runs
- **THEN** the system SHALL log a warning
- **AND** the system SHALL start immediately in degraded mode
- **AND** the system SHALL begin background retry for bootstrap.

### Requirement: Parallel Cluster Bootstrap

The system SHALL bootstrap per-cluster resources in parallel for scalability.

#### Scenario: Bootstrap 100 clusters
- **GIVEN** 100 monitored clusters are configured
- **WHEN** the bootstrap process runs
- **THEN** global resources (namespace, RBAC, API keys) SHALL be created first
- **AND** per-cluster secrets SHALL be created in parallel
- **AND** one failing cluster SHALL NOT block other clusters.

#### Scenario: Per-cluster status tracking
- **GIVEN** 100 monitored clusters with 5 having missing kubeconfigs
- **WHEN** the bootstrap process completes
- **THEN** the system SHALL report "95/100 clusters ready"
- **AND** each degraded cluster SHALL have independent retry.

### Requirement: Background Retry with Exponential Backoff

The system SHALL retry failed bootstrap operations in the background with exponential backoff.

#### Scenario: Exponential backoff behavior
- **GIVEN** a bootstrap operation fails
- **WHEN** retrying the operation
- **THEN** the initial retry delay SHALL be 5 seconds (configurable)
- **AND** each subsequent retry SHALL multiply the delay by 2
- **AND** the maximum delay SHALL be 300 seconds (configurable).

#### Scenario: Automatic recovery
- **GIVEN** the system is running with a degraded cluster
- **WHEN** the kubeconfig file becomes available
- **THEN** the background retry SHALL detect the availability
- **AND** the cluster's secret SHALL be created
- **AND** the cluster status SHALL change to ready
- **AND** the system SHALL log the recovery.

### Requirement: Startup Configuration

The system SHALL support configuration of retry behavior.

#### Scenario: Retry initial delay configurable
- **WHEN** `startup.credential_retry_initial` is set to "10s"
- **THEN** the first retry SHALL occur after 10 seconds.

#### Scenario: Retry max delay configurable
- **WHEN** `startup.credential_retry_max` is set to "600s"
- **THEN** retry delays SHALL NOT exceed 600 seconds.

#### Scenario: Default retry settings
- **WHEN** retry settings are not configured
- **THEN** the initial delay SHALL be 5 seconds
- **AND** the maximum delay SHALL be 300 seconds.
