# Implementation Tasks (Phase 1)

## 1. Project Skeleton and Dependencies
- [ ] 1.1 Initialize Go module with `go.mod` and `go.sum`
- [ ] 1.2 Scaffold directory structure: `cmd/runner/`, `internal/config/`, `internal/events/`, `internal/queue/`, `internal/dedup/`, `internal/agent/`
- [ ] 1.3 Add dependency: `github.com/r3labs/sse/v2` for SSE client
- [ ] 1.4 Add dependency: `github.com/spf13/cobra` for CLI framework
- [ ] 1.5 Add dependency: `github.com/spf13/viper` for configuration management
- [ ] 1.6 Add dependency: structured logging library (e.g., `go.uber.org/zap` or `log/slog`)
- [ ] 1.7 Create `cmd/runner/main.go` with basic command structure
- [ ] 1.8 Verify project builds with `go build ./cmd/runner`

## 2. Configuration System
- [ ] 2.1 Define configuration struct in `internal/config/config.go` with all fields from design.md
- [ ] 2.2 Implement environment variable loading with defaults
- [ ] 2.3 Implement command-line flag parsing using cobra
- [ ] 2.4 Implement configuration precedence (flags > env vars > defaults)
- [ ] 2.5 Add validation for required fields (SSE_ENDPOINT)
- [ ] 2.6 Add validation for numeric ranges (queue sizes >= 1, max agents >= 1)
- [ ] 2.7 Add validation for severity threshold against allowed values
- [ ] 2.8 Implement configuration logging at startup (mask sensitive values)
- [ ] 2.9 Write unit tests for configuration loading and validation
- [ ] 2.10 Add optional YAML configuration file support
- [ ] 2.11 Document all configuration options in README or config example file

## 3. Event Data Structures
- [ ] 3.1 Define `FaultEvent` struct in `internal/events/event.go`
- [ ] 3.2 Add JSON struct tags for all FaultEvent fields
- [ ] 3.3 Define `DeduplicationKey` struct
- [ ] 3.4 Implement `DeduplicationKey.String()` method for cache key generation
- [ ] 3.5 Define severity constants (DEBUG, INFO, WARNING, ERROR, CRITICAL)
- [ ] 3.6 Implement severity comparison function (returns -1, 0, 1)
- [ ] 3.7 Write unit tests for severity comparison logic
- [ ] 3.8 Write unit tests for deduplication key generation

## 4. SSE Client Implementation
- [ ] 4.1 Create `internal/events/client.go` for SSE client wrapper
- [ ] 4.2 Initialize r3labs/sse client with configured endpoint
- [ ] 4.3 Configure exponential backoff (1s initial, 60s max)
- [ ] 4.4 Configure Last-Event-ID tracking
- [ ] 4.5 Implement event subscription with context support
- [ ] 4.6 Add connection success logging
- [ ] 4.7 Add connection failure logging with error details
- [ ] 4.8 Implement reconnection logging with backoff duration
- [ ] 4.9 Handle SSE event data field extraction
- [ ] 4.10 Parse event ID field for Last-Event-ID tracking
- [ ] 4.11 Implement graceful shutdown on context cancellation
- [ ] 4.12 Write integration test with mock SSE server
- [ ] 4.13 Test reconnection behavior with temporary server failure
- [ ] 4.14 Test Last-Event-ID header transmission on reconnect

## 5. Event Validation and Parsing
- [ ] 5.1 Create `internal/events/validator.go`
- [ ] 5.2 Implement JSON parsing from SSE event data
- [ ] 5.3 Validate required fields (cluster_id, severity, resource_name)
- [ ] 5.4 Validate severity value against allowed constants
- [ ] 5.5 Log and discard malformed events with error details
- [ ] 5.6 Log and discard events with missing required fields
- [ ] 5.7 Add metrics counter for malformed events (placeholder for Phase 4)
- [ ] 5.8 Write unit tests for valid event parsing
- [ ] 5.9 Write unit tests for malformed JSON handling
- [ ] 5.10 Write unit tests for missing required fields
- [ ] 5.11 Write unit tests for invalid severity values

## 6. Severity Filtering
- [ ] 6.1 Create `internal/events/filter.go`
- [ ] 6.2 Implement severity threshold comparison
- [ ] 6.3 Return boolean indicating if event passes threshold
- [ ] 6.4 Log filtered events at debug level with event details
- [ ] 6.5 Write unit tests for each severity level combination
- [ ] 6.6 Write unit tests for boundary cases

## 7. Deduplication Cache
- [ ] 7.1 Create `internal/dedup/cache.go`
- [ ] 7.2 Implement in-memory cache with time-based expiry
- [ ] 7.3 Use sync.Map or mutex-protected map for thread safety
- [ ] 7.4 Implement `Check(key DeduplicationKey) bool` method
- [ ] 7.5 Implement `Add(key DeduplicationKey, window time.Duration)` method
- [ ] 7.6 Implement periodic cleanup of expired entries
- [ ] 7.7 Log duplicate detection at debug level
- [ ] 7.8 Write unit tests for cache operations
- [ ] 7.9 Write unit tests for expiry behavior
- [ ] 7.10 Write concurrent access tests
- [ ] 7.11 Add memory usage monitoring (log cache size periodically)

## 8. Global Circuit Breaker
- [ ] 8.1 Create `internal/queue/circuitbreaker.go`
- [ ] 8.2 Define `CircuitBreaker` struct with buffered channel semaphore
- [ ] 8.3 Implement `New(limit int)` constructor
- [ ] 8.4 Implement `TryAcquire() bool` non-blocking acquire
- [ ] 8.5 Implement `Release()` method
- [ ] 8.6 Add atomic counter for current active count (for logging)
- [ ] 8.7 Log circuit breaker state changes (opened, closed)
- [ ] 8.8 Log warnings when capacity is reached
- [ ] 8.9 Write unit tests for acquire/release operations
- [ ] 8.10 Write unit tests for capacity limits
- [ ] 8.11 Write concurrent access tests

## 9. Cluster-Specific Queues
- [ ] 9.1 Create `internal/queue/clusterqueue.go`
- [ ] 9.2 Define `ClusterQueue` struct with buffered channel and metadata
- [ ] 9.3 Implement `New(clusterID string, size int)` constructor
- [ ] 9.4 Implement `Enqueue(event *FaultEvent, policy string) error` method
- [ ] 9.5 Handle "drop" overflow policy (drop oldest from channel)
- [ ] 9.6 Handle "reject" overflow policy (return error immediately)
- [ ] 9.7 Log queuing actions with queue depth
- [ ] 9.8 Log overflow events with dropped event details
- [ ] 9.9 Implement `Dequeue() <-chan *FaultEvent` to return receive channel
- [ ] 9.10 Write unit tests for enqueue/dequeue operations
- [ ] 9.11 Write unit tests for overflow policies
- [ ] 9.12 Write tests for queue depth tracking

## 10. Event Router
- [ ] 10.1 Create `internal/queue/router.go`
- [ ] 10.2 Define `Router` struct with map of cluster ID to ClusterQueue
- [ ] 10.3 Implement thread-safe map access with mutex
- [ ] 10.4 Implement `GetOrCreateQueue(clusterID string) *ClusterQueue` method
- [ ] 10.5 Implement `Route(event *FaultEvent, policy string) error` method
- [ ] 10.6 Log cluster queue creation
- [ ] 10.7 Track number of active cluster queues
- [ ] 10.8 Write unit tests for queue creation and lookup
- [ ] 10.9 Write concurrent access tests
- [ ] 10.10 Write tests for multiple clusters

## 11. Cluster Worker Implementation
- [ ] 11.1 Create `internal/queue/worker.go`
- [ ] 11.2 Implement `StartWorker(ctx context.Context, queue *ClusterQueue, handler func(*FaultEvent))` function
- [ ] 11.3 Start goroutine that reads from queue channel
- [ ] 11.4 Implement panic recovery with logging and worker restart
- [ ] 11.5 Handle context cancellation for graceful shutdown
- [ ] 11.6 Log worker start with cluster ID
- [ ] 11.7 Log worker shutdown
- [ ] 11.8 Log event processing start with event details
- [ ] 11.9 Write unit tests for worker lifecycle
- [ ] 11.10 Write tests for panic recovery
- [ ] 11.11 Write tests for graceful shutdown

## 12. Agent Spawner Stub
- [ ] 12.1 Create `internal/agent/spawner.go`
- [ ] 12.2 Define `Spawner` interface with `Spawn(event *FaultEvent) error` method
- [ ] 12.3 Implement stub that logs event details and returns nil
- [ ] 12.4 Add configurable sleep duration to simulate agent execution
- [ ] 12.5 Log "agent started" and "agent completed" messages
- [ ] 12.6 Note: Full agent implementation is in Phase 2

## 13. Main Event Processing Loop
- [ ] 13.1 Create `internal/events/processor.go`
- [ ] 13.2 Define `Processor` struct that coordinates all components
- [ ] 13.3 Initialize SSE client, validator, filter, dedup cache, circuit breaker, router
- [ ] 13.4 Implement `Start(ctx context.Context) error` method
- [ ] 13.5 Subscribe to SSE events in goroutine
- [ ] 13.6 For each event: validate, filter, dedup, check circuit breaker, route
- [ ] 13.7 On successful routing, ensure worker exists for cluster
- [ ] 13.8 Pass event to worker via cluster queue
- [ ] 13.9 Handle all error cases with appropriate logging
- [ ] 13.10 Implement graceful shutdown on context cancellation
- [ ] 13.11 Write integration tests with mock SSE server
- [ ] 13.12 Test end-to-end flow from SSE event to agent spawner
- [ ] 13.13 Test error paths (validation failures, filtering, dedup)

## 14. Graceful Shutdown
- [ ] 14.1 Implement signal handling (SIGTERM, SIGINT) in main.go
- [ ] 14.2 Create shutdown context with configured timeout
- [ ] 14.3 Stop accepting new SSE events
- [ ] 14.4 Close cluster queues to signal workers
- [ ] 14.5 Wait for in-flight agents to complete (with timeout)
- [ ] 14.6 Log shutdown progress at each stage
- [ ] 14.7 If timeout expires, forcibly terminate remaining agents
- [ ] 14.8 Log final statistics (events processed, queued, dropped)
- [ ] 14.9 Close SSE connection cleanly
- [ ] 14.10 Exit with appropriate status code (0 for clean, 1 for timeout)
- [ ] 14.11 Write tests for graceful shutdown
- [ ] 14.12 Write tests for forced shutdown on timeout

## 15. Structured Logging
- [ ] 15.1 Initialize structured logger in main.go
- [ ] 15.2 Configure log level from configuration
- [ ] 15.3 Define consistent field names for structured logs
- [ ] 15.4 Use structured logging throughout codebase
- [ ] 15.5 Log all event state transitions with appropriate fields
- [ ] 15.6 Add request ID or correlation ID to logs (optional)
- [ ] 15.7 Test log output format and levels

## 16. Integration Testing
- [ ] 16.1 Create mock SSE server in `internal/testing/mocksse/`
- [ ] 16.2 Implement mock server that emits test events
- [ ] 16.3 Support reconnection testing (server restart)
- [ ] 16.4 Write integration test: successful event processing
- [ ] 16.5 Write integration test: severity filtering
- [ ] 16.6 Write integration test: deduplication
- [ ] 16.7 Write integration test: circuit breaker at capacity
- [ ] 16.8 Write integration test: per-cluster concurrency (queue buildup)
- [ ] 16.9 Write integration test: SSE reconnection
- [ ] 16.10 Write integration test: graceful shutdown
- [ ] 16.11 Write integration test: queue overflow with drop policy
- [ ] 16.12 Write integration test: queue overflow with reject policy

## 17. Documentation
- [ ] 17.1 Write README.md with project overview
- [ ] 17.2 Document all configuration options with examples
- [ ] 17.3 Document SSE event payload structure
- [ ] 17.4 Add architecture diagram to docs
- [ ] 17.5 Document how to run the runner locally
- [ ] 17.6 Document how to test with mock SSE server
- [ ] 17.7 Add troubleshooting section
- [ ] 17.8 Document logging format and key fields

## 18. Verification and Polish
- [ ] 18.1 Run `go fmt` on all code
- [ ] 18.2 Run `go vet` and fix issues
- [ ] 18.3 Run `golangci-lint` if available and fix issues
- [ ] 18.4 Verify all unit tests pass
- [ ] 18.5 Verify all integration tests pass
- [ ] 18.6 Test runner startup with invalid configuration
- [ ] 18.7 Test runner startup with valid configuration
- [ ] 18.8 Test end-to-end with real kubernetes-mcp-server (if available)
- [ ] 18.9 Review all log output for clarity and completeness
- [ ] 18.10 Check memory usage under load (optional performance test)
- [ ] 18.11 Check goroutine count for leaks
- [ ] 18.12 Update tasks.md to mark all items complete