# Tasks: Optimize Agent Startup

## Phase 1: Context Preloading Foundation

- [ ] 1.1 Add `preload_incident_context()` function to `agent-container/runners/common.sh`
  - Read incident.json and wrap in `<incident>` tags
  - Read incident_cluster_permissions.json and wrap in `<permissions>` tags
  - Return formatted context string

- [ ] 1.2 Add skill triage execution to `preload_incident_context()`
  - Check if skill's triage script is available (e.g., incident_triage.sh)
  - Execute with 30s timeout
  - Wrap output in `<triage>` tags
  - Handle execution failures gracefully (log warning, continue without triage)

- [ ] 1.3 Integrate preloading into `run-agent.sh`
  - Call `preload_incident_context()` before building agent command
  - Store result in `PRELOADED_CONTEXT` variable
  - Insert between system prompt and user prompt

- [ ] 1.4 Add context size monitoring
  - Count tokens in preloaded context (estimate: 4 chars ≈ 1 token)
  - Log warning if context exceeds 8,000 tokens
  - Implement truncation strategy if needed (triage output first)

## Phase 2: Testing and Validation

- [ ] 2.1 Create comparison test script
  - Run crashloopbackoff test with preloading OFF (baseline)
  - Run crashloopbackoff test with preloading ON
  - Compare: time to completion, token usage
  - Generate comparison report

- [ ] 2.2 Test all three baseline agents
  - Claude with context preloading
  - Codex with context preloading
  - Gemini with context preloading
  - Verify all receive preloaded context correctly

- [ ] 2.3 Performance benchmarking
  - Measure time savings: target 30-50% reduction
  - Measure token savings: target 20-30% reduction
  - Verify investigation quality maintained

## Phase 3: Documentation

- [ ] 3.1 Update agent-container README
  - Document preloading behavior
  - Explain context structure (<incident>, <permissions>, <triage> tags)
  - Note graceful degradation when triage unavailable

- [ ] 3.2 Update SKILL.md integration notes
  - Document that skill's triage script is called for preloading
  - Note expected output format

## Validation Criteria

Each task is considered complete when:
- Code changes committed and syntax-validated
- Manual testing confirms expected behavior
- Documentation updated to reflect changes

## Dependencies

- Task 1.3 depends on 1.1-1.2 (preloading functions must exist)
- Phase 2 depends on Phase 1 completion
- Report format validation handled by skill (see k8s4agents `add-standardized-report-format`)
- System prompt changes handled separately (see `generalize-triage-system-prompt`)

## Notes

Report structure template and validation have been moved to the skill layer:
- Report format defined in k8s-troubleshooter SKILL.md (see k8s4agents `add-standardized-report-format`)
- This proposal focuses solely on context preloading optimization
