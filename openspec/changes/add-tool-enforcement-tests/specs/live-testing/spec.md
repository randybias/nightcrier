# live-testing Spec Delta

## ADDED Requirements

### Requirement: Tool Enforcement Validation

The test harness SHALL validate that agent tool restrictions are enforced.

#### Scenario: Run tool enforcement test suite

**Given** the test harness is invoked with `--tool-enforcement` flag
**And** API keys are available for the specified agent
**When** the tool enforcement tests execute
**Then** positive tests SHALL verify allowed tools function correctly
**And** negative tests SHALL verify disallowed tools are blocked
**And** a structured test report SHALL be generated

#### Scenario: Positive tool enforcement test

**Given** an agent is configured with `allowedTools: "Read,Grep"`
**And** a test prompt requests use of the Read tool
**When** the agent executes
**Then** the agent SHALL successfully use the Read tool
**And** the test SHALL pass with exit code 0
**And** the expected output artifact SHALL be created

#### Scenario: Negative tool enforcement test

**Given** an agent is configured with `allowedTools: "Read,Grep"` (without Edit)
**And** a test prompt requests use of the Edit tool
**When** the agent executes
**Then** the agent SHALL NOT use the Edit tool
**And** the test SHALL verify no unauthorized file modifications occurred
**And** the agent log SHALL indicate the tool was blocked or refused

#### Scenario: Tool enforcement test failure handling

**Given** a tool enforcement test fails
**When** the test report is generated
**Then** the failure SHALL include the test name and category (positive/negative)
**And** the failure SHALL include which tool was being tested
**And** the failure SHALL include details about what went wrong
**And** the overall test suite SHALL report failure status

### Requirement: Tool Enforcement Test Isolation

The test harness SHALL isolate tool enforcement tests from production artifacts.

#### Scenario: Test workspace isolation

**Given** a tool enforcement test is running
**When** the test creates or modifies files
**Then** all artifacts SHALL be contained within `tests/tool-enforcement/results/`
**And** no artifacts SHALL be created in the `incidents/` directory
**And** the test workspace SHALL be cleaned up after completion

#### Scenario: Test fixture preparation

**Given** a tool enforcement test requires input files
**When** the test setup runs
**Then** fixture files SHALL be created in a temporary directory
**And** the fixture path SHALL be passed to the agent
**And** fixtures SHALL be cleaned up after test completion

### Requirement: Tool Enforcement Reporting

The test harness SHALL report tool enforcement results in a structured format.

#### Scenario: Tool enforcement JSON report

**Given** tool enforcement tests have completed
**When** the report is generated
**Then** a JSON report SHALL be written to `tests/tool-enforcement/results/report-<timestamp>.json`
**And** the report SHALL include test suite metadata (timestamp, agent, version)
**And** the report SHALL include summary counts (total, passed, failed, skipped)
**And** the report SHALL include per-test details (name, category, tool, status, duration)

#### Scenario: Tool enforcement human-readable summary

**Given** tool enforcement tests have completed
**When** the report is displayed to stdout
**Then** a human-readable summary SHALL show pass/fail status
**And** failed tests SHALL be listed with brief failure reasons
**And** the overall result (PASSED/FAILED) SHALL be clearly indicated

### Requirement: CI/CD Integration

The tool enforcement tests SHALL support CI/CD pipeline integration.

#### Scenario: CI/CD exit codes

**Given** tool enforcement tests are run in a CI/CD pipeline
**When** all tests pass
**Then** the script SHALL exit with code 0

**Given** tool enforcement tests are run in a CI/CD pipeline
**When** any test fails
**Then** the script SHALL exit with code 1
**And** the failed test details SHALL be output to stderr

#### Scenario: Release validation hook

**Given** a release is being prepared
**When** tool enforcement tests are invoked
**Then** the tests SHALL run against the release candidate
**And** failing tests SHALL block the release (when integrated)
**And** test results SHALL be available in release notes or artifacts
