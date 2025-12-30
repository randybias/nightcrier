package bootstrap

import (
	"context"
	"fmt"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestEnsureNamespace_Creates(t *testing.T) {
	// Setup: empty cluster (no namespace exists)
	client := fake.NewSimpleClientset()
	ctx := context.Background()
	namespaceName := "nightcrier"

	// Execute: ensure namespace
	created, err := ensureNamespace(ctx, client, namespaceName)

	// Assert: should succeed and report created
	if err != nil {
		t.Fatalf("ensureNamespace failed: %v", err)
	}
	if !created {
		t.Error("ensureNamespace should report created=true when namespace is created")
	}

	// Verify: namespace exists in cluster with correct labels
	ns, err := client.CoreV1().Namespaces().Get(ctx, namespaceName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get namespace after creation: %v", err)
	}
	if ns.Name != namespaceName {
		t.Errorf("Namespace name = %q, want %q", ns.Name, namespaceName)
	}

	// Verify labels
	expectedLabels := map[string]string{
		"app":        "nightcrier",
		"managed-by": "nightcrier-bootstrap",
	}
	for key, expectedValue := range expectedLabels {
		if actualValue, ok := ns.Labels[key]; !ok {
			t.Errorf("Label %q not found on namespace", key)
		} else if actualValue != expectedValue {
			t.Errorf("Label %q = %q, want %q", key, actualValue, expectedValue)
		}
	}
}

func TestEnsureNamespace_AlreadyExists(t *testing.T) {
	// Setup: namespace already exists
	namespaceName := "nightcrier"
	existingNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
			Labels: map[string]string{
				"pre-existing": "true",
			},
		},
	}
	client := fake.NewSimpleClientset(existingNamespace)
	ctx := context.Background()

	// Execute: ensure namespace
	created, err := ensureNamespace(ctx, client, namespaceName)

	// Assert: should succeed and report not created
	if err != nil {
		t.Fatalf("ensureNamespace failed: %v", err)
	}
	if created {
		t.Error("ensureNamespace should report created=false when namespace already exists")
	}

	// Verify: namespace still exists with original labels (unchanged)
	ns, err := client.CoreV1().Namespaces().Get(ctx, namespaceName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get namespace: %v", err)
	}
	if ns.Labels["pre-existing"] != "true" {
		t.Error("Existing namespace labels were modified")
	}
	if _, hasNewLabel := ns.Labels["managed-by"]; hasNewLabel {
		t.Error("ensureNamespace should not modify existing namespace labels")
	}
}

func TestEnsureNamespace_Idempotent(t *testing.T) {
	// Setup: empty cluster
	client := fake.NewSimpleClientset()
	ctx := context.Background()
	namespaceName := "nightcrier"

	// Execute: call ensureNamespace twice
	created1, err1 := ensureNamespace(ctx, client, namespaceName)
	created2, err2 := ensureNamespace(ctx, client, namespaceName)

	// Assert: first call creates, second call skips
	if err1 != nil {
		t.Fatalf("First ensureNamespace call failed: %v", err1)
	}
	if err2 != nil {
		t.Fatalf("Second ensureNamespace call failed: %v", err2)
	}
	if !created1 {
		t.Error("First call should report created=true")
	}
	if created2 {
		t.Error("Second call should report created=false (already exists)")
	}

	// Verify: only one namespace exists
	namespaces, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatalf("Failed to list namespaces: %v", err)
	}
	if len(namespaces.Items) != 1 {
		t.Errorf("Expected 1 namespace, got %d", len(namespaces.Items))
	}
}

func TestEnsureNamespace_GetError(t *testing.T) {
	// Setup: client that returns error on Get (not NotFound)
	client := fake.NewSimpleClientset()
	client.PrependReactor("get", "namespaces", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewInternalError(
			fmt.Errorf("simulated API server error"),
		)
	})
	ctx := context.Background()

	// Execute: ensure namespace
	created, err := ensureNamespace(ctx, client, "nightcrier")

	// Assert: should fail with wrapped error
	if err == nil {
		t.Fatal("ensureNamespace should return error when Get fails")
	}
	if created {
		t.Error("Should report created=false on error")
	}
	if !strings.Contains(err.Error(), "failed to check if namespace exists") {
		t.Errorf("Error message should mention check failure: %v", err)
	}
}

func TestEnsureNamespace_CreateError(t *testing.T) {
	// Setup: client that returns error on Create
	client := fake.NewSimpleClientset()
	client.PrependReactor("create", "namespaces", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(
			corev1.Resource("namespaces"),
			"nightcrier",
			fmt.Errorf("user lacks permission to create namespaces"),
		)
	})
	ctx := context.Background()

	// Execute: ensure namespace
	created, err := ensureNamespace(ctx, client, "nightcrier")

	// Assert: should fail with wrapped error
	if err == nil {
		t.Fatal("ensureNamespace should return error when Create fails")
	}
	if created {
		t.Error("Should report created=false on error")
	}
	if !strings.Contains(err.Error(), "failed to create namespace") {
		t.Errorf("Error message should mention create failure: %v", err)
	}
}

func TestManager_EnsureNamespace(t *testing.T) {
	// Setup: Manager with fake client
	client := fake.NewSimpleClientset()
	config := Config{
		Namespace: "test-namespace",
	}
	manager := NewManager(client, config)
	ctx := context.Background()

	// Execute: call Manager's ensureNamespace method
	created, err := manager.ensureNamespace(ctx, config.Namespace)

	// Assert: should succeed
	if err != nil {
		t.Fatalf("Manager.ensureNamespace failed: %v", err)
	}
	if !created {
		t.Error("Should report created=true for new namespace")
	}

	// Verify: namespace exists
	_, err = client.CoreV1().Namespaces().Get(ctx, config.Namespace, metav1.GetOptions{})
	if err != nil {
		t.Errorf("Namespace should exist after Manager.ensureNamespace: %v", err)
	}
}
