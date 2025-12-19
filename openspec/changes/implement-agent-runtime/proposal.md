# Change: Implement Agent Runtime (Phase 2)

## Walking Skeleton Baseline

The walking-skeleton implementation (archived 2025-12-18) provides substantial agent runtime functionality:

**Already Implemented:**
- `internal/agent/` package with workspace.go, context.go, executor.go
- Workspace creation with UUID-based incident directories
- Event context writing (event.json in workspace)
- Containerized agent execution via `agent-container/run-agent.sh`
- Multi-agent support: Claude (default), Codex, Gemini
- k8s-troubleshooter skill integration (built into container)
- Exit code capture and logging
- Configurable timeout (AGENT_TIMEOUT env var)
- System prompt file support
- Investigation report generation (output/investigation.md)

**This Change Adds:**
Advanced runtime features: in-memory state tracking, Prometheus metrics, progressive output streaming, workspace GC, and comprehensive testing.

## Why
To enable the runner to actually invoke AI agents in a safe, isolated manner to investigate the events accepted by the processing layer.

## What Changes
- Enhance `agent-runtime` capability (builds on walking skeleton):
    - **DONE**: "Headless" CLI wrapper to invoke agents (e.g., Claude, Codex).
    - **DONE**: Workspace creation (unique directory per incident).
    - **DONE**: Context construction (passing event details to the agent).
    - Read-only guardrails (via startup flags/env vars) - partially done.
    - In-memory state tracking for active agents.
    - Prometheus metrics for agent lifecycle.

## Impact
- **Enhanced Capabilities**: `agent-runtime` (builds on walking skeleton).
- **Modified Code**: `internal/agent` package enhancements.
- **Dependencies**: Depends on the event processing layer to trigger these actions.
