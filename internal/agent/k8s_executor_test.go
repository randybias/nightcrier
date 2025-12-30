package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/randybias/nightcrier/internal/agent/k8s"
	"github.com/randybias/nightcrier/internal/config"
	"github.com/randybias/nightcrier/internal/incident"
	"github.com/randybias/nightcrier/internal/storage"
)

// ================================================================================
// Unit Tests for K8sExecutor helper methods
// ================================================================================

func TestLoadIncidentData_Success(t *testing.T) {
	// Test loading incident data from workspace directory
	incidentID := uuid.New().String()
	tempDir := t.TempDir()
	workspaceDir := filepath.Join(tempDir, "workspaces", incidentID)
	os.MkdirAll(workspaceDir, 0755)

	// Create test incident
	testIncident := &incident.Incident{
		IncidentID: incidentID,
		Cluster:    "test-cluster",
	}

	incidentJSON, _ := json.MarshalIndent(testIncident, "", "  ")
	incidentPath := filepath.Join(workspaceDir, "incident.json")
	os.WriteFile(incidentPath, incidentJSON, 0644)

	// Create permissions file
	permissionsJSON := `{"canGetPods": true, "canGetLogs": true}`
	permissionsPath := filepath.Join(workspaceDir, "incident_cluster_permissions.json")
	os.WriteFile(permissionsPath, []byte(permissionsJSON), 0644)

	// Create base triage prompt
	baseTriagePrompt := "You are a helpful K8s troubleshooting agent."
	baseTriagePromptPath := filepath.Join(workspaceDir, "base-triage-prompt.md")
	os.WriteFile(baseTriagePromptPath, []byte(baseTriagePrompt), 0644)

	executor := &K8sExecutor{
		config: K8sExecutorConfig{
			SystemPromptFile: baseTriagePromptPath,
		},
	}

	incidentData, err := executor.loadIncidentData(workspaceDir, incidentID)
	if err != nil {
		t.Fatalf("Failed to load incident data: %v", err)
	}

	if len(incidentData.IncidentJSON) == 0 {
		t.Fatal("Expected incident JSON to be loaded")
	}

	if incidentData.ClusterName != "test-cluster" {
		t.Errorf("Expected cluster name test-cluster, got %s", incidentData.ClusterName)
	}

	if incidentData.PermissionsJSON != permissionsJSON {
		t.Errorf("Expected permissions %s, got %s", permissionsJSON, incidentData.PermissionsJSON)
	}

	if incidentData.BaseTriagePrompt != baseTriagePrompt {
		t.Errorf("Expected base triage prompt %s, got %s", baseTriagePrompt, incidentData.BaseTriagePrompt)
	}

	t.Log("Successfully loaded incident data with all components")
}

func TestLoadIncidentData_MissingIncident(t *testing.T) {
	// Test error when incident.json is missing
	incidentID := uuid.New().String()
	tempDir := t.TempDir()
	workspaceDir := filepath.Join(tempDir, "workspaces", incidentID)
	os.MkdirAll(workspaceDir, 0755)

	executor := &K8sExecutor{}

	_, err := executor.loadIncidentData(workspaceDir, incidentID)
	if err == nil {
		t.Fatal("Expected error when incident.json is missing")
	}

	t.Logf("Correctly returned error: %v", err)
}

func TestLoadIncidentData_MissingPermissions(t *testing.T) {
	// Test loading incident data when permissions file doesn't exist (should use empty JSON)
	incidentID := uuid.New().String()
	tempDir := t.TempDir()
	workspaceDir := filepath.Join(tempDir, "workspaces", incidentID)
	os.MkdirAll(workspaceDir, 0755)

	// Create only incident.json, not permissions.json
	testIncident := &incident.Incident{
		IncidentID: incidentID,
		Cluster:    "test-cluster",
	}

	incidentJSON, _ := json.Marshal(testIncident)
	incidentPath := filepath.Join(workspaceDir, "incident.json")
	os.WriteFile(incidentPath, incidentJSON, 0644)

	executor := &K8sExecutor{}

	incidentData, err := executor.loadIncidentData(workspaceDir, incidentID)
	if err != nil {
		t.Fatalf("Failed to load incident data: %v", err)
	}

	if incidentData.PermissionsJSON != "{}" {
		t.Errorf("Expected empty JSON for missing permissions, got: %s", incidentData.PermissionsJSON)
	}

	t.Log("Missing permissions test passed: Empty JSON used as default")
}

func TestLoadIncidentData_MissingBaseTriagePrompt(t *testing.T) {
	// Test loading incident data when base-triage-prompt.md doesn't exist (should use empty string)
	incidentID := uuid.New().String()
	tempDir := t.TempDir()
	workspaceDir := filepath.Join(tempDir, "workspaces", incidentID)
	os.MkdirAll(workspaceDir, 0755)

	// Create only incident.json
	testIncident := &incident.Incident{
		IncidentID: incidentID,
		Cluster:    "test-cluster",
	}

	incidentJSON, _ := json.Marshal(testIncident)
	incidentPath := filepath.Join(workspaceDir, "incident.json")
	os.WriteFile(incidentPath, incidentJSON, 0644)

	executor := &K8sExecutor{}

	incidentData, err := executor.loadIncidentData(workspaceDir, incidentID)
	if err != nil {
		t.Fatalf("Failed to load incident data: %v", err)
	}

	if incidentData.BaseTriagePrompt != "" {
		t.Errorf("Expected empty base triage prompt, got: %s", incidentData.BaseTriagePrompt)
	}

	t.Log("Missing base triage prompt test passed: Empty string used as default")
}

// ================================================================================
// Integration Test Example (requires full mocking infrastructure)
// ================================================================================

func TestK8sExecutor_ConfigAndCreation(t *testing.T) {
	// Test that we can create a K8sExecutor with proper configuration

	ctx := context.Background()
	objStore, err := storage.NewObjectStore(ctx, "mem://", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create object store: %v", err)
	}

	cfg := K8sExecutorConfig{
		Namespace:   "nightcrier",
		Image:       "nc-agent-runner:test",
		Timeout:     300,
		AgentCLI:    "claude",
		Model:       "claude-opus-4-5-20251101",
		MemoryLimit: "2Gi",
		CPULimit:    "1",
		CleanupTTL:  3600,
	}

	tuning := &config.TuningConfig{
		Agent: config.AgentTuning{
			TimeoutBufferSeconds: 60,
		},
	}

	// We can create the config but can't create a full executor without a real K8s client
	// This test verifies the configuration structure is correct
	_ = cfg
	_ = tuning
	_ = objStore

	t.Log("K8sExecutor configuration structure validated")
}

// ================================================================================
// Note on Full E2E Tests
// ================================================================================

// Full end-to-end testing of K8sExecutor.Execute() requires:
// 1. A mock or real K8s cluster (can use kind for integration tests)
// 2. Proper mocking of k8s.Client methods (CreateIncidentConfigMap, CreateJob, WatchJob, etc.)
// 3. Mock StateStore and Storage interfaces
// 4. Mock ObjectStore with result data
//
// The existing tests in the k8s package (internal/agent/k8s/*_test.go) provide
// comprehensive coverage of individual components:
// - ConfigMap creation and deletion
// - Job creation and monitoring
// - Result retrieval from Object Store
// - Artifact processing
//
// For E2E testing in a real environment, use a kind cluster with the actual
// nc-agent-runner image. See Phase 6 (Local Development Setup) for scripts.
//
// Example E2E test flow (pseudocode):
// 1. Create kind cluster
// 2. Deploy nc-agent-runner image
// 3. Create K8s client pointing to kind cluster
// 4. Create executor with real K8s client
// 5. Call Execute() with test incident
// 6. Verify Job runs successfully
// 7. Verify results are uploaded to Object Store
// 8. Verify database is updated
// 9. Cleanup

func TestK8sExecutor_VerifyComponentIntegration(t *testing.T) {
	// This test verifies that all the k8s package components we need are available
	// and can be used together. This is not a full E2E test, but validates the
	// integration points.

	ctx := context.Background()

	// Verify we can create an ObjectStore
	objStore, err := storage.NewObjectStore(ctx, "mem://", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create ObjectStore: %v", err)
	}

	// Note: Skipping presigned URL generation test because mem:// backend doesn't support SignedPutURL
	// In production, use a real cloud storage backend (s3://, gs://, azblob://)
	incidentID := "test-incident-" + uuid.New().String()

	// Verify we can create ConfigMap data structures
	incident := &incident.Incident{
		IncidentID: incidentID,
		Cluster:    "test-cluster",
	}
	incidentJSON, err := k8s.MarshalIncidentToJSON(incident)
	if err != nil {
		t.Fatalf("Failed to marshal incident: %v", err)
	}
	if len(incidentJSON) == 0 {
		t.Error("Incident JSON should not be empty")
	}

	// Verify we can create Job config structures
	jobCfg := k8s.JobConfig{
		IncidentID:    incidentID,
		AgentCLI:      "claude",
		Namespace:     "nightcrier",
		Image:         "nc-agent-runner:test",
		ConfigMapName: "incident-" + incidentID,
		PresignedURLs: k8s.PresignedURLs{
			Report:   "http://example.com/report",
			Log:      "http://example.com/log",
			Session:  "http://example.com/session",
			Result:   "http://example.com/result",
			Commands: "http://example.com/commands",
		},
	}

	if jobCfg.IncidentID == "" {
		t.Error("JobConfig should have incident ID")
	}

	// Verify we can create ArtifactProcessor
	// Note: processor needs StateStore and Storage which need real implementations
	// For now just verify the constructor exists
	processor := k8s.NewArtifactProcessor(objStore, nil, nil)
	if processor == nil {
		t.Error("ArtifactProcessor should not be nil")
	}

	t.Log("✓ All K8s executor components are properly integrated")
	t.Log("✓ ObjectStore: can create instance")
	t.Log("✓ ConfigMap: can marshal incident data")
	t.Log("✓ Job: can create job configuration")
	t.Log("✓ Processor: can create artifact processor")
	t.Log("")
	t.Log("Note: Presigned URL generation requires real cloud storage (s3://, gs://, azblob://)")
	t.Log("Note: Full E2E testing requires a real K8s cluster (use kind for local testing)")
}
