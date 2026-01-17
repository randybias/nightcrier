# Tasks: Enforce Triage Report Format

## Overview

Implement template-based report format enforcement using k8s-troubleshooter skill enhancements and Claude Code Stop hooks.

**Estimated Effort**: 4-6 hours
**Dependencies**: k8s4agents repository access
**Testing Strategy**: Manual validation with test reports, live triage run verification

---

## Phase 1: Skill Template Enhancement (k8s4agents)

### 1.1 Add Mandatory Template Section to SKILL.md

- [ ] Open `/Users/rbias/code/k8s4agents/skills/k8s-troubleshooter/SKILL.md`
- [ ] Locate "Report Template Overview" section (after line 121)
- [ ] Insert new "Report Template - MANDATORY STRUCTURE" section
- [ ] Include literal template with all 7 sections
- [ ] Add placeholders in brackets for agent to fill
- [ ] Document Template Compliance Rules
- [ ] List common violations to avoid
- [ ] Preserve existing depth guidelines (P1/P2 vs P3/P4)

**Validation**: Read SKILL.md and verify template is clear, complete, and unambiguous.

**Estimated Time**: 1 hour

---

### 1.2 Create Report Validation Script

- [ ] Create `/Users/rbias/code/k8s4agents/skills/k8s-troubleshooter/scripts/validate-report.sh`
- [ ] Add shebang and script metadata
- [ ] Implement `check_section()` function for required sections
- [ ] Implement `check_fact_inf_labels()` function for labeling
- [ ] Implement `check_required_elements()` function for key elements
- [ ] Add main validation logic with clear error messages
- [ ] Use exit code 0 for pass, 1 for critical failure, 2 for warnings
- [ ] Make script executable: `chmod +x validate-report.sh`
- [ ] Add shellcheck validation: `shellcheck validate-report.sh`

**Validation**: Test script against:
- Compliant report (exit 0)
- Report missing Section 3 (exit 1)
- Report with no FACT labels (exit 2)

**Estimated Time**: 2 hours

---

### 1.3 Configure Stop Hook

- [ ] Create `/Users/rbias/code/k8s4agents/skills/k8s-troubleshooter/skill-hooks.json`
- [ ] Add Stop hook configuration pointing to `./scripts/validate-report.sh`
- [ ] Set timeout to 10000ms (10 seconds)
- [ ] Add description: "Validate triage report format compliance"
- [ ] Test hook configuration syntax with `cat skill-hooks.json | jq`

**Validation**: Verify JSON is valid and follows Claude Code hook schema.

**Estimated Time**: 15 minutes

---

## Phase 2: Base Prompt Refinement (nightcrier)

### 2.1 Update base-triage-prompt.md

- [ ] Open `/Users/rbias/code/nightcrier/configs/base-triage-prompt.md`
- [ ] Locate "Required First Step" section (lines 18-34)
- [ ] Update to reference "Report Template - MANDATORY STRUCTURE" section
- [ ] Add explicit instructions to copy template structure
- [ ] Emphasize DO NOT add/remove/rename sections
- [ ] Keep changes minimal (5-10 lines modified)
- [ ] Preserve existing skill path references

**Validation**: Read updated prompt and verify it's clear but not redundant with skill.

**Estimated Time**: 30 minutes

---

## Phase 3: Testing & Validation

### 3.1 Create Test Reports

- [ ] Create `scratch/test-report-compliant.md` following template exactly
- [ ] Create `scratch/test-report-missing-sections.md` with only 5 sections
- [ ] Create `scratch/test-report-no-labels.md` without FACT/INF labels

**Validation**: Validation script should:
- Pass compliant report (exit 0)
- Fail missing sections (exit 1)
- Warn on no labels (exit 2)

**Estimated Time**: 30 minutes

---

### 3.2 Test Stop Hook with Claude

- [ ] Set up test workspace with k8s4agents skill
- [ ] Run Claude with test prompt: `claude -p --skill k8s-troubleshooter "Write a test report"`
- [ ] Verify Stop hook triggers when Claude attempts to end session
- [ ] Confirm validation errors appear in Claude's context
- [ ] Verify agent can regenerate report and retry
- [ ] Confirm agent can eventually exit (no infinite loop)

**Validation**:
- Hook runs on stop attempt
- Validation feedback is visible to agent
- Agent can fix and retry
- Agent can exit after passing validation

**Estimated Time**: 1 hour

---

### 3.3 Live Triage Run Test

- [ ] Trigger a test incident in westeu-cluster1 (crashloop pod)
- [ ] Monitor nightcrier logs for agent execution
- [ ] Wait for triage completion
- [ ] Retrieve generated report from object storage
- [ ] Verify report follows 7-section structure
- [ ] Check for FACT/INF labels
- [ ] Check for Most Dangerous Assumption
- [ ] Check for Falsification Tests
- [ ] Check for Proof of Work section
- [ ] Check for Supporting Evidence section

**Validation**: Generated report passes all validation checks.

**Estimated Time**: 1 hour

---

## Phase 4: Documentation & Cleanup

### 4.1 Update k8s4agents README

- [ ] Document new validation script in k8s4agents README
- [ ] Add usage example: `./scripts/validate-report.sh path/to/report.md`
- [ ] Document Stop hook behavior
- [ ] Note that hook is Claude-specific

**Validation**: README accurately describes validation feature.

**Estimated Time**: 20 minutes

---

### 4.2 Commit Changes to k8s4agents

- [ ] Stage changes: `SKILL.md`, `scripts/validate-report.sh`, `skill-hooks.json`
- [ ] Run shellcheck on validation script
- [ ] Commit with message: `feat(k8s-troubleshooter): add report format validation with Stop hook`
- [ ] Push to k8s4agents repository

**Validation**: Changes committed and pushed successfully.

**Estimated Time**: 10 minutes

---

### 4.3 Commit Changes to nightcrier

- [ ] Stage change: `configs/base-triage-prompt.md`
- [ ] Commit with message: `docs(config): reference k8s-troubleshooter template structure`
- [ ] Push to nightcrier repository

**Validation**: Change committed and pushed successfully.

**Estimated Time**: 5 minutes

---

## Dependencies

- **Phase 2 depends on Phase 1**: Can't reference template structure until it exists
- **Phase 3 depends on Phases 1-2**: Testing requires all components in place
- **Phase 4 can proceed in parallel**: Documentation can be written alongside implementation

## Rollback Plan

If validation proves too strict or causes issues:

1. **Immediate**: Remove `skill-hooks.json` to disable Stop hook
2. **Short-term**: Adjust validation script to use warnings (exit 2) instead of errors (exit 1)
3. **Long-term**: Revise template structure based on agent feedback

## Success Metrics

- **Primary**: Next 5 triage reports all include 7 sections with correct names
- **Secondary**: Reports include FACT/INF labels in Assessment & Findings
- **Tertiary**: Reports include Proof of Work with commands executed
- **Operational**: No infinite validation loops observed in production
