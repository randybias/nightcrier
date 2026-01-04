# Change: Refactor Cluster Configuration Structure

## Why

The current configuration structure conflates concepts and uses confusing key names:
- `clusters[].triage.kubeconfig` is ambiguous - it's the kubeconfig for the agent to access the *target* cluster, not the executor cluster
- The global `k8s:` section mixes executor settings with general Kubernetes configuration
- No separation between "monitored clusters" (where faults occur) and "execution clusters" (where agent Jobs run)
- No database persistence for clusters, preventing runtime cluster management
- No hot-reload capability for cluster configuration changes

## What Changes

**BREAKING** - Complete restructure of cluster configuration:

1. **Split cluster types**: Rename `clusters:` to `monitored_clusters:` and add new `execution_clusters:` section
2. **Rename confusing keys**:
   - `triage.kubeconfig` → `triage.target_kubeconfig_path` (clarifies it's for agent access to the target cluster)
   - Move executor-specific settings from global `k8s:` to `execution_clusters[]:` entries
3. **Add database persistence**: New `clusters` and `execution_clusters` tables for runtime management
4. **Add hot-reload**: SIGHUP triggers full config reload from database (priority) and YAML file
5. **Allow zero clusters at startup**: System starts without clusters, monitors DB for additions

## Impact

- Affected specs: `configuration`, `cluster-registry`
- Affected code:
  - `internal/config/config.go` - Config struct changes
  - `internal/cluster/config.go` - ClusterConfig struct changes
  - `internal/cluster/manager.go` - Reload logic
  - `internal/storage/` - New cluster storage interface
  - `migrations/` - New database schema
  - `configs/*.yaml` - All config files need updating
  - `cmd/runner/` - Signal handling for SIGHUP
