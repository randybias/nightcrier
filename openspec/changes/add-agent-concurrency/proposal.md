# Change: Add Agent Concurrency Control

## Why
Currently, the system processes events sequentially (`cmd/nightcrier/main.go` blocks on `processEvent`). This means if one agent takes 5 minutes to triage a cluster, all other events (even from different clusters) are blocked. This limits throughput and can cause event backlogs. Conversely, launching an unlimited number of agents would exhaust system resources (CPU/RAM) and potentially rate-limit the K8s API.

## What Changes
- Implement a **Dispatcher** component to manage agent scheduling.
- Enforce **Global Concurrency Limit** (`MaxConcurrentAgents`) to protect system resources.
- Enforce **Per-Cluster Concurrency Limit** (1 agent per cluster) to prevent "swarming" a single cluster.
- Decouple event ingestion from agent execution (Non-blocking Dispatch).
- Introduce a queueing mechanism for pending investigations.

**Architecture Note:** The Dispatcher is responsible for *policy* (who runs when), not *mechanism* (how they run). It manages the "slots" and locks. The actual execution logic remains in the `executor` package for now, but the Dispatcher paves the way for the future `AgentRuntime` interface.

## Impact
- **Affected Specs:**
    - `agent-execution` (New capability)
    - `configuration` (Add concurrency settings)
- **Affected Code:**
    - `cmd/nightcrier/main.go` (Refactor event loop)
    - `internal/dispatcher/` (New package)
    - `internal/config/` (Ensure config plumbing)
