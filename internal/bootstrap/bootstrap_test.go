package bootstrap

import (
	"context"
	"testing"
	"time"

	"k8s.io/client-go/kubernetes/fake"
)

func TestManager_BootstrapNonBlocking_DegradedStartup(t *testing.T) {
	// Test that bootstrap starts in degraded mode when resources fail
	kubeClient := fake.NewSimpleClientset()

	config := Config{
		Namespace:       "nightcrier",
		AnthropicAPIKey: "test-key",
		MonitoredClusters: []MonitoredClusterConfig{
			{
				Name:                 "cluster-1",
				TargetKubeconfigPath: "/nonexistent/kubeconfig.yaml", // Will fail
			},
			{
				Name:                 "cluster-2",
				TargetKubeconfigPath: "", // No kubeconfig needed - will succeed
			},
		},
	}

	manager := NewManager(kubeClient, config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	status := manager.BootstrapNonBlocking(ctx)

	// Global resources should be ready (fake client succeeds)
	if !status.GlobalReady {
		t.Error("GlobalReady should be true with fake client")
	}

	// API keys should be ready
	if !status.APIKeysReady {
		t.Error("APIKeysReady should be true")
	}

	// Cluster-1 should be degraded (kubeconfig not found)
	cs1 := status.ClusterStatuses["cluster-1"]
	if cs1 == nil {
		t.Fatal("cluster-1 status should exist")
	}
	if cs1.Ready {
		t.Error("cluster-1 should not be ready (kubeconfig not found)")
	}
	if cs1.Error == nil {
		t.Error("cluster-1 should have an error")
	}

	// Cluster-2 should be ready (no kubeconfig needed)
	cs2 := status.ClusterStatuses["cluster-2"]
	if cs2 == nil {
		t.Fatal("cluster-2 status should exist")
	}
	if !cs2.Ready {
		t.Error("cluster-2 should be ready (no kubeconfig needed)")
	}

	// Overall state should NOT be ready (cluster-1 is degraded)
	if status.IsReady() {
		t.Error("IsReady should be false when a cluster is degraded")
	}

	// But we should still have started (this is the key point of non-blocking bootstrap)
	if status.State == StateInitializing {
		t.Error("State should not be Initializing after BootstrapNonBlocking completes")
	}
}

func TestManager_BootstrapNonBlocking_AllReady(t *testing.T) {
	// Test that bootstrap reports ready when all resources succeed
	kubeClient := fake.NewSimpleClientset()

	config := Config{
		Namespace:       "nightcrier",
		AnthropicAPIKey: "test-key",
		MonitoredClusters: []MonitoredClusterConfig{
			{
				Name:                 "cluster-1",
				TargetKubeconfigPath: "", // No kubeconfig needed
			},
			{
				Name:                 "cluster-2",
				TargetKubeconfigPath: "", // No kubeconfig needed
			},
		},
	}

	manager := NewManager(kubeClient, config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	status := manager.BootstrapNonBlocking(ctx)

	// Everything should be ready
	if !status.GlobalReady {
		t.Error("GlobalReady should be true")
	}
	if !status.APIKeysReady {
		t.Error("APIKeysReady should be true")
	}
	if !status.IsReady() {
		t.Error("IsReady should be true when all resources are ready")
	}
	if status.State != StateReady {
		t.Errorf("State = %v, want %v", status.State, StateReady)
	}
}

func TestManager_BootstrapNonBlocking_NoAPIKeys(t *testing.T) {
	// Test that bootstrap handles missing API keys gracefully
	kubeClient := fake.NewSimpleClientset()

	config := Config{
		Namespace: "nightcrier",
		// No API keys configured
		MonitoredClusters: []MonitoredClusterConfig{
			{
				Name:                 "cluster-1",
				TargetKubeconfigPath: "", // No kubeconfig needed
			},
		},
	}

	manager := NewManager(kubeClient, config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	status := manager.BootstrapNonBlocking(ctx)

	// Global resources should be ready
	if !status.GlobalReady {
		t.Error("GlobalReady should be true")
	}

	// API keys should NOT be ready (none configured)
	if status.APIKeysReady {
		t.Error("APIKeysReady should be false when no keys configured")
	}

	// But we should NOT be in ready state overall
	if status.IsReady() {
		t.Error("IsReady should be false when API keys are not ready")
	}

	// Verify we got an appropriate error message
	if status.APIKeysErrorMessage == "" {
		t.Error("APIKeysErrorMessage should explain why keys are not ready")
	}
}

func TestManager_BootstrapNonBlocking_ParallelClusters(t *testing.T) {
	// Test that multiple clusters are bootstrapped in parallel
	kubeClient := fake.NewSimpleClientset()

	// Create many clusters to verify parallel execution
	clusters := make([]MonitoredClusterConfig, 50)
	for i := range clusters {
		clusters[i] = MonitoredClusterConfig{
			Name:                 "cluster-" + string(rune('A'+i%26)) + string(rune('0'+i/26)),
			TargetKubeconfigPath: "", // No kubeconfig needed
		}
	}

	config := Config{
		Namespace:         "nightcrier",
		AnthropicAPIKey:   "test-key",
		MonitoredClusters: clusters,
	}

	manager := NewManager(kubeClient, config)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()
	status := manager.BootstrapNonBlocking(ctx)
	duration := time.Since(start)

	// Should complete quickly if parallel (< 5s for 50 clusters)
	// Sequential would take much longer with any I/O
	if duration > 5*time.Second {
		t.Errorf("Bootstrap took %v, expected < 5s for parallel execution", duration)
	}

	// All clusters should be ready
	if status.TotalClusters() != 50 {
		t.Errorf("TotalClusters = %d, want 50", status.TotalClusters())
	}
	if status.ReadyClusters() != 50 {
		t.Errorf("ReadyClusters = %d, want 50", status.ReadyClusters())
	}
	if !status.IsReady() {
		t.Error("IsReady should be true")
	}
}

func TestManager_RetryCluster_Recovery(t *testing.T) {
	// Test that RetryCluster can recover a degraded cluster
	kubeClient := fake.NewSimpleClientset()

	config := Config{
		Namespace:       "nightcrier",
		AnthropicAPIKey: "test-key",
		MonitoredClusters: []MonitoredClusterConfig{
			{
				Name:                 "cluster-1",
				TargetKubeconfigPath: "", // No kubeconfig needed - will succeed on retry
			},
		},
	}

	manager := NewManager(kubeClient, config)

	// Manually set cluster to degraded state
	manager.status.SetGlobalReady(true, true)
	manager.status.SetAPIKeysStatus(true, nil)
	manager.status.SetClusterStatus("cluster-1", false, errSomethingWentWrong)

	// Verify it's degraded
	if manager.status.IsReady() {
		t.Fatal("status should not be ready before recovery")
	}

	// Retry the cluster
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := manager.RetryCluster(ctx, "cluster-1")
	if err != nil {
		t.Fatalf("RetryCluster failed: %v", err)
	}

	// Should now be recovered
	status := manager.GetStatus()
	cs := status.ClusterStatuses["cluster-1"]
	if cs == nil {
		t.Fatal("cluster-1 status should exist")
	}
	if !cs.Ready {
		t.Error("cluster-1 should be ready after retry")
	}
}

func TestManager_RetryGlobal_Recovery(t *testing.T) {
	// Test that RetryGlobal can recover global resources
	kubeClient := fake.NewSimpleClientset()

	config := Config{
		Namespace:       "nightcrier",
		AnthropicAPIKey: "test-key",
	}

	manager := NewManager(kubeClient, config)

	// Verify global is not ready initially
	if manager.status.GlobalReady {
		t.Fatal("GlobalReady should be false initially")
	}

	// Retry global resources
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := manager.RetryGlobal(ctx)
	if err != nil {
		t.Fatalf("RetryGlobal failed: %v", err)
	}

	// Should now be ready
	status := manager.GetStatus()
	if !status.GlobalReady {
		t.Error("GlobalReady should be true after retry")
	}
}

func TestManager_RetryAPIKeys_Recovery(t *testing.T) {
	// Test that RetryAPIKeys can recover API keys
	kubeClient := fake.NewSimpleClientset()

	config := Config{
		Namespace:       "nightcrier",
		AnthropicAPIKey: "test-key",
	}

	manager := NewManager(kubeClient, config)

	// First ensure global is ready (prerequisite for API keys)
	manager.status.SetGlobalReady(true, true)

	// Verify API keys not ready initially
	if manager.status.APIKeysReady {
		t.Fatal("APIKeysReady should be false initially")
	}

	// Retry API keys
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := manager.RetryAPIKeys(ctx)
	if err != nil {
		t.Fatalf("RetryAPIKeys failed: %v", err)
	}

	// Should now be ready
	status := manager.GetStatus()
	if !status.APIKeysReady {
		t.Error("APIKeysReady should be true after retry")
	}
}

func TestManager_GetStatus_ThreadSafe(t *testing.T) {
	// Test that GetStatus returns a thread-safe copy
	kubeClient := fake.NewSimpleClientset()

	config := Config{
		Namespace:       "nightcrier",
		AnthropicAPIKey: "test-key",
		MonitoredClusters: []MonitoredClusterConfig{
			{
				Name:                 "cluster-1",
				TargetKubeconfigPath: "",
			},
		},
	}

	manager := NewManager(kubeClient, config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	manager.BootstrapNonBlocking(ctx)

	// Get a status copy
	status1 := manager.GetStatus()

	// Modify original (simulate background changes)
	manager.status.SetClusterStatus("cluster-2", true, nil)

	// Get another copy
	status2 := manager.GetStatus()

	// First copy should not have cluster-2
	if _, exists := status1.ClusterStatuses["cluster-2"]; exists {
		t.Error("status1 should not have cluster-2 (it's a copy)")
	}

	// Second copy should have cluster-2
	if _, exists := status2.ClusterStatuses["cluster-2"]; !exists {
		t.Error("status2 should have cluster-2")
	}
}

// errSomethingWentWrong is a test error
var errSomethingWentWrong = &testError{msg: "something went wrong"}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
