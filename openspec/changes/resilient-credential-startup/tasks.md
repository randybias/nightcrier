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
- [ ] Store bootstrap status for health/admin endpoints
- [x] Remove fatal exit on bootstrap failure

### Phase 6: Health Endpoint

- [ ] Update health endpoint to include bootstrap status
- [ ] Return degraded clusters list
- [ ] Return missing API keys list
- [ ] Add aggregate status (ready/degraded)

### Phase 7: Admin UI

- [ ] Add System Status section to admin UI header
- [ ] Display global status (Ready/Degraded/Initializing)
- [ ] Display API keys status with provider names
- [ ] Display cluster summary (X/Y ready)
- [ ] Add Bootstrap column to Clusters table
- [ ] Add Last Error column or tooltip to Clusters table
- [ ] Add visual indicator (color) for degraded clusters
- [ ] Ensure auto-refresh updates bootstrap status

### Phase 8: Observability

- [ ] Add periodic warning log when degraded (every 5 min)
- [ ] Log recovery events at INFO level
- [ ] Add metrics: `nightcrier_bootstrap_degraded_clusters`
- [ ] Add metrics: `nightcrier_bootstrap_retry_total`

### Phase 9: Testing

- [ ] Add unit tests for parallel bootstrap
- [ ] Add unit tests for exponential backoff
- [ ] Add unit tests for recovery detection
- [ ] Add integration test for degraded startup
- [ ] Add integration test for recovery scenario

## Notes

- Never block startup - key design principle
- Retriable errors: file not found, K8s API unavailable, network errors
- Fatal errors: none - all errors become degraded state
- Per-cluster retry is independent - don't retry all clusters together
- Max backoff 300s (5 minutes) balances responsiveness with API load
