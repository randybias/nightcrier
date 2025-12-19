# Proposal: Multi-Cluster MCP Server Support

## Summary

Enable nightcrier to connect to multiple kubernetes-mcp-server instances simultaneously, allowing centralized incident triage across many Kubernetes clusters. Initial target: 10 MCP servers. Scale target: 100+ MCP servers.

## Motivation

Production environments typically span multiple Kubernetes clusters:
- Development, staging, production environments
- Regional deployments (us-east, eu-west, etc.)
- Multi-tenant clusters
- Specialized clusters (GPU, high-memory, etc.)

Currently, nightcrier only connects to a single MCP endpoint. Operators must run separate instances to monitor multiple clusters, leading to:
- Operational overhead managing many instances
- No unified view of incidents
- Wasted resources (each instance has its own agent pool)

## Design Goals

1. **Centralized Management**: Single nightcrier instance monitors all clusters
2. **Scalability**: Support 10 servers initially, 100+ at scale
3. **Resilience**: Individual server failures don't affect others
4. **Resource Efficiency**: Shared agent pool across clusters
5. **Operational Visibility**: Health status for all connections

## Architecture

### Connection Manager

A new `ConnectionManager` component manages the lifecycle of multiple MCP connections:

```
┌─────────────────────────────────────────────────────────────────┐
│                        nightcrier                                │
│                                                                  │
│  ┌────────────────────────────────────────────────────────┐     │
│  │              Connection Manager                         │     │
│  │                                                         │     │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐      ┌─────────┐│     │
│  │  │ MCP     │  │ MCP     │  │ MCP     │ ...  │ MCP     ││     │
│  │  │ Client  │  │ Client  │  │ Client  │      │ Client  ││     │
│  │  │ (prod)  │  │ (stage) │  │ (dev)   │      │ (N)     ││     │
│  │  └────┬────┘  └────┬────┘  └────┬────┘      └────┬────┘│     │
│  └───────┼────────────┼───────────┼────────────────┼─────┘     │
│          │            │           │                │            │
│          └────────────┴─────┬─────┴────────────────┘            │
│                             ▼                                    │
│                    ┌────────────────┐                           │
│                    │  Event Router  │                           │
│                    │  (fan-in)      │                           │
│                    └───────┬────────┘                           │
│                            ▼                                     │
│            ┌───────────────────────────────┐                    │
│            │     Global Agent Pool         │                    │
│            │  (max_concurrent_agents=20)   │                    │
│            └───────────────────────────────┘                    │
└─────────────────────────────────────────────────────────────────┘
```

### Configuration

Clusters are defined in a YAML configuration file:

```yaml
# clusters.yaml
clusters:
  - name: production-us-east
    mcp_endpoint: "http://mcp-prod-east.internal:8383/mcp"
    kubeconfig: "/etc/nightcrier/kubeconfigs/prod-east.yaml"
    labels:
      env: production
      region: us-east

  - name: production-eu-west
    mcp_endpoint: "http://mcp-prod-eu.internal:8383/mcp"
    kubeconfig: "/etc/nightcrier/kubeconfigs/prod-eu.yaml"
    labels:
      env: production
      region: eu-west

  - name: staging
    mcp_endpoint: "http://mcp-staging.internal:8383/mcp"
    kubeconfig: "/etc/nightcrier/kubeconfigs/staging.yaml"
    labels:
      env: staging
```

### Connection Lifecycle

Each cluster connection follows this lifecycle:

```
DISCONNECTED → CONNECTING → CONNECTED → SUBSCRIBING → ACTIVE
      ↑            │             │            │          │
      └────────────┴─────────────┴────────────┴──────────┘
                        (on error)
```

With exponential backoff on reconnection (1s initial, 60s max).

### HTTP Connection Management

To handle 100+ concurrent HTTP connections efficiently:

1. **Connection Pooling**: Shared `http.Transport` with tuned settings:
   ```go
   transport := &http.Transport{
       MaxIdleConns:        200,
       MaxIdleConnsPerHost: 2,
       IdleConnTimeout:     90 * time.Second,
   }
   ```

2. **Keep-Alive**: HTTP keep-alive enabled for persistent connections

3. **Timeout Configuration**: Per-connection timeouts prevent blocking

4. **Resource Limits**: OS file descriptor limits documented

### Event Aggregation

Events from all clusters are fanned-in to a single channel with cluster metadata:

```go
type ClusterEvent struct {
    ClusterName string
    Kubeconfig  string
    Event       *FaultEvent
}
```

The existing deduplication and circuit breaker logic applies globally.

### Health Monitoring

A health endpoint exposes connection status:

```json
GET /health/clusters
{
  "clusters": [
    {"name": "production-us-east", "status": "active", "last_event": "2025-12-18T10:00:00Z"},
    {"name": "staging", "status": "reconnecting", "error": "connection refused", "retry_in": "8s"}
  ],
  "summary": {
    "total": 10,
    "active": 9,
    "unhealthy": 1
  }
}
```

### Graceful Degradation

- Failed connections are retried independently
- Events continue flowing from healthy connections
- Alerts raised for persistently failed connections
- Dynamic cluster add/remove via config reload (SIGHUP)

## Backwards Compatibility

The single-endpoint `mcp_endpoint` config option remains supported:
- If `mcp_endpoint` is set and `clusters` is empty, use single-cluster mode
- If `clusters` is populated, use multi-cluster mode
- Both cannot be set simultaneously (validation error)

## Scale Considerations

| Clusters | Connections | Memory (est.) | CPU (est.) |
|----------|-------------|---------------|------------|
| 10       | 10-20       | ~100 MB       | <5%        |
| 50       | 50-100      | ~300 MB       | <15%       |
| 100      | 100-200     | ~500 MB       | <25%       |

Assumptions:
- Each MCP connection: ~1-2 MB memory (HTTP client, buffers)
- Event processing: negligible per-event overhead
- Agent execution: already rate-limited by max_concurrent_agents

## Implementation Phases

### Phase 1: Foundation (This Proposal)
- Cluster registry and configuration
- Connection manager with lifecycle management
- Event aggregation (fan-in)
- Basic health monitoring

### Phase 2: Operations
- Config hot-reload (SIGHUP)
- Detailed metrics per cluster
- Cluster-specific overrides (severity thresholds, etc.)

### Phase 3: Scale
- Connection pooling optimizations
- Sharded event processing
- Cluster groups and priorities

## Alternatives Considered

1. **Separate Instances**: Run one nightcrier per cluster
   - Rejected: Operational overhead, no shared agent pool

2. **Proxy Layer**: HAProxy/nginx in front of MCP servers
   - Rejected: MCP protocol doesn't aggregate well; loses cluster identity

3. **Message Queue**: Events via Kafka/NATS
   - Rejected: Adds infrastructure complexity; MCP already handles streaming

## Open Questions

1. **Kubeconfig Distribution**: How do kubeconfigs get to nightcrier?
   - Option A: Mount as files (current approach)
   - Option B: Kubernetes secrets with sidecar injection
   - Option C: External secret manager (Vault, etc.)

2. **Cluster Discovery**: Should clusters be auto-discovered?
   - Initial: No, explicit configuration only
   - Future: Could integrate with fleet management tools

## Decision

Proceed with explicit YAML configuration for cluster definitions, supporting both single-cluster (backwards-compatible) and multi-cluster modes.
