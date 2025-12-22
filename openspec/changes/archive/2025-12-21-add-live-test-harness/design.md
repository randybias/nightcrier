# Design: Live Test Harness

## Architecture Overview

The test harness uses a layered architecture:

```
┌─────────────────────────────────────────┐
│      run-live-test.sh (orchestrator)    │
├─────────────────────────────────────────┤
│  Config Gen │ Failure Inject │ Monitor  │
├─────────────────────────────────────────┤
│         nightcrier (background)          │
├─────────────────────────────────────────┤
│      Kubernetes Cluster (live)           │
└─────────────────────────────────────────┘
```

## Component Design

### 1. Secret Management

**Problem**: Secrets must not be committed but need to be available for testing.

**Solution**: Single source of truth in `~/dev-secrets/`:

```bash
# ~/dev-secrets/nightcrier-secrets.env
ANTHROPIC_API_KEY="sk-ant-..."
OPENAI_API_KEY="sk-proj-..."
GEMINI_API_KEY="AIza..."
SLACK_WEBHOOK_URL="https://hooks.slack.com/..."
AZURE_STORAGE_ACCOUNT="..."
AZURE_STORAGE_KEY="..."
AZURE_STORAGE_CONTAINER="..."
MCP_ENDPOINT="http://..."
MCP_API_KEY="..."
```

**Rationale**:
- Single file is easier to manage than multiple files
- Environment variable format is standard and easy to source
- Can be backed up/restored independently

### 2. Configuration Templates

**Problem**: Config files contain secrets and cluster-specific details.

**Solution**: Templates with placeholder variables:

```yaml
# tests/config-templates/test-claude.yaml.tmpl
clusters:
  - name: eastus-cluster1
    mcp:
      endpoint: "${MCP_ENDPOINT}"
      api_key: "${MCP_API_KEY}"
    triage:
      kubeconfig: "${KUBECONFIG_PATH}"

agent_cli: "claude"
agent_model: "sonnet"
agent_debug: ${DEBUG_MODE}

slack_webhook_url: "${SLACK_WEBHOOK_URL}"
azure_storage_account: "${AZURE_STORAGE_ACCOUNT}"
azure_storage_key: "${AZURE_STORAGE_KEY}"
```

**Interpolation**:
```bash
envsubst < test-claude.yaml.tmpl > ../configs/config-test-claude.yaml
```

**Rationale**:
- Templates can be version controlled safely
- `envsubst` is standard Unix tool (no custom parsing)
- Generated configs match production format exactly

### 3. Failure Induction

**Problem**: Need reproducible, real Kubernetes failures.

**Solution**: Pluggable scripts with start/stop lifecycle:

```bash
# tests/failure-induction/01_induce_failure_crashloopbackoff.sh
#!/usr/bin/env bash

case "$1" in
  start)
    kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: crashloop-test
  namespace: default
spec:
  containers:
  - name: crasher
    image: busybox
    command: ["sh", "-c", "exit 1"]
  restartPolicy: Always
EOF
    ;;
  stop)
    kubectl delete pod crashloop-test -n default --ignore-not-found
    ;;
esac
```

**Contract**:
- Script accepts `start` or `stop` argument
- `start` creates the failure condition and outputs timeout value (seconds) to stdout
- `stop` cleans up completely
- Exit code 0 on success
- Uses kubeconfig from environment

**Example output**:
```bash
$ ./01_induce_failure_crashloopbackoff.sh start
TIMEOUT=300
Pod crashloop-test created
```

**Rationale**:
- Simple contract makes it easy to add new tests
- Real Kubernetes objects = real MCP events
- Cleanup ensures test isolation

### 4. Test Orchestration

**Main script flow**:

```bash
#!/usr/bin/env bash
# run-live-test.sh <agent> <test-type> [--debug] [--json]

1. Generate unique test ID (e.g., test-20251221-180000-abc123)
2. Create log directory: tests/logs/<testid>/
3. Validate arguments (agent: claude|codex|gemini, test: crashloopbackoff)
4. Load secrets from ~/dev-secrets/nightcrier-secrets.env
5. Generate config from template
6. Start nightcrier in background, redirect to logs/<testid>/nightcrier.log
7. Tail log waiting for "subscribed to fault events"
8. Run failure induction script (start), capture TIMEOUT value
9. Tail log waiting for agent execution signs (with captured timeout)
10. Wait for agent completion (look for "Agent completed")
11. Run failure induction script (stop)
12. Stop nightcrier (SIGTERM)
13. Generate report from logs and artifacts
14. Display report to stdout (human-readable or JSON based on --json flag)
15. Write JSON report to logs/<testid>/report.json
```

**State management**:
- Generate and track unique test ID
- Track nightcrier PID for cleanup
- Track failure induction state
- Use timeout value from induction script

**Error handling**:
- Trap EXIT to ensure cleanup
- Kill nightcrier on script exit
- Stop failure induction on script exit

### 5. Log Monitoring

**Problem**: Need to detect test progress from nightcrier logs.

**Solution**: Pattern-based state detection:

```bash
# Wait for MCP subscription
wait_for_subscription() {
  timeout 60 grep -q "subscribed to fault events" <(tail -f "$LOG_FILE")
}

# Wait for agent start
wait_for_agent_start() {
  timeout 180 grep -q "starting.*agent" <(tail -f "$LOG_FILE")
}

# Wait for agent completion
wait_for_agent_completion() {
  timeout 300 grep -q "Agent completed" <(tail -f "$LOG_FILE")
}
```

**Rationale**:
- Non-invasive (doesn't modify nightcrier)
- Works with existing log format
- Timeouts prevent hanging tests

### 6. Report Generation

**Report structure**:

```
========================================
Test Report: claude / crashloopbackoff
========================================
Start Time:     2025-12-21 18:00:00
End Time:       2025-12-21 18:02:30
Duration:       2m30s
Status:         PASSED

Timeline:
  18:00:00  Started nightcrier
  18:00:05  MCP subscription confirmed
  18:00:10  Induced crashloopbackoff failure
  18:00:45  Agent execution started
  18:02:20  Agent completed (exit 0)
  18:02:25  Failure cleaned up
  18:02:30  Nightcrier stopped

Artifacts:
  Incident ID:          abc-123
  Investigation Report: ./incidents/abc-123/output/investigation.md (4.2KB)
  Commands Executed:    ./incidents/abc-123/logs/agent-commands-executed.log (14 commands)
  Full Logs:            ./tests/logs/nightcrier-20251221-180000.log

Validation:
  ✓ MCP subscription successful
  ✓ Failure detected by cluster
  ✓ Agent triggered
  ✓ Agent completed successfully
  ✓ Investigation report generated
  ✓ Commands extracted to audit log
  ✓ Artifacts uploaded to Azure
  ✓ Slack notification sent

========================================
```

**Output modes**:
- **Default (human-readable)**: Display formatted report to stdout
- **JSON mode (`--json` flag)**: Display JSON report to stdout
- **Always**: Write JSON to `logs/<testid>/report.json` regardless of output mode

**Note for AI agents**: The README will specify that AI agents should use `--json` flag for easier parsing.

## Trade-offs

### Template Approach vs Dynamic Config
**Chosen**: Templates with envsubst
**Alternative**: Generate configs programmatically in Go/Python
**Rationale**: Templates are simple, visual, and don't require code changes

### Background Process vs Foreground
**Chosen**: Run nightcrier in background
**Alternative**: Run in foreground with output parsing
**Rationale**: Background + log file allows flexible monitoring without blocking

### Single Test vs Test Suite
**Chosen**: Run one test at a time
**Alternative**: Support running multiple tests sequentially
**Rationale**: Start simple, add batch mode later if needed

### Admin Kubeconfig vs Limited Access
**Chosen**: Admin privileges for test cluster
**Alternative**: Minimal permissions for failure induction
**Rationale**: Need admin to create/delete resources; not production cluster

## Security Considerations

1. **Secrets file**: `~/dev-secrets/` with restrictive permissions (600)
2. **No secrets in repo**: Templates have placeholders only
3. **No secrets in logs**: Nightcrier already sanitizes logs
4. **Test cluster only**: Never run against production
5. **Cleanup always runs**: Trap ensures resources are deleted

## Future Enhancements

1. **More test types**: imagepullbackoff, oom, node-not-ready
2. **Multi-agent comparison**: Run all agents on same failure, compare outputs
3. **CI integration**: Run tests in GitHub Actions
4. **Performance metrics**: Track agent execution time, token usage
5. **Test matrix**: All agents × all failure types
