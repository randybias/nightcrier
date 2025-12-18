# Implementation Tasks: Walking Skeleton

## Parallelization Guide

Tasks 1-4 can run in parallel (no dependencies).
Task 5 depends on all of 1-4 completing.

---

## 1. Project Skeleton and Configuration

- [x] 1.1 Initialize Go module: `go mod init github.com/rbias/kubernetes-mcp-alerts-event-runner`
- [x] 1.2 Create directory structure: `cmd/runner/`, `internal/config/`, `internal/events/`, `internal/agent/`, `internal/reporting/`
- [x] 1.3 Add dependencies: `github.com/r3labs/sse/v2`, `github.com/spf13/cobra`
- [x] 1.4 Create `internal/config/config.go` with Config struct (SSE_ENDPOINT, WORKSPACE_ROOT, LOG_LEVEL)
- [x] 1.5 Implement config loading from environment variables with defaults
- [x] 1.6 Create `cmd/runner/main.go` with cobra root command and `--sse-endpoint`, `--workspace-root` flags
- [x] 1.7 Verify: `go build ./cmd/runner` compiles

## 2. SSE Client and Event Parsing

- [x] 2.1 Define `FaultEvent` struct in `internal/events/event.go` with JSON tags (cluster_id, namespace, resource_type, resource_name, severity, message, timestamp, logs)
- [x] 2.2 Create `internal/events/client.go` with `Client` struct holding SSE client
- [x] 2.3 Implement `NewClient(endpoint string) *Client`
- [x] 2.4 Implement `Subscribe(ctx context.Context) (<-chan *FaultEvent, error)` that connects and returns event channel
- [x] 2.5 Parse SSE data field as JSON into FaultEvent
- [x] 2.6 Log received events at info level
- [x] 2.7 Handle connection errors with log and return

## 3. Workspace and Context

- [x] 3.1 Create `internal/agent/workspace.go` with `WorkspaceManager` struct
- [x] 3.2 Implement `NewWorkspaceManager(root string) *WorkspaceManager`
- [x] 3.3 Implement `Create(incidentID string) (string, error)` that creates `<root>/<incidentID>/` directory
- [x] 3.4 Set directory permissions to 0700
- [x] 3.5 Create `internal/agent/context.go` with `WriteEventContext(workspacePath string, event *FaultEvent) error`
- [x] 3.6 Write event as JSON to `<workspace>/event.json`

## 4. Agent Executor (Stub)

- [x] 4.1 Create `scripts/stub-agent.sh` that reads INCIDENT_ID env var and echoes "Processing incident: $INCIDENT_ID", exits 0
- [x] 4.2 Create `internal/agent/executor.go` with `Executor` struct
- [x] 4.3 Implement `NewExecutor(scriptPath string) *Executor`
- [x] 4.4 Implement `Execute(ctx context.Context, workspacePath string, incidentID string) (int, error)`
- [x] 4.5 Use exec.CommandContext with working directory set to workspace
- [x] 4.6 Set INCIDENT_ID environment variable
- [x] 4.7 Capture and log stdout/stderr
- [x] 4.8 Return exit code

## 5. Reporting (Minimal) and Main Wiring

**Depends on: Tasks 1-4**

- [x] 5.1 Create `internal/reporting/result.go` with `Result` struct (incident_id, exit_code, started_at, completed_at, status)
- [x] 5.2 Implement `WriteResult(workspacePath string, result *Result) error` that writes `result.json`
- [x] 5.3 Wire all components in `cmd/runner/main.go`:
  - Load config
  - Create SSE client
  - Create workspace manager
  - Create executor
  - Subscribe to events
  - For each event: create workspace -> write context -> execute -> write result
- [x] 5.4 Add graceful shutdown on SIGINT/SIGTERM
- [x] 5.5 Verify end-to-end: `go run ./cmd/runner --sse-endpoint <url>` processes events

## 6. Manual Integration Test

- [ ] 6.1 Start kubernetes-mcp-server (or mock SSE server)
- [ ] 6.2 Run skeleton: `go run ./cmd/runner --sse-endpoint http://localhost:8080/events`
- [ ] 6.3 Trigger fault event
- [ ] 6.4 Verify workspace directory created
- [ ] 6.5 Verify event.json contains event data
- [ ] 6.6 Verify result.json shows exit_code 0
