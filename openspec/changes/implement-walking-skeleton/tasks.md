# Implementation Tasks: Walking Skeleton

## Parallelization Guide

Tasks 1-4 can run in parallel (no dependencies).
Task 5 depends on all of 1-4 completing.

---

## 1. Project Skeleton and Configuration

- [ ] 1.1 Initialize Go module: `go mod init github.com/rbias/kubernetes-mcp-alerts-event-runner`
- [ ] 1.2 Create directory structure: `cmd/runner/`, `internal/config/`, `internal/events/`, `internal/agent/`, `internal/reporting/`
- [ ] 1.3 Add dependencies: `github.com/r3labs/sse/v2`, `github.com/spf13/cobra`
- [ ] 1.4 Create `internal/config/config.go` with Config struct (SSE_ENDPOINT, WORKSPACE_ROOT, LOG_LEVEL)
- [ ] 1.5 Implement config loading from environment variables with defaults
- [ ] 1.6 Create `cmd/runner/main.go` with cobra root command and `--sse-endpoint`, `--workspace-root` flags
- [ ] 1.7 Verify: `go build ./cmd/runner` compiles

## 2. SSE Client and Event Parsing

- [ ] 2.1 Define `FaultEvent` struct in `internal/events/event.go` with JSON tags (cluster_id, namespace, resource_type, resource_name, severity, message, timestamp, logs)
- [ ] 2.2 Create `internal/events/client.go` with `Client` struct holding SSE client
- [ ] 2.3 Implement `NewClient(endpoint string) *Client`
- [ ] 2.4 Implement `Subscribe(ctx context.Context) (<-chan *FaultEvent, error)` that connects and returns event channel
- [ ] 2.5 Parse SSE data field as JSON into FaultEvent
- [ ] 2.6 Log received events at info level
- [ ] 2.7 Handle connection errors with log and return

## 3. Workspace and Context

- [ ] 3.1 Create `internal/agent/workspace.go` with `WorkspaceManager` struct
- [ ] 3.2 Implement `NewWorkspaceManager(root string) *WorkspaceManager`
- [ ] 3.3 Implement `Create(incidentID string) (string, error)` that creates `<root>/<incidentID>/` directory
- [ ] 3.4 Set directory permissions to 0700
- [ ] 3.5 Create `internal/agent/context.go` with `WriteEventContext(workspacePath string, event *FaultEvent) error`
- [ ] 3.6 Write event as JSON to `<workspace>/event.json`

## 4. Agent Executor (Stub)

- [ ] 4.1 Create `scripts/stub-agent.sh` that reads INCIDENT_ID env var and echoes "Processing incident: $INCIDENT_ID", exits 0
- [ ] 4.2 Create `internal/agent/executor.go` with `Executor` struct
- [ ] 4.3 Implement `NewExecutor(scriptPath string) *Executor`
- [ ] 4.4 Implement `Execute(ctx context.Context, workspacePath string, incidentID string) (int, error)`
- [ ] 4.5 Use exec.CommandContext with working directory set to workspace
- [ ] 4.6 Set INCIDENT_ID environment variable
- [ ] 4.7 Capture and log stdout/stderr
- [ ] 4.8 Return exit code

## 5. Reporting (Minimal) and Main Wiring

**Depends on: Tasks 1-4**

- [ ] 5.1 Create `internal/reporting/result.go` with `Result` struct (incident_id, exit_code, started_at, completed_at, status)
- [ ] 5.2 Implement `WriteResult(workspacePath string, result *Result) error` that writes `result.json`
- [ ] 5.3 Wire all components in `cmd/runner/main.go`:
  - Load config
  - Create SSE client
  - Create workspace manager
  - Create executor
  - Subscribe to events
  - For each event: create workspace -> write context -> execute -> write result
- [ ] 5.4 Add graceful shutdown on SIGINT/SIGTERM
- [ ] 5.5 Verify end-to-end: `go run ./cmd/runner --sse-endpoint <url>` processes events

## 6. Manual Integration Test

- [ ] 6.1 Start kubernetes-mcp-server (or mock SSE server)
- [ ] 6.2 Run skeleton: `go run ./cmd/runner --sse-endpoint http://localhost:8080/events`
- [ ] 6.3 Trigger fault event
- [ ] 6.4 Verify workspace directory created
- [ ] 6.5 Verify event.json contains event data
- [ ] 6.6 Verify result.json shows exit_code 0
