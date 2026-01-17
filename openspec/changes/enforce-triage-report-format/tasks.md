# Tasks: Enforce Triage Report Format

## Overview

Implement template-based report format enforcement using k8s-troubleshooter skill enhancements and Claude Code Stop hooks.

**Estimated Effort**: 4-6 hours
**Dependencies**: k8s4agents repository access
**Testing Strategy**: Manual validation with test reports, live triage run verification

**Status**: Implementation complete, ready for live testing

---

## Phase 1: Skill Template Enhancement (k8s4agents)

### 1.1 Add Mandatory Template Section to SKILL.md

- [x] Open `/Users/rbias/code/k8s4agents/skills/k8s-troubleshooter/SKILL.md`
- [x] Locate "Report Template Overview" section (after line 121)
- [x] Insert new "Report Template - MANDATORY STRUCTURE" section
- [x] Include literal template with all 7 sections
- [x] Add placeholders in brackets for agent to fill
- [x] Document Template Compliance Rules
- [x] List common violations to avoid
- [x] Preserve existing depth guidelines (P1/P2 vs P3/P4)

**Validation**: ✅ Template section added successfully with full structure

**Actual Time**: 1 hour

---

### 1.2 Create Report Validation Script

- [x] Create `/Users/rbias/code/k8s4agents/skills/k8s-troubleshooter/scripts/validate-report.sh`
- [x] Add shebang and script metadata
- [x] Implement `check_section()` function for required sections
- [x] Implement `check_fact_inf_labels()` function for labeling
- [x] Implement `check_required_elements()` function for key elements
- [x] Add main validation logic with clear error messages
- [x] Use exit code 0 for pass, 1 for critical failure, 2 for warnings
- [x] Make script executable: `chmod +x validate-report.sh`
- [x] Add shellcheck validation: `shellcheck validate-report.sh`

**Validation**: ✅ All test cases passed:
- Compliant report → exit 0
- Missing Section 3 → exit 1
- Warnings only → exit 2

**Actual Time**: 2 hours

---

### 1.3 Configure Stop Hook

- [x] Create `/Users/rbias/code/k8s4agents/skills/k8s-troubleshooter/skill-hooks.json`
- [x] Add Stop hook configuration pointing to `./scripts/validate-report.sh`
- [x] Set timeout to 10000ms (10 seconds)
- [x] Add description: "Validate triage report format compliance"
- [x] Test hook configuration syntax with `cat skill-hooks.json | jq`

**Validation**: ✅ JSON valid and follows Claude Code hook schema

**Actual Time**: 15 minutes

---

## Phase 2: Base Prompt Refinement (nightcrier)

### 2.1 Update base-triage-prompt.md

- [x] Open `/Users/rbias/code/nightcrier/configs/base-triage-prompt.md`
- [x] Locate "Required First Step" section (lines 18-34)
- [x] Update to reference "Report Template - MANDATORY STRUCTURE" section
- [x] Add explicit instructions to copy template structure
- [x] Emphasize DO NOT add/remove/rename sections
- [x] Keep changes minimal (9 lines modified)
- [x] Preserve existing skill path references

**Validation**: ✅ Prompt updated, references template without redundancy

**Actual Time**: 20 minutes

---

## Phase 2.2: Hook Integration (nightcrier)

### 2.2.1 Implement Skill Hook Loading in nc-agent-runner

- [x] Add `merge_skill_hooks()` function to entrypoint.sh
- [x] Scan ~/.claude/skills for skill-hooks.json files
- [x] Convert relative command paths to absolute paths
- [x] Merge skill hooks with NATS hooks (if enabled)
- [x] Write merged hooks to ~/.claude/settings.json
- [x] Call merge_skill_hooks() from setup_agent_paths()

**Validation**: ✅ Function implemented with jq-based merging logic

**Actual Time**: 45 minutes

---

### 2.2.2 Add NATS Publishing for Stop Hook

- [x] Create `/nc-agent-runner/hooks/nats-validating.sh` wrapper script
- [x] Publish `validating.started` event before validation
- [x] Publish `validating.completed` or `validating.failed` after validation
- [x] Pass validation exit code in NATS payload
- [x] Wrap Stop hook commands with NATS wrapper when NATS_ENABLED=true
- [x] Set VALIDATION_SCRIPT env var to original validation script path
- [x] Make nats-validating.sh executable
- [x] Run shellcheck on both scripts

**Validation**: ✅ Scripts created and tested locally with jq

**Actual Time**: 1 hour

---

## Phase 3: Testing & Validation

### 3.1 Create Test Reports

- [x] Create `scratch/test-report-compliant.md` following template exactly
- [x] Create `scratch/test-report-missing-section.md` with missing Section 3
- [x] Create `scratch/test-report-warnings-only.md` without FACT/INF labels

**Validation**: ✅ All test reports created and validated:
- Compliant report → exit 0 ✅
- Missing section → exit 1 ✅
- Warnings only → exit 2 ✅

**Actual Time**: 30 minutes

---

### 3.2 Test Stop Hook with Claude

- [ ] Set up test workspace with k8s4agents skill
- [ ] Run Claude with test prompt: `claude -p --skill k8s-troubleshooter "Write a test report"`
- [ ] Verify Stop hook triggers when Claude attempts to end session
- [ ] Confirm validation errors appear in Claude's context
- [ ] Verify agent can regenerate report and retry
- [ ] Confirm agent can eventually exit (no infinite loop)

**Validation**: Pending - requires live triage run

**Estimated Time**: 1 hour

**Note**: Hook integration implemented in nc-agent-runner. Stop hooks now auto-loaded from skill-hooks.json files.

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

**Validation**: Pending - requires k8s4agents main branch update

**Estimated Time**: 1 hour

**Note**: This will test end-to-end flow with nc-agent-runner cloning from GitHub

---

## Phase 4: Documentation & Cleanup

### 4.1 Update k8s4agents README

- [ ] Document new validation script in k8s4agents README
- [ ] Add usage example: `./scripts/validate-report.sh path/to/report.md`
- [ ] Document Stop hook behavior
- [ ] Note that hook is Claude-specific

**Validation**: Pending

**Estimated Time**: 20 minutes

---

### 4.2 Commit Changes to k8s4agents

- [x] Stage changes: `SKILL.md`, `scripts/validate-report.sh`, `skill-hooks.json`
- [x] Run shellcheck on validation script
- [x] Commit with message: `feat(k8s-troubleshooter): add report format validation with Stop hook`
- [x] Push to k8s4agents repository

**Validation**: ✅ Changes committed to `feature/add-report-format-validation` branch and pushed

**Branch**: https://github.com/randybias/k8s4agents/tree/feature/add-report-format-validation

**Actual Time**: 10 minutes

---

### 4.3 Commit Changes to nightcrier

- [x] Stage change: `configs/base-triage-prompt.md`, OpenSpec proposal files
- [x] Commit with message: `docs(config): reference k8s-troubleshooter template structure`
- [x] Commit to feature branch: `feature/enforce-triage-report-format`

**Validation**: ✅ Changes committed to feature branch

**Actual Time**: 5 minutes

---

## Dependencies

- **Phase 2 depends on Phase 1**: ✅ Complete
- **Phase 3 depends on Phases 1-2**: ⚠️ Partially complete (local testing done, live testing pending)
- **Phase 4 can proceed in parallel**: ⚠️ Documentation pending

## Next Steps for Testing

1. **Merge k8s4agents PR**: Merge `feature/add-report-format-validation` to main branch
2. **Test Stop Hook**: Run Claude locally with k8s-troubleshooter skill to verify hook behavior
3. **Live Triage Test**: Trigger incident and verify end-to-end flow with nc-agent-runner
4. **Update Documentation**: Complete k8s4agents README updates
5. **Merge nightcrier PR**: Merge `feature/enforce-triage-report-format` after validation

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
