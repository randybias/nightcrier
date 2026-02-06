# triage-report-validation Specification

## Purpose
TBD - created by archiving change enforce-triage-report-format. Update Purpose after archive.
## Requirements
### Requirement: Template Structure Documentation

The k8s-troubleshooter skill SHALL provide a literal report template that agents must follow exactly.

#### Scenario: Template section in SKILL.md

- **GIVEN** an agent reads the k8s-troubleshooter SKILL.md
- **WHEN** the agent reaches the "Report Template Overview" section
- **THEN** the skill SHALL include a "Report Template - MANDATORY STRUCTURE" subsection
- **AND** the template SHALL contain literal markdown with bracketed placeholders
- **AND** the template SHALL include all 7 required sections with exact header names
- **AND** the template SHALL include Template Compliance Rules
- **AND** the template SHALL list common violations to avoid

#### Scenario: Template completeness

- **GIVEN** the report template in SKILL.md
- **WHEN** an agent follows the template structure
- **THEN** the resulting report SHALL include:
  - Section 0: Executive Triage Card with status emoji, severity, impact, duration, primary hypothesis, most dangerous assumption, top 3 actions, escalation triggers, alternative hypotheses
  - Section 1: Problem Statement with symptoms, timeline, exit criteria
  - Section 2: Assessment & Findings with classification, scope, observed facts, derived inferences, what changed, constraints
  - Section 3: Root Cause Analysis with H1/H2/H3 hypotheses, evidence for/against, falsification tests
  - Section 4: Remediation Plan with immediate mitigation, fix forward, prevention improvements
  - Section 5: Proof of Work with inputs consulted, commands executed, constraints
  - Section 6: Supporting Evidence with log excerpts, kubectl output in collapsible details

#### Scenario: Template format constraints

- **GIVEN** the report template
- **THEN** agents SHALL NOT add sections beyond the 7 defined
- **AND** agents SHALL NOT rename sections
- **AND** agents SHALL NOT rearrange section order
- **AND** agents SHALL preserve table structures where specified
- **AND** agents SHALL use `[FACT-n]` labels for observed facts
- **AND** agents SHALL use `[INF-n]` labels for derived inferences

---

### Requirement: Report Validation Script

The k8s-troubleshooter skill SHALL provide a validation script that checks report format compliance.

#### Scenario: Validation script location

- **GIVEN** the k8s-troubleshooter skill is installed
- **THEN** a validation script SHALL exist at `skills/k8s-troubleshooter/scripts/validate-report.sh`
- **AND** the script SHALL be executable (chmod +x)
- **AND** the script SHALL accept a report file path as first argument

#### Scenario: Section header validation

- **GIVEN** a validation script invocation with a report file
- **WHEN** the script checks for required sections
- **THEN** it SHALL verify all 7 section headers are present:
  - "## 0. Executive Triage Card"
  - "## 1. Problem Statement"
  - "## 2. Assessment & Findings"
  - "## 3. Root Cause Analysis"
  - "## 4. Remediation Plan"
  - "## 5. Proof of Work"
  - "## 6. Supporting Evidence"
- **AND** it SHALL exit with code 1 if any section is missing
- **AND** it SHALL output a clear error message identifying the missing section

#### Scenario: FACT/INF label validation

- **GIVEN** a validation script invocation with a report file
- **WHEN** the script checks for labeling
- **THEN** it SHALL search for `[FACT-n]` patterns using regex
- **AND** it SHALL search for `[INF-n]` patterns using regex
- **AND** it SHALL exit with code 2 (warning) if labels are missing
- **AND** it SHALL output a warning message about missing labels

#### Scenario: Required elements validation

- **GIVEN** a validation script invocation with a report file
- **WHEN** the script checks for required elements
- **THEN** it SHALL verify presence of "Most Dangerous Assumption" text
- **AND** it SHALL verify presence of "Falsification Test" text
- **AND** it SHALL verify presence of "### Commands Executed" heading
- **AND** it SHALL exit with code 2 (warning) if elements are missing
- **AND** it SHALL output warning messages for each missing element

#### Scenario: Exit code semantics

- **GIVEN** a validation script execution
- **THEN** it SHALL exit with code 0 if all validations pass
- **AND** it SHALL exit with code 1 if critical structural issues are found (missing sections)
- **AND** it SHALL exit with code 2 if warnings exist but structure is correct (missing labels/elements)
- **AND** it SHALL complete within 10 seconds (hook timeout)

---

### Requirement: Stop Hook Configuration

The k8s-troubleshooter skill SHALL configure a Stop hook that triggers validation when agents attempt to end sessions.

#### Scenario: Stop hook configuration file

- **GIVEN** the k8s-troubleshooter skill is installed
- **THEN** a hook configuration file SHALL exist at `skills/k8s-troubleshooter/skill-hooks.json`
- **AND** it SHALL contain valid JSON
- **AND** it SHALL define a "Stop" hook entry
- **AND** the Stop hook SHALL reference "./scripts/validate-report.sh"
- **AND** the Stop hook SHALL set timeout to 10000ms (10 seconds)

#### Scenario: Stop hook trigger

- **GIVEN** an agent using the k8s-troubleshooter skill
- **WHEN** the agent attempts to end the session
- **THEN** Claude Code SHALL trigger the Stop hook
- **AND** the validation script SHALL execute with default report path "output/report.md"
- **AND** the script output SHALL be captured
- **AND** the exit code SHALL determine session end behavior

#### Scenario: Validation failure feedback

- **GIVEN** a Stop hook validation that exits with code 1
- **THEN** the agent session SHALL be blocked from ending
- **AND** the validation error message SHALL be injected into agent context
- **AND** the agent SHALL see which sections or elements are missing
- **AND** the agent SHALL have opportunity to regenerate report and retry

#### Scenario: Validation warning feedback

- **GIVEN** a Stop hook validation that exits with code 2
- **THEN** the agent session SHALL end normally
- **AND** the validation warnings SHALL be logged to agent.log
- **AND** the report SHALL be uploaded to object storage as-is

#### Scenario: Validation success

- **GIVEN** a Stop hook validation that exits with code 0
- **THEN** the agent session SHALL end normally
- **AND** no validation messages SHALL appear in agent context
- **AND** the report SHALL be uploaded to object storage

---

### Requirement: Single Validation Per Stop Attempt

The validation system SHALL execute once per stop attempt without persisting state between agent turns.

#### Scenario: Fresh validation on each stop

- **GIVEN** an agent that fails validation on first stop attempt
- **WHEN** the agent regenerates the report and stops again
- **THEN** the validation script SHALL run fresh (no cached state)
- **AND** the validation SHALL evaluate the new report content
- **AND** the validation SHALL not "remember" previous failures

#### Scenario: No infinite loop protection needed

- **GIVEN** the Stop hook validation is stateless
- **THEN** the system SHALL NOT implement loop counters
- **AND** the system SHALL NOT track validation attempt numbers
- **AND** the agent SHALL be able to exit after passing validation (exit 0)
- **AND** the agent SHALL be able to exit with warnings (exit 2)
- **AND** the agent SHALL only be blocked on critical failures (exit 1)

#### Scenario: Timeout protection

- **GIVEN** a validation script that hangs or runs long
- **WHEN** the 10-second timeout expires
- **THEN** the hook SHALL be aborted
- **AND** the session SHALL end with a timeout warning
- **AND** the report SHALL be uploaded as-is

---

### Requirement: Base Prompt Template Reference

The nightcrier base triage prompt SHALL reference the k8s-troubleshooter template structure explicitly but minimally.

#### Scenario: Template reference in base prompt

- **GIVEN** the base-triage-prompt.md file
- **WHEN** an agent reads the "Required First Step" section
- **THEN** it SHALL reference the "Report Template - MANDATORY STRUCTURE" section in SKILL.md
- **AND** it SHALL instruct the agent to copy the template structure exactly
- **AND** it SHALL emphasize DO NOT add/remove/rename sections
- **AND** it SHALL list the 7 required section names for quick reference
- **AND** it SHALL NOT duplicate the full template (domain isolation)

#### Scenario: Minimal base prompt changes

- **GIVEN** the base-triage-prompt.md update
- **THEN** the changes SHALL be limited to 5-10 lines modified
- **AND** the update SHALL preserve existing skill path references
- **AND** the update SHALL keep domain-specific knowledge in the skill
- **AND** the prompt SHALL remain agent-agnostic (works for claude/codex/gemini/goose)

---

