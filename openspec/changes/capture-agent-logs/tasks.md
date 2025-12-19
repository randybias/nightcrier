# Implementation Tasks: Capture Agent Container Logs

## 1. Executor Log Capture
- [ ] 1.1 Create `LogCapture` struct in `internal/agent/executor.go` to manage log files
- [ ] 1.2 Implement `NewLogCapture(workspacePath string)` to create log directory and files
- [ ] 1.3 Create `{workspace}/logs/` directory in executor before running agent
- [ ] 1.4 Write stdout to `{workspace}/logs/agent-stdout.log` using `io.TeeReader`
- [ ] 1.5 Write stderr to `{workspace}/logs/agent-stderr.log` using `io.TeeReader`
- [ ] 1.6 Create combined `agent-full.log` with timestamped interleaved output
- [ ] 1.7 Add `LogPaths` struct to return log file paths from `Execute()`
- [ ] 1.8 Continue logging snippets via slog for real-time visibility
- [ ] 1.9 Write unit tests for log capture functionality
- [ ] 1.10 Handle log file cleanup on executor errors (close files properly)

## 2. Storage Interface Updates
- [ ] 2.1 Add `AgentLogs` field to `IncidentArtifacts` struct in `storage.go`
- [ ] 2.2 Define `AgentLogs` struct with `Stdout`, `Stderr`, `Combined` byte slices
- [ ] 2.3 Add `LogURLs` map to `SaveResult` struct for log artifact URLs
- [ ] 2.4 Update interface documentation with log fields

## 3. Azure Storage Log Upload
- [ ] 3.1 Modify `SaveIncident()` to upload log files when present
- [ ] 3.2 Upload logs to `{incident-id}/logs/agent-stdout.log` path
- [ ] 3.3 Upload logs to `{incident-id}/logs/agent-stderr.log` path
- [ ] 3.4 Upload logs to `{incident-id}/logs/agent-full.log` path
- [ ] 3.5 Generate SAS URLs for each log file
- [ ] 3.6 Populate `LogURLs` in `SaveResult`
- [ ] 3.7 Handle empty log files gracefully (skip upload if empty)
- [ ] 3.8 Write integration test for log upload

## 4. Filesystem Storage Log Handling
- [ ] 4.1 Modify `SaveIncident()` to copy log files to storage location
- [ ] 4.2 Create `logs/` subdirectory in incident storage directory
- [ ] 4.3 Return local paths as `LogURLs` in `SaveResult`
- [ ] 4.4 Write unit test for filesystem log storage

## 5. Main Application Integration
- [ ] 5.1 Update `readIncidentArtifacts()` in `main.go` to read log files
- [ ] 5.2 Look for logs in `{workspace}/logs/` directory
- [ ] 5.3 Read `agent-stdout.log`, `agent-stderr.log`, `agent-full.log`
- [ ] 5.4 Populate `AgentLogs` in `IncidentArtifacts`
- [ ] 5.5 Handle missing log files gracefully (logs are optional)

## 6. Slack Notification Updates
- [ ] 6.1 Add `LogURLs` field to `IncidentSummary` struct
- [ ] 6.2 Update `SendIncidentNotification()` to include log links
- [ ] 6.3 Add "View Logs" link to Slack message when log URLs are available
- [ ] 6.4 Write unit test for Slack message with logs

## 7. Result Metadata Updates
- [ ] 7.1 Add `LogPaths` field to `Result` struct in `reporting/result.go`
- [ ] 7.2 Populate log paths in `result.json` for local reference
- [ ] 7.3 Add presigned log URLs to result when available

## 8. Documentation and Verification
- [ ] 8.1 Update `configs/config.example.yaml` with any new options
- [ ] 8.2 Run all unit tests
- [ ] 8.3 Test end-to-end with real agent execution
- [ ] 8.4 Verify logs appear in Azure Blob Storage
- [ ] 8.5 Verify Slack notification includes log link
- [ ] 8.6 Test with and without Azure storage configured

---

## Parallelization Notes

- Tasks 1 and 2 can proceed in parallel
- Task 3 depends on Task 2
- Task 4 depends on Task 2
- Task 5 depends on Tasks 1 and 2
- Task 6 depends on Task 5
- Task 7 depends on Tasks 1 and 5
- Task 8 depends on all previous tasks
