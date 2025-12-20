# Design: Remove Hardcoded Defaults and Consolidate Configuration

## Context

The codebase audit identified hardcoded values in multiple locations:

**Duplicated defaults** (same value in multiple files):
- Agent timeout `300` - in `config.go` and `executor.go`
- Allowed tools `"Read,Write,Grep,Glob,Bash,Skill"` - in `config.go` and `executor.go`
- Default model `"sonnet"` - in `config.go`, `executor.go`, and `run-agent.sh`
- Default prompt - duplicated in `config.go` and `executor.go`
- Timeout buffer `60` - in `main.go` and `executor.go`
- HTTP timeout `10 * time.Second` - in `main.go` and `slack.go`

**Operational tuning parameters** (not currently configurable):
- Investigation file size threshold `100` bytes
- Root cause truncation length `300`
- Failure reasons to display `3`
- Max failure reasons to track `5`
- Event channel buffer `100`
- I/O buffer sizes `1024`

## Goals

1. Eliminate all duplicate default definitions
2. Fail fast on missing required configuration
3. Separate rarely-changed tuning parameters from general configuration
4. Make all operational parameters configurable without code changes

## Non-Goals

- Changing the Dockerfile hardcoded values (out of scope)
- Changing display truncation length (deemed unimportant)
- Changing file/directory permissions (already appropriate)
- Modifying test files or config example files for defaults

## Decisions

### Decision 1: Separate Tuning from Configuration

**What**: Create a dedicated `tuning.yaml` file for operational parameters.

**Why**:
- Clear separation between "what to connect to" (config) and "how to operate" (tuning)
- Tuning values have sensible defaults that rarely need changing
- Reduces noise in the main config file
- Operators who need to tune can do so without touching primary config

**Structure**:
```yaml
# configs/tuning.yaml
http:
  slack_timeout_seconds: 10

agent:
  timeout_buffer_seconds: 60
  investigation_min_size_bytes: 100

reporting:
  root_cause_truncation_length: 300
  failure_reasons_display_count: 3
  max_failure_reasons_tracked: 5

events:
  channel_buffer_size: 100

io:
  stdout_buffer_size: 1024
  stderr_buffer_size: 1024
```

### Decision 2: Required vs Optional Configuration

**What**: Categorize all config parameters as required or optional.

**Required** (fail on missing):
- `mcp_endpoint` - Cannot function without MCP server
- `workspace_root` - Must know where to write incidents
- `agent_script_path` - Must know how to invoke agents
- `agent_timeout` - Critical operational parameter
- `subscribe_mode` - Must know what events to process
- At least one LLM API key

**Optional** (have sensible defaults in tuning.yaml or are truly optional features):
- `slack_webhook_url` - Feature is optional
- Azure storage config - Feature is optional
- All tuning parameters - Have defaults in tuning.yaml

### Decision 3: Configuration Loading Order

**What**: Define clear precedence for configuration sources.

**Order** (highest to lowest priority):
1. Command-line flags
2. Environment variables
3. `config.yaml` (or specified config file)
4. `tuning.yaml` (loaded automatically if present)

**Rationale**:
- Flags for ad-hoc overrides
- Env vars for container/deployment configuration
- Config file for persistent settings
- Tuning file for operational parameters

### Decision 4: Remove Defaults from Go Code

**What**: Remove all `viper.SetDefault()` calls and hardcoded fallback values.

**Why**:
- Forces explicit configuration
- Prevents silent misconfiguration
- Single source of truth in config files

**Exception**: Tuning parameters get defaults from `tuning.yaml`, which IS loaded by Go code but lives in a file, not hardcoded in source.

### Decision 5: run-agent.sh Parameter Handling

**What**: The shell script should receive ALL values from the Go code via environment variables.

**Why**:
- Go code is the single source of truth
- Eliminates duplicate defaults in shell script
- Shell script becomes a thin wrapper

**How**: Go sets environment variables before invoking the script:
```go
cmd.Env = append(os.Environ(),
    fmt.Sprintf("AGENT_MODEL=%s", cfg.AgentModel),
    fmt.Sprintf("AGENT_TIMEOUT=%d", cfg.AgentTimeout),
    // ... etc
)
```

## Risks / Trade-offs

### Risk: Breaking Existing Deployments
- **Mitigation**: Clear error messages explaining what's missing
- **Mitigation**: Updated `config.example.yaml` with all required fields
- **Mitigation**: Document migration in CHANGELOG

### Risk: Tuning File Complexity
- **Mitigation**: Tuning file is optional; sensible defaults embedded as fallback
- **Mitigation**: Only expose parameters that operators might actually tune

### Trade-off: Strictness vs Convenience
- **Choice**: Strictness - fail fast on missing config
- **Rationale**: Silent defaults cause production issues; explicit is better than implicit

## Migration Plan

1. Create `tuning.yaml` with current default values
2. Update `config.example.yaml` with all required fields documented
3. Remove `viper.SetDefault()` calls one category at a time
4. Add validation errors for missing required fields
5. Update `run-agent.sh` to require env vars instead of using defaults
6. Remove duplicate constants from `executor.go`
7. Test with various config combinations

## Open Questions

None - design is complete based on user requirements.
