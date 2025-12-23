# Proposal: State Persistence Migration (File-System to SQL)

## Why

The current filesystem-based state management limits queryability and observability. As nightcrier scales to handle thousands of MCP servers across multiple clusters, we need:

1. **Structured Querying**: Filter incidents by cluster, resource type, severity, and status without parsing JSON files.
2. **Concurrent Access**: Support multiple readers and potential future worker processes without filesystem locking issues.
3. **Data Integrity**: Use database transactions and foreign keys to maintain referential integrity between incidents, executions, and reports.
4. **Foundation for Observability**: Enable future dashboards and analytics without re-parsing thousands of directory structures.

## What Changes

Refactor the `nightcrier` application to move its primary state management from the local file system to a structured SQL database.

*   **Primary Goal:** Establish a robust, queryable data layer using a repository pattern.
*   **Constraint:** Zero changes to core business logic. The system must continue to treat every incoming MCP event as a unique trigger (1:1 event-to-incident relationship), mirroring current behavior.
*   **Scope:** Forward-looking only. No migration of existing data. No backfill. No reporting UI.

## Architecture: Pluggable Storage

We will implement a **Repository Pattern** to decouple the application logic from the underlying storage engine.

### Interface Definition
The core application will interact with state via a Go interface, likely located in `internal/storage`.

### Database Engines
To validate the pluggability of this design, we will support two drivers immediately:
1.  **SQLite (Default):** Embedded, zero-dependency, ideal for the single-binary CLI distribution.
2.  **PostgreSQL (Validation):** Implemented to prove the abstraction holds. Useful for future multi-replica deployments.

### The Database as a Queue
This data layer serves a dual purpose: persistence and **concurrency control**.
By decoupling ingestion from execution, the `incidents` table acts as the job queue.
*   **Ingestion:** Writes `PENDING` rows.
*   **Workers:** Poll for `PENDING` rows, locking them by updating state to `RUNNING`.

## Implementation Phases

### Phase 1: Storage Abstraction
1.  Define the `StateStore` interface in `internal/storage`.
2.  Create a `noop` or `filesystem` implementation that wraps the *existing* logic, ensuring the interface accurately captures all current side effects.

### Phase 2: SQL Adapters
1.  Initialize `internal/storage/sqlite` and `internal/storage/postgres`.
2.  Implement the `StateStore` interface for both.
3.  Use a migration tool (e.g., `golang-migrate`) to manage the schema creation.

### Phase 3: Integration
1.  Update `cmd/nightcrier/main.go` to initialize the database connection (controlled via config/flags).
2.  Inject the `StateStore` into the `processEvent` loop.
3.  **Logic Check:** Ensure that `CreateIncident` is called immediately upon event receipt, preserving the "firehose" behavior.

### Phase 4: Validation
1.  Run `nightcrier` with SQLite. Trigger a test fault. Verify rows in DB.
2.  Run `nightcrier` with Postgres (local Docker container). Trigger a test fault. Verify rows in DB.

## Non-Goals
*   **Deduplication:** We will allow duplicate `fault_fingerprint` entries in `fault_events`. If the upstream sends it twice, we triage it twice.
*   **UI/Observability:** No new CLI commands for querying the DB will be added in this scope.
*   **Data Migration:** The `incidents/` directory remains as an archive, but the application will not read from it.
