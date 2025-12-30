// Package bootstrap provides automated Kubernetes resource provisioning for Nightcrier.
package bootstrap

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ensureServiceAccount checks if the ServiceAccount "nightcrier-executor" exists
// and creates it if missing. It is idempotent and will skip creation if the
// ServiceAccount already exists.
func ensureServiceAccount(ctx context.Context, client kubernetes.Interface, namespace string) error {
	saName := "nightcrier-executor"

	// Check if ServiceAccount exists
	_, err := client.CoreV1().ServiceAccounts(namespace).Get(ctx, saName, metav1.GetOptions{})
	if err == nil {
		// ServiceAccount already exists, skip creation
		return nil
	}

	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to check ServiceAccount %s: %w", saName, err)
	}

	// Create ServiceAccount
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":       "nightcrier",
				"component": "executor",
			},
		},
	}

	_, err = client.CoreV1().ServiceAccounts(namespace).Create(ctx, sa, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create ServiceAccount %s: %w", saName, err)
	}

	return nil
}

// ensureRole checks if the Role "nightcrier-executor" exists and creates it
// if missing. The Role includes permissions matching deploy/dev/rbac.yaml:
// - configmaps: create, get, list, delete
// - jobs (batch): create, get, list, watch, delete
// - pods: get, list, watch
// - pods/log: get
// - secrets: get, list
// This function is idempotent and will skip creation if the Role already exists.
func ensureRole(ctx context.Context, client kubernetes.Interface, namespace string) error {
	roleName := "nightcrier-executor"

	// Check if Role exists
	_, err := client.RbacV1().Roles(namespace).Get(ctx, roleName, metav1.GetOptions{})
	if err == nil {
		// Role already exists, skip creation
		return nil
	}

	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to check Role %s: %w", roleName, err)
	}

	// Create Role with exact permissions from deploy/dev/rbac.yaml
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":       "nightcrier",
				"component": "executor",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				// ConfigMap permissions (for incident data)
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"create", "get", "list", "delete"},
			},
			{
				// Job permissions (for agent execution)
				APIGroups: []string{"batch"},
				Resources: []string{"jobs"},
				Verbs:     []string{"create", "get", "list", "watch", "delete"},
			},
			{
				// Pod permissions (for log retrieval and status)
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				// Pod logs (for streaming agent output)
				APIGroups: []string{""},
				Resources: []string{"pods/log"},
				Verbs:     []string{"get"},
			},
			{
				// Secret permissions (read-only for API keys and kubeconfigs)
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list"},
			},
		},
	}

	_, err = client.RbacV1().Roles(namespace).Create(ctx, role, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create Role %s: %w", roleName, err)
	}

	return nil
}

// ensureRoleBinding checks if the RoleBinding "nightcrier-executor" exists
// and creates it if missing. The RoleBinding binds the ServiceAccount to
// the Role. It is idempotent and will skip creation if the RoleBinding
// already exists.
func ensureRoleBinding(ctx context.Context, client kubernetes.Interface, namespace string) error {
	rbName := "nightcrier-executor"
	saName := "nightcrier-executor"
	roleName := "nightcrier-executor"

	// Check if RoleBinding exists
	_, err := client.RbacV1().RoleBindings(namespace).Get(ctx, rbName, metav1.GetOptions{})
	if err == nil {
		// RoleBinding already exists, skip creation
		return nil
	}

	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to check RoleBinding %s: %w", rbName, err)
	}

	// Create RoleBinding
	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":       "nightcrier",
				"component": "executor",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     roleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      saName,
				Namespace: namespace,
			},
		},
	}

	_, err = client.RbacV1().RoleBindings(namespace).Create(ctx, rb, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create RoleBinding %s: %w", rbName, err)
	}

	return nil
}
