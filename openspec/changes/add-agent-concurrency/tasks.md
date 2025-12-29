## 1. Core Components
- [ ] 1.1 Create `internal/dispatcher` package
- [ ] 1.2 Implement `Dispatcher` struct with `GlobalSemaphore` (buffered channel)
- [ ] 1.3 Implement `ClusterLocks` (map of mutexes) for per-cluster serialization
- [ ] 1.4 Implement `Dispatch(event)` method with non-blocking logic (goroutine spawn)

## 2. Integration
- [ ] 2.1 Refactor `processEvent` in `cmd/nightcrier/main.go` to separate setup (synchronous) from execution (async)
- [ ] 2.2 Update `main.go` to initialize `Dispatcher` with config values
- [ ] 2.3 Replace blocking `executor.Execute` call with `dispatcher.Dispatch`

## 3. Configuration & Validation
- [ ] 3.1 Verify `MaxConcurrentAgents` is correctly propagated from config
- [ ] 3.2 Add unit tests for `Dispatcher` concurrency logic (mock executor)
- [ ] 3.3 Add unit tests for locking behavior (same cluster vs different cluster)
