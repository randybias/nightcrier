# Meta-Plan: Nightcrier Evolution

## 1. Overview
This document sequences the execution of three major work streams: **State Migration**, **Concurrency/Resilience**, and **Test Coverage**.
*   **Goal:** Transform `nightcrier` from a synchronous, file-based CLI tool into a resilient, concurrent, stateful triage service.
*   **Key Constraint:** "1 Event = 1 Incident". No deduplication logic changes yet.

## 2. Phase 1: The Data Foundation (State Migration)
*Reference: `docs/PLAN-STATE-MIGRATION.md`*

**Objective:** Move state from "Filesystem folders" to "SQL Database".
**Why First?** The database table `incidents` is required to act as the **Job Queue** for Phase 2.

1.  **Storage Abstraction:** Create `internal/storage/repository.go` (Interface).
2.  **SQL Adapters:** Implement `sqlite` (and optionally `postgres`) adapters.
3.  **Schema Implementation:** Create `incidents`, `fault_events`, `agent_executions`, `triage_reports` tables.
4.  **Wiring:** Update `main.go` to initialize the DB.
    *   *Result:* Events are written to SQLite immediately upon receipt.

## 3. Phase 2: The Async Engine (Concurrency & Resilience)
*Reference: `docs/PLAN-CONCURRENCY-RESILIENCE.md`, `docs/RESEARCH-RESULTS-MCP.md`*

**Objective:** Decouple Ingestion from Execution. Stop "Event Storms" from blocking the system.
**Why Second?** Requires the DB from Phase 1 to buffer the events.

1.  **The "Producer" (Ingestion):**
    *   Modify `processEvent` to **STOP** running the agent.
    *   It should only: `DB.CreateIncident(status=PENDING)` -> Return.
    *   *Outcome:* Ingestion is now fast (ms).
2.  **The "Consumer" (Worker Pool):**
    *   Create `internal/worker`.
    *   Implement the Polling Loop (`SELECT ... WHERE status=PENDING`).
    *   Implement "One Agent Per Cluster" locking (using DB or memory).
    *   Move the "Run Agent" logic from `main` into this Worker.
3.  **Resilience (Best Effort):**
    *   Implement the "Fresh Reconnect" logic in `ConnectionManager` (tear down client on error).

## 4. Phase 3: Quality Assurance (Test Coverage)
*Reference: `docs/PLAN-TEST-COVERAGE.md`*

**Objective:** Verify the new engine and backfill tests for the old parts.
**Why Third?** Best to write tests for the *new* architecture (Worker/DB) than the old synchronous one.

1.  **Events:** Write `TestClient` using `httptest` (Mock Server).
2.  **Cluster:** Test the `ConnectionManager` fan-in logic.
3.  **Worker:** Test the new Polling Loop logic (using a Mock DB).
4.  **Integration:** Add `//go:build integration` tests for Docker execution.

## 5. Deferred / Future Work
*   **Orchestration Refactor:** (`docs/PLAN-REFACTOR-ORCHESTRATION.md`) - Clean up `main.go` later.
*   **NATS Transport:** (`docs/PROPOSAL-MCP-CUSTOM-TRANSPORT.md`) - Long-term goal for stateless sessions.

## 6. Execution Checklist (Immediate Next Steps)
1.  [ ] **State:** Define `internal/storage/repository.go` interfaces.
2.  [ ] **State:** Implement SQLite Driver.
3.  [ ] **State:** Wire up DB in `main.go`.
