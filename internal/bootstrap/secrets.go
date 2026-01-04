// Package bootstrap provides automatic provisioning of Kubernetes resources
// required for Nightcrier operation. It handles namespace, RBAC, and secret creation
// in an idempotent manner during application startup.
package bootstrap

import (
	"context"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// ensureTriageKubeconfigSecret creates or updates a Secret containing kubeconfig file contents
// for triage agent access to a monitored cluster. The Secret is named "triage-kubeconfig-{clusterName}"
// and contains a single key "config" with file contents.
//
// This function is specifically for monitored cluster target kubeconfigs that get mounted into
// agent pods. Execution cluster kubeconfigs are used directly by Nightcrier and are not handled here.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - client: Kubernetes clientset for API operations
//   - namespace: Target namespace for the Secret
//   - clusterName: Name of the monitored cluster (used in Secret name and labels)
//   - targetKubeconfigPath: File path to the kubeconfig file for triage access
//
// Validation:
//   - File must exist at targetKubeconfigPath
//   - File must be readable
//   - Returns clear error message if file is missing or unreadable
//
// Behavior:
//   - Validates file exists and reads contents into memory
//   - Checks if Secret "triage-kubeconfig-{clusterName}" exists
//   - If exists and content differs, updates Secret with new file contents
//   - If exists and content is same, skips update (idempotent)
//   - If not exists, creates Secret with file contents
//
// Note: This ensures transient credentials (like time-bounded tokens) are always kept fresh
//
// Labels applied:
//   - app=nightcrier: Identifies resource ownership
//   - cluster={clusterName}: Links to specific monitored cluster
//   - purpose=triage: Identifies this as a triage kubeconfig (used by agents)
//
// Returns error if file validation fails, Secret GET fails (except NotFound), or creation fails.
func ensureTriageKubeconfigSecret(ctx context.Context, client kubernetes.Interface, namespace, clusterName, targetKubeconfigPath string) error {
	// Validate file exists
	fileInfo, err := os.Stat(targetKubeconfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("target kubeconfig file not found at path: %s", targetKubeconfigPath)
		}
		return fmt.Errorf("failed to access target kubeconfig file at %s: %w", targetKubeconfigPath, err)
	}

	// Check if file is readable
	if fileInfo.IsDir() {
		return fmt.Errorf("target kubeconfig path is a directory, expected a file: %s", targetKubeconfigPath)
	}

	// Read file contents
	fileContents, err := os.ReadFile(targetKubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to read target kubeconfig file at %s: %w", targetKubeconfigPath, err)
	}

	// Validate file is not empty
	if len(fileContents) == 0 {
		return fmt.Errorf("target kubeconfig file is empty: %s", targetKubeconfigPath)
	}

	secretName := fmt.Sprintf("triage-kubeconfig-%s", clusterName)

	// Check if Secret already exists
	existingSecret, err := client.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err == nil {
		// Secret exists - check if content has changed
		existingContent, ok := existingSecret.Data["config"]
		if ok && string(existingContent) == string(fileContents) {
			// Content unchanged, skip update
			return nil
		}

		// Content has changed - update the secret
		existingSecret.Data["config"] = fileContents
		_, err = client.CoreV1().Secrets(namespace).Update(ctx, existingSecret, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update Secret %s with new target kubeconfig: %w", secretName, err)
		}
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
				"purpose":    "triage",
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
