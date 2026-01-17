# state-persistence Specification

## Purpose
TBD - created by archiving change migrate-state-to-sql. Update Purpose after archive.
## Requirements
### Requirement: Abstract State Interface
The application MUST interact with persistent state via a defined interface `StateStore` to decouple logic from storage implementation.

#### Scenario: Interface Definition
Given the `internal/storage` package
When the `StateStore` interface is defined
Then it SHOULD include methods for `CreateIncident`, `UpdateIncidentStatus`, and `CompleteIncident`.

### Requirement: SQLite Support
The application MUST support SQLite as a storage backend.

#### Scenario: SQLite Initialization
Given a configuration specifying `storage.type: "sqlite"`
When the application starts
Then it SHOULD initialize a SQLite database at the configured path and apply pending migrations.

### Requirement: PostgreSQL Support
The application MUST support PostgreSQL as a storage backend.

#### Scenario: Postgres Initialization
Given a configuration specifying `storage.type: "postgres"`
When the application starts
Then it SHOULD connect to the configured PostgreSQL instance and apply pending migrations.

### Requirement: Incident Creation
The application MUST persist new incidents to the storage backend upon receiving a fault event.

#### Scenario: Persist Event
Given a valid FaultEvent received from MCP
When `CreateIncident` is called
Then a new row SHOULD be inserted into `fault_events` and `incidents` with status `PENDING`.

### Requirement: Incident Completion

The application MUST update the incident status and record execution details upon agent completion.

#### Scenario: Record Completion
Given an active incident
When the agent finishes execution
Then `CompleteIncident` SHOULD be called to update `incidents` status and insert into `agent_executions` and `triage_reports`.

#### Scenario: Record Execution Metadata
Given an agent execution is being recorded
When `RecordAgentExecution` is called
Then the execution record SHALL include:
- `agent_cli` - the CLI tool used (claude, codex, gemini, goose)
- `agent_model` - the LLM model used (e.g., sonnet, opus, haiku)
- `cluster_name` - the name of the cluster being investigated

### Requirement: Execution Metadata Schema

The `agent_executions` table SHALL store metadata about the agent and cluster for each execution.

#### Scenario: Agent CLI stored
- **WHEN** an agent execution is recorded
- **THEN** the `agent_cli` column SHALL contain the CLI tool used (claude, codex, gemini, goose)
- **AND** the column SHALL NOT be NULL

#### Scenario: Agent model stored
- **WHEN** an agent execution is recorded
- **THEN** the `agent_model` column SHALL contain the LLM model used
- **AND** the column SHALL NOT be NULL

#### Scenario: Cluster name stored
- **WHEN** an agent execution is recorded
- **THEN** the `cluster_name` column SHALL contain the cluster being investigated
- **AND** the column SHALL NOT be NULL

#### Scenario: Metadata queryable
- **WHEN** querying agent executions
- **THEN** the system SHALL support filtering by `agent_cli`, `agent_model`, and `cluster_name`

### Requirement: Execution Metadata Migration

The database schema SHALL be migrated to add execution metadata columns.

#### Scenario: Forward migration
- **GIVEN** the database is at migration version 1
- **WHEN** migration 000002 is applied
- **THEN** the `agent_executions` table SHALL have `agent_cli`, `agent_model`, and `cluster_name` columns

#### Scenario: Backward migration
- **GIVEN** the database is at migration version 2
- **WHEN** migration 000002 is rolled back
- **THEN** the `agent_cli`, `agent_model`, and `cluster_name` columns SHALL be removed

#### Scenario: Existing data compatibility
- **GIVEN** the database has existing `agent_executions` rows
- **WHEN** migration 000002 is applied
- **THEN** existing rows SHALL have default values for new columns
- **AND** the migration SHALL NOT fail

