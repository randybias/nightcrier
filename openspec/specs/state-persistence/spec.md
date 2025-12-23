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

