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

	// Clusters is the list of clusters to create kubeconfig secrets for.
	Clusters []ClusterConfig
}

// ClusterConfig holds configuration for a single cluster's kubeconfig secret.
type ClusterConfig struct {
	// Name is the cluster name, used in the secret name: kubeconfig-{name}
	Name string

	// TriageKubeconfig is the path to the kubeconfig file for triage agent access.
	// This file will be read and stored in a Kubernetes Secret.
	TriageKubeconfig string
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

	// KubeconfigSecretsCreated is a map of cluster name to whether the Secret was created.
	KubeconfigSecretsCreated map[string]bool
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
	for _, created := range r.KubeconfigSecretsCreated {
		if created {
			count++
		}
	}
	return count
}

// ExistingCount returns the total number of resources that already existed.
func (r *Result) ExistingCount() int {
	// Total possible resources: namespace, SA, role, rolebinding, api-keys secret, + N kubeconfig secrets
	total := 5 + len(r.KubeconfigSecretsCreated)
	return total - r.CreatedCount()
}
