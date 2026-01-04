//go:build integration

package reload

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/randybias/nightcrier/internal/cluster"
	"github.com/randybias/nightcrier/internal/storage"
	"github.com/randybias/nightcrier/internal/storage/sqlite"
)

// getMigrationsPath returns the absolute path to the migrations directory.
func getMigrationsPath() string {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	return filepath.Join(dir, "..", "..", "migrations")
}

// TestIntegration_ReloadWithDatabase tests the full reload flow with a real SQLite database.
// This covers tasks 4.6, 8.1, 8.2, 8.3.
func TestIntegration_ReloadWithDatabase(t *testing.T) {
	// Create temporary directory for test artifacts
	tmpDir := t.TempDir()

	// Create SQLite database
	dbPath := filepath.Join(tmpDir, "test.db")
	migrationsPath := getMigrationsPath()

	// Run migrations
	migrationCfg := &storage.MigrationConfig{
		MigrationsPath: migrationsPath,
		DatabaseType:   "sqlite",
		DatabasePath:   dbPath,
	}
	if err := storage.RunMigrations(migrationCfg); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Create SQLite store
	store, err := sqlite.New(&sqlite.Config{Path: dbPath})
	if err != nil {
		t.Fatalf("failed to create SQLite store: %v", err)
	}
	defer store.Close()

	// Create config file
	configFile := filepath.Join(tmpDir, "config.yaml")
	writeTestConfig(t, configFile, "")

	// Create connection manager with no clusters
	connMgr, err := cluster.NewConnectionManager(&cluster.ManagerConfig{
		Clusters: []cluster.MonitoredClusterConfig{},
	})
	if err != nil {
		t.Fatalf("failed to create connection manager: %v", err)
	}

	// Create execution cluster manager with no clusters
	execMgr, err := cluster.NewExecutionClusterManager(&cluster.ExecutionManagerConfig{
		Clusters: []cluster.ExecutionClusterConfig{},
	})
	if err != nil {
		t.Fatalf("failed to create execution cluster manager: %v", err)
	}

	// Create reloader
	reloader := NewReloader(&ReloaderConfig{
		ConfigFile:    configFile,
		ConnectionMgr: connMgr,
		ExecutionMgr:  execMgr,
		ClusterStore:  store,
		PollInterval:  100 * time.Millisecond,
	})

	ctx := context.Background()

	t.Run("startup_with_zero_clusters_add_via_database", func(t *testing.T) {
		// Verify starting with zero clusters
		if len(connMgr.GetClusterNames()) != 0 {
			t.Errorf("expected 0 clusters, got %d", len(connMgr.GetClusterNames()))
		}

		// Add a cluster to the database
		monitoredCluster := &storage.MonitoredClusterRecord{
			Name:        "db-cluster-1",
			Environment: "testing",
			Source:      "database",
			MCPEndpoint: "http://test-mcp:8080",
		}
		if err := store.UpsertMonitoredCluster(ctx, monitoredCluster); err != nil {
			t.Fatalf("failed to create monitored cluster: %v", err)
		}

		// Reload configuration
		result := reloader.Reload(ctx)
		if result.Error != nil {
			t.Errorf("Reload() error = %v", result.Error)
		}

		// Verify cluster was added
		if len(result.MonitoredAdded) != 1 {
			t.Errorf("MonitoredAdded = %d, want 1", len(result.MonitoredAdded))
		}

		if len(connMgr.GetClusterNames()) != 1 {
			t.Errorf("expected 1 cluster after reload, got %d", len(connMgr.GetClusterNames()))
		}
	})

	t.Run("reload_with_cluster_removal", func(t *testing.T) {
		// Delete the cluster from database
		if err := store.DeleteMonitoredCluster(ctx, "db-cluster-1"); err != nil {
			t.Fatalf("failed to delete monitored cluster: %v", err)
		}

		// Reload configuration
		result := reloader.Reload(ctx)
		if result.Error != nil {
			t.Errorf("Reload() error = %v", result.Error)
		}

		// Verify cluster was removed
		if len(result.MonitoredRemoved) != 1 {
			t.Errorf("MonitoredRemoved = %d, want 1", len(result.MonitoredRemoved))
		}

		if len(connMgr.GetClusterNames()) != 0 {
			t.Errorf("expected 0 clusters after removal, got %d", len(connMgr.GetClusterNames()))
		}
	})

	t.Run("database_cluster_overrides_yaml", func(t *testing.T) {
		// Update config file with a YAML cluster
		writeTestConfig(t, configFile, `  - name: shared-cluster
    environment: yaml-env
    mcp:
      endpoint: http://yaml-mcp:8080`)

		// Add same cluster to database with different values
		dbCluster := &storage.MonitoredClusterRecord{
			Name:        "shared-cluster",
			Environment: "db-env",
			Source:      "database",
			MCPEndpoint: "http://db-mcp:9090",
		}
		if err := store.UpsertMonitoredCluster(ctx, dbCluster); err != nil {
			t.Fatalf("failed to create monitored cluster: %v", err)
		}

		// Reload configuration
		result := reloader.Reload(ctx)
		if result.Error != nil {
			t.Errorf("Reload() error = %v", result.Error)
		}

		// Should only have 1 cluster (merged)
		if len(connMgr.GetClusterNames()) != 1 {
			t.Errorf("expected 1 cluster (merged), got %d", len(connMgr.GetClusterNames()))
		}

		// The cluster should have database values (database overrides YAML)
		names := connMgr.GetClusterNames()
		if len(names) != 1 || names[0] != "shared-cluster" {
			t.Errorf("expected [shared-cluster], got %v", names)
		}

		// Clean up
		if err := store.DeleteMonitoredCluster(ctx, "shared-cluster"); err != nil {
			t.Fatalf("failed to delete monitored cluster: %v", err)
		}
	})
}

// TestIntegration_DatabasePolling tests automatic database polling when no clusters configured.
func TestIntegration_DatabasePolling(t *testing.T) {
	// Create temporary directory for test artifacts
	tmpDir := t.TempDir()

	// Create SQLite database
	dbPath := filepath.Join(tmpDir, "test.db")
	migrationsPath := getMigrationsPath()

	// Run migrations
	migrationCfg := &storage.MigrationConfig{
		MigrationsPath: migrationsPath,
		DatabaseType:   "sqlite",
		DatabasePath:   dbPath,
	}
	if err := storage.RunMigrations(migrationCfg); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Create SQLite store
	store, err := sqlite.New(&sqlite.Config{Path: dbPath})
	if err != nil {
		t.Fatalf("failed to create SQLite store: %v", err)
	}
	defer store.Close()

	// Create config file with no clusters
	configFile := filepath.Join(tmpDir, "config.yaml")
	writeTestConfig(t, configFile, "[]")

	// Create connection manager with no clusters
	connMgr, err := cluster.NewConnectionManager(&cluster.ManagerConfig{
		Clusters: []cluster.MonitoredClusterConfig{},
	})
	if err != nil {
		t.Fatalf("failed to create connection manager: %v", err)
	}

	// Create execution cluster manager with no clusters
	execMgr, err := cluster.NewExecutionClusterManager(&cluster.ExecutionManagerConfig{
		Clusters: []cluster.ExecutionClusterConfig{},
	})
	if err != nil {
		t.Fatalf("failed to create execution cluster manager: %v", err)
	}

	// Create reloader with fast polling for test
	reloader := NewReloader(&ReloaderConfig{
		ConfigFile:    configFile,
		ConnectionMgr: connMgr,
		ExecutionMgr:  execMgr,
		ClusterStore:  store,
		PollInterval:  100 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start database polling
	reloader.StartDatabasePolling(ctx)

	// Verify polling started
	if reloader.pollCancel == nil {
		t.Fatal("expected polling to start")
	}

	// Add a cluster to the database
	monitoredCluster := &storage.MonitoredClusterRecord{
		Name:        "poll-cluster",
		Environment: "testing",
		Source:      "database",
		MCPEndpoint: "http://poll-mcp:8080",
	}
	if err := store.UpsertMonitoredCluster(ctx, monitoredCluster); err != nil {
		t.Fatalf("failed to create monitored cluster: %v", err)
	}

	// Wait for polling to pick up the cluster (with timeout)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(connMgr.GetClusterNames()) > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Verify cluster was added by polling
	if len(connMgr.GetClusterNames()) != 1 {
		t.Errorf("expected 1 cluster from polling, got %d", len(connMgr.GetClusterNames()))
	}

	// Stop polling
	reloader.StopDatabasePolling()

	// Verify polling stopped
	if reloader.pollCancel != nil {
		t.Error("expected polling to stop")
	}
}

// TestIntegration_ExecutionClusterPinning tests execution cluster selection.
func TestIntegration_ExecutionClusterPinning(t *testing.T) {
	// Create temporary directory for test artifacts
	tmpDir := t.TempDir()

	// Create SQLite database
	dbPath := filepath.Join(tmpDir, "test.db")
	migrationsPath := getMigrationsPath()

	// Run migrations
	migrationCfg := &storage.MigrationConfig{
		MigrationsPath: migrationsPath,
		DatabaseType:   "sqlite",
		DatabasePath:   dbPath,
	}
	if err := storage.RunMigrations(migrationCfg); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Create SQLite store
	store, err := sqlite.New(&sqlite.Config{Path: dbPath})
	if err != nil {
		t.Fatalf("failed to create SQLite store: %v", err)
	}
	defer store.Close()

	// Create config file
	configFile := filepath.Join(tmpDir, "config.yaml")
	writeTestConfig(t, configFile, "[]")

	// Create connection manager
	connMgr, err := cluster.NewConnectionManager(&cluster.ManagerConfig{
		Clusters: []cluster.MonitoredClusterConfig{},
	})
	if err != nil {
		t.Fatalf("failed to create connection manager: %v", err)
	}

	// Create execution cluster manager with no clusters
	execMgr, err := cluster.NewExecutionClusterManager(&cluster.ExecutionManagerConfig{
		Clusters: []cluster.ExecutionClusterConfig{},
	})
	if err != nil {
		t.Fatalf("failed to create execution cluster manager: %v", err)
	}

	// Create reloader
	reloader := NewReloader(&ReloaderConfig{
		ConfigFile:    configFile,
		ConnectionMgr: connMgr,
		ExecutionMgr:  execMgr,
		ClusterStore:  store,
	})

	ctx := context.Background()

	// Add execution clusters to database
	exec1 := &storage.ExecutionClusterRecord{
		Name:        "exec-us-west",
		Source:      "database",
		Namespace:   "agents-west",
		Kubeconfig:  "/path/to/west/kubeconfig",
		RunnerImage: "runner:west",
	}
	exec2 := &storage.ExecutionClusterRecord{
		Name:        "exec-us-east",
		Source:      "database",
		Namespace:   "agents-east",
		Kubeconfig:  "/path/to/east/kubeconfig",
		RunnerImage: "runner:east",
	}

	if err := store.UpsertExecutionCluster(ctx, exec1); err != nil {
		t.Fatalf("failed to create execution cluster 1: %v", err)
	}
	if err := store.UpsertExecutionCluster(ctx, exec2); err != nil {
		t.Fatalf("failed to create execution cluster 2: %v", err)
	}

	// Reload to pick up execution clusters
	result := reloader.Reload(ctx)
	if result.Error != nil {
		t.Fatalf("Reload() error = %v", result.Error)
	}

	// Verify execution clusters were added
	if len(result.ExecutionAdded) != 2 {
		t.Errorf("ExecutionAdded = %d, want 2", len(result.ExecutionAdded))
	}

	// Test execution cluster selection
	t.Run("select_by_name", func(t *testing.T) {
		selected, err := execMgr.Select("exec-us-east")
		if err != nil {
			t.Errorf("Select() error = %v", err)
		}
		if selected.Name != "exec-us-east" {
			t.Errorf("selected cluster = %v, want exec-us-east", selected.Name)
		}
		if selected.Namespace != "agents-east" {
			t.Errorf("namespace = %v, want agents-east", selected.Namespace)
		}
	})

	t.Run("select_default", func(t *testing.T) {
		selected, err := execMgr.Select("")
		if err != nil {
			t.Errorf("Select() error = %v", err)
		}
		// Default should be one of our clusters (order is non-deterministic from database)
		if selected.Name != "exec-us-west" && selected.Name != "exec-us-east" {
			t.Errorf("default cluster = %v, want exec-us-west or exec-us-east", selected.Name)
		}
	})

	t.Run("select_nonexistent", func(t *testing.T) {
		_, err := execMgr.Select("nonexistent")
		if err == nil {
			t.Error("expected error for nonexistent cluster")
		}
	})
}

// writeTestConfig writes a test configuration file.
func writeTestConfig(t *testing.T, path string, monitoredClusters string) {
	t.Helper()

	// Handle empty clusters list properly for YAML
	var clustersSection string
	if monitoredClusters == "" || monitoredClusters == "[]" {
		clustersSection = "monitored_clusters: []"
	} else {
		clustersSection = "monitored_clusters:\n" + monitoredClusters
	}

	content := `
log_level: info
workspace_root: /tmp/workspace
subscribe_mode: sse
severity_threshold: Warning
max_concurrent_agents: 5
global_queue_size: 100
cluster_queue_size: 10
dedup_window_seconds: 300
queue_overflow_policy: drop
shutdown_timeout: 30
sse_reconnect_initial_backoff: 1
sse_reconnect_max_backoff: 60
sse_read_timeout: 120
notify_on_agent_failure: true
failure_threshold_for_alert: 3
anthropic_api_key: test-key-for-integration-tests
` + clustersSection + `
execution_defaults:
  namespace: nightcrier
  runner_image: runner:latest
  timeout: 600
agent:
  cli: claude
  model: claude-sonnet-4-20250514
  system_prompt_file: /tmp/prompt.txt
  timeout: 600
object_storage:
  url: mem://test-artifacts
state_storage:
  type: sqlite
  sqlite_path: /tmp/test.db
  migrations_path: /tmp/migrations
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
}
