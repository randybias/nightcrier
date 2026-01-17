# Tasks: Resilient Credential Startup

## Implementation Order

### Phase 1: Bootstrap Status Types

- [x] Create `internal/bootstrap/status.go` with status types
- [x] Add `ClusterBootstrapStatus` struct (Name, Ready, Error, LastRetry, Retries)
- [x] Add `BootstrapStatus` struct (GlobalReady, APIKeysReady, ClusterStatuses map)
- [x] Add `BootstrapState` enum: Ready, Degraded, Retrying

### Phase 2: Configuration

- [x] Add `startup` section to Config struct
- [x] Add `credential_retry_initial` (default: 5s)
- [x] Add `credential_retry_max` (default: 300s)
- [x] Add environment variable mappings
- [x] Update config-example files

### Phase 3: Non-Blocking Bootstrap

- [x] Create `internal/bootstrap/retry.go` with exponential backoff helper
- [x] Modify `Bootstrap()` to return `*BootstrapStatus` instead of `*Result`
- [x] Make `ensureAPIKeysSecret` non-fatal - return status, not error
- [x] Make `ensureTriageKubeconfigSecret` non-fatal - return status, not error
- [x] Bootstrap global resources first, then per-cluster in parallel
- [x] Use `errgroup` for parallel per-cluster bootstrap
- [x] Never block - always return status and start

### Phase 4: Background Retry

- [x] Create `internal/bootstrap/background.go` for retry goroutine
- [x] Add `StartBackgroundRetry(ctx, status, config)` function
- [x] Implement per-cluster independent retry loops
- [x] Implement global resource retry loop
- [x] Add recovery detection and status update
- [x] Add recovery logging

### Phase 5: Main Integration

- [x] Modify main.go to handle `BootstrapStatus` instead of error
- [x] Start background retry goroutine
- [x] Remove fatal exit on bootstrap failure

### Phase 6: Admin UI Cluster Status

- [x] Add cluster reachability tracking to database (connection_status, unreachable, unreachable_reason, last_error)
- [x] Display cluster connection state in admin UI
- [x] Add visual indicator for unreachable clusters
- [x] Start admin UI before permission validation (early availability)

### Phase 7: Connection Resilience

- [x] Detect permanent auth errors (Unauthorized, Forbidden, certificate) and skip to max backoff
- [x] Add tests for permanent vs transient error classification

### Phase 8: Testing

- [x] Add unit tests for parallel bootstrap
- [x] Add unit tests for exponential backoff
- [x] Add unit tests for recovery detection
- [x] Add integration test for degraded startup
- [x] Add integration test for recovery scenario

## Notes

- Never block startup - key design principle
- Retriable errors: file not found, K8s API unavailable, network errors
- Fatal errors: none - all errors become degraded state
- Per-cluster retry is independent - don't retry all clusters together
- Max backoff 300s (5 minutes) balances responsiveness with API load
- Permanent auth errors (Unauthorized, etc.) skip immediately to max backoff
