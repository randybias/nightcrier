# Plan: Concurrency, Resilience & Logging

## 1. Executive Summary
**Objective:** Decouple event ingestion from agent execution to prevent "event storms" from overwhelming the system.
**Current State:** A synchronous loop where 1 agent running blocks all incoming events.
**Target State:** An asynchronous "Producer-Consumer" architecture using the SQL database as the buffer.

## 2. Architecture: Producer-Consumer

### 2.1 The Producer (Ingestion Loop)
*   **Role:** Listen to MCP events -> Write to DB (`PENDING`) -> Ack/Drop -> Repeat.
*   **Performance:** Must be blazing fast (ms). Never waits for an agent.
*   **Resilience:** If the DB is down, it buffers in memory (channel) or drops with a metric/log.
*   **Panic Safety:** Wrap the `runConnection` and main loop in `recover()` to log panics without crashing the process.

### 2.2 The Consumer (Worker Pool)
*   **Role:** Poll DB for `status=PENDING` -> Claim Job -> Run Agent -> Update DB.
*   **Concurrency Control:**
    *   **Global Limit:** `MaxConcurrentAgents` (e.g., 5 total).
    *   **Per-Cluster Limit:** Enforce **"One Agent Per Cluster"** rule via a locking mechanism (in-memory `Mutex` map or DB-based advisory lock). This prevents 5 agents from trying to debug the same cluster simultaneously.

## 3. Logging & Debugging Strategy
Current Issue: "Noisy" logs when multiple agents run.

### 3.1 Structured Isolation
*   **System Logs (Stdout):** Minimal, high-signal. "Incident X created", "Worker Y started Incident X", "Worker Y finished (Success/Fail)". No raw agent output here.
*   **Agent Logs (Files):** We already redirect agent stdout/stderr to files in the incident workspace. We will continue this strictly.
*   **Live Tailing:** Add a CLI command `nightcrier tail <incident-id>` to stream the logs of a specific running agent if needed, rather than dumping all to main stdout.

## 4. Reconnects & Session Resilience (Updated based on Research)
*   **Finding:** The upstream `kubernetes-mcp-server` does not support session resumption or historical backfill (`since` parameter).
*   **Strategy: Best Effort / Fresh Start**
    *   **On Disconnect:** The `ConnectionManager` must treat any connection failure as fatal to the session.
    *   **Action:** Tear down the existing MCP client completely.
    *   **Recovery:** Create a fresh Client -> Handshake (New Session) -> Call `events_subscribe` again.
    *   **Data Impact:** Events occurring during the disconnected window are **lost**. This is accepted behavior for a live triage system (persistent faults will likely re-trigger).

## 5. Implementation Phases

### Phase 1: The Decoupling (Requires DB Migration First)
*   Modify `processEvent` to **only** insert a row into `incidents` and `fault_events`.
*   Remove the immediate `executor.Execute` call.

### Phase 2: The Worker
*   Create `internal/worker/pool.go`.
*   Implement a loop that ticks every X seconds (or listens to a signal).
*   Query: `SELECT * FROM incidents WHERE status='PENDING' ORDER BY created_at ASC`.
*   Logic:
    *   Check: Is `ActiveAgents[cluster_name]` > 0? If yes, skip.
    *   Else: Launch Goroutine, increment counter.

### Phase 3: Panic Recovery
*   Add `defer func() { if r := recover(); r != nil { ... } }()` to the worker goroutine.
*   If a worker panics, update incident status to `FAILED (Panic)` so it doesn't get stuck in `running` forever.

## 6. Loose Coupling Verification
*   If the "Worker" crashes/panics, the "Ingestor" keeps writing events to the DB.
*   If the "Ingestor" loses connection, the "Worker" finishes its current jobs.
*   **Recovery:** On restart, the system picks up all `PENDING` jobs from the DB.

