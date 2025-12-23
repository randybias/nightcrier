# Optimize Agent Startup

## Problem Statement

Current agent execution wastes significant time on repetitive upfront tasks:

1. **Sequential file reading**: Agents spend multiple turns reading incident.json, incident_cluster_permissions.json, and other context files one by one
2. **Redundant triage execution**: Every agent runs `incident_triage.sh --skip-dump` to get baseline diagnostics, but this could be precomputed

This results in:
- Slower time-to-diagnosis (30-60+ seconds of overhead per incident)
- Higher token costs from redundant operations

## Proposed Solution

### Context Preloading

Preload maximum relevant context into the agent's initial prompt by:

1. Reading incident.json and incident_cluster_permissions.json on the host
2. Running the skill's baseline triage script before agent invocation (e.g., `incident_triage.sh --skip-dump` for K8s)
3. Bundling all outputs into the initial agent prompt as embedded context
4. Providing file references so agents know what's already available

This eliminates 3-5 turns of file reading and subprocess execution, giving agents immediate access to:
- Incident details (resource, namespace, fault type, severity)
- Access permissions
- Baseline triage results from the skill

### Architecture

The runner preloads domain-specific context while the system prompt remains generic:

```
┌─────────────────────────────────────────────────────────────┐
│ System Prompt (generic, from configs/)                      │
├─────────────────────────────────────────────────────────────┤
│ Preloaded Context (domain-specific, injected by runner)     │
│ <incident>incident.json contents</incident>                 │
│ <permissions>permissions.json contents</permissions>        │
│ <triage>skill baseline triage output</triage>               │
├─────────────────────────────────────────────────────────────┤
│ User Prompt (the investigation request)                     │
└─────────────────────────────────────────────────────────────┘
```

This maintains separation of concerns:
- System prompt: generic IT triage guidance (see `generalize-triage-system-prompt`)
- Preloaded context: domain-specific data injected by runner
- Report format: defined by the mounted skill (see k8s4agents `add-standardized-report-format`)

## Success Criteria

1. **Performance**: Agent investigations complete 30-50% faster (measured wall-clock time)
2. **Token efficiency**: 20-30% reduction in total tokens used per investigation
3. **Quality**: Investigation quality maintained or improved

## Scope

**In Scope:**
- Modifying run-agent.sh to preload context
- Creating context bundling logic in agent runners
- Context size monitoring and truncation

**Out of Scope:**
- Changes to MCP server or event intake
- Modifications to agent CLI tools themselves
- Report format changes (handled by `add-standardized-report-format` in skill)
- System prompt content changes (handled by `generalize-triage-system-prompt`)

## Dependencies

- Requires agent-container spec (already exists)
- Requires skill with baseline triage script (k8s-troubleshooter has `incident_triage.sh`)
- Report format defined by skill (see k8s4agents `add-standardized-report-format`)
- System prompt generalization (see `generalize-triage-system-prompt`)

## Risks and Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Preloaded context too large | Token limits exceeded | Implement context size checks and truncation |
| Triage script errors | Agent starts with incomplete context | Graceful degradation - proceed without triage if fails |
| Skill not available | No baseline triage | Skip triage preloading, agent runs baseline itself |
