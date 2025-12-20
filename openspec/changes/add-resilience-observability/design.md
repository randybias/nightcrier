# Design: Resilience and Observability

## Context

Nightcrier needs production-grade resilience and observability to operate reliably in production environments. This design addresses three critical operational concerns:

1. **Resource Management**: Prevent runaway agent processes and memory exhaustion from stale queued events
2. **Connection Resilience**: Handle SSE connection failures gracefully with proper reconnection logic
3. **Operational Visibility**: Provide metrics and health endpoints for monitoring, alerting, and debugging

### Constraints

- Single binary deployment with minimal dependencies
- Go standard library + Prometheus client library only
- Must align with kubernetes-mcp-server patterns and conventions
- No external databases or state stores (in-memory state is sufficient)
- Internal environment with relaxed security requirements initially

### Stakeholders

- **Operations Team**: Needs visibility into system health, queue depths, and failure modes
- **SRE Team**: Requires metrics for alerting and capacity planning
- **Development Team**: Benefits from structured logging and health endpoints for debugging

## Goals / Non-Goals

### Goals

- Enforce execution timeouts for agent sessions to prevent resource exhaustion
- Implement queue expiration to drop stale events automatically
- Provide comprehensive metrics for operational visibility
- Handle SSE connection failures with automatic reconnection
- Expose health and readiness endpoints for Kubernetes integration
- Enable graceful shutdown without data loss

### Non-Goals

- Persistent metrics storage (Prometheus handles this)
- Custom metrics aggregation or dashboards (use Grafana)
- Complex distributed tracing (out of scope for Phase 4)
- Multi-cluster coordination (per-cluster state only)
- Advanced circuit breaker algorithms (simple global limit is sufficient)

## Decisions

### Decision 1: Use Prometheus for Metrics

**Choice**: Use the official Prometheus Go client library (`github.com/prometheus/client_golang`)

**Rationale**:
- Industry standard for cloud-native applications
- CNCF graduated project with excellent Go support
- Native integration with Kubernetes monitoring stacks
- Zero-dependency metrics exposition (simple HTTP handler)
- Aligns with kubernetes-mcp-server ecosystem

**Alternatives Considered**:
- **OpenTelemetry**: More complex, overkill for this phase, can migrate later if needed
- **StatsD/Graphite**: Older pattern, less cloud-native, requires additional infrastructure
- **Log-based metrics**: Insufficient for real-time alerting and dashboards

### Decision 2: Metrics Naming Convention

**Choice**: Follow Prometheus best practices with `kubernetes_mcp_runner_*` prefix

**Rationale**:
- Clear namespace avoids collisions with other applications
- Consistent with kubernetes-mcp-server naming patterns
- Enables easy scoping in multi-service Prometheus setups
- Human-readable and self-documenting

**Convention Details**:
- Prefix: `kubernetes_mcp_runner_`
- Base units: `_seconds`, `_bytes`, `_total`
- Snake case: `events_received_total` not `eventsReceivedTotal`
- Counter suffix: `_total` for monotonic counters
- Label dimensions: `cluster`, `severity`, `status`, `reason`

### Decision 3: Context-Based Timeout Implementation

**Choice**: Use `context.WithTimeout` for agent execution timeouts

**Rationale**:
- Standard Go pattern for cancellation and deadlines
- Propagates cancellation signal to all child operations
- Integrates naturally with exec.CommandContext for process management
- Allows graceful cleanup before forced termination

**Implementation Pattern**:
```go
ctx, cancel := context.WithTimeout(parentCtx, agentTimeout)
defer cancel()

cmd := exec.CommandContext(ctx, "claude", "--headless", ...)
if err := cmd.Run(); err != nil {
    if ctx.Err() == context.DeadlineExceeded {
        // Handle timeout specifically
        metrics.AgentTimeouts.Inc()
    }
}
```

### Decision 4: Queue Expiration Strategy

**Choice**: Periodic sweep with timestamp-based expiration

**Rationale**:
- Simple implementation with low overhead
- No need for per-event timers (reduces memory usage)
- Configurable sweep interval (default: 1 minute)
- Deterministic behavior for testing

**Implementation Pattern**:
- Each queued event stores `enqueuedAt time.Time`
- Background goroutine runs every 1 minute
- Compares `time.Since(enqueuedAt)` against `maxQueueAge`
- Removes expired events, increments `events_expired_total` metric

**Alternatives Considered**:
- **Per-event timers**: Higher memory overhead, more complex
- **Priority expiration**: Unnecessary complexity for single-cluster queues
- **Lazy expiration**: Could leave stale events in memory longer

### Decision 5: SSE Reconnection Strategy

**Choice**: Exponential backoff with jitter and max retry limit

**Rationale**:
- Prevents thundering herd when multiple clusters reconnect simultaneously
- Exponential backoff reduces load on kubernetes-mcp-server during outages
- Jitter prevents synchronized reconnection attempts
- Max retry limit prevents infinite loops if server is permanently down

**Parameters**:
- Initial delay: 1 second
- Max delay: 60 seconds
- Jitter: ±25%
- Max retries: unlimited (but respects parent context cancellation)
- Backoff library: `cenkalti/backoff` or simple custom implementation

**Reconnection Flow**:
1. SSE connection closes (network error, server restart, etc.)
2. Log warning with reason
3. Calculate next retry delay with exponential backoff + jitter
4. Sleep for delay duration (respecting context cancellation)
5. Attempt reconnection
6. On success: reset backoff timer, log success
7. On failure: increment failure counter, continue backoff

### Decision 6: Health and Readiness Endpoints

**Choice**: Separate `/healthz` (liveness) and `/readyz` (readiness) endpoints

**Rationale**:
- Standard Kubernetes probe convention
- Liveness: checks if process is alive and not deadlocked
- Readiness: checks if system can accept traffic (SSE connected, queue not full)
- Allows Kubernetes to automatically restart or remove unhealthy pods

**Endpoint Behaviors**:

**/healthz** (Liveness Probe):
- Returns 200 OK if basic health checks pass
- Checks: goroutine count not excessive, memory not exhausted
- Simple and fast (no external dependencies)
- Should rarely fail (only if process is fundamentally broken)

**/readyz** (Readiness Probe):
- Returns 200 OK if ready to process events
- Checks: at least one SSE connection active, global circuit breaker not open
- Returns 503 Service Unavailable if not ready
- Kubernetes will stop sending traffic if readiness fails

### Decision 7: Graceful Shutdown

**Choice**: Signal-based shutdown with timeout-bound drain period

**Rationale**:
- Standard Go pattern for clean termination
- Prevents data loss from in-flight agent sessions
- Kubernetes sends SIGTERM before SIGKILL
- Configurable shutdown timeout aligns with pod `terminationGracePeriodSeconds`

**Shutdown Sequence**:
1. Receive SIGTERM or SIGINT signal
2. Stop accepting new SSE events (mark readiness as false)
3. Close all SSE client connections gracefully
4. Wait for in-flight agent sessions to complete (up to shutdown timeout)
5. Cancel remaining agent contexts after timeout
6. Close metrics endpoint
7. Flush logs and exit

**Configuration**:
- Default shutdown timeout: 30 seconds
- Configurable via `--shutdown-timeout` flag or `SHUTDOWN_TIMEOUT_SECONDS` env var
- Should be less than Kubernetes `terminationGracePeriodSeconds` (default 30s)

## Metrics Catalog

### Event Flow Metrics

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `kubernetes_mcp_runner_events_received_total` | Counter | `cluster`, `severity` | Total events received from SSE streams |
| `kubernetes_mcp_runner_events_filtered_total` | Counter | `cluster`, `reason` | Events filtered before queuing (e.g., severity too low) |
| `kubernetes_mcp_runner_events_queued_total` | Counter | `cluster` | Events added to per-cluster queue |
| `kubernetes_mcp_runner_events_expired_total` | Counter | `cluster` | Events removed from queue due to age |
| `kubernetes_mcp_runner_events_dequeued_total` | Counter | `cluster` | Events removed from queue for processing |
| `kubernetes_mcp_runner_queue_depth` | Gauge | `cluster` | Current number of events in per-cluster queue |

### Agent Lifecycle Metrics

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `kubernetes_mcp_runner_agents_spawned_total` | Counter | `cluster` | Total agent sessions started |
| `kubernetes_mcp_runner_agents_completed_total` | Counter | `cluster`, `status` | Agent sessions completed (status: success, failure) |
| `kubernetes_mcp_runner_agents_timeout_total` | Counter | `cluster` | Agent sessions killed due to timeout |
| `kubernetes_mcp_runner_agent_duration_seconds` | Histogram | `cluster`, `status` | Agent execution duration distribution |
| `kubernetes_mcp_runner_agents_active` | Gauge | `cluster` | Current number of active agent sessions |

### Circuit Breaker Metrics

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `kubernetes_mcp_runner_circuit_breaker_state` | Gauge | none | Global circuit breaker state (0=closed, 1=open) |
| `kubernetes_mcp_runner_circuit_breaker_events_dropped_total` | Counter | `cluster` | Events dropped due to circuit breaker open |

### SSE Connection Metrics

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `kubernetes_mcp_runner_sse_connections_active` | Gauge | `cluster` | Current number of active SSE connections |
| `kubernetes_mcp_runner_sse_reconnections_total` | Counter | `cluster`, `reason` | SSE reconnection attempts (reason: network_error, server_restart, etc.) |
| `kubernetes_mcp_runner_sse_connection_errors_total` | Counter | `cluster`, `reason` | SSE connection failures |
| `kubernetes_mcp_runner_sse_connection_duration_seconds` | Histogram | `cluster` | SSE connection uptime before disconnect |

### System Metrics

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `kubernetes_mcp_runner_build_info` | Gauge | `version`, `git_commit` | Build information (value always 1) |
| `kubernetes_mcp_runner_up` | Gauge | none | Whether the runner is up (value always 1 when scraped) |

**Note**: Go runtime metrics (memory, goroutines, GC) are automatically exposed by the Prometheus Go client and do not need custom instrumentation.

## Histogram Bucket Configuration

### Agent Duration Histogram

**Buckets**: `[1, 5, 10, 30, 60, 120, 300]` (seconds)

**Rationale**:
- Expected agent sessions: 10-60 seconds for typical triage
- Timeout typically set at 300 seconds (5 minutes)
- Buckets capture distribution from fast (1s) to timeout boundary

### SSE Connection Duration Histogram

**Buckets**: `[60, 300, 600, 1800, 3600, 7200, 14400]` (seconds)

**Rationale**:
- Expected connection lifetime: hours to days
- Buckets capture short-lived (1 min), normal (30 min - 1 hour), and long-lived (4+ hours) connections
- Helps identify connection stability issues

## Implementation Details

### Timeout Enforcement

```go
// Pseudo-code for agent timeout enforcement
func (r *Runner) executeAgent(ctx context.Context, event Event) error {
    // Create timeout context
    ctx, cancel := context.WithTimeout(ctx, r.config.AgentTimeout)
    defer cancel()

    // Record start time
    start := time.Now()
    defer func() {
        duration := time.Since(start).Seconds()
        agentDurationHistogram.WithLabelValues(event.Cluster, "unknown").Observe(duration)
    }()

    // Build agent command
    cmd := exec.CommandContext(ctx, r.config.AgentBinary, buildArgs(event)...)

    // Run agent
    err := cmd.Run()

    // Check for timeout
    if ctx.Err() == context.DeadlineExceeded {
        agentTimeoutsCounter.WithLabelValues(event.Cluster).Inc()
        return fmt.Errorf("agent timeout after %s", r.config.AgentTimeout)
    }

    // Check for other errors
    if err != nil {
        agentCompletedCounter.WithLabelValues(event.Cluster, "failure").Inc()
        return err
    }

    agentCompletedCounter.WithLabelValues(event.Cluster, "success").Inc()
    return nil
}
```

### Queue Expiration

```go
// Pseudo-code for queue expiration sweep
func (r *Runner) startQueueSweeper(ctx context.Context) {
    ticker := time.NewTicker(r.config.QueueSweepInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            r.sweepExpiredEvents()
        }
    }
}

func (r *Runner) sweepExpiredEvents() {
    r.mu.Lock()
    defer r.mu.Unlock()

    now := time.Now()

    for cluster, queue := range r.queues {
        newQueue := make([]Event, 0, len(queue))

        for _, event := range queue {
            age := now.Sub(event.EnqueuedAt)

            if age > r.config.MaxQueueAge {
                eventsExpiredCounter.WithLabelValues(cluster).Inc()
                log.Warn().
                    Str("cluster", cluster).
                    Str("event_id", event.ID).
                    Dur("age", age).
                    Msg("Expired queued event")
            } else {
                newQueue = append(newQueue, event)
            }
        }

        r.queues[cluster] = newQueue
        queueDepthGauge.WithLabelValues(cluster).Set(float64(len(newQueue)))
    }
}
```

### SSE Reconnection

```go
// Pseudo-code for SSE reconnection with backoff
func (r *Runner) connectSSE(ctx context.Context, cluster string) error {
    backoff := &exponentialBackoff{
        initial: 1 * time.Second,
        max:     60 * time.Second,
        jitter:  0.25,
    }

    for {
        // Check context cancellation
        if ctx.Err() != nil {
            return ctx.Err()
        }

        // Attempt connection
        conn, err := r.dialSSE(cluster)
        if err != nil {
            sseConnectionErrorsCounter.WithLabelValues(cluster, classifyError(err)).Inc()

            // Calculate backoff
            delay := backoff.Next()
            log.Warn().
                Str("cluster", cluster).
                Err(err).
                Dur("retry_in", delay).
                Msg("SSE connection failed, retrying")

            // Sleep with context awareness
            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-time.After(delay):
                sseReconnectionsCounter.WithLabelValues(cluster, "retry").Inc()
                continue
            }
        }

        // Success - reset backoff
        backoff.Reset()
        sseConnectionsActiveGauge.WithLabelValues(cluster).Inc()
        defer sseConnectionsActiveGauge.WithLabelValues(cluster).Dec()

        log.Info().Str("cluster", cluster).Msg("SSE connection established")

        // Handle connection
        start := time.Now()
        err = r.handleSSEConnection(ctx, cluster, conn)
        duration := time.Since(start).Seconds()
        sseConnectionDurationHistogram.WithLabelValues(cluster).Observe(duration)

        // Connection closed
        sseReconnectionsCounter.WithLabelValues(cluster, classifyError(err)).Inc()
        log.Info().
            Str("cluster", cluster).
            Err(err).
            Float64("duration_seconds", duration).
            Msg("SSE connection closed")
    }
}
```

### Health Endpoints

```go
// Pseudo-code for health endpoints
func (r *Runner) healthzHandler(w http.ResponseWriter, req *http.Request) {
    // Basic liveness check
    if atomic.LoadInt32(&r.shutdown) == 1 {
        http.Error(w, "Shutting down", http.StatusServiceUnavailable)
        return
    }

    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
}

func (r *Runner) readyzHandler(w http.ResponseWriter, req *http.Request) {
    // Check if ready to accept traffic
    checks := []struct {
        name string
        fn   func() error
    }{
        {"sse_connections", r.checkSSEConnections},
        {"circuit_breaker", r.checkCircuitBreaker},
    }

    var errors []string

    for _, check := range checks {
        if err := check.fn(); err != nil {
            errors = append(errors, fmt.Sprintf("%s: %v", check.name, err))
        }
    }

    if len(errors) > 0 {
        http.Error(w, strings.Join(errors, "; "), http.StatusServiceUnavailable)
        return
    }

    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Ready"))
}

func (r *Runner) checkSSEConnections() error {
    activeConns := r.getActiveSSECount()
    if activeConns == 0 {
        return fmt.Errorf("no active SSE connections")
    }
    return nil
}

func (r *Runner) checkCircuitBreaker() error {
    if r.circuitBreaker.IsOpen() {
        return fmt.Errorf("circuit breaker is open")
    }
    return nil
}
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `AGENT_TIMEOUT_SECONDS` | `300` | Maximum agent execution time (5 minutes) |
| `MAX_QUEUE_AGE_SECONDS` | `600` | Maximum event queue age (10 minutes) |
| `QUEUE_SWEEP_INTERVAL_SECONDS` | `60` | Queue expiration check interval (1 minute) |
| `SHUTDOWN_TIMEOUT_SECONDS` | `30` | Graceful shutdown timeout |
| `METRICS_PORT` | `9090` | Prometheus metrics HTTP port |
| `HEALTH_PORT` | `8080` | Health/readiness endpoint HTTP port |
| `SSE_RECONNECT_INITIAL_DELAY_SECONDS` | `1` | Initial SSE reconnection delay |
| `SSE_RECONNECT_MAX_DELAY_SECONDS` | `60` | Maximum SSE reconnection delay |

### Command-Line Flags

All environment variables have corresponding CLI flags (e.g., `--agent-timeout`, `--max-queue-age`) following kubernetes-mcp-server conventions.

## Risks / Trade-offs

### Risk 1: Agent Process Leaks

**Risk**: Agent processes may not terminate cleanly on timeout

**Mitigation**:
- Use `CommandContext` with timeout for automatic SIGKILL
- Monitor orphaned processes via system metrics
- Add agent PID tracking for manual cleanup if needed
- Test timeout behavior extensively

### Risk 2: Metrics Cardinality Explosion

**Risk**: High-cardinality labels (e.g., event IDs) could create millions of time series

**Mitigation**:
- Use only low-cardinality labels: cluster (10-100s), severity (3-5), status (2-3)
- Avoid event IDs, timestamps, or resource names in labels
- Document label cardinality in metrics catalog
- Monitor Prometheus memory usage

### Risk 3: Queue Memory Growth

**Risk**: Large queue backlogs could exhaust memory

**Mitigation**:
- Queue expiration sweeper runs every minute
- Configurable max queue age (default 10 minutes)
- Per-cluster queue limits (TBD in future phase)
- Monitor queue depth gauge for alerting

### Risk 4: SSE Reconnection Storm

**Risk**: Many clusters reconnecting simultaneously could overwhelm kubernetes-mcp-server

**Mitigation**:
- Exponential backoff with jitter (±25%)
- Staggered startup delays for multiple clusters
- Kubernetes-mcp-server must handle connection spikes gracefully
- Monitor reconnection rates

### Risk 5: Graceful Shutdown Timeout

**Risk**: In-flight agents may not complete within shutdown timeout

**Mitigation**:
- Configurable shutdown timeout (default 30s, can increase)
- Force-kill agents after timeout
- Log which agents were interrupted for post-mortem
- Kubernetes `terminationGracePeriodSeconds` should be longer than shutdown timeout

## Migration Plan

### Phase 4A: Core Resilience (This Change)

1. Implement timeout enforcement with context
2. Add queue expiration background worker
3. Instrument code with Prometheus metrics
4. Expose `/metrics` endpoint
5. Add health and readiness endpoints
6. Implement SSE reconnection logic
7. Add graceful shutdown handler

### Phase 4B: Observability Enhancements (Future)

1. Add Grafana dashboard templates
2. Add Prometheus alerting rules examples
3. Implement structured logging with levels
4. Add request tracing (OpenTelemetry optional)

### Testing Strategy

1. **Unit tests**: timeout enforcement, queue expiration logic, backoff calculation
2. **Integration tests**: SSE reconnection scenarios, graceful shutdown
3. **Manual testing**: trigger timeouts, kill SSE server, measure metrics
4. **Load testing**: verify metrics cardinality under load

### Rollback Plan

If issues arise in production:
1. Revert to previous version (metrics/health endpoints are additive)
2. Timeout/queue expiration can be disabled via config flags
3. Metrics endpoint can be disabled or isolated to internal network

## Open Questions

1. **Q**: Should we implement metric push to Pushgateway for batch jobs?
   **A**: No, not for Phase 4. SSE-based runner is long-lived, so Prometheus pull model is appropriate.

2. **Q**: What Prometheus retention period should operators configure?
   **A**: Out of scope for this design. Recommend 15-30 days in operational documentation.

3. **Q**: Should we add custom readiness checks (e.g., disk space for incident artifacts)?
   **A**: Defer to Phase 5. Initial readiness checks are sufficient for Kubernetes integration.

4. **Q**: How to handle agent processes that ignore SIGTERM?
   **A**: CommandContext sends SIGKILL after timeout. If that fails, document manual cleanup procedures.

5. **Q**: Should metrics be scoped by agent type (e.g., Claude vs. Codex)?
   **A**: Not in Phase 4. Single agent type is assumed. Add label if multi-agent support is needed later.
