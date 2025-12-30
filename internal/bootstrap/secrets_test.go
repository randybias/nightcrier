package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

// TestEnsureAPIKeysSecret_Success tests successful creation of API keys Secret
func TestEnsureAPIKeysSecret_Success(t *testing.T) {
	tests := []struct {
		name      string
		anthropic string
		openai    string
		gemini    string
	}{
		{
			name:      "all three keys provided",
			anthropic: "sk-ant-123",
			openai:    "sk-openai-456",
			gemini:    "gemini-789",
		},
		{
			name:      "only anthropic key",
			anthropic: "sk-ant-123",
			openai:    "",
			gemini:    "",
		},
		{
			name:      "only openai key",
			anthropic: "",
			openai:    "sk-openai-456",
			gemini:    "",
		},
		{
			name:      "only gemini key",
			anthropic: "",
			openai:    "",
			gemini:    "gemini-789",
		},
		{
			name:      "two keys provided",
			anthropic: "sk-ant-123",
			openai:    "sk-openai-456",
			gemini:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewSimpleClientset()
			ctx := context.Background()
			namespace := "nightcrier"

			err := ensureAPIKeysSecret(ctx, fakeClient, namespace, tt.anthropic, tt.openai, tt.gemini)
			if err != nil {
				t.Fatalf("ensureAPIKeysSecret() failed: %v", err)
			}

			// Verify Secret was created
			secret, err := fakeClient.CoreV1().Secrets(namespace).Get(ctx, "ai-api-keys", metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Failed to get created Secret: %v", err)
			}

			// Verify Secret name
			if secret.Name != "ai-api-keys" {
				t.Errorf("Secret name = %s, want ai-api-keys", secret.Name)
			}

			// Verify Secret type
			if secret.Type != corev1.SecretTypeOpaque {
				t.Errorf("Secret type = %s, want %s", secret.Type, corev1.SecretTypeOpaque)
			}

			// Verify labels
			expectedLabels := map[string]string{
				"app":        "nightcrier",
				"managed-by": "nightcrier-bootstrap",
			}
			for k, expectedValue := range expectedLabels {
				if actualValue, ok := secret.Labels[k]; !ok {
					t.Errorf("Secret missing label %s", k)
				} else if actualValue != expectedValue {
					t.Errorf("Secret label %s = %s, want %s", k, actualValue, expectedValue)
				}
			}

			// Verify data
			if string(secret.Data["anthropic"]) != tt.anthropic {
				t.Errorf("Secret anthropic key = %s, want %s", string(secret.Data["anthropic"]), tt.anthropic)
			}
			if string(secret.Data["openai"]) != tt.openai {
				t.Errorf("Secret openai key = %s, want %s", string(secret.Data["openai"]), tt.openai)
			}
			if string(secret.Data["gemini"]) != tt.gemini {
				t.Errorf("Secret gemini key = %s, want %s", string(secret.Data["gemini"]), tt.gemini)
			}
		})
	}
}

// TestEnsureAPIKeysSecret_EmptyKeys tests that at least one key is required
func TestEnsureAPIKeysSecret_EmptyKeys(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	ctx := context.Background()
	namespace := "nightcrier"

	err := ensureAPIKeysSecret(ctx, fakeClient, namespace, "", "", "")
	if err == nil {
		t.Fatal("ensureAPIKeysSecret() should fail when all keys are empty")
	}

	expectedErrorSubstring := "at least one API key must be non-empty"
	if !contains(err.Error(), expectedErrorSubstring) {
		t.Errorf("ensureAPIKeysSecret() error = %v, should contain %s", err, expectedErrorSubstring)
	}
}

// TestEnsureAPIKeysSecret_Idempotent tests that Secret creation is idempotent
func TestEnsureAPIKeysSecret_Idempotent(t *testing.T) {
	// Pre-create a Secret
	existingSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ai-api-keys",
			Namespace: "nightcrier",
			Labels: map[string]string{
				"app": "nightcrier",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"anthropic": []byte("existing-key"),
			"openai":    []byte(""),
			"gemini":    []byte(""),
		},
	}

	fakeClient := fake.NewSimpleClientset(existingSecret)
	ctx := context.Background()
	namespace := "nightcrier"

	// Try to create Secret again with different values
	err := ensureAPIKeysSecret(ctx, fakeClient, namespace, "new-anthropic-key", "new-openai-key", "new-gemini-key")
	if err != nil {
		t.Fatalf("ensureAPIKeysSecret() should not fail when Secret exists: %v", err)
	}

	// Verify Secret was not modified
	secret, err := fakeClient.CoreV1().Secrets(namespace).Get(ctx, "ai-api-keys", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get Secret: %v", err)
	}

	// Secret should still have the original value
	if string(secret.Data["anthropic"]) != "existing-key" {
		t.Errorf("Secret was modified, anthropic key = %s, want existing-key", string(secret.Data["anthropic"]))
	}
}

// TestEnsureAPIKeysSecret_GetError tests error handling when Get fails
func TestEnsureAPIKeysSecret_GetError(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	// Inject an error for Get operation
	fakeClient.PrependReactor("get", "secrets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, &fakeAPIError{message: "simulated get error"}
	})

	ctx := context.Background()
	namespace := "nightcrier"

	err := ensureAPIKeysSecret(ctx, fakeClient, namespace, "key1", "", "")
	if err == nil {
		t.Fatal("ensureAPIKeysSecret() should fail when Get fails")
	}

	expectedErrorSubstring := "failed to check if Secret ai-api-keys exists"
	if !contains(err.Error(), expectedErrorSubstring) {
		t.Errorf("ensureAPIKeysSecret() error = %v, should contain %s", err, expectedErrorSubstring)
	}
}

// TestEnsureAPIKeysSecret_CreateError tests error handling when Create fails
func TestEnsureAPIKeysSecret_CreateError(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	// Inject an error for Create operation
	fakeClient.PrependReactor("create", "secrets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, &fakeAPIError{message: "simulated create error"}
	})

	ctx := context.Background()
	namespace := "nightcrier"

	err := ensureAPIKeysSecret(ctx, fakeClient, namespace, "key1", "", "")
	if err == nil {
		t.Fatal("ensureAPIKeysSecret() should fail when Create fails")
	}

	expectedErrorSubstring := "failed to create Secret ai-api-keys"
	if !contains(err.Error(), expectedErrorSubstring) {
		t.Errorf("ensureAPIKeysSecret() error = %v, should contain %s", err, expectedErrorSubstring)
	}
}

// TestEnsureKubeconfigSecret_Success tests successful creation of kubeconfig Secret
func TestEnsureKubeconfigSecret_Success(t *testing.T) {
	// Create a temporary kubeconfig file
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig.yaml")
	kubeconfigContent := []byte("apiVersion: v1\nkind: Config\nclusters:\n- name: test\n")

	err := os.WriteFile(kubeconfigPath, kubeconfigContent, 0600)
	if err != nil {
		t.Fatalf("Failed to create test kubeconfig file: %v", err)
	}

	fakeClient := fake.NewSimpleClientset()
	ctx := context.Background()
	namespace := "nightcrier"
	clusterName := "test-cluster"

	err = ensureKubeconfigSecret(ctx, fakeClient, namespace, clusterName, kubeconfigPath)
	if err != nil {
		t.Fatalf("ensureKubeconfigSecret() failed: %v", err)
	}

	// Verify Secret was created
	expectedSecretName := "kubeconfig-test-cluster"
	secret, err := fakeClient.CoreV1().Secrets(namespace).Get(ctx, expectedSecretName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get created Secret: %v", err)
	}

	// Verify Secret name
	if secret.Name != expectedSecretName {
		t.Errorf("Secret name = %s, want %s", secret.Name, expectedSecretName)
	}

	// Verify Secret type
	if secret.Type != corev1.SecretTypeOpaque {
		t.Errorf("Secret type = %s, want %s", secret.Type, corev1.SecretTypeOpaque)
	}

	// Verify labels
	expectedLabels := map[string]string{
		"app":        "nightcrier",
		"cluster":    clusterName,
		"managed-by": "nightcrier-bootstrap",
	}
	for k, expectedValue := range expectedLabels {
		if actualValue, ok := secret.Labels[k]; !ok {
			t.Errorf("Secret missing label %s", k)
		} else if actualValue != expectedValue {
			t.Errorf("Secret label %s = %s, want %s", k, actualValue, expectedValue)
		}
	}

	// Verify data
	if string(secret.Data["config"]) != string(kubeconfigContent) {
		t.Errorf("Secret config data = %s, want %s", string(secret.Data["config"]), string(kubeconfigContent))
	}
}

// TestEnsureKubeconfigSecret_FileNotFound tests error when file doesn't exist
func TestEnsureKubeconfigSecret_FileNotFound(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	ctx := context.Background()
	namespace := "nightcrier"
	clusterName := "test-cluster"
	kubeconfigPath := "/nonexistent/path/kubeconfig.yaml"

	err := ensureKubeconfigSecret(ctx, fakeClient, namespace, clusterName, kubeconfigPath)
	if err == nil {
		t.Fatal("ensureKubeconfigSecret() should fail when file doesn't exist")
	}

	expectedErrorSubstring := "kubeconfig file not found at path"
	if !contains(err.Error(), expectedErrorSubstring) {
		t.Errorf("ensureKubeconfigSecret() error = %v, should contain %s", err, expectedErrorSubstring)
	}
}

// TestEnsureKubeconfigSecret_DirectoryPath tests error when path is a directory
func TestEnsureKubeconfigSecret_DirectoryPath(t *testing.T) {
	tmpDir := t.TempDir()

	fakeClient := fake.NewSimpleClientset()
	ctx := context.Background()
	namespace := "nightcrier"
	clusterName := "test-cluster"

	err := ensureKubeconfigSecret(ctx, fakeClient, namespace, clusterName, tmpDir)
	if err == nil {
		t.Fatal("ensureKubeconfigSecret() should fail when path is a directory")
	}

	expectedErrorSubstring := "kubeconfig path is a directory"
	if !contains(err.Error(), expectedErrorSubstring) {
		t.Errorf("ensureKubeconfigSecret() error = %v, should contain %s", err, expectedErrorSubstring)
	}
}

// TestEnsureKubeconfigSecret_EmptyFile tests error when file is empty
func TestEnsureKubeconfigSecret_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "empty-kubeconfig.yaml")

	// Create an empty file
	err := os.WriteFile(kubeconfigPath, []byte(""), 0600)
	if err != nil {
		t.Fatalf("Failed to create empty test file: %v", err)
	}

	fakeClient := fake.NewSimpleClientset()
	ctx := context.Background()
	namespace := "nightcrier"
	clusterName := "test-cluster"

	err = ensureKubeconfigSecret(ctx, fakeClient, namespace, clusterName, kubeconfigPath)
	if err == nil {
		t.Fatal("ensureKubeconfigSecret() should fail when file is empty")
	}

	expectedErrorSubstring := "kubeconfig file is empty"
	if !contains(err.Error(), expectedErrorSubstring) {
		t.Errorf("ensureKubeconfigSecret() error = %v, should contain %s", err, expectedErrorSubstring)
	}
}

// TestEnsureKubeconfigSecret_UnreadableFile tests error when file is not readable
func TestEnsureKubeconfigSecret_UnreadableFile(t *testing.T) {
	// Skip this test on systems where we can't control file permissions
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root (file permission tests don't work)")
	}

	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "unreadable-kubeconfig.yaml")

	// Create a file with no read permissions
	err := os.WriteFile(kubeconfigPath, []byte("test content"), 0000)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	fakeClient := fake.NewSimpleClientset()
	ctx := context.Background()
	namespace := "nightcrier"
	clusterName := "test-cluster"

	err = ensureKubeconfigSecret(ctx, fakeClient, namespace, clusterName, kubeconfigPath)
	if err == nil {
		t.Fatal("ensureKubeconfigSecret() should fail when file is not readable")
	}

	expectedErrorSubstring := "failed to read kubeconfig file"
	if !contains(err.Error(), expectedErrorSubstring) {
		t.Errorf("ensureKubeconfigSecret() error = %v, should contain %s", err, expectedErrorSubstring)
	}
}

// TestEnsureKubeconfigSecret_Idempotent tests that Secret creation is idempotent
func TestEnsureKubeconfigSecret_Idempotent(t *testing.T) {
	// Pre-create a Secret
	existingSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubeconfig-test-cluster",
			Namespace: "nightcrier",
			Labels: map[string]string{
				"app":     "nightcrier",
				"cluster": "test-cluster",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"config": []byte("existing-kubeconfig-content"),
		},
	}

	fakeClient := fake.NewSimpleClientset(existingSecret)

	// Create a temporary kubeconfig file with different content
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig.yaml")
	newContent := []byte("new-kubeconfig-content")

	err := os.WriteFile(kubeconfigPath, newContent, 0600)
	if err != nil {
		t.Fatalf("Failed to create test kubeconfig file: %v", err)
	}

	ctx := context.Background()
	namespace := "nightcrier"
	clusterName := "test-cluster"

	// Try to create Secret again with different file
	err = ensureKubeconfigSecret(ctx, fakeClient, namespace, clusterName, kubeconfigPath)
	if err != nil {
		t.Fatalf("ensureKubeconfigSecret() should not fail when Secret exists: %v", err)
	}

	// Verify Secret was not modified
	secret, err := fakeClient.CoreV1().Secrets(namespace).Get(ctx, "kubeconfig-test-cluster", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get Secret: %v", err)
	}

	// Secret should still have the original content
	if string(secret.Data["config"]) != "existing-kubeconfig-content" {
		t.Errorf("Secret was modified, config = %s, want existing-kubeconfig-content", string(secret.Data["config"]))
	}
}

// TestEnsureKubeconfigSecret_GetError tests error handling when Get fails
func TestEnsureKubeconfigSecret_GetError(t *testing.T) {
	// Create a valid kubeconfig file
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig.yaml")
	err := os.WriteFile(kubeconfigPath, []byte("test content"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test kubeconfig file: %v", err)
	}

	fakeClient := fake.NewSimpleClientset()

	// Inject an error for Get operation
	fakeClient.PrependReactor("get", "secrets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, &fakeAPIError{message: "simulated get error"}
	})

	ctx := context.Background()
	namespace := "nightcrier"
	clusterName := "test-cluster"

	err = ensureKubeconfigSecret(ctx, fakeClient, namespace, clusterName, kubeconfigPath)
	if err == nil {
		t.Fatal("ensureKubeconfigSecret() should fail when Get fails")
	}

	expectedErrorSubstring := "failed to check if Secret kubeconfig-test-cluster exists"
	if !contains(err.Error(), expectedErrorSubstring) {
		t.Errorf("ensureKubeconfigSecret() error = %v, should contain %s", err, expectedErrorSubstring)
	}
}

// TestEnsureKubeconfigSecret_CreateError tests error handling when Create fails
func TestEnsureKubeconfigSecret_CreateError(t *testing.T) {
	// Create a valid kubeconfig file
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig.yaml")
	err := os.WriteFile(kubeconfigPath, []byte("test content"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test kubeconfig file: %v", err)
	}

	fakeClient := fake.NewSimpleClientset()

	// Inject an error for Create operation
	fakeClient.PrependReactor("create", "secrets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, &fakeAPIError{message: "simulated create error"}
	})

	ctx := context.Background()
	namespace := "nightcrier"
	clusterName := "test-cluster"

	err = ensureKubeconfigSecret(ctx, fakeClient, namespace, clusterName, kubeconfigPath)
	if err == nil {
		t.Fatal("ensureKubeconfigSecret() should fail when Create fails")
	}

	expectedErrorSubstring := "failed to create Secret kubeconfig-test-cluster"
	if !contains(err.Error(), expectedErrorSubstring) {
		t.Errorf("ensureKubeconfigSecret() error = %v, should contain %s", err, expectedErrorSubstring)
	}
}

// fakeAPIError is a fake error that doesn't implement IsNotFound
type fakeAPIError struct {
	message string
}

func (e *fakeAPIError) Error() string {
	return e.message
}
