# Change: Implement Walking Skeleton

## Why

The full OpenSpec proposals (Phases 1-5) contain 400+ tasks. Implementing them sequentially delays live integration testing with the real kubernetes-mcp-server until all components are complete.

A "walking skeleton" provides the thinnest possible end-to-end slice that proves the integration works. This enables:
- Early validation that SSE events flow through the entire pipeline
- Fast feedback on integration issues before investing in full implementation
- A working foundation to incrementally flesh out with features from the full proposals

## What Changes

Implements a minimal vertical slice across Phases 1-3:

| Phase | Full Proposal | Walking Skeleton |
|-------|---------------|------------------|
| Phase 1 (Event Intake) | SSE client, reconnection, dedup, circuit breaker, queuing | SSE connect + parse JSON only |
| Phase 2 (Agent Runtime) | Full workspace, k8s-troubleshooter, Claude CLI | Create dir, write event.json, exec stub script |
| Phase 3 (Reporting) | Markdown templates, Slack, artifacts | Write result.json to workspace |

What is explicitly **skipped**:
- Reconnection with exponential backoff (Phase 1)
- Severity filtering (Phase 1)
- Deduplication cache (Phase 1)
- Circuit breaker / rate limiting (Phase 1)
- Per-cluster queuing (Phase 1)
- k8s-troubleshooter skill loading (Phase 2)
- Real Claude CLI invocation (Phase 2)
- Markdown report templates (Phase 3)
- Slack webhook integration (Phase 3)
- All of Phase 4 (Resilience) and Phase 5 (Optimization)

## Impact

- **New Code**: `cmd/runner/`, `internal/config/`, `internal/events/`, `internal/agent/`, `internal/reporting/`
- **New Capabilities**: `walking-skeleton` (temporary capability, superseded by full phases)
- **Dependencies**: Requires running kubernetes-mcp-server emitting fault events

## Testing Strategy

Minimal testing for skeleton:
1. Manual end-to-end test with real kubernetes-mcp-server
2. Verify: event received -> workspace created -> stub script executed -> result.json written

Full test coverage deferred to individual phase implementations.

## Extension: Real Agent Integration

After proving the stub flow works, extend to use the real containerized Claude agent:
1. Create triage prompt templates (`configs/triage-system-prompt.md`, `configs/triage-prompt.md`)
2. Test manually with `agent-container/run-agent.sh`
3. Integrate into executor with proper environment (API keys, kubeconfig)
4. End-to-end test with real fault → real AI triage

This extension proves the full value proposition before investing in the remaining phases.

## Parallelization Strategy

Independent workstreams (can run in parallel):
1. **Project skeleton + config**: Go module, CLI entry, config loading
2. **SSE client + event parsing**: Connect to endpoint, parse FaultEvent JSON
3. **Workspace + context**: Create incident directory, write event.json
4. **Agent executor**: exec.Command wrapper, stub script

Serial dependency:
5. **Main wiring**: Connects all components (depends on 1-4)
