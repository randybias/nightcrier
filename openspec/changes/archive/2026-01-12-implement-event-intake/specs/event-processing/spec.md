## ADDED Requirements

### Requirement: SSE Event Subscription
The runner SHALL subscribe to the `kubernetes-mcp-server` via Server-Sent Events (SSE) to receive Kubernetes fault events.

#### Scenario: Successful connection
- **WHEN** the runner starts
- **THEN** it connects to the configured SSE endpoint
- **AND** listens for "fault" type events
- **AND** logs the connection establishment

#### Scenario: Initial connection failure
- **WHEN** the runner attempts to connect to the SSE endpoint
- **AND** the connection fails
- **THEN** the runner retries with exponential backoff
- **AND** logs the connection error with details
- **AND** continues retry attempts until successful or context is cancelled

### Requirement: SSE Reconnection Strategy
The runner SHALL implement automatic reconnection with exponential backoff when the SSE connection is lost.

#### Scenario: Connection drop during operation
- **WHEN** an established SSE connection is lost
- **THEN** the runner initiates a reconnection attempt
- **AND** uses exponential backoff starting at 1 second
- **AND** caps the maximum backoff at 60 seconds
- **AND** sends the Last-Event-ID header to resume from the last processed event

#### Scenario: Server sends retry time
- **WHEN** the SSE server sends a retry field
- **THEN** the runner uses the server-specified reconnection time
- **AND** overrides the default backoff strategy for the next reconnection

#### Scenario: Maximum reconnection attempts
- **WHEN** reconnection attempts exceed a configured threshold
- **THEN** the runner logs a critical error
- **AND** continues attempting reconnection indefinitely unless context is cancelled

### Requirement: SSE Event Payload Structure
The runner SHALL parse SSE events with a defined structure containing fault metadata.

#### Scenario: Valid event payload
- **WHEN** an SSE event is received
- **THEN** the runner parses the event data as JSON
- **AND** extracts fields: cluster_id, namespace, resource_type, resource_name, severity, timestamp, message, and event_id
- **AND** validates that required fields are present

#### Scenario: Malformed event payload
- **WHEN** an SSE event contains invalid JSON
- **THEN** the runner logs a parsing error
- **AND** discards the event
- **AND** continues processing subsequent events

#### Scenario: Missing required fields
- **WHEN** an SSE event is missing required fields (cluster_id, severity, or resource_name)
- **THEN** the runner logs a validation error
- **AND** discards the event
- **AND** increments a metric for malformed events

### Requirement: Event Filtering
The runner SHALL ignore events that do not meet the configured severity threshold.

#### Scenario: Low severity drop
- **WHEN** an event with severity "INFO" is received
- **AND** the threshold is "ERROR"
- **THEN** the event is discarded without action
- **AND** a debug-level log entry is recorded

#### Scenario: Severity level comparison
- **WHEN** comparing severity levels
- **THEN** the runner uses the ordering: DEBUG < INFO < WARNING < ERROR < CRITICAL
- **AND** only processes events at or above the configured threshold

### Requirement: Severity Configuration
The runner SHALL support configurable severity thresholds via environment variables and command-line flags.

#### Scenario: Environment variable configuration
- **WHEN** the SEVERITY_THRESHOLD environment variable is set
- **THEN** the runner uses that value as the minimum severity
- **AND** validates the value against allowed severity levels

#### Scenario: Command-line flag override
- **WHEN** both environment variable and command-line flag are set
- **THEN** the command-line flag takes precedence
- **AND** the effective configuration is logged at startup

#### Scenario: Invalid severity value
- **WHEN** an invalid severity value is provided
- **THEN** the runner fails to start
- **AND** logs a clear error message indicating valid values

### Requirement: Global Circuit Breaker
The runner SHALL strictly limit the total number of concurrent active agent sessions across all clusters.

#### Scenario: Max capacity reached
- **WHEN** the number of active agents equals the configured limit (e.g., 5)
- **AND** a new eligible event arrives
- **THEN** the new event is queued or dropped (depending on configuration)
- **AND** a warning is logged

#### Scenario: Capacity available
- **WHEN** the number of active agents is below the limit
- **AND** an eligible event arrives
- **THEN** the event is immediately processed
- **AND** the active agent count is incremented

#### Scenario: Agent completion
- **WHEN** an agent completes its processing
- **THEN** the active agent count is decremented
- **AND** the next queued event (if any) is processed
- **AND** the circuit breaker state is logged

### Requirement: Circuit Breaker Configuration
The runner SHALL support configurable global agent limits via environment variables and command-line flags.

#### Scenario: Default configuration
- **WHEN** no configuration is provided
- **THEN** the runner uses a default limit of 5 concurrent agents
- **AND** logs the effective limit at startup

#### Scenario: Configuration validation
- **WHEN** the configured limit is less than 1
- **THEN** the runner fails to start
- **AND** logs an error indicating the minimum valid value

### Requirement: Per-Cluster Concurrency
The runner SHALL ensure only one active agent session exists per Kubernetes cluster at any given time.

#### Scenario: Cluster busy
- **WHEN** an agent is already running for "cluster-A"
- **AND** a new fault arrives for "cluster-A"
- **THEN** the new fault is queued in the cluster-specific queue
- **AND** the agent is not spawned immediately
- **AND** a debug log entry is recorded

#### Scenario: Cluster available
- **WHEN** no agent is running for "cluster-B"
- **AND** a fault arrives for "cluster-B"
- **THEN** an agent is immediately spawned
- **AND** the cluster is marked as busy

#### Scenario: Cluster queue processing
- **WHEN** an agent completes for "cluster-A"
- **AND** the cluster-A queue is not empty
- **THEN** the next event from the cluster-A queue is immediately processed
- **AND** the cluster remains marked as busy

### Requirement: Event Deduplication
The runner SHALL prevent duplicate processing of events for the same resource within a time window.

#### Scenario: Duplicate event received
- **WHEN** an event is received for resource "pod/frontend" in namespace "production"
- **AND** an agent is already processing or has recently processed the same resource
- **THEN** the duplicate event is discarded
- **AND** a debug log entry is recorded

#### Scenario: Deduplication window expiry
- **WHEN** an event is received for a previously processed resource
- **AND** the deduplication window (e.g., 5 minutes) has expired
- **THEN** the event is processed as a new incident
- **AND** the deduplication cache is updated

#### Scenario: Different resources, same namespace
- **WHEN** events are received for "pod/frontend" and "pod/backend" in namespace "production"
- **THEN** both events are processed independently
- **AND** no deduplication occurs

### Requirement: Queue Overflow Handling
The runner SHALL implement bounded queues with configurable overflow behavior.

#### Scenario: Queue full with drop policy
- **WHEN** the global queue is full
- **AND** the overflow policy is "drop"
- **AND** a new event arrives
- **THEN** the oldest event in the queue is dropped
- **AND** an error is logged with dropped event details
- **AND** a metric for dropped events is incremented

#### Scenario: Queue full with reject policy
- **WHEN** a cluster-specific queue is full
- **AND** the overflow policy is "reject"
- **AND** a new event arrives for that cluster
- **THEN** the new event is rejected immediately
- **AND** an error is logged

#### Scenario: Queue size configuration
- **WHEN** queue size limits are configured
- **THEN** global queue size and per-cluster queue sizes are validated
- **AND** sizes must be at least 1
- **AND** default values are used if not specified

### Requirement: Event Logging and Audit Trail
The runner SHALL log all event state transitions for observability and debugging.

#### Scenario: Event received
- **WHEN** an SSE event is received
- **THEN** the runner logs the event with timestamp, event_id, cluster_id, resource, and severity
- **AND** uses structured logging with consistent field names

#### Scenario: Event filtered
- **WHEN** an event is filtered due to severity threshold
- **THEN** the runner logs the filtering decision at debug level
- **AND** includes the event severity and configured threshold

#### Scenario: Event queued
- **WHEN** an event is placed in a queue
- **THEN** the runner logs the queuing action
- **AND** includes the queue type (global or cluster-specific)
- **AND** includes the current queue depth

#### Scenario: Event processing started
- **WHEN** an agent begins processing an event
- **THEN** the runner logs the start of processing
- **AND** includes agent identifier and workspace path

#### Scenario: Event processing completed
- **WHEN** an agent completes processing
- **THEN** the runner logs completion with duration
- **AND** includes exit status and any error information

### Requirement: Configuration Management
The runner SHALL load configuration from environment variables, command-line flags, and optional configuration files.

#### Scenario: Configuration precedence
- **WHEN** the same configuration key is set in multiple sources
- **THEN** command-line flags override environment variables
- **AND** environment variables override configuration file values
- **AND** configuration file values override built-in defaults

#### Scenario: Required configuration
- **WHEN** the runner starts
- **THEN** it validates that SSE_ENDPOINT is configured
- **AND** fails to start if required configuration is missing
- **AND** logs clear error messages for missing values

#### Scenario: Configuration validation at startup
- **WHEN** the runner starts
- **THEN** it validates all configuration values
- **AND** logs the effective configuration (with sensitive values masked)
- **AND** fails fast if any values are invalid

### Requirement: Graceful Shutdown
The runner SHALL handle shutdown signals gracefully and drain in-flight events.

#### Scenario: SIGTERM received
- **WHEN** the runner receives a SIGTERM signal
- **THEN** it stops accepting new events from SSE
- **AND** allows in-flight agents to complete (with timeout)
- **AND** logs shutdown progress

#### Scenario: Shutdown timeout
- **WHEN** in-flight agents do not complete within the shutdown timeout
- **THEN** the runner forcibly terminates remaining agents
- **AND** logs which agents were terminated
- **AND** exits with appropriate status code

#### Scenario: Clean shutdown
- **WHEN** all in-flight agents complete during shutdown
- **THEN** the runner closes the SSE connection cleanly
- **AND** logs final statistics (events processed, queued, dropped)
- **AND** exits with status code 0
