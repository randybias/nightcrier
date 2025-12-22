# Design Proposal: Stateless Message-Based Transport for MCP

## 1. Executive Summary
**Objective:** Decouple `nightcrier` (Client) from `kubernetes-mcp-server` (Server) using a stateless, message-based transport.
**Driver:** Overcome the limitations of the current "Connection = Session" model (SSE/Stdio), enabling robust resume, horizontal scaling, and resilience against network blips.
**Core Concept:** Shift from *Persistent Connections* to *Logical Sessions* over a Message Bus.

## 2. Transport Protocol Analysis

We evaluated four candidates for the transport layer.

### Option A: NATS / JetStream (Recommended)
*   **Architecture:** Centralized (but lightweight) Broker.
*   **Pros:**
    *   **Native Request-Reply:** NATS has a built-in "Inbox" pattern (`nats.Request()`) that maps 1:1 with JSON-RPC calls.
    *   **Subject-Based Routing:** Easy to route `mcp.global.events` vs `mcp.session.123.tools`.
    *   **K8s Native:** Standard in cloud-native stacks; easy to run as a sidecar or cluster service.
*   **Cons:** Introduces a dependency (NATS Server).

### Option B: ZeroMQ / nanomsg
*   **Architecture:** Brokerless (Peer-to-Peer).
*   **Pros:** Ultra-fast, no central point of failure.
*   **Cons:**
    *   **Discovery Hell:** In K8s, dynamic pods need to find each other. ZMQ requires stable endpoints or a complex discovery side channel.
    *   **TCP State:** Still fundamentally relies on underlying TCP sockets for `REQ/REP` sockets.

### Option C: AMQP (RabbitMQ)
*   **Architecture:** Heavy Centralized Broker.
*   **Pros:** Durable queues (guaranteed delivery).
*   **Cons:** Overkill. We need *RPC speed* and *Event Streaming*, not necessarily durable transaction logs for every tool call. High operational overhead.

### Option D: UDP
*   **Architecture:** Datagrams.
*   **Pros:** True "Stateless" fire-and-forget.
*   **Cons:**
    *   **Reliability:** We would have to reimplement TCP (ordering, retries, chunking for large JSON payloads).
    *   **MTU Issues:** JSON-RPC payloads (LLM Context) can easily exceed 65KB, let alone 1500 bytes.
    *   **Verdict:** **Unsuitable** for the Control Plane (Tools/Sampling).

## 3. The Winner: NATS

We propose using **NATS Core** (not JetStream initially) to implement the custom transport. It aligns perfectly with the "Stateless" goal because the NATS client handles the connection management, and the application just publishes/subscribes.

### 3.5 Alternative: SSE-over-NATS (Lightweight)
*Reference: User Design "SSE over Plain NATS"*

**Concept:** Treat SSE as the **payload format**, not the transport.
Instead of rewriting the entire JSON-RPC layer to map to NATS Request-Reply immediately, we can simply tunnel the existing SSE text stream over NATS subjects.

**Flow:**
1.  **Server:** Generates standard SSE text (`event: message\ndata: {...}`).
2.  **Server:** Publishes this string to `mcp.session.<id>.out`.
3.  **Client:** Subscribes to `mcp.session.<id>.out` and parses the payload exactly as if it came from an HTTP stream.

**Pros:**
*   **Minimal Server Change:** The server already generates SSE text. We just redirect the `io.Writer` from the HTTP Response to a NATS Publisher.
*   **Decoupled:** NATS handles the fan-out and buffering.
*   **Semantics:** Matches the "Best Effort / At-Most-Once" nature of the current system.

**The "Server Restart" Hole:**
Even with NATS, if the Server process restarts, it loses its internal `Subscription` map and stops publishing.
*   *Scenario:* Client is listening on NATS. Server restarts. Connection to NATS Broker restores, but Server isn't publishing.
*   *Solution:* **Timeout-Based Detection (Dead Man's Switch)**.
    *   **Server:** Already generates SSE keepalives (`: keepalive`) periodically (standard behavior). It publishes these to the NATS subject.
    *   **Client:** Sets a `ReadDeadline` (e.g., 2x the keepalive interval).
    *   **Logic:** If the Client receives *silence* for too long, it assumes the Server logic has died or reset.
    *   **Recovery:** Client re-runs the HTTP `events_subscribe` tool call to tell the (new) Server to start publishing again.

## 4. Proposed Implementation: "MCP over NATS"

### 4.1 Topic Topology
Instead of a URL, clients connect to a NATS Subject hierarchy.

*   **Discovery/Handshake:** `mcp.discovery` (Queue Group: Load Balanced)
*   **Session Channel (Client -> Server):** `mcp.session.<session_id>.in`
*   **Session Channel (Server -> Client):** `mcp.session.<session_id>.out`
*   **Broadcast Events:** `mcp.events.<cluster>.<type>`

### 4.2 The "Logical Session" Handshake
1.  **Client Start:** Client generates a `SessionID` (UUID) locally (or requests one).
2.  **Connect:** Client subscribes to `mcp.session.<uuid>.out`.
3.  **Initialize:** Client publishes `JSON-RPC Initialize` to `mcp.discovery` with `reply_to: mcp.session.<uuid>.out`.
4.  **Server Response:** Server sees the message, registers the Logical Session, and replies to the Inbox.
5.  **Steady State:**
    *   **Tools:** Client publishes `CallTool` to `mcp.session.<uuid>.in`.
    *   **Events:** Server publishes `Notifications` (formatted as SSE text) to `mcp.session.<uuid>.out`.

### 4.3 Handling "Reconnects" (The Magic)
If `nightcrier` restarts:
1.  It loads its persisted `SessionID` from the DB (see `PLAN-STATE-MIGRATION`).
2.  It re-subscribes to `mcp.session.<uuid>.out`.
3.  It publishes a `Ping` or `Re-Attach` message.
4.  **The Server never knew we left.** The NATS broker buffered the messages (if JetStream) or simply dropped them, but the *Server Logic* didn't trash the session state just because a TCP socket closed.

## 5. Implementation Roadmap

### Phase 1: The Lab (`mcp-resilience-lab`)
1.  Spin up `nats-server` (Docker).
2.  Build `client-nats` and `server-nats`.
3.  Implement `mcp.Transport` interface for NATS.

### Phase 2: Nightcrier Integration
1.  Add `nats` dependency.
2.  Update `ConnectionManager` to support `nats://` URLs.

### Phase 3: Server Integration
1.  Fork/Update `kubernetes-mcp-server` to listen on NATS.

## 6. Pros/Cons for Nightcrier
*   **Pro:** Solves the "Ingress Gap" problem. If we use JetStream for the *Event* topics, we get **Durable Consumers** for free (Resumption!).
*   **Con:** Requires deploying NATS in the K8s cluster (very common pattern, but a dependency nonetheless).
