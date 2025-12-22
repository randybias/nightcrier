# Tasks: Add Live Test Harness

## Phase 1: Foundation (test infrastructure)

1. Create `tests/` directory structure with subdirectories
   - Validation: Directory structure matches design

2. Create `.gitignore` in `tests/` to exclude logs and generated configs
   - Validation: `git status` doesn't show logs/

3. Create config templates for all three agents (claude, codex, gemini)
   - Validation: Templates exist and have placeholders for all secrets

4. Document secrets file format in `tests/README.md`
   - Validation: README explains what goes in `~/dev-secrets/nightcrier-secrets.env`

## Phase 2: Configuration Generation

5. Create `lib/config-generator.sh` that sources secrets and generates configs
   - Validation: Script produces valid YAML configs from templates

6. Test config generation with sample secrets
   - Validation: Generated configs pass YAML validation

## Phase 3: Failure Induction

7. Create `failure-induction/01_induce_failure_crashloopbackoff.sh`
   - Validation: Script creates crashlooping pod with `start`, deletes with `stop`
   - Validation: Script outputs `TIMEOUT=<seconds>` on start

8. Test failure induction manually
   - Validation: Pod crashes repeatedly, MCP server emits fault event
   - Validation: Timeout value is captured and usable

## Phase 4: Log Monitoring

9. Create `lib/log-monitor.sh` with pattern-matching functions
   - Validation: Functions detect subscription, agent start, agent completion

10. Test log monitoring against sample nightcrier output
    - Validation: All detection functions work correctly

## Phase 5: Orchestration

11. Create `run-live-test.sh` main script skeleton
    - Validation: Script accepts arguments and validates them

12. Implement test ID generation and log directory creation
    - Validation: Unique test IDs generated, log directories created under tests/logs/<testid>/

13. Implement nightcrier lifecycle (start/stop)
    - Validation: Nightcrier starts in background, stops on script exit
    - Validation: Logs written to tests/logs/<testid>/nightcrier.log

14. Integrate config generation into orchestration
    - Validation: Script generates config before starting nightcrier

15. Integrate failure induction into orchestration
    - Validation: Script induces failure and cleans up afterwards
    - Validation: Script captures timeout value from induction script

16. Integrate log monitoring into orchestration
    - Validation: Script waits for expected log patterns
    - Validation: Script uses timeout value from induction script

## Phase 6: Reporting

17. Create `lib/report-generator.sh` that extracts test results
    - Validation: Generates human-readable report from logs and artifacts

18. Integrate report generation into orchestration
    - Validation: Report displayed after test completes (human-readable by default)

19. Add `--json` flag for JSON output mode
    - Validation: `--json` flag produces JSON to stdout
    - Validation: JSON always written to logs/<testid>/report.json

## Phase 7: Testing and Documentation

19. Run end-to-end test with Claude agent and crashloopbackoff
    - Validation: Complete test cycle passes, report shows success

20. Run tests with Codex and Gemini agents
    - Validation: All three agents work with test harness

21. Test DEBUG mode flag
    - Validation: agent_debug setting correctly set in generated config

22. Test error conditions (timeout, failure cleanup)
    - Validation: Script handles errors gracefully

23. Update `tests/README.md` with usage instructions
    - Validation: Can run test following README alone

24. Add example secrets file template to README
    - Validation: README shows exact format for secrets file

## Dependencies

- Tasks 5-6 depend on tasks 1-4 (need directory structure and templates)
- Tasks 11-15 depend on tasks 5-10 (orchestration needs all components)
- Tasks 16-18 depend on task 15 (reporting needs test execution)
- Tasks 19-24 depend on all previous tasks (validation phase)

## Parallelizable Work

- Tasks 3-4 (templates and docs) can be done in parallel
- Tasks 7-8 (failure induction) can be done in parallel with tasks 9-10 (monitoring)
- Tasks 20-21 (testing other agents and DEBUG mode) can be done in parallel
