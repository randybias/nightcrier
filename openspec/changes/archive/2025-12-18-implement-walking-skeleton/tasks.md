# Implementation Tasks: Walking Skeleton

## Parallelization Guide

Tasks 1-4 can run in parallel (no dependencies).
Task 5 depends on all of 1-4 completing.

---

## 1. Project Skeleton and Configuration

- [x] 1.1 Initialize Go module: `go mod init github.com/rbias/nightcrier`
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

- [x] 6.1 Start kubernetes-mcp-server on port 8383
- [x] 6.2 Run skeleton: `./runner --mcp-endpoint http://localhost:8383`
- [x] 6.3 Trigger fault event (create failing pod)
- [x] 6.4 Verify workspace directory created
- [x] 6.5 Verify event.json contains event data
- [x] 6.6 Verify result.json shows exit_code 0

## Implementation Notes (2025-12-18)

### Key Changes from Original Plan

1. **Transport**: Changed from raw SSE (`/sse` endpoint) to MCP StreamableHTTP (`/mcp` endpoint)
   - Uses `mcp.StreamableClientTransport` instead of `mcp.SSEClientTransport`
   - Required for proper session tracking and notification delivery

2. **Event Subscription**: Uses MCP protocol instead of raw SSE
   - Calls `events_subscribe` tool with `mode: "faults"`
   - Receives events via `logging/message` notifications with `logger: "kubernetes/faults"`
   - Requires `session.Wait()` goroutine to keep connection alive

3. **Event Structure**: Matches kubernetes-mcp-server `FaultEvent` format
   - `subscriptionId`, `cluster`, `event`, `logs[]`
   - `event.involvedObject` contains Pod details
   - Container logs enriched by server

4. **Executor Fix**: Script path converted to absolute path
   - Required because `cmd.Dir` changes working directory to workspace

### Test Results

```
Incident ID: 5cc31391-ca70-4550-87a2-b97e50b37031
Pod: final-test-1766040896 (StartError)
Status: success
Exit Code: 0
Duration: 35ms
```

### Files Modified

- `internal/events/client.go` - MCP client with StreamableHTTP transport
- `internal/events/event.go` - FaultEvent struct matching server format
- `internal/agent/executor.go` - Absolute path fix, pass args
- `internal/config/config.go` - K8S_CLUSTER_MCP_ENDPOINT environment variable
- `cmd/runner/main.go` - Updated to use new event structure
- `scripts/stub-agent.sh` - Created test stub script

## 7. Real Agent Integration (Extension)

**Purpose**: Replace stub agent with containerized Claude agent to prove real triage value.

- [x] 7.1 Create `configs/` directory for prompt templates
- [x] 7.2 Create `configs/triage-system-prompt.md` with agent constraints and output format
- [x] 7.3 Create `configs/triage-prompt.md` with production incident framing
- [x] 7.4 Test containerized agent manually with `agent-container/run-agent.sh`
- [x] 7.5 Verify investigation report generated in `output/investigation.md`
- [x] 7.6 Update `internal/config/config.go` to add agent configuration:
  - `SLACK_WEBHOOK_URL` (optional, for notifications)
  - `AGENT_SCRIPT_PATH` (path to run-agent.sh)
  - `AGENT_TIMEOUT` (default: 300s)
- [x] 7.7 Add Slack notification support:
  - Created `internal/reporting/slack.go` with SlackNotifier
  - Extracts root cause and confidence from investigation.md
  - Sends formatted notification to Slack webhook
- [x] 7.8 Update `internal/agent/executor.go` to support containerized agent:
  - Build prompt from event context
  - Pass system prompt file path via `--system-prompt-file` flag
  - Pass workspace, model, allowed-tools, timeout via command-line args
  - Handle longer timeout (configurable via `AGENT_TIMEOUT`)
- [x] 7.9 End-to-end test: kubernetes-mcp-server → runner → containerized Claude → investigation report → Slack notification
  - Verified containerized agent generates comprehensive investigation reports
  - Verified Slack summary extraction works (root cause + confidence)
  - Slack notification code complete (requires `SLACK_WEBHOOK_URL` env var to enable)

### Manual Test Results (2025-12-18)

**Test Setup:**
- Workspace: `scratch/test-incident-001/`
- Event: Pod `failing-test-triage` with StartError (invalid `/nonexistent` command)
- Agent: Claude sonnet via `agent-container/run-agent.sh`

**Command:**
```bash
source ~/dev-secrets/api-keys.env
./run-agent.sh -w ../scratch/test-incident-001 \
  --system-prompt-file ../configs/triage-system-prompt.md \
  -t "Read,Grep,Glob,Bash,Skill" \
  -m sonnet \
  "Production incident detected. Fault event details are in event.json. Perform immediate triage and root cause analysis."
```

**Result:**
- Exit code: 0
- Investigation report: 7.8KB detailed analysis
- Root cause correctly identified: Invalid container command `/nonexistent`
- Confidence: HIGH (100%)
- Recommendations: Fix pod spec with valid command, preventive measures

### Full Integration Test Results (2025-12-18)

**Test Setup:**
- Workspace: `scratch/e2e-test-001/`
- Event: Same StartError event (copied from test-incident-001)
- Agent: Claude sonnet via executor with full config

**Components Verified:**
1. `internal/config/config.go` - Added AgentSystemPromptFile, AgentAllowedTools, AgentModel
2. `internal/agent/executor.go` - Passes all args to run-agent.sh via NewExecutorWithConfig
3. `internal/reporting/slack.go` - Extracts root cause + confidence from investigation.md
4. `cmd/runner/main.go` - Wires executor with full config, integrates Slack notifier

**Result:**
- Exit code: 0
- Investigation report: 287 lines comprehensive analysis
- Root cause extracted: "The pod specification configured the container with args..."
- Confidence extracted: HIGH
- Slack notification: Code complete, requires SLACK_WEBHOOK_URL env var

**Key Config Options:**
```
AGENT_SCRIPT_PATH=./agent-container/run-agent.sh
AGENT_SYSTEM_PROMPT_FILE=./configs/triage-system-prompt.md
AGENT_ALLOWED_TOOLS=Read,Write,Grep,Glob,Bash,Skill
AGENT_MODEL=sonnet
AGENT_TIMEOUT=300
SLACK_WEBHOOK_URL= (optional)
```
