# configuration Specification

## Purpose
Defines requirements for application configuration management including required field validation, tuning parameters, environment variable mappings, and nested configuration structures for agent and Kubernetes settings.
## Requirements
### Requirement: Tuning Configuration

The system SHALL support a separate tuning configuration file for operational parameters that are rarely changed.

#### Scenario: Tuning file loaded
- **WHEN** the application starts
- **AND** `configs/tuning.yaml` exists
- **THEN** tuning parameters SHALL be loaded from that file

#### Scenario: Tuning file optional
- **WHEN** the application starts
- **AND** `configs/tuning.yaml` does not exist
- **THEN** the application SHALL use built-in fallback values for tuning parameters
- **AND** the application SHALL NOT fail to start

#### Scenario: Tuning parameters documented
- **WHEN** `configs/tuning.yaml` is created
- **THEN** each parameter SHALL include a comment explaining its purpose and default value

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

### Requirement: Single Source of Configuration Truth

The system SHALL NOT define default values for required configuration parameters in multiple locations.

#### Scenario: No deprecated helper methods
- **WHEN** checking storage backend type
- **THEN** code SHALL access `ObjectStorage.Type` directly
- **AND** deprecated helper methods like `IsAzureStorageEnabled()` SHALL NOT exist

_Note: This scenario documents removal of the deprecated `IsAzureStorageEnabled()` method. Code should access config fields directly rather than through deprecated wrapper methods._

### Requirement: Tuning Parameters

The system SHALL make the following operational parameters configurable via tuning.yaml.

#### Scenario: HTTP client timeout configurable
- **WHEN** tuning.yaml specifies `http.slack_timeout_seconds`
- **THEN** the Slack notifier SHALL use that timeout value for HTTP requests

#### Scenario: Agent timeout buffer configurable
- **WHEN** tuning.yaml specifies `agent.timeout_buffer_seconds`
- **THEN** the agent executor SHALL add that buffer to the configured agent timeout

#### Scenario: Investigation minimum size configurable
- **WHEN** tuning.yaml specifies `agent.investigation_min_size_bytes`
- **THEN** the agent failure detection SHALL use that threshold

#### Scenario: Root cause truncation configurable
- **WHEN** tuning.yaml specifies `reporting.root_cause_truncation_length`
- **THEN** Slack notifications SHALL truncate root cause to that length

#### Scenario: Failure reasons display count configurable
- **WHEN** tuning.yaml specifies `reporting.failure_reasons_display_count`
- **THEN** degradation alerts SHALL show that many recent failure reasons

#### Scenario: Max failure reasons tracked configurable
- **WHEN** tuning.yaml specifies `reporting.max_failure_reasons_tracked`
- **THEN** the circuit breaker SHALL track that many recent failure reasons

#### Scenario: Event channel buffer size configurable
- **WHEN** tuning.yaml specifies `events.channel_buffer_size`
- **THEN** the MCP client SHALL use that buffer size for the event channel

#### Scenario: I/O buffer sizes configurable
- **WHEN** tuning.yaml specifies `io.stdout_buffer_size` and `io.stderr_buffer_size`
- **THEN** the agent executor SHALL use those buffer sizes for output capture

### Requirement: Concurrency Configuration

The system SHALL require configuration for concurrency limits and queue behavior.

#### Scenario: Max concurrent agents required
- **WHEN** the application starts
- **AND** `max_concurrent_agents` is not configured
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state "max_concurrent_agents is required"

#### Scenario: Max concurrent agents validation
- **WHEN** the application starts
- **AND** `max_concurrent_agents` is less than 1
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state "max_concurrent_agents must be >= 1"

#### Scenario: Cluster queue size default
- **WHEN** the application starts
- **AND** `cluster_queue_size` is not configured
- **THEN** the default value of 10 SHALL be used

#### Scenario: Cluster queue size validation
- **WHEN** the application starts
- **AND** `cluster_queue_size` is less than 1
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state "cluster_queue_size must be >= 1"

#### Scenario: Event TTL default
- **WHEN** the application starts
- **AND** `event_ttl_seconds` is not configured
- **THEN** the default value of 300 (5 minutes) SHALL be used

#### Scenario: Event TTL validation
- **WHEN** the application starts
- **AND** `event_ttl_seconds` is less than 1
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state "event_ttl_seconds must be >= 1"

#### Scenario: Environment variable mapping
- **WHEN** the application starts
- **THEN** the following environment variables SHALL be recognized:
  - `MAX_CONCURRENT_AGENTS` maps to `max_concurrent_agents`
  - `CLUSTER_QUEUE_SIZE` maps to `cluster_queue_size`
  - `EVENT_TTL_SECONDS` maps to `event_ttl_seconds`

#### Scenario: Config file support
- **WHEN** a YAML config file is provided
- **THEN** the following keys SHALL be recognized:
  ```yaml
  max_concurrent_agents: 10
  cluster_queue_size: 10
  event_ttl_seconds: 300
  ```

### Requirement: Single Run Mode

The system SHALL support a single-run execution mode for test harness integration.

#### Scenario: Single run flag accepted

- **WHEN** nightcrier is started with `--single-run` flag
- **THEN** the application SHALL start normally and connect to the MCP server
- **AND** the flag SHALL be recognized without error

#### Scenario: Process first event and exit

- **WHEN** nightcrier is running in single-run mode
- **AND** the first fault event is received
- **THEN** the event SHALL be fully processed (agent execution, reporting, notifications)
- **AND** after processing completes, nightcrier SHALL exit with code 0

#### Scenario: Subsequent events dropped

- **WHEN** nightcrier is running in single-run mode
- **AND** the first fault event processing has triggered shutdown
- **AND** additional fault events arrive
- **THEN** those events SHALL be dropped (not processed)
- **AND** the application SHALL continue its graceful shutdown

#### Scenario: Normal mode unchanged

- **WHEN** nightcrier is started without `--single-run` flag
- **THEN** the application SHALL run continuously
- **AND** all fault events SHALL be processed
- **AND** the application SHALL only exit on SIGINT/SIGTERM

### Requirement: MCP Transport Connection Settings

The system SHALL support configuration for MCP transport connection management.

#### Scenario: MCP reconnect initial backoff required

- **WHEN** the application starts
- **AND** `mcp_reconnect_initial_backoff` is not configured
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL include the field name `mcp_reconnect_initial_backoff`
- **AND** the error message SHALL include the environment variable `MCP_RECONNECT_INITIAL_BACKOFF`

#### Scenario: MCP reconnect initial backoff validation

- **WHEN** the application starts
- **AND** `mcp_reconnect_initial_backoff` is less than 1
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state "mcp_reconnect_initial_backoff must be >= 1"

#### Scenario: MCP reconnect max backoff required

- **WHEN** the application starts
- **AND** `mcp_reconnect_max_backoff` is not configured
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL include the field name `mcp_reconnect_max_backoff`
- **AND** the error message SHALL include the environment variable `MCP_RECONNECT_MAX_BACKOFF`

#### Scenario: MCP reconnect max backoff validation

- **WHEN** the application starts
- **AND** `mcp_reconnect_max_backoff` is less than `mcp_reconnect_initial_backoff`
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL indicate max must be >= initial

#### Scenario: MCP read timeout required

- **WHEN** the application starts
- **AND** `mcp_read_timeout` is not configured
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL include the field name `mcp_read_timeout`
- **AND** the error message SHALL include the environment variable `MCP_READ_TIMEOUT_SECONDS`

#### Scenario: MCP read timeout validation

- **WHEN** the application starts
- **AND** `mcp_read_timeout` is less than 1
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state "mcp_read_timeout must be >= 1"

#### Scenario: Environment variable mapping for MCP settings

- **WHEN** the application starts
- **THEN** the following environment variables SHALL be recognized:
  - `MCP_RECONNECT_INITIAL_BACKOFF` maps to `mcp_reconnect_initial_backoff`
  - `MCP_RECONNECT_MAX_BACKOFF` maps to `mcp_reconnect_max_backoff`
  - `MCP_READ_TIMEOUT_SECONDS` maps to `mcp_read_timeout`

#### Scenario: Config file support for MCP settings

- **WHEN** a YAML config file is provided
- **THEN** the following keys SHALL be recognized:
  ```yaml
  mcp_reconnect_initial_backoff: 1
  mcp_reconnect_max_backoff: 300
  mcp_read_timeout: 120
  ```

### Requirement: MCP API Key Authentication

The system SHALL support API key authentication for MCP server connections when configured.

#### Scenario: API key sent in Authorization header
- **WHEN** a monitored cluster has `mcp.api_key` configured
- **AND** the events client connects to the MCP endpoint
- **THEN** the HTTP request SHALL include an `Authorization: Bearer <api_key>` header

#### Scenario: No Authorization header when API key absent
- **WHEN** a monitored cluster does not have `mcp.api_key` configured
- **AND** the events client connects to the MCP endpoint
- **THEN** the HTTP request SHALL NOT include an `Authorization` header

#### Scenario: API key not logged
- **WHEN** the application logs MCP connection information
- **THEN** the API key value SHALL NOT appear in log output
- **AND** the presence of authentication MAY be logged (e.g., "connecting with API key authentication")

### Requirement: TLS Enforcement for Authenticated Connections

The system SHALL require TLS (HTTPS) when API key authentication is configured to prevent credential exposure.

#### Scenario: HTTPS required with API key
- **WHEN** the application starts
- **AND** a monitored cluster has `mcp.api_key` configured
- **AND** the `mcp.endpoint` does not start with `https://`
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state that HTTPS is required when API key is configured

#### Scenario: HTTPS allowed without API key
- **WHEN** the application starts
- **AND** a monitored cluster does not have `mcp.api_key` configured
- **AND** the `mcp.endpoint` starts with `https://`
- **THEN** the application SHALL start successfully

#### Scenario: HTTP allowed without API key
- **WHEN** the application starts
- **AND** a monitored cluster does not have `mcp.api_key` configured
- **AND** the `mcp.endpoint` starts with `http://`
- **THEN** the application SHALL start successfully

#### Scenario: Valid HTTPS with API key
- **WHEN** the application starts
- **AND** a monitored cluster has `mcp.api_key` configured
- **AND** the `mcp.endpoint` starts with `https://`
- **THEN** the application SHALL start successfully
- **AND** the events client SHALL connect with the Authorization header

### Requirement: API Key Configuration

The system SHALL support API key configuration via config file and environment variables.

#### Scenario: API key from config file
- **WHEN** a monitored cluster specifies `mcp.api_key` in the config file
- **THEN** that value SHALL be used for authentication

#### Scenario: API key environment variable documented
- **WHEN** an operator configures API key authentication
- **THEN** the config.example.yaml SHALL document the `mcp.api_key` field
- **AND** the documentation SHALL note the TLS requirement
- **AND** the documentation SHALL recommend using environment variable substitution for secrets

### Requirement: Resilient Credential Startup

The system SHALL support non-blocking startup with background retry for failed components.

#### Scenario: Credential retry initial backoff configuration
- **WHEN** the application starts
- **AND** `startup.credential_retry_initial` is configured
- **THEN** the background retry loop SHALL use that value as the initial backoff duration

#### Scenario: Credential retry initial backoff default
- **WHEN** the application starts
- **AND** `startup.credential_retry_initial` is not configured
- **THEN** the default value of 5 seconds SHALL be used

#### Scenario: Credential retry max backoff configuration
- **WHEN** the application starts
- **AND** `startup.credential_retry_max` is configured
- **THEN** the background retry loop SHALL cap the backoff at that value

#### Scenario: Credential retry max backoff default
- **WHEN** the application starts
- **AND** `startup.credential_retry_max` is not configured
- **THEN** the default value of 300 seconds (5 minutes) SHALL be used

#### Scenario: Non-blocking bootstrap
- **WHEN** the application starts
- **AND** a component fails to bootstrap (namespace, RBAC, secrets, or cluster kubeconfig)
- **THEN** the application SHALL NOT exit with an error
- **AND** the application SHALL start in degraded mode
- **AND** the application SHALL retry failed components in the background

#### Scenario: Parallel cluster bootstrap
- **WHEN** the application starts with multiple monitored clusters
- **THEN** each cluster's kubeconfig secret SHALL be bootstrapped in parallel
- **AND** one cluster's failure SHALL NOT block other clusters

#### Scenario: Automatic recovery
- **WHEN** a component is in degraded state
- **AND** the background retry succeeds
- **THEN** the system SHALL automatically recover
- **AND** the system SHALL log the recovery at INFO level

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

