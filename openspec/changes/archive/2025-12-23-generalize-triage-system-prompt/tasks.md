# Tasks: Generalize Triage System Prompt

## Prerequisites

- [x] 0.1 Verify k8s4agents/add-standardized-report-format is applied (methodology in skill)

## 1. System Prompt Simplification

- [x] 1.1 Review current triage-system-prompt.md content
- [x] 1.2 Identify and remove Kubernetes-specific content
- [x] 1.3 Rewrite prompt as generic IT triage context (~20 lines)
- [x] 1.4 Include only:
  - Workspace file references (incident.json)
  - Operational constraints (read-only, audit documentation)
  - Output location (output/investigation.md)
  - Reference to skill for methodology

## 2. Implemented Content

Final triage-system-prompt.md (21 lines):

```markdown
# IT Incident Triage Agent

You are investigating a production incident.

## Workspace
- `incident.json` - Incident context from monitoring system

## Approach
Use your mounted skill for investigation methodology and report generation.
The skill provides:
- Systematic diagnostic workflows
- Standardized report format
- Domain-specific troubleshooting guidance

## Constraints
- **READ-ONLY**: Do not make changes to production systems
- **Document everything**: Your proof of work matters for audit

## Output
Write your investigation report to: `output/investigation.md`
Follow the standardized report format defined in your skill.
```

Note: `incident_cluster_permissions.json` removed - context will be injected into agent prompt separately.

## 3. Validation

- [x] 3.1 Test with Claude agent using k8s-troubleshooter skill (manual - owner verified)
- [x] 3.2 Verify agent still follows skill methodology (manual - owner verified)
- [x] 3.3 Verify standardized report format is used (manual - owner verified)
- [x] 3.4 Confirm no Kubernetes-specific assumptions in prompt (verified - prompt is domain-agnostic)

## Dependencies

- [x] k8s4agents skill updated with standardized report format (verified)
- [x] Skill available at https://github.com/randybias/k8s4agents

## Notes

- This change makes Nightcrier future-ready for non-Kubernetes triage domains
- The prompt becomes a thin runtime context layer
- All domain expertise lives in the skill
- Skill should be pulled from official GitHub repo (caching and versioning to be added later)
- Additional context (permissions, cluster info) injected into agent prompt at runtime
