# Tasks: Generalize Triage System Prompt

## Prerequisites

- [ ] 0.1 Verify k8s4agents/add-standardized-report-format is applied (methodology in skill)

## 1. System Prompt Simplification

- [ ] 1.1 Review current triage-system-prompt.md content
- [ ] 1.2 Identify and remove Kubernetes-specific content
- [ ] 1.3 Rewrite prompt as generic IT triage context (~20 lines)
- [ ] 1.4 Include only:
  - Workspace file references (incident.json, permissions)
  - Operational constraints (read-only, timeouts)
  - Output location (output/investigation.md)
  - Reference to skill for methodology

## 2. Proposed New Content

New triage-system-prompt.md structure:

```markdown
# IT Incident Triage Agent

You are investigating a production incident.

## Workspace
- `incident.json` - Incident context from monitoring system
- `incident_cluster_permissions.json` - Your validated access permissions (if applicable)

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

## 3. Validation

- [ ] 3.1 Test with Claude agent using k8s-troubleshooter skill
- [ ] 3.2 Verify agent still follows skill methodology
- [ ] 3.3 Verify standardized report format is used
- [ ] 3.4 Confirm no Kubernetes-specific assumptions in prompt

## Dependencies

- Requires k8s4agents/add-standardized-report-format to be applied first
- The methodology that was in the system prompt must exist in the skill before removing from prompt

## Notes

- This change makes Nightcrier future-ready for non-Kubernetes triage domains
- The prompt becomes a thin runtime context layer
- All domain expertise lives in the skill
