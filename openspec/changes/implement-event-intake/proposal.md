# Change: Implement Event Intake (Phase 1)

## Walking Skeleton Baseline

The walking-skeleton implementation (archived 2025-12-18) provides the foundation that this change builds upon:

**Already Implemented:**
- Project skeleton: `cmd/runner/`, `internal/config/`, `internal/events/`, `internal/agent/`, `internal/reporting/`
- Cobra CLI framework with flags
- Configuration from environment variables (`K8S_CLUSTER_MCP_ENDPOINT`, `WORKSPACE_ROOT`, `LOG_LEVEL`, etc.)
- MCP client using StreamableHTTP protocol (supersedes raw SSE - connects to `/mcp` endpoint)
- FaultEvent struct matching kubernetes-mcp-server format
- Event subscription via `events_subscribe(mode="faults")`
- Graceful shutdown on SIGTERM/SIGINT
- Structured logging with `log/slog`
- Basic event processing loop

**This Change Adds:**
Advanced event intake features not in the walking skeleton: reconnection logic, severity filtering, deduplication, circuit breaker, per-cluster queuing, and queue overflow management.

## Why
To establish the foundation of the Event Runner by connecting to the `kubernetes-mcp-server`, filtering noise, and managing concurrency. This corresponds to Phase 1 of the high-level plan and enables automated triage of Kubernetes faults by AI agents.

The Event Runner serves as a critical control plane between the kubernetes-mcp-server (which observes cluster state) and AI agents (which investigate faults). Without proper event intake and rate limiting, the system could easily overwhelm clusters with concurrent agent activity or waste resources investigating low-severity issues.

## What Changes

### New Capability: `event-processing`
Implements the complete event intake pipeline with the following components:

#### MCP Client Enhancement (builds on walking skeleton)
- **DONE**: Connect to kubernetes-mcp-server via MCP StreamableHTTP protocol
- Automatic reconnection with exponential backoff (1s to 60s)
- Session recovery on reconnection
- Heartbeat detection and timeout handling
- Comprehensive connection error handling

#### Event Validation and Parsing (builds on walking skeleton)
- **DONE**: Parse JSON event payloads from MCP notifications
- **DONE**: FaultEvent struct with cluster, event, logs fields
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

#### Configuration System (builds on walking skeleton)
- **DONE**: Environment variables with sensible defaults
- **DONE**: Command-line flag overrides
- Optional YAML configuration file
- **DONE**: Validation at startup with clear error messages
- Configuration precedence: flags > env vars > file > defaults

#### Structured Logging (builds on walking skeleton)
- **DONE**: Comprehensive audit trail of all event decisions
- **DONE**: State transitions: received, validated, filtered, queued, processing, completed
- **DONE**: Structured log format with consistent field names (using log/slog)
- **DONE**: Configurable log levels

#### Graceful Shutdown (builds on walking skeleton)
- **DONE**: Signal handling (SIGTERM, SIGINT)
- Drain in-flight events with timeout
- **DONE**: Clean MCP connection closure
- Final statistics logging

### Project Structure (partially implemented)
```
cmd/runner/              # Main entry point - DONE
internal/
  config/                # Configuration loading and validation - DONE
  events/                # MCP client, event parsing, validation, filtering - DONE (basic)
  queue/                 # Circuit breaker, cluster queues, router, workers - TODO
  dedup/                 # Deduplication cache - TODO
  agent/                 # Agent execution - DONE (full impl from walking skeleton)
  reporting/             # Result recording, Slack notifications - DONE
  testing/mocksse/       # Mock MCP server for testing - TODO
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
- `K8S_CLUSTER_MCP_ENDPOINT` - URL of kubernetes-mcp-server MCP endpoint (**DONE**)

Already implemented (walking skeleton):
- `WORKSPACE_ROOT` (default: ./incidents) - **DONE**
- `LOG_LEVEL` (default: info) - **DONE**
- `AGENT_TIMEOUT` (default: 300) - **DONE**
- `AGENT_MODEL` (default: sonnet) - **DONE**
- `SLACK_WEBHOOK_URL` (optional) - **DONE**

Still needed for this change:
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