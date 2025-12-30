// Package bootstrap provides automatic provisioning of Kubernetes resources
// required for Nightcrier operation. It handles namespace, RBAC, and secret creation
// in an idempotent manner during application startup.
package bootstrap

import (
	"context"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
)

// ensureAPIKeysSecret creates a Secret containing AI provider API keys if it doesn't exist.
// The Secret is named "ai-api-keys" and contains three keys: anthropic, openai, and gemini.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - client: Kubernetes clientset for API operations
//   - namespace: Target namespace for the Secret
//   - anthropic: Anthropic API key value (can be empty)
//   - openai: OpenAI API key value (can be empty)
//   - gemini: Gemini API key value (can be empty)
//
// Validation:
//   - At least one API key must be non-empty
//   - If all keys are empty, returns an error
//
// Behavior:
//   - Checks if Secret "ai-api-keys" exists
//   - If exists, returns nil (idempotent, never updates)
//   - If not exists, creates Secret with all three keys
//   - Empty key values are stored as empty strings
//
// Returns error if validation fails, Secret GET fails (except NotFound), or creation fails.
func ensureAPIKeysSecret(ctx context.Context, client kubernetes.Interface, namespace, anthropic, openai, gemini string) error {
	// Validate at least one key is non-empty
	if anthropic == "" && openai == "" && gemini == "" {
		return fmt.Errorf("at least one API key must be non-empty (anthropic, openai, or gemini)")
	}

	secretName := "ai-api-keys"

	// Check if Secret already exists
	_, err := client.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err == nil {
		// Secret exists, skip creation (idempotent)
		return nil
	}

	// If error is not NotFound, return the error
	if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check if Secret %s exists: %w", secretName, err)
	}

	// Secret doesn't exist, create it
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":        "nightcrier",
				"managed-by": "nightcrier-bootstrap",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"anthropic": []byte(anthropic),
			"openai":    []byte(openai),
			"gemini":    []byte(gemini),
		},
	}

	_, err = client.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create Secret %s: %w", secretName, err)
	}

	return nil
}

// ensureKubeconfigSecret creates a Secret containing kubeconfig file contents if it doesn't exist.
// The Secret is named "kubeconfig-{clusterName}" and contains a single key "config" with file contents.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - client: Kubernetes clientset for API operations
//   - namespace: Target namespace for the Secret
//   - clusterName: Name of the cluster (used in Secret name and labels)
//   - kubeconfigPath: File path to the kubeconfig file
//
// Validation:
//   - File must exist at kubeconfigPath
//   - File must be readable
//   - Returns clear error message if file is missing or unreadable
//
// Behavior:
//   - Validates file exists and reads contents into memory
//   - Checks if Secret "kubeconfig-{clusterName}" exists
//   - If exists, returns nil (idempotent, never updates)
//   - If not exists, creates Secret with file contents
//
// Labels applied:
//   - app=nightcrier: Identifies resource ownership
//   - cluster={clusterName}: Links to specific cluster
//
// Returns error if file validation fails, Secret GET fails (except NotFound), or creation fails.
func ensureKubeconfigSecret(ctx context.Context, client kubernetes.Interface, namespace, clusterName, kubeconfigPath string) error {
	// Validate file exists
	fileInfo, err := os.Stat(kubeconfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("kubeconfig file not found at path: %s", kubeconfigPath)
		}
		return fmt.Errorf("failed to access kubeconfig file at %s: %w", kubeconfigPath, err)
	}

	// Check if file is readable
	if fileInfo.IsDir() {
		return fmt.Errorf("kubeconfig path is a directory, expected a file: %s", kubeconfigPath)
	}

	// Read file contents
	fileContents, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to read kubeconfig file at %s: %w", kubeconfigPath, err)
	}

	// Validate file is not empty
	if len(fileContents) == 0 {
		return fmt.Errorf("kubeconfig file is empty: %s", kubeconfigPath)
	}

	secretName := fmt.Sprintf("kubeconfig-%s", clusterName)

	// Check if Secret already exists
	_, err = client.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err == nil {
		// Secret exists, skip creation (idempotent)
		return nil
	}

	// If error is not NotFound, return the error
	if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check if Secret %s exists: %w", secretName, err)
	}

	// Secret doesn't exist, create it
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":        "nightcrier",
				"cluster":    clusterName,
				"managed-by": "nightcrier-bootstrap",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"config": fileContents,
		},
	}

	_, err = client.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create Secret %s: %w", secretName, err)
	}

	return nil
}
