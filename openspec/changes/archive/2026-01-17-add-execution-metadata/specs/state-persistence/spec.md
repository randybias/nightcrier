## MODIFIED Requirements

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

## ADDED Requirements

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
