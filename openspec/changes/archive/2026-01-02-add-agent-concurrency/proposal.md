# Change: Add Agent Concurrency Control

## Why

Currently, the system processes events sequentially (`cmd/nightcrier/main.go` blocks on `processEvent`). This means if one agent takes 5 minutes to triage a cluster, all other events (even from different clusters) are blocked. This limits throughput and causes event backlogs.

Additionally, launching unlimited concurrent agents would exhaust system resources (CPU/RAM, LLM API quotas) and could overwhelm target clusters with kubectl commands.

## What Changes

### Core Components

1. **Dispatcher**: New component managing agent scheduling with:
   - **Global Concurrency Limit** (`MaxConcurrentAgents`) - Protects system resources and LLM API quotas
   - **Per-Cluster Locks** (in-memory mutexes) - Prevents multiple agents from swarming a single target cluster
   - **Non-blocking Dispatch** - Event ingestion decoupled from agent execution

2. **Event Queuing with TTL**:
   - Bounded per-cluster queues (configurable size, default: 10)
   - **Event TTL** (configurable, default: 5 minutes) - Stale events are dropped
   - Drop-oldest policy when queue full (newer events more relevant)

3. **Future: Database-backed Coordination**:
   - Current design uses in-memory mutexes (sufficient for single Nightcrier instance)
   - Note: Agents run on dedicated triage clusters, not on target clusters being triaged
   - Multi-instance Nightcrier deployments will require database-backed locks (out of scope)

### Architecture Note

The Dispatcher is responsible for *policy* (who runs when), not *mechanism* (how they run). The actual K8s Job execution remains in `internal/agent/k8s/`. The Dispatcher manages slots and locks, then delegates to the existing executor.

```
┌─────────────────────────────────────────────────────────────────┐
│                        Event Loop (main.go)                      │
│                                                                  │
│  MCP Notification → Parse Event → dispatcher.Dispatch(event)    │
│                         (non-blocking)                           │
└──────────────────────────────────┬──────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────┐
│                         Dispatcher                               │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │ Global       │  │ Per-Cluster  │  │ Per-Cluster  │          │
│  │ Semaphore    │  │ Locks        │  │ Queues       │          │
│  │ (N slots)    │  │ (mutexes)    │  │ (with TTL)   │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
│                                                                  │
│  Dispatch():                                                     │
│    1. Check event TTL → drop if stale                           │
│    2. Acquire per-cluster lock (or queue)                       │
│    3. Acquire global semaphore slot                             │
│    4. Launch goroutine → executor.Execute()                     │
│    5. On completion: release slot, release lock, process queue  │
└─────────────────────────────────────────────────────────────────┘
```

## Supersedes

This proposal supersedes the concurrency-related portions of `implement-event-intake`:
- Per-Cluster Concurrency Control (implement-event-intake Section 4.5)
- Queue Overflow Management (implement-event-intake Section 4.6)
- Agent Concurrency Limiter (implement-event-intake Section 4.4)

The `implement-event-intake` proposal should be updated to remove these sections and reference this proposal for concurrency control.

## Impact

### Affected Specs
- `agent-execution` (New capability - concurrent execution with per-cluster serialization)
- `configuration` (MODIFIED - add queue and TTL settings)

### Affected Code
- `cmd/nightcrier/main.go` - Refactor event loop to use Dispatcher
- `internal/dispatcher/` - New package
- `internal/config/` - Add queue size, TTL settings

### New Configuration
| Setting | Env Var | Default | Description |
|---------|---------|---------|-------------|
| `max_concurrent_agents` | `MAX_CONCURRENT_AGENTS` | 10 | Global agent limit |
| `cluster_queue_size` | `CLUSTER_QUEUE_SIZE` | 10 | Max queued events per cluster |
| `event_ttl_seconds` | `EVENT_TTL_SECONDS` | 300 | Events older than this are dropped |

Note: `max_concurrent_agents` already exists in config but is not enforced.

## Success Criteria

1. Events from different clusters process in parallel (up to MaxConcurrentAgents)
2. Events for the same cluster are strictly serialized
3. Event ingestion never blocks (non-blocking dispatch)
4. Stale events (older than TTL) are dropped with logging
5. Queue overflow drops oldest events (not newest)
6. All locks/semaphores released on agent failure/panic

## Out of Scope

- Multi-instance Nightcrier coordination (requires database locks)
- Priority-based scheduling (all events equal priority)
- Metrics/observability for queue depths (future work)
