# Optimize Triage Agent Runs

## Problem Statement

The live test harness lacks robust validation and error handling for agent execution:

1. **No model validation**: Tests don't validate model names before execution, leading to late failures
2. **Poor API error handling**: Quota exhaustion and API errors produce generic messages without resolution steps
3. **No timeout handling**: Slow or hung agent responses can block tests indefinitely
4. **Limited integration testing**: No systematic way to compare agent performance or validate reliability

This results in:
- Wasted time on invalid model names or exhausted quotas
- Difficult debugging when API issues occur
- Inconsistent test execution across agents
- Lack of comparative performance data

## Proposed Solution

### 1. Model Configuration Validation

Add pre-flight validation for all agents:

1. **Model name validation**: Verify model identifiers are valid before execution
2. **API quota checking**: Query API status to detect exhausted quotas early
3. **Pre-flight validation script**: Shared validation logic for all agents (Claude, Codex, Gemini)
4. **Clear error messages**: Provide resolution steps when validation fails

This applies to all agents equally and prevents wasted execution time.

### 2. Error Handling Improvements

Enhance agent runner error handling:

1. **API quota detection**: Parse provider-specific error responses for quota exhaustion
2. **Timeout handling**: Set reasonable timeouts for API calls and handle gracefully
3. **Enhanced error messages**: Include resolution steps in all error outputs
4. **Exit codes**: Use distinct exit codes for different failure types (quota, timeout, auth, etc.)

This provides better diagnostics and clearer action items when tests fail.

### 3. Integration Testing Framework

Add systematic comparison and validation:

1. **Parallel baseline tests**: Run multiple agents against same scenario for comparison
2. **Quality metrics**: Validate investigation quality (root cause identification, confidence levels)
3. **Reliability testing**: Run consecutive tests to identify intermittent failures
4. **Performance metrics**: Measure and compare time to completion, token usage

This enables data-driven agent selection and reliability monitoring.

## Success Criteria

1. **Pre-flight validation**: 100% of invalid configurations caught before agent execution
2. **Error clarity**: All API errors include resolution steps in messages
3. **Timeout handling**: No test runs indefinitely; all have bounded execution time
4. **Comparative data**: Ability to run parallel agent tests and generate comparison reports

## Scope

**In Scope:**
- Pre-flight model validation for Claude, Codex, Gemini
- API quota detection and reporting (all agents)
- Timeout handling in test harness
- Enhanced error messages with resolution steps
- Parallel test execution script
- Reliability testing framework (N consecutive runs)
- Performance comparison reporting

**Out of Scope:**
- Changes to agent CLI tools themselves
- Changes to nightcrier core logic
- Changes to MCP server
- Agent-specific fixes (those belong in agent-specific proposals)

## Dependencies

- Requires live test harness (already exists via add-live-test-harness)
- Requires agent runners (already exists)
- May require API client libraries for quota checking

## Risks and Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| API quota checks unreliable | False positives/negatives | Make quota checks advisory, not blocking |
| Timeout too short | Valid tests killed early | Make timeouts configurable per test type |
| Parallel tests overwhelm API | Rate limiting or quota exhaustion | Stagger test starts, add configurable delays |
| Comparison metrics misleading | Poor agent selection | Document metric limitations, require manual review |

## Open Questions

1. **What timeout values are reasonable?** Need to measure typical execution times for each agent
2. **How to check API quotas reliably?** Provider APIs may not expose this information
3. **What constitutes a reliability failure?** Define acceptable failure rate thresholds
