# Spec: Conditional Notifications

## ADDED Requirements

### Requirement: Conditional Slack Notifications

The system MUST NOT send Slack notifications when the agent execution failed to produce valid triage output.

#### Scenario: Skip notification on agent failure

**Given** an agent execution completed with status "agent_failed"
**When** the notification logic runs
**Then** NO Slack webhook call MUST be made
**And** an INFO level log MUST indicate the notification was skipped
**And** the log MUST include the reason: "agent failed to complete investigation"

#### Scenario: Send notification on successful investigation

**Given** an agent execution completed with status "success"
**When** the notification logic runs
**Then** a Slack webhook call MUST be made
**And** the notification MUST include the investigation summary

#### Scenario: Send notification on non-agent failures

**Given** an agent execution completed with status "failed" or "error"
**But** the status is NOT "agent_failed"
**When** the notification logic runs
**Then** a Slack webhook call MUST be made
**And** the notification MUST indicate the failure

### Requirement: Conditional Storage Upload

The system MUST NOT upload investigation artifacts to storage when the agent execution failed, unless explicitly configured to do so.

#### Scenario: Skip upload on agent failure

**Given** an agent execution completed with status "agent_failed"
**And** `UPLOAD_FAILED_INVESTIGATIONS` is false (default)
**When** the storage logic runs
**Then** NO storage upload MUST be performed
**And** an INFO level log MUST indicate the upload was skipped
**And** the log MUST include the incident_id

#### Scenario: Upload on successful investigation

**Given** an agent execution completed with status "success"
**When** the storage logic runs
**Then** the storage upload MUST be performed
**And** presigned URLs MUST be generated
**And** the URLs MUST be included in result.json

#### Scenario: Force upload failed investigations when configured

**Given** an agent execution completed with status "agent_failed"
**And** `UPLOAD_FAILED_INVESTIGATIONS` is true
**When** the storage logic runs
**Then** the storage upload MUST be performed
**And** the uploaded artifacts MUST include the failure metadata

### Requirement: Local Audit Trail Preservation

The system MUST always write result.json locally regardless of agent execution outcome.

#### Scenario: Local result always written

**Given** an agent execution completes with any status
**When** the result writing logic runs
**Then** result.json MUST be written to the workspace directory
**And** the file MUST include the status field
**And** the file MUST include start/completion timestamps

#### Scenario: Failed execution metadata captured

**Given** an agent execution completed with status "agent_failed"
**When** result.json is written
**Then** the file MUST include the failure_reason field
**And** the file MUST NOT include presigned_urls field (no uploads performed)

### Requirement: Circuit Breaker Alerting

The system MUST aggregate agent failures and send a single alert when a failure threshold is reached, rather than alerting on every failure.

#### Scenario: Track consecutive failures

**Given** the circuit breaker is in closed (healthy) state
**When** an agent execution completes with status "agent_failed"
**Then** the failure counter MUST be incremented
**And** the timestamp of the failure MUST be recorded

#### Scenario: Alert on threshold breach

**Given** the circuit breaker has recorded N-1 failures
**And** the threshold is N
**When** another agent execution completes with status "agent_failed"
**Then** the failure counter MUST reach N
**And** a system degraded alert MUST be sent to Slack
**And** the circuit breaker MUST enter "open" (degraded) state

#### Scenario: No repeated alerts while degraded

**Given** the circuit breaker is in open (degraded) state
**And** a system degraded alert has already been sent
**When** another agent execution completes with status "agent_failed"
**Then** NO additional alert MUST be sent
**And** the failure counter MUST continue incrementing

#### Scenario: Reset counter on success

**Given** the circuit breaker has recorded failures
**When** an agent execution completes with status "success"
**Then** the failure counter MUST be reset to zero
**And** the circuit breaker MUST return to closed (healthy) state

#### Scenario: Recovery notification

**Given** the circuit breaker is in open (degraded) state
**And** at least one system degraded alert was sent
**When** an agent execution completes with status "success"
**Then** a system recovered alert MUST be sent to Slack
**And** the alert MUST include the total downtime duration
**And** the alert MUST include the total failure count

### Requirement: Circuit Breaker Message Format

System degraded alerts MUST provide actionable information for operators.

#### Scenario: Degraded alert content

**Given** the circuit breaker sends a system degraded alert
**Then** the alert MUST include the text "AI Agent System Degraded"
**And** the alert MUST include the failure count
**And** the alert MUST include the time window (first failure to current)
**And** the alert MUST include up to 3 sample failure reasons
**And** the alert MUST use warning color (yellow/orange)

#### Scenario: Recovery alert content

**Given** the circuit breaker sends a system recovered alert
**Then** the alert MUST include the text "AI Agent System Recovered"
**And** the alert MUST include the total downtime duration
**And** the alert MUST include the total failures during outage
**And** the alert MUST use success color (green)

### Requirement: Configuration Options

The system MUST provide configuration to control notification and upload behavior for failed agent executions.

#### Scenario: Configure failure threshold

**Given** the configuration includes `FAILURE_THRESHOLD_FOR_ALERT=5`
**When** the circuit breaker is initialized
**Then** the threshold MUST be set to 5
**And** alerts MUST NOT be sent until 5 failures are recorded

#### Scenario: Configure upload behavior

**Given** the configuration includes `UPLOAD_FAILED_INVESTIGATIONS=true`
**When** an agent execution completes with status "agent_failed"
**Then** storage upload MUST be performed
**And** failed investigation artifacts MUST be uploaded

#### Scenario: Disable agent failure notifications entirely

**Given** the configuration includes `NOTIFY_ON_AGENT_FAILURE=false`
**When** an agent execution completes with status "agent_failed"
**Then** NO circuit breaker alert MUST be sent
**And** NO individual failure notification MUST be sent
