package k8s

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewClient_WithExplicitKubeconfig(t *testing.T) {
	// Create a temporary kubeconfig file
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")

	// Write a minimal valid kubeconfig
	kubeconfigContent := `apiVersion: v1
kind: Config
clusters:
- name: test-cluster
  cluster:
    server: https://test-cluster:6443
    insecure-skip-tls-verify: true
contexts:
- name: test-context
  context:
    cluster: test-cluster
    user: test-user
current-context: test-context
users:
- name: test-user
  user:
    token: test-token
`
	if err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600); err != nil {
		t.Fatalf("failed to create test kubeconfig: %v", err)
	}

	// Test client creation with explicit kubeconfig
	client, err := NewClient(ClientConfig{
		Kubeconfig: kubeconfigPath,
	})
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	if client == nil {
		t.Fatal("NewClient() returned nil client")
	}

	if client.Clientset() == nil {
		t.Error("Clientset() returned nil")
	}

	if client.Config() == nil {
		t.Error("Config() returned nil")
	}

	if client.IsInCluster() {
		t.Error("IsInCluster() should be false for out-of-cluster config")
	}

	// Verify the config has the expected server
	if client.Config().Host != "https://test-cluster:6443" {
		t.Errorf("Config().Host = %s, want https://test-cluster:6443", client.Config().Host)
	}
}

func TestNewClient_WithInvalidKubeconfig(t *testing.T) {
	// Test with non-existent file
	_, err := NewClient(ClientConfig{
		Kubeconfig: "/nonexistent/kubeconfig",
	})
	if err == nil {
		t.Error("NewClient() should fail with non-existent kubeconfig")
	}
}

func TestNewClient_WithMalformedKubeconfig(t *testing.T) {
	// Create a temporary kubeconfig file with invalid content
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")

	if err := os.WriteFile(kubeconfigPath, []byte("invalid yaml content"), 0600); err != nil {
		t.Fatalf("failed to create test kubeconfig: %v", err)
	}

	_, err := NewClient(ClientConfig{
		Kubeconfig: kubeconfigPath,
	})
	if err == nil {
		t.Error("NewClient() should fail with malformed kubeconfig")
	}
}

func TestBuildConfig_ExplicitPath(t *testing.T) {
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")

	kubeconfigContent := `apiVersion: v1
kind: Config
clusters:
- name: test-cluster
  cluster:
    server: https://explicit-path:6443
    insecure-skip-tls-verify: true
contexts:
- name: test-context
  context:
    cluster: test-cluster
    user: test-user
current-context: test-context
users:
- name: test-user
  user:
    token: test-token
`
	if err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600); err != nil {
		t.Fatalf("failed to create test kubeconfig: %v", err)
	}

	config, inCluster, err := buildConfig(kubeconfigPath)
	if err != nil {
		t.Fatalf("buildConfig() failed: %v", err)
	}

	if config == nil {
		t.Fatal("buildConfig() returned nil config")
	}

	if inCluster {
		t.Error("buildConfig() should return inCluster=false for explicit path")
	}

	if config.Host != "https://explicit-path:6443" {
		t.Errorf("config.Host = %s, want https://explicit-path:6443", config.Host)
	}
}

func TestBuildConfig_EnvironmentVariable(t *testing.T) {
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")

	kubeconfigContent := `apiVersion: v1
kind: Config
clusters:
- name: test-cluster
  cluster:
    server: https://env-var:6443
    insecure-skip-tls-verify: true
contexts:
- name: test-context
  context:
    cluster: test-cluster
    user: test-user
current-context: test-context
users:
- name: test-user
  user:
    token: test-token
`
	if err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600); err != nil {
		t.Fatalf("failed to create test kubeconfig: %v", err)
	}

	// Set KUBECONFIG env var
	oldEnv := os.Getenv("KUBECONFIG")
	defer os.Setenv("KUBECONFIG", oldEnv)
	os.Setenv("KUBECONFIG", kubeconfigPath)

	config, inCluster, err := buildConfig("")
	if err != nil {
		t.Fatalf("buildConfig() failed: %v", err)
	}

	if config == nil {
		t.Fatal("buildConfig() returned nil config")
	}

	if inCluster {
		t.Error("buildConfig() should return inCluster=false for KUBECONFIG env var")
	}

	if config.Host != "https://env-var:6443" {
		t.Errorf("config.Host = %s, want https://env-var:6443", config.Host)
	}
}

func TestBuildConfig_NoConfigFound(t *testing.T) {
	// Clear KUBECONFIG env var
	oldEnv := os.Getenv("KUBECONFIG")
	defer os.Setenv("KUBECONFIG", oldEnv)
	os.Unsetenv("KUBECONFIG")

	// Use a non-existent home directory to avoid finding ~/.kube/config
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)

	_, _, err := buildConfig("")
	if err == nil {
		t.Error("buildConfig() should fail when no config is found")
	}
}

func TestBuildConfig_DefaultLocation(t *testing.T) {
	// Create a temporary home directory with .kube/config
	tmpHome := t.TempDir()
	kubeDir := filepath.Join(tmpHome, ".kube")
	if err := os.MkdirAll(kubeDir, 0755); err != nil {
		t.Fatalf("failed to create .kube directory: %v", err)
	}

	kubeconfigPath := filepath.Join(kubeDir, "config")
	kubeconfigContent := `apiVersion: v1
kind: Config
clusters:
- name: test-cluster
  cluster:
    server: https://default-location:6443
    insecure-skip-tls-verify: true
contexts:
- name: test-context
  context:
    cluster: test-cluster
    user: test-user
current-context: test-context
users:
- name: test-user
  user:
    token: test-token
`
	if err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600); err != nil {
		t.Fatalf("failed to create test kubeconfig: %v", err)
	}

	// Clear KUBECONFIG env var and set HOME to tmpHome
	oldEnv := os.Getenv("KUBECONFIG")
	oldHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("KUBECONFIG", oldEnv)
		os.Setenv("HOME", oldHome)
	}()
	os.Unsetenv("KUBECONFIG")
	os.Setenv("HOME", tmpHome)

	config, inCluster, err := buildConfig("")
	if err != nil {
		t.Fatalf("buildConfig() failed: %v", err)
	}

	if config == nil {
		t.Fatal("buildConfig() returned nil config")
	}

	if inCluster {
		t.Error("buildConfig() should return inCluster=false for default location")
	}

	if config.Host != "https://default-location:6443" {
		t.Errorf("config.Host = %s, want https://default-location:6443", config.Host)
	}
}

func TestBuildConfig_InClusterDetection(t *testing.T) {
	// Note: This test cannot easily verify in-cluster config without mocking
	// the filesystem at /var/run/secrets/kubernetes.io/serviceaccount/token
	// In real in-cluster scenarios, rest.InClusterConfig() will work.
	// Here we just test that the logic path exists.

	// For a real in-cluster test, you would need to:
	// 1. Create the service account token file at the expected location
	// 2. Set required environment variables (KUBERNETES_SERVICE_HOST, etc.)
	// This is typically done in integration tests with a real cluster.

	t.Skip("Skipping in-cluster detection test - requires mocked filesystem or real cluster")
}
