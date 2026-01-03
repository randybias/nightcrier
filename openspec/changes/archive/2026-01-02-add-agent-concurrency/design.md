# Design: Agent Concurrency Control

## 1. Overview

This document details the architectural decisions for implementing concurrent agent execution with per-cluster serialization.

## 2. Architecture

### 2.1 Component Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                            main.go Event Loop                            │
│                                                                          │
│   for event := range eventChan {                                        │
│       dispatcher.Dispatch(ctx, event, clusterName, executor, ...)       │
│       // Non-blocking - returns immediately                              │
│   }                                                                      │
└──────────────────────────────────┬──────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                            Dispatcher                                    │
│                                                                          │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                      Global Semaphore                            │   │
│  │              (buffered channel, capacity = N)                    │   │
│  │                                                                  │   │
│  │  Acquire: sem <- struct{}{}   Release: <-sem                    │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                          │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    Per-Cluster State                             │   │
│  │                                                                  │   │
│  │  clusterStates map[string]*ClusterState                         │   │
│  │                                                                  │   │
│  │  type ClusterState struct {                                      │   │
│  │      mu       sync.Mutex      // Protects state                 │   │
│  │      running  bool            // Is agent currently running?    │   │
│  │      queue    *EventQueue     // Pending events with TTL        │   │
│  │  }                                                               │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### 2.2 Data Flow

1. **Event received** → Dispatcher.Dispatch() called (non-blocking)
2. **TTL check** → If event.Timestamp + TTL < now, drop and log
3. **Cluster lock check** → If cluster busy, enqueue (or drop oldest if full)
4. **Global semaphore** → Acquire slot (blocking within goroutine)
5. **Execute** → Call executor.Execute() in goroutine
6. **Completion** → Release semaphore, release cluster lock, process queue

## 3. Key Design Decisions

### Decision 1: In-Memory Mutexes (Not Database)

**Context**: Need to prevent multiple agents from running on the same target cluster.

**Decision**: Use in-memory sync.Mutex per cluster, not database locks.

**Rationale**:
- Simplest implementation for single Nightcrier instance
- No external dependencies
- Sufficient for current deployment model
- Database locks can be added later for multi-instance deployments

**Trade-offs**:
- Lost on restart (acceptable - agents complete or timeout)
- Single instance only (documented limitation)

**Future**: Multi-instance deployments will need database advisory locks (PostgreSQL `pg_advisory_lock`) or Redis distributed locks.

### Decision 2: Event TTL with Drop-Oldest Policy

**Context**: Queued events become stale - a fault from 10 minutes ago may be resolved.

**Decision**: Implement event TTL and drop-oldest policy.

**Implementation**:
```go
type QueuedEvent struct {
    Event     *events.FaultEvent
    EnqueuedAt time.Time
    // Derived: ExpiresAt = EnqueuedAt + EventTTL
}

func (q *EventQueue) Enqueue(event *QueuedEvent) {
    // 1. Remove expired events from queue
    q.pruneExpired()

    // 2. If full, drop oldest (it's the most stale)
    if q.Len() >= q.maxSize {
        dropped := q.PopFront()
        log.Warn("queue full, dropped oldest event",
            "cluster", q.cluster,
            "dropped_fault_id", dropped.Event.FaultID)
    }

    // 3. Add new event
    q.PushBack(event)
}
```

**Rationale**:
- Newer events more relevant than older ones
- Prevents unbounded memory growth
- Explicit TTL makes staleness visible in logs

### Decision 3: Global Semaphore via Buffered Channel

**Context**: Need to limit total concurrent agents across all clusters.

**Decision**: Use a buffered channel as a counting semaphore.

**Implementation**:
```go
type Dispatcher struct {
    globalSem chan struct{}  // Capacity = MaxConcurrentAgents
    // ...
}

func NewDispatcher(cfg *config.Config) *Dispatcher {
    return &Dispatcher{
        globalSem: make(chan struct{}, cfg.MaxConcurrentAgents),
        // ...
    }
}

func (d *Dispatcher) acquireSlot(ctx context.Context) error {
    select {
    case d.globalSem <- struct{}{}:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

func (d *Dispatcher) releaseSlot() {
    <-d.globalSem
}
```

**Rationale**:
- Idiomatic Go pattern for semaphores
- Built-in context cancellation support
- No external dependencies

### Decision 4: Non-Blocking Dispatch

**Context**: Event ingestion must not block on agent execution.

**Decision**: Dispatch() returns immediately, execution happens in goroutine.

**Implementation**:
```go
func (d *Dispatcher) Dispatch(ctx context.Context, event *events.FaultEvent,
    cluster string, executor AgentExecutor, ...) {

    // Check TTL immediately
    if d.isExpired(event) {
        log.Debug("dropping expired event", "fault_id", event.FaultID)
        return
    }

    cs := d.getOrCreateClusterState(cluster)

    cs.mu.Lock()
    if cs.running {
        // Cluster busy - queue the event
        cs.queue.Enqueue(&QueuedEvent{Event: event, EnqueuedAt: time.Now()})
        cs.mu.Unlock()
        return
    }
    cs.running = true
    cs.mu.Unlock()

    // Launch execution in background
    go d.executeWithLock(ctx, event, cluster, executor, ...)
}

func (d *Dispatcher) executeWithLock(ctx context.Context, event *events.FaultEvent,
    cluster string, executor AgentExecutor, ...) {

    defer d.onComplete(cluster)  // Release locks, process queue

    // Acquire global slot (blocking)
    if err := d.acquireSlot(ctx); err != nil {
        log.Error("failed to acquire slot", "error", err)
        return
    }
    defer d.releaseSlot()

    // Execute the agent
    if err := processEvent(ctx, event, ...); err != nil {
        log.Error("agent execution failed", "error", err)
    }
}

func (d *Dispatcher) onComplete(cluster string) {
    cs := d.clusterStates[cluster]
    cs.mu.Lock()
    defer cs.mu.Unlock()

    // Process next queued event if any
    if next := cs.queue.PopFrontIfValid(); next != nil {
        go d.executeWithLock(ctx, next.Event, cluster, ...)
        return
    }

    cs.running = false
}
```

### Decision 5: Graceful Shutdown

**Context**: Need to handle SIGTERM/SIGINT gracefully.

**Decision**: Wait for in-flight agents with timeout, then force exit.

**Implementation**:
```go
func (d *Dispatcher) Shutdown(ctx context.Context) error {
    // Stop accepting new dispatches
    d.closed.Store(true)

    // Wait for all in-flight agents (up to context deadline)
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            log.Warn("shutdown timeout, forcing exit",
                "in_flight", d.inFlightCount())
            return ctx.Err()
        case <-ticker.C:
            if d.inFlightCount() == 0 {
                log.Info("all agents completed, shutdown clean")
                return nil
            }
        }
    }
}
```

## 4. EventQueue Implementation

```go
type EventQueue struct {
    cluster  string
    maxSize  int
    ttl      time.Duration
    events   []*QueuedEvent
    mu       sync.Mutex
}

func (q *EventQueue) pruneExpired() {
    now := time.Now()
    valid := q.events[:0]
    for _, e := range q.events {
        if now.Before(e.EnqueuedAt.Add(q.ttl)) {
            valid = append(valid, e)
        } else {
            log.Debug("pruned expired event",
                "cluster", q.cluster,
                "fault_id", e.Event.FaultID,
                "age", now.Sub(e.EnqueuedAt))
        }
    }
    q.events = valid
}

func (q *EventQueue) PopFrontIfValid() *QueuedEvent {
    q.pruneExpired()
    if len(q.events) == 0 {
        return nil
    }
    event := q.events[0]
    q.events = q.events[1:]
    return event
}
```

## 5. Configuration

```go
type Config struct {
    // Existing
    MaxConcurrentAgents int `mapstructure:"max_concurrent_agents"`

    // New
    ClusterQueueSize    int `mapstructure:"cluster_queue_size"`    // Default: 10
    EventTTLSeconds     int `mapstructure:"event_ttl_seconds"`     // Default: 300
}
```

Environment variables:
- `MAX_CONCURRENT_AGENTS` (existing, now enforced)
- `CLUSTER_QUEUE_SIZE` (new)
- `EVENT_TTL_SECONDS` (new)

## 6. Logging

All dispatch decisions are logged for observability:

| Log Level | Event | Fields |
|-----------|-------|--------|
| DEBUG | Event expired before dispatch | fault_id, age, ttl |
| DEBUG | Event queued (cluster busy) | fault_id, cluster, queue_depth |
| DEBUG | Event pruned from queue (TTL) | fault_id, cluster, age |
| WARN | Queue full, dropped oldest | fault_id, cluster, dropped_fault_id |
| INFO | Agent started | fault_id, cluster, slot_usage |
| INFO | Agent completed | fault_id, cluster, duration |
| ERROR | Agent failed | fault_id, cluster, error |

## 7. Testing Strategy

### Unit Tests
- `dispatcher_test.go`: Global semaphore behavior
- `dispatcher_test.go`: Per-cluster locking
- `queue_test.go`: TTL expiration
- `queue_test.go`: Drop-oldest policy

### Integration Tests
- Concurrent events for different clusters → parallel execution
- Concurrent events for same cluster → serialized execution
- Queue overflow behavior
- Graceful shutdown with in-flight agents

## 8. Files Changed

### Added
- `internal/dispatcher/dispatcher.go` - Dispatcher implementation
- `internal/dispatcher/queue.go` - EventQueue with TTL
- `internal/dispatcher/dispatcher_test.go` - Unit tests
- `internal/dispatcher/queue_test.go` - Queue tests

### Modified
- `cmd/nightcrier/main.go` - Use Dispatcher instead of direct processEvent
- `internal/config/config.go` - Add ClusterQueueSize, EventTTLSeconds
- `internal/config/config_test.go` - Test new config fields

## 9. Future Considerations

### Multi-Instance Deployment
Current design assumes single Nightcrier instance. For HA deployments:
1. Use PostgreSQL advisory locks for per-cluster coordination
2. Or Redis distributed locks (SETNX with TTL)
3. Global semaphore becomes harder - may need distributed counter or accept over-provisioning

### Observability
Future work could add:
- Prometheus metrics for queue depths
- Metrics for slot utilization
- Alerts on sustained queue buildup
