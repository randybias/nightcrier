package bootstrap

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNewBootstrapStatus(t *testing.T) {
	status := NewBootstrapStatus()

	if status.State != StateInitializing {
		t.Errorf("State = %v, want %v", status.State, StateInitializing)
	}
	if status.GlobalReady {
		t.Error("GlobalReady should be false initially")
	}
	if status.APIKeysReady {
		t.Error("APIKeysReady should be false initially")
	}
	if status.ClusterStatuses == nil {
		t.Error("ClusterStatuses should not be nil")
	}
	if len(status.ClusterStatuses) != 0 {
		t.Errorf("ClusterStatuses should be empty, got %d", len(status.ClusterStatuses))
	}
}

func TestBootstrapStatus_SetGlobalReady(t *testing.T) {
	tests := []struct {
		name           string
		namespaceReady bool
		rbacReady      bool
		wantGlobal     bool
	}{
		{
			name:           "both ready",
			namespaceReady: true,
			rbacReady:      true,
			wantGlobal:     true,
		},
		{
			name:           "namespace not ready",
			namespaceReady: false,
			rbacReady:      true,
			wantGlobal:     false,
		},
		{
			name:           "rbac not ready",
			namespaceReady: true,
			rbacReady:      false,
			wantGlobal:     false,
		},
		{
			name:           "neither ready",
			namespaceReady: false,
			rbacReady:      false,
			wantGlobal:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := NewBootstrapStatus()
			status.SetGlobalReady(tt.namespaceReady, tt.rbacReady)

			if status.GlobalReady != tt.wantGlobal {
				t.Errorf("GlobalReady = %v, want %v", status.GlobalReady, tt.wantGlobal)
			}
			if status.NamespaceReady != tt.namespaceReady {
				t.Errorf("NamespaceReady = %v, want %v", status.NamespaceReady, tt.namespaceReady)
			}
			if status.RBACReady != tt.rbacReady {
				t.Errorf("RBACReady = %v, want %v", status.RBACReady, tt.rbacReady)
			}
		})
	}
}

func TestBootstrapStatus_SetAPIKeysStatus(t *testing.T) {
	tests := []struct {
		name      string
		ready     bool
		err       error
		wantMsg   string
		wantReady bool
	}{
		{
			name:      "ready with no error",
			ready:     true,
			err:       nil,
			wantMsg:   "",
			wantReady: true,
		},
		{
			name:      "not ready with error",
			ready:     false,
			err:       errors.New("no API keys configured"),
			wantMsg:   "no API keys configured",
			wantReady: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := NewBootstrapStatus()
			status.SetAPIKeysStatus(tt.ready, tt.err)

			if status.APIKeysReady != tt.wantReady {
				t.Errorf("APIKeysReady = %v, want %v", status.APIKeysReady, tt.wantReady)
			}
			if status.APIKeysErrorMessage != tt.wantMsg {
				t.Errorf("APIKeysErrorMessage = %q, want %q", status.APIKeysErrorMessage, tt.wantMsg)
			}
		})
	}
}

func TestBootstrapStatus_SetClusterStatus(t *testing.T) {
	status := NewBootstrapStatus()

	// Set a new cluster status
	status.SetClusterStatus("cluster-1", true, nil)

	if len(status.ClusterStatuses) != 1 {
		t.Errorf("ClusterStatuses len = %d, want 1", len(status.ClusterStatuses))
	}

	cs := status.ClusterStatuses["cluster-1"]
	if cs == nil {
		t.Fatal("cluster-1 status should exist")
	}
	if !cs.Ready {
		t.Error("cluster-1 should be ready")
	}
	if cs.Retries != 0 {
		t.Errorf("Retries = %d, want 0 (ready clusters don't increment)", cs.Retries)
	}

	// Update with failure
	status.SetClusterStatus("cluster-1", false, errors.New("connection refused"))

	cs = status.ClusterStatuses["cluster-1"]
	if cs.Ready {
		t.Error("cluster-1 should not be ready after failure")
	}
	if cs.Retries != 1 {
		t.Errorf("Retries = %d, want 1", cs.Retries)
	}
	if cs.ErrorMessage != "connection refused" {
		t.Errorf("ErrorMessage = %q, want %q", cs.ErrorMessage, "connection refused")
	}

	// Multiple failures increment retries
	status.SetClusterStatus("cluster-1", false, errors.New("still failing"))
	cs = status.ClusterStatuses["cluster-1"]
	if cs.Retries != 2 {
		t.Errorf("Retries = %d, want 2", cs.Retries)
	}
}

func TestBootstrapStatus_StateTransitions(t *testing.T) {
	status := NewBootstrapStatus()

	// Initial state
	if status.GetState() != StateInitializing {
		t.Errorf("initial state = %v, want %v", status.GetState(), StateInitializing)
	}

	// Set global ready and API keys ready, but add a failing cluster
	status.SetGlobalReady(true, true)
	status.SetAPIKeysStatus(true, nil)
	status.SetClusterStatus("cluster-1", false, errors.New("kubeconfig not found"))

	// Should be in retrying state (recent failure)
	state := status.GetState()
	if state != StateRetrying {
		t.Errorf("state after failure = %v, want %v", state, StateRetrying)
	}

	// Mark cluster as ready
	status.SetClusterStatus("cluster-1", true, nil)

	// Should now be ready
	if status.GetState() != StateReady {
		t.Errorf("state after recovery = %v, want %v", status.GetState(), StateReady)
	}

	if !status.IsReady() {
		t.Error("IsReady() should be true")
	}
}

func TestBootstrapStatus_ReadyAndDegradedClusters(t *testing.T) {
	status := NewBootstrapStatus()
	status.SetGlobalReady(true, true)
	status.SetAPIKeysStatus(true, nil)

	// Add some clusters
	status.SetClusterStatus("cluster-1", true, nil)
	status.SetClusterStatus("cluster-2", false, errors.New("error"))
	status.SetClusterStatus("cluster-3", true, nil)
	status.SetClusterStatus("cluster-4", false, errors.New("error"))

	if status.ReadyClusters() != 2 {
		t.Errorf("ReadyClusters() = %d, want 2", status.ReadyClusters())
	}

	if status.TotalClusters() != 4 {
		t.Errorf("TotalClusters() = %d, want 4", status.TotalClusters())
	}

	degraded := status.DegradedClusters()
	if len(degraded) != 2 {
		t.Errorf("DegradedClusters() len = %d, want 2", len(degraded))
	}

	// Verify degraded cluster names
	degradedNames := make(map[string]bool)
	for _, cs := range degraded {
		degradedNames[cs.Name] = true
	}
	if !degradedNames["cluster-2"] || !degradedNames["cluster-4"] {
		t.Error("DegradedClusters should contain cluster-2 and cluster-4")
	}
}

func TestBootstrapStatus_Clone(t *testing.T) {
	status := NewBootstrapStatus()
	status.SetGlobalReady(true, true)
	status.SetAPIKeysStatus(true, nil)
	status.SetClusterStatus("cluster-1", true, nil)
	status.SetClusterStatus("cluster-2", false, errors.New("error"))

	clone := status.Clone()

	// Verify clone has same values
	if clone.GlobalReady != status.GlobalReady {
		t.Error("Clone GlobalReady mismatch")
	}
	if clone.APIKeysReady != status.APIKeysReady {
		t.Error("Clone APIKeysReady mismatch")
	}
	if len(clone.ClusterStatuses) != len(status.ClusterStatuses) {
		t.Error("Clone ClusterStatuses length mismatch")
	}

	// Modify original and verify clone is unchanged
	status.SetClusterStatus("cluster-3", true, nil)
	if len(clone.ClusterStatuses) != 2 {
		t.Error("Clone should not be affected by original modifications")
	}
}

func TestBootstrapStatus_ConcurrentAccess(t *testing.T) {
	status := NewBootstrapStatus()

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent writers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			clusterName := "cluster-" + string(rune('a'+id%26))
			status.SetClusterStatus(clusterName, id%2 == 0, nil)
			status.SetGlobalReady(true, true)
			status.SetAPIKeysStatus(true, nil)
		}(i)
	}

	// Concurrent readers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = status.GetState()
			_ = status.IsReady()
			_ = status.IsDegraded()
			_ = status.ReadyClusters()
			_ = status.TotalClusters()
			_ = status.DegradedClusters()
			_ = status.Clone()
		}()
	}

	wg.Wait()
	// If we get here without a race condition, the test passes
}

func TestBootstrapStatus_IsDegraded(t *testing.T) {
	status := NewBootstrapStatus()
	status.SetGlobalReady(true, true)
	status.SetAPIKeysStatus(true, nil)

	// All ready - not degraded
	status.SetClusterStatus("cluster-1", true, nil)
	if status.IsDegraded() {
		t.Error("IsDegraded should be false when all ready")
	}

	// Add a failing cluster - should be degraded
	status.SetClusterStatus("cluster-2", false, errors.New("error"))
	if !status.IsDegraded() {
		t.Error("IsDegraded should be true when a cluster is failing")
	}
}

func TestBootstrapStatus_StateRetryingVsDegraded(t *testing.T) {
	status := NewBootstrapStatus()
	status.SetGlobalReady(true, true)
	status.SetAPIKeysStatus(true, nil)

	// Add a failing cluster - should be retrying (recent failure)
	status.SetClusterStatus("cluster-1", false, errors.New("error"))
	if status.GetState() != StateRetrying {
		t.Errorf("state = %v, want %v (recent failure)", status.GetState(), StateRetrying)
	}

	// Manually set LastRetry to more than a minute ago to simulate old failure
	status.mu.Lock()
	status.ClusterStatuses["cluster-1"].LastRetry = time.Now().Add(-2 * time.Minute)
	status.updateState()
	status.mu.Unlock()

	if status.GetState() != StateDegraded {
		t.Errorf("state = %v, want %v (old failure)", status.GetState(), StateDegraded)
	}
}
