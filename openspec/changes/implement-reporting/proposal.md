# Change: Implement Reporting (Phase 3)

## Why
To make the agent's work visible and actionable by generating readable reports, saving artifacts, and notifying the Ops team via Slack.

## What Changes
- Implement `reporting` capability:
    - Markdown report generation (template-based).
    - Artifact collection (logs, outputs) to disk.
    - Slack webhook client.
    - Notification dispatch logic.

## Impact
- **New Capabilities**: `reporting`.
- **New Code**: `internal/reporting` package.
- **Dependencies**: Requires Slack webhook URL.
