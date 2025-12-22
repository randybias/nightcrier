# Plan: Increasing Test Coverage (Existing Codebase)

## 1. Executive Summary
**Objective:** Increase test coverage for critical components of `nightcrier` without refactoring the main loop yet.
**Strategy:** "Hand-rolled" mocks, in-process mock servers, and integration-tagged tests for Docker.
**Priorities:** `internal/events` (Ingestion) > `internal/cluster` (Connection Management) > `internal/storage` (Persistence).

## 2. Testing Strategy Per Component

### 2.1 `internal/events` (MCP Client)
*   **Current State:** 6.3% Coverage.
*   **Gap:** No verification of the handshake or subscription logic.
*   **Strategy: In-Process Mock Server**
    *   Create `client_test.go`.
    *   Use `net/http/httptest` to spin up a local server.
    *   Handler logic: Verify `Mcp-Session-Id` header, reply to `CallTool("events_subscribe")`, and stream SSE events.
    *   **Tests:**
        *   `TestClient_Connect_Success`: Verify handshake.
        *   `TestClient_Subscribe_Success`: Verify tool call arguments.
        *   `TestClient_ReceiveEvent`: Verify SSE -> Event parsing.
        *   `TestClient_Reconnect`: (Basic logic check).

### 2.2 `internal/cluster` (Connection Manager)
*   **Current State:** 0.0% Coverage.
*   **Gap:** The multi-cluster fan-in logic is completely untested.
*   **Strategy: Mocked Client**
    *   Define an interface `MCPClient` (if not already present) or use the struct with a swapped transport.
    *   Since `ConnectionManager` uses the concrete `events.Client` struct, we might need to introduce a tiny interface `EventSource` in `internal/cluster` to mock it.
    *   **Tests:**
        *   `TestManager_FanIn`: 3 clusters -> 1 channel. Verify source attribution.
        *   `TestManager_QueueOverflow`: Simulate slow consumer, verify `Drop` vs `Block` policy.

### 2.3 `internal/storage` (Persistence)
*   **Current State:** 37.2% Coverage.
*   **Gap:** Azure implementation is untested (requires creds).
*   **Strategy: Interface Mock**
    *   Ensure `Storage` interface in `internal/storage` is robust.
    *   Create `mock_storage.go` (Hand-rolled mock) for use by other packages.
    *   **Unit Tests:**
        *   `TestFilesystem_Save`: Verify permission bits (`0600`) and directory creation.
        *   `TestFilesystem_Sanitization`: Verify path traversal attempts are blocked (e.g., incident ID `../../etc/passwd`).

### 2.4 `internal/agent` (Executor)
*   **Current State:** 51.9% Coverage.
*   **Strategy: Build Tags**
    *   Create `executor_integration_test.go` with `//go:build integration`.
    *   This test will assume Docker is present and run `hello-world`.
    *   This ensures the *actual* Docker interaction works, but keeps unit tests fast.

## 3. Negative Testing Scenarios (Integration Level)
*   **Bad Config:** Test `NewClient` with invalid URLs (`htpt://...`).
*   **Broken Server:** Start mock server, kill it mid-stream, verify `Client` error channel behavior.
*   **Permissions:** Test `FilesystemStorage` with a read-only directory.

## 4. Execution Roadmap
1.  **Events:** Implement `TestClient_...` suite with `httptest`.
2.  **Cluster:** Refactor `ConnectionManager` to accept an interface for `Client` (minimal change), then test fan-in.
3.  **Storage:** Add path traversal/permissions tests for Filesystem.
4.  **Integration:** Add the Docker test with build tags.

## 5. Note on Orchestration
Per user instruction, the refactor of `processEvent` into `TriageOrchestrator` is deferred. We will focus on unit testing the *components* that `processEvent` calls, rather than the main loop itself.
