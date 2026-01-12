# Change: Add MCP API Key Authentication

## Why

Nightcrier connects to remote MCP servers (kubernetes-mcp-server) to receive fault events. Currently, these connections are unauthenticated, which is a security risk in production environments. As the first step toward secure MCP communication, we need to implement shared API key authentication when connecting through AgentGateway.

AgentGateway fronts the kubernetes-mcp-server and supports API key authentication via Bearer tokens in the Authorization header. This is transparent to the MCP protocol since authentication happens at the HTTP transport layer.

## What Changes

- **Configuration validation**: When `mcp.api_key` is configured, enforce HTTPS endpoint (TLS required for credential security)
- **HTTP client enhancement**: Add `Authorization: Bearer <api_key>` header to MCP transport requests when API key is configured
- **Example configuration**: Provide AgentGateway configuration example showing how to set up API key authentication
- **Documentation**: Update config.example.yaml with API key usage guidance

## Impact

- Affected specs: `configuration`
- Affected code:
  - `internal/cluster/config.go` - Add TLS enforcement validation
  - `internal/events/client.go` - Add Authorization header to HTTP transport
  - `configs/config.example.yaml` - Document API key configuration
  - New file: `configs/agentgateway-example.yaml` - Example AgentGateway config

## Security Considerations

- API keys MUST only be transmitted over TLS (HTTPS) - enforced by configuration validation
- API keys should be stored securely (environment variables or secrets management)
- This is a stepping stone; JWT/OAuth authentication will follow for more sophisticated auth needs

## Out of Scope

- JWT authentication (future work)
- OAuth2 flows (future work)
- Mutual TLS (mTLS) authentication
- AgentGateway deployment/configuration automation
