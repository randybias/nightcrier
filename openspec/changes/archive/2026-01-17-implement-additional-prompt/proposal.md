# Proposal: Implement Additional Prompt

## Summary

Wire up the existing `agent.additional_prompt` config field to actually append operator-specified text to the unified triage prompt at runtime. The additional prompt will be postpended as the final step of prompt aggregation.

## Motivation

The `AdditionalPrompt` config field is specified in the configuration spec (line 59-63) but never used in execution logic. Operators need a way to inject cluster-specific context, escalation procedures, SLO targets, or other operational guidance without modifying the base triage prompt.

**Current state:**
- Config field exists and is parsed: `internal/config/config.go:33`
- Tests verify parsing: `internal/config/config_test.go:2051-2052`
- Field is displayed in banner but never passed to agent
- Example config shows intended use: `configs/config.example.yaml:99`

**Intended use cases:**
- Cluster-specific escalation: "Page oncall@example.com for P1 issues"
- SLO context: "This cluster has 99.9% uptime SLA"
- Environment hints: "This is a staging cluster, be aggressive with debugging"
- Team context: "Contact platform-team Slack channel for infra issues"

## Design

### Prompt Aggregation Order

The unified triage prompt is built in `build_triage_prompt()` in entrypoint.sh:

```
1. Base triage prompt (base-triage-prompt.md from ConfigMap)
2. Incident data (incident.json)
3. Cluster permissions (incident_cluster_permissions.json)
4. [NEW] Additional prompt (operator-specified, optional)  <-- APPENDED LAST
```

The additional prompt goes last because:
- It's operator override/customization
- Later content has more influence on agent behavior
- Matches the spec: "system prompt SHALL drive investigation methodology"

### Data Flow

```
Config (agent.additional_prompt)
    ↓
K8sExecutor.loadIncidentData()  ← read additional prompt
    ↓
ConfigMapData.AdditionalPrompt  ← add to struct
    ↓
ConfigMap (additional-prompt.md) ← mount in container
    ↓
entrypoint.sh build_triage_prompt() ← append to prompt
    ↓
Agent receives unified prompt
```

### File Changes

| File | Change |
|------|--------|
| `internal/agent/k8s/configmap.go` | Add `AdditionalPrompt` field to `ConfigMapData` |
| `internal/agent/k8s_executor.go` | Pass `AdditionalPrompt` from config to ConfigMap |
| `internal/agent/k8s/job.go` | Mount `additional-prompt.md` from ConfigMap |
| `nc-agent-runner/entrypoint.sh` | Append additional prompt in `build_triage_prompt()` |

## Scope

- Pass `AdditionalPrompt` from config through to K8s Job
- Mount additional prompt file in container
- Append to unified triage prompt as final step
- Capture in `prompt-sent.md` for audit trail

## Out of Scope

- Per-cluster additional prompts (would require multi-cluster registry changes)
- Template variables in additional prompt (e.g., `{{cluster_name}}`)
- Validation of additional prompt content
