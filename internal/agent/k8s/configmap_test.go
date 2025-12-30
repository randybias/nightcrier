package k8s

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/randybias/nightcrier/internal/cluster"
	"github.com/randybias/nightcrier/internal/incident"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateIncidentConfigMap(t *testing.T) {
	// Create a fake clientset
	fakeClientset := fake.NewSimpleClientset()

	// Create a client with the fake clientset
	client := &Client{
		clientset: fakeClientset,
	}

	// Prepare test data
	testIncidentID := "test-incident-123"
	testClusterName := "test-cluster"
	testNamespace := "nightcrier"

	data := ConfigMapData{
		IncidentJSON:     `{"incidentId": "test-incident-123"}`,
		PermissionsJSON:  `{"cluster_name": "test-cluster"}`,
		BaseTriagePrompt: "You are a Kubernetes troubleshooting assistant.",
	}

	cfg := ConfigMapConfig{
		Namespace:   testNamespace,
		IncidentID:  testIncidentID,
		ClusterName: testClusterName,
		Labels: map[string]string{
			"custom-label": "custom-value",
		},
	}

	// Test ConfigMap creation
	ctx := context.Background()
	configMapName, err := client.CreateIncidentConfigMap(ctx, cfg, data)
	if err != nil {
		t.Fatalf("CreateIncidentConfigMap() failed: %v", err)
	}

	expectedName := "incident-test-incident-123"
	if configMapName != expectedName {
		t.Errorf("CreateIncidentConfigMap() returned name %s, want %s", configMapName, expectedName)
	}

	// Verify ConfigMap was created
	createdCM, err := fakeClientset.CoreV1().ConfigMaps(testNamespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get created ConfigMap: %v", err)
	}

	// Verify name
	if createdCM.Name != expectedName {
		t.Errorf("ConfigMap name = %s, want %s", createdCM.Name, expectedName)
	}

	// Verify namespace
	if createdCM.Namespace != testNamespace {
		t.Errorf("ConfigMap namespace = %s, want %s", createdCM.Namespace, testNamespace)
	}

	// Verify labels
	expectedLabels := map[string]string{
		"app":          "nc-agent-runner",
		"incident-id":  testIncidentID,
		"cluster":      testClusterName,
		"custom-label": "custom-value",
	}
	for k, expectedValue := range expectedLabels {
		if actualValue, ok := createdCM.Labels[k]; !ok {
			t.Errorf("ConfigMap missing label %s", k)
		} else if actualValue != expectedValue {
			t.Errorf("ConfigMap label %s = %s, want %s", k, actualValue, expectedValue)
		}
	}

	// Verify data
	if createdCM.Data["incident.json"] != data.IncidentJSON {
		t.Errorf("ConfigMap incident.json = %s, want %s", createdCM.Data["incident.json"], data.IncidentJSON)
	}
	if createdCM.Data["permissions.json"] != data.PermissionsJSON {
		t.Errorf("ConfigMap permissions.json = %s, want %s", createdCM.Data["permissions.json"], data.PermissionsJSON)
	}
	if createdCM.Data["base-triage-prompt.md"] != data.BaseTriagePrompt {
		t.Errorf("ConfigMap base-triage-prompt.md = %s, want %s", createdCM.Data["base-triage-prompt.md"], data.BaseTriagePrompt)
	}
}

func TestCreateIncidentConfigMap_NoAdditionalLabels(t *testing.T) {
	// Create a fake clientset
	fakeClientset := fake.NewSimpleClientset()

	client := &Client{
		clientset: fakeClientset,
	}

	testIncidentID := "test-incident-456"
	testClusterName := "test-cluster-2"
	testNamespace := "nightcrier"

	data := ConfigMapData{
		IncidentJSON:    `{"incidentId": "test-incident-456"}`,
		PermissionsJSON: `{}`,
		BaseTriagePrompt:    "Test prompt",
	}

	cfg := ConfigMapConfig{
		Namespace:   testNamespace,
		IncidentID:  testIncidentID,
		ClusterName: testClusterName,
		// No additional labels
	}

	ctx := context.Background()
	configMapName, err := client.CreateIncidentConfigMap(ctx, cfg, data)
	if err != nil {
		t.Fatalf("CreateIncidentConfigMap() failed: %v", err)
	}

	// Verify ConfigMap was created
	createdCM, err := fakeClientset.CoreV1().ConfigMaps(testNamespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get created ConfigMap: %v", err)
	}

	// Verify only default labels are present
	expectedLabels := map[string]string{
		"app":         "nc-agent-runner",
		"incident-id": testIncidentID,
		"cluster":     testClusterName,
	}
	if len(createdCM.Labels) != len(expectedLabels) {
		t.Errorf("ConfigMap has %d labels, want %d", len(createdCM.Labels), len(expectedLabels))
	}
	for k, expectedValue := range expectedLabels {
		if actualValue, ok := createdCM.Labels[k]; !ok {
			t.Errorf("ConfigMap missing label %s", k)
		} else if actualValue != expectedValue {
			t.Errorf("ConfigMap label %s = %s, want %s", k, actualValue, expectedValue)
		}
	}
}

func TestDeleteConfigMap(t *testing.T) {
	// Create a fake clientset with a pre-existing ConfigMap
	testNamespace := "nightcrier"
	testConfigMapName := "incident-test-123"

	fakeClientset := fake.NewSimpleClientset()

	client := &Client{
		clientset: fakeClientset,
	}

	// Create a ConfigMap first
	ctx := context.Background()
	data := ConfigMapData{
		IncidentJSON:    `{"test": "data"}`,
		PermissionsJSON: `{"test": "permissions"}`,
		BaseTriagePrompt:    "Test prompt",
	}
	cfg := ConfigMapConfig{
		Namespace:   testNamespace,
		IncidentID:  "test-123",
		ClusterName: "test-cluster",
	}

	_, err := client.CreateIncidentConfigMap(ctx, cfg, data)
	if err != nil {
		t.Fatalf("Failed to create test ConfigMap: %v", err)
	}

	// Verify ConfigMap exists
	_, err = fakeClientset.CoreV1().ConfigMaps(testNamespace).Get(ctx, testConfigMapName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("ConfigMap should exist before deletion: %v", err)
	}

	// Test deletion
	err = client.DeleteConfigMap(ctx, testNamespace, testConfigMapName)
	if err != nil {
		t.Fatalf("DeleteConfigMap() failed: %v", err)
	}

	// Verify ConfigMap no longer exists
	_, err = fakeClientset.CoreV1().ConfigMaps(testNamespace).Get(ctx, testConfigMapName, metav1.GetOptions{})
	if err == nil {
		t.Error("ConfigMap should not exist after deletion")
	}
}

func TestDeleteConfigMap_NotFound(t *testing.T) {
	// Create a fake clientset
	fakeClientset := fake.NewSimpleClientset()

	client := &Client{
		clientset: fakeClientset,
	}

	// Try to delete a non-existent ConfigMap
	ctx := context.Background()
	err := client.DeleteConfigMap(ctx, "nightcrier", "nonexistent-configmap")
	if err == nil {
		t.Error("DeleteConfigMap() should fail for non-existent ConfigMap")
	}
}

func TestMarshalIncidentToJSON(t *testing.T) {
	// Create a test incident
	now := time.Now()
	inc := &incident.Incident{
		IncidentID: "test-123",
		FaultID:    "fault-456",
		Status:     "investigating",
		CreatedAt:  now,
		Cluster:    "test-cluster",
		Namespace:  "default",
		Resource: &incident.ResourceInfo{
			APIVersion: "v1",
			Kind:       "Pod",
			Name:       "test-pod",
			Namespace:  "default",
		},
		FaultType: "PodFailing",
		Severity:  "high",
		Context:   "Pod is crashing",
		Timestamp: now.Format(time.RFC3339),
	}

	// Marshal to JSON
	jsonStr, err := MarshalIncidentToJSON(inc)
	if err != nil {
		t.Fatalf("MarshalIncidentToJSON() failed: %v", err)
	}

	// Verify JSON is not empty
	if jsonStr == "" {
		t.Error("MarshalIncidentToJSON() returned empty string")
	}

	// Verify JSON contains expected fields
	expectedFields := []string{
		`"incidentId"`,
		`"faultId"`,
		`"status"`,
		`"cluster"`,
		`"namespace"`,
		`"resource"`,
	}
	for _, field := range expectedFields {
		if !contains(jsonStr, field) {
			t.Errorf("MarshalIncidentToJSON() missing field %s", field)
		}
	}
}

func TestMarshalPermissionsToJSON(t *testing.T) {
	// Create test permissions
	perms := &cluster.ClusterPermissions{
		ClusterName:          "test-cluster",
		ValidatedAt:          time.Now(),
		CanGetPods:           true,
		CanGetLogs:           true,
		CanGetEvents:         true,
		CanGetDeployments:    true,
		CanGetServices:       true,
		SecretsAccessAllowed: false,
		CanGetSecrets:        false,
		CanGetConfigMaps:     false,
		CanGetNodes:          true,
		Warnings:             []string{"test warning"},
	}

	// Marshal to JSON
	jsonStr, err := MarshalPermissionsToJSON(perms)
	if err != nil {
		t.Fatalf("MarshalPermissionsToJSON() failed: %v", err)
	}

	// Verify JSON is not empty
	if jsonStr == "" {
		t.Error("MarshalPermissionsToJSON() returned empty string")
	}

	// Verify JSON contains expected fields
	expectedFields := []string{
		`"cluster_name"`,
		`"validated_at"`,
		`"can_get_pods"`,
		`"can_get_logs"`,
		`"warnings"`,
	}
	for _, field := range expectedFields {
		if !contains(jsonStr, field) {
			t.Errorf("MarshalPermissionsToJSON() missing field %s", field)
		}
	}
}

func TestMarshalIncidentToJSON_NilIncident(t *testing.T) {
	// Test with nil incident - should succeed and return "null"
	jsonStr, err := MarshalIncidentToJSON(nil)
	if err != nil {
		t.Fatalf("MarshalIncidentToJSON() failed with nil incident: %v", err)
	}
	// nil marshals to "null" in JSON
	if jsonStr != "null" {
		t.Errorf("MarshalIncidentToJSON() with nil = %s, want null", jsonStr)
	}
}

func TestMarshalPermissionsToJSON_NilPermissions(t *testing.T) {
	// Test with nil permissions - should succeed and return "null"
	jsonStr, err := MarshalPermissionsToJSON(nil)
	if err != nil {
		t.Fatalf("MarshalPermissionsToJSON() failed with nil permissions: %v", err)
	}
	// nil marshals to "null" in JSON
	if jsonStr != "null" {
		t.Errorf("MarshalPermissionsToJSON() with nil = %s, want null", jsonStr)
	}
}

func TestCleanupOrphanedConfigMaps(t *testing.T) {
	// Create a fake clientset
	fakeClientset := fake.NewSimpleClientset()

	client := &Client{
		clientset: fakeClientset,
	}

	testNamespace := "nightcrier"
	ctx := context.Background()

	// Create some ConfigMaps with different ages
	// Old ConfigMap (25 hours old)
	oldTime := metav1.NewTime(time.Now().Add(-25 * time.Hour))
	oldCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "incident-old-123",
			Namespace:         testNamespace,
			CreationTimestamp: oldTime,
			Labels: map[string]string{
				"app":         "nc-agent-runner",
				"incident-id": "old-123",
				"cluster":     "test-cluster",
			},
		},
		Data: map[string]string{
			"incident.json": `{"test": "old"}`,
		},
	}

	// Recent ConfigMap (1 hour old)
	recentTime := metav1.NewTime(time.Now().Add(-1 * time.Hour))
	recentCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "incident-recent-456",
			Namespace:         testNamespace,
			CreationTimestamp: recentTime,
			Labels: map[string]string{
				"app":         "nc-agent-runner",
				"incident-id": "recent-456",
				"cluster":     "test-cluster",
			},
		},
		Data: map[string]string{
			"incident.json": `{"test": "recent"}`,
		},
	}

	// ConfigMap without nc-agent-runner label (should not be affected)
	otherCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "other-configmap",
			Namespace:         testNamespace,
			CreationTimestamp: oldTime,
			Labels: map[string]string{
				"app": "other-app",
			},
		},
		Data: map[string]string{
			"data": "other",
		},
	}

	// Create all ConfigMaps
	_, err := fakeClientset.CoreV1().ConfigMaps(testNamespace).Create(ctx, oldCM, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create old ConfigMap: %v", err)
	}
	_, err = fakeClientset.CoreV1().ConfigMaps(testNamespace).Create(ctx, recentCM, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create recent ConfigMap: %v", err)
	}
	_, err = fakeClientset.CoreV1().ConfigMaps(testNamespace).Create(ctx, otherCM, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create other ConfigMap: %v", err)
	}

	// Run cleanup with 24h max age
	deleted, err := client.CleanupOrphanedConfigMaps(ctx, testNamespace, "24h")
	if err != nil {
		t.Fatalf("CleanupOrphanedConfigMaps() failed: %v", err)
	}

	// Verify only old ConfigMap was deleted
	if len(deleted) != 1 {
		t.Errorf("Expected 1 ConfigMap deleted, got %d", len(deleted))
	}
	if len(deleted) > 0 && deleted[0] != "incident-old-123" {
		t.Errorf("Expected incident-old-123 to be deleted, got %s", deleted[0])
	}

	// Verify old ConfigMap no longer exists
	_, err = fakeClientset.CoreV1().ConfigMaps(testNamespace).Get(ctx, "incident-old-123", metav1.GetOptions{})
	if err == nil {
		t.Error("Old ConfigMap should have been deleted")
	}

	// Verify recent ConfigMap still exists
	_, err = fakeClientset.CoreV1().ConfigMaps(testNamespace).Get(ctx, "incident-recent-456", metav1.GetOptions{})
	if err != nil {
		t.Error("Recent ConfigMap should not have been deleted")
	}

	// Verify other ConfigMap still exists (no nc-agent-runner label)
	_, err = fakeClientset.CoreV1().ConfigMaps(testNamespace).Get(ctx, "other-configmap", metav1.GetOptions{})
	if err != nil {
		t.Error("Other ConfigMap should not have been deleted")
	}
}

func TestCleanupOrphanedConfigMaps_NoOrphans(t *testing.T) {
	// Create a fake clientset
	fakeClientset := fake.NewSimpleClientset()

	client := &Client{
		clientset: fakeClientset,
	}

	testNamespace := "nightcrier"
	ctx := context.Background()

	// Create a recent ConfigMap (1 hour old)
	recentTime := metav1.NewTime(time.Now().Add(-1 * time.Hour))
	recentCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "incident-recent-123",
			Namespace:         testNamespace,
			CreationTimestamp: recentTime,
			Labels: map[string]string{
				"app":         "nc-agent-runner",
				"incident-id": "recent-123",
			},
		},
		Data: map[string]string{
			"incident.json": `{"test": "data"}`,
		},
	}

	_, err := fakeClientset.CoreV1().ConfigMaps(testNamespace).Create(ctx, recentCM, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create ConfigMap: %v", err)
	}

	// Run cleanup
	deleted, err := client.CleanupOrphanedConfigMaps(ctx, testNamespace, "24h")
	if err != nil {
		t.Fatalf("CleanupOrphanedConfigMaps() failed: %v", err)
	}

	// Verify no ConfigMaps were deleted
	if len(deleted) != 0 {
		t.Errorf("Expected 0 ConfigMaps deleted, got %d", len(deleted))
	}

	// Verify ConfigMap still exists
	_, err = fakeClientset.CoreV1().ConfigMaps(testNamespace).Get(ctx, "incident-recent-123", metav1.GetOptions{})
	if err != nil {
		t.Error("ConfigMap should still exist")
	}
}

func TestCleanupOrphanedConfigMaps_InvalidDuration(t *testing.T) {
	// Create a fake clientset
	fakeClientset := fake.NewSimpleClientset()

	client := &Client{
		clientset: fakeClientset,
	}

	testNamespace := "nightcrier"
	ctx := context.Background()

	// Test with invalid duration string
	_, err := client.CleanupOrphanedConfigMaps(ctx, testNamespace, "invalid")
	if err == nil {
		t.Error("Expected error for invalid duration, got nil")
	}
}

func TestCleanupOrphanedConfigMaps_EmptyNamespace(t *testing.T) {
	// Create a fake clientset
	fakeClientset := fake.NewSimpleClientset()

	client := &Client{
		clientset: fakeClientset,
	}

	ctx := context.Background()

	// Test with empty namespace
	_, err := client.CleanupOrphanedConfigMaps(ctx, "", "24h")
	if err == nil {
		t.Error("Expected error for empty namespace, got nil")
	}

	expectedErrMsg := "namespace is required"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

func TestCleanupOrphanedConfigMaps_MultipleOldConfigMaps(t *testing.T) {
	// Create a fake clientset
	fakeClientset := fake.NewSimpleClientset()

	client := &Client{
		clientset: fakeClientset,
	}

	testNamespace := "nightcrier"
	ctx := context.Background()

	// Create multiple old ConfigMaps
	oldTime := metav1.NewTime(time.Now().Add(-30 * time.Hour))

	for i := 1; i <= 3; i++ {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:              fmt.Sprintf("incident-old-%d", i),
				Namespace:         testNamespace,
				CreationTimestamp: oldTime,
				Labels: map[string]string{
					"app":         "nc-agent-runner",
					"incident-id": fmt.Sprintf("old-%d", i),
				},
			},
			Data: map[string]string{
				"incident.json": fmt.Sprintf(`{"test": "old-%d"}`, i),
			},
		}
		_, err := fakeClientset.CoreV1().ConfigMaps(testNamespace).Create(ctx, cm, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("Failed to create ConfigMap %d: %v", i, err)
		}
	}

	// Run cleanup
	deleted, err := client.CleanupOrphanedConfigMaps(ctx, testNamespace, "24h")
	if err != nil {
		t.Fatalf("CleanupOrphanedConfigMaps() failed: %v", err)
	}

	// Verify all 3 ConfigMaps were deleted
	if len(deleted) != 3 {
		t.Errorf("Expected 3 ConfigMaps deleted, got %d", len(deleted))
	}

	// Verify all ConfigMaps are gone
	for i := 1; i <= 3; i++ {
		cmName := fmt.Sprintf("incident-old-%d", i)
		_, err := fakeClientset.CoreV1().ConfigMaps(testNamespace).Get(ctx, cmName, metav1.GetOptions{})
		if err == nil {
			t.Errorf("ConfigMap %s should have been deleted", cmName)
		}
	}
}

func TestCleanupOrphanedConfigMaps_DifferentMaxAges(t *testing.T) {
	tests := []struct {
		name          string
		maxAge        string
		configMapAge  time.Duration
		shouldDelete  bool
	}{
		{
			name:         "1 hour old, 2h max age",
			maxAge:       "2h",
			configMapAge: 1 * time.Hour,
			shouldDelete: false,
		},
		{
			name:         "3 hours old, 2h max age",
			maxAge:       "2h",
			configMapAge: 3 * time.Hour,
			shouldDelete: true,
		},
		{
			name:         "30 minutes old, 1h max age",
			maxAge:       "1h",
			configMapAge: 30 * time.Minute,
			shouldDelete: false,
		},
		{
			name:         "90 minutes old, 1h max age",
			maxAge:       "1h",
			configMapAge: 90 * time.Minute,
			shouldDelete: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fake clientset
			fakeClientset := fake.NewSimpleClientset()

			client := &Client{
				clientset: fakeClientset,
			}

			testNamespace := "nightcrier"
			ctx := context.Background()

			// Create ConfigMap with specified age
			cmTime := metav1.NewTime(time.Now().Add(-tt.configMapAge))
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "incident-test",
					Namespace:         testNamespace,
					CreationTimestamp: cmTime,
					Labels: map[string]string{
						"app":         "nc-agent-runner",
						"incident-id": "test",
					},
				},
				Data: map[string]string{
					"incident.json": `{"test": "data"}`,
				},
			}

			_, err := fakeClientset.CoreV1().ConfigMaps(testNamespace).Create(ctx, cm, metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("Failed to create ConfigMap: %v", err)
			}

			// Run cleanup
			deleted, err := client.CleanupOrphanedConfigMaps(ctx, testNamespace, tt.maxAge)
			if err != nil {
				t.Fatalf("CleanupOrphanedConfigMaps() failed: %v", err)
			}

			// Verify deletion matches expectation
			if tt.shouldDelete {
				if len(deleted) != 1 {
					t.Errorf("Expected 1 ConfigMap deleted, got %d", len(deleted))
				}
			} else {
				if len(deleted) != 0 {
					t.Errorf("Expected 0 ConfigMaps deleted, got %d", len(deleted))
				}
			}
		})
	}
}

func TestParseAge(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldError bool
	}{
		{"Valid 24 hours", "24h", false},
		{"Valid 1 hour 30 minutes", "1h30m", false},
		{"Valid 30 minutes", "30m", false},
		{"Valid 1 hour", "1h", false},
		{"Valid 2 days", "48h", false},
		{"Invalid format", "invalid", true},
		{"Empty string", "", true},
		{"Negative duration", "-1h", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseAge(tt.input)
			if tt.shouldError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
