## ADDED Requirements

### Requirement: Execution Cluster Configuration

The system SHALL support an `execution_clusters` array defining Kubernetes clusters where agent Jobs run.

#### Scenario: Execution cluster with all settings

- **WHEN** an execution cluster is configured with all settings
- **THEN** the configuration SHALL accept:
  - `name` (required, unique identifier)
  - `kubeconfig_path` (required, path to kubeconfig file)
  - `namespace` (optional, defaults to "nightcrier")
  - `runner_image` (optional, defaults to "nc-agent-runner:latest")
  - `image_pull_policy` (optional, defaults to "IfNotPresent")
  - `timeout` (optional, defaults to 600)
  - `memory_limit` (optional, defaults to "2Gi")
  - `cpu_limit` (optional, defaults to "1")
  - `cleanup_ttl` (optional, defaults to 3600)
  - `max_concurrent_agents` (optional, defaults to 10)

#### Scenario: Execution cluster name uniqueness

- **WHEN** two execution clusters have the same `name` field
- **THEN** the configuration SHALL be rejected with a validation error

#### Scenario: Execution cluster kubeconfig validation

- **WHEN** an execution cluster specifies `kubeconfig_path`
- **AND** the file does not exist
- **THEN** the configuration SHALL be rejected with a validation error

### Requirement: Execution Defaults Configuration

The system SHALL support an `execution_defaults` section providing defaults for all execution clusters.

#### Scenario: Defaults applied to execution clusters

- **WHEN** an execution cluster omits optional fields
- **AND** `execution_defaults` specifies those fields
- **THEN** the execution cluster SHALL use the values from `execution_defaults`

#### Scenario: Per-cluster override of defaults

- **WHEN** an execution cluster specifies a field
- **AND** `execution_defaults` also specifies that field
- **THEN** the execution cluster's value SHALL take precedence

### Requirement: SIGHUP Configuration Reload

The system SHALL reload configuration when receiving SIGHUP signal.

#### Scenario: SIGHUP triggers full reload

- **WHEN** the process receives SIGHUP
- **THEN** it SHALL re-read the YAML configuration file
- **AND** it SHALL re-read cluster configuration from the database
- **AND** database clusters SHALL override YAML clusters by name
- **AND** all changed configuration SHALL be applied

#### Scenario: Reload failure handling

- **WHEN** the process receives SIGHUP
- **AND** the new configuration fails validation
- **THEN** the previous valid configuration SHALL remain in effect
- **AND** an error SHALL be logged

#### Scenario: Reload logging

- **WHEN** configuration is reloaded successfully
- **THEN** the system SHALL log at INFO level:
  - Number of monitored clusters added/removed
  - Number of execution clusters added/removed
  - Any other significant configuration changes

### Requirement: Zero Clusters Startup

The system SHALL start successfully even when no clusters are configured.

#### Scenario: Startup with empty cluster configuration

- **WHEN** the application starts
- **AND** no monitored clusters are configured in YAML or database
- **THEN** the application SHALL start successfully
- **AND** it SHALL log a warning about no clusters configured
- **AND** it SHALL poll the database every 30 seconds for new clusters

#### Scenario: Cluster added after startup

- **WHEN** the application is running with no clusters
- **AND** a cluster is added to the database
- **THEN** the next database poll SHALL detect the new cluster
- **AND** the system SHALL establish a connection to the new cluster

## MODIFIED Requirements

### Requirement: Required Configuration Validation

The system SHALL fail fast at startup when required configuration parameters are missing.

#### Scenario: Missing workspace root

- **WHEN** the application starts
- **AND** `workspace_root` is not configured
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state "workspace_root is required"

#### Scenario: Missing agent timeout

- **WHEN** the application starts
- **AND** `agent.timeout` is not configured
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state "agent.timeout must be >= 1"

#### Scenario: Missing subscribe mode

- **WHEN** the application starts
- **AND** `subscribe_mode` is not configured
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state "subscribe_mode is required"

#### Scenario: Missing agent model

- **WHEN** the application starts
- **AND** `agent.model` is not configured
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state "agent.model is required"

#### Scenario: Missing agent CLI

- **WHEN** the application starts
- **AND** `agent.cli` is not configured
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state "agent.cli is required"

#### Scenario: Optional additional agent prompt

- **WHEN** the application starts
- **AND** `agent.additional_prompt` is not configured
- **THEN** the application SHALL start successfully
- **AND** the system prompt SHALL drive investigation methodology

#### Scenario: Clear error guidance

- **WHEN** a required configuration parameter is missing
- **THEN** the error message SHALL include the config key name
- **AND** the error message SHALL include the environment variable name if applicable
- **AND** the error message SHALL suggest checking config.example.yaml

#### Scenario: Zero clusters allowed at startup

- **WHEN** the application starts
- **AND** no monitored clusters are configured
- **THEN** the application SHALL start successfully with a warning

## REMOVED Requirements

### Requirement: At least one cluster required at startup

**Reason**: The new design allows starting with zero clusters and adding them via database or SIGHUP reload.

**Migration**: Remove validation that requires at least one cluster. Add warning log instead.
