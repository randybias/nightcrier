# Codebase Analysis Report
**Date:** 2025-12-22
**Scope:** Architecture, Quality, and Roadmap Alignment

## 1. Executive Summary
The `nightcrier` codebase is currently a functional **synchronous CLI prototype**. It successfully implements the core logic of listening to MCP events, executing a Docker-based agent, and reporting results. However, it is structurally unprepared for high-volume production use due to its blocking event loop and filesystem dependency.

The codebase is clean, idiomatic Go, but strictly follows a "script-like" execution flow in `main.go`. To achieve the goals outlined in `PLAN-STATE-MIGRATION.md` and `PLAN-CONCURRENCY-RESILIENCE.md`, significant architectural changes are required.

## 2. Architecture Analysis

### 2.1 The Blocking Event Loop
The critical bottleneck is in `cmd/nightcrier/main.go`.
*   **Current Behavior:** The `select` loop receives an event from `eventChan` and immediately calls `processEvent`.
*   **The Problem:** `processEvent` is synchronous and includes the agent execution time (potentially minutes).
*   **Impact:** While one incident is being triaged, all other incoming events are blocked. If the channel buffer fills, the upstream listener may drop events.
*   **Alignment:** This confirms the urgent need for the "Producer-Consumer" model described in the Concurrency Plan.

### 2.2 The `processEvent` Monolith
The `processEvent` function (~200 lines) is a "God Function" that knows too much.
*   **Responsibilities:**
    1.  Domain Object Creation (`incident.NewFromEvent`)
    2.  Permission Validation (`permissions.MinimumPermissionsMet`)
    3.  IO Operations (`workspaceMgr.Create`, writing JSONs)
    4.  Core Business Logic (`executor.Execute`)
    5.  Error Handling & Circuit Breaking
    6.  Artifact Storage (`storageBackend.SaveIncident`)
    7.  Notifications (`slackNotifier`)
*   **Refactoring Risk:** Moving this logic "as-is" into a Worker will result in a complex, untestable Worker.
*   **Recommendation:** Break this down into an `Orchestrator` or distinct services (IncidentService, AgentService) as per `PLAN-REFACTOR-ORCHESTRATION.md` *before* or *during* the worker implementation.

### 2.3 Storage Abstraction Mismatch
There is a semantic gap between the *current* Storage interface and the *planned* State Store.
*   **Current (`internal/storage`):** `SaveIncident(ctx, id, artifacts)`. This is an **Archive** interface (Blob Storage).
*   **Planned (`StateStore`):** `CreateIncident`, `UpdateStatus`. This is a **Transactional** interface (SQL DB).
*   **Gap:** The current interface does not support the "Job Queue" pattern needed for Phase 2. We need a new interface (e.g., `Repository`) or a major expansion of the existing one.

## 3. Code Quality & Standards

*   **Go Idioms:** Excellent. Context propagation is consistent (`ctx`), error wrapping is standard (`fmt.Errorf("...: %w", err)`), and structured logging (`slog`) is used effectively throughout.
*   **Configuration:** The use of `cobra` and `viper` (implied by `config.BindFlags`) follows standard CLI patterns.
*   **Modularity:** Packages like `internal/cluster` and `internal/agent` have clear boundaries, but `main` violates this by gluing them too tightly.
*   **Safety:** The code handles signals (`os.Interrupt`) and defer cancellations correctly.

## 4. Gap Analysis (vs. Plans)

| Component | Plan | Current State | Gap |
| :--- | :--- | :--- | :--- |
| **State** | SQL Database (SQLite/Postgres) | Filesystem (`incident.json`) | **100%**. No SQL drivers or schema present. |
| **Concurrency** | Async Worker Pool | Synchronous Loop | **100%**. Single-threaded execution. |
| **Resilience** | DB-based Queue | In-Memory Channel | **High**. Events lost on restart. |
| **Testing** | Integration Suites | Unit Tests | **Medium**. `executor` has tests, but the main flow is untested. |

## 5. Strategic Recommendations

1.  **Execute `PLAN-STATE-MIGRATION` First:**
    *   Do not try to build the Worker Pool yet. The Worker needs a persistent Queue (the DB).
    *   Implement `internal/storage/repository.go` (the SQL interface) alongside the current `storage.go` (the Artifact interface).

2.  **Refactor `processEvent`:**
    *   Extract the logic into `internal/orchestrator` as a prerequisite for the Worker.
    *   This will make it easier to wrap the logic in a `go routine` later.

3.  **Accept Data Loss (for now):**
    *   As found in `RESEARCH-RESULTS-MCP.md`, upstream limitations prevent zero-data-loss resumption. Focus on *internal* resilience (not crashing on bad inputs) rather than *ingress* resilience (replaying missed events).
