# Tasks: Add Agent Concurrency Control

## 1. Configuration

- [x] 1.1 Add `ClusterQueueSize` field to Config struct (default: 10)
- [x] 1.2 Add `EventTTLSeconds` field to Config struct (default: 300)
- [x] 1.3 Add environment variable mappings (`CLUSTER_QUEUE_SIZE`, `EVENT_TTL_SECONDS`)
- [x] 1.4 Add validation for new config fields (>= 1)
- [x] 1.5 Update config example file with new settings
- [x] 1.6 Add unit tests for new config fields

## 2. Event Queue

- [x] 2.1 Create `internal/dispatcher/queue.go` with EventQueue struct
- [x] 2.2 Implement `QueuedEvent` struct with EnqueuedAt timestamp
- [x] 2.3 Implement `Enqueue()` with TTL pruning and drop-oldest logic
- [x] 2.4 Implement `PopFrontIfValid()` that prunes expired before returning
- [x] 2.5 Implement `Len()` and `IsEmpty()` helpers
- [x] 2.6 Add comprehensive unit tests for queue in `queue_test.go`:
  - [x] 2.6.1 Test TTL expiration
  - [x] 2.6.2 Test drop-oldest when full
  - [x] 2.6.3 Test concurrent access safety
  - [x] 2.6.4 Test empty queue behavior

## 3. Dispatcher Core

- [x] 3.1 Create `internal/dispatcher/dispatcher.go`
- [x] 3.2 Implement Dispatcher struct with:
  - [x] 3.2.1 Global semaphore (buffered channel)
  - [x] 3.2.2 Per-cluster state map (sync.RWMutex protected)
  - [x] 3.2.3 Closed flag for shutdown
- [x] 3.3 Implement `NewDispatcher(cfg)` constructor
- [x] 3.4 Implement `Dispatch()` method (non-blocking):
  - [x] 3.4.1 Check event TTL
  - [x] 3.4.2 Get or create cluster state
  - [x] 3.4.3 If cluster busy, queue event
  - [x] 3.4.4 If cluster free, mark busy and launch goroutine
- [x] 3.5 Implement `executeWithLock()` internal method:
  - [x] 3.5.1 Acquire global semaphore slot
  - [x] 3.5.2 Call processEvent (via callback/interface)
  - [x] 3.5.3 Release slot on completion
  - [x] 3.5.4 Call onComplete to release cluster lock and process queue
- [x] 3.6 Implement `onComplete()` method:
  - [x] 3.6.1 Get next valid event from queue
  - [x] 3.6.2 If event exists, launch new goroutine
  - [x] 3.6.3 If no event, mark cluster as not running
- [x] 3.7 Implement `Shutdown(ctx)` method:
  - [x] 3.7.1 Set closed flag
  - [x] 3.7.2 Wait for in-flight agents (with timeout)
  - [x] 3.7.3 Log queue contents on exit

## 4. Dispatcher Tests

- [x] 4.1 Create `internal/dispatcher/dispatcher_test.go`
- [x] 4.2 Test: Events for different clusters run in parallel
- [x] 4.3 Test: Events for same cluster are serialized
- [x] 4.4 Test: Global semaphore limits total concurrent agents
- [x] 4.5 Test: Non-blocking dispatch (returns immediately)
- [x] 4.6 Test: Expired events dropped before dispatch
- [x] 4.7 Test: Queue overflow drops oldest
- [x] 4.8 Test: Failure in one agent doesn't block others
- [x] 4.9 Test: Graceful shutdown waits for in-flight

## 5. Integration with main.go

- [x] 5.1 Create Dispatcher in main() with config
- [x] 5.2 Replace blocking `processEvent()` call with `dispatcher.Dispatch()`
- [x] 5.3 Pass required dependencies to Dispatch (executor, notifier, etc.)
- [x] 5.4 Add dispatcher.Shutdown() to graceful shutdown sequence
- [x] 5.5 Update startup logging to show dispatcher config

## 6. Integration Tests

- [x] 6.1 Add integration test: concurrent events across clusters
- [x] 6.2 Add integration test: serialized events for same cluster
- [x] 6.3 Add integration test: shutdown with in-flight agents

## 7. Documentation

- [x] 7.1 Update configuration docs with new settings
- [x] 7.2 Add concurrency behavior to architecture docs
- [x] 7.3 Update README if needed

## Dependencies

- Tasks 1.x (Config) must complete before 3.x (Dispatcher Core)
- Task 2.x (Queue) must complete before 3.x (Dispatcher Core)
- Tasks 3.x must complete before 5.x (Integration)
- Tasks 4.x can run in parallel with 5.x

## Validation

After completing all tasks:
- [x] All unit tests pass
- [x] Integration tests pass
- [x] Manual testing with multi-cluster setup confirms parallel execution
- [x] Manual testing confirms same-cluster events are serialized
