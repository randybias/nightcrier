# Plan: State Persistence Migration (File-System to SQL)

## 1. Objective
Refactor the `nightcrier` application to move its primary state management from the local file system to a structured SQL database.
*   **Primary Goal:** Establish a robust, queryable data layer using a repository pattern.
*   **Constraint:** Zero changes to core business logic. The system must continue to treat every incoming MCP event as a unique trigger (1:1 event-to-incident relationship), mirroring current behavior.
*   **Scope:** Forward-looking only. No migration of existing data. No backfill. No reporting UI.

## 2. Architecture: Pluggable Storage

We will implement a **Repository Pattern** to decouple the application logic from the underlying storage engine.

### 2.1 Interface Definition
The core application will interact with state via a Go interface, likely located in `internal/storage`:

```go
type StateStore interface {
    // Create a new incident from a raw MCP event
    CreateIncident(ctx context.Context, event *events.FaultEvent) (*incident.Incident, error)
    
    // Update the status of an ongoing incident (e.g., Running -> Completed)
    UpdateIncidentStatus(ctx context.Context, id uuid.UUID, status string) error
    
    // Record the result of an agent's execution
    CompleteIncident(ctx context.Context, id uuid.UUID, result *incident.Result) error
}
```

### 2.2 Database Engines
To validate the pluggability of this design, we will support two drivers immediately:
1.  **SQLite (Default):** Embedded, zero-dependency, ideal for the single-binary CLI distribution.
2.  **PostgreSQL (Validation):** Implemented to prove the abstraction holds. Useful for future multi-replica deployments.

### 2.3 The Database as a Queue
This data layer serves a dual purpose: persistence and **concurrency control**.
By decoupling ingestion from execution (see `docs/PLAN-CONCURRENCY-RESILIENCE.md`), the `incidents` table acts as the job queue.
*   **Ingestion:** Writes `PENDING` rows.
*   **Workers:** Poll for `PENDING` rows, locking them by updating state to `RUNNING`.

## 3. Data Schema

The schema will be minimal, strictly supporting the current data flow.

### A. `fault_events`
Stores the raw trigger received from the MCP server.
*   `id` (PK): Auto-incrementing integer or UUID.
*   `received_at`: Timestamp.
*   `source_cluster`: String (e.g., "prod-us-east-1").
*   `mcp_payload`: JSONB/JSON. Stores the exact JSON-RPC message body received from the MCP server.
*   `fault_fingerprint`: String. The upstream `FaultID` (hash) provided in the event. *Note: We store this for future deduplication logic, but we do not enforce uniqueness constraints on it yet.*

### B. `incidents`
Represents the lifecycle of a triage attempt. Replaces the root-level directories in `incidents/`.
*   `id` (PK): UUID.
*   `event_id`: FK to `fault_events`.
*   `status`: String (Pending, Running, Completed, Failed). **(Indexed for Polling)**
*   `created_at`: Timestamp. **(Indexed for FIFO)**
*   `updated_at`: Timestamp.

### C. `agent_executions`
Tracks the specific run of the agent container.
*   `id` (PK): UUID.
*   `incident_id`: FK to `incidents`.
*   `agent_image`: String (e.g., `nightcrier-agent:latest`).
*   `exit_code`: Integer.
*   `started_at`: Timestamp.
*   `finished_at`: Timestamp.
*   `tokens_used`: Integer (Nullable). *Reserved for future implementation.*

### D. `triage_reports` (Artifact Pointer & Future Data)
Acts as the bridge between the SQL metadata and the heavy artifacts in blob/file storage.
*   `id`: PK.
*   `incident_id`: FK.
*   `raw_report_path`: String. The URI/path to the artifact (e.g., `/app/incidents/uuid/...` or `https://account.blob.core.windows.net/...`).
*   `storage_type`: String. Enum-like field (e.g., "filesystem", "azure_blob", "s3").
*   `structured_data`: JSONB. (Empty for now, reserved for future machine-readable output).
*   `summary_text`: Text.

## 4. Implementation Plan

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

## 5. Non-Goals
*   **Deduplication:** We will allow duplicate `fault_fingerprint` entries in `fault_events`. If the upstream sends it twice, we triage it twice.
*   **UI/Observability:** No new CLI commands for querying the DB will be added in this scope.
*   **Data Migration:** The `incidents/` directory remains as an archive, but the application will not read from it.
