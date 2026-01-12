## ADDED Requirements

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
