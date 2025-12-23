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
2. Running the K8s skill's baseline triage script before agent invocation (`incident_triage.sh --skip-dump`)
3. Bundling all outputs into the initial agent prompt as embedded context
4. Providing file references so agents know what's already available

This eliminates 3-5 turns of file reading and subprocess execution, giving agents immediate access to:
- Incident details (resource, namespace, fault type, severity)
- Access permissions
- Initial K8s triage report from the skill

### Architecture

The runner preloads domain-specific context while the system prompt remains generic:

```
┌─────────────────────────────────────────────────────────────────────────┐
│ System Prompt (generic, from configs/)                                  │
├─────────────────────────────────────────────────────────────────────────┤
│ Preloaded Context (K8s-specific, injected by runner)                    │
│ <incident>incident.json contents</incident>                             │
│ <kubernetes_cluster_access_permissions>                                 │
│   permissions.json contents                                             │
│ </kubernetes_cluster_access_permissions>                                │
│ <initial_triage_report>                                                 │
│   K8s baseline triage output from skill                                 │
│ </initial_triage_report>                                                │
├─────────────────────────────────────────────────────────────────────────┤
│ User Prompt (the investigation request)                                 │
└─────────────────────────────────────────────────────────────────────────┘
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
- Adding skills configuration to config files
- K8s-specific triage preloading

**Out of Scope:**
- Changes to MCP server or event intake
- Modifications to agent CLI tools themselves
- Report format changes (handled by `add-standardized-report-format` in skill)
- System prompt content changes (handled by `generalize-triage-system-prompt`)
- Multi-domain skill support (AWS, Azure, etc.) - K8s only for now

## Dependencies

- Requires agent-container spec (already exists)
- k8s4agents skill automatically cached from https://github.com/randybias/k8s4agents
- Report format defined by skill (see k8s4agents `add-standardized-report-format`)
- System prompt generalization (see `generalize-triage-system-prompt`)

## Skills Setup

Skills are cached in `./agent-home/skills/` (gitignored runtime directory).

**Automatic Caching**: Nightcrier automatically ensures skills are cached on startup. If the k8s-troubleshooter triage script is not found, nightcrier will:
1. Create the cache directory if needed
2. Clone k8s4agents from GitHub: https://github.com/randybias/k8s4agents
3. Continue startup (non-fatal if clone fails)

Configuration (added to existing config.yaml):
```yaml
skills:
  cache_dir: "./agent-home/skills"  # Default: ./agent-home/skills
  disable_triage_preload: false      # Default: false
```

Environment variable overrides:
- `SKILLS_CACHE_DIR`: Override cache directory location
- `SKILLS_DISABLE_TRIAGE_PRELOAD`: Disable preloading (agent will run triage itself)

## Risks and Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Preloaded context too large | Token limits exceeded | Implement context size checks and truncation |
| Triage script errors | Agent starts with incomplete context | Graceful degradation - proceed without triage if fails |
| Skill not available | No baseline triage | Skip triage preloading, agent runs baseline itself |
