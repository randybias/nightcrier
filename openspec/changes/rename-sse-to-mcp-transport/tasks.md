# Tasks: Rename SSE References to MCP Transport

## Phase 1: Core Code Changes

- [x] Rename config struct fields in `internal/config/config.go`:
  - `SSEReconnectInitialBackoff` -> `MCPReconnectInitialBackoff`
  - `SSEReconnectMaxBackoff` -> `MCPReconnectMaxBackoff`
  - `SSEReadTimeout` -> `MCPReadTimeout`
- [x] Update mapstructure tags to `mcp_reconnect_*` and `mcp_read_timeout`
- [x] Update environment variable mappings to `MCP_*`
- [x] Update validation error messages to reference new field/env names
- [x] Rename internal field in `internal/cluster/manager.go`:
  - `sseReconnectInitialBackoff` -> `mcpReconnectInitialBackoff`
- [x] Update `ClusterManagerConfig` field name and all references
- [x] Update `cmd/nightcrier/main.go` to use new field names

## Phase 2: Config Files

- [x] Update `configs/config.example.yaml` with new field names and comments
- [x] Update `configs/config-example-claude.yaml`
- [x] Update `configs/config-example-codex.yaml`
- [x] Update `configs/config-example-gemini.yaml`
- [x] Update `configs/config-example-goose.yaml`
- [x] Update `configs/tuning.yaml` if it contains SSE references (N/A - no SSE references)

## Phase 3: Documentation

- [x] Update `docs/configuration.md`:
  - Rename section "SSE Connection Settings" to "MCP Connection Settings"
  - Update all env var and field references
- [x] Update `docs/architecture.md`:
  - Replace "SSE Events" label in diagram with "MCP Events"
  - Update "Non-blocking ingestion" description
- [x] Update `docs/troubleshooting.md` comment about 405 response
- [x] Update comment in `internal/cluster/config.go` (line 37):
  - Remove "Server-Sent Events (SSE)" reference
  - Clarify it uses MCP Streamable HTTP transport

## Phase 4: Tests

- [x] Update `internal/config/config_test.go`:
  - Rename `TestValidation_SSEReconnectSettings` to `TestValidation_MCPReconnectSettings`
  - Update all YAML config snippets with new field names
  - Update expected field names and env vars in error tests
- [x] Update test config templates in `tests/config-templates/`:
  - `test-claude.yaml.tmpl`
  - `test-codex.yaml.tmpl`
  - `test-gemini.yaml.tmpl`
- [x] Update `internal/reload/reloader_test.go` embedded YAML config
- [x] Update `internal/reload/integration_test.go` embedded YAML config

## Phase 5: Verification

- [x] Run `go build ./...` to verify compilation
- [x] Run `go test ./...` to verify all tests pass
- [x] Grep for remaining "SSE" references in non-archive files - none found
