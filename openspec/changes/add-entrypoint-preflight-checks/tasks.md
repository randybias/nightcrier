# Implementation Tasks

## Phase 1: Core Preflight Function (Foundation)

### Task 1.1: Create preflight_check() function structure
- [ ] Add `preflight_check()` function in entrypoint.sh before `main()`
- [ ] Document function purpose, globals, and arguments
- [ ] Add standardized error message format
- [ ] Create helper function `fail_preflight()` for consistent error reporting

**Validation**: Function stub exists with proper documentation

### Task 1.2: Move environment variable validation into preflight
- [ ] Move `validate_env()` call into `preflight_check()`
- [ ] Add descriptive banner: "Running preflight checks..."
- [ ] Add success message after all checks pass
- [ ] Update `main()` to call `preflight_check()` immediately after startup banner

**Validation**: Existing env var validation runs earlier in execution flow

## Phase 2: File Existence Checks

### Task 2.1: Validate required mounted files
- [ ] Check `/home/agent/incident.json` exists and is readable
- [ ] Check `/home/agent/incident_cluster_permissions.json` exists and is readable
- [ ] Check `/home/agent/base-triage-prompt.md` exists and is readable
- [ ] Report which file is missing in error message

**Validation**: Jobs fail fast when ConfigMap files aren't mounted correctly

### Task 2.2: Validate kubeconfig presence
- [ ] Check `/home/agent/.kube/config` exists
- [ ] Check file is readable (not just exists)
- [ ] Report clear message if kubeconfig is missing

**Validation**: Jobs fail fast when kubeconfig secret isn't mounted

## Phase 3: Runtime Validation

### Task 3.1: Test kubectl connectivity
- [ ] Run `kubectl version --client -o json` to verify kubectl binary works
- [ ] Run `kubectl auth can-i get pods --all-namespaces` to test cluster connectivity
- [ ] Capture and report kubectl error messages if connectivity fails
- [ ] Add timeout (5 seconds) to prevent hanging on unreachable clusters

**Validation**: Jobs fail fast when kubeconfig contains expired tokens

### Task 3.2: Validate API key presence
- [ ] Check at least one API key environment variable is set (ANTHROPIC_API_KEY, OPENAI_API_KEY, GEMINI_API_KEY)
- [ ] For selected agent (AGENT_CLI), verify corresponding API key is present
  - claude → ANTHROPIC_API_KEY
  - codex → OPENAI_API_KEY
  - gemini → GEMINI_API_KEY
  - goose → depends on LLM_PROVIDER
- [ ] Report which API key is missing

**Validation**: Jobs fail fast when agent-specific API key is missing

### Task 3.3: Test output directory writability
- [ ] Create test file in `/home/agent/output/`
- [ ] Write content to test file
- [ ] Delete test file
- [ ] Report error if filesystem is read-only or full

**Validation**: Jobs fail fast when output directory isn't writable

## Phase 4: Logging and Observability

### Task 4.1: Add structured preflight logging
- [ ] Print "Preflight check: <name>" for each check
- [ ] Print "✓ <name> passed" or "✗ <name> failed: <reason>" after each check
- [ ] Summarize total checks run and failures at the end
- [ ] Use consistent formatting for easy log parsing

**Validation**: Preflight failures are easy to identify in logs

### Task 4.2: Add preflight duration tracking
- [ ] Record start time at beginning of `preflight_check()`
- [ ] Print "Preflight checks completed in X.XXs" after all checks pass
- [ ] Include in failure messages how long before failure occurred

**Validation**: Can measure preflight check performance impact

## Phase 5: Testing and Documentation

### Task 5.1: Add preflight check testing scenarios
- [ ] Document test scenarios in `tests/preflight/` directory
- [ ] Test missing kubeconfig
- [ ] Test expired kubeconfig token
- [ ] Test missing incident.json
- [ ] Test missing API keys
- [ ] Test read-only output directory
- [ ] Test all checks passing (baseline)

**Validation**: All failure scenarios are reproducible and validated

### Task 5.2: Update entrypoint.sh documentation
- [ ] Update file header comments to mention preflight checks
- [ ] Document new environment variables (if any)
- [ ] Add troubleshooting section for common preflight failures
- [ ] Update example kubectl command in comments

**Validation**: Documentation reflects current behavior

## Phase 6: Integration and Validation

### Task 6.1: Test with real incidents
- [ ] Run full integration test with valid configuration (should succeed)
- [ ] Manually trigger each failure scenario and verify error messages
- [ ] Measure time-to-failure for config errors (target: < 5 seconds)
- [ ] Verify no regression in successful execution paths

**Validation**: All test scenarios pass, no performance regression

### Task 6.2: Update related documentation
- [ ] Update troubleshooting guide with preflight check errors
- [ ] Add preflight check details to architecture documentation
- [ ] Document how to skip preflight checks (if needed for testing)
- [ ] Update example Job manifests with comments about required volumes

**Validation**: Users can troubleshoot preflight failures independently

## Dependencies

- Task 1.2 depends on 1.1
- Tasks in Phase 2 can run in parallel after Phase 1
- Tasks in Phase 3 can run in parallel after Phase 1
- Phase 4 can run in parallel with Phases 2-3
- Phase 5-6 require all previous phases complete

## Estimated Effort

- Phase 1: 1 hour
- Phase 2: 1 hour
- Phase 3: 2 hours
- Phase 4: 1 hour
- Phase 5: 2 hours
- Phase 6: 2 hours
- **Total: ~9 hours** (can be completed in 1-2 sessions with parallelization)
