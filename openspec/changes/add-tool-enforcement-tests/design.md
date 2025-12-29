# Design: Add Tool Enforcement Tests

## Overview

This document captures architectural decisions for the tool enforcement test suite.

## Decision: Test Execution Strategy

### Options Considered

1. **Mock-based testing**: Simulate agent responses without calling real APIs
2. **Dry-run testing**: Use agent `--dry-run` flags if available
3. **Live agent testing**: Run actual agent invocations with restricted tools

### Decision: Live Agent Testing

**Rationale**: Tool enforcement is implemented by the agent CLIs, not by Nightcrier. The only way to truly validate enforcement is to run real agents with restricted tool lists and verify behavior. Mock tests would only validate our assumptions, not actual CLI behavior.

**Trade-offs**:
- (+) Tests actual enforcement behavior
- (+) Catches regressions in agent CLI updates
- (-) Requires API keys and costs money
- (-) Tests take longer than unit tests
- (-) AI responses have some non-determinism

**Mitigations**:
- Use minimal prompts to reduce token usage
- Allow test retries for flaky failures
- Cache test results for development iteration

## Decision: Test Assertion Strategy

### Options Considered

1. **Exit code only**: Check agent exit code (0 = success, non-zero = failure)
2. **Log parsing**: Search agent logs for "tool blocked" messages
3. **Side effect validation**: Verify no unauthorized files/changes created

### Decision: Multi-layer Validation

Use all three approaches in combination:

```
Test Pass Criteria:
├── Positive Tests (tool should work)
│   ├── Exit code = 0
│   └── Expected output artifact exists
│
└── Negative Tests (tool should be blocked)
    ├── Exit code = non-zero OR agent completed without using tool
    ├── Log contains "not allowed" or similar
    └── No unauthorized side effects
```

**Rationale**: Different agent CLIs may handle tool restrictions differently. Claude may exit non-zero, while another CLI might just refuse the operation. Multi-layer validation catches all cases.

## Decision: Test Prompt Design

### Approach

Test prompts must be:
1. **Deterministic**: Avoid open-ended requests that could lead to different tool choices
2. **Direct**: Explicitly request a specific tool by name
3. **Minimal**: Use as few tokens as possible
4. **Observable**: Produce verifiable output or side effects

### Example Test Prompts

```markdown
# Positive: Test Read tool
"Use the Read tool to read /etc/hostname. Output only the hostname."

# Negative: Test blocked Edit tool
"Use the Edit tool to append 'test' to /tmp/testfile.txt"

# Boundary: Empty tool list
"List the contents of the current directory."
(with AGENT_ALLOWED_TOOLS="" - should fail completely)
```

## Decision: Directory Structure

```
tests/
├── tool-enforcement/
│   ├── run-tests.sh              # Main orchestrator
│   ├── test-cases/
│   │   ├── positive/
│   │   │   ├── test-read-allowed.sh
│   │   │   ├── test-grep-allowed.sh
│   │   │   └── ...
│   │   └── negative/
│   │       ├── test-edit-blocked.sh
│   │       ├── test-write-blocked.sh
│   │       └── ...
│   ├── fixtures/
│   │   └── test-files/           # Files for Read tests
│   └── results/
│       └── .gitignore
└── lib/
    └── tool-enforcement-helpers.sh
```

## Decision: Results Format

Test results will be output as JSON for CI/CD integration:

```json
{
  "suite": "tool-enforcement",
  "timestamp": "2025-12-27T12:00:00Z",
  "agent": "claude",
  "summary": {
    "total": 15,
    "passed": 14,
    "failed": 1,
    "skipped": 0
  },
  "tests": [
    {
      "name": "test-read-allowed",
      "category": "positive",
      "tool": "Read",
      "status": "passed",
      "duration_ms": 3500,
      "details": null
    },
    {
      "name": "test-edit-blocked",
      "category": "negative",
      "tool": "Edit",
      "status": "failed",
      "duration_ms": 4200,
      "details": "Edit tool was used despite not being in allowedTools"
    }
  ]
}
```

## Decision: Integration with Live Test Harness

### Options Considered

1. **Separate harness**: Independent test runner
2. **Sub-mode of existing**: Add `--tool-enforcement` flag to `run-live-test.sh`
3. **Parallel execution**: Run alongside failure-induction tests

### Decision: Sub-mode of Existing Harness

**Rationale**: Reuses existing infrastructure for config generation, secret management, and reporting. Reduces code duplication.

**Integration**:
```bash
# Existing usage (failure-induction tests)
./tests/run-live-test.sh claude crashloopbackoff --debug

# New usage (tool enforcement tests)
./tests/run-live-test.sh claude --tool-enforcement --debug
```

## Open Questions

1. **Q**: Should we test tool enforcement for all agent CLIs or just Claude?
   **A**: Start with Claude (primary agent), add others if they support tool restrictions.

2. **Q**: How often should tool enforcement tests run?
   **A**: On every release; optionally nightly for regression detection.

3. **Q**: What's the cost budget for running these tests?
   **A**: Defer to implementation - minimize tokens per test, allow skip flag for cost-sensitive runs.
