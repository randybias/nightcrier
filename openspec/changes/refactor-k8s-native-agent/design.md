# Design: K8s-Native Stateless Agent Execution

## 1. Overview

This design document details the architectural decisions for refactoring Nightcrier's agent execution from Docker-based to K8s-native with stateless containers.

## 2. Architecture

### 2.1 Component Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         Go Orchestrator                                  │
│                                                                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                  │
│  │ Incident     │→ │ K8s Executor │→ │ Object Store │                  │
│  │ Handler      │  │ (client-go)  │  │ Client       │                  │
│  └──────────────┘  └──────────────┘  └──────────────┘                  │
│         │                 │                 │                           │
│         │                 │                 │                           │
│         ▼                 ▼                 ▼                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                  │
│  │ ConfigMap    │  │ Job          │  │ Presigned    │                  │
│  │ Generator    │  │ Generator    │  │ URL Gen      │                  │
│  └──────────────┘  └──────────────┘  └──────────────┘                  │
└─────────────────────────────────────────────────────────────────────────┘
         │                 │                 │
         │    K8s API      │                 │
         ▼                 ▼                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         Kubernetes Cluster                               │
│                                                                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                  │
│  │ ConfigMap    │  │ Secret       │  │ Job          │                  │
│  │ (incident)   │  │ (kubeconfig) │  │ (nc-agent)   │                  │
│  └──────────────┘  └──────────────┘  └──────────────┘                  │
│                           │                                             │
│                           ▼                                             │
│                    ┌──────────────┐                                     │
│                    │ Pod          │──────────────────┐                  │
│                    │ (stateless)  │                  │                  │
│                    └──────────────┘                  │                  │
└─────────────────────────────────────────────────────────────────────────┘
                                                       │
                                                       │ HTTP PUT
                                                       ▼
                                              ┌──────────────┐
                                              │ Object Store │
                                              │ (outputs)    │
                                              └──────────────┘
```

### 2.2 Data Flow

1. **Incident received** → Go orchestrator creates ConfigMap with incident data
2. **Job created** → References ConfigMap and existing Secrets (kubeconfig, API keys)
3. **Pod scheduled** → Kubelet mounts ConfigMap/Secrets as files
4. **Agent executes** → Reads mounted files, performs triage (with real-time logging)
5. **Post-run extraction** → Container extracts commands executed from session data
6. **Teardown** → Container uploads outputs to Object Store via presigned URLs
7. **Completion** → Go orchestrator watches Job, retrieves report from Object Store

## 3. Key Design Decisions

### Decision 1: K8s-Native Input Delivery (Not Object Store)

**Context**: Previous design proposed using Object Store with presigned GET URLs for all inputs.

**Decision**: Use K8s ConfigMaps and Secrets for input delivery.

**Rationale**:
- K8s already solves input delivery idiomatically
- No extra infrastructure required during container startup
- No "manifest URL as key to the kingdom" attack surface
- Simpler failure modes - if the Job starts, inputs are already mounted
- ConfigMaps support up to 1MB, sufficient for incident data

**Trade-offs**:
- ConfigMap 1MB limit (incident.json should be well under this)
- Secrets are base64 encoded (slight overhead)

### Decision 2: Object Store for Outputs Only

**Context**: Need durable storage for reports and session archives.

**Decision**: Container uploads outputs to Object Store before exit using presigned PUT URLs.

**Rationale**:
- Outputs need to persist beyond container lifecycle
- Presigned URLs avoid credential management in container
- Write-only URLs limit blast radius if compromised
- Existing Object Store integration can be reused

**Implementation**:
```bash
# Environment variables passed to container
OUTPUT_URL_REPORT="https://storage.../report.md?sig=..."
OUTPUT_URL_LOG="https://storage.../agent.log?sig=..."
OUTPUT_URL_SESSION="https://storage.../session.tar.gz?sig=..."
OUTPUT_URL_RESULT="https://storage.../result.json?sig=..."
OUTPUT_URL_COMMANDS="https://storage.../commands-executed.log?sig=..."

# Teardown uploads via curl
curl -X PUT -T /home/agent/output/report.md "$OUTPUT_URL_REPORT"
```

### Decision 3: Agent Skills Setup (Unified Mount + Symlinks)

**Context**: Each agent has different skill loading mechanisms and paths.

**Decision**: Mount skills at unified location, create agent-specific symlinks in entrypoint.

**Agent Skills Research**:

| Agent | Skills Location(s) | Mechanism | Notes |
|-------|-------------------|-----------|-------|
| Claude | `~/.claude/skills/` | Native SKILL.md loading | Single location |
| Codex | `~/.codex/skills/` | `--enable skills` flag | Single location |
| Gemini | N/A | Uses `GEMINI.md` context file | No skill directory |
| Goose | Multiple with priority | Searches in order: `~/.claude/skills/`, `~/.config/agents/skills/`, `~/.config/goose/skills/`, `./.goose/skills/`, `./.agents/skills/` | Latest (Dec 2024) skill support |

**Entrypoint Logic**:
```bash
setup_agent_paths() {
    case "$AGENT_CLI" in
        claude)
            mkdir -p ~/.claude
            ln -sf /home/agent/skills ~/.claude/skills
            SESSION_DIR=~/.claude
            ;;
        codex)
            mkdir -p ~/.codex
            ln -sf /home/agent/skills ~/.codex/skills
            SESSION_DIR=~/.codex
            ;;
        gemini)
            # Gemini uses GEMINI.md context file, already in /home/agent
            SESSION_DIR=~/.gemini
            ;;
        goose)
            # Goose searches multiple locations - use portable path
            mkdir -p ~/.config/agents
            ln -sf /home/agent/skills ~/.config/agents/skills
            # Also create goose-specific for priority
            mkdir -p ~/.config/goose
            ln -sf /home/agent/skills ~/.config/goose/skills
            export GOOSE_DISABLE_KEYRING=1
            SESSION_DIR=~/.config/goose
            ;;
    esac
    export SESSION_DIR
}
```

### Decision 4: In-Container Command Extraction

**Context**: Extracting "what commands were run" is critical for validation. Previously done via `docker cp` in post-run hooks.

**Decision**: Extract commands inside the container before upload, as part of teardown.

**Session Data Formats**:

| Agent | Format | Location | Extraction Method |
|-------|--------|----------|-------------------|
| Claude | JSONL | `~/.claude/projects/*/` | `jq` parsing |
| Codex | JSONL | `~/.codex/sessions/` | `jq` parsing |
| Gemini | JSON | `~/.gemini/tmp/*/chats/session-*.json` | `jq` parsing |
| Goose | SQLite | `~/.config/goose/sessions.db` | `sqlite3` queries |

**Extraction Logic** (in teardown):
```bash
extract_commands() {
    local commands_file="/tmp/commands-executed.log"

    case "$AGENT_CLI" in
        claude)
            # Find most recent JSONL
            local jsonl=$(find ~/.claude/projects -name "*.jsonl" -type f 2>/dev/null | xargs ls -t | head -1)
            if [[ -f "$jsonl" ]]; then
                jq -r 'select(.type == "assistant") |
                    .message.content[]? |
                    select(.type == "tool_use" and .name == "Bash") |
                    "$ " + .input.command' "$jsonl" > "$commands_file"
            fi
            ;;
        codex)
            local jsonl=$(find ~/.codex/sessions -name "*.jsonl" -type f 2>/dev/null | xargs ls -t | head -1)
            if [[ -f "$jsonl" ]]; then
                jq -r 'select(.type == "response_item" and .payload.type == "function_call" and .payload.name == "shell_command") |
                    .payload.arguments | fromjson | "$ " + .command' "$jsonl" > "$commands_file"
            fi
            ;;
        gemini)
            local json=$(find ~/.gemini/tmp -path "*/chats/session-*.json" -type f 2>/dev/null | xargs ls -t | head -1)
            if [[ -f "$json" ]]; then
                jq -r '.[] | select(.type == "tool_use" and .tool_name == "bash") |
                    "$ " + .tool_input.command' "$json" > "$commands_file"
            fi
            ;;
        goose)
            if [[ -f ~/.config/goose/sessions.db ]]; then
                # Extract shell commands from most recent session
                sqlite3 ~/.config/goose/sessions.db \
                    "SELECT '$ ' || json_extract(content, '$.command')
                     FROM messages
                     WHERE role = 'tool' AND json_extract(content, '$.command') IS NOT NULL
                     ORDER BY created_at" > "$commands_file" 2>/dev/null || true
            fi
            ;;
    esac

    # Upload if we have commands
    if [[ -s "$commands_file" ]]; then
        curl -X PUT -T "$commands_file" "$OUTPUT_URL_COMMANDS" || true
    fi
}
```

### Decision 5: Container Naming

**Context**: Container image name should be descriptive and distinct.

**Decision**: Use `nc-agent-runner` (or `nightcrier-agent-runner` for full name).

**Rationale**:
- More descriptive than generic `nightcrier-agent`
- Indicates the purpose (running agents)
- Short form `nc-` prefix for convenience
- Consistent naming for disk paths and container registry

### Decision 6: No Parallel Docker/K8s Operation

**Context**: Previous design proposed running Docker and K8s executors in parallel during migration.

**Decision**: Switch directly to K8s. Docker scripts kept for reference only.

**Rationale**:
- Reduces complexity - single code path
- Git provides rollback if needed
- No backwards compatibility requirement
- Faster migration with less maintenance burden

### Decision 7: Basic Real-Time Logging

**Context**: Need visibility into agent execution progress.

**Decision**: Implement basic real-time logging with room for future expansion.

**Implementation**:
- Agent stdout/stderr displayed to console AND captured to file (via `tee`)
- K8s `kubectl logs -f` provides real-time streaming
- Go orchestrator can optionally tail logs via K8s API
- Minimalist initial implementation, note for future expansion

```bash
# In entrypoint - capture and display
run_agent() {
    local log_file="/home/agent/logs/agent.log"
    mkdir -p /home/agent/logs
    local triage_prompt=$(build_triage_prompt)

    # Run agent with tee for real-time output + file capture
    case "$AGENT_CLI" in
        claude)
            # Claude: -p flag (print mode), prompt is last positional
            claude -p --model "$LLM_MODEL" \
                --allowedTools "Read,Grep,Glob,Bash,Write" --max-turns 50 \
                "$triage_prompt" 2>&1 | tee "$log_file"
            ;;
        codex)
            # Codex: no json output option, prompt is last positional
            codex exec --skip-git-repo-check --enable skills \
                --dangerously-bypass-approvals-and-sandbox \
                -m "$LLM_MODEL" "$triage_prompt" 2>&1 | tee "$log_file"
            ;;
        gemini)
            # Gemini: --yolo for auto-approval, positional prompt
            gemini --model "$LLM_MODEL" --yolo \
                "$triage_prompt" 2>&1 | tee "$log_file"
            ;;
        goose)
            # Goose: use 'goose run' with --text for headless mode
            # Requires LLM_PROVIDER env var (e.g., openai, anthropic)
            goose run --model "$LLM_MODEL" --provider "$LLM_PROVIDER" \
                --text "$triage_prompt" 2>&1 | tee "$log_file"
            ;;
    esac
}
```

### Decision 8: Container Image Version Updates

**Context**: Agent CLIs and kubectl need regular updates.

**Decision**: Convenience scripts for manual updates, with optional GitHub Action.

**Components to update**:
- Claude Code CLI
- OpenAI Codex CLI
- Google Gemini CLI
- Goose CLI
- kubectl

**Implementation**:
```bash
# scripts/update-agent-versions.sh
#!/usr/bin/env bash
# Convenience script to update agent CLI versions in Dockerfile

# Check latest versions
echo "Checking latest versions..."
# (version check commands)

# Update Dockerfile
# (sed/awk to update version numbers)

echo "Updated Dockerfile. Rebuild with: make build-agent-image"
```

**Future**: GitHub Action running weekly to check for updates and create PRs.

## 4. Job Specification

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: triage-{{ .IncidentID }}
  namespace: nightcrier
  labels:
    app: nc-agent-runner
    incident-id: {{ .IncidentID }}
    cluster: {{ .ClusterName }}
spec:
  ttlSecondsAfterFinished: 3600  # Cleanup after 1 hour
  activeDeadlineSeconds: 600     # Job timeout (sends SIGTERM)
  backoffLimit: 0                # No retries - triage is point-in-time
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: agent
        image: nc-agent-runner:{{ .Version }}
        env:
        - name: AGENT_CLI
          value: "{{ .AgentCLI }}"
        - name: LLM_MODEL
          value: "{{ .Model }}"
        - name: INCIDENT_ID
          value: "{{ .IncidentID }}"
        - name: PROMPT
          value: "{{ .Prompt }}"
        - name: ANTHROPIC_API_KEY
          valueFrom:
            secretKeyRef:
              name: ai-api-keys
              key: anthropic
              optional: true
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: ai-api-keys
              key: openai
              optional: true
        - name: GEMINI_API_KEY
          valueFrom:
            secretKeyRef:
              name: ai-api-keys
              key: gemini
              optional: true
        # Presigned PUT URLs for outputs
        - name: OUTPUT_URL_REPORT
          value: "{{ .PresignedURLReport }}"
        - name: OUTPUT_URL_LOG
          value: "{{ .PresignedURLLog }}"
        - name: OUTPUT_URL_SESSION
          value: "{{ .PresignedURLSession }}"
        - name: OUTPUT_URL_RESULT
          value: "{{ .PresignedURLResult }}"
        - name: OUTPUT_URL_COMMANDS
          value: "{{ .PresignedURLCommands }}"
        volumeMounts:
        - name: incident-data
          mountPath: /home/agent/incident.json
          subPath: incident.json
          readOnly: true
        - name: incident-data
          mountPath: /home/agent/incident_cluster_permissions.json
          subPath: permissions.json
          readOnly: true
        - name: incident-data
          mountPath: /home/agent/base-triage-prompt.md
          subPath: base-triage-prompt.md
          readOnly: true
        - name: kubeconfig
          mountPath: /home/agent/.kube/config
          subPath: config
          readOnly: true
        resources:
          limits:
            memory: "2Gi"
            cpu: "1"
          requests:
            memory: "512Mi"
            cpu: "250m"
      volumes:
      - name: incident-data
        configMap:
          name: incident-{{ .IncidentID }}
      - name: kubeconfig
        secret:
          secretName: triage-kubeconfig-{{ .ClusterName }}
```

## 5. Security Model

### Input Security

- **Kubeconfig**: Stored in K8s Secret, uses read-only ServiceAccount with TTL
- **Incident data**: ConfigMap, namespace-scoped RBAC
- **API keys**: K8s Secret, injected as env vars (not written to disk)

### Output Security

- **Presigned URLs**: Short expiration matching job timeout
- **Write-only**: PUT URLs cannot read other data
- **Scoped paths**: Each incident gets isolated storage path

### Container Security

- **No privileged access**: Standard container
- **Read-only inputs**: All ConfigMap/Secret mounts are read-only
- **Resource limits**: Prevent runaway consumption
- **Network policy**: Can restrict egress to Object Store + target cluster

## 6. Migration Path

**Single-phase migration** (no parallel Docker/K8s):

1. **Implement K8s executor** (`internal/agent/k8s_executor.go`)
2. **Create container entrypoint** (`nc-agent-runner/entrypoint.sh`)
3. **Update Dockerfile** to use new entrypoint and image name
4. **Update Go orchestrator** to use K8s executor
5. **Move Docker scripts to reference directory** (`agent-container/reference/`)
6. **Update documentation** for `kind`-based local development
7. **Delete Docker execution code** from active paths

Git history provides rollback if needed.

## 7. Files Changed

### Moved to Reference
- `agent-container/run-agent.sh` → `agent-container/reference/`
- `agent-container/runners/*.sh` → `agent-container/reference/`
- `agent-container/test_*.sh` → `agent-container/reference/`

### Added
- `internal/agent/k8s_executor.go` - K8s Job creation via client-go
- `nc-agent-runner/entrypoint.sh` - Unified container entrypoint
- `nc-agent-runner/Dockerfile` - Container image definition
- `scripts/update-agent-versions.sh` - CLI version update script
- `scripts/build-agent-image.sh` - Image build convenience script
- `deploy/namespace.yaml` - Nightcrier namespace
- `deploy/rbac.yaml` - RBAC for Job creation
- `deploy/dev/` - Local development setup for `kind`

### Modified
- `internal/agent/executor.go` - Remove Docker, delegate to K8s executor
- `internal/config/config.go` - K8s-only configuration

## 8. Container Image Requirements

The `nc-agent-runner` image must include:

**Agent CLIs**:
- Claude Code CLI
- OpenAI Codex CLI
- Google Gemini CLI
- Goose CLI

**K8s Tools**:
- kubectl 1.31+
- helm 3.x

**Utilities**:
- curl (for presigned URL uploads)
- jq (for JSON/JSONL parsing)
- sqlite3 (for Goose session extraction)
- tar (for session archiving)
- ripgrep, fd, fzf (for agent use)

**Skills**:
- k8s-troubleshooter skill baked in at `/home/agent/skills/`

## 9. Open Questions (Deferred)

1. **Multiple skills support**: Current design bakes in only `k8s-troubleshooter`. Future work needed for:
   - Loading multiple skills (baked in vs ConfigMap-mounted)
   - Per-incident skill selection (which skills are relevant?)
   - Skill versioning and updates separate from container image
   - Dynamic skill loading at runtime vs build-time

2. **ConfigMap size limits**: K8s ConfigMaps are limited to 1MB. Verify `incident.json` stays well under this.

3. **Log streaming infrastructure**: Current design is minimalist (`kubectl logs -f`). May need dedicated log aggregation for production observability.
