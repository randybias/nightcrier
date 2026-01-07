# Proposal: Rename SSE References to MCP Transport

## Summary

Rename all configuration fields, environment variables, and documentation references from "SSE" to "MCP" to accurately reflect that nightcrier uses MCP's Streamable HTTP transport, not the deprecated SSE transport.

## Motivation

The current codebase uses "SSE" terminology (e.g., `sse_reconnect_initial_backoff`, `SSE_READ_TIMEOUT`) which is misleading:

1. **Technical inaccuracy**: We use `mcp.StreamableClientTransport` (the 2025-03-26 MCP spec), not the deprecated SSE transport (2024-11-05 spec)
2. **User confusion**: Users may think we're using the deprecated SSE transport when we're using the modern Streamable HTTP transport
3. **Future-proofing**: MCP may evolve the transport further; "MCP" is transport-agnostic while "SSE" implies a specific (deprecated) implementation

### Background

MCP transports evolved as follows:
- **2024-11-05**: SSE Transport - two endpoints (`/sse`, `/messages`), persistent SSE connection
- **2025-03-26**: Streamable HTTP Transport - single endpoint, uses SSE format internally but different architecture

We use the Streamable HTTP transport via `mcp.StreamableClientTransport`. While it uses SSE format (`text/event-stream`) for streaming responses, it's architecturally different from the deprecated "SSE Transport".

## Scope

### In Scope

1. **Config struct fields** (internal/config/config.go):
   - `SSEReconnectInitialBackoff` -> `MCPReconnectInitialBackoff`
   - `SSEReconnectMaxBackoff` -> `MCPReconnectMaxBackoff`
   - `SSEReadTimeout` -> `MCPReadTimeout`

2. **Config file keys** (mapstructure tags):
   - `sse_reconnect_initial_backoff` -> `mcp_reconnect_initial_backoff`
   - `sse_reconnect_max_backoff` -> `mcp_reconnect_max_backoff`
   - `sse_read_timeout` -> `mcp_read_timeout`

3. **Environment variables**:
   - `SSE_RECONNECT_INITIAL_BACKOFF` -> `MCP_RECONNECT_INITIAL_BACKOFF`
   - `SSE_RECONNECT_MAX_BACKOFF` -> `MCP_RECONNECT_MAX_BACKOFF`
   - `SSE_READ_TIMEOUT_SECONDS` -> `MCP_READ_TIMEOUT_SECONDS`

4. **Internal struct fields** (internal/cluster/manager.go):
   - `sseReconnectInitialBackoff` -> `mcpReconnectInitialBackoff`

5. **Documentation**:
   - docs/configuration.md
   - docs/architecture.md
   - docs/troubleshooting.md
   - internal/cluster/config.go comment

6. **Config examples**:
   - configs/config.example.yaml
   - configs/config-example-*.yaml
   - configs/tuning.yaml (if present)

7. **Test files**:
   - internal/config/config_test.go
   - Test config templates in tests/config-templates/

### Out of Scope

- Archived OpenSpec changes (historical record)
- Research and planning documents (historical context)
- The actual transport implementation (no code logic changes)

## Acceptance Criteria

1. All config fields, env vars, and mapstructure tags use "mcp" prefix instead of "sse"
2. All documentation accurately describes "MCP transport" or "Streamable HTTP"
3. No remaining "SSE" references in active code/config (excluding archives)
4. All tests pass with renamed configuration
5. Example configs work without modification after update

## Risks

- **Low**: Pure rename with no logic changes
- **Migration**: Clean break - no backwards compatibility needed (pre-release software)
