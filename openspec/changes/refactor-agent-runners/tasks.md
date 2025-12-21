# Tasks: Refactor Agent Runners

## 1. Setup and Common Infrastructure
- [x] 1.1 Create `agent-container/runners/` directory
- [x] 1.2 Create `runners/common.sh` with shared functions (logging, validation, path helpers)
- [x] 1.3 Define standardized environment variable contract in `common.sh` header comments

## 2. Claude Sub-Runner (Reference Implementation)
- [x] 2.1 Create `runners/claude.sh` - extract Claude CLI command building from run-agent.sh
- [x] 2.2 Create `runners/claude-post.sh` - move `post_run_extract_claude_session` and `extract_agent_commands` logic
- [x] 2.3 Test Claude sub-runner produces identical command to current implementation
- [x] 2.4 Test Claude post-run extracts session and commands correctly

## 3. Codex Sub-Runner
- [x] 3.1 Create `runners/codex.sh` - extract Codex CLI command building (login, exec, flags)
- [x] 3.2 Research Codex session storage location (`.codex/` or alternative)
- [x] 3.3 Create `runners/codex-post.sh` - implement session extraction for Codex
- [x] 3.4 Implement Codex command extraction (if session format allows)
- [x] 3.5 Test Codex sub-runner command generation validated

## 4. Gemini Sub-Runner
- [x] 4.1 Create `runners/gemini.sh` - extract Gemini CLI command building
- [x] 4.2 Research Gemini session/logging capabilities
- [x] 4.3 Create `runners/gemini-post.sh` - stub or implement session extraction
- [x] 4.4 Test Gemini sub-runner command generation validated

## 5. Goose Sub-Runner
- [ ] 5.1 Create `runners/goose.sh` - extract Goose CLI command building (DEFERRED per user request)
- [ ] 5.2 Research Goose session storage (varies by provider)
- [ ] 5.3 Create `runners/goose-post.sh` - stub or implement session extraction
- [ ] 5.4 Test Goose sub-runner (if configured)

## 6. Refactor Main Orchestrator
- [x] 6.1 Refactor `run-agent.sh` to source `common.sh` for shared functions
- [x] 6.2 Replace `build_agent_command()` with sub-runner dispatch
- [x] 6.3 Replace hardcoded `post_run_extract_claude_session` with agent-agnostic dispatcher
- [x] 6.4 Clean up legacy agent-specific code from main script
- [x] 6.5 Update help text to reflect new architecture

## 7. Integration Testing
- [x] 7.1 Syntax validation of all scripts completed
- [x] 7.2 Command generation tests for Claude, Codex, Gemini completed
- [x] 7.3 End-to-end testing with live agents completed (Claude, Codex, Gemini)
- [x] 7.4 Verify DEBUG mode produces expected artifacts (session archives, command logs)

## 8. Documentation
- [x] 8.1 Update `agent-container/README.md` with new architecture
- [x] 8.2 Add inline documentation to each sub-runner (already included in scripts)
- [x] 8.3 Document how to add a new agent runner

## 9. Gemini Integration (Added)
- [x] 9.1 Research Gemini CLI context file system (GEMINI.md)
- [x] 9.2 Add GEMINI.md creation to Dockerfile with k8s-troubleshooter reference
- [x] 9.3 Fix gemini-post.sh session file path (session-*.json in chats/)
- [x] 9.4 Test Gemini reads GEMINI.md and has skill knowledge
- [x] 9.5 Verify Gemini DEBUG mode session extraction

## Dependencies
- Tasks 2.x must complete before Task 6.x (Claude is reference implementation) ✓
- Tasks 3.x, 4.x, 5.x can run in parallel after 1.x and 2.x ✓
- Task 7.x requires all sub-runners to be created ✓
