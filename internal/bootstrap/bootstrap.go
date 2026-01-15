package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"k8s.io/client-go/kubernetes"
)

// Bootstrapper defines the interface for bootstrapping Kubernetes resources.
type Bootstrapper interface {
	// Bootstrap creates all required Kubernetes resources for Nightcrier.
	// It is idempotent - resources that already exist will be skipped.
	// Returns a Result indicating which resources were created vs. already existing.
	Bootstrap(ctx context.Context) (*Result, error)

	// BootstrapNonBlocking attempts to bootstrap all resources and returns status.
	// Unlike Bootstrap, this method never returns an error - failures are tracked
	// in the returned BootstrapStatus and can be retried later.
	BootstrapNonBlocking(ctx context.Context) *BootstrapStatus
}

// Manager implements the Bootstrapper interface and orchestrates the creation
// of all required Kubernetes resources.
type Manager struct {
	kubeClient kubernetes.Interface
	config     Config

	// status tracks the current bootstrap state
	status *BootstrapStatus
	mu     sync.RWMutex
}

// NewManager creates a new bootstrap Manager.
func NewManager(kubeClient kubernetes.Interface, config Config) *Manager {
	return &Manager{
		kubeClient: kubeClient,
		config:     config,
		status:     NewBootstrapStatus(),
	}
}

// GetStatus returns a copy of the current bootstrap status.
func (m *Manager) GetStatus() *BootstrapStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status.Clone()
}

// Bootstrap orchestrates the creation of all required Kubernetes resources.
// It performs the following steps in order:
//  1. Ensure namespace exists
//  2. Ensure RBAC resources exist (ServiceAccount, Role, RoleBinding)
//  3. Ensure API keys secret exists
//  4. Ensure triage kubeconfig secrets exist for each monitored cluster
//
// All operations are idempotent - existing resources will not be modified.
// Note: Execution cluster kubeconfigs are used directly by Nightcrier to create Jobs,
// not mounted into agent pods, so they are not handled here.
//
// Deprecated: Use BootstrapNonBlocking for resilient startup.
func (m *Manager) Bootstrap(ctx context.Context) (*Result, error) {
	result := &Result{
		TriageKubeconfigSecretsCreated: make(map[string]bool),
	}

	// Step 1: Ensure namespace exists
	namespaceCreated, err := ensureNamespace(ctx, m.kubeClient, m.config.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure namespace: %w", err)
	}
	result.NamespaceCreated = namespaceCreated

	// Step 2: Ensure RBAC resources exist
	if err := ensureServiceAccount(ctx, m.kubeClient, m.config.Namespace); err != nil {
		return nil, fmt.Errorf("failed to ensure ServiceAccount: %w", err)
	}

	if err := ensureRole(ctx, m.kubeClient, m.config.Namespace); err != nil {
		return nil, fmt.Errorf("failed to ensure Role: %w", err)
	}

	if err := ensureRoleBinding(ctx, m.kubeClient, m.config.Namespace); err != nil {
		return nil, fmt.Errorf("failed to ensure RoleBinding: %w", err)
	}

	// Step 3: Ensure API keys secret exists
	if err := ensureAPIKeysSecret(ctx, m.kubeClient, m.config.Namespace,
		m.config.AnthropicAPIKey, m.config.OpenAIAPIKey, m.config.GeminiAPIKey); err != nil {
		return nil, fmt.Errorf("failed to ensure API keys secret: %w", err)
	}

	// Step 4: Ensure triage kubeconfig secrets exist for each monitored cluster
	// These secrets are mounted into agent pods for triage access to target clusters
	for _, cluster := range m.config.MonitoredClusters {
		if cluster.TargetKubeconfigPath == "" {
			continue // Skip clusters without target kubeconfig
		}

		if err := ensureTriageKubeconfigSecret(ctx, m.kubeClient, m.config.Namespace,
			cluster.Name, cluster.TargetKubeconfigPath); err != nil {
			return nil, fmt.Errorf("failed to ensure triage kubeconfig secret for cluster %s: %w", cluster.Name, err)
		}
	}

	return result, nil
}

// BootstrapNonBlocking attempts to bootstrap all resources without blocking on failures.
// Any failures are logged and tracked in the returned BootstrapStatus.
// The caller should start background retry for any failed components.
//
// This method:
//  1. Bootstraps global resources (namespace, RBAC) - these are blocking prerequisites
//  2. Attempts API keys secret creation (non-fatal if no keys configured)
//  3. Bootstraps per-cluster resources in parallel (non-fatal failures)
//
// Returns a BootstrapStatus that tracks which components are ready vs degraded.
func (m *Manager) BootstrapNonBlocking(ctx context.Context) *BootstrapStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	slog.Info("starting non-blocking bootstrap",
		"namespace", m.config.Namespace,
		"clusters", len(m.config.MonitoredClusters))

	// Step 1: Bootstrap global resources (namespace, RBAC)
	// These are prerequisites - if they fail, we can't do anything else
	globalReady := m.bootstrapGlobalResources(ctx)
	m.status.SetGlobalReady(m.status.NamespaceReady, m.status.RBACReady)

	if !globalReady {
		slog.Warn("global bootstrap failed - will retry in background",
			"namespace_ready", m.status.NamespaceReady,
			"rbac_ready", m.status.RBACReady)
	}

	// Step 2: Bootstrap API keys secret (non-fatal)
	m.bootstrapAPIKeysSecret(ctx)

	// Step 3: Bootstrap per-cluster resources in parallel (non-fatal)
	m.bootstrapClusterResources(ctx)

	// Log summary
	ready := m.status.ReadyClusters()
	total := m.status.TotalClusters()
	slog.Info("non-blocking bootstrap complete",
		"state", m.status.State,
		"global_ready", m.status.GlobalReady,
		"api_keys_ready", m.status.APIKeysReady,
		"clusters_ready", fmt.Sprintf("%d/%d", ready, total))

	return m.status.Clone()
}

// bootstrapGlobalResources bootstraps namespace and RBAC resources.
// Returns true if all global resources are ready.
func (m *Manager) bootstrapGlobalResources(ctx context.Context) bool {
	// Namespace
	_, err := ensureNamespace(ctx, m.kubeClient, m.config.Namespace)
	if err != nil {
		slog.Error("failed to ensure namespace", "error", err)
		m.status.NamespaceReady = false
	} else {
		m.status.NamespaceReady = true
	}

	// RBAC resources - only attempt if namespace is ready
	if m.status.NamespaceReady {
		rbacReady := true

		if err := ensureServiceAccount(ctx, m.kubeClient, m.config.Namespace); err != nil {
			slog.Error("failed to ensure ServiceAccount", "error", err)
			rbacReady = false
		}

		if err := ensureRole(ctx, m.kubeClient, m.config.Namespace); err != nil {
			slog.Error("failed to ensure Role", "error", err)
			rbacReady = false
		}

		if err := ensureRoleBinding(ctx, m.kubeClient, m.config.Namespace); err != nil {
			slog.Error("failed to ensure RoleBinding", "error", err)
			rbacReady = false
		}

		m.status.RBACReady = rbacReady
	} else {
		m.status.RBACReady = false
	}

	return m.status.NamespaceReady && m.status.RBACReady
}

// bootstrapAPIKeysSecret attempts to create the API keys secret.
// This is non-fatal - if no API keys are configured, we just log a warning.
func (m *Manager) bootstrapAPIKeysSecret(ctx context.Context) {
	// Check if any API keys are configured
	hasKeys := m.config.AnthropicAPIKey != "" ||
		m.config.OpenAIAPIKey != "" ||
		m.config.GeminiAPIKey != ""

	if !hasKeys {
		slog.Warn("no API keys configured - agent dispatch will be disabled until keys are available")
		m.status.SetAPIKeysStatus(false, fmt.Errorf("no API keys configured"))
		return
	}

	// Only attempt if global resources are ready
	if !m.status.GlobalReady {
		slog.Debug("skipping API keys secret - global resources not ready")
		m.status.SetAPIKeysStatus(false, fmt.Errorf("waiting for global resources"))
		return
	}

	err := ensureAPIKeysSecret(ctx, m.kubeClient, m.config.Namespace,
		m.config.AnthropicAPIKey, m.config.OpenAIAPIKey, m.config.GeminiAPIKey)
	if err != nil {
		slog.Error("failed to ensure API keys secret", "error", err)
		m.status.SetAPIKeysStatus(false, err)
	} else {
		m.status.SetAPIKeysStatus(true, nil)
	}
}

// bootstrapClusterResources bootstraps per-cluster resources in parallel.
// Each cluster's failure is tracked independently.
func (m *Manager) bootstrapClusterResources(ctx context.Context) {
	// Initialize status for all clusters
	for _, cluster := range m.config.MonitoredClusters {
		m.status.SetClusterStatus(cluster.Name, false, nil)
	}

	// Only attempt if global resources are ready
	if !m.status.GlobalReady {
		slog.Debug("skipping cluster resources - global resources not ready")
		for _, cluster := range m.config.MonitoredClusters {
			m.status.SetClusterStatus(cluster.Name, false, fmt.Errorf("waiting for global resources"))
		}
		return
	}

	// Bootstrap each cluster's resources
	// Using a simple loop for now - could use errgroup for parallelism if needed
	var wg sync.WaitGroup
	for _, cluster := range m.config.MonitoredClusters {
		if cluster.TargetKubeconfigPath == "" {
			// No kubeconfig needed for this cluster
			m.status.SetClusterStatus(cluster.Name, true, nil)
			continue
		}

		wg.Add(1)
		go func(c MonitoredClusterConfig) {
			defer wg.Done()
			m.bootstrapSingleCluster(ctx, c)
		}(cluster)
	}
	wg.Wait()
}

// bootstrapSingleCluster bootstraps resources for a single cluster.
func (m *Manager) bootstrapSingleCluster(ctx context.Context, cluster MonitoredClusterConfig) {
	err := ensureTriageKubeconfigSecret(ctx, m.kubeClient, m.config.Namespace,
		cluster.Name, cluster.TargetKubeconfigPath)

	m.mu.Lock()
	defer m.mu.Unlock()

	if err != nil {
		slog.Warn("failed to bootstrap cluster resources",
			"cluster", cluster.Name,
			"error", err)
		m.status.SetClusterStatus(cluster.Name, false, err)
	} else {
		slog.Info("cluster resources bootstrapped",
			"cluster", cluster.Name)
		m.status.SetClusterStatus(cluster.Name, true, nil)
	}
}

// RetryCluster attempts to bootstrap a single cluster's resources.
// This is used by the background retry loop.
func (m *Manager) RetryCluster(ctx context.Context, clusterName string) error {
	// Find the cluster config
	var cluster *MonitoredClusterConfig
	for i := range m.config.MonitoredClusters {
		if m.config.MonitoredClusters[i].Name == clusterName {
			cluster = &m.config.MonitoredClusters[i]
			break
		}
	}

	if cluster == nil {
		return fmt.Errorf("cluster %s not found in config", clusterName)
	}

	if cluster.TargetKubeconfigPath == "" {
		m.mu.Lock()
		m.status.SetClusterStatus(clusterName, true, nil)
		m.mu.Unlock()
		return nil
	}

	err := ensureTriageKubeconfigSecret(ctx, m.kubeClient, m.config.Namespace,
		cluster.Name, cluster.TargetKubeconfigPath)

	m.mu.Lock()
	defer m.mu.Unlock()

	if err != nil {
		m.status.SetClusterStatus(clusterName, false, err)
		return err
	}

	m.status.SetClusterStatus(clusterName, true, nil)
	slog.Info("cluster bootstrap recovered", "cluster", clusterName)
	return nil
}

// RetryGlobal attempts to bootstrap global resources.
// This is used by the background retry loop.
func (m *Manager) RetryGlobal(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.bootstrapGlobalResources(ctx) {
		slog.Info("global bootstrap recovered")
		return nil
	}

	return fmt.Errorf("global resources not ready")
}

// RetryAPIKeys attempts to bootstrap the API keys secret.
// This is used by the background retry loop.
func (m *Manager) RetryAPIKeys(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.bootstrapAPIKeysSecret(ctx)
	if m.status.APIKeysReady {
		slog.Info("API keys bootstrap recovered")
		return nil
	}

	return m.status.APIKeysError
}
