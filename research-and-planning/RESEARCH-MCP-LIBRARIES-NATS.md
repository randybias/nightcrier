# Research: MCP Libraries & Custom NATS Transport

## 1. Objective
Evaluate the ecosystem of Go MCP libraries to determine:
1.  Feature completeness and support for the standard.
2.  Feasibility of implementing a custom **NATS Transport** to enable stateless/resilient sessions.

## 2. Library Analysis

### 2.1 Reference SDK (`github.com/modelcontextprotocol/go-sdk`)
*   **Role:** The official reference implementation.
*   **Current Usage:** Used by `nightcrier`.
*   **Transport Abstraction:** `Stream` Oriented.
    *   Interface: `Read()` / `Write()` of `jsonrpc.Message`.
    *   **Implication:** The SDK manages the JSON-RPC state machine (correlation of Request IDs to pending calls).
    *   **Fit for NATS:** **Asynchronous Pattern**.
        *   We cannot use `nats.Request()` (blocking) directly inside `Write()`.
        *   Instead, we must use `nats.Publish()` for `Write()` and a background goroutine subscribing to the NATS Inbox to feed `Read()`.
    *   **Pros:** Flexible, unopinionated, standard.
    *   **Cons:** Requires slightly more boilerplate to bridge the NATS subscription to the `Read()` channel.

### 2.2 Mark3Labs SDK (`github.com/mark3labs/mcp-go`)
*   **Role:** A popular, feature-rich community alternative.
*   **Transport Abstraction:** `Request/Reply` Oriented.
    *   Interface: `SendRequest()` (Synchronous blocking).
    *   **Implication:** The Transport implementation is responsible for waiting for the response.
    *   **Fit for NATS:** **Synchronous Pattern**.
        *   Maps 1:1 with `nats.Request()`.
    *   **Pros:** Simpler implementation for Request-Reply protocols like NATS.
    *   **Cons:** Opinionated; switching from the Reference SDK would be a major refactor.

## 3. Recommended Implementation Strategy (Reference SDK)

Since `nightcrier` already relies on the Reference SDK, we should stick with it. The implementation of a NATS Transport would look like this:

### 3.1 `NatsTransport` Struct
```go
type NatsTransport struct {
    NC           *nats.Conn
    Subject      string
    ReplySubject string // "Inbox"
    Sub          *nats.Subscription
    MsgChan      chan jsonrpc.Message
}
```

### 3.2 Methods
*   **`Connect(ctx)`**:
    *   Subscribe to `ReplySubject` (Inbox).
    *   Start a goroutine: `for msg := range sub.Channel { Unmarshal; MsgChan <- msg }`.
*   **`Write(ctx, msg)`**:
    *   Marshal `msg` to JSON.
    *   `nats.PublishMsg({Subject: Target, Reply: ReplySubject, Data: json})`.
*   **`Read(ctx)`**:
    *   `select { case msg := <-MsgChan: return msg ... }`.

## 4. Hybrid Transport Feasibility (HTTP + NATS)
We evaluated splitting traffic (e.g., Tools via HTTP, Events via NATS) for a single session.
*   **Finding:** The `go-sdk` `ClientSession` binds 1:1 to a `Connection`.
*   **Constraint:** To support hybrid, we would need a "Proxy Connection" that routes messages.
*   **Blocker:** The Server must recognize the same "Session ID" across both transports. Currently, the upstream server treats the HTTP connection and any potential NATS connection as distinct sessions with isolated state.
*   **Conclusion:** Hybrid is infeasible without major Server-side re-architecture (Shared Session Store). We should pursue a **Pure NATS** transport instead.

## 5. Conclusion
*   **Feasibility:** High (for Pure NATS).
*   **Path:** Implement `mcp.Transport` using the Asynchronous NATS pattern.
*   **Status:** Documented for future work. No immediate action required.