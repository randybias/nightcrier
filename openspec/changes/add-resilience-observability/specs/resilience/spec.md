## ADDED Requirements

### Requirement: Execution Timeout
The runner SHALL enforce a maximum execution time for any agent session.

#### Scenario: Agent hangs
- **WHEN** an agent runs longer than the configured timeout (e.g., 5 minutes)
- **THEN** the runner kills the process
- **AND** logs a timeout error with cluster and incident details
- **AND** increments the timeout counter metric
- **AND** records the execution duration in the histogram

#### Scenario: Normal completion before timeout
- **WHEN** an agent completes within the timeout period
- **THEN** the runner allows normal completion
- **AND** records success status in metrics
- **AND** does not trigger any timeout handling

#### Scenario: Graceful cleanup on timeout
- **WHEN** an agent times out
- **THEN** the runner sends SIGTERM first
- **AND** waits up to 5 seconds for graceful shutdown
- **AND** sends SIGKILL if process still running
- **AND** cleans up agent workspace resources

### Requirement: Queue Expiration
The runner SHALL discard queued events that are older than a specific threshold.

#### Scenario: Stale event
- **WHEN** an event sits in the per-cluster queue for longer than 10 minutes
- **THEN** it is removed from the queue without processing
- **AND** a warning is logged with cluster, event ID, and age
- **AND** the expired events counter metric is incremented
- **AND** the queue depth gauge is updated

#### Scenario: Queue sweep operation
- **WHEN** the queue sweeper runs every minute
- **THEN** it checks all queued events across all clusters
- **AND** removes events older than the configured max age
- **AND** preserves events that are still within the age limit
- **AND** updates metrics for each cluster's queue depth

#### Scenario: Fresh events preserved
- **WHEN** an event is younger than the max queue age
- **THEN** it remains in the queue
- **AND** is eligible for processing when the cluster worker becomes available

### Requirement: Metrics Emission
The runner SHALL expose Prometheus-compatible metrics for operational visibility.

#### Scenario: Event flow metrics
- **WHEN** events are received from SSE streams
- **THEN** the runner increments events_received_total counter with cluster and severity labels
- **AND** increments events_filtered_total if filtered before queuing
- **AND** increments events_queued_total when added to queue
- **AND** updates queue_depth gauge to reflect current queue size
- **AND** increments events_dequeued_total when removed for processing

#### Scenario: Agent lifecycle metrics
- **WHEN** an agent session starts
- **THEN** the runner increments agents_spawned_total counter
- **AND** increments agents_active gauge
- **WHEN** an agent session completes
- **THEN** the runner decrements agents_active gauge
- **AND** increments agents_completed_total with status label (success or failure)
- **AND** records execution duration in agent_duration_seconds histogram
- **WHEN** an agent times out
- **THEN** the runner increments agents_timeout_total counter

#### Scenario: Circuit breaker metrics
- **WHEN** the global circuit breaker opens
- **THEN** the runner sets circuit_breaker_state gauge to 1
- **AND** increments circuit_breaker_events_dropped_total for each rejected event
- **WHEN** the circuit breaker closes
- **THEN** the runner sets circuit_breaker_state gauge to 0

#### Scenario: SSE connection metrics
- **WHEN** an SSE connection is established
- **THEN** the runner increments sse_connections_active gauge
- **WHEN** an SSE connection closes
- **THEN** the runner decrements sse_connections_active gauge
- **AND** increments sse_reconnections_total with reason label
- **AND** records connection duration in sse_connection_duration_seconds histogram
- **WHEN** an SSE connection fails
- **THEN** the runner increments sse_connection_errors_total with reason label

#### Scenario: Metrics endpoint accessibility
- **WHEN** Prometheus scrapes the /metrics endpoint
- **THEN** the runner returns all metrics in Prometheus text format
- **AND** includes Go runtime metrics (memory, goroutines, GC)
- **AND** responds within 5 seconds

### Requirement: Metrics Naming Convention
The runner SHALL follow Prometheus best practices for metric naming and labeling.

#### Scenario: Metric naming standards
- **WHEN** defining any metric
- **THEN** the runner uses the prefix kubernetes_mcp_runner_
- **AND** uses snake_case for metric names
- **AND** includes base units in names (_seconds, _bytes, _total)
- **AND** uses _total suffix for counters

#### Scenario: Label cardinality control
- **WHEN** adding labels to metrics
- **THEN** the runner uses only low-cardinality labels
- **AND** includes cluster (10-100 unique values)
- **AND** includes severity (3-5 unique values: critical, high, medium, low, info)
- **AND** includes status (2-3 unique values: success, failure, timeout)
- **AND** includes reason (5-10 unique values for errors)
- **AND** MUST NOT use event IDs, timestamps, or resource names as labels

#### Scenario: Histogram bucket configuration
- **WHEN** configuring agent_duration_seconds histogram
- **THEN** the runner uses buckets [1, 5, 10, 30, 60, 120, 300] seconds
- **WHEN** configuring sse_connection_duration_seconds histogram
- **THEN** the runner uses buckets [60, 300, 600, 1800, 3600, 7200, 14400] seconds

### Requirement: SSE Reconnection
The runner SHALL automatically reconnect to SSE streams when connections fail.

#### Scenario: Network error triggers reconnection
- **WHEN** an SSE connection closes due to network error
- **THEN** the runner logs a warning with cluster and error details
- **AND** increments sse_reconnections_total counter
- **AND** waits for the initial backoff delay (1 second)
- **AND** attempts to reconnect

#### Scenario: Exponential backoff
- **WHEN** reconnection attempts fail repeatedly
- **THEN** the runner doubles the delay after each failure
- **AND** adds random jitter (±25%)
- **AND** caps the maximum delay at 60 seconds
- **AND** continues attempting until successful or context cancelled

#### Scenario: Successful reconnection
- **WHEN** an SSE connection is re-established
- **THEN** the runner resets the backoff delay to initial value
- **AND** logs success with cluster and total downtime
- **AND** resumes processing events from the stream

#### Scenario: Reconnection during shutdown
- **WHEN** the runner receives shutdown signal during reconnection backoff
- **THEN** the runner cancels the reconnection attempt immediately
- **AND** does not start new connections
- **AND** exits cleanly without waiting for max delay

#### Scenario: Connection duration tracking
- **WHEN** an SSE connection closes for any reason
- **THEN** the runner calculates the connection uptime
- **AND** records duration in sse_connection_duration_seconds histogram
- **AND** includes cluster label

### Requirement: Health Endpoint
The runner SHALL expose a health endpoint for liveness probes.

#### Scenario: Healthy state
- **WHEN** the /healthz endpoint is called
- **AND** the runner is not shutting down
- **THEN** the runner returns HTTP 200 OK
- **AND** responds within 1 second

#### Scenario: Shutdown state
- **WHEN** the /healthz endpoint is called
- **AND** the runner is shutting down
- **THEN** the runner returns HTTP 503 Service Unavailable
- **AND** includes "Shutting down" message in response body

#### Scenario: Liveness probe configuration
- **WHEN** deployed in Kubernetes
- **THEN** the liveness probe uses /healthz endpoint
- **AND** has initialDelaySeconds of 10
- **AND** has periodSeconds of 10
- **AND** has failureThreshold of 3

### Requirement: Readiness Endpoint
The runner SHALL expose a readiness endpoint for traffic management.

#### Scenario: Ready state
- **WHEN** the /readyz endpoint is called
- **AND** at least one SSE connection is active
- **AND** the circuit breaker is closed
- **THEN** the runner returns HTTP 200 OK
- **AND** responds within 1 second

#### Scenario: Not ready - no connections
- **WHEN** the /readyz endpoint is called
- **AND** no SSE connections are active
- **THEN** the runner returns HTTP 503 Service Unavailable
- **AND** includes "no active SSE connections" in response body

#### Scenario: Not ready - circuit breaker open
- **WHEN** the /readyz endpoint is called
- **AND** the global circuit breaker is open
- **THEN** the runner returns HTTP 503 Service Unavailable
- **AND** includes "circuit breaker is open" in response body

#### Scenario: Readiness probe configuration
- **WHEN** deployed in Kubernetes
- **THEN** the readiness probe uses /readyz endpoint
- **AND** has initialDelaySeconds of 5
- **AND** has periodSeconds of 5
- **AND** has failureThreshold of 2

### Requirement: Graceful Shutdown
The runner SHALL shut down cleanly without data loss when receiving termination signals.

#### Scenario: SIGTERM received
- **WHEN** the runner receives SIGTERM signal
- **THEN** the runner logs "Shutting down gracefully"
- **AND** marks readiness as false (returns 503 from /readyz)
- **AND** closes all SSE client connections
- **AND** stops accepting new events from queues
- **AND** waits for in-flight agent sessions to complete
- **AND** enforces shutdown timeout (default 30 seconds)
- **AND** cancels remaining agent contexts after timeout
- **AND** closes metrics endpoint
- **AND** exits with code 0

#### Scenario: Multiple agents during shutdown
- **WHEN** the runner is shutting down
- **AND** multiple agent sessions are in-flight
- **THEN** the runner waits for all agents up to shutdown timeout
- **AND** logs which agents completed successfully
- **AND** logs which agents were interrupted due to timeout
- **AND** allows agents up to their individual timeouts if within shutdown budget

#### Scenario: Force shutdown after timeout
- **WHEN** shutdown timeout expires
- **AND** some agent sessions are still running
- **THEN** the runner cancels all remaining agent contexts
- **AND** sends SIGKILL to agent processes
- **AND** logs warning with list of interrupted agents
- **AND** exits anyway to respect Kubernetes termination deadline

#### Scenario: SIGINT received (Ctrl+C)
- **WHEN** the runner receives SIGINT signal
- **THEN** the runner follows the same graceful shutdown sequence as SIGTERM
- **AND** exits cleanly

### Requirement: Configuration
The runner SHALL support configuration of resilience parameters via environment variables and CLI flags.

#### Scenario: Timeout configuration
- **WHEN** the AGENT_TIMEOUT_SECONDS environment variable is set
- **THEN** the runner uses that value for agent execution timeout
- **AND** defaults to 300 seconds (5 minutes) if not set
- **WHEN** the --agent-timeout flag is provided
- **THEN** the flag value overrides the environment variable

#### Scenario: Queue age configuration
- **WHEN** the MAX_QUEUE_AGE_SECONDS environment variable is set
- **THEN** the runner uses that value for queue expiration threshold
- **AND** defaults to 600 seconds (10 minutes) if not set
- **WHEN** the --max-queue-age flag is provided
- **THEN** the flag value overrides the environment variable

#### Scenario: Shutdown timeout configuration
- **WHEN** the SHUTDOWN_TIMEOUT_SECONDS environment variable is set
- **THEN** the runner uses that value for graceful shutdown timeout
- **AND** defaults to 30 seconds if not set
- **WHEN** the --shutdown-timeout flag is provided
- **THEN** the flag value overrides the environment variable

#### Scenario: Metrics port configuration
- **WHEN** the METRICS_PORT environment variable is set
- **THEN** the runner exposes /metrics on that port
- **AND** defaults to 9090 if not set
- **WHEN** the --metrics-port flag is provided
- **THEN** the flag value overrides the environment variable

#### Scenario: Health port configuration
- **WHEN** the HEALTH_PORT environment variable is set
- **THEN** the runner exposes /healthz and /readyz on that port
- **AND** defaults to 8080 if not set
- **WHEN** the --health-port flag is provided
- **THEN** the flag value overrides the environment variable

### Requirement: Error Recovery
The runner SHALL recover from transient errors without manual intervention.

#### Scenario: SSE stream error
- **WHEN** an SSE stream encounters a read error
- **THEN** the runner logs the error
- **AND** closes the connection gracefully
- **AND** initiates reconnection with backoff
- **AND** does not crash or exit

#### Scenario: Agent process crash
- **WHEN** an agent process exits with non-zero code
- **THEN** the runner logs the failure with exit code
- **AND** increments agents_completed_total with status=failure
- **AND** marks the incident as failed
- **AND** continues processing other events

#### Scenario: Metrics endpoint error
- **WHEN** the metrics HTTP handler encounters an error
- **THEN** the runner logs the error
- **AND** returns HTTP 500 to the client
- **AND** continues running (does not crash)

#### Scenario: Queue sweeper panic
- **WHEN** the queue sweeper goroutine panics
- **THEN** the runner logs the panic with stack trace
- **AND** recovers from the panic
- **AND** restarts the queue sweeper
- **AND** continues normal operation

### Requirement: Logging
The runner SHALL emit structured logs for debugging and audit purposes.

#### Scenario: Timeout event logging
- **WHEN** an agent times out
- **THEN** the runner logs at WARN level
- **AND** includes fields: cluster, incident_id, duration, timeout_threshold
- **AND** includes message "Agent execution timeout"

#### Scenario: Queue expiration logging
- **WHEN** events are expired from the queue
- **THEN** the runner logs at WARN level for each expired event
- **AND** includes fields: cluster, event_id, age, max_age
- **AND** includes message "Expired queued event"

#### Scenario: SSE reconnection logging
- **WHEN** SSE reconnection is attempted
- **THEN** the runner logs at WARN level
- **AND** includes fields: cluster, error, retry_in, attempt_number
- **AND** includes message "SSE connection failed, retrying"
- **WHEN** SSE reconnection succeeds
- **THEN** the runner logs at INFO level
- **AND** includes fields: cluster, downtime_duration
- **AND** includes message "SSE connection established"

#### Scenario: Graceful shutdown logging
- **WHEN** shutdown signal is received
- **THEN** the runner logs at INFO level
- **AND** includes fields: signal, in_flight_agents, shutdown_timeout
- **AND** includes message "Shutting down gracefully"
- **WHEN** shutdown completes
- **THEN** the runner logs at INFO level
- **AND** includes fields: duration, agents_completed, agents_interrupted
- **AND** includes message "Shutdown complete"

### Requirement: Build Information
The runner SHALL expose version and build metadata in metrics.

#### Scenario: Build info metric
- **WHEN** Prometheus scrapes the /metrics endpoint
- **THEN** the runner includes kubernetes_mcp_runner_build_info gauge
- **AND** the gauge has value 1
- **AND** includes label version (e.g., "1.2.3")
- **AND** includes label git_commit (e.g., "abc123def")
- **AND** includes label build_date (e.g., "2025-01-15T10:30:00Z")

#### Scenario: Version flag
- **WHEN** the runner is invoked with --version flag
- **THEN** it prints version information to stdout
- **AND** includes version number
- **AND** includes git commit hash
- **AND** includes build date
- **AND** exits with code 0
