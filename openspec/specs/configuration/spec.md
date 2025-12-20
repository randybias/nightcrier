# configuration Specification

## Purpose
TBD - created by archiving change remove-hardcoded-defaults. Update Purpose after archive.
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

#### Scenario: Missing MCP endpoint
- **WHEN** the application starts
- **AND** `mcp_endpoint` is not configured
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state "mcp_endpoint is required"

#### Scenario: Missing workspace root
- **WHEN** the application starts
- **AND** `workspace_root` is not configured
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state "workspace_root is required"

#### Scenario: Missing agent script path
- **WHEN** the application starts
- **AND** `agent_script_path` is not configured
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state "agent_script_path is required"

#### Scenario: Missing agent timeout
- **WHEN** the application starts
- **AND** `agent_timeout` is not configured
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state "agent_timeout is required"

#### Scenario: Missing subscribe mode
- **WHEN** the application starts
- **AND** `subscribe_mode` is not configured
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state "subscribe_mode is required"

#### Scenario: Missing agent model
- **WHEN** the application starts
- **AND** `agent_model` is not configured
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state "agent_model is required"

#### Scenario: Missing agent CLI
- **WHEN** the application starts
- **AND** `agent_cli` is not configured
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state "agent_cli is required"

#### Scenario: Missing agent image
- **WHEN** the application starts
- **AND** `agent_image` is not configured
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state "agent_image is required"

#### Scenario: Missing agent prompt
- **WHEN** the application starts
- **AND** `agent_prompt` is not configured
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state "agent_prompt is required"

#### Scenario: Clear error guidance
- **WHEN** a required configuration parameter is missing
- **THEN** the error message SHALL include the config key name
- **AND** the error message SHALL include the environment variable name if applicable
- **AND** the error message SHALL suggest checking config.example.yaml

### Requirement: Single Source of Configuration Truth

The system SHALL NOT define default values for required configuration parameters in multiple locations.

#### Scenario: No duplicate defaults in executor
- **WHEN** the agent executor is initialized
- **THEN** it SHALL receive all configuration values from the Config struct
- **AND** it SHALL NOT define its own default values for agent_timeout, agent_model, agent_allowed_tools, or agent_prompt

#### Scenario: No duplicate defaults in shell script
- **WHEN** the agent shell script is invoked
- **THEN** it SHALL receive all configuration values via environment variables
- **AND** it SHALL NOT define default values for AGENT_MODEL, AGENT_TIMEOUT, or CLAUDE_ALLOWED_TOOLS

#### Scenario: Environment variables passed to script
- **WHEN** the Go application invokes the agent script
- **THEN** it SHALL set environment variables for all agent configuration values
- **AND** the script SHALL use those environment variables without fallback defaults

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

