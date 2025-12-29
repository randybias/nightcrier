## ADDED Requirements
### Requirement: Concurrency Configuration

The system SHALL require configuration for concurrency limits to ensure resource protection.

#### Scenario: Max concurrent agents required
- **WHEN** the application starts
- **AND** `max_concurrent_agents` is not configured
- **THEN** the application SHALL exit with a non-zero status
- **AND** the error message SHALL state "max_concurrent_agents is required"

#### Scenario: Queue size configuration
- **WHEN** the application starts
- **AND** `global_queue_size` or `cluster_queue_size` is not configured
- **THEN** the application SHALL exit with a non-zero status
