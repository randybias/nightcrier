# Implementation Tasks (Phase 3)

## 1. Package Structure and Models
- [ ] 1.1 Create `internal/reporting/` package directory structure
- [ ] 1.2 Define `ReportData` struct in `models.go` with all required fields (incident ID, timestamps, severity, agent output, etc.)
- [ ] 1.3 Define `SlackPayload` and related structs for Block Kit message format
- [ ] 1.4 Define `Reporter` interface with `GenerateReport`, `SendNotification`, and `Report` methods
- [ ] 1.5 Create `ReportingConfig` struct with validation tags for environment variables

## 2. Configuration and Validation
- [ ] 2.1 Implement config loading from environment variables (SLACK_WEBHOOK_URL, REPORT_ROOT_DIR, etc.)
- [ ] 2.2 Implement startup validation for required SLACK_WEBHOOK_URL (non-empty, valid HTTPS URL)
- [ ] 2.3 Implement startup validation for REPORT_ROOT_DIR (create if missing, check writability)
- [ ] 2.4 Add config validation unit tests (missing URL, malformed URL, non-writable directory)
- [ ] 2.5 Implement default values for optional config (timeout: 10s, max retries: 3)

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
- [ ] 5.1 Implement HTTP client with configurable timeout (default 10s)
- [ ] 5.2 Implement POST request to webhook URL with JSON payload
- [ ] 5.3 Implement response parsing and logging (log status code and body)
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
- [ ] 7.1 Implement Block Kit header block with incident ID and summary
- [ ] 7.2 Implement section block with metadata fields (severity, cluster, resource, duration)
- [ ] 7.3 Implement section block with key findings (limit to 3 bullet points)
- [ ] 7.4 Implement context block with report filesystem path in monospace
- [ ] 7.5 Implement message attachment wrapper with color field
- [ ] 7.6 Implement severity-to-color mapping (critical=red, warning=yellow, info=green, failure=purple)
- [ ] 7.7 Implement fallback text field for basic notification support
- [ ] 7.8 Add unit tests for success notification payload structure
- [ ] 7.9 Add unit tests for failure notification payload structure
- [ ] 7.10 Add unit tests for color mapping for each severity level

## 8. Notification Triggers
- [ ] 8.1 Implement `SendNotification()` method that builds Slack payload from ReportData
- [ ] 8.2 Implement notification trigger for successful agent completion (exit code 0)
- [ ] 8.3 Implement notification trigger for agent failure (exit code > 0)
- [ ] 8.4 Implement notification trigger for agent timeout
- [ ] 8.5 Ensure notification happens AFTER disk persistence (disk first, Slack second)
- [ ] 8.6 Add unit tests for notification payload generation for success case
- [ ] 8.7 Add unit tests for notification payload generation for failure case
- [ ] 8.8 Add unit tests for notification payload generation for timeout case

## 9. Error Handling and Resilience
- [ ] 9.1 Implement error logging for all disk write failures with full context
- [ ] 9.2 Implement error logging for all Slack failures with response details
- [ ] 9.3 Ensure Slack failures do not block report generation (best-effort notification)
- [ ] 9.4 Ensure disk failures do not prevent Slack notification attempt
- [ ] 9.5 Implement context timeout handling for report generation (prevent indefinite hangs)
- [ ] 9.6 Implement context timeout handling for Slack notification (prevent indefinite hangs)
- [ ] 9.7 Add integration test for report generation when Slack fails
- [ ] 9.8 Add integration test for Slack notification when disk write fails

## 10. Integration with Runner
- [ ] 10.1 Wire Reporter into main runner flow after agent process exits
- [ ] 10.2 Capture agent exit code and pass to ReportData
- [ ] 10.3 Capture agent stdout/stderr streams and pass to ReportData
- [ ] 10.4 Capture agent duration (start/end timestamps) and pass to ReportData
- [ ] 10.5 Pass incident context (ID, severity, cluster, namespace, resource) to ReportData
- [ ] 10.6 Implement goroutine for async report generation + notification (non-blocking)
- [ ] 10.7 Add logging for report generation start and completion
- [ ] 10.8 Add logging for notification send start and completion

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
