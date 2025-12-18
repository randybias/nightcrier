# Implementation Tasks (Phase 4)

## 1. Project Setup and Dependencies

- [ ] 1.1 Add Prometheus client library to go.mod
  - [ ] 1.1.1 Run `go get github.com/prometheus/client_golang@latest`
  - [ ] 1.1.2 Run `go mod tidy` to clean dependencies
  - [ ] 1.1.3 Verify no version conflicts with existing dependencies

## 2. Metrics Infrastructure

- [ ] 2.1 Create metrics package structure
  - [ ] 2.1.1 Create `internal/metrics/metrics.go` file
  - [ ] 2.1.2 Create `internal/metrics/registry.go` file for custom registry
  - [ ] 2.1.3 Create `internal/metrics/metrics_test.go` for unit tests

- [ ] 2.2 Define event flow metrics
  - [ ] 2.2.1 Define `events_received_total` counter with cluster and severity labels
  - [ ] 2.2.2 Define `events_filtered_total` counter with cluster and reason labels
  - [ ] 2.2.3 Define `events_queued_total` counter with cluster label
  - [ ] 2.2.4 Define `events_expired_total` counter with cluster label
  - [ ] 2.2.5 Define `events_dequeued_total` counter with cluster label
  - [ ] 2.2.6 Define `queue_depth` gauge with cluster label

- [ ] 2.3 Define agent lifecycle metrics
  - [ ] 2.3.1 Define `agents_spawned_total` counter with cluster label
  - [ ] 2.3.2 Define `agents_completed_total` counter with cluster and status labels
  - [ ] 2.3.3 Define `agents_timeout_total` counter with cluster label
  - [ ] 2.3.4 Define `agents_active` gauge with cluster label
  - [ ] 2.3.5 Define `agent_duration_seconds` histogram with buckets [1, 5, 10, 30, 60, 120, 300]

- [ ] 2.4 Define circuit breaker metrics
  - [ ] 2.4.1 Define `circuit_breaker_state` gauge (0=closed, 1=open)
  - [ ] 2.4.2 Define `circuit_breaker_events_dropped_total` counter with cluster label

- [ ] 2.5 Define SSE connection metrics
  - [ ] 2.5.1 Define `sse_connections_active` gauge with cluster label
  - [ ] 2.5.2 Define `sse_reconnections_total` counter with cluster and reason labels
  - [ ] 2.5.3 Define `sse_connection_errors_total` counter with cluster and reason labels
  - [ ] 2.5.4 Define `sse_connection_duration_seconds` histogram with buckets [60, 300, 600, 1800, 3600, 7200, 14400]

- [ ] 2.6 Define system metrics
  - [ ] 2.6.1 Define `build_info` gauge with version, git_commit, and build_date labels
  - [ ] 2.6.2 Define `up` gauge (always 1 when scraped)

- [ ] 2.7 Implement metrics HTTP endpoint
  - [ ] 2.7.1 Create HTTP handler using promhttp.Handler()
  - [ ] 2.7.2 Configure metrics server on port from config (default 9090)
  - [ ] 2.7.3 Add graceful shutdown for metrics server
  - [ ] 2.7.4 Test metrics endpoint returns Prometheus format

## 3. Execution Timeout Implementation

- [ ] 3.1 Create timeout utilities
  - [ ] 3.1.1 Create `internal/executor/timeout.go` file
  - [ ] 3.1.2 Implement context.WithTimeout wrapper for agent execution
  - [ ] 3.1.3 Add helper function to classify timeout vs other errors

- [ ] 3.2 Integrate timeout into agent execution
  - [ ] 3.2.1 Modify agent spawner to create timeout context
  - [ ] 3.2.2 Use exec.CommandContext with timeout context
  - [ ] 3.2.3 Add timeout detection logic in error handling
  - [ ] 3.2.4 Increment timeout metric when timeout occurs
  - [ ] 3.2.5 Record execution duration in histogram on completion

- [ ] 3.3 Implement graceful process termination
  - [ ] 3.3.1 Send SIGTERM to agent process on timeout
  - [ ] 3.3.2 Wait up to 5 seconds for graceful shutdown
  - [ ] 3.3.3 Send SIGKILL if process still running after 5 seconds
  - [ ] 3.3.4 Clean up agent workspace resources (temp files, etc.)

- [ ] 3.4 Add timeout configuration
  - [ ] 3.4.1 Add AGENT_TIMEOUT_SECONDS environment variable support
  - [ ] 3.4.2 Add --agent-timeout CLI flag
  - [ ] 3.4.3 Set default to 300 seconds (5 minutes)
  - [ ] 3.4.4 Validate timeout is positive integer
  - [ ] 3.4.5 Add timeout value to structured logs

- [ ] 3.5 Add timeout logging
  - [ ] 3.5.1 Log at WARN level when timeout occurs
  - [ ] 3.5.2 Include cluster, incident_id, duration, and threshold in log
  - [ ] 3.5.3 Log successful cleanup after timeout

## 4. Queue Expiration Implementation

- [ ] 4.1 Add timestamp tracking to queue events
  - [ ] 4.1.1 Add EnqueuedAt time.Time field to event struct
  - [ ] 4.1.2 Set EnqueuedAt when adding event to queue
  - [ ] 4.1.3 Ensure timestamp survives queue operations

- [ ] 4.2 Implement queue sweeper
  - [ ] 4.2.1 Create `internal/queue/sweeper.go` file
  - [ ] 4.2.2 Implement background goroutine with ticker (1 minute interval)
  - [ ] 4.2.3 Add context cancellation support for shutdown
  - [ ] 4.2.4 Add panic recovery with logging and restart

- [ ] 4.3 Implement expiration logic
  - [ ] 4.3.1 Calculate age for each queued event (time.Since(EnqueuedAt))
  - [ ] 4.3.2 Compare age against max queue age threshold
  - [ ] 4.3.3 Remove expired events from queue
  - [ ] 4.3.4 Preserve events within age limit
  - [ ] 4.3.5 Update queue_depth gauge after sweep

- [ ] 4.4 Add expiration metrics
  - [ ] 4.4.1 Increment events_expired_total for each expired event
  - [ ] 4.4.2 Update queue_depth gauge after expiration

- [ ] 4.5 Add expiration configuration
  - [ ] 4.5.1 Add MAX_QUEUE_AGE_SECONDS environment variable support
  - [ ] 4.5.2 Add --max-queue-age CLI flag
  - [ ] 4.5.3 Set default to 600 seconds (10 minutes)
  - [ ] 4.5.4 Validate max age is positive integer
  - [ ] 4.5.5 Add QUEUE_SWEEP_INTERVAL_SECONDS environment variable
  - [ ] 4.5.6 Add --queue-sweep-interval CLI flag (default 60 seconds)

- [ ] 4.6 Add expiration logging
  - [ ] 4.6.1 Log at WARN level for each expired event
  - [ ] 4.6.2 Include cluster, event_id, age, and max_age in log

## 5. SSE Reconnection Implementation

- [ ] 5.1 Create exponential backoff utility
  - [ ] 5.1.1 Create `internal/sse/backoff.go` file
  - [ ] 5.1.2 Implement exponential backoff with jitter (±25%)
  - [ ] 5.1.3 Set initial delay to 1 second
  - [ ] 5.1.4 Set max delay to 60 seconds
  - [ ] 5.1.5 Add Reset() method for successful connections
  - [ ] 5.1.6 Add unit tests for backoff calculation

- [ ] 5.2 Implement reconnection loop
  - [ ] 5.2.1 Wrap SSE connection logic in infinite retry loop
  - [ ] 5.2.2 Add context cancellation check before each attempt
  - [ ] 5.2.3 Classify errors (network, server, timeout, etc.)
  - [ ] 5.2.4 Calculate backoff delay with jitter
  - [ ] 5.2.5 Sleep with context-aware select statement
  - [ ] 5.2.6 Reset backoff on successful connection

- [ ] 5.3 Add reconnection metrics
  - [ ] 5.3.1 Increment sse_reconnections_total on each retry
  - [ ] 5.3.2 Include cluster and reason labels
  - [ ] 5.3.3 Increment sse_connection_errors_total on failure
  - [ ] 5.3.4 Update sse_connections_active gauge on connect/disconnect
  - [ ] 5.3.5 Record connection duration in histogram on disconnect

- [ ] 5.4 Add reconnection configuration
  - [ ] 5.4.1 Add SSE_RECONNECT_INITIAL_DELAY_SECONDS env var
  - [ ] 5.4.2 Add --sse-reconnect-initial-delay CLI flag
  - [ ] 5.4.3 Add SSE_RECONNECT_MAX_DELAY_SECONDS env var
  - [ ] 5.4.4 Add --sse-reconnect-max-delay CLI flag

- [ ] 5.5 Add reconnection logging
  - [ ] 5.5.1 Log at WARN level on connection failure
  - [ ] 5.5.2 Include cluster, error, retry_in, and attempt_number in log
  - [ ] 5.5.3 Log at INFO level on successful reconnection
  - [ ] 5.5.4 Include cluster and downtime_duration in success log

## 6. Health and Readiness Endpoints

- [ ] 6.1 Implement liveness endpoint
  - [ ] 6.1.1 Create `internal/health/liveness.go` file
  - [ ] 6.1.2 Implement /healthz HTTP handler
  - [ ] 6.1.3 Check shutdown flag (return 503 if shutting down)
  - [ ] 6.1.4 Return 200 OK if healthy
  - [ ] 6.1.5 Add 1-second response timeout
  - [ ] 6.1.6 Test liveness probe behavior

- [ ] 6.2 Implement readiness endpoint
  - [ ] 6.2.1 Create `internal/health/readiness.go` file
  - [ ] 6.2.2 Implement /readyz HTTP handler
  - [ ] 6.2.3 Add checkSSEConnections helper (at least 1 active)
  - [ ] 6.2.4 Add checkCircuitBreaker helper (must be closed)
  - [ ] 6.2.5 Return 503 with error message if checks fail
  - [ ] 6.2.6 Return 200 OK if all checks pass
  - [ ] 6.2.7 Add 1-second response timeout
  - [ ] 6.2.8 Test readiness probe behavior

- [ ] 6.3 Configure health HTTP server
  - [ ] 6.3.1 Create separate HTTP server for health endpoints
  - [ ] 6.3.2 Add HEALTH_PORT env var support (default 8080)
  - [ ] 6.3.3 Add --health-port CLI flag
  - [ ] 6.3.4 Register /healthz and /readyz handlers
  - [ ] 6.3.5 Add graceful shutdown for health server

- [ ] 6.4 Add Kubernetes probe documentation
  - [ ] 6.4.1 Document liveness probe configuration in deploy manifests
  - [ ] 6.4.2 Set initialDelaySeconds: 10, periodSeconds: 10, failureThreshold: 3
  - [ ] 6.4.3 Document readiness probe configuration in deploy manifests
  - [ ] 6.4.4 Set initialDelaySeconds: 5, periodSeconds: 5, failureThreshold: 2

## 7. Graceful Shutdown Implementation

- [ ] 7.1 Implement signal handling
  - [ ] 7.1.1 Create `internal/shutdown/handler.go` file
  - [ ] 7.1.2 Use signal.NotifyContext for SIGTERM and SIGINT
  - [ ] 7.1.3 Create root context that cancels on signal
  - [ ] 7.1.4 Propagate context to all goroutines

- [ ] 7.2 Implement shutdown sequence
  - [ ] 7.2.1 Log "Shutting down gracefully" at INFO level
  - [ ] 7.2.2 Set shutdown flag (atomic.StoreInt32)
  - [ ] 7.2.3 Close all SSE client connections gracefully
  - [ ] 7.2.4 Stop accepting new events from queues

- [ ] 7.3 Implement agent drain logic
  - [ ] 7.3.1 Create shutdown context with timeout
  - [ ] 7.3.2 Track in-flight agent sessions with sync.WaitGroup
  - [ ] 7.3.3 Wait for WaitGroup with timeout
  - [ ] 7.3.4 Log which agents completed vs interrupted
  - [ ] 7.3.5 Cancel remaining agent contexts after timeout
  - [ ] 7.3.6 Send SIGKILL to remaining agent processes

- [ ] 7.4 Implement server cleanup
  - [ ] 7.4.1 Close metrics HTTP server with timeout
  - [ ] 7.4.2 Close health HTTP server with timeout
  - [ ] 7.4.3 Flush logs
  - [ ] 7.4.4 Exit with code 0

- [ ] 7.5 Add shutdown configuration
  - [ ] 7.5.1 Add SHUTDOWN_TIMEOUT_SECONDS env var (default 30)
  - [ ] 7.5.2 Add --shutdown-timeout CLI flag
  - [ ] 7.5.3 Validate timeout aligns with Kubernetes terminationGracePeriodSeconds

- [ ] 7.6 Add shutdown logging
  - [ ] 7.6.1 Log signal received with in-flight agent count
  - [ ] 7.6.2 Log each major shutdown step
  - [ ] 7.6.3 Log final shutdown summary (duration, agents completed/interrupted)

## 8. Instrumentation of Existing Code

- [ ] 8.1 Instrument event reception
  - [ ] 8.1.1 Add events_received_total increment in SSE event handler
  - [ ] 8.1.2 Include cluster and severity labels

- [ ] 8.2 Instrument event filtering
  - [ ] 8.2.1 Add events_filtered_total increment for filtered events
  - [ ] 8.2.2 Include cluster and reason labels (severity_low, duplicate, etc.)

- [ ] 8.3 Instrument queue operations
  - [ ] 8.3.1 Add events_queued_total increment when adding to queue
  - [ ] 8.3.2 Add events_dequeued_total increment when removing from queue
  - [ ] 8.3.3 Update queue_depth gauge on every queue modification

- [ ] 8.4 Instrument agent lifecycle
  - [ ] 8.4.1 Add agents_spawned_total increment when starting agent
  - [ ] 8.4.2 Increment agents_active gauge on start
  - [ ] 8.4.3 Decrement agents_active gauge on completion
  - [ ] 8.4.4 Add agents_completed_total increment with status label
  - [ ] 8.4.5 Record agent duration in histogram

- [ ] 8.5 Instrument circuit breaker
  - [ ] 8.5.1 Update circuit_breaker_state gauge on state changes
  - [ ] 8.5.2 Increment circuit_breaker_events_dropped_total for dropped events

## 9. Configuration and CLI

- [ ] 9.1 Create configuration struct
  - [ ] 9.1.1 Create `internal/config/config.go` file
  - [ ] 9.1.2 Define Config struct with all resilience parameters
  - [ ] 9.1.3 Add validation logic for config values

- [ ] 9.2 Implement environment variable parsing
  - [ ] 9.2.1 Read all *_SECONDS environment variables
  - [ ] 9.2.2 Read all port configuration variables
  - [ ] 9.2.3 Set defaults for unset variables

- [ ] 9.3 Implement CLI flag parsing
  - [ ] 9.3.1 Define all flags using flag or cobra package
  - [ ] 9.3.2 Implement flag precedence over env vars
  - [ ] 9.3.3 Add --version flag with version, commit, build date
  - [ ] 9.3.4 Add --help flag with all options documented

## 10. Testing

- [ ] 10.1 Unit tests for timeout logic
  - [ ] 10.1.1 Test timeout triggers after configured duration
  - [ ] 10.1.2 Test normal completion before timeout
  - [ ] 10.1.3 Test graceful process termination (SIGTERM then SIGKILL)
  - [ ] 10.1.4 Test metrics increment on timeout

- [ ] 10.2 Unit tests for queue expiration
  - [ ] 10.2.1 Test expiration of old events
  - [ ] 10.2.2 Test preservation of fresh events
  - [ ] 10.2.3 Test queue depth gauge updates
  - [ ] 10.2.4 Test sweeper respects context cancellation

- [ ] 10.3 Unit tests for backoff logic
  - [ ] 10.3.1 Test exponential backoff calculation
  - [ ] 10.3.2 Test jitter adds ±25% variation
  - [ ] 10.3.3 Test max delay cap at 60 seconds
  - [ ] 10.3.4 Test reset after successful connection

- [ ] 10.4 Unit tests for health endpoints
  - [ ] 10.4.1 Test /healthz returns 200 when healthy
  - [ ] 10.4.2 Test /healthz returns 503 when shutting down
  - [ ] 10.4.3 Test /readyz returns 200 when ready
  - [ ] 10.4.4 Test /readyz returns 503 when no SSE connections
  - [ ] 10.4.5 Test /readyz returns 503 when circuit breaker open

- [ ] 10.5 Unit tests for metrics
  - [ ] 10.5.1 Test all metrics are registered
  - [ ] 10.5.2 Test metric naming follows conventions
  - [ ] 10.5.3 Test label cardinality is low
  - [ ] 10.5.4 Test histogram buckets are correct

- [ ] 10.6 Integration tests
  - [ ] 10.6.1 Test SSE reconnection after server restart
  - [ ] 10.6.2 Test graceful shutdown with in-flight agents
  - [ ] 10.6.3 Test timeout handling end-to-end
  - [ ] 10.6.4 Test metrics endpoint scraping

- [ ] 10.7 Manual testing
  - [ ] 10.7.1 Deploy to test cluster
  - [ ] 10.7.2 Trigger agent timeout manually (long-running agent)
  - [ ] 10.7.3 Kill kubernetes-mcp-server and verify reconnection
  - [ ] 10.7.4 Send SIGTERM and verify graceful shutdown
  - [ ] 10.7.5 Scrape /metrics and verify Prometheus format
  - [ ] 10.7.6 Check Kubernetes liveness/readiness probe behavior

## 11. Documentation

- [ ] 11.1 Update operational documentation
  - [ ] 11.1.1 Document all new configuration parameters
  - [ ] 11.1.2 Document default values and recommended ranges
  - [ ] 11.1.3 Document metrics endpoint and format
  - [ ] 11.1.4 Document health endpoint behavior

- [ ] 11.2 Create metrics reference
  - [ ] 11.2.1 Document all metrics with descriptions
  - [ ] 11.2.2 Document label dimensions for each metric
  - [ ] 11.2.3 Provide example PromQL queries

- [ ] 11.3 Create deployment examples
  - [ ] 11.3.1 Add Kubernetes manifest example with probes
  - [ ] 11.3.2 Add Prometheus scrape config example
  - [ ] 11.3.3 Add example alert rules (high timeout rate, queue depth, etc.)

## 12. Verification

- [ ] 12.1 Verify timeout behavior
  - [ ] 12.1.1 Confirm agent is killed after configured timeout
  - [ ] 12.1.2 Confirm SIGTERM then SIGKILL sequence works
  - [ ] 12.1.3 Confirm timeout metric increments
  - [ ] 12.1.4 Confirm logs include timeout details

- [ ] 12.2 Verify queue expiration
  - [ ] 12.2.1 Confirm old events are dropped from queue
  - [ ] 12.2.2 Confirm fresh events remain in queue
  - [ ] 12.2.3 Confirm expiration metric increments
  - [ ] 12.2.4 Confirm queue depth gauge is accurate

- [ ] 12.3 Verify SSE reconnection
  - [ ] 12.3.1 Confirm reconnection after network error
  - [ ] 12.3.2 Confirm exponential backoff with jitter
  - [ ] 12.3.3 Confirm reconnection metrics are accurate
  - [ ] 12.3.4 Confirm logs show reconnection attempts and success

- [ ] 12.4 Verify metrics
  - [ ] 12.4.1 Confirm /metrics endpoint is accessible
  - [ ] 12.4.2 Confirm all expected metrics are present
  - [ ] 12.4.3 Confirm metrics update in real-time
  - [ ] 12.4.4 Confirm Prometheus can scrape successfully
  - [ ] 12.4.5 Confirm metric naming follows conventions
  - [ ] 12.4.6 Confirm label cardinality is reasonable

- [ ] 12.5 Verify health endpoints
  - [ ] 12.5.1 Confirm /healthz returns 200 when healthy
  - [ ] 12.5.2 Confirm /readyz returns 200 when ready
  - [ ] 12.5.3 Confirm Kubernetes probes work correctly
  - [ ] 12.5.4 Confirm pod is restarted on liveness failure
  - [ ] 12.5.5 Confirm pod removed from service on readiness failure

- [ ] 12.6 Verify graceful shutdown
  - [ ] 12.6.1 Confirm shutdown on SIGTERM
  - [ ] 12.6.2 Confirm in-flight agents complete within timeout
  - [ ] 12.6.3 Confirm remaining agents are killed after timeout
  - [ ] 12.6.4 Confirm clean exit with code 0
  - [ ] 12.6.5 Confirm logs show shutdown sequence
