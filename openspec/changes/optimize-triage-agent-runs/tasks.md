# Tasks: Optimize Triage Agent Runs

## Phase 1: Model Configuration Validation

- [ ] 1.1 Design model validation interface
  - Define validation function signature for all agents
  - Specify validation outputs (valid/invalid, error messages)
  - Document expected behavior for quota checks
  - Create validation contract for agent runners

- [ ] 1.2 Implement Claude model validation
  - Create `validate-claude-model.sh` script
  - Check ANTHROPIC_API_KEY validity
  - Validate model name against known Claude models
  - Query API for quota status (if possible)
  - Return clear error messages

- [ ] 1.3 Implement Codex model validation
  - Create `validate-codex-model.sh` script
  - Check OPENAI_API_KEY validity
  - Validate model name against known Codex models
  - Query API for quota status (if possible)
  - Return clear error messages

- [ ] 1.4 Implement Gemini model validation
  - Create `validate-gemini-model.sh` script
  - Check GEMINI_API_KEY/GOOGLE_API_KEY validity
  - Validate model name against known Gemini models
  - Query API for quota status (if possible)
  - Return clear error messages

- [ ] 1.5 Integrate validation into test harness
  - Call validation before agent execution in run-live-test.sh
  - Display validation results to user
  - Fail fast with clear message if validation fails
  - Add --skip-validation flag for advanced users

## Phase 2: Error Handling Improvements

- [ ] 2.1 Design error handling framework
  - Define standard error codes (quota=10, timeout=11, auth=12, etc.)
  - Create error message template with resolution steps
  - Document provider-specific error patterns
  - Design error logging format

- [ ] 2.2 Implement quota detection for all agents
  - Parse Claude API quota errors
  - Parse Codex API quota errors
  - Parse Gemini API quota errors
  - Map provider errors to standard codes
  - Generate resolution messages

- [ ] 2.3 Add timeout handling to test harness
  - Determine reasonable timeout values per agent
  - Implement timeout in run-live-test.sh
  - Handle timeout gracefully (partial results)
  - Log timeout events with context
  - Make timeout configurable via CLI arg

- [ ] 2.4 Enhance error messages across runners
  - Update claude.sh error messages
  - Update codex.sh error messages
  - Update gemini.sh error messages
  - Update goose.sh error messages
  - Add resolution steps to all errors

- [ ] 2.5 Test error handling paths
  - Simulate quota exhaustion for each agent
  - Test timeout scenarios
  - Verify error messages include resolution steps
  - Validate exit codes are distinct

## Phase 3: Integration Testing Framework

- [ ] 3.1 Create parallel test execution script
  - Create `run-parallel-tests.sh` script
  - Support running multiple agents simultaneously
  - Capture all outputs to separate log directories
  - Wait for all tests to complete
  - Generate comparison report

- [ ] 3.2 Implement quality metrics collection
  - Extract root cause from investigation reports
  - Extract confidence levels from reports
  - Extract evidence completeness metrics
  - Create metrics JSON output format
  - Store metrics with test run results

- [ ] 3.3 Create comparison report generator
  - Compare time to completion across agents
  - Compare investigation quality metrics
  - Compare token usage (if available)
  - Generate side-by-side report (markdown/JSON)
  - Highlight significant differences

- [ ] 3.4 Implement reliability testing framework
  - Create `run-reliability-test.sh` script
  - Run agent N consecutive times (default: 10)
  - Track success/failure rate
  - Identify intermittent failures
  - Generate reliability report with failure patterns

- [ ] 3.5 Add performance metrics
  - Measure wall-clock time for each agent
  - Track API call latency (if possible)
  - Estimate token usage from logs
  - Compare performance across agents
  - Store metrics in structured format

## Phase 4: Validation and Testing

- [ ] 4.1 Test model validation
  - Test with valid model names
  - Test with invalid model names
  - Test with expired API keys
  - Test with exhausted quotas
  - Verify error messages are clear

- [ ] 4.2 Test error handling
  - Trigger quota exhaustion scenarios
  - Trigger timeout scenarios
  - Trigger authentication failures
  - Verify exit codes are correct
  - Verify error messages include resolution steps

- [ ] 4.3 Test parallel execution
  - Run 2-agent parallel test
  - Run 3-agent parallel test
  - Verify no resource conflicts
  - Verify all logs captured correctly
  - Verify comparison report accurate

- [ ] 4.4 Test reliability framework
  - Run 10-iteration reliability test
  - Verify all results captured
  - Verify failure patterns identified
  - Verify report format correct
  - Test with intermittent failures

## Phase 5: Documentation

- [ ] 5.1 Document validation system
  - Explain model validation process
  - Document validation script usage
  - Provide examples of validation failures
  - Document --skip-validation flag

- [ ] 5.2 Document error handling
  - List all error codes and meanings
  - Document resolution steps for each error
  - Provide troubleshooting guide
  - Document timeout configuration

- [ ] 5.3 Document integration testing
  - Explain parallel test execution
  - Document comparison metrics
  - Explain reliability testing framework
  - Provide usage examples

- [ ] 5.4 Update test harness README
  - Add validation section
  - Add error handling section
  - Add integration testing section
  - Add performance comparison examples

## Validation Criteria

Each task is complete when:
- Changes implemented and tested
- Works for all agents (Claude, Codex, Gemini, Goose)
- No regressions in existing functionality
- Documentation updated
- Manual verification confirms expected behavior

## Dependencies

- Phase 2-5 can start after Phase 1 validation framework complete
- Phase 3 (integration testing) can run in parallel with Phase 2 (error handling)
- Phase 5 (documentation) can be drafted while other phases run

## Parallelizable Work

- Phase 1 tasks (1.2, 1.3, 1.4) can be implemented in parallel
- Phase 2 tasks (2.4 error message updates) can run in parallel
- Phase 3 tasks (3.2 metrics, 3.3 comparison) can develop in parallel
