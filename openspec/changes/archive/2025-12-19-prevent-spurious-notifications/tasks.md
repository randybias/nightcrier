# Tasks: Prevent Spurious Notifications on Agent Failures

## Phase 1: Basic Validation (Minimal Fix)

### Task 1: Add agent execution validation helper
- [x] Create `detectAgentFailure()` function in `cmd/runner/main.go`
- [x] Check exit code is 0
- [x] Verify `output/investigation.md` exists
- [x] Verify investigation file size > 100 bytes
- [x] Return failure reason string for logging
- [x] Add unit tests for validation logic

**Dependencies**: None
**Validation**: Unit tests pass, function correctly identifies failure scenarios

### Task 2: Modify processEvent to detect agent failures
- [x] Call `detectAgentFailure()` after agent execution
- [x] Set status to `"agent_failed"` when validation fails
- [x] Log WARNING with failure reason
- [x] Update `result.json` schema to include failure_reason field

**Dependencies**: Task 1
**Validation**: Failed executions set correct status and log warnings

### Task 3: Skip storage upload on agent failure
- [x] Add conditional check before `storageBackend.SaveIncident()`
- [x] Only upload when status is `"success"` or `"failed"` (not `"agent_failed"`)
- [x] Log INFO message explaining why upload was skipped
- [x] Ensure result.json still written locally

**Dependencies**: Task 2
**Validation**: Azure storage not called when agent fails, local result.json still exists

### Task 4: Skip Slack notification on agent failure
- [x] Add conditional check before `slackNotifier.SendIncidentNotification()`
- [x] Only notify when status is not `"agent_failed"`
- [x] Log INFO message explaining why notification was skipped
- [x] Preserve notification behavior for other status values

**Dependencies**: Task 2
**Validation**: No Slack webhooks sent when agent fails, webhooks still sent for successful/failed executions

### Task 5: Integration testing for Phase 1
- [x] Add integration test that simulates LLM API failure
- [x] Mock agent execution to return exit code 1
- [x] Verify no Slack notification sent
- [x] Verify no Azure upload performed
- [x] Verify result.json written with agent_failed status
- [x] Test with real kind cluster (manual test)

**Dependencies**: Tasks 1-4
**Validation**: All integration tests pass

## Phase 2: Circuit Breaker (Enhanced)

### Task 6: Create circuit breaker package
- [x] Create `internal/reporting/circuit_breaker.go`
- [x] Define `CircuitBreaker` struct with failure counter and timestamps
- [x] Implement `RecordFailure()` method
- [x] Implement `RecordSuccess()` method (resets counter)
- [x] Implement `ShouldAlert()` method (checks threshold)
- [x] Thread-safe with mutex protection
- [x] Add unit tests for circuit breaker logic

**Dependencies**: Phase 1 complete
**Validation**: Unit tests pass, thread-safety verified

### Task 7: Integrate circuit breaker in main loop
- [x] Create circuit breaker instance in `run()` function
- [x] Call `RecordFailure()` when status is `"agent_failed"`
- [x] Call `RecordSuccess()` when status is `"success"`
- [x] Check `ShouldAlert()` before sending aggregated alert
- [x] Pass circuit breaker to `processEvent()` function

**Dependencies**: Task 6
**Validation**: Circuit breaker state updated correctly during event processing

### Task 8: Implement aggregated failure notification
- [x] Create `SendSystemDegradedAlert()` in `internal/reporting/slack.go`
- [x] Include failure count in message
- [x] Include time window (first failure to current)
- [x] Include sample failure reasons (last 3)
- [x] Use distinct color/format (warning level)
- [x] Add unit tests for message formatting

**Dependencies**: Tasks 6-7
**Validation**: Correct message format, includes all required data

### Task 9: Send circuit breaker alert
- [x] In `processEvent()`, check if `ShouldAlert()` returns true after failure
- [x] Call `SendSystemDegradedAlert()` when threshold reached
- [x] Include failure count and time window
- [x] Mark circuit breaker as alerted (don't spam)
- [x] Add integration test for circuit breaker flow

**Dependencies**: Tasks 6-8
**Validation**: Single alert sent after N failures, not repeated

### Task 10: Add configuration options
- [x] Add `NotifyOnAgentFailure` to `internal/config/config.go`
- [x] Add `FailureThresholdForAlert` to config
- [x] Add `UploadFailedInvestigations` to config
- [x] Add env var bindings: `NOTIFY_ON_AGENT_FAILURE`, `FAILURE_THRESHOLD_FOR_ALERT`, `UPLOAD_FAILED_INVESTIGATIONS`
- [x] Add CLI flag bindings
- [x] Update `configs/config.example.yaml`
- [x] Add config validation tests

**Dependencies**: None (can be done in parallel with Tasks 6-9)
**Validation**: Config values loaded correctly from env vars, flags, and YAML

### Task 11: Use configuration in notification logic
- [x] Check `cfg.NotifyOnAgentFailure` before skipping notifications
- [x] Use `cfg.FailureThresholdForAlert` in circuit breaker
- [x] Check `cfg.UploadFailedInvestigations` before skipping storage
- [x] Update integration tests to test config options

**Dependencies**: Tasks 4, 10
**Validation**: Configuration controls behavior as expected

### Task 12: Add circuit breaker recovery notification
- [x] Detect when circuit breaker transitions from failed to success
- [x] Send `SendSystemRecoveredAlert()` to Slack
- [x] Include downtime duration
- [x] Include total failures during outage
- [x] Add unit tests for recovery notification

**Dependencies**: Tasks 6-9
**Validation**: Recovery notification sent on first success after failures

### Task 13: Integration testing for Phase 2
- [x] Test circuit breaker with threshold=3
- [x] Verify single alert after 3 failures
- [x] Verify no further alerts until recovery
- [x] Verify recovery notification on first success
- [x] Test with different threshold values
- [x] Test with real kind cluster and API failures

**Dependencies**: Tasks 6-12
**Validation**: All integration tests pass, manual testing successful

## Phase 3: Documentation and Cleanup

### Task 14: Update documentation
- [x] Update README.md with new configuration options
- [x] Document circuit breaker behavior
- [x] Add troubleshooting section for agent failures
- [x] Update `configs/config.example.yaml` with comments
- [x] Add architecture diagram showing validation flow

**Dependencies**: Phase 2 complete
**Validation**: Documentation reviewed and accurate

### Task 15: Add operational metrics (moved to ROADMAP.md)
- [x] Moved to ROADMAP.md as future enhancement
- [x] Add Prometheus counter for agent_failures_total - DEFERRED
- [x] Add gauge for circuit_breaker_state - DEFERRED
- [x] Add histogram for agent_execution_success_rate - DEFERRED
- [x] Document metrics in README - DEFERRED

**Dependencies**: Phase 2 complete
**Status**: Deferred to future work, documented in ROADMAP.md

## Verification Checklist

After implementation, verify:

- [x] No Slack notifications during simulated LLM API outage (Phase 1: 0, Phase 2: 1 circuit breaker alert)
- [x] No Azure uploads for failed agent executions (unless `UPLOAD_FAILED_INVESTIGATIONS=true`)
- [x] result.json still written locally for all events
- [x] Successful executions continue to notify and upload normally
- [x] Circuit breaker sends exactly one alert after threshold
- [x] Recovery notification sent when system returns to healthy state
- [x] All configuration options work as documented
- [x] All unit tests pass
- [x] All integration tests pass
- [x] Manual testing with kind cluster successful

## Parallel Work

Tasks that can be done in parallel:
- Task 10 (configuration) can be done anytime during Phase 2
- Task 15 (metrics) can be done independently after Phase 2
- Task 14 (documentation) can be drafted alongside implementation
