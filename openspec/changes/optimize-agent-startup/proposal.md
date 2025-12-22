# Optimize Agent Startup

## Problem Statement

Current agent execution wastes significant time on repetitive upfront tasks:

1. **Sequential file reading**: Agents spend multiple turns reading incident.json, incident_cluster_permissions.json, and other context files one by one
2. **Redundant triage execution**: Every agent runs `incident_triage.sh --skip-dump` to get baseline diagnostics, but this could be precomputed
3. **Inconsistent report structure**: Investigation reports vary significantly in format, making it harder to parse and compare results programmatically

This results in:
- Slower time-to-diagnosis (30-60+ seconds of overhead per incident)
- Higher token costs from redundant operations
- Inconsistent output quality and structure

## Proposed Solution

### 1. Context Preloading

Preload maximum relevant context into the agent's initial prompt by:

1. Reading incident.json and incident_cluster_permissions.json on the host
2. Running `incident_triage.sh --skip-dump` before agent invocation
3. Bundling all outputs into the initial agent prompt as embedded context
4. Providing file references so agents know what's already available

This eliminates 3-5 turns of file reading and subprocess execution, giving agents immediate access to:
- Incident details (resource, namespace, fault type, severity)
- Cluster access permissions
- Baseline triage results (node health, warning events, crashlooping pods)

### 2. Structured Report Template

Enforce a standardized investigation report structure with exactly four sections:

1. **Problem Statement** - Concise description of the incident (2-3 sentences)
2. **Summary of Findings** - Root cause analysis with confidence level
3. **Recommended Immediate Remediation Steps** - Prioritized action items
4. **Supporting Evidence and Work Done** - Detailed kubectl outputs, logs, and diagnostic workflow

This improves:
- **Consistency**: Predictable format for downstream parsing
- **Readability**: Key information always in the same location
- **Completeness**: Template ensures all required sections are present

## Success Criteria

1. **Performance**: Agent investigations complete 30-50% faster (measured wall-clock time)
2. **Token efficiency**: 20-30% reduction in total tokens used per investigation
3. **Report consistency**: 100% of reports follow the four-section structure
4. **Quality**: Confidence levels and root cause identification remain at or above current levels

## Scope

**In Scope:**
- Modifying run-agent.sh to preload context
- Creating context bundling logic in agent runners
- Updating triage-system-prompt.md with new structure requirements
- Validating report structure in post-processing

**Out of Scope:**
- Changes to MCP server or event intake
- Modifications to agent CLI tools themselves
- Changes to existing k8s-troubleshooter skill scripts
- Alterations to Docker container base image

## Dependencies

- Requires agent-container spec (already exists)
- Builds on existing incident_triage.sh script from k8s-troubleshooter skill
- No external service dependencies

## Risks and Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Preloaded context too large | Token limits exceeded | Implement context size checks and truncation |
| Triage script errors | Agent starts with incomplete context | Graceful degradation - proceed without triage if fails |
| Report structure too rigid | Loss of valuable agent insights | Allow "Additional Notes" section for unstructured findings |
| Breaking existing reports | Downstream tooling failures | Version the report format, support both during migration |
