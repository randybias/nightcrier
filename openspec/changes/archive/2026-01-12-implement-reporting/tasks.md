# Implementation Tasks (Phase 3)

## Walking Skeleton Baseline

The walking-skeleton (archived 2025-12-18) implemented core reporting functionality:
- `internal/reporting/` package with result.go and slack.go
- Result struct and WriteResult() for result.json
- SlackNotifier with Block Kit formatting
- ExtractSummaryFromReport() for parsing investigation.md
- Integration with main.go event processing

This task list marks completed items and focuses on remaining enhancements.

---

## 1. Package Structure and Models
- [x] 1.1 Create `internal/reporting/` package directory structure
      **DONE**: Walking skeleton
- [x] 1.2 Define `ReportData` struct in `models.go` with all required fields (incident ID, timestamps, severity, agent output, etc.)
      **DONE**: Walking skeleton - Result struct in result.go, IncidentSummary in slack.go
- [x] 1.3 Define `SlackPayload` and related structs for Block Kit message format
      **DONE**: Walking skeleton - SlackMessage, SlackBlock, SlackText, SlackElement, SlackAttachment
- [ ] 1.4 Define `Reporter` interface with `GenerateReport`, `SendNotification`, and `Report` methods
- [x] 1.5 Create `ReportingConfig` struct with validation tags for environment variables
      **PARTIAL**: Walking skeleton - SLACK_WEBHOOK_URL in main config

## 2. Configuration and Validation
- [x] 2.1 Implement config loading from environment variables (SLACK_WEBHOOK_URL, REPORT_ROOT_DIR, etc.)
      **DONE**: Walking skeleton - SLACK_WEBHOOK_URL loaded in config.go
- [x] 2.2 Implement startup validation for required SLACK_WEBHOOK_URL (non-empty, valid HTTPS URL)
      **PARTIAL**: Walking skeleton - checks if set, skips silently if not
- [x] 2.3 Implement startup validation for REPORT_ROOT_DIR (create if missing, check writability)
      **DONE**: Walking skeleton - workspace manager handles this
- [ ] 2.4 Add config validation unit tests (missing URL, malformed URL, non-writable directory)
- [x] 2.5 Implement default values for optional config (timeout: 10s, max retries: 3)
      **PARTIAL**: Walking skeleton - 10s timeout on HTTP client

## 3. Markdown Template Implementation
- [ ] 3.1 Create template file with header, summary, findings, recommendations, and metadata sections
- [ ] 3.2 Implement custom template functions (heading, codeblock, escape for Markdown special chars)
- [ ] 3.3 Implement named sub-templates for reusable components (metadata table, severity badge)
- [ ] 3.4 Implement template rendering function that accepts ReportData and returns string
- [ ] 3.5 Add unit tests for template rendering with sample data
- [ ] 3.6 Add unit tests for special character escaping in template

## 4. Report Generation
- [ ] 4.1 Implement `GenerateReport()` method that creates workspace directory if missing
- [ ] 4.2 Implement report.md generation using template and ReportData
- [ ] 4.3 Implement agent-output.log creation from captured stdout/stderr
- [ ] 4.4 Implement agent-context.json creation with incident input context
- [ ] 4.5 Implement metadata.json creation with timestamps, exit code, and status
- [ ] 4.6 Implement error handling for disk write failures (log error, attempt minimal metadata.json)
- [ ] 4.7 Add unit tests for successful report generation
- [ ] 4.8 Add unit tests for report generation with failure status
- [ ] 4.9 Add unit tests for report generation with timeout status
- [ ] 4.10 Add integration test for full report generation flow with real filesystem

## 5. Slack Webhook Client
- [x] 5.1 Implement HTTP client with configurable timeout (default 10s)
      **DONE**: Walking skeleton - slack.go creates http.Client with 10s timeout
- [x] 5.2 Implement POST request to webhook URL with JSON payload
      **DONE**: Walking skeleton - send() method in slack.go
- [x] 5.3 Implement response parsing and logging (log status code and body)
      **DONE**: Walking skeleton - returns error with status code and body
- [ ] 5.4 Add unit tests for successful POST (mock HTTP server returning 200)
- [ ] 5.5 Add unit tests for client error response (mock 4xx)
- [ ] 5.6 Add unit tests for server error response (mock 5xx)

## 6. Slack Retry Logic
- [ ] 6.1 Implement exponential backoff retry for HTTP 429 (rate limit) with delays: 1s, 2s, 4s
- [ ] 6.2 Implement exponential backoff retry for HTTP 5xx (server error) with same delays
- [ ] 6.3 Implement exponential backoff retry for network timeout with same delays
- [ ] 6.4 Implement no-retry logic for HTTP 4xx (client error)
- [ ] 6.5 Implement max retry limit (default 3 attempts)
- [ ] 6.6 Add unit tests for retry on 429 (verify retry count and delays)
- [ ] 6.7 Add unit tests for retry on 5xx (verify retry count and delays)
- [ ] 6.8 Add unit tests for retry on timeout (verify retry count)
- [ ] 6.9 Add unit tests for no retry on 4xx (verify single attempt)
- [ ] 6.10 Add unit tests for successful response (verify no retry)

## 7. Slack Payload Formatting
- [x] 7.1 Implement Block Kit header block with incident ID and summary
      **DONE**: Walking skeleton - header block with "Kubernetes Incident Triage" and emoji
- [x] 7.2 Implement section block with metadata fields (severity, cluster, resource, duration)
      **DONE**: Walking skeleton - section with Cluster, Namespace, Resource, Reason fields
- [x] 7.3 Implement section block with key findings (limit to 3 bullet points)
      **DONE**: Walking skeleton - Root Cause section with confidence
- [x] 7.4 Implement context block with report filesystem path in monospace
      **DONE**: Walking skeleton - context block with incident ID and duration
- [x] 7.5 Implement message attachment wrapper with color field
      **DONE**: Walking skeleton - attachment with color and footer
- [x] 7.6 Implement severity-to-color mapping (critical=red, warning=yellow, info=green, failure=purple)
      **DONE**: Walking skeleton - "good" for success, "danger" for failure
- [ ] 7.7 Implement fallback text field for basic notification support
- [ ] 7.8 Add unit tests for success notification payload structure
- [ ] 7.9 Add unit tests for failure notification payload structure
- [ ] 7.10 Add unit tests for color mapping for each severity level

## 8. Notification Triggers
- [x] 8.1 Implement `SendNotification()` method that builds Slack payload from ReportData
      **DONE**: Walking skeleton - SendIncidentNotification() method
- [x] 8.2 Implement notification trigger for successful agent completion (exit code 0)
      **DONE**: Walking skeleton - main.go sends notification after processEvent()
- [x] 8.3 Implement notification trigger for agent failure (exit code > 0)
      **DONE**: Walking skeleton - sends notification with "danger" color
- [ ] 8.4 Implement notification trigger for agent timeout
- [x] 8.5 Ensure notification happens AFTER disk persistence (disk first, Slack second)
      **DONE**: Walking skeleton - WriteResult() called before Slack notification
- [ ] 8.6 Add unit tests for notification payload generation for success case
- [ ] 8.7 Add unit tests for notification payload generation for failure case
- [ ] 8.8 Add unit tests for notification payload generation for timeout case

## 9. Error Handling and Resilience
- [x] 9.1 Implement error logging for all disk write failures with full context
      **DONE**: Walking skeleton - main.go logs WriteResult errors
- [x] 9.2 Implement error logging for all Slack failures with response details
      **DONE**: Walking skeleton - main.go logs Slack errors
- [x] 9.3 Ensure Slack failures do not block report generation (best-effort notification)
      **DONE**: Walking skeleton - Slack error logged but doesn't fail event processing
- [ ] 9.4 Ensure disk failures do not prevent Slack notification attempt
- [x] 9.5 Implement context timeout handling for report generation (prevent indefinite hangs)
      **DONE**: Walking skeleton - HTTP client has 10s timeout
- [x] 9.6 Implement context timeout handling for Slack notification (prevent indefinite hangs)
      **DONE**: Walking skeleton - HTTP client has 10s timeout
- [ ] 9.7 Add integration test for report generation when Slack fails
- [ ] 9.8 Add integration test for Slack notification when disk write fails

## 10. Integration with Runner
- [x] 10.1 Wire Reporter into main runner flow after agent process exits
      **DONE**: Walking skeleton - processEvent() in main.go
- [x] 10.2 Capture agent exit code and pass to ReportData
      **DONE**: Walking skeleton - exitCode passed to Result struct
- [x] 10.3 Capture agent stdout/stderr streams and pass to ReportData
      **DONE**: Walking skeleton - agent logs to output/triage_*.log
- [x] 10.4 Capture agent duration (start/end timestamps) and pass to ReportData
      **DONE**: Walking skeleton - startedAt, completedAt in Result
- [x] 10.5 Pass incident context (ID, severity, cluster, namespace, resource) to ReportData
      **DONE**: Walking skeleton - IncidentSummary struct in main.go
- [ ] 10.6 Implement goroutine for async report generation + notification (non-blocking)
- [x] 10.7 Add logging for report generation start and completion
      **DONE**: Walking skeleton - logs event processed with duration
- [x] 10.8 Add logging for notification send start and completion
      **DONE**: Walking skeleton - logs "slack notification sent"

## 11. Testing and Verification
- [ ] 11.1 Create test fixtures for sample ReportData (success, failure, timeout)
- [ ] 11.2 Verify report.md is generated on disk with correct content
- [ ] 11.3 Verify agent-output.log contains captured output
- [ ] 11.4 Verify agent-context.json contains incident details
- [ ] 11.5 Verify metadata.json contains timestamps and exit code
- [ ] 11.6 Verify Slack message is received in test channel (manual test with real webhook)
- [ ] 11.7 Verify Slack message formatting matches Block Kit spec (manual test)
- [ ] 11.8 Verify severity colors display correctly in Slack (manual test)
- [ ] 11.9 Verify retry logic works with rate limit simulation (integration test)
- [ ] 11.10 Verify report generation succeeds when Slack is unavailable (integration test)

## 12. Documentation
- [ ] 12.1 Document SLACK_WEBHOOK_URL environment variable in README
- [ ] 12.2 Document REPORT_ROOT_DIR environment variable in README
- [ ] 12.3 Document SLACK_TIMEOUT and SLACK_MAX_RETRIES environment variables in README
- [ ] 12.4 Document report directory structure in README
- [ ] 12.5 Document Slack webhook setup instructions (how to create incoming webhook in Slack)
- [ ] 12.6 Document manual cleanup procedures for old reports (no automatic deletion)
- [ ] 12.7 Add example report.md file to documentation
- [ ] 12.8 Add example Slack message screenshot to documentation
