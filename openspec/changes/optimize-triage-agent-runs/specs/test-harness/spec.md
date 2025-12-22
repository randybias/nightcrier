# test-harness Spec Delta

## ADDED Requirements

### Requirement: Parallel Test Execution
The test harness SHALL support running multiple agents in parallel for comparison.

#### Scenario: Parallel baseline comparison
- **WHEN** parallel test execution is requested
- **THEN** multiple agents are launched simultaneously against the same test scenario
- **AND** each agent writes to a separate log directory
- **AND** all agent executions complete before generating comparison report
- **AND** test harness waits for all processes to finish

#### Scenario: Comparison report generation
- **WHEN** parallel tests complete
- **THEN** a comparison report is generated showing relative performance
- **AND** the report includes time to completion for each agent
- **AND** the report includes investigation quality metrics for each agent
- **AND** the report highlights significant differences between agents
- **AND** the report is available in both markdown and JSON formats

### Requirement: Reliability Testing
The test harness SHALL support reliability validation via consecutive test runs.

#### Scenario: N-iteration reliability test
- **WHEN** reliability testing is requested
- **THEN** the same agent and test scenario are executed N consecutive times
- **AND** success/failure status is recorded for each iteration
- **AND** all artifacts are preserved with iteration-specific names
- **AND** execution continues after individual failures

#### Scenario: Reliability report generation
- **WHEN** N-iteration test completes
- **THEN** a reliability report is generated with success/failure rate
- **AND** the report includes failure patterns (intermittent vs consistent)
- **AND** the report lists which iterations failed and why
- **AND** the report includes timing statistics (mean, min, max, stddev)

### Requirement: Quality Metrics Collection
The test harness SHALL extract and compare investigation quality metrics.

#### Scenario: Quality metric extraction
- **WHEN** an agent investigation completes
- **THEN** root cause identification is extracted from the report
- **AND** confidence level is extracted from the report
- **AND** evidence completeness is assessed
- **AND** metrics are stored in structured JSON format

#### Scenario: Quality metric comparison
- **WHEN** comparing multiple agent results
- **THEN** quality metrics are compared side-by-side
- **AND** agents with higher confidence are highlighted
- **AND** agents with more complete evidence are highlighted
- **AND** agents with incorrect root cause are flagged

### Requirement: Performance Metrics Collection
The test harness SHALL measure and compare agent performance.

#### Scenario: Performance metric collection
- **WHEN** an agent investigation executes
- **THEN** wall-clock time is measured from start to completion
- **AND** token usage is estimated from logs if available
- **AND** API call latency is tracked if possible
- **AND** metrics are stored in structured format

#### Scenario: Performance comparison
- **WHEN** comparing multiple agent results
- **THEN** performance metrics are compared side-by-side
- **AND** fastest agent is highlighted
- **AND** token efficiency is compared if available
- **AND** performance outliers are flagged

## MODIFIED Requirements

### Requirement: Test Orchestration
The test harness SHALL orchestrate end-to-end validation of triage agents.

#### Scenario: Pre-flight validation (NEW)
- **WHEN** a test is started
- **THEN** model configuration is validated before execution
- **AND** API keys are validated before execution
- **AND** API quota status is checked with warning if low
- **AND** test fails fast with clear error if validation fails
- **AND** validation can be skipped via --skip-validation flag

#### Scenario: Timeout enforcement (NEW)
- **WHEN** a test is started
- **THEN** a timeout is configured (default or user-specified)
- **AND** test execution is bounded by timeout value
- **AND** timeout expiry triggers cleanup and partial result capture
- **AND** timeout value is configurable via CLI argument

## Cross-References

- **Depends on**: agent-container - model validation, error handling, timeout support
- **Impacts**: None (test-harness is leaf capability)
- **Related to**: agent-logging - metrics extracted from structured logs
