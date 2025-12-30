package bootstrap

import (
	"context"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestEnsureServiceAccount(t *testing.T) {
	t.Run("creates ServiceAccount when it doesn't exist", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		ctx := context.Background()
		namespace := "nightcrier"

		err := ensureServiceAccount(ctx, client, namespace)
		if err != nil {
			t.Fatalf("ensureServiceAccount() failed: %v", err)
		}

		// Verify ServiceAccount was created
		sa, err := client.CoreV1().ServiceAccounts(namespace).Get(ctx, "nightcrier-executor", metav1.GetOptions{})
		if err != nil {
			t.Fatalf("failed to get ServiceAccount: %v", err)
		}

		if sa.Name != "nightcrier-executor" {
			t.Errorf("ServiceAccount name = %s, want nightcrier-executor", sa.Name)
		}

		if sa.Namespace != namespace {
			t.Errorf("ServiceAccount namespace = %s, want %s", sa.Namespace, namespace)
		}

		// Verify labels
		if sa.Labels["app"] != "nightcrier" {
			t.Errorf("ServiceAccount label app = %s, want nightcrier", sa.Labels["app"])
		}

		if sa.Labels["component"] != "executor" {
			t.Errorf("ServiceAccount label component = %s, want executor", sa.Labels["component"])
		}
	})

	t.Run("skips creation when ServiceAccount already exists", func(t *testing.T) {
		// Pre-create ServiceAccount
		existingSA := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nightcrier-executor",
				Namespace: "nightcrier",
				Labels: map[string]string{
					"app":       "nightcrier",
					"component": "executor",
					"custom":    "label",
				},
			},
		}
		client := fake.NewSimpleClientset(existingSA)
		ctx := context.Background()

		err := ensureServiceAccount(ctx, client, "nightcrier")
		if err != nil {
			t.Fatalf("ensureServiceAccount() failed: %v", err)
		}

		// Verify ServiceAccount still exists with original labels
		sa, err := client.CoreV1().ServiceAccounts("nightcrier").Get(ctx, "nightcrier-executor", metav1.GetOptions{})
		if err != nil {
			t.Fatalf("failed to get ServiceAccount: %v", err)
		}

		// Check that custom label is preserved (proving we didn't recreate)
		if sa.Labels["custom"] != "label" {
			t.Error("ServiceAccount was recreated instead of skipped")
		}
	})

	t.Run("is idempotent when called multiple times", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		ctx := context.Background()
		namespace := "nightcrier"

		// Call multiple times
		for i := 0; i < 3; i++ {
			err := ensureServiceAccount(ctx, client, namespace)
			if err != nil {
				t.Fatalf("ensureServiceAccount() call %d failed: %v", i+1, err)
			}
		}

		// Verify only one ServiceAccount exists
		saList, err := client.CoreV1().ServiceAccounts(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list ServiceAccounts: %v", err)
		}

		if len(saList.Items) != 1 {
			t.Errorf("ServiceAccount count = %d, want 1", len(saList.Items))
		}
	})

	t.Run("returns error when Get fails with non-NotFound error", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		ctx := context.Background()

		// Inject error for Get operation
		client.PrependReactor("get", "serviceaccounts", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierrors.NewInternalError(fmt.Errorf("simulated API error"))
		})

		err := ensureServiceAccount(ctx, client, "nightcrier")
		if err == nil {
			t.Error("ensureServiceAccount() should fail when Get returns non-NotFound error")
		}
	})

	t.Run("returns error when Create fails", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		ctx := context.Background()

		// Inject error for Create operation
		client.PrependReactor("create", "serviceaccounts", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierrors.NewForbidden(
				corev1.Resource("serviceaccounts"),
				"nightcrier-executor",
				fmt.Errorf("permission denied"),
			)
		})

		err := ensureServiceAccount(ctx, client, "nightcrier")
		if err == nil {
			t.Error("ensureServiceAccount() should fail when Create returns error")
		}
	})
}

func TestEnsureRole(t *testing.T) {
	t.Run("creates Role when it doesn't exist", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		ctx := context.Background()
		namespace := "nightcrier"

		err := ensureRole(ctx, client, namespace)
		if err != nil {
			t.Fatalf("ensureRole() failed: %v", err)
		}

		// Verify Role was created
		role, err := client.RbacV1().Roles(namespace).Get(ctx, "nightcrier-executor", metav1.GetOptions{})
		if err != nil {
			t.Fatalf("failed to get Role: %v", err)
		}

		if role.Name != "nightcrier-executor" {
			t.Errorf("Role name = %s, want nightcrier-executor", role.Name)
		}

		if role.Namespace != namespace {
			t.Errorf("Role namespace = %s, want %s", role.Namespace, namespace)
		}

		// Verify labels
		if role.Labels["app"] != "nightcrier" {
			t.Errorf("Role label app = %s, want nightcrier", role.Labels["app"])
		}

		if role.Labels["component"] != "executor" {
			t.Errorf("Role label component = %s, want executor", role.Labels["component"])
		}

		// Verify rules count
		if len(role.Rules) != 5 {
			t.Errorf("Role rules count = %d, want 5", len(role.Rules))
		}
	})

	t.Run("creates Role with correct ConfigMaps permissions", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		ctx := context.Background()

		err := ensureRole(ctx, client, "nightcrier")
		if err != nil {
			t.Fatalf("ensureRole() failed: %v", err)
		}

		role, _ := client.RbacV1().Roles("nightcrier").Get(ctx, "nightcrier-executor", metav1.GetOptions{})

		// Find ConfigMaps rule
		var cmRule *rbacv1.PolicyRule
		for i := range role.Rules {
			if len(role.Rules[i].Resources) > 0 && role.Rules[i].Resources[0] == "configmaps" {
				cmRule = &role.Rules[i]
				break
			}
		}

		if cmRule == nil {
			t.Fatal("ConfigMaps rule not found")
		}

		// Verify APIGroups
		if len(cmRule.APIGroups) != 1 || cmRule.APIGroups[0] != "" {
			t.Errorf("ConfigMaps APIGroups = %v, want [\"\"]", cmRule.APIGroups)
		}

		// Verify verbs
		expectedVerbs := []string{"create", "get", "list", "delete"}
		if len(cmRule.Verbs) != len(expectedVerbs) {
			t.Errorf("ConfigMaps verbs count = %d, want %d", len(cmRule.Verbs), len(expectedVerbs))
		}
		for _, verb := range expectedVerbs {
			found := false
			for _, v := range cmRule.Verbs {
				if v == verb {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("ConfigMaps missing verb: %s", verb)
			}
		}
	})

	t.Run("creates Role with correct Jobs permissions", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		ctx := context.Background()

		err := ensureRole(ctx, client, "nightcrier")
		if err != nil {
			t.Fatalf("ensureRole() failed: %v", err)
		}

		role, _ := client.RbacV1().Roles("nightcrier").Get(ctx, "nightcrier-executor", metav1.GetOptions{})

		// Find Jobs rule
		var jobRule *rbacv1.PolicyRule
		for i := range role.Rules {
			if len(role.Rules[i].Resources) > 0 && role.Rules[i].Resources[0] == "jobs" {
				jobRule = &role.Rules[i]
				break
			}
		}

		if jobRule == nil {
			t.Fatal("Jobs rule not found")
		}

		// Verify APIGroups
		if len(jobRule.APIGroups) != 1 || jobRule.APIGroups[0] != "batch" {
			t.Errorf("Jobs APIGroups = %v, want [\"batch\"]", jobRule.APIGroups)
		}

		// Verify verbs
		expectedVerbs := []string{"create", "get", "list", "watch", "delete"}
		if len(jobRule.Verbs) != len(expectedVerbs) {
			t.Errorf("Jobs verbs count = %d, want %d", len(jobRule.Verbs), len(expectedVerbs))
		}
	})

	t.Run("creates Role with correct Pods permissions", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		ctx := context.Background()

		err := ensureRole(ctx, client, "nightcrier")
		if err != nil {
			t.Fatalf("ensureRole() failed: %v", err)
		}

		role, _ := client.RbacV1().Roles("nightcrier").Get(ctx, "nightcrier-executor", metav1.GetOptions{})

		// Find Pods rule
		var podRule *rbacv1.PolicyRule
		for i := range role.Rules {
			if len(role.Rules[i].Resources) > 0 && role.Rules[i].Resources[0] == "pods" {
				podRule = &role.Rules[i]
				break
			}
		}

		if podRule == nil {
			t.Fatal("Pods rule not found")
		}

		// Verify verbs
		expectedVerbs := []string{"get", "list", "watch"}
		if len(podRule.Verbs) != len(expectedVerbs) {
			t.Errorf("Pods verbs count = %d, want %d", len(podRule.Verbs), len(expectedVerbs))
		}
	})

	t.Run("creates Role with correct Pods/log permissions", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		ctx := context.Background()

		err := ensureRole(ctx, client, "nightcrier")
		if err != nil {
			t.Fatalf("ensureRole() failed: %v", err)
		}

		role, _ := client.RbacV1().Roles("nightcrier").Get(ctx, "nightcrier-executor", metav1.GetOptions{})

		// Find Pods/log rule
		var logRule *rbacv1.PolicyRule
		for i := range role.Rules {
			if len(role.Rules[i].Resources) > 0 && role.Rules[i].Resources[0] == "pods/log" {
				logRule = &role.Rules[i]
				break
			}
		}

		if logRule == nil {
			t.Fatal("Pods/log rule not found")
		}

		// Verify verbs
		if len(logRule.Verbs) != 1 || logRule.Verbs[0] != "get" {
			t.Errorf("Pods/log verbs = %v, want [\"get\"]", logRule.Verbs)
		}
	})

	t.Run("creates Role with correct Secrets permissions", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		ctx := context.Background()

		err := ensureRole(ctx, client, "nightcrier")
		if err != nil {
			t.Fatalf("ensureRole() failed: %v", err)
		}

		role, _ := client.RbacV1().Roles("nightcrier").Get(ctx, "nightcrier-executor", metav1.GetOptions{})

		// Find Secrets rule
		var secretRule *rbacv1.PolicyRule
		for i := range role.Rules {
			if len(role.Rules[i].Resources) > 0 && role.Rules[i].Resources[0] == "secrets" {
				secretRule = &role.Rules[i]
				break
			}
		}

		if secretRule == nil {
			t.Fatal("Secrets rule not found")
		}

		// Verify verbs (read-only)
		expectedVerbs := []string{"get", "list"}
		if len(secretRule.Verbs) != len(expectedVerbs) {
			t.Errorf("Secrets verbs count = %d, want %d", len(secretRule.Verbs), len(expectedVerbs))
		}
	})

	t.Run("skips creation when Role already exists", func(t *testing.T) {
		// Pre-create Role
		existingRole := &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nightcrier-executor",
				Namespace: "nightcrier",
				Labels: map[string]string{
					"app":    "nightcrier",
					"custom": "label",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"configmaps"},
					Verbs:     []string{"get"},
				},
			},
		}
		client := fake.NewSimpleClientset(existingRole)
		ctx := context.Background()

		err := ensureRole(ctx, client, "nightcrier")
		if err != nil {
			t.Fatalf("ensureRole() failed: %v", err)
		}

		// Verify Role still exists with original labels
		role, err := client.RbacV1().Roles("nightcrier").Get(ctx, "nightcrier-executor", metav1.GetOptions{})
		if err != nil {
			t.Fatalf("failed to get Role: %v", err)
		}

		// Check that custom label is preserved (proving we didn't recreate)
		if role.Labels["custom"] != "label" {
			t.Error("Role was recreated instead of skipped")
		}

		// Check that rules weren't updated
		if len(role.Rules) != 1 {
			t.Error("Role was modified instead of skipped")
		}
	})

	t.Run("is idempotent when called multiple times", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		ctx := context.Background()
		namespace := "nightcrier"

		// Call multiple times
		for i := 0; i < 3; i++ {
			err := ensureRole(ctx, client, namespace)
			if err != nil {
				t.Fatalf("ensureRole() call %d failed: %v", i+1, err)
			}
		}

		// Verify only one Role exists
		roleList, err := client.RbacV1().Roles(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list Roles: %v", err)
		}

		if len(roleList.Items) != 1 {
			t.Errorf("Role count = %d, want 1", len(roleList.Items))
		}
	})

	t.Run("returns error when Get fails with non-NotFound error", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		ctx := context.Background()

		// Inject error for Get operation
		client.PrependReactor("get", "roles", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierrors.NewInternalError(fmt.Errorf("simulated API error"))
		})

		err := ensureRole(ctx, client, "nightcrier")
		if err == nil {
			t.Error("ensureRole() should fail when Get returns non-NotFound error")
		}
	})

	t.Run("returns error when Create fails", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		ctx := context.Background()

		// Inject error for Create operation
		client.PrependReactor("create", "roles", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierrors.NewForbidden(
				rbacv1.Resource("roles"),
				"nightcrier-executor",
				fmt.Errorf("permission denied"),
			)
		})

		err := ensureRole(ctx, client, "nightcrier")
		if err == nil {
			t.Error("ensureRole() should fail when Create returns error")
		}
	})
}

func TestEnsureRoleBinding(t *testing.T) {
	t.Run("creates RoleBinding when it doesn't exist", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		ctx := context.Background()
		namespace := "nightcrier"

		err := ensureRoleBinding(ctx, client, namespace)
		if err != nil {
			t.Fatalf("ensureRoleBinding() failed: %v", err)
		}

		// Verify RoleBinding was created
		rb, err := client.RbacV1().RoleBindings(namespace).Get(ctx, "nightcrier-executor", metav1.GetOptions{})
		if err != nil {
			t.Fatalf("failed to get RoleBinding: %v", err)
		}

		if rb.Name != "nightcrier-executor" {
			t.Errorf("RoleBinding name = %s, want nightcrier-executor", rb.Name)
		}

		if rb.Namespace != namespace {
			t.Errorf("RoleBinding namespace = %s, want %s", rb.Namespace, namespace)
		}

		// Verify labels
		if rb.Labels["app"] != "nightcrier" {
			t.Errorf("RoleBinding label app = %s, want nightcrier", rb.Labels["app"])
		}

		if rb.Labels["component"] != "executor" {
			t.Errorf("RoleBinding label component = %s, want executor", rb.Labels["component"])
		}
	})

	t.Run("creates RoleBinding with correct RoleRef", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		ctx := context.Background()

		err := ensureRoleBinding(ctx, client, "nightcrier")
		if err != nil {
			t.Fatalf("ensureRoleBinding() failed: %v", err)
		}

		rb, _ := client.RbacV1().RoleBindings("nightcrier").Get(ctx, "nightcrier-executor", metav1.GetOptions{})

		// Verify RoleRef
		if rb.RoleRef.APIGroup != "rbac.authorization.k8s.io" {
			t.Errorf("RoleRef.APIGroup = %s, want rbac.authorization.k8s.io", rb.RoleRef.APIGroup)
		}

		if rb.RoleRef.Kind != "Role" {
			t.Errorf("RoleRef.Kind = %s, want Role", rb.RoleRef.Kind)
		}

		if rb.RoleRef.Name != "nightcrier-executor" {
			t.Errorf("RoleRef.Name = %s, want nightcrier-executor", rb.RoleRef.Name)
		}
	})

	t.Run("creates RoleBinding with correct Subject", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		ctx := context.Background()

		err := ensureRoleBinding(ctx, client, "nightcrier")
		if err != nil {
			t.Fatalf("ensureRoleBinding() failed: %v", err)
		}

		rb, _ := client.RbacV1().RoleBindings("nightcrier").Get(ctx, "nightcrier-executor", metav1.GetOptions{})

		// Verify Subjects
		if len(rb.Subjects) != 1 {
			t.Fatalf("Subjects count = %d, want 1", len(rb.Subjects))
		}

		subject := rb.Subjects[0]
		if subject.Kind != "ServiceAccount" {
			t.Errorf("Subject.Kind = %s, want ServiceAccount", subject.Kind)
		}

		if subject.Name != "nightcrier-executor" {
			t.Errorf("Subject.Name = %s, want nightcrier-executor", subject.Name)
		}

		if subject.Namespace != "nightcrier" {
			t.Errorf("Subject.Namespace = %s, want nightcrier", subject.Namespace)
		}
	})

	t.Run("skips creation when RoleBinding already exists", func(t *testing.T) {
		// Pre-create RoleBinding
		existingRB := &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nightcrier-executor",
				Namespace: "nightcrier",
				Labels: map[string]string{
					"app":    "nightcrier",
					"custom": "label",
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     "nightcrier-executor",
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      "nightcrier-executor",
					Namespace: "nightcrier",
				},
			},
		}
		client := fake.NewSimpleClientset(existingRB)
		ctx := context.Background()

		err := ensureRoleBinding(ctx, client, "nightcrier")
		if err != nil {
			t.Fatalf("ensureRoleBinding() failed: %v", err)
		}

		// Verify RoleBinding still exists with original labels
		rb, err := client.RbacV1().RoleBindings("nightcrier").Get(ctx, "nightcrier-executor", metav1.GetOptions{})
		if err != nil {
			t.Fatalf("failed to get RoleBinding: %v", err)
		}

		// Check that custom label is preserved (proving we didn't recreate)
		if rb.Labels["custom"] != "label" {
			t.Error("RoleBinding was recreated instead of skipped")
		}
	})

	t.Run("is idempotent when called multiple times", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		ctx := context.Background()
		namespace := "nightcrier"

		// Call multiple times
		for i := 0; i < 3; i++ {
			err := ensureRoleBinding(ctx, client, namespace)
			if err != nil {
				t.Fatalf("ensureRoleBinding() call %d failed: %v", i+1, err)
			}
		}

		// Verify only one RoleBinding exists
		rbList, err := client.RbacV1().RoleBindings(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list RoleBindings: %v", err)
		}

		if len(rbList.Items) != 1 {
			t.Errorf("RoleBinding count = %d, want 1", len(rbList.Items))
		}
	})

	t.Run("returns error when Get fails with non-NotFound error", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		ctx := context.Background()

		// Inject error for Get operation
		client.PrependReactor("get", "rolebindings", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierrors.NewInternalError(fmt.Errorf("simulated API error"))
		})

		err := ensureRoleBinding(ctx, client, "nightcrier")
		if err == nil {
			t.Error("ensureRoleBinding() should fail when Get returns non-NotFound error")
		}
	})

	t.Run("returns error when Create fails", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		ctx := context.Background()

		// Inject error for Create operation
		client.PrependReactor("create", "rolebindings", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierrors.NewForbidden(
				rbacv1.Resource("rolebindings"),
				"nightcrier-executor",
				fmt.Errorf("permission denied"),
			)
		})

		err := ensureRoleBinding(ctx, client, "nightcrier")
		if err == nil {
			t.Error("ensureRoleBinding() should fail when Create returns error")
		}
	})
}
