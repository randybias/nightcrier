// Package cluster provides cluster management functionality.
package cluster

import (
	"fmt"
	"log/slog"
	"sync"
)

// ExecutionClusterConfig defines a Kubernetes cluster where agent Jobs run.
// This mirrors config.ExecutionClusterConfig to avoid import cycles.
type ExecutionClusterConfig struct {
	// Name is a unique identifier for this execution cluster (required)
	Name string

	// KubeconfigPath is the path to the kubeconfig file for this cluster (required)
	KubeconfigPath string

	// Namespace where Jobs and ConfigMaps are created
	Namespace string

	// RunnerImage is the container image for the agent runner
	RunnerImage string

	// ImagePullPolicy: Always, Never, IfNotPresent
	ImagePullPolicy string

	// Timeout is the Job timeout in seconds
	Timeout int

	// MemoryLimit for Job containers (e.g., "2Gi")
	MemoryLimit string

	// CPULimit for Job containers (e.g., "1")
	CPULimit string

	// CleanupTTL is the TTL for Job cleanup after completion in seconds
	CleanupTTL int

	// MaxConcurrentAgents is the maximum number of concurrent agent Jobs
	MaxConcurrentAgents int
}

// ExecutionDefaults provides default values for all execution clusters.
// This mirrors config.ExecutionDefaults to avoid import cycles.
type ExecutionDefaults struct {
	Namespace           string
	RunnerImage         string
	ImagePullPolicy     string
	Timeout             int
	MemoryLimit         string
	CPULimit            string
	CleanupTTL          int
	MaxConcurrentAgents int
}

// ExecutionClusterManager manages execution clusters where agent Jobs run.
// Unlike ConnectionManager which manages MCP connections for monitored clusters,
// ExecutionClusterManager simply tracks execution cluster configurations and
// provides lookup functionality for the K8sExecutor.
//
// This manager is thread-safe and supports runtime configuration reload.
type ExecutionClusterManager struct {
	// clusters maps cluster name to its configuration
	clusters map[string]*ExecutionClusterConfig

	// defaultCluster is the name of the default execution cluster
	// Used when no specific cluster is requested
	defaultCluster string

	// defaults provides default values for cluster settings
	defaults *ExecutionDefaults

	// mu protects access to the clusters map
	mu sync.RWMutex
}

// ExecutionManagerConfig holds configuration for creating an ExecutionClusterManager.
type ExecutionManagerConfig struct {
	// Clusters is the list of execution cluster configurations
	Clusters []ExecutionClusterConfig

	// Defaults provides default values for execution settings
	Defaults *ExecutionDefaults
}

// NewExecutionClusterManager creates a new execution cluster manager.
// The first cluster in the list becomes the default cluster.
// Returns an error if duplicate cluster names are found.
func NewExecutionClusterManager(cfg *ExecutionManagerConfig) (*ExecutionClusterManager, error) {
	mgr := &ExecutionClusterManager{
		clusters: make(map[string]*ExecutionClusterConfig),
		defaults: cfg.Defaults,
	}

	// Process cluster configurations
	for i := range cfg.Clusters {
		cluster := &cfg.Clusters[i]

		// Check for duplicates
		if _, exists := mgr.clusters[cluster.Name]; exists {
			return nil, fmt.Errorf("duplicate execution cluster name: %s", cluster.Name)
		}

		// Apply defaults to the cluster config
		mgr.applyDefaults(cluster)

		// Store in map
		mgr.clusters[cluster.Name] = cluster

		// First cluster becomes default
		if mgr.defaultCluster == "" {
			mgr.defaultCluster = cluster.Name
		}

		slog.Info("execution cluster registered",
			"name", cluster.Name,
			"namespace", cluster.Namespace,
			"runner_image", cluster.RunnerImage)
	}

	if len(cfg.Clusters) == 0 {
		slog.Warn("no execution clusters configured - using defaults for single cluster mode")
	}

	return mgr, nil
}

// applyDefaults applies default values from ExecutionDefaults to a cluster config.
// Only empty/zero fields in the cluster config are filled from defaults.
func (m *ExecutionClusterManager) applyDefaults(cluster *ExecutionClusterConfig) {
	if m.defaults == nil {
		return
	}

	if cluster.Namespace == "" {
		cluster.Namespace = m.defaults.Namespace
	}
	if cluster.RunnerImage == "" {
		cluster.RunnerImage = m.defaults.RunnerImage
	}
	if cluster.ImagePullPolicy == "" {
		cluster.ImagePullPolicy = m.defaults.ImagePullPolicy
	}
	if cluster.Timeout == 0 {
		cluster.Timeout = m.defaults.Timeout
	}
	if cluster.MemoryLimit == "" {
		cluster.MemoryLimit = m.defaults.MemoryLimit
	}
	if cluster.CPULimit == "" {
		cluster.CPULimit = m.defaults.CPULimit
	}
	if cluster.CleanupTTL == 0 {
		cluster.CleanupTTL = m.defaults.CleanupTTL
	}
	if cluster.MaxConcurrentAgents == 0 {
		cluster.MaxConcurrentAgents = m.defaults.MaxConcurrentAgents
	}
}

// Get retrieves an execution cluster by name.
// Returns nil if not found.
func (m *ExecutionClusterManager) Get(name string) *ExecutionClusterConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.clusters[name]
}

// GetDefault returns the default execution cluster configuration.
// Returns nil if no clusters are configured.
func (m *ExecutionClusterManager) GetDefault() *ExecutionClusterConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.defaultCluster == "" {
		return nil
	}
	return m.clusters[m.defaultCluster]
}

// Select returns the execution cluster for the given name.
// If name is empty, returns the default cluster.
// Returns an error if the cluster is not found or no clusters are configured.
func (m *ExecutionClusterManager) Select(name string) (*ExecutionClusterConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.clusters) == 0 {
		return nil, fmt.Errorf("no execution clusters configured")
	}

	if name != "" {
		if cluster, ok := m.clusters[name]; ok {
			return cluster, nil
		}
		return nil, fmt.Errorf("execution cluster %q not found", name)
	}

	// Return default cluster
	if m.defaultCluster != "" {
		if cluster, ok := m.clusters[m.defaultCluster]; ok {
			return cluster, nil
		}
		return nil, fmt.Errorf("default execution cluster %q not found", m.defaultCluster)
	}

	// Fallback: return first available cluster
	for _, cluster := range m.clusters {
		return cluster, nil
	}

	return nil, fmt.Errorf("no execution clusters configured")
}

// List returns all execution cluster configurations.
func (m *ExecutionClusterManager) List() []*ExecutionClusterConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*ExecutionClusterConfig, 0, len(m.clusters))
	for _, cluster := range m.clusters {
		result = append(result, cluster)
	}
	return result
}

// GetClustersMap returns a map of cluster name to configuration.
// This is useful for passing to K8sExecutor.
func (m *ExecutionClusterManager) GetClustersMap() map[string]*ExecutionClusterConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*ExecutionClusterConfig, len(m.clusters))
	for name, cluster := range m.clusters {
		result[name] = cluster
	}
	return result
}

// DefaultClusterName returns the name of the default execution cluster.
func (m *ExecutionClusterManager) DefaultClusterName() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.defaultCluster
}

// Count returns the number of registered execution clusters.
func (m *ExecutionClusterManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clusters)
}

// Names returns a list of all execution cluster names.
func (m *ExecutionClusterManager) Names() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.clusters))
	for name := range m.clusters {
		names = append(names, name)
	}
	return names
}

// ExecutionReloadResult contains the outcome of an execution cluster reload.
type ExecutionReloadResult struct {
	// Added contains names of newly added clusters
	Added []string

	// Removed contains names of clusters that were removed
	Removed []string

	// Updated contains names of clusters whose configuration changed
	Updated []string
}

// Reload applies a new execution cluster configuration.
// It compares the new configuration with the current state and:
//   - Removes clusters no longer in the configuration
//   - Adds newly configured clusters
//   - Updates changed cluster configurations
//
// The first cluster in the new list becomes the new default (if any).
// This method is thread-safe.
func (m *ExecutionClusterManager) Reload(newClusters []ExecutionClusterConfig) (*ExecutionReloadResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := &ExecutionReloadResult{
		Added:   []string{},
		Removed: []string{},
		Updated: []string{},
	}

	// Build map of old cluster names
	oldNames := make(map[string]bool)
	for name := range m.clusters {
		oldNames[name] = true
	}

	// Build map of new cluster configs
	newClustersMap := make(map[string]*ExecutionClusterConfig)
	var newDefault string
	for i := range newClusters {
		cluster := &newClusters[i]

		// Check for duplicates in new config
		if _, exists := newClustersMap[cluster.Name]; exists {
			return nil, fmt.Errorf("duplicate execution cluster name in reload: %s", cluster.Name)
		}

		// Apply defaults
		m.applyDefaults(cluster)
		newClustersMap[cluster.Name] = cluster

		// First cluster is new default
		if newDefault == "" {
			newDefault = cluster.Name
		}
	}

	// Find and remove clusters that no longer exist
	for name := range oldNames {
		if _, exists := newClustersMap[name]; !exists {
			result.Removed = append(result.Removed, name)
			delete(m.clusters, name)

			slog.Info("execution cluster removed",
				"name", name)
		}
	}

	// Find added and updated clusters
	for name, newCfg := range newClustersMap {
		if !oldNames[name] {
			// New cluster
			result.Added = append(result.Added, name)
			m.clusters[name] = newCfg

			slog.Info("execution cluster added",
				"name", name,
				"namespace", newCfg.Namespace)
		} else {
			// Check if config changed
			oldCfg := m.clusters[name]
			if executionConfigChanged(oldCfg, newCfg) {
				result.Updated = append(result.Updated, name)
				m.clusters[name] = newCfg

				slog.Info("execution cluster updated",
					"name", name)
			}
		}
	}

	// Update default cluster
	m.defaultCluster = newDefault

	// Log summary
	if len(result.Added) > 0 || len(result.Removed) > 0 || len(result.Updated) > 0 {
		slog.Info("execution cluster configuration reloaded",
			"added", len(result.Added),
			"removed", len(result.Removed),
			"updated", len(result.Updated),
			"total", len(m.clusters),
			"default", m.defaultCluster)
	} else {
		slog.Info("execution cluster configuration unchanged",
			"total", len(m.clusters))
	}

	return result, nil
}

// executionConfigChanged compares two execution cluster configurations.
func executionConfigChanged(old, new *ExecutionClusterConfig) bool {
	if old.KubeconfigPath != new.KubeconfigPath {
		return true
	}
	if old.Namespace != new.Namespace {
		return true
	}
	if old.RunnerImage != new.RunnerImage {
		return true
	}
	if old.ImagePullPolicy != new.ImagePullPolicy {
		return true
	}
	if old.Timeout != new.Timeout {
		return true
	}
	if old.MemoryLimit != new.MemoryLimit {
		return true
	}
	if old.CPULimit != new.CPULimit {
		return true
	}
	if old.CleanupTTL != new.CleanupTTL {
		return true
	}
	if old.MaxConcurrentAgents != new.MaxConcurrentAgents {
		return true
	}
	return false
}
