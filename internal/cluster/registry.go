package cluster

import (
	"fmt"
	"sync"
)

// Registry manages cluster configurations with thread-safe access.
type Registry struct {
	clusters map[string]*ClusterConfig
	mu       sync.RWMutex
}

// NewRegistry creates a new cluster registry.
func NewRegistry() *Registry {
	return &Registry{
		clusters: make(map[string]*ClusterConfig),
	}
}

// Load populates the registry with cluster configurations.
func (r *Registry) Load(clusters []ClusterConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range clusters {
		cluster := &clusters[i]
		if _, exists := r.clusters[cluster.Name]; exists {
			return fmt.Errorf("duplicate cluster name: %s", cluster.Name)
		}
		r.clusters[cluster.Name] = cluster
	}
	return nil
}

// Get retrieves a cluster configuration by name.
// Returns nil if not found.
func (r *Registry) Get(name string) *ClusterConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.clusters[name]
}

// List returns all cluster configurations.
func (r *Registry) List() []*ClusterConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*ClusterConfig, 0, len(r.clusters))
	for _, cluster := range r.clusters {
		result = append(result, cluster)
	}
	return result
}

// Count returns the number of registered clusters.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clusters)
}
