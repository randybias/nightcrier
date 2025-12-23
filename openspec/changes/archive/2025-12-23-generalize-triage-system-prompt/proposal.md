# Change: Generalize Triage System Prompt

## Why

The current `triage-system-prompt.md` contains Kubernetes-specific content that should live in the triage skill (k8s-troubleshooter), not in Nightcrier's orchestration layer.

Nightcrier is a generic incident triage orchestrator. In the future, it may run triage agents for:
- Kubernetes clusters (current)
- Cloud services (AWS, Azure, GCP)
- Databases
- Network infrastructure
- Application-specific systems

The system prompt should focus on:
- Runtime context (what files are in the workspace)
- Operational constraints (read-only, timeout)
- Output expectations (where to write, what format)
- "Use your skill for investigation methodology"

Domain-specific triage knowledge belongs in the skill, not the orchestrator.

## What Changes

1. **Simplify triage-system-prompt.md** to ~20 lines of generic IT triage context:
   - Workspace file locations
   - Operational constraints
   - Output location
   - Reference to skill for methodology

2. **Remove Kubernetes-specific content**:
   - No references to kubectl, pods, namespaces
   - No references to incident_triage.sh (that's in the skill)
   - No Kubernetes-specific file formats

3. **Make prompt skill-agnostic**:
   - Works with any mounted triage skill
   - Skill provides domain-specific methodology
   - Prompt provides runtime context only

## Impact

- Affected specs: agent-runtime
- Affected code: configs/triage-system-prompt.md
- Breaking changes: None - agents using k8s-troubleshooter skill will have same capabilities (methodology now in skill)
- Dependencies: Requires k8s4agents/add-standardized-report-format to be applied first (moves methodology to skill)
