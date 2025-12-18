# Design: Reporting and Notification System

## Context

After the AI agent completes its triage investigation, the runner needs to:
1. Generate a human-readable summary report
2. Persist all artifacts (agent outputs, logs) to disk for audit trail
3. Notify the operations team via Slack webhook

This reporting system bridges the gap between the agent's automated analysis and human operators who need to review findings and take action. The design prioritizes clarity, reliability, and audit trail preservation.

## Goals
- **Actionable Reports**: Generate structured Markdown reports that ops teams can quickly scan for severity, findings, and recommendations
- **Durable Artifacts**: Persist all investigation data to local filesystem with clear organization
- **Timely Notifications**: Send Slack alerts immediately upon investigation completion or failure
- **Audit Trail**: Maintain complete history of all investigations with timestamps and metadata

## Non-Goals
- Email notifications (Slack webhook only)
- Interactive Slack components or slash commands
- Database storage (filesystem only for MVP)
- Report retention policies or cleanup (manual management)
- Multi-channel routing based on severity

## Decisions

### Report Template Structure

Use Go's `text/template` package to generate Markdown reports with the following sections:

**Required Sections:**
1. **Header**: Incident ID, timestamp, severity, cluster info
2. **Summary**: One-paragraph executive summary of the fault
3. **Findings**: Detailed analysis from the agent (bullet points)
4. **Recommendations**: Suggested next steps or remediation actions
5. **Metadata**: Agent version, duration, workspace path, links

**Rationale**: Structured format aligns with SRE incident report best practices (Google SRE postmortem format). Markdown is human-readable, version-control friendly, and easily convertible to other formats.

**Template Approach:**
- Define custom template functions for markdown formatting (heading, codeblock, escape)
- Use named sub-templates for reusable components (metadata table, severity badge)
- Trim whitespace with `{{-` and `-}}` to control Markdown rendering
- Escape user-provided content to prevent Markdown injection

### Artifact Directory Layout

```
/var/lib/event-runner/incidents/
├── <incident-id>/
│   ├── report.md                    # Final report
│   ├── agent-output.log             # Raw agent stdout/stderr
│   ├── agent-context.json           # Input context provided to agent
│   ├── metadata.json                # Runner metadata (timestamps, status, exit code)
│   └── artifacts/                   # Additional agent-generated files
│       ├── kubectl-output.txt
│       └── ...
```

**Rationale**:
- One directory per incident for easy isolation and cleanup
- Predictable file names for scripted access
- Separate raw logs from structured data (JSON) from final report (Markdown)
- `artifacts/` subdirectory allows agent to produce multiple files without cluttering root

**Directory Creation:**
- Runner creates workspace directory before agent invocation (already planned in Phase 2)
- Report generation writes to existing workspace after agent completes
- Base path configurable via `REPORT_ROOT_DIR` environment variable or config file

### Slack Payload Format

Use Slack's **Block Kit** for structured messages with **Message Attachments** for color coding.

**Payload Structure:**
```json
{
  "text": "Incident <incident-id> triage completed",
  "attachments": [
    {
      "color": "#E01E5A",  // Severity color (red=critical, yellow=warning, green=info)
      "blocks": [
        {
          "type": "header",
          "text": {
            "type": "plain_text",
            "text": "Incident <incident-id>: <summary>"
          }
        },
        {
          "type": "section",
          "fields": [
            {"type": "mrkdwn", "text": "*Severity:*\n<severity>"},
            {"type": "mrkdwn", "text": "*Cluster:*\n<cluster>"},
            {"type": "mrkdwn", "text": "*Resource:*\n<resource>"},
            {"type": "mrkdwn", "text": "*Duration:*\n<duration>"}
          ]
        },
        {
          "type": "section",
          "text": {
            "type": "mrkdwn",
            "text": "*Key Findings:*\n• <finding-1>\n• <finding-2>"
          }
        },
        {
          "type": "context",
          "elements": [
            {
              "type": "mrkdwn",
              "text": "Report: `<report-path>`"
            }
          ]
        }
      ]
    }
  ]
}
```

**Severity Color Mapping:**
- Critical/High: `#E01E5A` (Slack red)
- Warning/Medium: `#ECB22E` (Slack yellow)
- Info/Low: `#2EB67D` (Slack green)
- Failed Investigation: `#611F69` (Slack purple)

**Rationale**:
- Block Kit provides better structure and readability than plain text
- Message attachments allow color bars for quick visual severity identification
- Section fields create scannable two-column layout for metadata
- Context element for file path uses monospace formatting
- Fallback `text` field ensures basic notifications work even if blocks aren't supported

**Alternatives Considered:**
- Plain text messages: Too cluttered, harder to scan
- Rich text blocks: Overkill for simple notifications, added complexity
- Interactive components (buttons): Out of scope for MVP (no actions needed)

### Notification Trigger Flow

```
Agent Process Exits
    ↓
Runner Captures Exit Code
    ↓
Generate Report
    ↓
    ├─→ [Exit Code 0] → Success Report
    ├─→ [Exit Code >0] → Failure Report
    └─→ [Timeout]      → Timeout Report
    ↓
Write report.md and metadata.json
    ↓
Build Slack Payload
    ↓
POST to Webhook URL (with retry)
    ↓
    ├─→ [200-299] → Log success, continue
    ├─→ [4xx]     → Log error (config issue), continue
    └─→ [5xx/net] → Retry with backoff
```

**Key Points:**
- Always generate report first (even on agent failure) so filesystem has record
- Slack notification happens after disk persistence (disk is source of truth)
- Non-blocking: Slack failures don't prevent report generation
- Log all Slack responses for debugging

### Error Handling

**Disk Write Failures:**
- If report directory creation fails: Log error, skip report generation, still attempt Slack notification with error context
- If report write fails: Log error, attempt to write minimal metadata.json, send Slack notification with error
- Treat disk writes as critical but non-fatal (notification may still succeed)

**Slack Notification Failures:**
- HTTP 200-299: Success, log response
- HTTP 429 (Rate Limited): Retry with exponential backoff (3 attempts: 1s, 2s, 4s)
- HTTP 4xx (Client Error): Log error, don't retry (likely config issue: bad webhook URL or invalid payload)
- HTTP 5xx (Server Error): Retry with exponential backoff (3 attempts: 1s, 2s, 4s)
- Network timeout: Retry with backoff (3 attempts)
- After exhausting retries: Log final failure, continue (disk report is preserved)

**Rationale**: Slack is a best-effort notification. The source of truth is the filesystem report. Network or Slack API issues shouldn't block the runner or lose data.

**Agent Failure Handling:**
- If agent exits non-zero: Generate failure report with exit code and any captured stderr
- If agent times out: Generate timeout report with duration and last known state
- If agent crashes: Generate crash report with stack trace if available
- Always send Slack notification for failures (ops team needs to know)

### Configuration

Required configuration values (environment variables or config file):

```go
type ReportingConfig struct {
    // Slack webhook URL (required)
    SlackWebhookURL string `env:"SLACK_WEBHOOK_URL"`

    // Base directory for reports (default: /var/lib/event-runner/incidents)
    ReportRootDir string `env:"REPORT_ROOT_DIR" default:"/var/lib/event-runner/incidents"`

    // HTTP client timeout for Slack requests (default: 10s)
    SlackTimeout time.Duration `env:"SLACK_TIMEOUT" default:"10s"`

    // Max retry attempts for Slack (default: 3)
    SlackMaxRetries int `env:"SLACK_MAX_RETRIES" default:"3"`

    // Whether to include full agent output in Slack (default: false, too verbose)
    IncludeFullOutputInSlack bool `env:"SLACK_INCLUDE_FULL_OUTPUT" default:"false"`
}
```

**Validation:**
- Fail fast on startup if `SLACK_WEBHOOK_URL` is empty or malformed
- Validate `ReportRootDir` is writable on startup
- Log warnings for non-default values

### Go Package Structure

```
internal/reporting/
├── reporting.go        # Main Reporter interface and implementation
├── template.go         # Markdown template definitions and rendering
├── slack.go            # Slack webhook client with retry logic
├── models.go           # Data structures (ReportData, SlackPayload)
└── template_test.go    # Template rendering tests
```

**Reporter Interface:**
```go
type Reporter interface {
    // GenerateReport creates Markdown report and writes to disk
    GenerateReport(ctx context.Context, data ReportData) (reportPath string, err error)

    // SendNotification posts to Slack webhook
    SendNotification(ctx context.Context, data ReportData) error

    // Report is the convenience method that calls both
    Report(ctx context.Context, data ReportData) error
}
```

**ReportData Structure:**
```go
type ReportData struct {
    IncidentID    string
    Timestamp     time.Time
    Severity      string
    ClusterName   string
    Namespace     string
    ResourceType  string
    ResourceName  string

    // Agent outputs
    AgentExitCode int
    AgentDuration time.Duration
    Summary       string       // One-line summary
    Findings      []string     // Bullet points
    Recommendations []string   // Action items

    // Context
    WorkspacePath string
    AgentVersion  string
    RawOutput     string       // Full stdout/stderr
}
```

## Risks / Trade-offs

**Risk: Slack webhook URL exposure**
- Mitigation: Never log the full URL, document as secret in config guide
- Mitigation: Use environment variable, not config file (avoid checking into git)

**Risk: Disk space exhaustion**
- Mitigation: Document manual cleanup procedures in operations guide
- Mitigation: Consider log rotation for future iteration (not in MVP)

**Risk: Slack rate limiting**
- Mitigation: Retry logic with backoff
- Mitigation: Agent concurrency limits (already planned in Phase 2) naturally limit notification rate

**Risk: Report generation blocking agent workflow**
- Mitigation: Run report generation and Slack notification in goroutine after agent exits
- Mitigation: Use context with timeout to prevent indefinite hangs

**Trade-off: Filesystem vs Database**
- Decision: Filesystem for MVP simplicity
- Trade-off: Harder to query/aggregate, but easier to inspect and debug
- Future: Consider database for long-term storage and analytics

**Trade-off: Slack only vs multi-channel**
- Decision: Slack webhook only for MVP
- Trade-off: Less flexible, but simpler integration
- Future: Add pluggable notification backends (email, PagerDuty, etc.)

## Migration Plan

N/A - This is a new capability, no migration needed.

## Open Questions

1. Should we include raw kubectl outputs in the Slack message (as files) or just link to the report?
   - **Decision**: Link to report only (Slack messages should be scannable, not data dumps)

2. How do we handle multi-cluster notifications - single channel or multiple?
   - **Decision**: Single channel for MVP, use cluster name in message for filtering

3. Should the runner support templated Slack messages (admin-customizable)?
   - **Decision**: Not in MVP, hardcoded format is sufficient
   - Future: Consider template customization if requested

4. What happens if the workspace directory is deleted while the agent is running?
   - **Decision**: Runner should check workspace exists before writing report, recreate if missing

5. Should we send intermediate progress notifications to Slack (agent started, still running)?
   - **Decision**: No for MVP, only completion/failure notifications
   - Future: Consider optional progress updates for long-running investigations
