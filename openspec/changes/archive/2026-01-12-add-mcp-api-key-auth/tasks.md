## 1. Configuration Changes

- [x] 1.1 Add TLS enforcement validation in `internal/cluster/config.go`
  - When `mcp.api_key` is non-empty, validate that `mcp.endpoint` starts with `https://`
  - Return clear error message if HTTP endpoint used with API key
- [x] 1.2 Add environment variable binding for `MCP_API_KEY` in `internal/config/config.go`
  - Note: Not needed - viper handles nested config automatically via `mcp.api_key`
- [x] 1.3 Update `configs/config.example.yaml` with API key documentation and TLS requirement

## 2. HTTP Transport Enhancement

- [x] 2.1 Modify `internal/events/client.go` to accept API key parameter
- [x] 2.2 Create custom HTTP client with Authorization header when API key is configured
- [x] 2.3 Pass API key from cluster config through to events client construction
- [x] 2.4 Update client construction in manager to pass API key

## 3. AgentGateway Example Configuration

- [x] 3.1 Create `configs/agentgateway-example.yaml` demonstrating:
  - API key authentication policy with strict mode
  - TLS listener configuration
  - MCP backend target configuration
  - Example API keys with metadata

## 4. Testing

- [x] 4.1 Add unit test for TLS enforcement validation (api_key + http:// should fail)
- [x] 4.2 Add unit test for valid configuration (api_key + https://)
- [x] 4.3 Add unit test for no api_key (http:// or https:// both allowed)
- [x] 4.4 Verify HTTP client adds Authorization header when API key present
- [x] 4.5 Verify HTTP client omits Authorization header when API key absent

## 5. Documentation

- [x] 5.1 Update config.example.yaml with authentication setup guide
- [x] 5.2 Document the AgentGateway + Nightcrier authentication flow in agentgateway-example.yaml
