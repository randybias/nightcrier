package bootstrap

import (
	"sync"
	"time"
)

// BootstrapState represents the overall bootstrap state.
type BootstrapState string

const (
	// StateReady indicates all resources are bootstrapped successfully.
	StateReady BootstrapState = "ready"
	// StateDegraded indicates some resources failed to bootstrap.
	StateDegraded BootstrapState = "degraded"
	// StateRetrying indicates bootstrap is actively retrying failed resources.
	StateRetrying BootstrapState = "retrying"
	// StateInitializing indicates bootstrap has not yet completed first attempt.
	StateInitializing BootstrapState = "initializing"
)

// ClusterBootstrapStatus tracks the bootstrap status for a single monitored cluster.
type ClusterBootstrapStatus struct {
	// Name is the cluster name.
	Name string `json:"name"`
	// Ready indicates whether the cluster's resources are bootstrapped.
	Ready bool `json:"ready"`
	// Error is the last error encountered, if any.
	Error error `json:"-"`
	// ErrorMessage is the string representation of Error for JSON serialization.
	ErrorMessage string `json:"error,omitempty"`
	// LastRetry is the timestamp of the last retry attempt.
	LastRetry time.Time `json:"last_retry,omitempty"`
	// Retries is the number of retry attempts.
	Retries int `json:"retries"`
}

// BootstrapStatus tracks the overall bootstrap status for the system.
// It is thread-safe for concurrent access.
type BootstrapStatus struct {
	mu sync.RWMutex

	// State is the overall bootstrap state.
	State BootstrapState `json:"state"`

	// GlobalReady indicates whether global resources (namespace, RBAC) are ready.
	GlobalReady bool `json:"global_ready"`

	// APIKeysReady indicates whether API keys secret is ready.
	APIKeysReady bool `json:"api_keys_ready"`

	// APIKeysError is the error from API keys bootstrap, if any.
	APIKeysError error `json:"-"`

	// APIKeysErrorMessage is the string representation for JSON.
	APIKeysErrorMessage string `json:"api_keys_error,omitempty"`

	// NamespaceReady indicates whether the namespace is ready.
	NamespaceReady bool `json:"namespace_ready"`

	// RBACReady indicates whether RBAC resources are ready.
	RBACReady bool `json:"rbac_ready"`

	// ClusterStatuses maps cluster name to its bootstrap status.
	ClusterStatuses map[string]*ClusterBootstrapStatus `json:"cluster_statuses"`

	// LastUpdated is the timestamp of the last status update.
	LastUpdated time.Time `json:"last_updated"`
}

// NewBootstrapStatus creates a new BootstrapStatus in initializing state.
func NewBootstrapStatus() *BootstrapStatus {
	return &BootstrapStatus{
		State:           StateInitializing,
		ClusterStatuses: make(map[string]*ClusterBootstrapStatus),
		LastUpdated:     time.Now(),
	}
}

// SetGlobalReady updates the global resources status.
func (s *BootstrapStatus) SetGlobalReady(namespaceReady, rbacReady bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.NamespaceReady = namespaceReady
	s.RBACReady = rbacReady
	s.GlobalReady = namespaceReady && rbacReady
	s.LastUpdated = time.Now()
	s.updateState()
}

// SetAPIKeysStatus updates the API keys bootstrap status.
func (s *BootstrapStatus) SetAPIKeysStatus(ready bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.APIKeysReady = ready
	s.APIKeysError = err
	if err != nil {
		s.APIKeysErrorMessage = err.Error()
	} else {
		s.APIKeysErrorMessage = ""
	}
	s.LastUpdated = time.Now()
	s.updateState()
}

// SetClusterStatus updates or adds a cluster's bootstrap status.
func (s *BootstrapStatus) SetClusterStatus(name string, ready bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	status, exists := s.ClusterStatuses[name]
	if !exists {
		status = &ClusterBootstrapStatus{Name: name}
		s.ClusterStatuses[name] = status
	}

	status.Ready = ready
	status.Error = err
	if err != nil {
		status.ErrorMessage = err.Error()
	} else {
		status.ErrorMessage = ""
	}
	status.LastRetry = time.Now()
	if !ready {
		status.Retries++
	}

	s.LastUpdated = time.Now()
	s.updateState()
}

// GetState returns the current bootstrap state.
func (s *BootstrapStatus) GetState() BootstrapState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State
}

// IsReady returns true if all resources are bootstrapped.
func (s *BootstrapStatus) IsReady() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State == StateReady
}

// IsDegraded returns true if the system is in degraded state.
func (s *BootstrapStatus) IsDegraded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State == StateDegraded || s.State == StateRetrying
}

// ReadyClusters returns the count of ready clusters.
func (s *BootstrapStatus) ReadyClusters() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, cs := range s.ClusterStatuses {
		if cs.Ready {
			count++
		}
	}
	return count
}

// TotalClusters returns the total number of clusters being tracked.
func (s *BootstrapStatus) TotalClusters() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.ClusterStatuses)
}

// DegradedClusters returns a list of clusters that are not ready.
func (s *BootstrapStatus) DegradedClusters() []*ClusterBootstrapStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var degraded []*ClusterBootstrapStatus
	for _, cs := range s.ClusterStatuses {
		if !cs.Ready {
			degraded = append(degraded, cs)
		}
	}
	return degraded
}

// Clone returns a deep copy of the status for safe reading.
func (s *BootstrapStatus) Clone() *BootstrapStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clone := &BootstrapStatus{
		State:               s.State,
		GlobalReady:         s.GlobalReady,
		APIKeysReady:        s.APIKeysReady,
		APIKeysError:        s.APIKeysError,
		APIKeysErrorMessage: s.APIKeysErrorMessage,
		NamespaceReady:      s.NamespaceReady,
		RBACReady:           s.RBACReady,
		ClusterStatuses:     make(map[string]*ClusterBootstrapStatus),
		LastUpdated:         s.LastUpdated,
	}

	for name, cs := range s.ClusterStatuses {
		clone.ClusterStatuses[name] = &ClusterBootstrapStatus{
			Name:         cs.Name,
			Ready:        cs.Ready,
			Error:        cs.Error,
			ErrorMessage: cs.ErrorMessage,
			LastRetry:    cs.LastRetry,
			Retries:      cs.Retries,
		}
	}

	return clone
}

// updateState recalculates the overall state based on component statuses.
// Must be called with lock held.
func (s *BootstrapStatus) updateState() {
	// Check if everything is ready
	allReady := s.GlobalReady && s.APIKeysReady
	if allReady {
		for _, cs := range s.ClusterStatuses {
			if !cs.Ready {
				allReady = false
				break
			}
		}
	}

	if allReady {
		s.State = StateReady
		return
	}

	// Check if any cluster is actively retrying (recent retry within last minute)
	hasRecentRetry := false
	oneMinuteAgo := time.Now().Add(-time.Minute)
	for _, cs := range s.ClusterStatuses {
		if !cs.Ready && cs.LastRetry.After(oneMinuteAgo) {
			hasRecentRetry = true
			break
		}
	}

	if hasRecentRetry {
		s.State = StateRetrying
	} else {
		s.State = StateDegraded
	}
}
