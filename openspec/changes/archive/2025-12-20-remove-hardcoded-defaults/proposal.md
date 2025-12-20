# Change: Remove Hardcoded Defaults and Consolidate Configuration

## Why

The codebase has hardcoded default values scattered across multiple files, leading to:
1. **Duplication**: Same defaults defined in `config.go`, `executor.go`, and `run-agent.sh`
2. **Silent failures**: Missing required config uses hidden defaults instead of failing fast
3. **No separation of concerns**: Operational tuning parameters mixed with general configuration

Operators cannot tune critical operational parameters (timeouts, buffer sizes, thresholds) without code changes.

## What Changes

### **BREAKING**: No More Implicit Defaults

- Remove all `viper.SetDefault()` calls from `config.go`
- Remove duplicate default constants from `executor.go`
- If a required config parameter is missing, fail at startup with clear error message
- All configuration must be explicit in config file or environment variables

### New: Separate Tuning Configuration

Introduce `configs/tuning.yaml` for operational parameters that are rarely changed:
- HTTP client timeouts
- I/O buffer sizes
- Agent timeout buffer
- Investigation file size threshold
- Root cause truncation length
- Failure reasons display count
- Event channel buffer capacity

### Consolidation

- Single source of truth for all configurable values
- `run-agent.sh` reads values from environment (set by Go code) instead of hardcoding
- Remove all duplicate definitions

## Impact

- **Affected specs**: `walking-skeleton` (configuration loading), `agent-failure-detection` (threshold)
- **Affected code**:
  - `internal/config/config.go` - Major refactor
  - `internal/config/tuning.go` - New file
  - `internal/agent/executor.go` - Remove defaults
  - `internal/reporting/slack.go` - Use tuning config
  - `internal/reporting/circuit_breaker.go` - Use tuning config
  - `internal/events/client.go` - Use tuning config
  - `cmd/nightcrier/main.go` - Use tuning config, remove hardcoded values
  - `agent-container/run-agent.sh` - Remove hardcoded defaults
  - `configs/config.example.yaml` - Update with all required fields
  - `configs/tuning.yaml` - New file with tuning defaults

## Migration

Existing deployments will fail to start until they provide explicit configuration. The `configs/config.example.yaml` will be updated to include all required fields with sensible values that users can copy.
