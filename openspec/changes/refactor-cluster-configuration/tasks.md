## 1. Database Schema

- [x] 1.1 Create migration `000005_add_cluster_tables.up.sql` with `monitored_clusters` and `execution_clusters` tables
- [x] 1.2 Create corresponding down migration
- [x] 1.3 Add cluster storage interface in `internal/storage/clusters.go`
- [x] 1.4 Implement SQLite cluster storage in `internal/storage/sqlite/clusters.go`
- [x] 1.5 Implement PostgreSQL cluster storage in `internal/storage/postgres/clusters.go`
- [x] 1.6 Add unit tests for cluster storage operations

## 2. Configuration Structs

- [x] 2.1 Create `ExecutionClusterConfig` struct in `internal/config/config.go`
- [x] 2.2 Create `MonitoredClusterConfig` struct replacing `ClusterConfig` in `internal/cluster/config.go`
- [x] 2.3 Rename `triage.kubeconfig` to `triage.target_kubeconfig_path` in struct tags
- [x] 2.4 Add `ExecutionDefaults` struct in `internal/config/config.go`
- [x] 2.5 Update main `Config` struct with new sections: `ExecutionClusters`, `MonitoredClusters`, `ExecutionDefaults`
- [x] 2.6 Remove old `K8sConfig` usage, replace with `ExecutionDefaults` + per-cluster overrides
- [x] 2.7 Update environment variable bindings in `bindEnvVars()`
- [x] 2.8 Update validation logic for new config structure
- [x] 2.9 Add unit tests for config loading and validation

## 3. Cluster Manager Refactor

- [x] 3.1 Update `ClusterManager` to use `MonitoredClusterConfig` instead of `ClusterConfig`
- [x] 3.2 Add `ExecutionClusterManager` for managing execution clusters
- [x] 3.3 Implement cluster lookup by name for execution cluster pinning
- [x] 3.4 Add `Reload(ctx context.Context)` method to cluster manager
- [x] 3.5 Implement graceful connection shutdown for removed clusters
- [x] 3.6 Implement new connection startup for added clusters
- [x] 3.7 Add support for zero clusters startup mode
- [x] 3.8 Add periodic database polling (30s) when no clusters configured
- [x] 3.9 Add unit tests for reload scenarios

## 4. Signal Handling

- [x] 4.1 Add SIGHUP handler in `cmd/nightcrier/main.go`
- [x] 4.2 Create `ConfigReloader` interface for coordinating reload across components
- [x] 4.3 Implement full config reload: re-read YAML, merge with database, apply changes
- [x] 4.4 Add reload status logging (what changed, what failed)
- [x] 4.5 Handle reload failures gracefully (keep previous config on validation failure)
- [x] 4.6 Add integration test for SIGHUP reload (in internal/reload/integration_test.go)

## 5. K8s Executor Updates

- [x] 5.1 Update `K8sExecutor` to accept execution cluster config at runtime
- [x] 5.2 Implement execution cluster selection based on `execution_cluster` reference
- [x] 5.3 Add fallback to default execution cluster when reference is empty
- [x] 5.4 Update Job creation to use per-cluster namespace and image settings
- [x] 5.5 Add unit tests for execution cluster selection

## 6. Configuration File Updates

- [x] 6.1 Update `configs/config.example.yaml` with new structure and documentation
- [x] 6.2 Update `configs/config-test.yaml`
- [x] 6.3 Update `configs/config-multicluster.yaml`
- [x] 6.4 Update `configs/config-weu.yaml`
- [x] 6.5 Update `configs/config-example-claude.yaml`
- [x] 6.6 Update `configs/config-example-codex.yaml`
- [x] 6.7 Update `configs/config-example-gemini.yaml`
- [x] 6.8 Update `configs/config-example-goose.yaml`
- [x] 6.9 Update `configs/config-test-claude.yaml`
- [x] 6.10 Update `configs/config-test-codex.yaml`
- [x] 6.11 Update `configs/config-test-gemini.yaml`
- [x] 6.12 Update `configs/config-test-goose.yaml`
- [x] 6.13 Update `configs/config-codex.yaml`
- [x] 6.14 Update `configs/config-codex-gpt41.yaml`
- [x] 6.15 Update `configs/config-codex-gpt52.yaml`
- [x] 6.16 Update `configs/tuning.yaml` if needed (N/A - no changes needed)

## 7. Bootstrap Updates

- [x] 7.1 Update bootstrap to use new config structure for kubeconfig secret creation
- [x] 7.2 Handle execution cluster kubeconfigs separately from monitored cluster target kubeconfigs
- [x] 7.3 Add tests for bootstrap with new config structure

## 8. Integration Testing

- [x] 8.1 Add integration test: startup with zero clusters, add via database, verify connection (in internal/reload/integration_test.go)
- [x] 8.2 Add integration test: SIGHUP reload with cluster additions/removals (in internal/reload/integration_test.go)
- [x] 8.3 Add integration test: database cluster overrides YAML cluster (in internal/reload/integration_test.go)
- [x] 8.4 Add integration test: execution cluster pinning (in internal/reload/integration_test.go)

## 9. Cleanup

- [x] 9.1 Remove deprecated `K8sConfig` struct after migration
- [x] 9.2 Remove old `clusters:` config key support
- [x] 9.3 Update any remaining references to old config structure
- [x] 9.4 Run full test suite, fix any failures
