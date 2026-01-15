# Change: Implement Reporting (Phase 3)

## Walking Skeleton Baseline

The walking-skeleton implementation (archived 2025-12-18) provides core reporting functionality:

**Already Implemented:**
- `internal/reporting/` package with result.go and slack.go
- Result struct with incident_id, exit_code, started_at, completed_at, status
- WriteResult() for result.json generation
- SlackNotifier with webhook client
- Block Kit message formatting (header, sections, context, attachments)
- ExtractSummaryFromReport() to parse investigation.md for root cause and confidence
- Severity-to-color mapping (success=good, failure=danger)

**This Change Adds:**
Enhanced reporting features: markdown templates, retry logic, additional artifact files, and comprehensive testing.

## Why
To make the agent's work visible and actionable by generating readable reports, saving artifacts, and notifying the Ops team via Slack.

## What Changes
- Enhance `reporting` capability (builds on walking skeleton):
    - Markdown report generation (template-based) - agent generates investigation.md
    - **DONE**: Artifact collection (result.json, investigation.md) to disk.
    - **DONE**: Slack webhook client.
    - **DONE**: Notification dispatch logic.
    - Retry logic with exponential backoff for Slack.
    - Additional artifact files (metadata.json, agent-context.json).

## Impact
- **Enhanced Capabilities**: `reporting` (builds on walking skeleton).
- **Modified Code**: `internal/reporting` package enhancements.
- **Dependencies**: Requires Slack webhook URL (optional - skips if not set).
