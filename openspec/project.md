# Project Context

## Purpose

Nightcrier is an MCP client that listens for fault events from kubernetes-mcp-server and spawns AI agents to perform read-only triage of Kubernetes incidents. It acts as a bridge between Kubernetes cluster monitoring and AI-powered incident investigation.

### Goals

1. **Automated Triage**: Automatically investigate Kubernetes faults using AI agents
2. **Read-Only Safety**: Ensure agents can only read cluster state, never modify it
3. **Audit Trail**: Create workspace directories with full investigation artifacts
4. **Extensibility**: Support different AI agent backends (Claude, Codex, etc.)

## Tech Stack

- **Language**: Go 1.23+
- **MCP SDK**: github.com/modelcontextprotocol/go-sdk v1.1.0
- **CLI Framework**: github.com/spf13/cobra
- **Logging**: log/slog (structured JSON logging)
- **UUID Generation**: github.com/google/uuid (for incident IDs only)

## Project Conventions

### Code Style

- Follow Go idioms and effective Go guidelines
- Use structured logging with slog
- Error wrapping with `fmt.Errorf("context: %w", err)`
- Context propagation for cancellation
- No global state; dependency injection preferred

### Architecture Patterns

- **Package Structure**:
  - `cmd/runner/` - CLI entry point
  - `internal/config/` - Configuration loading
  - `internal/events/` - MCP client and event types
  - `internal/agent/` - Workspace and executor
  - `internal/reporting/` - Result writing

- **Component Design**:
  - Each component has a constructor (`NewXxx`) and methods
  - Interfaces defined where abstraction is needed
  - No shared mutable state between components

### Testing Strategy

- Unit tests for individual components
- Integration tests with kubernetes-mcp-server
- Manual testing with kind cluster for e2e validation
- Test fixtures in `testdata/` directories

### Git Workflow

- Conventional Commits: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`
- Main branch is `main`
- Feature work in worktrees using `wtp`
- No emoji in commit messages

## Domain Context

### MCP (Model Context Protocol)

- Standard protocol for AI-tool integration
- Uses JSON-RPC over various transports (stdio, SSE, StreamableHTTP)
- Server exposes tools, resources, and prompts
- Client can call tools and receive notifications

### kubernetes-mcp-server

- MCP server that exposes Kubernetes operations as tools
- `events_subscribe` tool with modes: `events`, `faults`
- Fault events delivered via `logging/message` notifications
- Logger name: `kubernetes/faults`

### Fault Events

A fault is a Warning event targeting a Pod, enriched with:
- FaultID: Stable identifier from kubernetes-mcp-server (hex hash, not UUID)
- Event details (reason, message, timestamp)
- Involved object (Pod name, namespace, UID)
- Container logs (with panic detection)

**Ownership model:**
- **FaultID**: Owned by kubernetes-mcp-server (stable, deterministic hash)
- **IncidentID**: Owned by nightcrier (internal tracking identifier)

## Important Constraints

1. **Read-Only Operations**: Agents must never modify cluster state
2. **Single Concurrency**: One active investigation per cluster
3. **Workspace Isolation**: Each incident gets unique directory
4. **Timeout Limits**: Agent invocations have maximum duration (e.g., 10 min)

## External Dependencies

### kubernetes-mcp-server

- Location: `../kubernetes-mcp-server`
- Endpoint: `http://localhost:8383/mcp` (StreamableHTTP)
- Required for event subscription and fault notifications

### Claude Code CLI

- Agent backend for AI-powered investigation
- Supports headless mode via `-p` flag
- Tool restrictions via `--allowedTools`
- Skill loading from `.claude/skills/` directory

### Kubernetes Cluster

- kind cluster for development: `kind-events-test`
- Read-only ServiceAccount required for agent kubeconfig
- Verbs allowed: get, list, watch, logs

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                   kubernetes-mcp-server                      │
│           (port 8383, /mcp StreamableHTTP)                  │
│                                                              │
│  events_subscribe(mode=faults) → logging/message notifs     │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                        Nightcrier                           │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ MCP Client   │→ │ Workspace    │→ │ Agent        │      │
│  │ (events)     │  │ Manager      │  │ Executor     │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│                             │                                │
│                             ▼                                │
│                    ┌──────────────┐                         │
│                    │ Reporting    │                         │
│                    │ (result.json)│                         │
│                    └──────────────┘                         │
└─────────────────────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                   AI Agent (Claude Code)                     │
│                                                              │
│  - Reads event.json context                                 │
│  - Loads k8s-troubleshooter skill                           │
│  - Performs read-only kubectl commands                       │
│  - Writes investigation artifacts                           │
└─────────────────────────────────────────────────────────────┘
```

## Development Phases

### Phase 1: Walking Skeleton (COMPLETE)
- MCP client connection and subscription
- Basic workspace creation
- Stub agent executor
- Minimal reporting

### Phase 2: Agent Runtime (NEXT)
- Claude Code CLI integration
- Skill loading (k8s-troubleshooter)
- Read-only enforcement layers
- Process lifecycle management

### Phase 3: Reporting
- Investigation report generation
- Metrics and observability
- Slack/webhook notifications

### Phase 4: Resilience
- Connection retry logic
- Circuit breakers
- Graceful degradation

### Phase 5: Optimizations
- Multiple cluster support
- Agent result caching
- Performance tuning
