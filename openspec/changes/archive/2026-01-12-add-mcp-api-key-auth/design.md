## Context

Nightcrier subscribes to fault events from kubernetes-mcp-server instances via MCP Streamable HTTP transport. In production, these MCP servers are fronted by AgentGateway, which provides authentication, rate limiting, and other security features.

AgentGateway supports API key authentication using the standard `Authorization: Bearer <key>` header pattern. The authentication happens at the HTTP layer, before the MCP protocol, making it transparent to MCP operations.

### Stakeholders
- Operators deploying Nightcrier in production environments
- Security teams requiring authenticated service-to-service communication

### Constraints
- Must not break existing unauthenticated deployments (api_key is optional)
- Must enforce TLS when credentials are transmitted
- Must follow AgentGateway's authentication patterns

## Goals / Non-Goals

### Goals
- Enable API key authentication for MCP server connections
- Enforce TLS when authentication is enabled
- Provide clear configuration examples for both Nightcrier and AgentGateway

### Non-Goals
- Implement JWT or OAuth authentication (future work)
- Automate AgentGateway deployment or configuration
- Support client certificate authentication (mTLS)

## Decisions

### Decision: Use Authorization Bearer Header

AgentGateway extracts API keys from the `Authorization: Bearer <key>` header. This is the standard pattern we will follow.

**Alternatives considered:**
- Custom `X-Api-Key` header: Not supported by AgentGateway's apiKey policy
- Query parameter: Less secure, appears in logs

### Decision: Enforce TLS When API Key Configured

When `mcp.api_key` is set, the `mcp.endpoint` MUST use HTTPS. Transmitting credentials over unencrypted HTTP is a security risk.

**Alternatives considered:**
- Warn but allow HTTP: Too risky; credentials could be intercepted
- Always require HTTPS: Too restrictive for development/testing scenarios

### Decision: Custom HTTP Client with Header

Create a custom `http.Client` with a `RoundTripper` that adds the Authorization header to all requests. This keeps the authentication logic isolated from the MCP client code.

**Alternatives considered:**
- Modify MCP SDK: Not maintainable
- Wrap transport at creation: Less clean separation

## Architecture

```
┌─────────────┐         HTTPS + Bearer Token         ┌──────────────────┐
│  Nightcrier │ ─────────────────────────────────────▶│   AgentGateway   │
│             │    Authorization: Bearer sk-xxx      │                  │
└─────────────┘                                      │  apiKey policy   │
                                                     │  (strict mode)   │
                                                     └────────┬─────────┘
                                                              │
                                                              ▼
                                                     ┌──────────────────┐
                                                     │ kubernetes-mcp-  │
                                                     │     server       │
                                                     └──────────────────┘
```

## Implementation Details

### HTTP Client with Auth Header

```go
// authTransport adds Authorization header to all requests
type authTransport struct {
    base   http.RoundTripper
    apiKey string
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    req = req.Clone(req.Context())
    req.Header.Set("Authorization", "Bearer "+t.apiKey)
    return t.base.RoundTrip(req)
}
```

### Configuration Validation

```go
// In MCPConfig.Validate() or MonitoredClusterConfig.Validate()
if c.MCP.APIKey != "" && !strings.HasPrefix(c.MCP.Endpoint, "https://") {
    return fmt.Errorf("cluster %s: mcp.api_key requires HTTPS endpoint (TLS) for security", c.Name)
}
```

## Risks / Trade-offs

### Risk: API Key Exposure in Logs
- **Mitigation**: Never log the API key value; use redacted logging

### Risk: Backward Compatibility
- **Mitigation**: API key is optional; existing configs without it continue to work

### Trade-off: TLS Enforcement Strictness
- Strict TLS requirement may complicate local development
- Acceptable trade-off for production security

## AgentGateway Configuration

Example AgentGateway configuration for reference:

```yaml
binds:
  - port: 8443
    listeners:
      - tls:
          cert: /certs/server.crt
          key: /certs/server.key
        routes:
          - path: /mcp
            policies:
              apiKey:
                mode: strict
                keys:
                  - key: "${MCP_API_KEY}"
                    metadata:
                      client: nightcrier
            backends:
              - mcp:
                  targets:
                    - name: kubernetes-mcp
                      http:
                        url: http://kubernetes-mcp-server:8080
```

## Open Questions

1. Should we support multiple API keys per cluster (key rotation)?
   - Current answer: No, single key for simplicity. Key rotation via config reload.

2. Should api_key support file references like `file:/path/to/key`?
   - Current answer: No, use environment variables. May add later if needed.
