# Design: Event Runner Core

## Context
The Event Runner is a standalone process that connects the `kubernetes-mcp-server` (source of truth) with AI agents (triage workers). It is designed as a prototype to validate the value of AI-led triage for Kubernetes faults.

## Goals
- **Automated Triage**: Automatically investigate high-severity faults.
- **Safety**: Ensure agents are strictly read-only and do not disrupt the cluster.
- **Resilience**: Prevent agent storms via strict rate limiting and circuit breaking.
- **Observability**: Provide clear reports and notifications to Ops teams.

## Non-Goals
- Active remediation (auto-healing).
- Multi-cloud or non-Kubernetes support (initial phase).
- Complex ticketing system integration.

## Decisions

### Architecture
- **Language**: Go, aligning with `kubernetes-mcp-server`.
- **Event Source**: SSE (Server-Sent Events) from `kubernetes-mcp-server`. This decouples the runner from the server's internal implementation.
- **Agent Invocation**: Executing the agent as a subprocess (CLI wrapper). This allows swapping agent implementations (Claude, etc.) by changing the command/script without recompiling the runner.

### SSE Client Library Selection
After evaluating available Go SSE client libraries, we will use **r3labs/sse** for the following reasons:
- **Automatic Reconnection**: Built-in reconnection with backoff strategy using cenkalti/backoff library.
- **Last-Event-ID Tracking**: Automatically tracks and sends Last-Event-ID header for resumable connections.
- **Context Support**: Native support for context.Context for cancellation and timeouts.
- **Active Maintenance**: Recent activity and well-documented API.
- **Customization**: Supports custom headers, retry strategies, and response validators.

Alternative considered: tmaxmax/go-sse was evaluated but r3labs/sse has better client-side features and more mature reconnection handling.

### Rate Limiting & Concurrency
- **Per-Cluster**: Strict Mutex/Queue. Only 1 agent active per cluster at a time. This prevents race conditions and overwhelming the cluster API.
- **Global Circuit Breaker**: Hard limit on total concurrent agents across all clusters (default: 5, configurable).
- **Deduplication**: Time-window-based suppression (5 minutes default) for the same Resource+Namespace combination using an in-memory cache with expiry.

### Concurrency Implementation Strategy
Based on Go concurrency best practices:
- **Buffered Channels as Queues**: Use buffered channels for event queuing to decouple event intake from processing.
- **Semaphore Pattern**: Implement global circuit breaker using a buffered channel as a counting semaphore.
- **Per-Cluster Mutexes**: Maintain a map of cluster IDs to mutexes (or single-element buffered channels) for per-cluster concurrency control.
- **Worker Pool**: Dedicated goroutine per cluster that processes events from the cluster-specific queue.
- **Context for Cancellation**: Use context.Context throughout for graceful shutdown and timeout handling.

### Data Structures

#### Event Queue Architecture
```
SSE Stream → Event Validator → Severity Filter → Dedup Cache → Routing Layer
                                                                      ↓
                ┌────────────────────────────────────────────────────┴──────────┐
                ↓                                ↓                               ↓
        Global Semaphore              Global Semaphore                  Global Semaphore
        (Circuit Breaker)             (Circuit Breaker)                 (Circuit Breaker)
                ↓                                ↓                               ↓
        Cluster-A Queue               Cluster-B Queue                   Cluster-N Queue
        (buffered chan)               (buffered chan)                   (buffered chan)
                ↓                                ↓                               ↓
        Cluster-A Worker              Cluster-B Worker                  Cluster-N Worker
        (1 goroutine)                 (1 goroutine)                     (1 goroutine)
                ↓                                ↓                               ↓
        Agent Process                 Agent Process                     Agent Process
```

#### Core Data Structures
```go
// FaultEvent represents a parsed SSE event
type FaultEvent struct {
    EventID      string    // Unique event identifier from SSE
    ClusterID    string    // Kubernetes cluster identifier
    Namespace    string    // Resource namespace
    ResourceType string    // e.g., "Pod", "Deployment"
    ResourceName string    // Resource name
    Severity     string    // DEBUG, INFO, WARNING, ERROR, CRITICAL
    Timestamp    time.Time // Event timestamp
    Message      string    // Fault description
    RawData      string    // Original SSE data for debugging
}

// DeduplicationKey uniquely identifies a resource for dedup
type DeduplicationKey struct {
    ClusterID    string
    Namespace    string
    ResourceType string
    ResourceName string
}

// CircuitBreaker manages global concurrency
type CircuitBreaker struct {
    semaphore chan struct{} // Buffered channel as counting semaphore
    limit     int            // Maximum concurrent agents
}

// ClusterQueue manages per-cluster event processing
type ClusterQueue struct {
    clusterID string
    queue     chan *FaultEvent // Buffered channel for events
    mutex     sync.Mutex       // Ensures single worker per cluster
}
```

### SSE Reconnection Strategy

#### Backoff Configuration
- **Initial Backoff**: 1 second
- **Maximum Backoff**: 60 seconds
- **Backoff Multiplier**: 2.0 (exponential)
- **Jitter**: 10% randomization to avoid thundering herd
- **Max Attempts**: Unlimited (continues until context cancellation)

#### Connection State Machine
```
[DISCONNECTED] → (Connect) → [CONNECTING] → (Success) → [CONNECTED]
                                  ↓                           ↓
                              (Failure)                   (Error/EOF)
                                  ↓                           ↓
                            [BACKOFF] ← ← ← ← ← ← ← ← ← [RECONNECTING]
                                  ↓
                            (Wait + Retry)
```

#### Last-Event-ID Tracking
- Store the last successfully processed event ID in memory
- On reconnection, send "Last-Event-ID" header to resume stream
- Server should replay events after the specified ID
- If server does not support resumption, start from current events

#### Heartbeat Handling
- Server may send comment lines (`:` prefix) as heartbeats
- Client must not timeout on heartbeat-only connections
- Implement read timeout (default: 120 seconds) to detect silent failures
- Reset timeout on any data received (event or comment)

### Data Flow

#### Detailed Event Processing Pipeline
1. **Intake**: SSE client receives event from `kubernetes-mcp-server`
   - Parse event data as JSON
   - Extract event ID and store for reconnection
   - Log reception with timestamp

2. **Validation**: Check event structure
   - Validate JSON schema
   - Ensure required fields present (cluster_id, severity, resource_name)
   - Log and discard malformed events
   - Increment error metrics

3. **Filter**: Apply severity threshold
   - Compare event severity against configured threshold
   - Use ordering: DEBUG < INFO < WARNING < ERROR < CRITICAL
   - Log filtered events at debug level
   - Continue to next event if filtered

4. **Deduplication**: Check against recent events
   - Generate dedup key from cluster_id, namespace, resource_type, resource_name
   - Check in-memory cache with time-based expiry
   - If duplicate within window, discard and log
   - If new or expired, add to cache and continue

5. **Global Circuit Breaker**: Check total capacity
   - Attempt to acquire semaphore (non-blocking)
   - If available, proceed to routing
   - If full, enqueue to global overflow queue or drop based on policy
   - Log circuit breaker state

6. **Routing**: Send to cluster-specific queue
   - Look up or create cluster queue
   - Send event to cluster's buffered channel
   - If cluster queue full, apply overflow policy (drop oldest or reject)
   - Log queuing action with queue depth

7. **Execution**: Cluster worker processes event
   - Worker goroutine reads from cluster queue (blocks when empty)
   - Spawn agent subprocess with event context
   - Monitor agent execution
   - Collect artifacts from workspace

8. **Cleanup**: Release resources
   - Agent completes or times out
   - Release global semaphore slot
   - Update dedup cache if needed
   - Log completion with duration and status

9. **Notification**: Send summary to Slack (Phase 3 - not in this change)

### Error Handling Strategy

#### SSE Connection Errors
- **Temporary Errors** (network timeout, connection refused): Retry with backoff
- **Permanent Errors** (401 unauthorized, 404 not found): Log critical error, exit process
- **Parse Errors**: Log error, discard event, continue processing
- **Rate Limit Errors** (429): Honor Retry-After header if present, otherwise use backoff

#### Agent Execution Errors
- **Spawn Failure**: Log error, release semaphore, continue with next event
- **Timeout**: Kill agent process, log timeout, release semaphore
- **Exit with Error**: Log exit code and stderr, release semaphore
- **Panic Recovery**: Defer/recover in worker goroutines, log panic, restart worker

#### Queue Overflow Errors
- **Global Queue Full**: Apply configured policy (drop oldest or reject new)
- **Cluster Queue Full**: Apply configured policy (drop oldest or reject new)
- **Always Log**: Include event details, queue depths, and policy applied

### Configuration Schema

#### Environment Variables
```
SSE_ENDPOINT                    # Required: SSE server URL
SEVERITY_THRESHOLD              # Default: ERROR
MAX_CONCURRENT_AGENTS           # Default: 5
GLOBAL_QUEUE_SIZE               # Default: 100
CLUSTER_QUEUE_SIZE              # Default: 10
DEDUP_WINDOW_SECONDS            # Default: 300 (5 minutes)
QUEUE_OVERFLOW_POLICY           # Default: drop (options: drop, reject)
AGENT_TIMEOUT_SECONDS           # Default: 600 (10 minutes)
SHUTDOWN_TIMEOUT_SECONDS        # Default: 30
SSE_RECONNECT_INITIAL_BACKOFF   # Default: 1s
SSE_RECONNECT_MAX_BACKOFF       # Default: 60s
SSE_READ_TIMEOUT_SECONDS        # Default: 120
LOG_LEVEL                       # Default: info (options: debug, info, warn, error)
```

#### Command-Line Flags
All environment variables can be overridden via flags:
```
--sse-endpoint string
--severity-threshold string
--max-concurrent-agents int
--global-queue-size int
--cluster-queue-size int
--dedup-window duration
--queue-overflow-policy string
--agent-timeout duration
--shutdown-timeout duration
--log-level string
```

#### Configuration File (Optional)
YAML format, loaded from `--config-file` flag:
```yaml
sse:
  endpoint: "https://k8s-mcp-server:8080/events"
  reconnect:
    initial_backoff: "1s"
    max_backoff: "60s"
  read_timeout: "120s"

filtering:
  severity_threshold: "ERROR"

concurrency:
  max_concurrent_agents: 5
  global_queue_size: 100
  cluster_queue_size: 10

deduplication:
  window: "5m"

queues:
  overflow_policy: "drop"

agent:
  timeout: "10m"

shutdown:
  timeout: "30s"

logging:
  level: "info"
```

## Component Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         Event Runner                            │
│                                                                 │
│  ┌───────────────┐                                             │
│  │  SSE Client   │ ← (Last-Event-ID, reconnect, backoff)       │
│  │  (r3labs/sse) │                                             │
│  └───────┬───────┘                                             │
│          │ events                                              │
│          ↓                                                      │
│  ┌───────────────┐     ┌──────────────┐                       │
│  │   Validator   │ →   │    Logger    │                       │
│  └───────┬───────┘     └──────────────┘                       │
│          │ validated events                                    │
│          ↓                                                      │
│  ┌───────────────┐                                             │
│  │Severity Filter│                                             │
│  └───────┬───────┘                                             │
│          │ filtered events                                     │
│          ↓                                                      │
│  ┌───────────────┐                                             │
│  │  Dedup Cache  │ (in-memory with TTL)                       │
│  └───────┬───────┘                                             │
│          │ unique events                                       │
│          ↓                                                      │
│  ┌─────────────────────────────────────┐                      │
│  │    Global Circuit Breaker           │                      │
│  │  (semaphore: chan struct{})         │                      │
│  │  Capacity: configurable (default 5) │                      │
│  └──────────────┬──────────────────────┘                      │
│                 │ events with semaphore slot                   │
│                 ↓                                               │
│  ┌─────────────────────────────────────┐                      │
│  │         Event Router                │                      │
│  │  (map[clusterID]*ClusterQueue)      │                      │
│  └──┬────────────┬────────────┬────────┘                      │
│     │            │            │                                │
│     ↓            ↓            ↓                                │
│  ┌─────┐     ┌─────┐     ┌─────┐                             │
│  │Queue│     │Queue│     │Queue│ ... (per cluster)            │
│  │  A  │     │  B  │     │  N  │                             │
│  └──┬──┘     └──┬──┘     └──┬──┘                             │
│     │            │            │                                │
│     ↓            ↓            ↓                                │
│  ┌──────┐    ┌──────┐    ┌──────┐                            │
│  │Worker│    │Worker│    │Worker│ (1 per cluster)             │
│  │  A   │    │  B   │    │  N   │                            │
│  └──┬───┘    └──┬───┘    └──┬───┘                            │
│     │            │            │                                │
│     └────────────┴────────────┘                                │
│                  │                                              │
│                  ↓                                              │
│         ┌────────────────┐                                     │
│         │ Agent Spawner  │                                     │
│         └────────┬───────┘                                     │
│                  │                                              │
└──────────────────┼─────────────────────────────────────────────┘
                   │
                   ↓
         ┌─────────────────┐
         │  Agent Process  │ (subprocess)
         │   (Claude CLI)  │
         └─────────────────┘
```

## Risks / Trade-offs

### Risk: Agent hallucinations or non-deterministic behavior
**Mitigation**:
- Strict read-only tools in agent environment
- Human-in-the-loop (reports only, no auto-fix)
- Timeout limits to prevent runaway agents
- Resource isolation via separate workspaces

### Risk: Event flood overwhelming the system
**Mitigation**:
- Aggressive severity filtering
- Global circuit breaker with hard limits
- Per-cluster concurrency control
- Bounded queues with overflow policies
- Deduplication to eliminate redundant work

### Risk: SSE connection instability
**Mitigation**:
- Automatic reconnection with exponential backoff
- Last-Event-ID tracking for resumable connections
- Heartbeat/timeout detection
- Comprehensive error logging

### Risk: Memory leaks from unbounded caches
**Mitigation**:
- Time-based expiry on deduplication cache
- Bounded queue sizes
- Regular cleanup of completed agent workspaces
- Monitoring of memory usage

### Risk: Cluster worker goroutine leaks
**Mitigation**:
- Panic recovery in all worker goroutines
- Context-based cancellation on shutdown
- Monitoring of goroutine count
- Graceful shutdown with timeout

### Trade-off: In-memory state vs. persistence
**Decision**: Use in-memory state for Phase 1
**Rationale**:
- Simpler implementation for prototype
- Acceptable loss of queue state on restart
- No external dependencies (database, Redis)
**Future**: Consider persistence if reliability requirements increase

### Trade-off: Event buffering vs. backpressure
**Decision**: Use bounded buffers with drop/reject policies
**Rationale**:
- Prevents unbounded memory growth
- Explicit handling of overload scenarios
- Clear observability via logs and metrics
**Alternative**: Could implement backpressure to SSE server, but adds complexity

## Migration Plan
N/A - This is a new implementation (Phase 1).

## Open Questions

### Resolved
- ~~Exact format of the "Setup Script" for the agent sandbox~~ → Addressed in Phase 2 (agent-runtime)
- ~~Specific CLI arguments for the target agent's headless mode~~ → Addressed in Phase 2 (agent-runtime)
- ~~SSE client library selection~~ → r3labs/sse chosen

### Outstanding
- Should we persist Last-Event-ID to disk for crash recovery? → Defer to Phase 4 (resilience)
- Should we implement metrics export (Prometheus)? → Defer to Phase 4 (observability)
- Should we support multiple SSE endpoints for HA? → Defer to future if needed
