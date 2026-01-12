# configuration Specification Delta

## ADDED Requirements

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
  mcp_reconnect_max_backoff: 60
  mcp_read_timeout: 120
  ```
