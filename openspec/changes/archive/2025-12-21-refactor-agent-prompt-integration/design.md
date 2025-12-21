# Design: Refactor Agent Prompt Integration

## Context

The k8s-troubleshooter skill is designed as an **automation-first** diagnostic framework with structured investigation workflows:

- `incident_triage.sh` - Comprehensive incident assessment (start here for any incident)
- `pod_diagnostics.sh` - Pod-specific troubleshooting
- `helm_release_debug.sh` - Helm issue diagnostics
- `cluster_assessment.sh` - Cluster-wide health assessment

The skill encodes expert Kubernetes troubleshooting workflows with **progressive disclosure**: core workflows in SKILL.md, deep dives in references/ when needed.

**Key insight**: The skill itself tells agents what to do. Nightcrier's role is to invoke the agent with the right context, not micromanage the investigation methodology.

## Current Architecture (Problems)

```
Config (agent_prompt)          System Prompt File
         │                            │
         │ "-p" flag                   │ "--append-system-prompt-file"
         ▼                            ▼
┌──────────────────────────────────────────────────────┐
│                    Claude CLI                         │
│  Receives BOTH prompts with conflicting instructions │
└──────────────────────────────────────────────────────┘
```

**Problems**:
1. `agent_prompt` is **mandatory** - forces every config to specify investigation steps
2. System prompt specifies step-by-step investigation (1. Read X, 2. Read Y, 3. Use skill, 4. Write output)
3. Two sources of truth: config prompt says "do triage", system prompt says "use skill"
4. No audit trail: we don't capture what prompt was actually sent

## Implemented Architecture

```
Config (additional_agent_prompt)     System Prompt File (minimal)
         │ OPTIONAL                          │ skill-aware
         │                                   │
         ▼                                   ▼
┌───────────────────────────────────────────────────────┐
│                   Executor (Go)                        │
│                                                        │
│  1. Read system prompt file content                   │
│  2. Combine: system_prompt + additional (if provided) │
│  3. Write combined prompt to prompt-sent.md           │
│  4. Pass combined prompt as positional arg to script  │
└───────────────────────────────────────────────────────┘
         │
         ├─────────────────────────────────────┐
         ▼                                     ▼
┌─────────────────────────┐    ┌─────────────────────────────────┐
│   prompt-sent.md        │    │     run-agent.sh                │
│                         │    │                                 │
│  Metadata + Full prompt │    │  Receives: combined prompt      │
│  (audit artifact)       │    │  Passes to: claude -p "..."     │
└─────────────────────────┘    └─────────────────────────────────┘
         │                                     │
         │                                     ▼ (DEBUG mode)
         │                     ┌─────────────────────────────────┐
         │                     │   Post-run hooks                │
         │                     │                                 │
         │                     │  1. Extract ~/.claude session   │
         │                     │  2. Parse JSONL for Bash cmds   │
         │                     │  3. Write commands-executed.log │
         │                     │  4. Archive session tar.gz      │
         │                     └─────────────────────────────────┘
         │                                     │
         ▼                                     ▼
┌───────────────────────────────────────────────────────┐
│               Azure Storage + Local Filesystem        │
│                                                       │
│  Artifacts: investigation.md, incident.json,          │
│             prompt-sent.md, agent-commands-executed.log,│
│             claude-session.tar.gz (DEBUG only)        │
└───────────────────────────────────────────────────────┘
```

**Key Implementation Detail**: The system prompt content is read and combined with the additional prompt in executor.go, then passed as a single positional argument to run-agent.sh. This is different from the original design which used separate `-p` and `--system-prompt-file` flags. The combined approach:
- Ensures run-agent.sh always receives a valid prompt
- Allows the additional prompt to be truly optional
- Creates a single source of truth for what the agent received

## Key Design Decisions

### 1. Optional vs Mandatory Additional Prompt

**Decision**: `additional_agent_prompt` is **optional** (empty string by default)

**Rationale**:
- System prompt alone is sufficient for skill-driven workflows
- Additional prompt is for cluster-specific context (SLOs, escalation contacts, special constraints)
- Removes boilerplate from config files
- Lets the skill drive the investigation methodology

### 2. System Prompt Content

**Decision**: Minimal, skill-enabling system prompt (~20 lines)

**New structure**:
```markdown
# Kubernetes Fault Triage Agent

You are investigating a production Kubernetes incident.

## Workspace
- incident.json: Fault event context from monitoring system
- incident_cluster_permissions.json: Your cluster access permissions

## Approach
Use the k8s-troubleshooter skill for systematic investigation:
- Start with: incident_triage.sh --skip-dump
- Follow the skill's recommendations for next steps
- The skill provides structured workflows for all common issues

## Constraints
- READ-ONLY ONLY: No kubectl apply/delete/patch/edit
- KUBECONFIG already set with validated permissions

## Output
Write findings to output/investigation.md with:
- Executive summary (root cause + confidence level)
- Evidence and analysis
- Prioritized recommendations
```

**Rationale**:
- Tells agent WHAT files are available, not HOW to use them
- Defers to skill for investigation methodology
- Keeps output location specification (needed for artifact collection)
- Read-only constraint is safety-critical, must be explicit

### 3. Prompt Capture Format

**Decision**: Markdown file with metadata header

**Format**:
```markdown
# Prompt Sent to Agent

## Metadata
- Timestamp: 2025-12-21T14:30:00Z
- Incident ID: abc123-def456
- Cluster: westeu-cluster1
- Agent CLI: claude
- Model: haiku

## System Prompt
(contents of triage-system-prompt.md)

## Additional Prompt
(contents of additional_agent_prompt, or "None provided")
```

**Rationale**:
- Human-readable for debugging
- Machine-parseable for analysis
- Includes context needed for forensics
- Markdown renders nicely in Azure Blob Storage browser

### 4. Capture Timing

**Decision**: Capture prompt in executor.go before subprocess launch

**Rationale**:
- Single location for capture logic
- Has access to all prompt sources (config, file)
- Can include metadata (incident ID, cluster, model)
- Before subprocess means we capture even if agent crashes

### 5. Storage Pattern

**Decision**: Follow existing artifact pattern

**Implementation**:
- Add `PromptSent []byte` to `IncidentArtifacts` struct
- Upload alongside incident.json, investigation.md, logs
- Include in index.html for Azure browser
- Store in local filesystem same as other artifacts

**Rationale**:
- Consistent with existing architecture
- No new storage mechanisms needed
- Existing upload/download paths work

## Migration Path

### Config Migration

Old:
```yaml
agent_prompt: "Production incident detected. Incident context..."
```

New:
```yaml
# agent_prompt removed - system prompt drives investigation
# additional_agent_prompt is optional for cluster-specific context
additional_agent_prompt: ""  # or omit entirely
```

### Breaking Change Handling

1. Remove `agent_prompt` from validation requirements in config.go
2. Add optional `additional_agent_prompt` field
3. Update all example configs to remove hardcoded prompts
4. Document migration in CHANGELOG

## File Changes Summary

| File | Change |
|------|--------|
| internal/config/config.go | Remove `agent_prompt` validation, add optional `additional_agent_prompt` |
| configs/triage-system-prompt.md | Rewrite to be skill-aware and minimal |
| configs/config.example.yaml | Remove agent_prompt, add commented additional_agent_prompt |
| configs/config-test.yaml | Keep agent_prompt for backwards compat testing |
| configs/config-multicluster.yaml | Keep agent_prompt for backwards compat testing |
| configs/config-codex.yaml | Keep agent_prompt for backwards compat testing |
| internal/agent/executor.go | Combine system+additional prompt, write prompt-sent.md, pass as positional arg |
| internal/storage/storage.go | Add PromptSent to IncidentArtifacts, add CommandsExecuted to AgentLogs |
| internal/storage/azure.go | Upload prompt-sent.md and agent-commands-executed.log |
| internal/storage/filesystem.go | Store prompt-sent.md and agent-commands-executed.log |
| cmd/nightcrier/main.go | Read prompt-sent.md and agent-commands-executed.log |
| agent-container/run-agent.sh | Add extract_agent_commands() post-run hook |
| openspec/specs/agent-logging/spec.md | Add Agent Commands Extraction requirement |
| openspec/specs/cloud-storage/spec.md | Add Debug Log Artifacts requirement |

## Risks and Mitigations

### Risk: Agent doesn't invoke skill
**Mitigation**: System prompt explicitly tells agent to use k8s-troubleshooter. Skill is pre-installed in container. Agent has tool access via allowed tools.

### Risk: Breaking existing deployments
**Mitigation**: Clear migration documentation. Config validation gives helpful error if old agent_prompt field is present.

### Risk: Prompt file missing from workspace
**Mitigation**: Write prompt-sent.md before launching subprocess. If executor crashes, we still have the file. Include in artifacts even if empty.

## Additional Feature: Agent Commands Extraction

### Purpose

During debugging and forensic analysis, operators need to understand exactly what commands the agent executed. The Claude session JSONL files contain this information but are not easily readable.

### Implementation

In DEBUG mode, after the agent completes:

1. `post_run_extract_claude_session()` extracts `~/.claude` from the container
2. `extract_agent_commands()` parses the session JSONL files using jq
3. All Bash tool calls are extracted with their commands and descriptions
4. Output is written to `logs/agent-commands-executed.log`

### Format

```
# Agent Commands Executed
# Generated: 2025-12-21T08:50:55Z
# Incident: ad50a15b-bf1c-491e-9fc1-b1ccd00a0b9e
# Session: 301d3531-af44-470f-a9c7-6654ce48a14a.jsonl
#

$ ~/.claude/skills/k8s-troubleshooter/scripts/incident_triage.sh --skip-dump # Run incident triage workflow
$ kubectl get pod crashloop-xyz -n mcp-test -o wide # Check pod status
```

### Benefits

- **Audit trail**: Know exactly what the agent did
- **Debugging**: Reproduce agent behavior manually
- **Compliance**: Evidence of agent actions for security review
- **Learning**: Understand how the agent approaches different incidents
