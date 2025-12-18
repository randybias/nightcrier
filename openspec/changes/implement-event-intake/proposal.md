# Change: Implement Event Intake (Phase 1)

## Why
To establish the foundation of the Event Runner by connecting to the `kubernetes-mcp-server`, filtering noise, and managing concurrency. This corresponds to Phase 1 of the high-level plan and enables automated triage of Kubernetes faults by AI agents.

The Event Runner serves as a critical control plane between the kubernetes-mcp-server (which observes cluster state) and AI agents (which investigate faults). Without proper event intake and rate limiting, the system could easily overwhelm clusters with concurrent agent activity or waste resources investigating low-severity issues.

## What Changes

### New Capability: `event-processing`
Implements the complete event intake pipeline with the following components:

#### SSE Client Integration
- Connect to kubernetes-mcp-server via Server-Sent Events (SSE)
- Automatic reconnection with exponential backoff (1s to 60s)
- Last-Event-ID tracking for resumable connections
- Heartbeat detection and timeout handling
- Comprehensive connection error handling

#### Event Validation and Parsing
- Parse JSON event payloads from SSE stream
- Validate required fields (cluster_id, severity, resource_name)
- Handle malformed events gracefully
- Extract event metadata for routing and processing

#### Severity-Based Filtering
- Configurable severity threshold (DEBUG, INFO, WARNING, ERROR, CRITICAL)
- Drop events below threshold to reduce noise
- Debug logging for filtered events

#### Deduplication System
- Time-window-based deduplication (5-minute default)
- Prevent redundant processing of same resource+namespace
- In-memory cache with automatic expiry
- Thread-safe concurrent access

#### Global Circuit Breaker
- Hard limit on total concurrent agents across all clusters (default: 5)
- Semaphore-based implementation using buffered channels
- Prevents agent storms and resource exhaustion
- Configurable capacity

#### Per-Cluster Concurrency Control
- Strictly one agent per cluster at any time
- Buffered event queues per cluster
- Dedicated worker goroutine per cluster
- Prevents race conditions on cluster API

#### Queue Overflow Management
- Bounded queues with configurable policies (drop/reject)
- Drop policy: remove oldest event when full
- Reject policy: immediately reject new events when full
- Comprehensive logging of overflow events

#### Configuration System
- Environment variables with sensible defaults
- Command-line flag overrides
- Optional YAML configuration file
- Validation at startup with clear error messages
- Configuration precedence: flags > env vars > file > defaults

#### Structured Logging
- Comprehensive audit trail of all event decisions
- State transitions: received, validated, filtered, queued, processing, completed
- Structured log format with consistent field names
- Configurable log levels

#### Graceful Shutdown
- Signal handling (SIGTERM, SIGINT)
- Drain in-flight events with timeout
- Clean SSE connection closure
- Final statistics logging

### Project Structure
```
cmd/runner/              # Main entry point
internal/
  config/                # Configuration loading and validation
  events/                # SSE client, event parsing, validation, filtering
  queue/                 # Circuit breaker, cluster queues, router, workers
  dedup/                 # Deduplication cache
  agent/                 # Agent spawner stub (full impl in Phase 2)
  testing/mocksse/       # Mock SSE server for testing
```

## Impact

### New Capabilities
- `event-processing` - Complete event intake and routing system

### New Code
- Core runner skeleton with CLI framework (cobra)
- Configuration system (viper)
- SSE client wrapper (r3labs/sse)
- Event validation and filtering logic
- Deduplication cache with TTL
- Global circuit breaker (semaphore pattern)
- Per-cluster queuing and worker management
- Event router with dynamic cluster queue creation
- Structured logging throughout
- Graceful shutdown handling
- Comprehensive unit and integration tests
- Mock SSE server for testing

### Dependencies
- **Runtime**: Requires running `kubernetes-mcp-server` with SSE endpoint
- **Go Libraries**:
  - `github.com/r3labs/sse/v2` - SSE client with reconnection
  - `github.com/spf13/cobra` - CLI framework
  - `github.com/spf13/viper` - Configuration management
  - Structured logging library (zap or log/slog)

### Configuration Requirements
Minimum required configuration:
- `SSE_ENDPOINT` - URL of kubernetes-mcp-server SSE endpoint

Optional configuration with defaults:
- `SEVERITY_THRESHOLD` (default: ERROR)
- `MAX_CONCURRENT_AGENTS` (default: 5)
- `GLOBAL_QUEUE_SIZE` (default: 100)
- `CLUSTER_QUEUE_SIZE` (default: 10)
- `DEDUP_WINDOW_SECONDS` (default: 300)
- See design.md for complete configuration schema

### Testing Requirements
- Unit tests for all core components
- Integration tests with mock SSE server
- Test coverage for error paths and edge cases
- Concurrent access tests for thread safety
- Graceful shutdown tests

### Non-Breaking Changes
This is a new capability with no impact on existing code. There is no existing codebase to maintain compatibility with.

### Future Phases
This change provides the foundation for:
- **Phase 2**: Agent runtime (workspace creation, subprocess management)
- **Phase 3**: Reporting and notifications (Slack integration)
- **Phase 4**: Resilience and observability (metrics, alerting, persistence)