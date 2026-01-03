## ADDED Requirements

### Requirement: NATS Configuration

The system SHALL support optional NATS configuration for progress tracking.

#### Scenario: NATS disabled by default
- **WHEN** the application starts
- **AND** `nats.enabled` is not configured
- **THEN** NATS progress tracking SHALL be disabled
- **AND** the application SHALL function normally without NATS

#### Scenario: NATS enabled via config
- **WHEN** the application starts
- **AND** `nats.enabled` is true
- **THEN** the application SHALL validate NATS configuration
- **AND** the application SHALL attempt to connect to the NATS server

#### Scenario: NATS server URL required when enabled
- **WHEN** the application starts
- **AND** `nats.enabled` is true
- **AND** `nats.server` is not configured
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state "nats.server is required when nats.enabled is true"

#### Scenario: NATS token authentication
- **WHEN** the application starts
- **AND** `nats.enabled` is true
- **AND** `nats.token` is configured
- **THEN** the NATS connection SHALL use token authentication

#### Scenario: NATS token optional warning
- **WHEN** the application starts
- **AND** `nats.enabled` is true
- **AND** `nats.token` is not configured
- **THEN** the application SHALL log a warning "NATS connection without authentication is not recommended"
- **AND** the application SHALL continue with unauthenticated connection

#### Scenario: NATS environment variable mapping
- **WHEN** the application starts
- **THEN** the following environment variables SHALL be recognized:
  - `NATS_ENABLED` maps to `nats.enabled`
  - `NATS_SERVER` maps to `nats.server`
  - `NATS_TOKEN` maps to `nats.token`
  - `NATS_CONNECT_TIMEOUT` maps to `nats.connect_timeout`
  - `NATS_RECONNECT_WAIT` maps to `nats.reconnect_wait`

#### Scenario: NATS config file support
- **WHEN** a YAML config file is provided
- **THEN** the following keys SHALL be recognized:
  ```yaml
  nats:
    enabled: true
    server: "nats://nats.nightcrier.svc:14222"
    token: "${NATS_AUTH_TOKEN}"
    connect_timeout: 5
    reconnect_wait: 2
  ```

#### Scenario: NATS connection timeout default
- **WHEN** the application starts
- **AND** `nats.enabled` is true
- **AND** `nats.connect_timeout` is not configured
- **THEN** the default value of 5 seconds SHALL be used

#### Scenario: NATS reconnect wait default
- **WHEN** the application starts
- **AND** `nats.enabled` is true
- **AND** `nats.reconnect_wait` is not configured
- **THEN** the default value of 2 seconds SHALL be used
