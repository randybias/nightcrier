package cluster

import (
	"testing"
)

func TestNewExecutionClusterManager(t *testing.T) {
	t.Run("creates manager with clusters", func(t *testing.T) {
		cfg := &ExecutionManagerConfig{
			Clusters: []ExecutionClusterConfig{
				{Name: "cluster-a", Namespace: "ns-a"},
				{Name: "cluster-b", Namespace: "ns-b"},
			},
			Defaults: &ExecutionDefaults{
				Namespace:   "default-ns",
				RunnerImage: "runner:latest",
				Timeout:     600,
			},
		}

		mgr, err := NewExecutionClusterManager(cfg)
		if err != nil {
			t.Fatalf("NewExecutionClusterManager() error = %v", err)
		}

		if mgr.Count() != 2 {
			t.Errorf("Count() = %d, want 2", mgr.Count())
		}

		// First cluster should be default
		if mgr.DefaultClusterName() != "cluster-a" {
			t.Errorf("DefaultClusterName() = %v, want cluster-a", mgr.DefaultClusterName())
		}
	})

	t.Run("rejects duplicate names", func(t *testing.T) {
		cfg := &ExecutionManagerConfig{
			Clusters: []ExecutionClusterConfig{
				{Name: "same-name", Namespace: "ns-1"},
				{Name: "same-name", Namespace: "ns-2"},
			},
		}

		_, err := NewExecutionClusterManager(cfg)
		if err == nil {
			t.Fatal("Expected error for duplicate names")
		}
	})

	t.Run("allows zero clusters", func(t *testing.T) {
		cfg := &ExecutionManagerConfig{
			Clusters: []ExecutionClusterConfig{},
		}

		mgr, err := NewExecutionClusterManager(cfg)
		if err != nil {
			t.Fatalf("NewExecutionClusterManager() error = %v", err)
		}

		if mgr.Count() != 0 {
			t.Errorf("Count() = %d, want 0", mgr.Count())
		}
	})

	t.Run("applies defaults to clusters", func(t *testing.T) {
		cfg := &ExecutionManagerConfig{
			Clusters: []ExecutionClusterConfig{
				{Name: "cluster-1"}, // No settings, should get defaults
			},
			Defaults: &ExecutionDefaults{
				Namespace:           "default-ns",
				RunnerImage:         "default-image:v1",
				ImagePullPolicy:     "Always",
				Timeout:             900,
				MemoryLimit:         "4Gi",
				CPULimit:            "2",
				CleanupTTL:          7200,
				MaxConcurrentAgents: 5,
			},
		}

		mgr, err := NewExecutionClusterManager(cfg)
		if err != nil {
			t.Fatalf("NewExecutionClusterManager() error = %v", err)
		}

		cluster := mgr.Get("cluster-1")
		if cluster == nil {
			t.Fatal("cluster not found")
		}

		if cluster.Namespace != "default-ns" {
			t.Errorf("Namespace = %v, want default-ns", cluster.Namespace)
		}
		if cluster.RunnerImage != "default-image:v1" {
			t.Errorf("RunnerImage = %v, want default-image:v1", cluster.RunnerImage)
		}
		if cluster.Timeout != 900 {
			t.Errorf("Timeout = %v, want 900", cluster.Timeout)
		}
	})

	t.Run("cluster overrides defaults", func(t *testing.T) {
		cfg := &ExecutionManagerConfig{
			Clusters: []ExecutionClusterConfig{
				{
					Name:        "cluster-1",
					Namespace:   "custom-ns",
					RunnerImage: "custom-image:v2",
				},
			},
			Defaults: &ExecutionDefaults{
				Namespace:   "default-ns",
				RunnerImage: "default-image:v1",
			},
		}

		mgr, err := NewExecutionClusterManager(cfg)
		if err != nil {
			t.Fatalf("NewExecutionClusterManager() error = %v", err)
		}

		cluster := mgr.Get("cluster-1")
		if cluster.Namespace != "custom-ns" {
			t.Errorf("Namespace = %v, want custom-ns", cluster.Namespace)
		}
		if cluster.RunnerImage != "custom-image:v2" {
			t.Errorf("RunnerImage = %v, want custom-image:v2", cluster.RunnerImage)
		}
	})
}

func TestExecutionClusterManager_Get(t *testing.T) {
	cfg := &ExecutionManagerConfig{
		Clusters: []ExecutionClusterConfig{
			{Name: "cluster-a", Namespace: "ns-a"},
			{Name: "cluster-b", Namespace: "ns-b"},
		},
	}

	mgr, _ := NewExecutionClusterManager(cfg)

	t.Run("returns existing cluster", func(t *testing.T) {
		cluster := mgr.Get("cluster-b")
		if cluster == nil {
			t.Fatal("Get() returned nil")
		}
		if cluster.Name != "cluster-b" {
			t.Errorf("Name = %v, want cluster-b", cluster.Name)
		}
	})

	t.Run("returns nil for non-existent", func(t *testing.T) {
		cluster := mgr.Get("nonexistent")
		if cluster != nil {
			t.Error("Get() should return nil for nonexistent cluster")
		}
	})
}

func TestExecutionClusterManager_Select(t *testing.T) {
	cfg := &ExecutionManagerConfig{
		Clusters: []ExecutionClusterConfig{
			{Name: "cluster-a", Namespace: "ns-a"},
			{Name: "cluster-b", Namespace: "ns-b"},
		},
	}

	mgr, _ := NewExecutionClusterManager(cfg)

	t.Run("selects by name", func(t *testing.T) {
		cluster, err := mgr.Select("cluster-b")
		if err != nil {
			t.Fatalf("Select() error = %v", err)
		}
		if cluster.Name != "cluster-b" {
			t.Errorf("Name = %v, want cluster-b", cluster.Name)
		}
	})

	t.Run("returns default when empty name", func(t *testing.T) {
		cluster, err := mgr.Select("")
		if err != nil {
			t.Fatalf("Select() error = %v", err)
		}
		if cluster.Name != "cluster-a" {
			t.Errorf("Name = %v, want cluster-a (default)", cluster.Name)
		}
	})

	t.Run("errors on nonexistent name", func(t *testing.T) {
		_, err := mgr.Select("nonexistent")
		if err == nil {
			t.Fatal("Expected error for nonexistent cluster")
		}
	})

	t.Run("errors when no clusters configured", func(t *testing.T) {
		emptyMgr, _ := NewExecutionClusterManager(&ExecutionManagerConfig{})
		_, err := emptyMgr.Select("")
		if err == nil {
			t.Fatal("Expected error when no clusters configured")
		}
	})
}

func TestExecutionClusterManager_List(t *testing.T) {
	cfg := &ExecutionManagerConfig{
		Clusters: []ExecutionClusterConfig{
			{Name: "cluster-a"},
			{Name: "cluster-b"},
			{Name: "cluster-c"},
		},
	}

	mgr, _ := NewExecutionClusterManager(cfg)

	list := mgr.List()
	if len(list) != 3 {
		t.Errorf("List() returned %d items, want 3", len(list))
	}
}

func TestExecutionClusterManager_GetClustersMap(t *testing.T) {
	cfg := &ExecutionManagerConfig{
		Clusters: []ExecutionClusterConfig{
			{Name: "cluster-a", Namespace: "ns-a"},
			{Name: "cluster-b", Namespace: "ns-b"},
		},
	}

	mgr, _ := NewExecutionClusterManager(cfg)

	m := mgr.GetClustersMap()
	if len(m) != 2 {
		t.Errorf("GetClustersMap() returned %d items, want 2", len(m))
	}

	if m["cluster-a"] == nil || m["cluster-a"].Namespace != "ns-a" {
		t.Error("cluster-a not found or has wrong namespace")
	}
}

func TestExecutionClusterManager_Names(t *testing.T) {
	cfg := &ExecutionManagerConfig{
		Clusters: []ExecutionClusterConfig{
			{Name: "alpha"},
			{Name: "beta"},
		},
	}

	mgr, _ := NewExecutionClusterManager(cfg)

	names := mgr.Names()
	if len(names) != 2 {
		t.Errorf("Names() returned %d items, want 2", len(names))
	}

	// Check both names are present (order is not guaranteed due to map iteration)
	found := make(map[string]bool)
	for _, n := range names {
		found[n] = true
	}
	if !found["alpha"] || !found["beta"] {
		t.Errorf("Names() = %v, want [alpha, beta]", names)
	}
}

func TestExecutionClusterManager_Reload(t *testing.T) {
	t.Run("adds new clusters", func(t *testing.T) {
		cfg := &ExecutionManagerConfig{
			Clusters: []ExecutionClusterConfig{
				{Name: "original"},
			},
		}

		mgr, _ := NewExecutionClusterManager(cfg)

		result, err := mgr.Reload([]ExecutionClusterConfig{
			{Name: "original"},
			{Name: "new-cluster"},
		})
		if err != nil {
			t.Fatalf("Reload() error = %v", err)
		}

		if len(result.Added) != 1 || result.Added[0] != "new-cluster" {
			t.Errorf("Added = %v, want [new-cluster]", result.Added)
		}

		if mgr.Count() != 2 {
			t.Errorf("Count() = %d after reload, want 2", mgr.Count())
		}
	})

	t.Run("removes old clusters", func(t *testing.T) {
		cfg := &ExecutionManagerConfig{
			Clusters: []ExecutionClusterConfig{
				{Name: "keep"},
				{Name: "remove"},
			},
		}

		mgr, _ := NewExecutionClusterManager(cfg)

		result, err := mgr.Reload([]ExecutionClusterConfig{
			{Name: "keep"},
		})
		if err != nil {
			t.Fatalf("Reload() error = %v", err)
		}

		if len(result.Removed) != 1 || result.Removed[0] != "remove" {
			t.Errorf("Removed = %v, want [remove]", result.Removed)
		}

		if mgr.Get("remove") != nil {
			t.Error("Removed cluster should not be accessible")
		}
	})

	t.Run("updates changed clusters", func(t *testing.T) {
		cfg := &ExecutionManagerConfig{
			Clusters: []ExecutionClusterConfig{
				{Name: "cluster", Namespace: "old-ns"},
			},
		}

		mgr, _ := NewExecutionClusterManager(cfg)

		result, err := mgr.Reload([]ExecutionClusterConfig{
			{Name: "cluster", Namespace: "new-ns"},
		})
		if err != nil {
			t.Fatalf("Reload() error = %v", err)
		}

		if len(result.Updated) != 1 || result.Updated[0] != "cluster" {
			t.Errorf("Updated = %v, want [cluster]", result.Updated)
		}

		cluster := mgr.Get("cluster")
		if cluster.Namespace != "new-ns" {
			t.Errorf("Namespace = %v after reload, want new-ns", cluster.Namespace)
		}
	})

	t.Run("updates default cluster", func(t *testing.T) {
		cfg := &ExecutionManagerConfig{
			Clusters: []ExecutionClusterConfig{
				{Name: "first"},
				{Name: "second"},
			},
		}

		mgr, _ := NewExecutionClusterManager(cfg)
		if mgr.DefaultClusterName() != "first" {
			t.Errorf("Initial default = %v, want first", mgr.DefaultClusterName())
		}

		// Reload with different order
		_, err := mgr.Reload([]ExecutionClusterConfig{
			{Name: "second"},
			{Name: "first"},
		})
		if err != nil {
			t.Fatalf("Reload() error = %v", err)
		}

		if mgr.DefaultClusterName() != "second" {
			t.Errorf("Default after reload = %v, want second", mgr.DefaultClusterName())
		}
	})

	t.Run("rejects duplicates in reload", func(t *testing.T) {
		cfg := &ExecutionManagerConfig{
			Clusters: []ExecutionClusterConfig{
				{Name: "cluster"},
			},
		}

		mgr, _ := NewExecutionClusterManager(cfg)

		_, err := mgr.Reload([]ExecutionClusterConfig{
			{Name: "dup"},
			{Name: "dup"},
		})
		if err == nil {
			t.Fatal("Expected error for duplicate names in reload")
		}
	})

	t.Run("handles reload to empty", func(t *testing.T) {
		cfg := &ExecutionManagerConfig{
			Clusters: []ExecutionClusterConfig{
				{Name: "cluster-1"},
				{Name: "cluster-2"},
			},
		}

		mgr, _ := NewExecutionClusterManager(cfg)

		result, err := mgr.Reload([]ExecutionClusterConfig{})
		if err != nil {
			t.Fatalf("Reload() error = %v", err)
		}

		if len(result.Removed) != 2 {
			t.Errorf("Removed = %d, want 2", len(result.Removed))
		}

		if mgr.Count() != 0 {
			t.Errorf("Count() = %d after reload to empty, want 0", mgr.Count())
		}

		if mgr.DefaultClusterName() != "" {
			t.Errorf("Default should be empty after reload to empty")
		}
	})
}

func TestExecutionConfigChanged(t *testing.T) {
	base := &ExecutionClusterConfig{
		Name:                "test",
		KubeconfigPath:      "/path/to/kubeconfig",
		Namespace:           "ns",
		RunnerImage:         "image:v1",
		ImagePullPolicy:     "Always",
		Timeout:             600,
		MemoryLimit:         "2Gi",
		CPULimit:            "1",
		CleanupTTL:          3600,
		MaxConcurrentAgents: 5,
	}

	tests := []struct {
		name    string
		modify  func(c *ExecutionClusterConfig)
		changed bool
	}{
		{
			name:    "no change",
			modify:  func(c *ExecutionClusterConfig) {},
			changed: false,
		},
		{
			name:    "kubeconfig changed",
			modify:  func(c *ExecutionClusterConfig) { c.KubeconfigPath = "/new/path" },
			changed: true,
		},
		{
			name:    "namespace changed",
			modify:  func(c *ExecutionClusterConfig) { c.Namespace = "new-ns" },
			changed: true,
		},
		{
			name:    "image changed",
			modify:  func(c *ExecutionClusterConfig) { c.RunnerImage = "image:v2" },
			changed: true,
		},
		{
			name:    "pull policy changed",
			modify:  func(c *ExecutionClusterConfig) { c.ImagePullPolicy = "Never" },
			changed: true,
		},
		{
			name:    "timeout changed",
			modify:  func(c *ExecutionClusterConfig) { c.Timeout = 900 },
			changed: true,
		},
		{
			name:    "memory changed",
			modify:  func(c *ExecutionClusterConfig) { c.MemoryLimit = "4Gi" },
			changed: true,
		},
		{
			name:    "cpu changed",
			modify:  func(c *ExecutionClusterConfig) { c.CPULimit = "2" },
			changed: true,
		},
		{
			name:    "cleanup ttl changed",
			modify:  func(c *ExecutionClusterConfig) { c.CleanupTTL = 7200 },
			changed: true,
		},
		{
			name:    "max concurrent changed",
			modify:  func(c *ExecutionClusterConfig) { c.MaxConcurrentAgents = 10 },
			changed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of base
			modified := &ExecutionClusterConfig{
				Name:                base.Name,
				KubeconfigPath:      base.KubeconfigPath,
				Namespace:           base.Namespace,
				RunnerImage:         base.RunnerImage,
				ImagePullPolicy:     base.ImagePullPolicy,
				Timeout:             base.Timeout,
				MemoryLimit:         base.MemoryLimit,
				CPULimit:            base.CPULimit,
				CleanupTTL:          base.CleanupTTL,
				MaxConcurrentAgents: base.MaxConcurrentAgents,
			}

			tt.modify(modified)

			if executionConfigChanged(base, modified) != tt.changed {
				t.Errorf("executionConfigChanged() = %v, want %v", !tt.changed, tt.changed)
			}
		})
	}
}
