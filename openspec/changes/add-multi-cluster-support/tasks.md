# Implementation Tasks: Multi-Cluster MCP Server Support

## Phase 1: Foundation

### 1. Configuration Extension
- [ ] 1.1 Define `ClusterConfig` struct in `internal/config/cluster.go`
- [ ] 1.2 Add `Clusters []ClusterConfig` field to main `Config` struct
- [ ] 1.3 Add validation for cluster config (name uniqueness, required fields)
- [ ] 1.4 Add mutual exclusivity check (mcp_endpoint XOR clusters)
- [ ] 1.5 Add cluster config defaults (enabled=true, etc.)
- [ ] 1.6 Update `configs/config.example.yaml` with clusters section
- [ ] 1.7 Write unit tests for cluster config validation
- [ ] 1.8 Write unit tests for mutual exclusivity validation

### 2. Cluster Registry
- [ ] 2.1 Create `internal/cluster/registry.go`
- [ ] 2.2 Implement `Registry` struct to hold cluster configurations
- [ ] 2.3 Implement `Registry.Load(cfg *config.Config)` to populate from config
- [ ] 2.4 Implement `Registry.Get(name string) *ClusterConfig`
- [ ] 2.5 Implement `Registry.List() []*ClusterConfig`
- [ ] 2.6 Implement `Registry.GetEnabled() []*ClusterConfig` (filter disabled)
- [ ] 2.7 Write unit tests for registry operations

### 3. Connection Lifecycle
- [ ] 3.1 Create `internal/cluster/connection.go`
- [ ] 3.2 Define `ConnectionStatus` enum (disconnected, connecting, connected, subscribing, active, failed)
- [ ] 3.3 Define `ClusterConnection` struct with status, timestamps, error tracking
- [ ] 3.4 Implement `NewClusterConnection(cfg *ClusterConfig)` constructor
- [ ] 3.5 Implement `ClusterConnection.Connect(ctx)` method
- [ ] 3.6 Implement `ClusterConnection.Subscribe(ctx)` method
- [ ] 3.7 Implement `ClusterConnection.Close()` method
- [ ] 3.8 Implement `ClusterConnection.Status()` method
- [ ] 3.9 Implement `ClusterConnection.HealthSnapshot()` for monitoring
- [ ] 3.10 Write unit tests for connection state transitions

### 4. Reconnection Logic
- [ ] 4.1 Define `ReconnectConfig` struct (initial backoff, max backoff, multiplier, jitter)
- [ ] 4.2 Implement exponential backoff calculation with jitter
- [ ] 4.3 Implement `ClusterConnection.reconnectLoop(ctx)` goroutine
- [ ] 4.4 Add logging for reconnection attempts with backoff duration
- [ ] 4.5 Add retry counter and last error tracking
- [ ] 4.6 Write unit tests for backoff calculation
- [ ] 4.7 Write integration test for reconnection behavior

### 5. Connection Manager
- [ ] 5.1 Create `internal/cluster/manager.go`
- [ ] 5.2 Define `ConnectionManager` struct
- [ ] 5.3 Implement shared `http.Transport` with connection pooling settings
- [ ] 5.4 Implement `NewConnectionManager(cfg *config.Config)` constructor
- [ ] 5.5 Implement `ConnectionManager.Start(ctx) <-chan *ClusterEvent`
- [ ] 5.6 Implement per-connection goroutine management
- [ ] 5.7 Implement `ConnectionManager.Stop()` for graceful shutdown
- [ ] 5.8 Implement `ConnectionManager.GetHealth() HealthSummary`
- [ ] 5.9 Write unit tests for manager lifecycle
- [ ] 5.10 Write integration test with multiple mock MCP servers

### 6. Event Routing
- [ ] 6.1 Define `ClusterEvent` struct (wraps FaultEvent with cluster metadata)
- [ ] 6.2 Modify event channel to use `ClusterEvent` instead of `FaultEvent`
- [ ] 6.3 Implement fan-in logic in connection goroutines
- [ ] 6.4 Apply overflow policy (drop/reject) at fan-in point
- [ ] 6.5 Add cluster name and labels to event logging
- [ ] 6.6 Update deduplication key to include cluster name
- [ ] 6.7 Write unit tests for event routing
- [ ] 6.8 Write tests for overflow handling

### 7. MCP Client Updates
- [ ] 7.1 Add `NewClientWithHTTPClient(endpoint, httpClient)` constructor to `events.Client`
- [ ] 7.2 Update `Client` to accept external HTTP client (for shared transport)
- [ ] 7.3 Ensure backward compatibility with `NewClient(endpoint)` constructor
- [ ] 7.4 Write unit tests for new constructor

### 8. Agent Executor Updates
- [ ] 8.1 Add `Kubeconfig` field to `ExecutorConfig`
- [ ] 8.2 Pass kubeconfig path to agent script via `-k` flag
- [ ] 8.3 Update `run-agent.sh` to accept and use kubeconfig argument
- [ ] 8.4 Update agent Dockerfile if needed for kubeconfig handling
- [ ] 8.5 Write integration test with cluster-specific kubeconfig

### 9. Main Application Integration
- [ ] 9.1 Update `cmd/nightcrier/main.go` to detect single vs multi-cluster mode
- [ ] 9.2 Create `ConnectionManager` when multi-cluster mode is enabled
- [ ] 9.3 Create single `events.Client` when single-cluster mode (backwards compat)
- [ ] 9.4 Update event processing loop to handle `ClusterEvent`
- [ ] 9.5 Pass cluster kubeconfig to agent executor
- [ ] 9.6 Update logging to include cluster context
- [ ] 9.7 Update Slack notifications to include cluster name
- [ ] 9.8 Write integration test for single-cluster mode
- [ ] 9.9 Write integration test for multi-cluster mode

### 10. Health Monitoring
- [ ] 10.1 Define `ClusterHealth` struct for per-cluster status
- [ ] 10.2 Define `HealthSummary` struct for aggregate health
- [ ] 10.3 Implement `/health/clusters` HTTP endpoint
- [ ] 10.4 Add health server startup (optional, config-driven)
- [ ] 10.5 Add health port configuration option
- [ ] 10.6 Write unit tests for health endpoint

---

## Phase 2: Operations (Future)

### 11. Config Hot-Reload
- [ ] 11.1 Implement SIGHUP handler for config reload
- [ ] 11.2 Implement cluster diff (add/remove/modify detection)
- [ ] 11.3 Add new connections for added clusters
- [ ] 11.4 Remove connections for deleted clusters
- [ ] 11.5 Update config for modified clusters (reconnect if needed)
- [ ] 11.6 Write integration test for hot-reload

### 12. Per-Cluster Metrics
- [ ] 12.1 Add event counter per cluster
- [ ] 12.2 Add connection uptime per cluster
- [ ] 12.3 Add error counter per cluster
- [ ] 12.4 Expose metrics via Prometheus endpoint
- [ ] 12.5 Write unit tests for metrics

### 13. Cluster-Specific Overrides
- [ ] 13.1 Support per-cluster severity threshold
- [ ] 13.2 Support per-cluster agent model override
- [ ] 13.3 Support per-cluster enabled/disabled toggle
- [ ] 13.4 Write unit tests for override behavior

---

## Phase 3: Scale (Future)

### 14. Connection Optimization
- [ ] 14.1 Tune HTTP transport settings for 100+ connections
- [ ] 14.2 Add connection health check (ping/keepalive)
- [ ] 14.3 Implement connection prioritization
- [ ] 14.4 Add OS file descriptor limit documentation

### 15. Sharded Processing
- [ ] 15.1 Design worker pool for parallel event processing
- [ ] 15.2 Implement event sharding by cluster
- [ ] 15.3 Add per-shard circuit breakers
- [ ] 15.4 Write load tests for 100 clusters

---

## Documentation

### 16. User Documentation
- [ ] 16.1 Document multi-cluster configuration in README
- [ ] 16.2 Document kubeconfig distribution patterns
- [ ] 16.3 Document health endpoint usage
- [ ] 16.4 Document resource requirements at scale
- [ ] 16.5 Add troubleshooting guide for connection issues

### 17. Operations Guide
- [ ] 17.1 Document recommended OS tuning (file descriptors, etc.)
- [ ] 17.2 Document monitoring and alerting patterns
- [ ] 17.3 Document graceful rollout procedures

---

## Verification

### 18. Testing
- [ ] 18.1 Run all unit tests: `go test ./...`
- [ ] 18.2 Run integration tests with mock MCP servers
- [ ] 18.3 Test with 2 real kubernetes-mcp-server instances
- [ ] 18.4 Test with 10 mock MCP servers (scale test)
- [ ] 18.5 Test reconnection under network failures
- [ ] 18.6 Test graceful shutdown with active connections
- [ ] 18.7 Verify backwards compatibility (single endpoint mode)

### 19. Cleanup
- [ ] 19.1 Run `go fmt` and `go vet`
- [ ] 19.2 Review all new code for error handling
- [ ] 19.3 Update CHANGELOG
- [ ] 19.4 Archive this change proposal

---

## Parallelization Notes

- Tasks 1-2 (config, registry) can proceed in parallel
- Tasks 3-4 (connection lifecycle) depend on task 2
- Task 5 (manager) depends on tasks 3-4
- Task 6 (routing) can proceed in parallel with tasks 3-5
- Task 7 (client updates) can proceed independently
- Tasks 8-9 depend on tasks 5-7
- Task 10 (health) can proceed after task 5
