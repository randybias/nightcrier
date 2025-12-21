# Proposal: Refactor Agent Prompt Integration

## Executive Summary

Currently, Nightcrier requires a hardcoded `agent_prompt` field in configuration that over-specifies investigation steps, conflicting with how the k8s-troubleshooter skill is designed to work. This proposal:

1. **Removes mandatory agent_prompt** - Makes it optional via `additional_agent_prompt`
2. **Rewrites system prompt to be skill-aware** - Minimal instructions that let the skill drive investigation
3. **Captures full prompt before execution** - For auditability, debugging, and compliance
4. **Stores prompt in incident artifacts** - Enables forensic analysis and integration with Azure storage

## Problem Statement

### Issue 1: Over-Specified Prompt Duplication

The k8s-troubleshooter skill is **automation-first**, with its own investigation workflows:
- `incident_triage.sh` - Comprehensive incident assessment
- `pod_diagnostics.sh` - Pod-specific troubleshooting
- `helm_release_debug.sh` - Helm issue diagnostics
- `cluster_assessment.sh` - Cluster-wide health

Current system prompt tells the agent to use the skill, but the config prompt tells the agent detailed investigation steps. Two sources of truth for methodology.

**Current config prompt** (config-test.yaml:92):
```
"Production incident detected. Incident context is in incident.json. Review incident_cluster_permissions.json to understand your cluster access constraints. Perform immediate triage and root cause analysis. Write findings to output/investigation.md"
```

**Current system prompt** (triage-system-prompt.md):
```
1. Read incident_cluster_permissions.json...
2. Read incident.json...
3. Use the k8s-troubleshooter skill...
4. Write findings to output/investigation.md
```

Result: Conflicting instructions, reduced agent autonomy within the skill's framework.

### Issue 2: Mandatory Field Prevents Optional Usage

Configuration validates that `agent_prompt` is **required** (config/config.go:259). This forces every invocation to have an agent prompt, even when the system prompt alone is sufficient for skill-driven workflows.

For skill-driven triage, only the system prompt matters; the agent prompt should be optional context (e.g., cluster-specific escalation info, expected SLOs).

### Issue 3: Missing Prompt Auditability

The actual prompt sent to the agent (system + config combined) is never captured or stored. This creates gaps for:
- **Forensic analysis**: "Why did the agent recommend this remediation?"
- **Compliance audits**: "What instructions was the system given?"
- **Debugging**: "What did the agent actually see as context?"

Log files show the command-line arguments, but not the resolved prompt content.

## Solution Overview

### Phase 1: Configuration (Optional Agent Prompt)
- Rename `agent_prompt` → `additional_agent_prompt`
- Make optional (validation no longer requires it)
- Allows clusters to provide context without dictating investigation steps

### Phase 2: System Prompt (Skill-Aware and Minimal)
- Remove step-by-step investigation instructions
- Add skill context and constraints
- ~20 lines instead of 36; minimal and enabling

### Phase 3: Prompt Capture (Auditability)
- Before agent execution, capture combined prompt
- Include system prompt + additional_agent_prompt + metadata
- Store in workspace as `prompt-sent.md`

### Phase 4: Storage Integration (Compliance)
- Upload `prompt-sent.md` to Azure Blob Storage
- Include in local filesystem storage
- Make available for audits and incident forensics

## Success Criteria

1. ✅ `additional_agent_prompt` is optional in configuration
2. ✅ System prompt is <25 lines and skill-enabling
3. ✅ Full prompt (system + additional + metadata) captured before execution
4. ✅ Captured prompt stored locally and uploaded to Azure
5. ✅ Agent successfully invokes k8s-troubleshooter skill with full autonomy
6. ✅ No breaking changes to existing logs or incident artifacts (prompt is additional data)

## Impact

**Breaking Changes:**
- `agent_prompt` field no longer exists; replaced with optional `additional_agent_prompt`
- Existing configs must be updated (migration: rename field if used)

**Non-Breaking Enhancements:**
- Captured prompts are additional artifacts (don't affect existing processing)
- System prompt rewrite doesn't change agent tools or capabilities
- Storage changes are backward compatible (Azure upload extends existing pattern)

**Scope:**
- Configuration parsing and validation
- System prompt content
- Executor prompt handling
- Storage and artifact upload
- Does NOT change: kubeconfig handling, cluster permissions, incident schema, Slack notifications

## Dependencies

- Assumes k8s-troubleshooter skill is installed in agent container (pre-existing requirement)
- Assumes azure.go storage module is functional (from multi-cluster-support change)
- Assumes executor.go can be extended without major refactoring

## Out of Scope

- Agent container modifications (skill loading is container responsibility)
- Changes to incident.json schema
- Changes to Slack notification format
- Changes to cluster permissions validation
