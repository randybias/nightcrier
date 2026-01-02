# Tasks: Configuration Structure Cleanup

## Phase 0: Pre-Implementation Validation

### 0.1 Establish baseline
- [ ] Run `go test ./...` and record all passing tests
- [ ] Run `go build ./...` and verify clean build
- [ ] Run `go vet ./...` and verify no issues
- [ ] Document current test count: `go test ./... -v 2>&1 | grep -c "=== RUN"`

### 0.2 Identify all code references to config fields
- [ ] `rg "AgentScriptPath|cfg\.AgentScriptPath" --type go` - document all usages
- [ ] `rg "AgentImage|cfg\.AgentImage" --type go` - document all usages
- [ ] `rg "AgentVerbose|cfg\.AgentVerbose" --type go` - document all usages
- [ ] `rg "Skills\.|SkillsConfig" --type go` - document all usages
- [ ] `rg "AgentCLI|AgentModel|AgentTimeout" --type go` - document all usages
- [ ] `rg "K8sNamespace|K8sImage|K8sTimeout" --type go` - document all usages

### 0.3 Backup working configs
- [ ] Copy `configs/config-weu.yaml` to `configs/config-weu.yaml.bak`
- [ ] Copy `configs/config-multicluster.yaml` to `configs/config-multicluster.yaml.bak`

## Phase 1: Remove Dead Code

### 1.1 Remove obsolete config fields from config.go
- [ ] Remove `AgentScriptPath` field from Config struct
- [ ] Remove `AgentImage` field from Config struct
- [ ] Remove `AgentVerbose` field from Config struct
- [ ] Remove `SkillsConfig` struct definition
- [ ] Remove `Skills` field from Config struct
- [ ] Remove env bindings for `agent_script_path`, `agent_image`, `agent_verbose`
- [ ] Remove env bindings for `skills.cache_dir`, `skills.disable_triage_preload`

### 1.2 Remove obsolete directories
- [ ] Remove `agent-container/` directory (only contains empty scratch/)
- [ ] Remove `agent-home/` directory (unused skills cache)
- [ ] Update `.gitignore` if these directories are referenced

### 1.3 Phase 1 checkpoint - verify no breakage
- [ ] Run `go build ./...` - must succeed
- [ ] Run `go vet ./...` - must be clean

## Phase 2: Create Nested Config Structs

### 2.1 Define AgentConfig struct
- [ ] Create `AgentConfig` struct with fields: CLI, Model, Timeout, SystemPromptFile, AllowedTools, AdditionalPrompt
- [ ] Add `Agent AgentConfig` field to main Config struct
- [ ] Add `Validate()` method to AgentConfig
- [ ] Add mapstructure tags for nested binding

### 2.2 Define K8sConfig struct
- [ ] Create `K8sConfig` struct with fields: Namespace, Image, ImagePullPolicy, Timeout, MemoryLimit, CPULimit, CleanupTTL
- [ ] Add `K8s K8sConfig` field to main Config struct
- [ ] Add `Validate()` method to K8sConfig
- [ ] Add `ApplyDefaults()` method to K8sConfig
- [ ] Add mapstructure tags for nested binding

### 2.3 Update environment variable bindings
- [ ] Update `bindEnvVars()` to use nested keys: `agent.cli`, `agent.model`, etc.
- [ ] Update `bindEnvVars()` to use nested keys: `k8s.namespace`, `k8s.image`, etc.
- [ ] Verify env var names remain unchanged (AGENT_CLI, K8S_NAMESPACE, etc.)

### 2.4 Phase 2 checkpoint - verify struct compilation
- [ ] Run `go build ./...` - must succeed
- [ ] Both old flat fields AND new nested fields exist temporarily

## Phase 3: Migrate Code to Nested Accessors

### 3.1 Update main.go
- [ ] Update K8sExecutorConfig construction to use `cfg.K8s.*` and `cfg.Agent.*`
- [ ] Update status display (line ~1087) to use `cfg.Agent.AllowedTools`
- [ ] Update any other references to old flat fields

### 3.2 Update config validation
- [ ] Move agent validation to `AgentConfig.Validate()`
- [ ] Move k8s validation to `K8sConfig.Validate()`
- [ ] Call nested validation from main `Config.Validate()`
- [ ] Call `K8sConfig.ApplyDefaults()` in validation

### 3.3 Remove old flat fields
- [ ] Remove `AgentSystemPromptFile` field (now `Agent.SystemPromptFile`)
- [ ] Remove `AgentAllowedTools` field (now `Agent.AllowedTools`)
- [ ] Remove `AgentModel` field (now `Agent.Model`)
- [ ] Remove `AgentTimeout` field (now `Agent.Timeout`)
- [ ] Remove `AgentCLI` field (now `Agent.CLI`)
- [ ] Remove `AdditionalAgentPrompt` field (now `Agent.AdditionalPrompt`)
- [ ] Remove `K8sNamespace` field (now `K8s.Namespace`)
- [ ] Remove `K8sImage` field (now `K8s.Image`)
- [ ] Remove `K8sImagePullPolicy` field (now `K8s.ImagePullPolicy`)
- [ ] Remove `K8sTimeout` field (now `K8s.Timeout`)
- [ ] Remove `K8sMemoryLimit` field (now `K8s.MemoryLimit`)
- [ ] Remove `K8sCPULimit` field (now `K8s.CPULimit`)
- [ ] Remove `K8sCleanupTTL` field (now `K8s.CleanupTTL`)

### 3.4 Phase 3 checkpoint - verify no compilation errors
- [ ] Run `go build ./...` - must succeed
- [ ] Run `go vet ./...` - must be clean

## Phase 4: Update Tests

### 4.1 Update config_test.go YAML fixtures
- [ ] Update all test YAML to use nested `agent:` structure
- [ ] Update all test YAML to use nested `k8s:` structure
- [ ] Remove all references to `agent_script_path`
- [ ] Remove all references to `agent_image`
- [ ] Remove all references to `agent_verbose`
- [ ] Remove all references to `skills:` section
- [ ] Update validation test cases for new structure

### 4.2 Add new tests for nested config
- [ ] Test `AgentConfig.Validate()` with valid config
- [ ] Test `AgentConfig.Validate()` with invalid CLI
- [ ] Test `AgentConfig.Validate()` with missing required fields
- [ ] Test `K8sConfig.Validate()` with valid config
- [ ] Test `K8sConfig.Validate()` with invalid ImagePullPolicy
- [ ] Test `K8sConfig.ApplyDefaults()` sets all defaults correctly
- [ ] Test nested config loads from YAML correctly
- [ ] Test nested config loads from env vars correctly
- [ ] Test env vars override nested YAML values

### 4.3 Run and verify all tests pass
- [ ] Run `go test ./internal/config/...` - all tests must pass
- [ ] Run `go test ./...` - full test suite must pass
- [ ] Verify test count matches or exceeds baseline from Phase 0

## Phase 5: Update Config Files

### 5.1 Update example config
- [ ] Rewrite `configs/config.example.yaml` with nested structure
- [ ] Remove obsolete options section (agent_script_path, agent_image, agent_verbose)
- [ ] Remove skills section
- [ ] Update all comments to reflect new structure
- [ ] Add migration note at top of file

### 5.2 Update working configs
- [ ] Update `configs/config-weu.yaml` to nested structure
- [ ] Update `configs/config-multicluster.yaml` to nested structure
- [ ] Update `configs/config-test.yaml` to nested structure

### 5.3 Update agent-specific example configs
- [ ] Update `configs/config-example-claude.yaml`
- [ ] Update `configs/config-example-codex.yaml`
- [ ] Update `configs/config-example-gemini.yaml`
- [ ] Update `configs/config-example-goose.yaml`

## Phase 6: Integration Testing

### 6.1 Config loading validation
- [ ] Create test script that loads each config file and validates
- [ ] Test `configs/config.example.yaml` loads without error
- [ ] Test `configs/config-weu.yaml` loads without error
- [ ] Test `configs/config-multicluster.yaml` loads without error

### 6.2 Environment variable validation
- [ ] Test AGENT_CLI env var overrides `agent.cli` in YAML
- [ ] Test AGENT_MODEL env var overrides `agent.model` in YAML
- [ ] Test K8S_NAMESPACE env var overrides `k8s.namespace` in YAML
- [ ] Test K8S_IMAGE env var overrides `k8s.image` in YAML
- [ ] Test all env vars work when config file has no agent/k8s sections

### 6.3 Live harness validation (if available)
- [ ] Run `tests/validate-harness.sh` if cluster available
- [ ] Verify nightcrier starts with updated config
- [ ] Verify K8s executor receives correct configuration

## Phase 7: Final Verification

### 7.1 Code verification
- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes with same or more tests than baseline
- [ ] `go vet ./...` clean
- [ ] `staticcheck ./...` clean (if available)

### 7.2 Codebase cleanup verification
- [ ] `rg "agent_script_path" --type go` returns NO results
- [ ] `rg "AgentScriptPath" --type go` returns NO results
- [ ] `rg "agent_image" --type go` returns NO results (except in comments about k8s_image)
- [ ] `rg "AgentImage" --type go` returns NO results
- [ ] `rg "agent_verbose" --type go` returns NO results
- [ ] `rg "AgentVerbose" --type go` returns NO results
- [ ] `rg "SkillsConfig" --type go` returns NO results
- [ ] `rg "skills\.cache_dir|skills\.disable" --type go` returns NO results

### 7.3 Documentation verification
- [ ] Example config accurately documents all options
- [ ] No references to agent-container directory in any file
- [ ] No references to agent-home directory in any file
- [ ] README or docs updated if they reference old config format

### 7.4 Config file verification
- [ ] All YAML config files use nested `agent:` structure
- [ ] All YAML config files use nested `k8s:` structure
- [ ] No YAML config files contain `agent_script_path`
- [ ] No YAML config files contain `agent_image`
- [ ] No YAML config files contain `agent_verbose`
- [ ] No YAML config files contain `skills:` section

## Phase 8: Cleanup

### 8.1 Remove backup files
- [ ] Remove `configs/config-weu.yaml.bak`
- [ ] Remove `configs/config-multicluster.yaml.bak`

### 8.2 Final commit preparation
- [ ] Review all changed files
- [ ] Ensure no unintended changes
- [ ] Prepare commit message following conventional commits

## Dependencies

- Phase 1 depends on Phase 0 (need baseline before changes)
- Phase 2 depends on Phase 1 (can't add new structs until dead code removed)
- Phase 3 depends on Phase 2 (can't use nested accessors until structs exist)
- Phase 4 depends on Phase 3 (tests must match new structure)
- Phase 5 depends on Phase 4 (config files must match final structure)
- Phase 6 depends on Phase 5 (integration tests need updated configs)
- Phase 7 depends on all above (final verification)
- Phase 8 depends on Phase 7 (cleanup after verification passes)

## Parallelizable Work

Within each phase:
- 0.2 tasks can all run in parallel
- 1.1 and 1.2 can be done in parallel
- 2.1 and 2.2 can be done in parallel
- 4.1 and 4.2 can be done in parallel
- 5.1-5.3 can be done in parallel
- 7.2-7.4 can all run in parallel

## Rollback Plan

If issues are discovered after implementation:
1. Restore backup config files from .bak
2. Revert commits using `git revert`
3. Old flat config fields are well-documented in this proposal for re-implementation if needed
