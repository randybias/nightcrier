# Implementation Tasks: Multi-Cluster MCP Server Support

## Phase 1: Array Configuration ✅ COMPLETE

**Goal**: Config parses clusters array correctly but behavior is unchanged.

### 1.1 Define Cluster Configuration Types
- [x] Create `internal/cluster/config.go` with `ClusterConfig`, `MCPConfig`, `TriageConfig` structs
- [x] Add mapstructure tags for YAML parsing
- [x] Add `Validate()` method to `ClusterConfig`
- [ ] Write unit tests for cluster config validation (deferred)

### 1.2 Extend Main Config
- [x] Add `Clusters []cluster.ClusterConfig` field to `internal/config/config.go`
- [x] Remove `MCPEndpoint` field from Config (no backwards compatibility)
- [x] Update `Validate()` to require at least one cluster
- [x] Add validation for cluster name uniqueness
- [x] Add validation for required fields (name, mcp.endpoint)
- [x] Add validation for triage.kubeconfig when triage.enabled=true
- [ ] Write unit tests for new config validation rules (deferred)

### 1.3 Update Example Configuration
- [x] Create `kubeconfigs/` directory with `.gitkeep`
- [x] Update `configs/config.example.yaml` with clusters array structure
- [x] Update `configs/config-test.yaml` with clusters array structure
- [x] Add placeholder MCP API key comments explaining future use
- [x] Document kubeconfig path conventions

### 1.4 Update Environment Variable Bindings
- [x] Update `bindEnvVars()` to handle nested cluster config (or document that clusters must come from file)
- [x] Update `BindFlags()` to remove single-endpoint flag
- [x] Update flag help text

### 1.5 Verification
- [x] Run `go build ./...` to verify compilation
- [x] Run existing tests to verify they still pass (with updated test configs)
- [x] Manually test config loading with new structure

---

## Phase 2: Refactor to Use Array ✅ COMPLETE

**Goal**: Works with array of 1 cluster, behavior identical to before.

### 2.1 Create Cluster Package Foundation
- [x] Create `internal/cluster/registry.go` with `Registry` struct
- [x] Implement `Registry.Load(clusters []ClusterConfig)`
- [x] Implement `Registry.Get(name string) *ClusterConfig`
- [x] Implement `Registry.List() []*ClusterConfig`
- [ ] Write unit tests for registry operations (deferred)

### 2.2 Create Connection Types
- [x] Create `internal/cluster/connection.go`
- [x] Define `ConnectionStatus` enum (disconnected, connecting, connected, subscribing, active, failed)
- [x] Define `ClusterConnection` struct with status tracking
- [x] Implement `NewClusterConnection(cfg *ClusterConfig)`
- [ ] Write unit tests for connection state (deferred)

### 2.3 Create ClusterEvent Type
- [x] ClusterEvent implemented as map[string]interface{} with cluster metadata
- [x] Add cluster name, kubeconfig path, permissions, labels fields

### 2.4 Create Connection Manager
- [x] Create `internal/cluster/manager.go`
- [x] Implement `NewConnectionManager(cfg *config.Config)`
- [x] Implement shared `http.Transport` with pooling settings
- [x] Implement `Start(ctx) <-chan interface{}`
- [x] Implement per-connection goroutine that fans-in events
- [x] Implement `Stop()` for graceful shutdown
- [ ] Write unit tests for manager lifecycle (deferred)

### 2.5 Update Main Application
- [x] Update `cmd/nightcrier/main.go` to create `ConnectionManager`
- [x] Update event loop to receive `ClusterEvent` map instead of `FaultEvent`
- [x] Add cluster name to all event logging
- [x] Pass cluster kubeconfig to agent executor
- [x] Update startup banner to show cluster count

### 2.6 Update Agent Executor
- [x] Add `Kubeconfig` field to `ExecutorConfig`
- [x] Update `Execute()` to pass `--kubeconfig` flag to run-agent.sh
- [x] Update executor creation to receive kubeconfig from cluster config
- [ ] Write unit test for kubeconfig parameter passing (deferred)

### 2.7 Update Slack Notifications
- [x] Add cluster name to Slack notification messages
- [x] Update notification formatting to include cluster context

### 2.8 Verification
- [x] Run full test suite
- [x] Test with multiple cluster config
- [x] Verify logs include cluster name
- [x] Verified with live clusters (westeu-cluster1, eastus-cluster1)

---

## Phase 3: Enable Real Triage ✅ COMPLETE

**Goal**: Triage agents produce meaningful output by connecting to actual clusters.

### 3.1 Implement Permission Validation
- [x] Create `internal/cluster/permissions.go`
- [x] Define `ClusterPermissions` struct with permission flags
- [x] Implement `validateClusterPermissions(ctx, cfg)` using `kubectl auth can-i --list`
- [x] Parse kubectl output to determine specific permissions
- [x] Implement `MinimumPermissionsMet()` check
- [x] Build warnings list for missing permissions
- [ ] Write unit tests with mock kubectl output (deferred)

### 3.2 Integrate Permission Validation at Startup
- [x] Add permissions field to `ClusterConnection`
- [x] Call `validateClusterPermissions()` during `ConnectionManager.Initialize()`
- [x] Skip validation if `triage.enabled=false`
- [x] Log warning if minimum permissions not met
- [x] Fail startup if kubeconfig file doesn't exist (when triage enabled)

### 3.3 Update Workspace Creation
- [x] Add `ClusterPermissions` to `ClusterEvent` (passed as map field)
- [x] Update processEvent to write `incident_cluster_permissions.json` to workspace
- [x] Include cluster name in incident.json metadata (already done in Phase 2)

### 3.4 Update Triage Skip Logic
- [x] Check `triage.enabled` before spawning agent (via permissions==nil check)
- [x] Log "triage disabled for cluster" when skipping
- [x] Skip notification when triage not performed (return early from processEvent)
- [ ] Add metrics/counter for skipped triage (optional - deferred)

### 3.5 Agent Integration (Critical Fix)
- [x] Update `configs/triage-system-prompt.md` to instruct agent about permissions file
- [x] Add Docker volume mount for `incident_cluster_permissions.json` in run-agent.sh
- [x] Update all `agent_prompt` configs to reference permissions file
- [x] Verify agent can read cluster context and permissions
- [x] Verify agent successfully runs kubectl commands with provided kubeconfig
- [x] Confirmed working with live clusters (westeu-cluster1, eastus-cluster1)

### 3.6 Permissions File Storage Upload
- [x] Add `ClusterPermissionsJSON` field to `IncidentArtifacts` struct
- [x] Update `readIncidentArtifacts()` to read permissions file from workspace
- [x] Update Azure storage to upload permissions file
- [x] Update index.html generation to include permissions file
- [x] Update filesystem storage to copy permissions file (inherited from artifact upload)

### 3.7 Runtime Verification (Completed with Live Clusters)
- [x] Verify `incident_cluster_permissions.json` is created in workspaces
- [x] Verify `incident_cluster_permissions.json` is uploaded to Azure blob storage
- [x] Verify agent can read pods, logs, events from cluster
- [x] Verify triage output references actual cluster resources
- [x] Verify investigation.md contains real cluster data (pod names, node IPs, etc.)
- [x] Verify cluster name appears in investigation report header
- [x] Verify permissions file appears in index.html with SAS URL

---

## Phase 4: Multi-Cluster Validation ✅ CODE COMPLETE

**Goal**: Production-ready multi-cluster support.

### 4.1 Test with Two Clusters ✅ VERIFIED
- [x] Configure Nightcrier with two clusters (westeu-cluster1, eastus-cluster1)
- [x] Verify both connections establish (both show "active" status)
- [x] Trigger fault on each cluster
- [x] Verify correct kubeconfig used for each (confirmed in investigation reports)
- [x] Verify event counting per cluster (health endpoint shows event_count)

### 4.2 Verify Independent Connection Lifecycle
- [ ] Stop one kubernetes-mcp-server (user runtime test)
- [ ] Verify other cluster continues receiving events (user runtime test)
- [ ] Verify reconnection attempts on failed cluster (user runtime test)
- [ ] Restart the stopped server (user runtime test)
- [ ] Verify reconnection succeeds (user runtime test)

### 4.3 Verify Event Fan-In
- [ ] Generate faults on both clusters simultaneously (user runtime test)
- [ ] Verify both events are processed (user runtime test)
- [ ] Verify each event uses correct cluster kubeconfig (user runtime test)
- [ ] Verify deduplication works per-cluster (user runtime test)

### 4.4 Add Health Monitoring ✅ COMPLETE
- [x] Implement `/health/clusters` endpoint (internal/health/server.go)
- [x] Return per-cluster status, event count, last error
- [x] Return summary with total/active/unhealthy counts
- [x] Include triage enabled status per cluster
- [x] Include full permissions in health response
- [x] Add --health-port flag (default 8080, 0 to disable)
- [x] Verified working with live clusters

### 4.5 Test with Three Clusters
- [ ] Add third cluster configuration (user runtime test)
- [ ] Verify all three connections work (user runtime test)
- [ ] Stress test with concurrent faults (user runtime test)

### 4.6 Documentation ✅ COMPLETE
- [x] Update README with multi-cluster configuration
- [x] Document kubeconfig creation and placement
- [x] Document triage enable/disable behavior
- [x] Add troubleshooting section for connection issues
- [x] Add ServiceAccount RBAC examples
- [x] Add kubeconfig extraction script
- [x] Document health monitoring endpoint

### 4.7 Final Verification
- [x] Verify basic functionality with two live clusters
- [ ] Test graceful shutdown with multiple active connections (user runtime test)
- [ ] Verify no resource leaks (goroutines, connections) (user runtime test)
- [ ] Extended runtime testing (hours) (user runtime test)

---

## Task Dependencies

```
Phase 1: Configuration
    1.1 → 1.2 → 1.3 → 1.4 → 1.5

Phase 2: Refactor (depends on Phase 1)
    2.1 ─┐
    2.2 ─┼→ 2.4 → 2.5 → 2.6 → 2.7 → 2.8
    2.3 ─┘

Phase 3: Triage (depends on Phase 2)
    3.1 → 3.2 ─┐
               ├→ 3.4 → 3.5 → 3.6 → 3.7
    3.3 ───────┘

Phase 4: Multi-Cluster (depends on Phase 3)
    4.1 → 4.2 → 4.3 → 4.4 → 4.5 → 4.6 → 4.7
```

---

## Out of Scope (Future Work)

- Config hot-reload (SIGHUP)
- Per-cluster Prometheus metrics
- Per-cluster overrides (severity threshold, agent model)
- MCP API key authentication
- Scale testing (100+ clusters)
- Cluster discovery / auto-registration
