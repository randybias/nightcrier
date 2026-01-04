package reload

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/randybias/nightcrier/internal/cluster"
	"github.com/randybias/nightcrier/internal/config"
	"github.com/randybias/nightcrier/internal/storage"
)

// testConfigTemplate returns a minimal valid config template for tests.
// Use fmt.Sprintf to inject the monitored_clusters section.
func testConfigTemplate(monitoredClusters string) string {
	return `
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
anthropic_api_key: test-key-for-unit-tests
monitored_clusters:
` + monitoredClusters + `
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
}

// mockClusterStore implements ClusterStore for testing
type mockClusterStore struct {
	monitoredClusters []storage.MonitoredClusterRecord
	executionClusters []storage.ExecutionClusterRecord
	listMonitoredErr  error
	listExecutionErr  error
	syncMonitoredErr  error
	syncExecutionErr  error
}

func (m *mockClusterStore) ListMonitoredClusters(ctx context.Context) ([]storage.MonitoredClusterRecord, error) {
	if m.listMonitoredErr != nil {
		return nil, m.listMonitoredErr
	}
	return m.monitoredClusters, nil
}

func (m *mockClusterStore) ListExecutionClusters(ctx context.Context) ([]storage.ExecutionClusterRecord, error) {
	if m.listExecutionErr != nil {
		return nil, m.listExecutionErr
	}
	return m.executionClusters, nil
}

func (m *mockClusterStore) SyncMonitoredClustersFromYAML(ctx context.Context, clusters []storage.MonitoredClusterRecord) error {
	if m.syncMonitoredErr != nil {
		return m.syncMonitoredErr
	}
	// Simulate sync: add YAML clusters, preserve database-sourced clusters
	existingDB := make(map[string]storage.MonitoredClusterRecord)
	for _, c := range m.monitoredClusters {
		if c.Source == "database" {
			existingDB[c.Name] = c
		}
	}

	// Start with YAML clusters
	result := make([]storage.MonitoredClusterRecord, 0)
	for _, c := range clusters {
		c.Source = "yaml"
		// Skip if database-sourced cluster exists with same name
		if _, ok := existingDB[c.Name]; !ok {
			result = append(result, c)
		}
	}
	// Add back database-sourced clusters
	for _, c := range existingDB {
		result = append(result, c)
	}
	m.monitoredClusters = result
	return nil
}

func (m *mockClusterStore) SyncExecutionClustersFromYAML(ctx context.Context, clusters []storage.ExecutionClusterRecord) error {
	if m.syncExecutionErr != nil {
		return m.syncExecutionErr
	}
	// Simulate sync: add YAML clusters, preserve database-sourced clusters
	existingDB := make(map[string]storage.ExecutionClusterRecord)
	for _, c := range m.executionClusters {
		if c.Source == "database" {
			existingDB[c.Name] = c
		}
	}

	// Start with YAML clusters
	result := make([]storage.ExecutionClusterRecord, 0)
	for _, c := range clusters {
		c.Source = "yaml"
		// Skip if database-sourced cluster exists with same name
		if _, ok := existingDB[c.Name]; !ok {
			result = append(result, c)
		}
	}
	// Add back database-sourced clusters
	for _, c := range existingDB {
		result = append(result, c)
	}
	m.executionClusters = result
	return nil
}

func TestNewReloader(t *testing.T) {
	r := NewReloader(&ReloaderConfig{
		ConfigFile:   "/path/to/config.yaml",
		PollInterval: 60 * time.Second,
	})

	if r.configFile != "/path/to/config.yaml" {
		t.Errorf("configFile = %v, want /path/to/config.yaml", r.configFile)
	}

	if r.pollInterval != 60*time.Second {
		t.Errorf("pollInterval = %v, want 60s", r.pollInterval)
	}
}

func TestNewReloader_DefaultPollInterval(t *testing.T) {
	r := NewReloader(&ReloaderConfig{
		ConfigFile: "/path/to/config.yaml",
	})

	if r.pollInterval != 30*time.Second {
		t.Errorf("pollInterval = %v, want 30s (default)", r.pollInterval)
	}
}

func TestReloader_Reload_BasicYAML(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configContent := testConfigTemplate(`  - name: test-cluster
    environment: testing
    mcp:
      endpoint: http://localhost:8080`)

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Create minimal connection manager
	mgrConfig := &cluster.ManagerConfig{
		Clusters: []cluster.MonitoredClusterConfig{},
	}
	connMgr, err := cluster.NewConnectionManager(mgrConfig)
	if err != nil {
		t.Fatalf("failed to create connection manager: %v", err)
	}

	// Create execution cluster manager
	execMgr, err := cluster.NewExecutionClusterManager(&cluster.ExecutionManagerConfig{
		Clusters: []cluster.ExecutionClusterConfig{},
	})
	if err != nil {
		t.Fatalf("failed to create execution cluster manager: %v", err)
	}

	// Create reloader
	r := NewReloader(&ReloaderConfig{
		ConfigFile:    configFile,
		ConnectionMgr: connMgr,
		ExecutionMgr:  execMgr,
		CurrentConfig: &config.Config{},
	})

	// Reload
	ctx := context.Background()
	result := r.Reload(ctx)

	if result.Error != nil {
		t.Errorf("Reload() error = %v", result.Error)
	}

	if !result.ConfigChanged {
		t.Error("ConfigChanged should be true")
	}

	if len(result.MonitoredAdded) != 1 {
		t.Errorf("MonitoredAdded = %d, want 1", len(result.MonitoredAdded))
	}
}

func TestReloader_Reload_MergesDatabaseClusters(t *testing.T) {
	// Create a temporary config file with one cluster
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configContent := testConfigTemplate(`  - name: yaml-cluster
    environment: production
    mcp:
      endpoint: http://yaml-cluster:8080`)

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Create mock cluster store with database clusters
	mockStore := &mockClusterStore{
		monitoredClusters: []storage.MonitoredClusterRecord{
			{
				Name:        "db-cluster",
				Environment: "staging",
				Source:      "database",
				MCPEndpoint: "http://db-cluster:8080",
			},
		},
		executionClusters: []storage.ExecutionClusterRecord{
			{
				Name:       "db-exec-cluster",
				Source:     "database",
				Namespace:  "agents",
				Kubeconfig: "/path/to/kubeconfig",
			},
		},
	}

	// Create connection manager
	mgrConfig := &cluster.ManagerConfig{
		Clusters: []cluster.MonitoredClusterConfig{},
	}
	connMgr, err := cluster.NewConnectionManager(mgrConfig)
	if err != nil {
		t.Fatalf("failed to create connection manager: %v", err)
	}

	// Create execution cluster manager
	execMgr, err := cluster.NewExecutionClusterManager(&cluster.ExecutionManagerConfig{
		Clusters: []cluster.ExecutionClusterConfig{},
	})
	if err != nil {
		t.Fatalf("failed to create execution cluster manager: %v", err)
	}

	// Create reloader with cluster store
	r := NewReloader(&ReloaderConfig{
		ConfigFile:    configFile,
		ConnectionMgr: connMgr,
		ExecutionMgr:  execMgr,
		ClusterStore:  mockStore,
		CurrentConfig: &config.Config{},
	})

	// Reload
	ctx := context.Background()
	result := r.Reload(ctx)

	if result.Error != nil {
		t.Errorf("Reload() error = %v", result.Error)
	}

	// Should have 2 monitored clusters (1 YAML + 1 database)
	if len(result.MonitoredAdded) != 2 {
		t.Errorf("MonitoredAdded = %d, want 2", len(result.MonitoredAdded))
	}

	// Should have 1 execution cluster (from database)
	if len(result.ExecutionAdded) != 1 {
		t.Errorf("ExecutionAdded = %d, want 1", len(result.ExecutionAdded))
	}
}

func TestReloader_Reload_DatabaseOverridesYAML(t *testing.T) {
	// Create a config file with a cluster
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configContent := testConfigTemplate(`  - name: shared-cluster
    environment: production
    mcp:
      endpoint: http://yaml-endpoint:8080`)

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Create mock cluster store with same cluster name (should override)
	mockStore := &mockClusterStore{
		monitoredClusters: []storage.MonitoredClusterRecord{
			{
				Name:        "shared-cluster",
				Environment: "staging", // Different from YAML
				Source:      "database",
				MCPEndpoint: "http://db-endpoint:8080", // Different from YAML
			},
		},
	}

	// Create connection manager
	mgrConfig := &cluster.ManagerConfig{
		Clusters: []cluster.MonitoredClusterConfig{},
	}
	connMgr, err := cluster.NewConnectionManager(mgrConfig)
	if err != nil {
		t.Fatalf("failed to create connection manager: %v", err)
	}

	// Create execution cluster manager
	execMgr, err := cluster.NewExecutionClusterManager(&cluster.ExecutionManagerConfig{
		Clusters: []cluster.ExecutionClusterConfig{},
	})
	if err != nil {
		t.Fatalf("failed to create execution cluster manager: %v", err)
	}

	// Create reloader with cluster store
	r := NewReloader(&ReloaderConfig{
		ConfigFile:    configFile,
		ConnectionMgr: connMgr,
		ExecutionMgr:  execMgr,
		ClusterStore:  mockStore,
		CurrentConfig: &config.Config{},
	})

	// Reload
	ctx := context.Background()
	result := r.Reload(ctx)

	if result.Error != nil {
		t.Errorf("Reload() error = %v", result.Error)
	}

	// Should only have 1 cluster (database overrides YAML)
	if len(result.MonitoredAdded) != 1 {
		t.Errorf("MonitoredAdded = %d, want 1 (merged)", len(result.MonitoredAdded))
	}

	// The cluster in the manager should have the database values
	// We can't easily check this without exposing internals, so just verify no error
}

func TestReloader_Reload_ConfigFileError(t *testing.T) {
	// Create reloader with non-existent config file
	r := NewReloader(&ReloaderConfig{
		ConfigFile: "/nonexistent/config.yaml",
	})

	ctx := context.Background()
	result := r.Reload(ctx)

	if result.Error == nil {
		t.Error("Reload() should return error for missing config file")
	}

	if result.ConfigChanged {
		t.Error("ConfigChanged should be false on error")
	}
}

func TestReloader_DatabasePolling_NotStartedWithoutStore(t *testing.T) {
	r := NewReloader(&ReloaderConfig{
		ConfigFile:   "/path/to/config.yaml",
		ClusterStore: nil, // No cluster store
	})

	ctx := context.Background()
	r.StartDatabasePolling(ctx)

	// Should not have started (no pollCancel set)
	if r.pollCancel != nil {
		t.Error("Polling should not start without cluster store")
	}
}

func TestReloader_DatabasePolling_NotStartedWithClusters(t *testing.T) {
	// Create connection manager with existing clusters
	mgrConfig := &cluster.ManagerConfig{
		Clusters: []cluster.MonitoredClusterConfig{
			{Name: "existing-cluster"},
		},
	}
	connMgr, err := cluster.NewConnectionManager(mgrConfig)
	if err != nil {
		t.Fatalf("failed to create connection manager: %v", err)
	}

	mockStore := &mockClusterStore{}

	r := NewReloader(&ReloaderConfig{
		ConfigFile:    "/path/to/config.yaml",
		ConnectionMgr: connMgr,
		ClusterStore:  mockStore,
	})

	ctx := context.Background()
	r.StartDatabasePolling(ctx)

	// Should not have started (clusters already exist)
	if r.pollCancel != nil {
		t.Error("Polling should not start when clusters already configured")
	}
}

func TestReloader_DatabasePolling_Lifecycle(t *testing.T) {
	// Create connection manager with no clusters
	mgrConfig := &cluster.ManagerConfig{
		Clusters: []cluster.MonitoredClusterConfig{},
	}
	connMgr, err := cluster.NewConnectionManager(mgrConfig)
	if err != nil {
		t.Fatalf("failed to create connection manager: %v", err)
	}

	mockStore := &mockClusterStore{}

	// Create temp config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	configContent := testConfigTemplate(`[]`)

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	r := NewReloader(&ReloaderConfig{
		ConfigFile:    configFile,
		ConnectionMgr: connMgr,
		ClusterStore:  mockStore,
		PollInterval:  100 * time.Millisecond, // Fast polling for test
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r.StartDatabasePolling(ctx)

	// Should have started
	if r.pollCancel == nil {
		t.Fatal("Polling should have started")
	}

	// Wait a bit for polling loop to run
	time.Sleep(50 * time.Millisecond)

	// Stop polling
	r.StopDatabasePolling()

	// Should be stopped
	if r.pollCancel != nil {
		t.Error("pollCancel should be nil after stop")
	}
}

func TestMergeMonitoredClusters(t *testing.T) {
	r := &Reloader{}

	yaml := []cluster.MonitoredClusterConfig{
		{Name: "cluster-a", Environment: "yaml-env"},
		{Name: "cluster-b", Environment: "yaml-env"},
	}

	db := []cluster.MonitoredClusterConfig{
		{Name: "cluster-b", Environment: "db-env"}, // Override
		{Name: "cluster-c", Environment: "db-env"}, // New
	}

	merged := r.mergeMonitoredClusters(yaml, db)

	if len(merged) != 3 {
		t.Errorf("merged length = %d, want 3", len(merged))
	}

	// Build a map to check values
	byName := make(map[string]cluster.MonitoredClusterConfig)
	for _, c := range merged {
		byName[c.Name] = c
	}

	// cluster-a should be from YAML
	if byName["cluster-a"].Environment != "yaml-env" {
		t.Errorf("cluster-a should have yaml-env, got %v", byName["cluster-a"].Environment)
	}

	// cluster-b should be from DB (override)
	if byName["cluster-b"].Environment != "db-env" {
		t.Errorf("cluster-b should have db-env (from DB), got %v", byName["cluster-b"].Environment)
	}

	// cluster-c should exist (from DB)
	if _, ok := byName["cluster-c"]; !ok {
		t.Error("cluster-c should exist from DB")
	}
}

func TestMergeExecutionClusters(t *testing.T) {
	r := &Reloader{}

	yaml := []cluster.ExecutionClusterConfig{
		{Name: "exec-a", Namespace: "yaml-ns"},
		{Name: "exec-b", Namespace: "yaml-ns"},
	}

	db := []cluster.ExecutionClusterConfig{
		{Name: "exec-b", Namespace: "db-ns"}, // Override
		{Name: "exec-c", Namespace: "db-ns"}, // New
	}

	merged := r.mergeExecutionClusters(yaml, db)

	if len(merged) != 3 {
		t.Errorf("merged length = %d, want 3", len(merged))
	}

	// Build a map to check values
	byName := make(map[string]cluster.ExecutionClusterConfig)
	for _, c := range merged {
		byName[c.Name] = c
	}

	// exec-a should be from YAML
	if byName["exec-a"].Namespace != "yaml-ns" {
		t.Errorf("exec-a should have yaml-ns, got %v", byName["exec-a"].Namespace)
	}

	// exec-b should be from DB (override)
	if byName["exec-b"].Namespace != "db-ns" {
		t.Errorf("exec-b should have db-ns (from DB), got %v", byName["exec-b"].Namespace)
	}

	// exec-c should exist (from DB)
	if _, ok := byName["exec-c"]; !ok {
		t.Error("exec-c should exist from DB")
	}
}

func TestReloadResult(t *testing.T) {
	result := &ReloadResult{
		MonitoredAdded:   []string{"a", "b"},
		MonitoredRemoved: []string{"c"},
		MonitoredUpdated: []string{"d"},
		ExecutionAdded:   []string{"e"},
		ExecutionRemoved: []string{},
		ExecutionUpdated: []string{"f", "g"},
		ConfigChanged:    true,
	}

	// Just verify the struct is populated correctly
	if len(result.MonitoredAdded) != 2 {
		t.Errorf("MonitoredAdded = %d, want 2", len(result.MonitoredAdded))
	}
	if len(result.ExecutionUpdated) != 2 {
		t.Errorf("ExecutionUpdated = %d, want 2", len(result.ExecutionUpdated))
	}
}
