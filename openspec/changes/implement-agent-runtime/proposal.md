# Change: Implement Agent Runtime (Phase 2)

## Why
To enable the runner to actually invoke AI agents in a safe, isolated manner to investigate the events accepted by the processing layer.

## What Changes
- Implement `agent-runtime` capability:
    - "Headless" CLI wrapper to invoke agents (e.g., Claude, Codex).
    - Workspace creation (unique directory per incident).
    - Context construction (passing event details to the agent).
    - Read-only guardrails (via startup flags/env vars).

## Impact
- **New Capabilities**: `agent-runtime`.
- **New Code**: `internal/agent` package, process execution logic.
- **Dependencies**: Depends on the event processing layer to trigger these actions.
