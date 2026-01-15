// Package sqlite provides SQLite implementation of cluster storage interfaces.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/randybias/nightcrier/internal/storage"
)

// Ensure Store implements ClusterStorage
var _ storage.ClusterStorage = (*Store)(nil)

// ListMonitoredClusters returns all monitored clusters from the database.
func (s *Store) ListMonitoredClusters(ctx context.Context) ([]storage.MonitoredClusterRecord, error) {
	query := `
		SELECT name, environment, labels, mcp_endpoint, mcp_api_key,
		       triage_enabled, target_kubeconfig, allow_secrets_access,
		       execution_cluster, created_at, updated_at, source,
		       connection_status, unreachable, unreachable_reason, last_status_check, last_error
		FROM monitored_clusters
		ORDER BY name`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query monitored clusters: %w", err)
	}
	defer rows.Close()

	var clusters []storage.MonitoredClusterRecord
	for rows.Next() {
		var c storage.MonitoredClusterRecord
		var labelsJSON sql.NullString
		var environment, mcpAPIKey, targetKubeconfig, executionCluster sql.NullString
		var triageEnabled, allowSecretsAccess, unreachable int
		var connectionStatus, unreachableReason, lastError sql.NullString
		var lastStatusCheck sql.NullTime

		err := rows.Scan(
			&c.Name, &environment, &labelsJSON, &c.MCPEndpoint, &mcpAPIKey,
			&triageEnabled, &targetKubeconfig, &allowSecretsAccess,
			&executionCluster, &c.CreatedAt, &c.UpdatedAt, &c.Source,
			&connectionStatus, &unreachable, &unreachableReason, &lastStatusCheck, &lastError,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan monitored cluster: %w", err)
		}

		c.Environment = environment.String
		c.MCPAPIKey = mcpAPIKey.String
		c.TriageEnabled = triageEnabled != 0
		c.TargetKubeconfig = targetKubeconfig.String
		c.AllowSecretsAccess = allowSecretsAccess != 0
		c.ExecutionCluster = executionCluster.String
		c.ConnectionStatus = connectionStatus.String
		c.Unreachable = unreachable != 0
		c.UnreachableReason = unreachableReason.String
		c.LastError = lastError.String
		if lastStatusCheck.Valid {
			c.LastStatusCheck = &lastStatusCheck.Time
		}

		if labelsJSON.Valid && labelsJSON.String != "" {
			if err := json.Unmarshal([]byte(labelsJSON.String), &c.Labels); err != nil {
				return nil, fmt.Errorf("failed to unmarshal labels for cluster %s: %w", c.Name, err)
			}
		}

		clusters = append(clusters, c)
	}

	return clusters, rows.Err()
}

// GetMonitoredCluster returns a single monitored cluster by name.
func (s *Store) GetMonitoredCluster(ctx context.Context, name string) (*storage.MonitoredClusterRecord, error) {
	query := `
		SELECT name, environment, labels, mcp_endpoint, mcp_api_key,
		       triage_enabled, target_kubeconfig, allow_secrets_access,
		       execution_cluster, created_at, updated_at, source,
		       connection_status, unreachable, unreachable_reason, last_status_check, last_error
		FROM monitored_clusters
		WHERE name = ?`

	var c storage.MonitoredClusterRecord
	var labelsJSON sql.NullString
	var environment, mcpAPIKey, targetKubeconfig, executionCluster sql.NullString
	var triageEnabled, allowSecretsAccess, unreachable int
	var connectionStatus, unreachableReason, lastError sql.NullString
	var lastStatusCheck sql.NullTime

	err := s.db.QueryRowContext(ctx, query, name).Scan(
		&c.Name, &environment, &labelsJSON, &c.MCPEndpoint, &mcpAPIKey,
		&triageEnabled, &targetKubeconfig, &allowSecretsAccess,
		&executionCluster, &c.CreatedAt, &c.UpdatedAt, &c.Source,
		&connectionStatus, &unreachable, &unreachableReason, &lastStatusCheck, &lastError,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get monitored cluster %s: %w", name, err)
	}

	c.Environment = environment.String
	c.MCPAPIKey = mcpAPIKey.String
	c.TriageEnabled = triageEnabled != 0
	c.TargetKubeconfig = targetKubeconfig.String
	c.AllowSecretsAccess = allowSecretsAccess != 0
	c.ExecutionCluster = executionCluster.String
	c.ConnectionStatus = connectionStatus.String
	c.Unreachable = unreachable != 0
	c.UnreachableReason = unreachableReason.String
	c.LastError = lastError.String
	if lastStatusCheck.Valid {
		c.LastStatusCheck = &lastStatusCheck.Time
	}

	if labelsJSON.Valid && labelsJSON.String != "" {
		if err := json.Unmarshal([]byte(labelsJSON.String), &c.Labels); err != nil {
			return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
		}
	}

	return &c, nil
}

// UpsertMonitoredCluster inserts or updates a monitored cluster.
func (s *Store) UpsertMonitoredCluster(ctx context.Context, cluster *storage.MonitoredClusterRecord) error {
	var labelsJSON []byte
	var err error
	if len(cluster.Labels) > 0 {
		labelsJSON, err = json.Marshal(cluster.Labels)
		if err != nil {
			return fmt.Errorf("failed to marshal labels: %w", err)
		}
	}

	now := time.Now().UTC()
	triageEnabled := 0
	if cluster.TriageEnabled {
		triageEnabled = 1
	}
	allowSecretsAccess := 0
	if cluster.AllowSecretsAccess {
		allowSecretsAccess = 1
	}

	query := `
		INSERT INTO monitored_clusters (
			name, environment, labels, mcp_endpoint, mcp_api_key,
			triage_enabled, target_kubeconfig, allow_secrets_access,
			execution_cluster, created_at, updated_at, source
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			environment = excluded.environment,
			labels = excluded.labels,
			mcp_endpoint = excluded.mcp_endpoint,
			mcp_api_key = excluded.mcp_api_key,
			triage_enabled = excluded.triage_enabled,
			target_kubeconfig = excluded.target_kubeconfig,
			allow_secrets_access = excluded.allow_secrets_access,
			execution_cluster = excluded.execution_cluster,
			updated_at = excluded.updated_at,
			source = excluded.source`

	_, err = s.db.ExecContext(ctx, query,
		cluster.Name, cluster.Environment, string(labelsJSON), cluster.MCPEndpoint, cluster.MCPAPIKey,
		triageEnabled, cluster.TargetKubeconfig, allowSecretsAccess,
		cluster.ExecutionCluster, now, now, cluster.Source,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert monitored cluster %s: %w", cluster.Name, err)
	}

	return nil
}

// DeleteMonitoredCluster removes a monitored cluster by name.
func (s *Store) DeleteMonitoredCluster(ctx context.Context, name string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM monitored_clusters WHERE name = ?", name)
	if err != nil {
		return fmt.Errorf("failed to delete monitored cluster %s: %w", name, err)
	}
	return nil
}

// ListExecutionClusters returns all execution clusters from the database.
func (s *Store) ListExecutionClusters(ctx context.Context) ([]storage.ExecutionClusterRecord, error) {
	query := `
		SELECT name, kubeconfig, namespace, runner_image, image_pull_policy,
		       timeout, memory_limit, cpu_limit, cleanup_ttl, max_concurrent_agents,
		       created_at, updated_at, source
		FROM execution_clusters
		ORDER BY name`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query execution clusters: %w", err)
	}
	defer rows.Close()

	var clusters []storage.ExecutionClusterRecord
	for rows.Next() {
		var c storage.ExecutionClusterRecord
		err := rows.Scan(
			&c.Name, &c.Kubeconfig, &c.Namespace, &c.RunnerImage, &c.ImagePullPolicy,
			&c.Timeout, &c.MemoryLimit, &c.CPULimit, &c.CleanupTTL, &c.MaxConcurrentAgents,
			&c.CreatedAt, &c.UpdatedAt, &c.Source,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan execution cluster: %w", err)
		}
		clusters = append(clusters, c)
	}

	return clusters, rows.Err()
}

// GetExecutionCluster returns a single execution cluster by name.
func (s *Store) GetExecutionCluster(ctx context.Context, name string) (*storage.ExecutionClusterRecord, error) {
	query := `
		SELECT name, kubeconfig, namespace, runner_image, image_pull_policy,
		       timeout, memory_limit, cpu_limit, cleanup_ttl, max_concurrent_agents,
		       created_at, updated_at, source
		FROM execution_clusters
		WHERE name = ?`

	var c storage.ExecutionClusterRecord
	err := s.db.QueryRowContext(ctx, query, name).Scan(
		&c.Name, &c.Kubeconfig, &c.Namespace, &c.RunnerImage, &c.ImagePullPolicy,
		&c.Timeout, &c.MemoryLimit, &c.CPULimit, &c.CleanupTTL, &c.MaxConcurrentAgents,
		&c.CreatedAt, &c.UpdatedAt, &c.Source,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get execution cluster %s: %w", name, err)
	}

	return &c, nil
}

// UpsertExecutionCluster inserts or updates an execution cluster.
func (s *Store) UpsertExecutionCluster(ctx context.Context, cluster *storage.ExecutionClusterRecord) error {
	now := time.Now().UTC()

	query := `
		INSERT INTO execution_clusters (
			name, kubeconfig, namespace, runner_image, image_pull_policy,
			timeout, memory_limit, cpu_limit, cleanup_ttl, max_concurrent_agents,
			created_at, updated_at, source
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			kubeconfig = excluded.kubeconfig,
			namespace = excluded.namespace,
			runner_image = excluded.runner_image,
			image_pull_policy = excluded.image_pull_policy,
			timeout = excluded.timeout,
			memory_limit = excluded.memory_limit,
			cpu_limit = excluded.cpu_limit,
			cleanup_ttl = excluded.cleanup_ttl,
			max_concurrent_agents = excluded.max_concurrent_agents,
			updated_at = excluded.updated_at,
			source = excluded.source`

	_, err := s.db.ExecContext(ctx, query,
		cluster.Name, cluster.Kubeconfig, cluster.Namespace, cluster.RunnerImage, cluster.ImagePullPolicy,
		cluster.Timeout, cluster.MemoryLimit, cluster.CPULimit, cluster.CleanupTTL, cluster.MaxConcurrentAgents,
		now, now, cluster.Source,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert execution cluster %s: %w", cluster.Name, err)
	}

	return nil
}

// DeleteExecutionCluster removes an execution cluster by name.
func (s *Store) DeleteExecutionCluster(ctx context.Context, name string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM execution_clusters WHERE name = ?", name)
	if err != nil {
		return fmt.Errorf("failed to delete execution cluster %s: %w", name, err)
	}
	return nil
}

// SyncMonitoredClustersFromYAML syncs YAML-defined clusters to the database.
// YAML clusters are inserted with source='yaml'. Database-sourced clusters are NOT overwritten.
func (s *Store) SyncMonitoredClustersFromYAML(ctx context.Context, clusters []storage.MonitoredClusterRecord) error {
	existing, err := s.ListMonitoredClusters(ctx)
	if err != nil {
		return err
	}

	// Build maps for efficient lookup
	existingByName := make(map[string]storage.MonitoredClusterRecord)
	for _, e := range existing {
		existingByName[e.Name] = e
	}

	yamlClusterNames := make(map[string]bool)
	for _, c := range clusters {
		yamlClusterNames[c.Name] = true
	}

	// Delete YAML-sourced clusters that are no longer in config
	for _, e := range existing {
		if e.Source == "yaml" && !yamlClusterNames[e.Name] {
			if err := s.DeleteMonitoredCluster(ctx, e.Name); err != nil {
				return err
			}
		}
	}

	// Upsert clusters from YAML, but skip if database-sourced cluster exists
	for i := range clusters {
		if existing, ok := existingByName[clusters[i].Name]; ok && existing.Source == "database" {
			// Don't overwrite database-sourced clusters
			continue
		}
		clusters[i].Source = "yaml"
		if err := s.UpsertMonitoredCluster(ctx, &clusters[i]); err != nil {
			return err
		}
	}

	return nil
}

// SyncExecutionClustersFromYAML syncs YAML-defined execution clusters to the database.
// YAML clusters are inserted with source='yaml'. Database-sourced clusters are NOT overwritten.
func (s *Store) SyncExecutionClustersFromYAML(ctx context.Context, clusters []storage.ExecutionClusterRecord) error {
	existing, err := s.ListExecutionClusters(ctx)
	if err != nil {
		return err
	}

	// Build maps for efficient lookup
	existingByName := make(map[string]storage.ExecutionClusterRecord)
	for _, e := range existing {
		existingByName[e.Name] = e
	}

	yamlClusterNames := make(map[string]bool)
	for _, c := range clusters {
		yamlClusterNames[c.Name] = true
	}

	// Delete YAML-sourced clusters that are no longer in config
	for _, e := range existing {
		if e.Source == "yaml" && !yamlClusterNames[e.Name] {
			if err := s.DeleteExecutionCluster(ctx, e.Name); err != nil {
				return err
			}
		}
	}

	// Upsert clusters from YAML, but skip if database-sourced cluster exists
	for i := range clusters {
		if existing, ok := existingByName[clusters[i].Name]; ok && existing.Source == "database" {
			// Don't overwrite database-sourced clusters
			continue
		}
		clusters[i].Source = "yaml"
		if err := s.UpsertExecutionCluster(ctx, &clusters[i]); err != nil {
			return err
		}
	}

	return nil
}

// UpdateMonitoredClusterStatus updates only the runtime reachability fields for a cluster.
// This is called frequently by the connection manager to track connection health.
func (s *Store) UpdateMonitoredClusterStatus(ctx context.Context, status *storage.ClusterStatusUpdate) error {
	now := time.Now().UTC()

	// Convert unreachable bool to integer for SQLite
	unreachable := 0
	if status.Unreachable {
		unreachable = 1
	}

	query := `
		UPDATE monitored_clusters
		SET connection_status = ?,
		    unreachable = ?,
		    unreachable_reason = ?,
		    last_error = ?,
		    last_status_check = ?
		WHERE name = ?`

	result, err := s.db.ExecContext(ctx, query,
		status.ConnectionStatus,
		unreachable,
		status.UnreachableReason,
		status.LastError,
		now,
		status.Name,
	)
	if err != nil {
		return fmt.Errorf("failed to update cluster status for %s: %w", status.Name, err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// Cluster might not exist in DB yet (YAML-only config)
		// This is OK - status will be persisted when cluster is synced
		return nil
	}

	return nil
}
