# Tasks: Add Tool Enforcement Tests

## Phase 1: Test Infrastructure

- [ ] 1.1 Create `tests/tool-enforcement/` directory structure
- [ ] 1.2 Create `run-tool-enforcement-tests.sh` orchestrator script
- [ ] 1.3 Add shared test utilities in `tests/lib/tool-enforcement-helpers.sh`
- [ ] 1.4 Create test result JSON schema for structured output

## Phase 2: Positive Tool Tests (Allowed Tools Work)

- [ ] 2.1 Create test case: Read tool can read files when allowed
- [ ] 2.2 Create test case: Grep tool can search files when allowed
- [ ] 2.3 Create test case: Glob tool can find files when allowed
- [ ] 2.4 Create test case: Bash tool can execute commands when allowed
- [ ] 2.5 Create test case: Write tool can create files when allowed
- [ ] 2.6 Create test case: Skill tool can invoke skills when allowed

## Phase 3: Negative Tool Tests (Disallowed Tools Blocked)

- [ ] 3.1 Create test case: Edit tool blocked when not in allowedTools
- [ ] 3.2 Create test case: Write tool blocked when not in allowedTools
- [ ] 3.3 Create test case: Bash tool blocked when not in allowedTools
- [ ] 3.4 Create test case: Task tool blocked when not in allowedTools
- [ ] 3.5 Create test validation: No side effects from blocked tool attempts

## Phase 4: Boundary Tests

- [ ] 4.1 Create test case: Empty allowedTools list blocks all tools
- [ ] 4.2 Create test case: Invalid tool name in list is handled gracefully
- [ ] 4.3 Create test case: Case sensitivity of tool names
- [ ] 4.4 Create test case: Whitespace handling in tool list

## Phase 5: Integration

- [ ] 5.1 Add `--tool-enforcement` mode to `run-live-test.sh`
- [ ] 5.2 Create config template for tool enforcement testing
- [ ] 5.3 Add tool enforcement results to test reports
- [ ] 5.4 Document tool enforcement tests in `tests/README.md`

## Phase 6: CI/CD Integration

- [ ] 6.1 Add tool enforcement tests to release checklist
- [ ] 6.2 Create script for running tool enforcement tests in CI
- [ ] 6.3 Define pass/fail criteria for release gating

## Validation Checklist

- [ ] All positive tests pass with Claude CLI
- [ ] All negative tests pass with Claude CLI
- [ ] Test results include actionable failure details
- [ ] Tests complete within reasonable time (< 5 min per test)
- [ ] Documentation updated
