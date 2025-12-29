# Tasks: Add Agent Concurrency Control

## 1. Configuration

- [ ] 1.1 Add `ClusterQueueSize` field to Config struct (default: 10)
- [ ] 1.2 Add `EventTTLSeconds` field to Config struct (default: 300)
- [ ] 1.3 Add environment variable mappings (`CLUSTER_QUEUE_SIZE`, `EVENT_TTL_SECONDS`)
- [ ] 1.4 Add validation for new config fields (>= 1)
- [ ] 1.5 Update config example file with new settings
- [ ] 1.6 Add unit tests for new config fields

## 2. Event Queue

- [ ] 2.1 Create `internal/dispatcher/queue.go` with EventQueue struct
- [ ] 2.2 Implement `QueuedEvent` struct with EnqueuedAt timestamp
- [ ] 2.3 Implement `Enqueue()` with TTL pruning and drop-oldest logic
- [ ] 2.4 Implement `PopFrontIfValid()` that prunes expired before returning
- [ ] 2.5 Implement `Len()` and `IsEmpty()` helpers
- [ ] 2.6 Add comprehensive unit tests for queue in `queue_test.go`:
  - [ ] 2.6.1 Test TTL expiration
  - [ ] 2.6.2 Test drop-oldest when full
  - [ ] 2.6.3 Test concurrent access safety
  - [ ] 2.6.4 Test empty queue behavior

## 3. Dispatcher Core

- [ ] 3.1 Create `internal/dispatcher/dispatcher.go`
- [ ] 3.2 Implement Dispatcher struct with:
  - [ ] 3.2.1 Global semaphore (buffered channel)
  - [ ] 3.2.2 Per-cluster state map (sync.RWMutex protected)
  - [ ] 3.2.3 Closed flag for shutdown
- [ ] 3.3 Implement `NewDispatcher(cfg)` constructor
- [ ] 3.4 Implement `Dispatch()` method (non-blocking):
  - [ ] 3.4.1 Check event TTL
  - [ ] 3.4.2 Get or create cluster state
  - [ ] 3.4.3 If cluster busy, queue event
  - [ ] 3.4.4 If cluster free, mark busy and launch goroutine
- [ ] 3.5 Implement `executeWithLock()` internal method:
  - [ ] 3.5.1 Acquire global semaphore slot
  - [ ] 3.5.2 Call processEvent (via callback/interface)
  - [ ] 3.5.3 Release slot on completion
  - [ ] 3.5.4 Call onComplete to release cluster lock and process queue
- [ ] 3.6 Implement `onComplete()` method:
  - [ ] 3.6.1 Get next valid event from queue
  - [ ] 3.6.2 If event exists, launch new goroutine
  - [ ] 3.6.3 If no event, mark cluster as not running
- [ ] 3.7 Implement `Shutdown(ctx)` method:
  - [ ] 3.7.1 Set closed flag
  - [ ] 3.7.2 Wait for in-flight agents (with timeout)
  - [ ] 3.7.3 Log queue contents on exit

## 4. Dispatcher Tests

- [ ] 4.1 Create `internal/dispatcher/dispatcher_test.go`
- [ ] 4.2 Test: Events for different clusters run in parallel
- [ ] 4.3 Test: Events for same cluster are serialized
- [ ] 4.4 Test: Global semaphore limits total concurrent agents
- [ ] 4.5 Test: Non-blocking dispatch (returns immediately)
- [ ] 4.6 Test: Expired events dropped before dispatch
- [ ] 4.7 Test: Queue overflow drops oldest
- [ ] 4.8 Test: Failure in one agent doesn't block others
- [ ] 4.9 Test: Graceful shutdown waits for in-flight

## 5. Integration with main.go

- [ ] 5.1 Create Dispatcher in main() with config
- [ ] 5.2 Replace blocking `processEvent()` call with `dispatcher.Dispatch()`
- [ ] 5.3 Pass required dependencies to Dispatch (executor, notifier, etc.)
- [ ] 5.4 Add dispatcher.Shutdown() to graceful shutdown sequence
- [ ] 5.5 Update startup logging to show dispatcher config

## 6. Integration Tests

- [ ] 6.1 Add integration test: concurrent events across clusters
- [ ] 6.2 Add integration test: serialized events for same cluster
- [ ] 6.3 Add integration test: shutdown with in-flight agents

## 7. Documentation

- [ ] 7.1 Update configuration docs with new settings
- [ ] 7.2 Add concurrency behavior to architecture docs
- [ ] 7.3 Update README if needed

## Dependencies

- Tasks 1.x (Config) must complete before 3.x (Dispatcher Core)
- Task 2.x (Queue) must complete before 3.x (Dispatcher Core)
- Tasks 3.x must complete before 5.x (Integration)
- Tasks 4.x can run in parallel with 5.x

## Validation

After completing all tasks:
- [ ] All unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing with multi-cluster setup confirms parallel execution
- [ ] Manual testing confirms same-cluster events are serialized
