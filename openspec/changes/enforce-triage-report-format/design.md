# Design: Enforce Triage Report Format

## Architecture Overview

This change implements format enforcement through **template constraints** (reducing agent creativity) and **validation hooks** (catching deviations). The solution is split across two repositories to maintain domain separation.

### Component Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                    k8s4agents Repository                        │
│                                                                 │
│  skills/k8s-troubleshooter/                                     │
│  ├── SKILL.md                                                   │
│  │   └── Report Template - MANDATORY STRUCTURE                 │
│  │       (Literal template with placeholders)                  │
│  ├── scripts/                                                   │
│  │   └── validate-report.sh                                    │
│  │       (Checks sections, labels, required elements)          │
│  └── skill-hooks.json                                          │
│      └── Stop: ./scripts/validate-report.sh                    │
│          (Triggers on Claude session end)                      │
└─────────────────────────────────────────────────────────────────┘
                             │
                             │ git clone (at runtime)
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│              nc-agent-runner Container                          │
│                                                                 │
│  /home/agent/skills/k8s4agents/                                 │
│  └── (skill files cloned from GitHub)                          │
│                                                                 │
│  ~/.claude/skills -> /home/agent/skills/k8s4agents/skills      │
│  (symlink created by entrypoint.sh)                            │
└─────────────────────────────────────────────────────────────────┘
                             │
                             │ skill loaded
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Claude Code Session                          │
│                                                                 │
│  1. Read SKILL.md template section                             │
│  2. Generate report → output/report.md                         │
│  3. Attempt to stop session                                    │
│  4. Stop hook triggers → validate-report.sh runs               │
│  5. Validation passes/fails                                    │
│     - Exit 0: Session ends normally                            │
│     - Exit 1: Session blocked, error shown to agent            │
│     - Exit 2: Session ends with warnings                       │
└─────────────────────────────────────────────────────────────────┘
```

---

## Design Decisions

### Decision 1: Stop Hook vs. PostToolUse Hook

**Options Considered:**

1. **Stop Hook** (Chosen)
   - Triggers when agent attempts to end session
   - Validates complete report before session closes
   - Agent sees validation errors and can regenerate

2. **PostToolUse Hook on Write Tool**
   - Triggers after every file write
   - Could validate partial reports during generation
   - More invasive, runs many times

**Choice: Stop Hook**

**Rationale:**
- Validates **complete** report (not partial drafts)
- Runs **once per stop attempt** (not on every file write)
- Clear feedback loop: agent → stop → validation → fix → stop again
- Aligns with user's requirement: "only want to run one validation [per stop attempt]"

---

### Decision 2: Exit Code Semantics

**Design:**

| Exit Code | Meaning | Behavior |
|-----------|---------|----------|
| 0 | Validation passed | Session ends normally |
| 1 | Critical failure | Session blocked, error message shown to agent |
| 2 | Warnings only | Session ends, warnings logged |

**Rationale:**
- Exit 1 for **structural issues** (missing sections) - these are objective and fixable
- Exit 2 for **content warnings** (missing labels) - less critical, allows agent to proceed
- Exit 0 for **full compliance** - gold standard

This prevents infinite loops because:
- Agent can **always exit** by fixing critical issues (or ignoring warnings)
- Validation is **deterministic** (same input = same output)
- No subjective quality checks (e.g., "is this explanation good enough?")

---

### Decision 3: Template Location

**Options Considered:**

1. **In SKILL.md** (Chosen)
2. **In base-triage-prompt.md**
3. **Separate template.md file**

**Choice: In SKILL.md**

**Rationale:**
- **Domain isolation**: Format knowledge belongs in k8s-troubleshooter skill
- **Reusability**: Other projects using k8s4agents benefit from template
- **Maintainability**: Single source of truth in skill repository
- **User requirement**: "keep base-triage-prompt.md as clean as possible"

---

### Decision 4: Validation Scope

**Included (Objective Checks):**
- ✅ Section headers present (7 required sections)
- ✅ Section names exact match
- ✅ FACT/INF labels exist (regex search)
- ✅ Required elements present (Most Dangerous Assumption, Falsification Test, etc.)

**Excluded (Subjective Checks):**
- ❌ Content quality (is hypothesis convincing?)
- ❌ Evidence sufficiency (enough facts?)
- ❌ Writing style (too verbose?)
- ❌ Technical accuracy (is diagnosis correct?)

**Rationale:**
- Objective checks are **deterministic and fixable**
- Subjective checks are **ambiguous and risk loops**
- Focus on **structure compliance**, not content quality

---

### Decision 5: Single Validation Guarantee

**Problem:** How to ensure validation runs only once per stop attempt (no infinite loops)?

**Solution:** Claude Code Stop hooks are **stateless between turns**:

```
Turn 1: Agent writes report → Stops → Hook runs → Exit 1 (missing Section 3)
        Agent sees error: "Missing Section 3"

Turn 2: Agent adds Section 3 → Stops → Hook runs fresh → Exit 0 (pass)
        Session ends normally
```

**Key Insight:** Hook doesn't "remember" previous runs. Each stop attempt gets fresh validation. Agent must fix issues before next stop.

**Edge Case Handling:**

1. **Agent gives up**: After multiple failures, agent might stop trying
   - **Mitigation**: Use warnings (exit 2) for non-critical issues so agent can exit

2. **Validation script crashes**: Script error (e.g., missing file)
   - **Mitigation**: Hook timeout (10 seconds) prevents hanging
   - Script uses `set -euo pipefail` for early exit on errors

---

## Data Flow

### Validation Flow Detail

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. Agent Generation Phase                                       │
│    - Read SKILL.md template                                     │
│    - Generate report following template                         │
│    - Write to output/report.md                                  │
└────────────────────┬────────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────────┐
│ 2. Stop Attempt                                                 │
│    - Agent decides investigation complete                       │
│    - Invokes stop command                                       │
└────────────────────┬────────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────────┐
│ 3. Stop Hook Trigger                                            │
│    - Claude Code detects Stop event                             │
│    - Loads skill-hooks.json                                     │
│    - Finds Stop hook → ./scripts/validate-report.sh            │
│    - Runs script with timeout (10s)                             │
└────────────────────┬────────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────────┐
│ 4. Validation Execution                                         │
│    Script checks:                                               │
│    ✓ All 7 sections present?                                    │
│    ✓ FACT/INF labels found?                                     │
│    ✓ Required elements present?                                 │
│                                                                 │
│    Outputs:                                                     │
│    - STDOUT: Human-readable messages                            │
│    - Exit code: 0/1/2                                           │
└────────────────────┬────────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────────┐
│ 5. Decision Point                                               │
│                                                                 │
│    Exit 0? → Session ends, report uploaded                      │
│    Exit 1? → Session blocked, agent sees error                  │
│    Exit 2? → Session ends, warnings logged                      │
└─────────────────────────────────────────────────────────────────┘
                     │
                     │ (if Exit 1)
                     ▼
┌─────────────────────────────────────────────────────────────────┐
│ 6. Agent Feedback Loop                                          │
│    - Error message injected into agent context                  │
│    - Example: "ERROR: Missing required section: ## 3. Root..."  │
│    - Agent regenerates report with fixes                        │
│    - Attempts stop again (returns to step 2)                    │
└─────────────────────────────────────────────────────────────────┘
```

---

## Security Considerations

### Script Execution Safety

**Risk**: Validation script runs in agent container with elevated privileges.

**Mitigations:**
1. **Read-only operations**: Script only reads `output/report.md`, never writes
2. **Minimal dependencies**: Uses only bash, grep, and standard tools
3. **No network access**: Script is fully local
4. **Timeout protection**: 10-second timeout prevents hanging
5. **Exit code contract**: Only uses 0/1/2, no side effects

### Skill Tampering

**Risk**: Malicious changes to k8s4agents repository could compromise validation.

**Mitigations:**
1. **Version pinning**: Nightcrier can pin to specific k8s4agents commit
2. **Git clone verification**: Entrypoint.sh verifies SKILL.md exists after clone
3. **Repository trust**: k8s4agents is owned by same maintainer (randybias)

---

## Performance Considerations

### Validation Script Performance

**Expected Runtime**: < 1 second

**Operations:**
- 7 grep calls for section headers (~100ms total)
- 2 grep calls for FACT/INF labels (~50ms total)
- 3 grep calls for required elements (~50ms total)

**Total**: ~200ms on typical report (500-1000 lines)

**Timeout**: 10 seconds (50x safety margin)

### Impact on Agent Runtime

**Before**: Agent stops immediately after generating report

**After**: Agent stops + 200ms validation + feedback

**Impact**: Negligible (<1% of total triage runtime which is typically 60-300 seconds)

---

## Error Handling

### Validation Script Errors

| Error Condition | Behavior | User Impact |
|----------------|----------|-------------|
| Report file not found | Exit 1, clear error message | Agent sees "Report file not found at output/report.md" |
| Grep command fails | Exit 1, logs error | Agent sees "Validation script error" |
| Script timeout | Hook aborted, session ends | Warning logged, session proceeds |
| Invalid exit code | Treated as failure | Session blocked |

### Agent Response Strategies

**If Exit 1 (Critical Failure):**
1. Agent reads error message
2. Identifies missing sections/elements
3. Regenerates report with fixes
4. Attempts stop again

**If Exit 2 (Warnings):**
1. Agent sees warnings but session ends
2. Warnings logged to agent.log
3. Report uploaded as-is

**If Agent Gives Up:**
1. After 3-5 failed attempts, agent might stop trying
2. Session remains open (user can intervene)
3. User can force-stop or manually fix report

---

## Testing Strategy

### Unit Testing (Validation Script)

```bash
# Test 1: Compliant report
./validate-report.sh scratch/test-report-compliant.md
# Expected: Exit 0, "VALIDATION PASSED"

# Test 2: Missing Section 3
./validate-report.sh scratch/test-report-missing-section3.md
# Expected: Exit 1, "ERROR: Missing required section: ## 3. Root Cause Analysis"

# Test 3: No FACT labels
./validate-report.sh scratch/test-report-no-labels.md
# Expected: Exit 2, "WARNING: No [FACT-n] labels found"
```

### Integration Testing (Stop Hook)

```bash
# Test 1: Hook triggers on stop
claude -p --skill k8s-troubleshooter "Write test report and stop"
# Verify: Hook runs, validation output appears

# Test 2: Agent fixes errors
claude -p --skill k8s-troubleshooter "Write report missing Section 3, then fix when told"
# Verify: First stop fails, agent fixes, second stop passes

# Test 3: Agent can exit with warnings
claude -p --skill k8s-troubleshooter "Write report without FACT labels"
# Verify: Session ends with warnings, report uploaded
```

### End-to-End Testing (Live Triage)

```bash
# Trigger incident
kubectl run crashloop --image=busybox -- /bin/sh -c "exit 1"

# Monitor nightcrier
tail -f logs/nightcrier.log

# Verify report
# - Check object storage for investigation.md
# - Verify 7 sections present
# - Verify FACT/INF labels
# - Verify Proof of Work section
```

---

## Rollback Strategy

### Level 1: Disable Hook (Immediate)

```bash
# In k8s4agents repository
rm skills/k8s-troubleshooter/skill-hooks.json
git commit -m "fix: temporarily disable report validation hook"
git push
```

**Impact**: Next agent run won't have validation, reports proceed as before.

### Level 2: Relax Validation (Hours)

```bash
# Edit validate-report.sh
# Change all "exit 1" to "exit 2" (errors become warnings)
git commit -m "fix: relax validation to warnings only"
git push
```

**Impact**: Agents can always exit, validation becomes advisory.

### Level 3: Revert Template (Days)

```bash
# Remove template section from SKILL.md
# Revert base-triage-prompt.md changes
git commit -m "revert: remove report template enforcement"
git push
```

**Impact**: Return to pre-enforcement behavior.

---

## Future Enhancements

### Programmatic Report Parsing

Once format is consistent, enable:
- Metric extraction (confidence levels, hypothesis counts)
- Executive summary generation for dashboards
- Trend analysis across incidents

### Quality Scoring

After template adoption:
- Score reports on completeness (0-100)
- Track improvement over time
- Flag reports with minimal evidence

### Multi-Agent Support

Extend validation to other agent types:
- Codex: PostToolUse hook (no Stop hook support)
- Gemini: Custom validation in entrypoint.sh
- Goose: TBD based on hook capabilities

---

## Open Questions

None - design is complete and ready for implementation.
