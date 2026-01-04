// Package reload provides configuration reload functionality for runtime updates.
package reload

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/randybias/nightcrier/internal/cluster"
	"github.com/randybias/nightcrier/internal/config"
	"github.com/randybias/nightcrier/internal/storage"
)

// ClusterStore is an optional interface for loading clusters from database.
// If provided, YAML clusters are synced to database, then all clusters are loaded from database.
type ClusterStore interface {
	ListMonitoredClusters(ctx context.Context) ([]storage.MonitoredClusterRecord, error)
	ListExecutionClusters(ctx context.Context) ([]storage.ExecutionClusterRecord, error)
	SyncMonitoredClustersFromYAML(ctx context.Context, clusters []storage.MonitoredClusterRecord) error
	SyncExecutionClustersFromYAML(ctx context.Context, clusters []storage.ExecutionClusterRecord) error
}

// Reloader coordinates configuration reload across all components.
// It handles:
//   - Re-reading YAML configuration
//   - Merging clusters from database (if ClusterStore provided)
//   - Applying changes to ConnectionManager and ExecutionClusterManager
//   - Periodic database polling when no clusters configured
type Reloader struct {
	// Configuration file path
	configFile string

	// Managers to reload
	connectionMgr *cluster.ConnectionManager
	executionMgr  *cluster.ExecutionClusterManager

	// Optional database storage for dynamic clusters
	clusterStore ClusterStore

	// Current configuration
	currentConfig *config.Config

	// Database polling
	pollInterval time.Duration
	pollCancel   context.CancelFunc
	pollWg       sync.WaitGroup

	// Protects reload operations
	mu sync.Mutex
}

// ReloaderConfig holds configuration for creating a Reloader.
type ReloaderConfig struct {
	// ConfigFile is the path to the YAML configuration file
	ConfigFile string

	// ConnectionManager handles monitored cluster connections
	ConnectionMgr *cluster.ConnectionManager

	// ExecutionManager handles execution cluster configurations
	ExecutionMgr *cluster.ExecutionClusterManager

	// ClusterStore is optional database storage for dynamic clusters
	ClusterStore ClusterStore

	// CurrentConfig is the initial configuration
	CurrentConfig *config.Config

	// PollInterval is how often to check the database for new clusters (default: 30s)
	PollInterval time.Duration
}

// ReloadResult contains the outcome of a reload operation.
type ReloadResult struct {
	// MonitoredClusters changes
	MonitoredAdded   []string
	MonitoredRemoved []string
	MonitoredUpdated []string

	// ExecutionClusters changes
	ExecutionAdded   []string
	ExecutionRemoved []string
	ExecutionUpdated []string

	// ConfigChanged indicates if the YAML config changed
	ConfigChanged bool

	// Error if reload failed
	Error error
}

// NewReloader creates a new configuration reloader.
func NewReloader(cfg *ReloaderConfig) *Reloader {
	pollInterval := cfg.PollInterval
	if pollInterval == 0 {
		pollInterval = 30 * time.Second
	}

	return &Reloader{
		configFile:    cfg.ConfigFile,
		connectionMgr: cfg.ConnectionMgr,
		executionMgr:  cfg.ExecutionMgr,
		clusterStore:  cfg.ClusterStore,
		currentConfig: cfg.CurrentConfig,
		pollInterval:  pollInterval,
	}
}

// SetClusterStore sets the cluster store for dynamic cluster loading.
// This allows setting the cluster store after initialization, which is needed
// because the stateStore is initialized after the reloader is created.
func (r *Reloader) SetClusterStore(store ClusterStore) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clusterStore = store
}

// Reload performs a full configuration reload.
// It re-reads the YAML config, syncs to database, then loads all clusters from database.
func (r *Reloader) Reload(ctx context.Context) *ReloadResult {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := &ReloadResult{}

	slog.Info("starting configuration reload")

	// Step 1: Re-read YAML configuration
	newConfig, err := config.LoadWithConfigFile(r.configFile)
	if err != nil {
		result.Error = fmt.Errorf("failed to reload configuration: %w", err)
		slog.Error("configuration reload failed", "error", err)
		return result
	}

	var monitoredClusters []cluster.MonitoredClusterConfig
	var executionClusters []cluster.ExecutionClusterConfig

	if r.clusterStore != nil {
		// Step 2: Sync YAML clusters to database (database becomes source of truth)
		if err := r.syncYAMLToDatabase(ctx, newConfig); err != nil {
			slog.Warn("failed to sync YAML clusters to database", "error", err)
			// Fall back to YAML-only mode
			monitoredClusters = newConfig.MonitoredClusters
			executionClusters = r.convertExecutionClusters(newConfig.ExecutionClusters)
		} else {
			// Step 3: Load all clusters from database (includes both YAML and database-sourced)
			monitoredClusters, executionClusters, err = r.loadClustersFromDatabase(ctx)
			if err != nil {
				slog.Warn("failed to load clusters from database after sync", "error", err)
				// Fall back to YAML-only mode
				monitoredClusters = newConfig.MonitoredClusters
				executionClusters = r.convertExecutionClusters(newConfig.ExecutionClusters)
			}
		}
	} else {
		// No database - use YAML directly
		monitoredClusters = newConfig.MonitoredClusters
		executionClusters = r.convertExecutionClusters(newConfig.ExecutionClusters)
	}

	// Step 4: Reload ConnectionManager (monitored clusters)
	if r.connectionMgr != nil {
		connResult, err := r.connectionMgr.Reload(ctx, monitoredClusters)
		if err != nil {
			result.Error = fmt.Errorf("failed to reload connection manager: %w", err)
			slog.Error("connection manager reload failed", "error", err)
			return result
		}
		result.MonitoredAdded = connResult.Added
		result.MonitoredRemoved = connResult.Removed
		result.MonitoredUpdated = connResult.Updated
	}

	// Step 5: Reload ExecutionClusterManager
	if r.executionMgr != nil {
		execResult, err := r.executionMgr.Reload(executionClusters)
		if err != nil {
			result.Error = fmt.Errorf("failed to reload execution cluster manager: %w", err)
			slog.Error("execution cluster manager reload failed", "error", err)
			return result
		}
		result.ExecutionAdded = execResult.Added
		result.ExecutionRemoved = execResult.Removed
		result.ExecutionUpdated = execResult.Updated
	}

	// Update current config
	r.currentConfig = newConfig
	result.ConfigChanged = true

	// Log summary
	r.logReloadSummary(result)

	return result
}

// convertExecutionClusters converts config execution clusters to cluster package type.
func (r *Reloader) convertExecutionClusters(configs []config.ExecutionClusterConfig) []cluster.ExecutionClusterConfig {
	var result []cluster.ExecutionClusterConfig
	for _, ec := range configs {
		result = append(result, cluster.ExecutionClusterConfig{
			Name:                ec.Name,
			KubeconfigPath:      ec.KubeconfigPath,
			Namespace:           ec.Namespace,
			RunnerImage:         ec.RunnerImage,
			ImagePullPolicy:     ec.ImagePullPolicy,
			Timeout:             ec.Timeout,
			MemoryLimit:         ec.MemoryLimit,
			CPULimit:            ec.CPULimit,
			CleanupTTL:          ec.CleanupTTL,
			MaxConcurrentAgents: ec.MaxConcurrentAgents,
		})
	}
	return result
}

// syncYAMLToDatabase syncs YAML-defined clusters to the database.
// This ensures the database is the single source of truth.
// Database-sourced clusters are not overwritten by YAML.
// Warnings are logged when YAML config differs from database-sourced clusters.
func (r *Reloader) syncYAMLToDatabase(ctx context.Context, cfg *config.Config) error {
	// First, check for conflicts between YAML and database-sourced clusters
	if err := r.warnOnClusterConflicts(ctx, cfg); err != nil {
		slog.Warn("failed to check for cluster conflicts", "error", err)
	}

	// Convert monitored clusters to storage records
	var monitoredRecords []storage.MonitoredClusterRecord
	for _, mc := range cfg.MonitoredClusters {
		// Read kubeconfig file content if triage is enabled
		var kubeconfigContent string
		if mc.Triage.Enabled && mc.Triage.Kubeconfig != "" {
			// For YAML clusters, store the path (not content) since file access is local
			kubeconfigContent = mc.Triage.Kubeconfig
		}

		rec := storage.MonitoredClusterRecord{
			Name:               mc.Name,
			Environment:        mc.Environment,
			Labels:             mc.Labels,
			MCPEndpoint:        mc.MCP.Endpoint,
			MCPAPIKey:          mc.MCP.APIKey,
			TriageEnabled:      mc.Triage.Enabled,
			TargetKubeconfig:   kubeconfigContent,
			AllowSecretsAccess: mc.Triage.AllowSecretsAccess,
			ExecutionCluster:   mc.Triage.ExecutionCluster,
			Source:             "yaml",
		}
		monitoredRecords = append(monitoredRecords, rec)
	}

	// Convert execution clusters to storage records
	var executionRecords []storage.ExecutionClusterRecord
	for _, ec := range cfg.ExecutionClusters {
		rec := storage.ExecutionClusterRecord{
			Name:                ec.Name,
			Kubeconfig:          ec.KubeconfigPath, // Store path for YAML clusters
			Namespace:           ec.Namespace,
			RunnerImage:         ec.RunnerImage,
			ImagePullPolicy:     ec.ImagePullPolicy,
			Timeout:             ec.Timeout,
			MemoryLimit:         ec.MemoryLimit,
			CPULimit:            ec.CPULimit,
			CleanupTTL:          ec.CleanupTTL,
			MaxConcurrentAgents: ec.MaxConcurrentAgents,
			Source:              "yaml",
		}
		executionRecords = append(executionRecords, rec)
	}

	// Sync monitored clusters
	if err := r.clusterStore.SyncMonitoredClustersFromYAML(ctx, monitoredRecords); err != nil {
		return fmt.Errorf("failed to sync monitored clusters: %w", err)
	}
	slog.Debug("synced monitored clusters from YAML to database", "count", len(monitoredRecords))

	// Sync execution clusters
	if err := r.clusterStore.SyncExecutionClustersFromYAML(ctx, executionRecords); err != nil {
		return fmt.Errorf("failed to sync execution clusters: %w", err)
	}
	slog.Debug("synced execution clusters from YAML to database", "count", len(executionRecords))

	return nil
}

// warnOnClusterConflicts checks for differences between YAML and database-sourced clusters
// and logs warnings when database takes precedence over differing YAML config.
func (r *Reloader) warnOnClusterConflicts(ctx context.Context, cfg *config.Config) error {
	// Load existing clusters from database
	dbMonitored, err := r.clusterStore.ListMonitoredClusters(ctx)
	if err != nil {
		return fmt.Errorf("failed to list monitored clusters: %w", err)
	}

	dbExecution, err := r.clusterStore.ListExecutionClusters(ctx)
	if err != nil {
		return fmt.Errorf("failed to list execution clusters: %w", err)
	}

	// Build maps of database-sourced clusters
	dbMonitoredMap := make(map[string]storage.MonitoredClusterRecord)
	for _, c := range dbMonitored {
		if c.Source == "database" {
			dbMonitoredMap[c.Name] = c
		}
	}

	dbExecutionMap := make(map[string]storage.ExecutionClusterRecord)
	for _, c := range dbExecution {
		if c.Source == "database" {
			dbExecutionMap[c.Name] = c
		}
	}

	// Check monitored clusters for conflicts
	for _, yamlCluster := range cfg.MonitoredClusters {
		if dbCluster, exists := dbMonitoredMap[yamlCluster.Name]; exists {
			// Database-sourced cluster exists with same name - check for differences
			var diffs []string

			if yamlCluster.MCP.Endpoint != dbCluster.MCPEndpoint {
				diffs = append(diffs, fmt.Sprintf("mcp.endpoint: yaml=%q db=%q", yamlCluster.MCP.Endpoint, dbCluster.MCPEndpoint))
			}
			if yamlCluster.Environment != dbCluster.Environment {
				diffs = append(diffs, fmt.Sprintf("environment: yaml=%q db=%q", yamlCluster.Environment, dbCluster.Environment))
			}
			if yamlCluster.Triage.Enabled != dbCluster.TriageEnabled {
				diffs = append(diffs, fmt.Sprintf("triage.enabled: yaml=%v db=%v", yamlCluster.Triage.Enabled, dbCluster.TriageEnabled))
			}
			if yamlCluster.Triage.ExecutionCluster != dbCluster.ExecutionCluster {
				diffs = append(diffs, fmt.Sprintf("triage.execution_cluster: yaml=%q db=%q", yamlCluster.Triage.ExecutionCluster, dbCluster.ExecutionCluster))
			}

			if len(diffs) > 0 {
				slog.Warn("YAML config differs from database for monitored cluster - database takes precedence",
					"cluster", yamlCluster.Name,
					"differences", diffs,
					"hint", "update database or remove database entry to use YAML config")
			}
		}
	}

	// Check execution clusters for conflicts
	for _, yamlCluster := range cfg.ExecutionClusters {
		if dbCluster, exists := dbExecutionMap[yamlCluster.Name]; exists {
			// Database-sourced cluster exists with same name - check for differences
			var diffs []string

			if yamlCluster.Namespace != dbCluster.Namespace {
				diffs = append(diffs, fmt.Sprintf("namespace: yaml=%q db=%q", yamlCluster.Namespace, dbCluster.Namespace))
			}
			if yamlCluster.RunnerImage != dbCluster.RunnerImage {
				diffs = append(diffs, fmt.Sprintf("runner_image: yaml=%q db=%q", yamlCluster.RunnerImage, dbCluster.RunnerImage))
			}
			if yamlCluster.KubeconfigPath != dbCluster.Kubeconfig {
				diffs = append(diffs, fmt.Sprintf("kubeconfig_path: yaml=%q db=%q", yamlCluster.KubeconfigPath, dbCluster.Kubeconfig))
			}
			if yamlCluster.Timeout != dbCluster.Timeout {
				diffs = append(diffs, fmt.Sprintf("timeout: yaml=%d db=%d", yamlCluster.Timeout, dbCluster.Timeout))
			}

			if len(diffs) > 0 {
				slog.Warn("YAML config differs from database for execution cluster - database takes precedence",
					"cluster", yamlCluster.Name,
					"differences", diffs,
					"hint", "update database or remove database entry to use YAML config")
			}
		}
	}

	return nil
}

// StartDatabasePolling starts periodic database polling for new clusters.
// This is useful when starting with zero clusters configured.
// Polling stops automatically when clusters are found or Stop is called.
func (r *Reloader) StartDatabasePolling(ctx context.Context) {
	if r.clusterStore == nil {
		slog.Debug("database polling not started: no cluster store configured")
		return
	}

	// Check if we have clusters already
	if r.connectionMgr != nil && len(r.connectionMgr.GetClusterNames()) > 0 {
		slog.Debug("database polling not started: clusters already configured")
		return
	}

	pollCtx, cancel := context.WithCancel(ctx)
	r.pollCancel = cancel

	r.pollWg.Add(1)
	go r.pollDatabase(pollCtx)

	slog.Info("database polling started",
		"interval", r.pollInterval,
		"reason", "no clusters configured")
}

// StopDatabasePolling stops the periodic database polling.
func (r *Reloader) StopDatabasePolling() {
	if r.pollCancel != nil {
		r.pollCancel()
		r.pollWg.Wait()
		r.pollCancel = nil
		slog.Info("database polling stopped")
	}
}

// pollDatabase periodically checks the database for new clusters.
func (r *Reloader) pollDatabase(ctx context.Context) {
	defer r.pollWg.Done()

	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()

	slog.Info("database polling loop started", "interval", r.pollInterval)

	for {
		select {
		case <-ctx.Done():
			slog.Debug("database polling loop exiting", "reason", "context cancelled")
			return
		case <-ticker.C:
			slog.Debug("database polling: checking for clusters")

			// Try to load clusters from database
			result := r.Reload(ctx)
			if result.Error != nil {
				slog.Warn("database poll reload failed", "error", result.Error)
				continue
			}

			// Check if we found any clusters
			totalAdded := len(result.MonitoredAdded) + len(result.ExecutionAdded)
			if totalAdded > 0 {
				slog.Info("database polling found clusters, stopping polling",
					"monitored_added", len(result.MonitoredAdded),
					"execution_added", len(result.ExecutionAdded))
				return
			}
		}
	}
}

// loadClustersFromDatabase loads ALL cluster configurations from the database.
// This includes both YAML-sourced and database-sourced clusters.
func (r *Reloader) loadClustersFromDatabase(ctx context.Context) ([]cluster.MonitoredClusterConfig, []cluster.ExecutionClusterConfig, error) {
	var monitoredClusters []cluster.MonitoredClusterConfig
	var executionClusters []cluster.ExecutionClusterConfig

	// Load all monitored clusters
	dbMonitored, err := r.clusterStore.ListMonitoredClusters(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list monitored clusters: %w", err)
	}

	for _, rec := range dbMonitored {
		cfg := cluster.MonitoredClusterConfig{
			Name:        rec.Name,
			Environment: rec.Environment,
			Labels:      rec.Labels,
			MCP: cluster.MCPConfig{
				Endpoint: rec.MCPEndpoint,
				APIKey:   rec.MCPAPIKey,
			},
			Triage: cluster.TriageConfig{
				Enabled:            rec.TriageEnabled,
				Kubeconfig:         rec.TargetKubeconfig,
				AllowSecretsAccess: rec.AllowSecretsAccess,
				ExecutionCluster:   rec.ExecutionCluster,
			},
		}
		monitoredClusters = append(monitoredClusters, cfg)
	}

	// Load all execution clusters
	dbExecution, err := r.clusterStore.ListExecutionClusters(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list execution clusters: %w", err)
	}

	for _, rec := range dbExecution {
		cfg := cluster.ExecutionClusterConfig{
			Name:                rec.Name,
			KubeconfigPath:      rec.Kubeconfig,
			Namespace:           rec.Namespace,
			RunnerImage:         rec.RunnerImage,
			ImagePullPolicy:     rec.ImagePullPolicy,
			Timeout:             rec.Timeout,
			MemoryLimit:         rec.MemoryLimit,
			CPULimit:            rec.CPULimit,
			CleanupTTL:          rec.CleanupTTL,
			MaxConcurrentAgents: rec.MaxConcurrentAgents,
		}
		executionClusters = append(executionClusters, cfg)
	}

	slog.Debug("loaded clusters from database",
		"monitored", len(monitoredClusters),
		"execution", len(executionClusters))

	return monitoredClusters, executionClusters, nil
}

// mergeMonitoredClusters merges YAML and database monitored clusters.
// Database clusters take precedence (override YAML clusters with same name).
func (r *Reloader) mergeMonitoredClusters(yaml []cluster.MonitoredClusterConfig, db []cluster.MonitoredClusterConfig) []cluster.MonitoredClusterConfig {
	// Build map from YAML clusters
	merged := make(map[string]cluster.MonitoredClusterConfig)
	for _, c := range yaml {
		merged[c.Name] = c
	}

	// Database clusters override
	for _, c := range db {
		merged[c.Name] = c
		slog.Debug("database cluster overrides YAML", "name", c.Name)
	}

	// Convert back to slice
	result := make([]cluster.MonitoredClusterConfig, 0, len(merged))
	for _, c := range merged {
		result = append(result, c)
	}

	return result
}

// mergeExecutionClusters merges YAML and database execution clusters.
// Database clusters take precedence (override YAML clusters with same name).
func (r *Reloader) mergeExecutionClusters(yaml []cluster.ExecutionClusterConfig, db []cluster.ExecutionClusterConfig) []cluster.ExecutionClusterConfig {
	// Build map from YAML clusters
	merged := make(map[string]cluster.ExecutionClusterConfig)
	for _, c := range yaml {
		merged[c.Name] = c
	}

	// Database clusters override
	for _, c := range db {
		merged[c.Name] = c
		slog.Debug("database execution cluster overrides YAML", "name", c.Name)
	}

	// Convert back to slice
	result := make([]cluster.ExecutionClusterConfig, 0, len(merged))
	for _, c := range merged {
		result = append(result, c)
	}

	return result
}

// logReloadSummary logs a summary of what changed during reload.
func (r *Reloader) logReloadSummary(result *ReloadResult) {
	monitoredChanges := len(result.MonitoredAdded) + len(result.MonitoredRemoved) + len(result.MonitoredUpdated)
	executionChanges := len(result.ExecutionAdded) + len(result.ExecutionRemoved) + len(result.ExecutionUpdated)

	if monitoredChanges == 0 && executionChanges == 0 {
		slog.Info("configuration reload complete: no changes")
		return
	}

	slog.Info("configuration reload complete",
		"monitored_added", len(result.MonitoredAdded),
		"monitored_removed", len(result.MonitoredRemoved),
		"monitored_updated", len(result.MonitoredUpdated),
		"execution_added", len(result.ExecutionAdded),
		"execution_removed", len(result.ExecutionRemoved),
		"execution_updated", len(result.ExecutionUpdated))

	// Log details for added clusters
	for _, name := range result.MonitoredAdded {
		slog.Info("monitored cluster added", "name", name)
	}
	for _, name := range result.ExecutionAdded {
		slog.Info("execution cluster added", "name", name)
	}

	// Log details for removed clusters
	for _, name := range result.MonitoredRemoved {
		slog.Info("monitored cluster removed", "name", name)
	}
	for _, name := range result.ExecutionRemoved {
		slog.Info("execution cluster removed", "name", name)
	}
}
