# Tasks: add-preflight-validation

## Phase 1: Package Foundation (3 hours)

### 1.1 Create package structure
- [ ] Create `internal/preflight/` directory
- [ ] Create `preflight.go` with core types (Validator interface, Runner, Result, ValidationError)
- [ ] Add package documentation
- [ ] Verify imports compile

### 1.2 Implement Runner
- [ ] Implement `Runner.Add(validator Validator)` method
- [ ] Implement `Runner.Run(ctx, cfg) error` - execute validators in sequence
- [ ] Implement `Result` aggregation logic
- [ ] Add early exit on first failure (fail-fast behavior)

### 1.3 Create base validator implementations
- [ ] Implement `ConfigConsistencyValidator` (wraps config.Validate())
- [ ] Implement `FileExistenceValidator` base type
- [ ] Add helper functions for error formatting with remediation

### 1.4 Add unit tests for infrastructure
- [ ] Test Runner with multiple validators
- [ ] Test ValidationError formatting
- [ ] Test fail-fast behavior (stops after first failure)
- [ ] Test empty validator list (should pass)

## Phase 2: Standard Validators (4 hours)

### 2.1 Implement SystemPromptValidator
- [ ] Create validator checking `cfg.AgentSystemPromptFile` exists
- [ ] Test with existing file (pass)
- [ ] Test with missing file (fail with remediation)
- [ ] Test with empty path (fail - required field)
- [ ] Test with unreadable file (permissions error)

### 2.2 Implement TriageKubeconfigValidator
- [ ] Create validator checking each cluster's triage kubeconfig path
- [ ] Skip disabled clusters
- [ ] Skip clusters without triage.kubeconfig set
- [ ] Test with all valid paths
- [ ] Test with one missing path (fail with cluster name in error)
- [ ] Test with multiple missing paths (report first failure)

### 2.3 Implement MigrationsPathValidator
- [ ] Create validator checking `cfg.StateStorage.MigrationsPath` directory exists
- [ ] Only validate if state storage requires migrations (SQLite, Postgres)
- [ ] Test with existing directory
- [ ] Test with missing directory
- [ ] Test with file instead of directory (error)

### 2.4 Add comprehensive validator unit tests
- [ ] Test each validator independently
- [ ] Mock file system operations where appropriate
- [ ] Verify error messages contain remediation steps
- [ ] Test with various config scenarios

## Phase 3: Integration (2 hours)

### 3.1 Wire pre-flight into main.go
- [ ] Import `internal/preflight` package
- [ ] Add `preflight.Validate()` call after config load, before logging setup
- [ ] Add "running pre-flight validation..." log message
- [ ] Add "pre-flight validation passed" log message on success
- [ ] Return validation error with proper formatting on failure
- [ ] Verify error includes newlines and remediation

### 3.2 Remove old validation code
- [ ] Remove system prompt file check from main.go (lines 124-131)
- [ ] Verify no duplicate checks remain
- [ ] Update error handling to use pre-flight errors

### 3.3 Update startup banner
- [ ] Add pre-flight validation status to banner
- [ ] Show "Pre-flight: ✓ Passed" in startup output

## Phase 4: Documentation (2 hours)

### 4.1 Create developer README
- [ ] Create `internal/preflight/README.md`
- [ ] Document when to add validators
- [ ] Document when NOT to add validators
- [ ] Provide code example for adding new validator
- [ ] Link to validation error formatting guidelines

### 4.2 Update main README
- [ ] Add "Pre-flight Validation" section to main README
- [ ] List what gets validated
- [ ] Show example error message
- [ ] Link to troubleshooting

### 4.3 Update configuration documentation
- [ ] Update `configs/config.example.yaml` with validation notes
- [ ] Add comments indicating which fields are validated during pre-flight
- [ ] Document required vs optional fields more clearly

## Phase 5: Testing (2 hours)

### 5.1 Integration tests
- [ ] Test full preflight with valid configuration (passes)
- [ ] Test with missing system prompt file (fails with clear error)
- [ ] Test with missing kubeconfig file (fails with cluster name)
- [ ] Test with missing migrations path (fails with directory name)
- [ ] Test with invalid config values (fails via ConfigConsistencyValidator)

### 5.2 Manual testing
- [ ] Start nightcrier with valid config → verify passes quickly
- [ ] Start with missing system prompt → verify error message is helpful
- [ ] Start with wrong kubeconfig path → verify error mentions cluster name
- [ ] Start with empty agent_system_prompt_file → verify required field error
- [ ] Verify startup time impact < 100ms

### 5.3 Error message validation
- [ ] Review all error messages for clarity
- [ ] Ensure remediation steps are actionable
- [ ] Verify configuration references include values and sources
- [ ] Test error formatting (newlines, indentation)

## Phase 6: Validation (1 hour)

### 6.1 Code quality
- [ ] Run `go vet ./internal/preflight/...`
- [ ] Run `golangci-lint run ./internal/preflight/...` (if available)
- [ ] Verify test coverage > 80%
- [ ] Check for error message consistency

### 6.2 End-to-end validation
- [ ] Rebuild nightcrier binary
- [ ] Test startup with production-like config
- [ ] Verify all validators execute
- [ ] Confirm no performance regression
- [ ] Test with CI/CD environment

## Dependencies

- **Phase 2** depends on Phase 1 (needs infrastructure)
- **Phase 3** depends on Phase 2 (needs validators implemented)
- **Phase 4** can run in parallel with Phase 3
- **Phase 5** depends on Phase 3 (needs integration complete)

## Parallelization Opportunities

- Phase 1.4 (infrastructure tests) can run alongside Phase 1.2-1.3 implementation
- Phase 2 validators (2.1, 2.2, 2.3) can be implemented in parallel by different developers
- Phase 4 documentation can start once Phase 2 validators are defined (doesn't need implementation complete)

## Estimated Total Time

- **Sequential execution**: 14 hours
- **With parallelization**: ~8-10 hours (if 2-3 developers)

## Success Criteria

- [ ] All validators execute in < 100ms total
- [ ] Application fails fast with clear errors on misconfiguration
- [ ] Zero silent failures or degraded operation modes
- [ ] Developer README makes it trivial to add new validators
- [ ] Error messages include actionable remediation steps
- [ ] Test coverage > 80% for preflight package
- [ ] No duplicate validation code in codebase
