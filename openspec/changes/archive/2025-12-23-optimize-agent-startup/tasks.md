# Tasks: Optimize Agent Startup

## Phase 1: Configuration and Foundation

- [x] 1.1 Add skills configuration to existing config structure
  - Add `SkillsConfig` struct to `internal/config/config.go`
  - Add `skills.cache_dir` field (default: "./agent-home/skills")
  - Add `skills.disable_triage_preload` field (default: false)
  - Update config loading to support SKILLS_CACHE_DIR env var override
  - Update example config files

- [x] 1.2 Add automatic skills caching to Go
  - Created `internal/skills/cache.go` with `EnsureSkillsCached()`
  - Automatically clones k8s4agents from GitHub on first run
  - Integrated into `cmd/nightcrier/main.go` startup sequence
  - Non-fatal: continues if git clone fails

- [x] 1.3 Add `preload_incident_context()` function to `agent-container/runners/common.sh`
  - Read incident.json and wrap in `<incident>` tags
  - Read incident_cluster_permissions.json and wrap in `<permissions>` tags
  - Return formatted context string

- [x] 1.4 Add K8s triage execution to `preload_incident_context()`
  - Check if K8s triage script exists at: `${skills_cache_dir}/k8s-troubleshooter/scripts/incident_triage.sh`
  - Execute with 30s timeout when `disable_triage` is false
  - Wrap output in `<initial_triage_report>` tags
  - Handle execution failures gracefully (log warning, agent will run triage itself)

- [x] 1.5 Integrate preloading into `run-agent.sh`
  - Read SKILLS_CACHE_DIR from environment (passed from Go config)
  - Call `preload_incident_context()` before building agent command
  - Store result in `PRELOADED_CONTEXT` variable
  - Export to agent runners

- [x] 1.6 Add context size monitoring
  - Count tokens in preloaded context (estimate: 4 chars ≈ 1 token)
  - Log warning if context exceeds 8,000 tokens
  - Implement truncation strategy if needed (initial_triage_report output first)

## Phase 2: Agent Runner Updates

- [x] 2.1 Update `runners/claude.sh` to inject preloaded context
  - Add logic to read PRELOADED_CONTEXT from environment
  - Inject via --append-system-prompt flag (after system prompt file)
  - Escape single quotes properly

- [x] 2.2 Update `runners/codex.sh` to inject preloaded context
  - Prepend context to user prompt with proper escaping

- [x] 2.3 Update `runners/gemini.sh` to inject preloaded context
  - Prepend context to user prompt with proper escaping

- [x] 2.4 Update `runners/goose.sh` to inject preloaded context
  - Prepend context to user prompt with proper escaping

## Phase 3: Bug Fixes and Refinements

- [x] 3.1 Fix skill path mismatch
  - Updated path in common.sh to include nested skills/ directory
  - Fixed: k8s-troubleshooter/scripts → k8s-troubleshooter/skills/k8s-troubleshooter/scripts

- [x] 3.2 Wire environment variables through executor
  - Added SkillsCacheDir and DisableTriagePreload to ExecutorConfig
  - Pass SKILLS_DIR and DISABLE_TRIAGE_PRELOAD to run-agent.sh
  - Fixed in internal/agent/executor.go and cmd/nightcrier/main.go

- [x] 3.3 Rename skill directory to match repo name
  - Changed K8sSkillName from "k8s-troubleshooter" to "k8s4agents"
  - Updated paths in common.sh to use k8s4agents
  - Cleaned up old cache directory

- [x] 3.4 Fix prompt-sent.md audit trail accuracy
  - Added append_preloaded_context_to_audit() function to common.sh
  - Append preloaded context to prompt-sent.md after preloading completes
  - Ensures audit trail shows actual context sent to agent

- [x] 3.5 Improve context tag clarity
  - Renamed `<permissions>` to `<kubernetes_cluster_access_permissions>`
  - Updated in agent-container/runners/common.sh

- [x] 3.6 Update system prompt for skill compliance
  - Removed incident.json file reference (data is preloaded)
  - Added explicit instruction to READ skill file first
  - Listed all 7 required report sections with mandatory language
  - Fixed agent not following skill template format

## Phase 4: Testing and Validation

- [x] 4.1 Initial testing with CrashLoopBackOff incident
  - Verified preloading works (visible in docker command logs)
  - Confirmed context includes incident, permissions, and triage report
  - Validated audit trail accuracy with prompt-sent.md updates

- [ ] 4.2 Create comparison test script
  - Run crashloopbackoff test with preloading OFF (baseline)
  - Run crashloopbackoff test with preloading ON
  - Compare: time to completion, token usage
  - Generate comparison report

- [ ] 4.3 Test all four agent CLIs
  - Claude with context preloading
  - Codex with context preloading
  - Gemini with context preloading
  - Goose with context preloading
  - Verify all receive preloaded context correctly

- [ ] 4.4 Performance benchmarking
  - Measure time savings: target 30-50% reduction
  - Measure token savings: target 20-30% reduction
  - Verify investigation quality maintained

## Phase 5: Documentation and Setup

- [x] 5.1 Update `.gitignore`
  - Add `/agent-home/` to gitignore

- [ ] 5.2 Update agent-container README
  - Document preloading behavior
  - Explain context structure (<incident>, <kubernetes_cluster_access_permissions>, <initial_triage_report> tags)
  - Document automatic skills caching
  - Note graceful degradation when triage unavailable

- [ ] 5.3 Update main README or setup docs
  - Document automatic skills caching on startup
  - Document configuration options (skills.cache_dir, skills.disable_triage_preload)

## Validation Criteria

Each task is considered complete when:
- Code changes committed and syntax-validated
- Manual testing confirms expected behavior
- Documentation updated to reflect changes

## Dependencies

- Task 1.4 depends on 1.1-1.3 (config and preloading functions must exist)
- Phase 2 depends on Phase 1 completion
- Phase 3 depends on Phase 2 completion
- Phase 4 can proceed in parallel with testing
- Report format validation handled by skill (see k8s4agents `add-standardized-report-format`)
- System prompt changes handled separately (see `generalize-triage-system-prompt`)

## Notes

Report structure template and validation have been moved to the skill layer:
- Report format defined in k8s-troubleshooter SKILL.md (see k8s4agents `add-standardized-report-format`)
- This proposal focuses solely on context preloading optimization
