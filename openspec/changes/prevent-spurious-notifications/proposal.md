# Change: Prevent Spurious Notifications on Agent Failures

## Problem Statement

When the triage agent cannot access the LLM API or experiences other critical failures, the system currently proceeds to send Slack notifications and upload artifacts to Azure Blob Storage. This results in:

- **Notification Spam**: Slack channels flooded with error notifications for every event when the LLM is unavailable
- **Storage Waste**: Failed triage attempts uploaded to Azure, consuming storage and generating unnecessary SAS URLs
- **Alert Fatigue**: Operations teams receive notifications about failures that provide no actionable information
- **Misleading Reports**: Notifications may contain placeholder or error content instead of actual triage results

The root cause is in `cmd/runner/main.go:processEvent()` (lines 254-286) where Slack notifications and storage uploads happen unconditionally after agent execution, regardless of whether the agent successfully completed its investigation.

## Why

Production incident response systems must be reliable signal sources. When the AI backend is unavailable, operators need to know *once* that the system is degraded, not receive a notification for every single Kubernetes event. The current behavior:

1. **Obscures Real Issues**: Spam from LLM failures drowns out legitimate incident alerts
2. **Wastes Resources**: Storage and notification services are invoked for non-investigations
3. **Degrades Confidence**: Teams lose trust in the system when it cries wolf repeatedly
4. **Complicates Ops**: Manual cleanup required for Azure storage and Slack thread management

## What Changes

### 1. Agent Execution Validation

Add validation to detect when the agent failed to produce meaningful output:

- Check if `output/investigation.md` exists and has substantial content (>100 bytes)
- Verify the agent exit code indicates success (exit code 0)
- Detect common LLM API error patterns in agent logs
- Set a new status value: `"agent_failed"` distinct from `"error"` or `"failed"`

**Code Location**: `cmd/runner/main.go:processEvent()` lines 192-203

### 2. Conditional Notification Logic

Modify notification and storage logic to skip when agent execution is invalid:

- Skip Slack notification when status is `"agent_failed"` or investigation file is missing/empty
- Skip Azure storage upload when no valid investigation was produced
- Log a single WARN message explaining why notifications were suppressed
- Still write `result.json` locally for audit purposes

**Code Locations**:
- `cmd/runner/main.go:processEvent()` lines 254-286
- `internal/reporting/slack.go:SendIncidentNotification()` line 88

### 3. Failure Mode Detection

Add helper function to detect common agent failure patterns:

```go
// detectAgentFailure checks if the agent failed to complete a valid investigation
func detectAgentFailure(workspacePath string, exitCode int, err error) (bool, string) {
    // Check exit code
    if exitCode != 0 {
        return true, fmt.Sprintf("non-zero exit code: %d", exitCode)
    }

    // Check if investigation file exists and has content
    investigationPath := filepath.Join(workspacePath, "output", "investigation.md")
    stat, err := os.Stat(investigationPath)
    if err != nil || stat.Size() < 100 {
        return true, "investigation report missing or too small"
    }

    // Check for LLM API error markers in logs
    // (Implementation detail: scan agent logs for known error patterns)

    return false, ""
}
```

**New File**: Could go in `internal/agent/validation.go` or inline in `cmd/runner/main.go`

### 4. Aggregated Failure Notifications

Instead of per-event spam, implement a circuit-breaker pattern:

- Track consecutive agent failures (in-memory counter)
- Send a single Slack alert when failure count crosses threshold (e.g., 3 failures)
- Include failure count and time window in the alert
- Reset counter on first successful agent execution
- Log failure metrics for observability

**New File**: `internal/reporting/circuit_breaker.go`

### 5. Configuration

Add configuration options:

- `NOTIFY_ON_AGENT_FAILURE` (bool, default: false) - Whether to notify on agent failures at all
- `FAILURE_THRESHOLD_FOR_ALERT` (int, default: 3) - How many failures before circuit breaker alert
- `UPLOAD_FAILED_INVESTIGATIONS` (bool, default: false) - Whether to upload failed attempts to storage

**Code Location**: `internal/config/config.go`

## Impact

### Enhanced Capabilities
- `agent-execution` - Add validation and failure detection
- `reporting` - Add conditional notification logic and circuit breaker

### Modified Code
- `cmd/runner/main.go` - Add validation and conditional logic in processEvent()
- `internal/reporting/slack.go` - Skip notification when appropriate
- `internal/config/config.go` - Add new configuration fields

### New Code
- `internal/reporting/circuit_breaker.go` - Circuit breaker for aggregated failure alerts
- `internal/agent/validation.go` - Agent execution validation helpers

### Dependencies
None - purely internal logic changes

## Implementation Approach

### Phase 1: Basic Validation (Minimal Fix)
1. Add `detectAgentFailure()` helper in `cmd/runner/main.go`
2. Modify `processEvent()` to skip notifications when agent failed
3. Still write result.json but skip Slack and Azure uploads
4. Add structured logging for suppressed notifications

**Impact**: Immediately stops notification spam with minimal code changes

### Phase 2: Circuit Breaker (Enhanced)
1. Implement `CircuitBreaker` type in `internal/reporting/circuit_breaker.go`
2. Track failure counts and send aggregated alerts
3. Add configuration options
4. Add metrics for failure tracking

**Impact**: Provides operators with degradation visibility without spam

## Risks and Mitigations

**Risk**: Operators might miss legitimate failures if validation is too aggressive
**Mitigation**:
- Keep validation simple (file exists, exit code 0, file size >100 bytes)
- Always log locally to result.json for audit trail
- Make behavior configurable via `NOTIFY_ON_AGENT_FAILURE`

**Risk**: Circuit breaker state lost on restart
**Mitigation**:
- Keep state in-memory only (acceptable for this use case)
- Restart resets counter, which is reasonable behavior
- Could persist to disk in future if needed

## Testing Strategy

1. **Unit Tests**:
   - Test `detectAgentFailure()` with various workspace states
   - Test circuit breaker threshold logic
   - Test conditional notification logic

2. **Integration Tests**:
   - Mock LLM API failure and verify no notifications sent
   - Verify circuit breaker sends single alert after threshold
   - Verify successful executions still send notifications

3. **Manual Testing**:
   - Deploy with invalid API key and generate events
   - Verify Slack channel receives 0 or 1 notification (not N)
   - Verify Azure storage not polluted with failed attempts

## Success Criteria

- Zero Slack notifications during LLM API outage (with circuit breaker: exactly 1)
- Zero Azure uploads for failed agent executions
- result.json still written locally for all events (audit trail)
- Successful executions continue to notify and upload normally
- Operators alerted once when system enters degraded state
