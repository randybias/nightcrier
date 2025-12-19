# Change: Add Resilience and Observability (Phase 4)

## Walking Skeleton Baseline

The walking-skeleton implementation (archived 2025-12-18) provides minimal resilience features:

**Already Implemented:**
- Basic graceful shutdown on SIGTERM/SIGINT
- Configurable agent timeout (AGENT_TIMEOUT env var, default 300s)
- Context-based cancellation for event processing

**This Change Adds:**
All advanced resilience and observability features: reconnection with backoff, queue expiration, Prometheus metrics, health endpoints, and comprehensive shutdown handling.

## Why

The Kubernetes MCP Alerts Event Runner currently lacks production-grade resilience mechanisms and operational visibility. Without these capabilities, the system is vulnerable to:

- **Resource Exhaustion**: Long-running or hung agent processes can consume unbounded CPU and memory
- **Queue Memory Leaks**: Stale events accumulate indefinitely in per-cluster queues
- **Connection Fragility**: MCP connection failures require manual intervention to recover
- **Operational Blindness**: No metrics or health endpoints for monitoring, alerting, or debugging
- **Graceless Failures**: Unclean shutdowns can lose in-flight work and leave orphaned processes

Phase 4 hardens the system for production deployment by adding execution timeouts, queue expiration, automatic reconnection, comprehensive metrics, health endpoints, and graceful shutdown capabilities.

## What Changes

### 1. Resilience Mechanisms

- **Execution Timeouts**: Enforce maximum agent execution time (default 5 minutes)
  - Use `context.WithTimeout` for cancellation propagation
  - Send SIGTERM then SIGKILL for graceful process termination
  - Log timeout events with cluster and incident details
  - Increment timeout metrics

- **Queue Expiration**: Automatically drop stale events from per-cluster queues
  - Background sweeper runs every minute (configurable)
  - Remove events older than max age threshold (default 10 minutes)
  - Update queue depth metrics after expiration
  - Log expired events for audit trail

- **SSE Reconnection**: Handle connection failures with automatic retry
  - Exponential backoff with jitter (±25%)
  - Initial delay: 1 second, max delay: 60 seconds
  - Classify errors (network, server, timeout) for metrics
  - Reset backoff on successful connection
  - Respect context cancellation for clean shutdown

- **Graceful Shutdown**: Clean termination without data loss
  - Handle SIGTERM and SIGINT signals
  - Wait for in-flight agent sessions to complete (up to shutdown timeout)
  - Force-kill remaining agents after timeout
  - Close SSE connections and HTTP servers gracefully
  - Exit with code 0 for Kubernetes integration

### 2. Observability Instrumentation

- **Prometheus Metrics**: Comprehensive operational metrics
  - Event flow: received, filtered, queued, expired, dequeued
  - Agent lifecycle: spawned, completed, failed, timed out, active
  - Circuit breaker: state, events dropped
  - SSE connections: active, reconnections, errors, duration
  - System: build info, uptime
  - Naming convention: `kubernetes_mcp_runner_*` prefix
  - Labels: cluster, severity, status, reason (low cardinality only)
  - Histograms: agent duration (1-300s), SSE connection duration (60s-4h)

- **Metrics HTTP Endpoint**: Expose `/metrics` on port 9090 (configurable)
  - Prometheus text format
  - Include Go runtime metrics (memory, goroutines, GC)
  - Graceful shutdown support

- **Health Endpoints**: Kubernetes liveness and readiness probes
  - `/healthz` (liveness): Returns 200 OK when process is healthy, 503 when shutting down
  - `/readyz` (readiness): Returns 200 OK when at least one SSE connection active and circuit breaker closed
  - Separate HTTP server on port 8080 (configurable)

- **Structured Logging**: Enhanced logs for debugging and audit
  - Timeout events: WARN level with cluster, incident_id, duration, threshold
  - Queue expiration: WARN level with cluster, event_id, age, max_age
  - SSE reconnection: WARN level on failure, INFO level on success
  - Graceful shutdown: INFO level with signal, in-flight agents, completion status

### 3. Configuration

All resilience parameters configurable via environment variables and CLI flags:

- `AGENT_TIMEOUT_SECONDS` / `--agent-timeout` (default: 300)
- `MAX_QUEUE_AGE_SECONDS` / `--max-queue-age` (default: 600)
- `QUEUE_SWEEP_INTERVAL_SECONDS` / `--queue-sweep-interval` (default: 60)
- `SHUTDOWN_TIMEOUT_SECONDS` / `--shutdown-timeout` (default: 30)
- `METRICS_PORT` / `--metrics-port` (default: 9090)
- `HEALTH_PORT` / `--health-port` (default: 8080)
- `SSE_RECONNECT_INITIAL_DELAY_SECONDS` / `--sse-reconnect-initial-delay` (default: 1)
- `SSE_RECONNECT_MAX_DELAY_SECONDS` / `--sse-reconnect-max-delay` (default: 60)

CLI flags take precedence over environment variables.

### 4. Error Recovery

- SSE stream errors trigger automatic reconnection (no crash)
- Agent process crashes increment failure metrics but don't stop event processing
- Metrics endpoint errors return HTTP 500 but don't crash the runner
- Queue sweeper panics are recovered with restart

## Impact

### New Capabilities

- `resilience`: Execution timeouts, queue expiration, SSE reconnection, graceful shutdown
- `observability`: Prometheus metrics, health endpoints, structured logging

### New Code

- `internal/metrics/`: Metric definitions and HTTP endpoint
- `internal/executor/`: Timeout utilities and agent execution wrapper
- `internal/queue/`: Queue expiration sweeper
- `internal/sse/`: Exponential backoff and reconnection logic
- `internal/health/`: Liveness and readiness HTTP handlers
- `internal/shutdown/`: Signal handling and graceful shutdown orchestration
- `internal/config/`: Configuration parsing and validation

### Modified Code

- Agent spawner: Add timeout context and metrics instrumentation
- SSE client: Wrap with reconnection loop and metrics
- Queue manager: Add EnqueuedAt timestamp and sweeper integration
- Main entrypoint: Add signal handling, HTTP servers, graceful shutdown

### Dependencies

- **Added**: `github.com/prometheus/client_golang` (official Prometheus Go client)
- No other external dependencies (use Go standard library for backoff, signals, HTTP)

### Deployment Changes

- **Kubernetes Manifests**: Add liveness and readiness probes
  - Liveness: `httpGet: /healthz:8080`, `initialDelaySeconds: 10`, `periodSeconds: 10`, `failureThreshold: 3`
  - Readiness: `httpGet: /readyz:8080`, `initialDelaySeconds: 5`, `periodSeconds: 5`, `failureThreshold: 2`

- **Prometheus Scrape Config**: Add job for event-runner
  ```yaml
  - job_name: 'kubernetes-mcp-runner'
    static_configs:
      - targets: ['event-runner:9090']
  ```

- **Grafana Dashboards**: (Optional) Import dashboard template for runner metrics

- **Alert Rules**: (Optional) Add Prometheus alerting rules
  - High timeout rate: `rate(kubernetes_mcp_runner_agents_timeout_total[5m]) > 0.1`
  - Queue depth: `kubernetes_mcp_runner_queue_depth > 50`
  - No SSE connections: `sum(kubernetes_mcp_runner_sse_connections_active) == 0`
  - Circuit breaker open: `kubernetes_mcp_runner_circuit_breaker_state == 1`

### Breaking Changes

- None. All changes are additive. Existing behavior is preserved with new defaults.

### Non-Breaking Changes

- New HTTP endpoints on ports 8080 (health) and 9090 (metrics) - ensure these ports are available
- Shutdown behavior changes from immediate exit to graceful drain (respects Kubernetes terminationGracePeriodSeconds)
- SSE reconnection is automatic instead of crashing on connection failure

### Configuration Changes

- Eight new environment variables / CLI flags (all optional with sensible defaults)
- Existing configuration is unchanged

### Testing Strategy

- **Unit Tests**: Timeout logic, queue expiration, backoff calculation, health endpoints, metrics registration
- **Integration Tests**: SSE reconnection after server restart, graceful shutdown with in-flight agents, timeout end-to-end
- **Manual Testing**: Deploy to test cluster, trigger timeouts, kill SSE server, send SIGTERM, scrape metrics

### Rollback Plan

- Revert to previous version if issues arise (metrics/health endpoints are additive, safe to remove)
- Timeout and queue expiration can be disabled by setting very high values
- No data migration required (all state is in-memory)

### Documentation

- Update operational documentation with new configuration parameters
- Create metrics reference guide with all metrics and labels
- Provide deployment examples with Kubernetes probes and Prometheus config
- Add troubleshooting guide for common issues (timeouts, reconnection failures, shutdown)

## Success Criteria

Phase 4 is complete when:

1. Agent sessions are automatically killed after configured timeout
2. Old events are automatically removed from queues
3. SSE connections reconnect automatically with exponential backoff
4. `/metrics` endpoint exposes all defined metrics in Prometheus format
5. `/healthz` and `/readyz` endpoints work correctly with Kubernetes probes
6. Graceful shutdown completes in-flight agents within timeout
7. All configuration parameters are documented and tested
8. Unit and integration tests pass
9. Manual testing in test cluster validates all behaviors
10. OpenSpec validation passes with `--strict` flag
