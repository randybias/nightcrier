# Tasks: Prevent Spurious Notifications on Agent Failures

## Phase 1: Basic Validation (Minimal Fix)

### Task 1: Add agent execution validation helper
- [ ] Create `detectAgentFailure()` function in `cmd/runner/main.go`
- [ ] Check exit code is 0
- [ ] Verify `output/investigation.md` exists
- [ ] Verify investigation file size > 100 bytes
- [ ] Return failure reason string for logging
- [ ] Add unit tests for validation logic

**Dependencies**: None
**Validation**: Unit tests pass, function correctly identifies failure scenarios

### Task 2: Modify processEvent to detect agent failures
- [ ] Call `detectAgentFailure()` after agent execution
- [ ] Set status to `"agent_failed"` when validation fails
- [ ] Log WARNING with failure reason
- [ ] Update `result.json` schema to include failure_reason field

**Dependencies**: Task 1
**Validation**: Failed executions set correct status and log warnings

### Task 3: Skip storage upload on agent failure
- [ ] Add conditional check before `storageBackend.SaveIncident()`
- [ ] Only upload when status is `"success"` or `"failed"` (not `"agent_failed"`)
- [ ] Log INFO message explaining why upload was skipped
- [ ] Ensure result.json still written locally

**Dependencies**: Task 2
**Validation**: Azure storage not called when agent fails, local result.json still exists

### Task 4: Skip Slack notification on agent failure
- [ ] Add conditional check before `slackNotifier.SendIncidentNotification()`
- [ ] Only notify when status is not `"agent_failed"`
- [ ] Log INFO message explaining why notification was skipped
- [ ] Preserve notification behavior for other status values

**Dependencies**: Task 2
**Validation**: No Slack webhooks sent when agent fails, webhooks still sent for successful/failed executions

### Task 5: Integration testing for Phase 1
- [ ] Add integration test that simulates LLM API failure
- [ ] Mock agent execution to return exit code 1
- [ ] Verify no Slack notification sent
- [ ] Verify no Azure upload performed
- [ ] Verify result.json written with agent_failed status
- [ ] Test with real kind cluster (manual test)

**Dependencies**: Tasks 1-4
**Validation**: All integration tests pass

## Phase 2: Circuit Breaker (Enhanced)

### Task 6: Create circuit breaker package
- [ ] Create `internal/reporting/circuit_breaker.go`
- [ ] Define `CircuitBreaker` struct with failure counter and timestamps
- [ ] Implement `RecordFailure()` method
- [ ] Implement `RecordSuccess()` method (resets counter)
- [ ] Implement `ShouldAlert()` method (checks threshold)
- [ ] Thread-safe with mutex protection
- [ ] Add unit tests for circuit breaker logic

**Dependencies**: Phase 1 complete
**Validation**: Unit tests pass, thread-safety verified

### Task 7: Integrate circuit breaker in main loop
- [ ] Create circuit breaker instance in `run()` function
- [ ] Call `RecordFailure()` when status is `"agent_failed"`
- [ ] Call `RecordSuccess()` when status is `"success"`
- [ ] Check `ShouldAlert()` before sending aggregated alert
- [ ] Pass circuit breaker to `processEvent()` function

**Dependencies**: Task 6
**Validation**: Circuit breaker state updated correctly during event processing

### Task 8: Implement aggregated failure notification
- [ ] Create `SendSystemDegradedAlert()` in `internal/reporting/slack.go`
- [ ] Include failure count in message
- [ ] Include time window (first failure to current)
- [ ] Include sample failure reasons (last 3)
- [ ] Use distinct color/format (warning level)
- [ ] Add unit tests for message formatting

**Dependencies**: Tasks 6-7
**Validation**: Correct message format, includes all required data

### Task 9: Send circuit breaker alert
- [ ] In `processEvent()`, check if `ShouldAlert()` returns true after failure
- [ ] Call `SendSystemDegradedAlert()` when threshold reached
- [ ] Include failure count and time window
- [ ] Mark circuit breaker as alerted (don't spam)
- [ ] Add integration test for circuit breaker flow

**Dependencies**: Tasks 6-8
**Validation**: Single alert sent after N failures, not repeated

### Task 10: Add configuration options
- [ ] Add `NotifyOnAgentFailure` to `internal/config/config.go`
- [ ] Add `FailureThresholdForAlert` to config
- [ ] Add `UploadFailedInvestigations` to config
- [ ] Add env var bindings: `NOTIFY_ON_AGENT_FAILURE`, `FAILURE_THRESHOLD_FOR_ALERT`, `UPLOAD_FAILED_INVESTIGATIONS`
- [ ] Add CLI flag bindings
- [ ] Update `configs/config.example.yaml`
- [ ] Add config validation tests

**Dependencies**: None (can be done in parallel with Tasks 6-9)
**Validation**: Config values loaded correctly from env vars, flags, and YAML

### Task 11: Use configuration in notification logic
- [ ] Check `cfg.NotifyOnAgentFailure` before skipping notifications
- [ ] Use `cfg.FailureThresholdForAlert` in circuit breaker
- [ ] Check `cfg.UploadFailedInvestigations` before skipping storage
- [ ] Update integration tests to test config options

**Dependencies**: Tasks 4, 10
**Validation**: Configuration controls behavior as expected

### Task 12: Add circuit breaker recovery notification
- [ ] Detect when circuit breaker transitions from failed to success
- [ ] Send `SendSystemRecoveredAlert()` to Slack
- [ ] Include downtime duration
- [ ] Include total failures during outage
- [ ] Add unit tests for recovery notification

**Dependencies**: Tasks 6-9
**Validation**: Recovery notification sent on first success after failures

### Task 13: Integration testing for Phase 2
- [ ] Test circuit breaker with threshold=3
- [ ] Verify single alert after 3 failures
- [ ] Verify no further alerts until recovery
- [ ] Verify recovery notification on first success
- [ ] Test with different threshold values
- [ ] Test with real kind cluster and API failures

**Dependencies**: Tasks 6-12
**Validation**: All integration tests pass, manual testing successful

## Phase 3: Documentation and Cleanup

### Task 14: Update documentation
- [ ] Update README.md with new configuration options
- [ ] Document circuit breaker behavior
- [ ] Add troubleshooting section for agent failures
- [ ] Update `configs/config.example.yaml` with comments
- [ ] Add architecture diagram showing validation flow

**Dependencies**: Phase 2 complete
**Validation**: Documentation reviewed and accurate

### Task 15: Add operational metrics (future)
- [ ] Add Prometheus counter for agent_failures_total
- [ ] Add gauge for circuit_breaker_state
- [ ] Add histogram for agent_execution_success_rate
- [ ] Document metrics in README

**Dependencies**: Phase 2 complete
**Validation**: Metrics exported correctly, documented

## Verification Checklist

After implementation, verify:

- [ ] No Slack notifications during simulated LLM API outage (Phase 1: 0, Phase 2: 1 circuit breaker alert)
- [ ] No Azure uploads for failed agent executions (unless `UPLOAD_FAILED_INVESTIGATIONS=true`)
- [ ] result.json still written locally for all events
- [ ] Successful executions continue to notify and upload normally
- [ ] Circuit breaker sends exactly one alert after threshold
- [ ] Recovery notification sent when system returns to healthy state
- [ ] All configuration options work as documented
- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] Manual testing with kind cluster successful

## Parallel Work

Tasks that can be done in parallel:
- Task 10 (configuration) can be done anytime during Phase 2
- Task 15 (metrics) can be done independently after Phase 2
- Task 14 (documentation) can be drafted alongside implementation
