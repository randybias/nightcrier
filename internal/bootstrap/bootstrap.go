package bootstrap

import (
	"context"
	"fmt"

	"k8s.io/client-go/kubernetes"
)

// Bootstrapper defines the interface for bootstrapping Kubernetes resources.
type Bootstrapper interface {
	// Bootstrap creates all required Kubernetes resources for Nightcrier.
	// It is idempotent - resources that already exist will be skipped.
	// Returns a Result indicating which resources were created vs. already existing.
	Bootstrap(ctx context.Context) (*Result, error)
}

// Manager implements the Bootstrapper interface and orchestrates the creation
// of all required Kubernetes resources.
type Manager struct {
	kubeClient kubernetes.Interface
	config     Config
}

// NewManager creates a new bootstrap Manager.
func NewManager(kubeClient kubernetes.Interface, config Config) *Manager {
	return &Manager{
		kubeClient: kubeClient,
		config:     config,
	}
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
