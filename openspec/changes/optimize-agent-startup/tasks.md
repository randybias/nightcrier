# Tasks: Optimize Agent Startup

## Phase 1: Context Preloading Foundation

- [ ] 1.1 Add `preload_incident_context()` function to `agent-container/runners/common.sh`
  - Read incident.json and wrap in `<incident>` tags
  - Read incident_cluster_permissions.json and wrap in `<permissions>` tags
  - Return formatted context string

- [ ] 1.2 Add triage execution to `preload_incident_context()`
  - Check if incident_triage.sh is available
  - Execute `--skip-dump` mode with 30s timeout
  - Wrap output in `<triage>` tags
  - Handle execution failures gracefully (log warning, continue)

- [ ] 1.3 Integrate preloading into `run-agent.sh`
  - Call `preload_incident_context()` before building agent command
  - Store result in `PRELOADED_CONTEXT` variable
  - Append to system prompt before user prompt

- [ ] 1.4 Add context size monitoring
  - Count tokens in preloaded context (estimate: 4 chars ≈ 1 token)
  - Log warning if context exceeds 8,000 tokens
  - Implement truncation strategy if needed (triage output first)

- [ ] 1.5 Test context preloading
  - Run agent with preloading against crashloopbackoff test
  - Verify incident.json content appears in agent context
  - Verify triage results appear in agent context
  - Measure time savings vs baseline

## Phase 2: Report Structure Template

- [ ] 2.1 Update `triage-system-prompt.md` with structured template
  - Add "Report Structure" section with four required sections
  - Specify exact markdown formatting (### headings, **bold** for labels)
  - Include example template
  - Emphasize Root Cause and Confidence Level requirements

- [ ] 2.2 Create `validate-report.sh` script
  - Implement `validate_report_structure()` function
  - Check for four required section headers
  - Verify Root Cause and Confidence Level presence
  - Return exit code 0 (valid) or 1 (invalid) with error message

- [ ] 2.3 Integrate validation into `run-agent.sh`
  - Call `validate-report.sh` after agent completion
  - Log validation results (pass/fail)
  - Initially run in warn-only mode (don't fail on structure issues)
  - Track validation pass rate in debug logs

- [ ] 2.4 Add report versioning
  - Add `<!-- Report Version: 2.0 -->` header to template
  - Update validation to check version header
  - Document version 1.0 (legacy) vs 2.0 (structured) formats

## Phase 3: Testing and Validation

- [ ] 3.1 Create comparison test script
  - Run crashloopbackoff test with preloading OFF (baseline)
  - Run crashloopbackoff test with preloading ON
  - Compare: time to completion, token usage, investigation quality
  - Generate comparison report

- [ ] 3.2 Test all three baseline agents
  - Claude with context preloading
  - Codex with context preloading
  - Gemini with context preloading
  - Verify all produce structured reports

- [ ] 3.3 Validate report structure compliance
  - Run 10 test incidents per agent
  - Check validation pass rate (target: 100%)
  - Review failures and adjust template if needed

- [ ] 3.4 Performance benchmarking
  - Measure time savings: target 30-50% reduction
  - Measure token savings: target 20-30% reduction
  - Verify investigation quality maintained (confidence levels ≥90%)

## Phase 4: Enforcement and Migration

- [ ] 4.1 Enable strict validation mode
  - Change validation from warn-only to error mode
  - Agent runs fail if report structure invalid
  - Update nightcrier to handle validation failures

- [ ] 4.2 Update documentation
  - Update agent-container README with preloading behavior
  - Document report structure requirements
  - Add troubleshooting guide for validation failures

- [ ] 4.3 Create migration guide
  - Document changes for report consumers
  - Provide parsing examples for both v1.0 and v2.0 formats
  - Set deprecation timeline for v1.0 support

## Validation Criteria

Each task is considered complete when:
- Code changes committed and syntax-validated
- Unit/integration tests pass (where applicable)
- Manual testing confirms expected behavior
- Documentation updated to reflect changes

## Dependencies

- Task 1.3 depends on 1.1-1.2 (preloading functions must exist)
- Task 2.3 depends on 2.2 (validation script must exist)
- Phase 3 depends on completion of Phases 1-2
- Phase 4 depends on Phase 3 validation results

## Parallelizable Work

- Phase 1 (context preloading) and Phase 2 (report structure) can be developed in parallel
- Testing tasks (3.1-3.4) can run concurrently once implementation complete
