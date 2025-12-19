# walking-skeleton Specification

## Purpose

End-to-end event runner that subscribes to Kubernetes fault events via MCP protocol and spawns containerized Claude agents to perform automated incident triage, generating investigation reports and optional Slack notifications.

## Status

**IMPLEMENTED** - Validated end-to-end 2025-12-18

## Architecture

```
kubernetes-mcp-server  -->  Event Runner  -->  Containerized Claude Agent
     (MCP /mcp)              (Go binary)        (k8s-triage-agent Docker)
                                  |
                                  v
                          Workspace artifacts:
                          - event.json
                          - output/investigation.md
                          - result.json
                                  |
                                  v
                          Slack notification (optional)
```
## Requirements
### Requirement: MCP Event Subscription

The system SHALL connect to kubernetes-mcp-server via MCP StreamableHTTP protocol and subscribe to fault events.

#### Scenario: Successful MCP connection
- **GIVEN** a valid `K8S_CLUSTER_MCP_ENDPOINT` URL
- **WHEN** the runner starts
- **THEN** it connects via StreamableHTTP, initializes a session, and subscribes with `events_subscribe(mode=<configured-mode>)`

#### Scenario: Event reception (faults mode)
- **GIVEN** an active MCP subscription with mode="faults"
- **WHEN** a fault occurs in the cluster
- **THEN** the runner receives a `logging/message` notification with `logger="kubernetes/faults"` containing FaultEvent data with nested event structure

#### Scenario: Event reception (resource-faults mode)
- **GIVEN** an active MCP subscription with mode="resource-faults"
- **WHEN** a fault occurs in the cluster
- **THEN** the runner receives a `logging/message` notification with `logger="kubernetes/resource-faults"` containing FaultEvent data with flat structure (resource, context, faultType, severity, timestamp)

#### Scenario: Event parsing (faults mode)
- **GIVEN** a fault notification from faults mode
- **WHEN** the event is received
- **THEN** the JSON is parsed into a FaultEvent struct with nested event object (subscriptionId, cluster, event.namespace, event.reason, event.message, event.involvedObject)

#### Scenario: Event parsing (resource-faults mode)
- **GIVEN** a fault notification from resource-faults mode
- **WHEN** the event is received
- **THEN** the JSON is parsed into a FaultEvent struct with flat structure (subscriptionId, cluster, resource.kind, resource.name, resource.namespace, context, faultType, severity, timestamp)

#### Scenario: Helper method compatibility
- **GIVEN** a parsed FaultEvent from either subscription mode
- **WHEN** helper methods (GetResourceName, GetResourceKind, GetNamespace, GetSeverity) are called
- **THEN** the correct values are returned regardless of which mode was used

### Requirement: Incident Workspace Creation

The system SHALL create a unique workspace directory for each incident with proper file structure.

#### Scenario: Workspace directory created
- **GIVEN** a received fault event passes filtering
- **WHEN** processing begins
- **THEN** an Incident is created with a unique incidentId (UUID)
- **AND** a directory is created at `<WORKSPACE_ROOT>/<incident-uuid>/` with 0700 permissions

#### Scenario: Incident context written
- **GIVEN** a created workspace
- **WHEN** context is prepared for the agent
- **THEN** an `incident.json` file is written containing incidentId, status, cluster, namespace, resource, faultType, severity, context, and timestamp
- **AND** the status is set to "investigating"

#### Scenario: Incident updated after completion
- **GIVEN** agent execution completes
- **WHEN** results are recorded
- **THEN** the `incident.json` file is updated in place with completedAt, exitCode, and updated status
- **AND** no separate result.json file is created

### Requirement: Containerized Agent Execution

The system SHALL execute a containerized agent for incident triage using incident context.

#### Scenario: Agent invocation
- **GIVEN** a workspace with incident.json
- **WHEN** the executor runs
- **THEN** the agent container is launched with access to incident.json
- **AND** the agent reads incident context from incident.json (not event.json)

#### Scenario: Investigation report
- **GIVEN** agent execution completes successfully
- **WHEN** the agent finishes triage
- **THEN** an investigation report is written to `<workspace>/output/investigation.md`

### Requirement: Slack Notification (Optional)

The system SHALL send Slack notifications when configured.

#### Scenario: Slack notification sent
- **GIVEN** `SLACK_WEBHOOK_URL` is configured
- **WHEN** agent execution completes
- **THEN** a formatted Slack message is sent containing cluster, namespace, resource, root cause summary, and confidence level

#### Scenario: Slack disabled
- **GIVEN** `SLACK_WEBHOOK_URL` is not set
- **WHEN** agent execution completes
- **THEN** no Slack notification is attempted

### Requirement: Event-Incident Separation

The system SHALL maintain clean separation between events (input) and incidents (response).

#### Scenario: Event remains immutable
- **GIVEN** a fault event is received from MCP
- **WHEN** the event is parsed
- **THEN** an eventId is generated for tracing
- **AND** the event data is not modified with incident metadata

#### Scenario: Incident created from event
- **GIVEN** an event passes severity filtering and deduplication
- **WHEN** processing begins
- **THEN** an Incident is created with relevant event data flattened into it
- **AND** the Incident references the triggeringEventId for traceability

### Requirement: Incident Lifecycle Status

The system SHALL track incident status through its lifecycle.

#### Scenario: Status transitions
- **GIVEN** an incident is created
- **THEN** status progresses through: investigating → resolved|failed|agent_failed
- **AND** timestamps are recorded for each transition (createdAt, startedAt, completedAt)

#### Scenario: Resolved status
- **GIVEN** agent exits with code 0 and produces valid investigation.md
- **WHEN** incident is updated
- **THEN** status is set to "resolved"

#### Scenario: Failed status
- **GIVEN** agent exits with non-zero code or crashes
- **WHEN** incident is updated
- **THEN** status is set to "failed"
- **AND** failureReason describes the failure

#### Scenario: Agent failed status
- **GIVEN** agent exits successfully but produces no valid output
- **WHEN** incident is updated
- **THEN** status is set to "agent_failed"
- **AND** failureReason indicates missing or invalid investigation report

## Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `K8S_CLUSTER_MCP_ENDPOINT` | Yes | - | kubernetes-mcp-server URL |
| `ANTHROPIC_API_KEY` | Conditional | - | Claude API key (required if AGENT_CLI=claude) |
| `OPENAI_API_KEY` | Conditional | - | OpenAI API key (required if AGENT_CLI=codex) |
| `WORKSPACE_ROOT` | No | `./incidents` | Incident workspace directory |
| `SUBSCRIBE_MODE` | No | `faults` | Subscription mode: `faults` or `resource-faults` |
| `AGENT_SCRIPT_PATH` | No | `./agent-container/run-agent.sh` | Path to agent script |
| `AGENT_SYSTEM_PROMPT_FILE` | No | `./configs/triage-system-prompt.md` | System prompt for agent |
| `AGENT_ALLOWED_TOOLS` | No | `Read,Write,Grep,Glob,Bash,Skill` | Tools available to agent |
| `AGENT_MODEL` | No | `sonnet` | Model to use |
| `AGENT_TIMEOUT` | No | `300` | Agent timeout in seconds |
| `AGENT_CLI` | No | `claude` | CLI tool: `claude`, `codex`, `goose`, `gemini` |
| `SLACK_WEBHOOK_URL` | No | - | Slack webhook for notifications |
| `LOG_LEVEL` | No | `info` | Logging level |

See `cloud-storage` spec for Azure Blob Storage configuration.

## Implementation Files

- `cmd/runner/main.go` - CLI entrypoint with Cobra
- `internal/config/config.go` - Configuration loading
- `internal/events/client.go` - MCP client with StreamableHTTP
- `internal/events/event.go` - FaultEvent struct (dual-mode support)
- `internal/agent/workspace.go` - Workspace creation
- `internal/agent/context.go` - Event context writing
- `internal/agent/executor.go` - Agent execution
- `internal/reporting/result.go` - Result recording
- `internal/reporting/slack.go` - Slack notifications
- `internal/storage/storage.go` - Storage interface
- `internal/storage/azure.go` - Azure Blob Storage adapter
- `internal/storage/filesystem.go` - Filesystem storage adapter
- `agent-container/run-agent.sh` - Container orchestration (multi-CLI)
- `agent-container/Dockerfile` - Agent container image
- `configs/triage-system-prompt.md` - Agent system prompt

