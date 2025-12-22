# Research Plan: MCP Connection Resilience

## 1. Objective
Validate the behavior of the `modelcontextprotocol/go-sdk` (v1.1.0) regarding connection drops, reconnects, and session resumption.
**Goal:** Define the exact mechanism required to ensure `nightcrier` automatically reconnects to `kubernetes-mcp-server` and minimizes event loss after a disruption.

## 2. The "Safe Space" Environment
We will create a standalone research lab in `scratch/mcp-resilience-lab/`.
This isolated environment will verify the protocol mechanics without involving the complex `nightcrier` business logic or the heavyweight `kubernetes-mcp-server`.

### 2.1 Component Setup
*   **Directory:** `scratch/mcp-resilience-lab/`
*   **Mock Server (`cmd/server/main.go`):**
    *   Uses `modelcontextprotocol/go-sdk` to serve a simple SSE endpoint.
    *   Maintains an internal counter and emits a log event every 1 second.
    *   Prints "Client Connected / Disconnected" to stdout.
    *   *Hypothesis Check:* Does the SDK expose "Session ID" to the server handler?
*   **Minimal Client (`cmd/client/main.go`):**
    *   Uses `modelcontextprotocol/go-sdk` to connect to the mock server.
    *   Subscribes to notifications.
    *   Logs received events (ID and Timestamp).
    *   Implements the current `nightcrier` retry logic (naive reconnect).

## 3. Experiments

### Experiment A: The "Hard" Disconnect (TCP Sever)
**Scenario:**
1.  Server starts emitting events (1, 2, 3...).
2.  Client connects and receives (4, 5, 6...).
3.  **Action:** We kill the Server process (SIGKILL).
4.  Client should log an error.
5.  **Action:** We restart the Server process.
6.  **Observation:**
    *   Does the client automatically reconnect?
    *   Does the client need to re-issue the `notifications/subscribe` tool call?
    *   Does the server recognize the client as "returning" or "new"?

### Experiment B: The "Soft" Disconnect (Network/Proxy)
**Scenario:**
1.  Client connects to Server via a proxy (or we just close the TCP socket from the server side without killing the process).
2.  **Observation:**
    *   Does the standard SSE implementation in the SDK handle the `Last-Event-ID` header automatically?
    *   Does the SDK transparently resume the stream, or does it throw an error requiring a manual rebuild of the client?

### Experiment C: Session Persistence (The Critical Gap)
**Scenario:**
1.  Modify Mock Server to hold a buffer of the last 100 events.
2.  Connect Client. Receive event 10.
3.  Disconnect Client.
4.  Server emits 11, 12, 13.
5.  Reconnect Client.
6.  **Observation:**
    *   Can we send a "Resume Token" or `Last-Event-ID` during the handshake?
    *   Does the SDK API allow us to access this header?
    *   If yes, does the Mock Server re-emit 11, 12, 13?

## 4. Execution Steps
1.  **Scaffold:** Create the `scratch/mcp-resilience-lab` folder and `go.mod`.
2.  **Build Mock Server:** Implement the event emitter.
3.  **Build Minimal Client:** Implement the consumer.
4.  **Run Experiments:** Execute A, B, and C manually and log results.
5.  **Report:** Produce `docs/RESEARCH-RESULTS-MCP.md` with recommendations.

## 5. Expected Outcome
A definitive answer on whether `nightcrier` can achieve **Zero Data Loss** (via resumption) or if we must accept **Best Effort** (reconnect + gap) and design our downstream logic accordingly.
