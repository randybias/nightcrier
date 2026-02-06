# Proposal: Enforce Triage Report Format

## Why

Agent-generated triage reports currently deviate from the k8s-troubleshooter skill's 7-section format despite explicit instructions. Reports show redundancy (e.g., "Incident Overview" → "Primary Issue" → "Key Findings" repeating the same information), missing required elements (FACT/INF labeling, falsification tests, proof of work), and verbosity issues.

**User Value**: Operators receive consistent, structured triage reports that enable rapid decision-making and clear audit trails.

**Technical Value**: Enforced format compliance ensures reports can be parsed programmatically, metrics can be extracted reliably, and report quality is predictable.

## Problem Statement

Analysis of two recent triage reports (`f106c582-3185-4183-908a-ac376cbe843c` and `f59f1908-ddec-4f25-9b66-c6248df8d05b`) revealed:

1. **Format Non-Compliance**: Reports create custom sections ("Incident Overview", "Primary Issue") instead of using the mandatory 7-section structure defined in k8s-troubleshooter SKILL.md
2. **Missing Labels**: Facts and inferences are mixed without `[FACT-n]`/`[INF-n]` labeling
3. **No Falsification Tests**: Hypotheses lack concrete commands with expected results
4. **Missing Sections**: "Proof of Work" (commands executed) and "Supporting Evidence" (raw output) are entirely absent
5. **Verbosity**: Agents generate prose paragraphs instead of structured tables and lists

Root cause: The current prompt instructions in `base-triage-prompt.md` use "MUST" language but lack enforcement mechanisms. Agents interpret the 7-section requirement as aspirational guidance rather than a literal template.

## Proposed Solution

Implement a hybrid enforcement approach:

1. **Template-Constrained Skill Documentation**: Add explicit "Report Template - MANDATORY STRUCTURE" section to k8s-troubleshooter SKILL.md with literal template and compliance rules
2. **Stop Hook Validation**: Create validation script that runs when Claude attempts to end session, checking for required sections and elements
3. **Single Validation Run**: Use state tracking to ensure validation runs only once, preventing infinite loops
4. **Minimal Base Prompt**: Keep `base-triage-prompt.md` domain-agnostic, referencing skill for format details

### Stop Hook Safety

The Stop hook validator prevents infinite validation loops through:

1. **Exit Code Semantics**:
   - Exit 0: Validation passed, session ends normally
   - Exit 1: Critical failure, session blocked with error message to agent
   - Exit 2: Warnings only, session ends but warnings logged

2. **Single Validation Guarantee**:
   - Hook state is **not persisted** between agent turns
   - When agent regenerates report and attempts to stop again, hook re-runs fresh
   - Agent receives immediate feedback on each stop attempt
   - No loop risk: Each stop attempt gets one validation, agent must fix issues before next stop

3. **Validation Scope**:
   - Check section headers present (7 required sections)
   - Check FACT/INF labels exist
   - Check required elements (Most Dangerous Assumption, Falsification Tests, Proof of Work)
   - Do NOT validate content quality (prevents subjective loops)

### Integration Points

- **k8s4agents repository**: Add template section to `skills/k8s-troubleshooter/SKILL.md`
- **k8s4agents repository**: Create `skills/k8s-troubleshooter/scripts/validate-report.sh`
- **k8s4agents repository**: Add `skills/k8s-troubleshooter/skill-hooks.json` for Stop hook configuration
- **nightcrier repository**: Minor update to `configs/base-triage-prompt.md` (reference template explicitly)

## Success Criteria

1. **100% Section Compliance**: All generated reports include all 7 required sections with exact names
2. **FACT/INF Labeling**: All reports use `[FACT-n]` and `[INF-n]` labels in Assessment & Findings
3. **Falsification Tests**: All hypotheses include concrete commands with expected results
4. **Proof of Work**: All reports document commands executed during investigation
5. **No Infinite Loops**: Validation runs once per stop attempt, agent can always exit (with or without fixing issues)
6. **Domain Isolation**: Template and validation logic remain in k8s4agents skill, not nightcrier

## Impact Assessment

### Benefits

1. **Reduced Redundancy**: Template structure eliminates repeated information across sections
2. **Better Decision Speed**: Executive Card enables 30-60 second decisions
3. **Audit Trail**: Proof of Work documents all investigation steps
4. **Programmatic Parsing**: Consistent format enables metric extraction and reporting
5. **Agent Feedback Loop**: Stop hook provides immediate, actionable feedback

### Risks

1. **Agent Frustration**: If validation is too strict, agents might struggle to pass
   - **Mitigation**: Use warnings (exit code 2) for non-critical issues
2. **Skill Distribution**: Changes to k8s4agents affect all consumers
   - **Mitigation**: k8s4agents is managed separately; nightcrier pins to specific version
3. **Claude Code Compatibility**: Stop hooks are Claude-specific
   - **Mitigation**: Other agents (codex/gemini/goose) benefit from template even without hook

### Backwards Compatibility

- Existing reports remain valid (no migration needed)
- New reports follow stricter format going forward
- Validation is additive (doesn't break existing functionality)

## Alternatives Considered

### Option 1: Post-Upload Validation (Rejected)

Validate report after upload to object storage, alert if non-compliant.

**Rejected because**: Feedback comes too late - agent session already ended, can't fix issues.

### Option 2: Pre-Generation Template Injection (Rejected)

Inject full template into agent prompt at runtime.

**Rejected because**: Violates domain isolation principle - template details should live in skill, not nightcrier config.

### Option 3: LLM-Based Validation (Rejected)

Use second LLM call to validate report quality.

**Rejected because**: Expensive, slow, and subjective - would require defining "good enough" criteria that are hard to quantify.

## Dependencies

- k8s4agents repository write access (for skill and validation script changes)
- Claude Code v2.1.5+ (Stop hook support)
- No nightcrier code changes required (only config file update)

## Open Questions

None - design is complete and ready for implementation.
