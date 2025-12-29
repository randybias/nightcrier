# Architecture

This document provides detailed technical information about Nightcrier's architecture, execution model, and failure handling.

For a high-level overview and philosophy, see the [README](../README.md).

## High-Level Flow

```
kubernetes-mcp-server -> Filtered MCP Events (faults) -> Nightcrier -> K8s Job (AI Agent) -> Investigation Report
                                                              |
                                                              v
                                                      Object Storage (Azure/S3)
                                                              |
                                                              v
                                                  Slack Notification (with Report URL)
```

## Agent Execution Model

Nightcrier uses **Kubernetes-native stateless agent execution**:

1. **ConfigMap Creation** - Incident data (incident.json, permissions.json, system prompt) is stored in a ConfigMap
2. **Job Creation** - A Kubernetes Job is created with the `nc-agent-runner` container image
3. **Presigned URLs** - PUT URLs are generated for artifact upload (report.md, agent.log, session.tar.gz)
4. **Agent Execution** - The Job runs an AI agent (Claude, Codex, Gemini, or Goose) to investigate
5. **Artifact Upload** - Results are uploaded directly to Object Storage from within the container
6. **Cleanup** - ConfigMap is deleted after Job completion

## Detailed Execution Flow

```
┌─────────────────┐
│  Fault Event    │
│  from MCP       │
└────────┬────────┘
         │
         v
┌─────────────────────────────────────────────────────────────┐
│  Nightcrier                                                  │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 1. Create Workspace & Generate Presigned URLs        │  │
│  │    - incident.json, permissions.json                 │  │
│  │    - PUT URLs for: report.md, agent.log,            │  │
│  │      session.tar.gz, result.json                    │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          v                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 2. Create K8s ConfigMap                              │  │
│  │    - incident.json (event context)                   │  │
│  │    - permissions.json (cluster access)               │  │
│  │    - base-triage-prompt.md (system prompt)          │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          v                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 3. Create K8s Job (nc-agent-runner)                  │  │
│  │    - Mounts ConfigMap and Secrets (API keys)        │  │
│  │    - Environment: presigned URLs, agent config      │  │
│  │    - Runs: entrypoint.sh -> AI agent -> uploads     │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          v                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 4. Watch Job Completion                              │  │
│  │    - Monitor via K8s Watch API                      │  │
│  │    - Timeout with activeDeadlineSeconds             │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          v                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 5. Retrieve Results from Object Storage             │  │
│  │    - Download report.md, result.json                │  │
│  │    - Convert markdown to HTML                       │  │
│  │    - Record in database (StateStore)                │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│              ┌───────────┴───────────┐                      │
│              │                       │                       │
│              v                       v                       │
│      ┌──────────────┐        ┌──────────────┐              │
│      │   SUCCESS    │        │   FAILURE    │              │
│      └──────┬───────┘        └──────┬───────┘              │
│             │                       │                       │
│             │                       v                       │
│             │              ┌─────────────────────┐         │
│             │              │ Circuit Breaker     │         │
│             │              │ Track Failures      │         │
│             │              │ (count: 1/3, 2/3..) │         │
│             │              └─────────┬───────────┘         │
│             │                        │                      │
│             │                        v                      │
│             │              ┌─────────────────────┐         │
│             │              │ Threshold Reached?  │         │
│             │              │ (default: 3 failures)│         │
│             │              └─────────┬───────────┘         │
│             │                        │                      │
│             │              ┌─────────┴─────────┐           │
│             │              │                   │           │
│             │              v                   v           │
│             │         ┌────────┐      ┌──────────────┐   │
│             │         │  YES   │      │     NO       │   │
│             │         └───┬────┘      └──────────────┘   │
│             │             │                                │
│             │             v                                │
│             │    ┌─────────────────────┐                  │
│             │    │ Send System         │                  │
│             │    │ Degraded Alert      │                  │
│             │    │ (if configured)     │                  │
│             │    └─────────────────────┘                  │
│             │                                               │
│             │  Skip individual notification                │
│             │                                               │
│             v                                               │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 6. Send Slack Notification                           │  │
│  │    - Incident details + root cause                   │  │
│  │    - Link to investigation report (signed URL)       │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          v                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 7. Cleanup ConfigMap                                 │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                               │
└───────────────────────────────────────────────────────────────┘

Circuit Breaker States:
┌──────────┐  threshold failures   ┌──────────┐
│  CLOSED  │ ─────────────────────> │   OPEN   │
│ (Normal) │                        │(Degraded)│
└──────────┘ <───────────────────── └──────────┘
              1 success + alert sent
```

## Circuit Breaker and Agent Failure Handling

The system includes intelligent agent failure handling to prevent spurious notifications and improve reliability.

### How It Works

When an AI agent executes, the system validates the output to ensure the agent successfully completed its investigation:

1. **Validation Checks**: Each agent execution is validated for:
   - Job completed successfully (not failed/timed out)
   - Output file exists in Object Storage (`report.md`)
   - Output file size is substantial (> 100 bytes)

2. **Circuit Breaker**: If validation fails, the system records the failure and tracks consecutive failures:
   - **Closed State** (Normal): Agent failures are tracked but no system-level alerts are sent
   - **Open State** (Degraded): After reaching the failure threshold, a system degraded alert is sent

3. **Automatic Recovery**: When an agent successfully completes an investigation after the circuit breaker opened:
   - Circuit breaker resets to closed state
   - A system recovered alert is sent
   - Failure counter resets to zero

### Configuration Options

Three environment variables control circuit breaker behavior:

```bash
# Enable/disable system degraded alerts (default: true)
export NOTIFY_ON_AGENT_FAILURE=true

# Number of consecutive failures before sending alert (default: 3)
export FAILURE_THRESHOLD_FOR_ALERT=3

# Upload failed investigations to storage (default: false)
export UPLOAD_FAILED_INVESTIGATIONS=false
```

In `config.yaml`:
```yaml
# Circuit breaker and failure notification configuration
notify_on_agent_failure: true
failure_threshold_for_alert: 3
upload_failed_investigations: false
```

### Notification Behavior

**Individual Incident Notifications (per-incident):**
- Sent for successful investigations only
- Skipped when agent validation fails
- Prevents spam from failed LLM API calls or agent issues

**System Degraded Alerts (aggregated):**
- Sent when `failure_threshold_for_alert` consecutive failures occur
- Only sent if `notify_on_agent_failure` is `true`
- Includes failure statistics and recent failure reasons
- Indicates the AI agent system may be experiencing issues

**System Recovered Alerts:**
- Sent when agent successfully completes after circuit opened
- Includes total downtime and failure count
- Indicates system returned to healthy state

### Storage Upload Behavior

By default (`upload_failed_investigations: false`):
- Only successful investigations are uploaded to storage
- Failed investigations remain in local workspace for debugging
- Reduces storage costs and prevents uploading incomplete data

When `upload_failed_investigations: true`:
- All investigations are uploaded, even if validation failed
- Useful for debugging agent issues
- Allows inspection of partial output

### Example Flow

```
Event -> Agent Execution -> Validation -> Outcome
---------------------------------------------------

Event 1 -> Agent runs -> Valid     -> Upload + Notify
Event 2 -> API error  -> Failed    -> Skip upload/notify (failure 1/3)
Event 3 -> Agent runs -> Empty     -> Skip upload/notify (failure 2/3)
Event 4 -> Timeout    -> Failed    -> Skip upload/notify (failure 3/3)
                                      SEND SYSTEM DEGRADED ALERT
Event 5 -> Agent runs -> Valid     -> Upload + Notify
                                      SEND SYSTEM RECOVERED ALERT
```
