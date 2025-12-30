package bootstrap

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ensureNamespace ensures the specified namespace exists.
// It first checks if the namespace exists via GET.
// If not found (404), creates it with the label app=nightcrier.
// Returns true if the namespace was created, false if it already existed.
// This function is idempotent - calling it multiple times is safe.
func (m *Manager) ensureNamespace(ctx context.Context, name string) (bool, error) {
	return ensureNamespace(ctx, m.kubeClient, name)
}

// ensureNamespace is the implementation function that can be tested independently.
// It checks if a namespace exists and creates it if not found.
func ensureNamespace(ctx context.Context, client kubernetes.Interface, name string) (bool, error) {
	// Check if namespace already exists
	_, err := client.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		// Namespace already exists, nothing to do
		return false, nil
	}

	// If error is not NotFound, return the error
	if !apierrors.IsNotFound(err) {
		return false, fmt.Errorf("failed to check if namespace exists: %w", err)
	}

	// Namespace doesn't exist, create it
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app":        "nightcrier",
				"managed-by": "nightcrier-bootstrap",
			},
		},
	}

	_, err = client.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to create namespace: %w", err)
	}

	return true, nil
}
