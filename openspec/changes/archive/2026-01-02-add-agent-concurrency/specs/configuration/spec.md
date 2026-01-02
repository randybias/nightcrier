## MODIFIED Requirements

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
