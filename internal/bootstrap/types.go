// Package bootstrap provides automatic provisioning of Kubernetes resources
// required for Nightcrier to run.
package bootstrap

// Config holds the configuration for bootstrapping Kubernetes resources.
type Config struct {
	// Namespace is the Kubernetes namespace where resources will be created.
	Namespace string

	// AnthropicAPIKey is the API key for Anthropic Claude.
	AnthropicAPIKey string

	// OpenAIAPIKey is the API key for OpenAI GPT models.
	OpenAIAPIKey string

	// GeminiAPIKey is the API key for Google Gemini models.
	GeminiAPIKey string

	// MonitoredClusters is the list of monitored clusters to create triage kubeconfig secrets for.
	// These kubeconfigs are mounted into agent pods for triage access to target clusters.
	MonitoredClusters []MonitoredClusterConfig
}

// MonitoredClusterConfig holds configuration for a monitored cluster's triage kubeconfig secret.
// This is separate from execution clusters - execution cluster kubeconfigs are used directly
// by Nightcrier to create Jobs, not mounted into agent pods.
type MonitoredClusterConfig struct {
	// Name is the cluster name, used in the secret name: triage-kubeconfig-{name}
	Name string

	// TargetKubeconfigPath is the path to the kubeconfig file for triage agent access
	// to the monitored cluster. This file will be read and stored in a Kubernetes Secret
	// that gets mounted into agent pods.
	TargetKubeconfigPath string
}

// Result tracks the outcome of the bootstrap process.
type Result struct {
	// NamespaceCreated indicates whether the namespace was created (true) or already existed (false).
	NamespaceCreated bool

	// ServiceAccountCreated indicates whether the ServiceAccount was created.
	ServiceAccountCreated bool

	// RoleCreated indicates whether the Role was created.
	RoleCreated bool

	// RoleBindingCreated indicates whether the RoleBinding was created.
	RoleBindingCreated bool

	// APIKeysSecretCreated indicates whether the ai-api-keys Secret was created.
	APIKeysSecretCreated bool

	// TriageKubeconfigSecretsCreated is a map of cluster name to whether the Secret was created.
	TriageKubeconfigSecretsCreated map[string]bool
}

// CreatedCount returns the total number of resources created during bootstrap.
func (r *Result) CreatedCount() int {
	count := 0
	if r.NamespaceCreated {
		count++
	}
	if r.ServiceAccountCreated {
		count++
	}
	if r.RoleCreated {
		count++
	}
	if r.RoleBindingCreated {
		count++
	}
	if r.APIKeysSecretCreated {
		count++
	}
	for _, created := range r.TriageKubeconfigSecretsCreated {
		if created {
			count++
		}
	}
	return count
}

// ExistingCount returns the total number of resources that already existed.
func (r *Result) ExistingCount() int {
	// Total possible resources: namespace, SA, role, rolebinding, api-keys secret, + N triage kubeconfig secrets
	total := 5 + len(r.TriageKubeconfigSecretsCreated)
	return total - r.CreatedCount()
}
