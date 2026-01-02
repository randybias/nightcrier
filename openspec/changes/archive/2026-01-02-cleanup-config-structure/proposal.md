# Change: Cleanup Configuration Structure

## Why

The configuration system has accumulated technical debt from the transition to Kubernetes-native agent execution. Several configuration options (`agent_script_path`, `agent_image`, `agent_verbose`, `skills.*`) are defined but never used, and the agent/k8s configuration uses flat naming inconsistent with the rest of the codebase which uses nested structure.

## What Changes

- **REMOVE** obsolete config options: `agent_script_path`, `agent_image`, `agent_verbose`
- **REMOVE** unused `skills` configuration section (skills are cloned from GitHub at runtime in `entrypoint.sh`)
- **REMOVE** obsolete directories: `agent-container/`, `agent-home/`
- **RESTRUCTURE** flat agent options to nested `agent:` section
- **RESTRUCTURE** flat k8s options to nested `k8s:` section
- **UPDATE** all config files and tests to match new structure

## Summary

Remove obsolete configuration options left over from the Docker-based agent execution model, and restructure the remaining agent/k8s configuration from flat keys to nested structure for consistency with the rest of the configuration.

## Problem Statement

The configuration system has accumulated technical debt from the transition to Kubernetes-native agent execution:

1. **Obsolete options still defined**: `agent_script_path`, `agent_image`, `agent_verbose` are defined in `config.go` but never used by any executor
2. **Obsolete options in example configs**: All config files reference these unused options
3. **Unused skills configuration**: The `skills` section (`cache_dir`, `disable_triage_preload`) is defined but never consumed - skills are now cloned from GitHub at runtime in `entrypoint.sh`
4. **Inconsistent structure**: Agent/K8s options use flat naming (`agent_foo`, `k8s_foo`) while other config sections use nested structure (`object_storage.url`, `state_storage.type`, `clusters[].mcp.endpoint`)
5. **Obsolete directories**: `agent-container/` and `agent-home/` are no longer needed
6. **Test files reference removed options**: `config_test.go` has 40+ references to obsolete config keys

## Current State

### Config files still reference:
- `agent_script_path: "./agent-container/run-agent.sh"` - file doesn't exist
- `agent_image: "nightcrier-agent:latest"` - superseded by `k8s_image`
- `agent_verbose: true` - never read by any code
- `skills.cache_dir: "./agent-home/skills"` - never used (K8s executor clones from GitHub)
- `skills.disable_triage_preload: false` - never used

### Skills loading reality:
The `nc-agent-runner/entrypoint.sh` script (lines 42-68) clones skills from GitHub at container startup:
```bash
git clone --depth 1 https://github.com/randybias/k8s4agents "$k8s_skill_path"
```
No local caching mechanism exists or is needed.

## Proposed Changes

### 1. Remove obsolete configuration options

**From `internal/config/config.go`:**
- `AgentScriptPath` field and env binding
- `AgentImage` field and env binding
- `AgentVerbose` field and env binding
- `Skills` / `SkillsConfig` struct and env bindings

### 2. Restructure flat agent/k8s options to nested format

**Before (flat):**
```yaml
agent_cli: "claude"
agent_model: "sonnet"
agent_timeout: 300
agent_system_prompt_file: "./configs/base-triage-prompt.md"
agent_allowed_tools: "Read,Write,Grep,Glob,Bash,Skill"
additional_agent_prompt: "..."

k8s_namespace: "nightcrier"
k8s_image: "nc-agent-runner:latest"
k8s_image_pull_policy: "IfNotPresent"
k8s_timeout: 600
k8s_memory_limit: "2Gi"
k8s_cpu_limit: "1"
k8s_cleanup_ttl: 3600
```

**After (nested):**
```yaml
agent:
  cli: "claude"
  model: "sonnet"
  timeout: 300
  system_prompt_file: "./configs/base-triage-prompt.md"
  allowed_tools: "Read,Write,Grep,Glob,Bash,Skill"
  additional_prompt: "..."

k8s:
  namespace: "nightcrier"
  image: "nc-agent-runner:latest"
  image_pull_policy: "IfNotPresent"
  timeout: 600
  memory_limit: "2Gi"
  cpu_limit: "1"
  cleanup_ttl: 3600
```

### 3. Remove obsolete directories

- `agent-container/` - only contains empty `scratch/` subdirectory
- `agent-home/` - unused skills cache directory

### 4. Update all config files

- `configs/config.example.yaml`
- `configs/config-weu.yaml`
- `configs/config-multicluster.yaml`
- `configs/config-test.yaml`
- All `configs/config-example-*.yaml` files

### 5. Update code references

- `internal/config/config.go` - restructure Config struct
- `internal/config/config_test.go` - update all test YAML fixtures
- `cmd/nightcrier/main.go` - update K8sExecutorConfig construction and status display
- Any other files referencing the old config keys

## Benefits

1. **Clarity**: Configuration only contains options that are actually used
2. **Consistency**: All configuration sections use nested structure
3. **Maintainability**: Removes ~150 lines of dead code and tests
4. **Discoverability**: Nested structure groups related options together
5. **Clean repository**: Removes obsolete directories

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Breaking existing deployments | Environment variables still work with new nested paths (e.g., `AGENT_CLI` → `AGENT_CLI` unchanged) |
| Migration confusion | Document the changes clearly in example config |
| Test failures | Update all test fixtures systematically |

## Scope

- **In scope**: Config struct, config files, config tests, main.go usage, obsolete directories
- **Out of scope**: K8sExecutorConfig struct (already well-structured), entrypoint.sh (no changes needed)

## Testing Strategy

This change requires comprehensive testing to ensure no regressions:

### Pre-Implementation Baseline
- Record all passing tests with `go test ./...`
- Document test count for comparison
- Backup working config files

### Unit Tests (Phase 4)
- Update all 35+ test functions in `config_test.go`
- Add new tests for `AgentConfig.Validate()` and `K8sConfig.Validate()`
- Add tests for `K8sConfig.ApplyDefaults()`
- Test nested YAML loading
- Test environment variable override of nested values

### Integration Tests (Phase 6)
- Validate all config files load without errors
- Test env var precedence over YAML
- Run live harness if cluster available

### Verification Checkpoints
- **Phase 1 checkpoint**: `go build` succeeds after removing dead code
- **Phase 2 checkpoint**: `go build` succeeds with new structs
- **Phase 3 checkpoint**: `go build` and `go vet` clean after migration
- **Phase 4 checkpoint**: All tests pass
- **Phase 7 final**: Full verification suite passes

### Rollback Plan
1. Restore backup config files from `.bak`
2. Revert commits using `git revert`
3. Old flat config fields documented in design.md for re-implementation if needed

## Success Criteria

1. All obsolete config options removed from codebase
2. Config structure uses consistent nested format
3. **All tests pass** - same count or more than baseline
4. **`go build ./...` succeeds**
5. **`go vet ./...` is clean**
6. Example config accurately reflects available options
7. No references to agent-container or agent-home in codebase
8. Environment variables continue to work unchanged
