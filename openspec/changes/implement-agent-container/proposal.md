# Change: Implement Agent Container

## Why

The agent runtime needs a containerized execution environment that:
1. Provides consistent tooling (kubectl, helm, ripgrep, etc.) across environments
2. Supports multiple AI CLI backends (Claude, Codex, Gemini)
3. Includes the k8s-troubleshooter skill for Kubernetes diagnostics
4. Enforces workspace isolation for security

Building a Docker container achieves these goals and enables the event runner to invoke agents in a reproducible, sandboxed manner.

## What Changes

- **New Component**: `agent-container/` directory at repository root
  - `Dockerfile` - Multi-tool container with AI CLIs and Kubernetes tools
  - `run-agent.sh` - Parameterized wrapper script for agent invocation
  - `Makefile` - Build, test, and debug targets
  - `README.md` - Comprehensive documentation

- **Capabilities Delivered**:
  - Multi-agent support: Claude (default/sonnet), Codex, Gemini
  - Built-in k8s-troubleshooter skill from GitHub
  - Workspace isolation (required `-w` flag)
  - Output capture with timestamped logs
  - Configurable timeouts, memory limits, and tool restrictions

## Impact

- **New Capability**: `agent-container` - containerized agent execution
- **Relation to `agent-runtime`**: This implements the container/execution portion of the agent-runtime design
- **Dependencies**: None (standalone container)
- **Consumers**: Event runner will call `run-agent.sh` to invoke agents

## Delivered Artifacts

| Artifact | Description |
|----------|-------------|
| `agent-container/Dockerfile` | 2.8GB image with kubectl, helm, AI CLIs, k8s-troubleshooter |
| `agent-container/run-agent.sh` | CLI wrapper with full configuration options |
| `agent-container/Makefile` | Build and test automation |
| `agent-container/README.md` | Architecture, configuration, troubleshooting docs |

## Status

**COMPLETE** - All tasks implemented and tested.
