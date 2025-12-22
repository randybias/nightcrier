# Research Results: MCP Resilience

## 1. Executive Summary
We investigated the resilience of the MCP connection between `nightcrier` (client) and `kubernetes-mcp-server` (server).
**Key Finding:** The current architecture supports **Best Effort** delivery only. Zero Data Loss (resumption of missed events) is **not possible** without modifications to the upstream server.

## 2. Experiments & Findings

### Experiment A: Hard Disconnect (Server Kill)
*   **Behavior:** When the server dies, the client detects the connection drop (TCP/HTTP error).
*   **Result:** The `go-sdk` attempts to reconnect using the *same* Session ID.
*   **Failure:** The restarted server rejects this Session ID ("session not found") because it holds session state in memory only.
*   **Recovery:** The client must catch this error, create a *new* MCP Client (handshake -> new session), and re-subscribe.

### Experiment B: Client Restart
*   **Behavior:** When `nightcrier` restarts, it creates a new session.
*   **Result:** The server accepts the new session.
*   **Data Gap:** The server's `events_subscribe` tool **intentionally** starts watching from "Now" (Current ResourceVersion). It explicitly filters out historical events.
*   **Constraint:** The `events_subscribe` tool schema does **not** accept a `since` or `resourceVersion` parameter.

## 3. Implications for Nightcrier

### 3.1 No "Cursor" Persistence
There is no value in `nightcrier` persisting a "Last Received Event ID" or "Timestamp" to the database for resumption purposes, because the upstream server provides no API to use it.

### 3.2 Robust Reconnect Loop
The `ConnectionManager` must be improved to handle the "Session Not Found" error specifically.
*   *Current:* Simple retry.
*   *Required:* If `Connect()` or `Wait()` returns a session error, strictly teardown and re-initialize the entire client to ensure a fresh handshake.

### 3.3 Data Loss Acceptance
We must document that `nightcrier` is a **Live Triage** system, not an Audit system.
*   If the connection drops for 10 seconds, faults occurring in those 10 seconds are missed.
*   This is acceptable for "Incident Triage" (if the fault persists, it will likely be re-emitted by the K8s controller loop anyway, or `nightcrier` will catch the next one).

## 4. Next Steps
1.  **Approve:** Accept these constraints.
2.  **Execute:** Proceed with the "State Migration" and "Concurrency" plans, knowing they improve *internal* reliability (processing queue) but do not solve *ingress* gaps.
