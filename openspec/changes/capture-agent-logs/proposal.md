# Proposal: Capture Agent Container Logs

## Summary

Capture complete logs from agent container runs and persist them both locally and to Azure Blob Storage for debugging, auditing, and observability.

## Motivation

Currently, agent execution has limited observability:

1. **Executor captures only snippets**: The Go executor reads stdout/stderr in 1KB chunks and logs via slog, but doesn't persist full output
2. **run-agent.sh tees to a file**: The script already saves output to `${WORKSPACE}/output/triage_*.log`, but this isn't uploaded to Azure
3. **Azure storage misses logs**: Only `event.json`, `result.json`, and `investigation.md` are uploaded

When troubleshooting agent failures:
- Full conversation logs are critical for debugging AI agent behavior
- API errors, tool execution details, and reasoning traces are lost
- No way to audit what the agent actually did during investigation

## Proposed Solution

### 1. Capture Full Logs in Executor

Modify the Go executor to:
- Write complete stdout/stderr to log files in the workspace
- Create `{workspace}/logs/agent-stdout.log` and `agent-stderr.log`
- Optionally create combined `agent-full.log`

### 2. Upload Logs to Azure Blob Storage

Extend the storage interface to:
- Upload log files to a `logs/{incident-id}/` prefix in the blob container
- Include log URLs in the `SaveResult` response
- Add log URLs to Slack notifications

### 3. Storage Structure

```
Container: incident-reports
├── {incident-id}/
│   ├── event.json
│   ├── result.json
│   └── investigation.md
└── logs/
    └── {incident-id}/
        ├── agent-stdout.log
        ├── agent-stderr.log
        └── agent-full.log      (combined, timestamped)
```

Alternative (simpler - keeps logs with incident):
```
Container: incident-reports
└── {incident-id}/
    ├── event.json
    ├── result.json
    ├── investigation.md
    └── logs/
        ├── agent-stdout.log
        ├── agent-stderr.log
        └── agent-full.log
```

## Design Decisions

### Log File Location

**Decision**: Store logs as a subdirectory within the incident (`{incident-id}/logs/`)

Rationale:
- Keeps all incident artifacts together
- Simpler cleanup (delete incident directory removes everything)
- Easier to correlate logs with incidents
- No separate permission/lifecycle management needed

### Log Content

The agent logs will include:
- Full AI conversation (prompts, responses)
- Tool execution output (kubectl commands, file reads)
- Error messages and stack traces
- Timing information

### Log Size Management

- No immediate size limits (agents are time-bounded)
- Future: Consider compression for large logs
- Future: Retention policy for old logs

## Impact

| Component | Changes |
|-----------|---------|
| `internal/agent/executor.go` | Write logs to workspace files |
| `internal/storage/storage.go` | Add `AgentLogs` field to `IncidentArtifacts` |
| `internal/storage/azure.go` | Upload log files with incident |
| `internal/storage/filesystem.go` | Copy log files to storage location |
| `cmd/runner/main.go` | Pass log files to storage |
| `internal/reporting/slack.go` | Include log URL in notification |

## Alternatives Considered

1. **Stream logs to external service (CloudWatch, Loki)**: Adds infrastructure complexity
2. **Separate logs container**: Harder to correlate with incidents
3. **Only keep logs locally**: Loses logs when pods restart or storage is ephemeral

## Open Questions

1. **Log retention**: How long should logs be kept? (Default to same as incident artifacts)
2. **Log compression**: Compress before upload? (Start without, add if needed)
3. **Combined vs separate files**: Keep stdout/stderr separate or combine? (Recommend combined with timestamps)
