package k8s

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/randybias/nightcrier/internal/cluster"
	"github.com/randybias/nightcrier/internal/incident"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// TestFullWorkflowPhase1Through3 demonstrates the complete workflow from Phase 1
// (K8s client and presigned URLs) through Phase 3 (results retrieval and cleanup).
//
// This integration test validates:
// 1. K8s client creation with fake clientset
// 2. Presigned URL generation for outputs
// 3. ConfigMap creation with incident data (Phase 1)
// 4. Job creation that references the ConfigMap (Phase 2)
// 5. Job completion watching
// 6. Results retrieval from Object Store (Phase 3)
// 7. Cleanup of ConfigMap
func TestFullWorkflowPhase1Through3(t *testing.T) {
	// Setup
	const (
		namespace     = "nightcrier"
		incidentID    = "integration-test-123"
		clusterName   = "test-cluster"
		testImage     = "nc-agent-runner:test"
	)

	// Create a fake clientset for testing
	fakeClientset := fake.NewSimpleClientset()

	// Create the Kubernetes client wrapper
	k8sClient := &Client{
		clientset: fakeClientset,
	}

	ctx := context.Background()

	// PHASE 1: Generate presigned URLs for outputs
	// ============================================
	t.Logf("Phase 1: Generating presigned URLs for outputs")

	outputURLs, err := generateMockPresignedURLs(incidentID)
	if err != nil {
		t.Fatalf("Failed to generate presigned URLs: %v", err)
	}
	t.Logf("Generated presigned URLs for incident %s", incidentID)

	// Verify all URL types are present
	if outputURLs.Report == "" || outputURLs.Log == "" || outputURLs.Session == "" ||
		outputURLs.Result == "" || outputURLs.Commands == "" {
		t.Fatal("Not all presigned URLs were generated")
	}

	// PHASE 1 (continued): Create ConfigMap with incident data
	// ========================================================
	t.Logf("Phase 1: Creating ConfigMap with incident data")

	// Create test incident data
	testIncident := &incident.Incident{
		IncidentID: incidentID,
		FaultID:    "fault-456",
		Status:     "investigating",
		CreatedAt:  time.Now(),
		Cluster:    clusterName,
		Namespace:  "default",
		Resource: &incident.ResourceInfo{
			APIVersion: "v1",
			Kind:       "Pod",
			Name:       "test-pod",
			Namespace:  "default",
		},
		FaultType: "PodFailing",
		Severity:  "high",
		Context:   "Pod is crashing due to image pull failure",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Create test permissions
	testPermissions := &cluster.ClusterPermissions{
		ClusterName:          clusterName,
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
		Warnings:             []string{},
	}

	// Marshal incident and permissions to JSON
	incidentJSON, err := MarshalIncidentToJSON(testIncident)
	if err != nil {
		t.Fatalf("Failed to marshal incident: %v", err)
	}

	permissionsJSON, err := MarshalPermissionsToJSON(testPermissions)
	if err != nil {
		t.Fatalf("Failed to marshal permissions: %v", err)
	}

	baseTriagePrompt := "You are a Kubernetes troubleshooting assistant. Help investigate this cluster issue."

	// Create ConfigMap
	configMapData := ConfigMapData{
		IncidentJSON:     incidentJSON,
		PermissionsJSON:  permissionsJSON,
		BaseTriagePrompt: baseTriagePrompt,
	}

	configMapCfg := ConfigMapConfig{
		Namespace:   namespace,
		IncidentID:  incidentID,
		ClusterName: clusterName,
		Labels: map[string]string{
			"workflow-phase": "1",
		},
	}

	configMapName, err := k8sClient.CreateIncidentConfigMap(ctx, configMapCfg, configMapData)
	if err != nil {
		t.Fatalf("Failed to create ConfigMap: %v", err)
	}
	t.Logf("Created ConfigMap: %s", configMapName)

	// Verify ConfigMap was created
	createdCM, err := fakeClientset.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("ConfigMap should exist after creation: %v", err)
	}

	if createdCM.Data["incident.json"] != incidentJSON {
		t.Error("ConfigMap incident.json doesn't match")
	}
	if createdCM.Data["permissions.json"] != permissionsJSON {
		t.Error("ConfigMap permissions.json doesn't match")
	}
	if createdCM.Data["base-triage-prompt.md"] != baseTriagePrompt {
		t.Error("ConfigMap base-triage-prompt.md doesn't match")
	}

	// PHASE 2: Create a Job that references the ConfigMap
	// ===================================================
	t.Logf("Phase 2: Creating Job that references ConfigMap")

	jobCfg := JobConfig{
		Namespace:       namespace,
		IncidentID:      incidentID,
		ClusterName:     clusterName,
		Image:           testImage,
		AgentCLI:        "claude",
		LLMModel:        "claude-opus-4-5-20251101",
		Prompt:          "Investigate the pod failure",
		ConfigMapName:   configMapName,
		SecretName:      "triage-kubeconfig-test-cluster",
		PresignedURLs:   outputURLs.ToPresignedURLs(),
		Resources: ResourceConfig{
			MemoryLimit:   "2Gi",
			CPULimit:      "1",
			MemoryRequest: "512Mi",
			CPURequest:    "250m",
		},
		TTLSecondsAfterFinished: 3600,
		ActiveDeadlineSeconds:   600,
		BackoffLimit:            0,
		Labels: map[string]string{
			"workflow-phase": "2",
		},
	}

	jobName, err := k8sClient.CreateJob(ctx, jobCfg)
	if err != nil {
		t.Fatalf("Failed to create Job: %v", err)
	}
	t.Logf("Created Job: %s", jobName)

	// Verify Job was created with correct configuration
	createdJob, err := fakeClientset.BatchV1().Jobs(namespace).Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Job should exist after creation: %v", err)
	}

	// Verify Job references the ConfigMap
	foundConfigMapVolume := false
	for _, volume := range createdJob.Spec.Template.Spec.Volumes {
		if volume.ConfigMap != nil && volume.ConfigMap.Name == configMapName {
			foundConfigMapVolume = true
			break
		}
	}
	if !foundConfigMapVolume {
		t.Errorf("Job should reference ConfigMap %s", configMapName)
	}

	// Verify presigned URLs are in environment variables
	container := createdJob.Spec.Template.Spec.Containers[0]
	envMap := make(map[string]string)
	for _, env := range container.Env {
		if env.Value != "" {
			envMap[env.Name] = env.Value
		}
	}

	if envMap["OUTPUT_URL_REPORT"] != outputURLs.Report {
		t.Error("Job missing presigned URL for report")
	}
	if envMap["OUTPUT_URL_LOG"] != outputURLs.Log {
		t.Error("Job missing presigned URL for log")
	}
	if envMap["OUTPUT_URL_RESULT"] != outputURLs.Result {
		t.Error("Job missing presigned URL for result")
	}

	// PHASE 2 (continued): Watch Job until completion
	// ===============================================
	t.Logf("Phase 2: Watching Job for completion")

	// Simulate Job completion by updating its status
	// In a real scenario, the Job would actually run and complete
	completedJob := createdJob.DeepCopy()
	completedJob.Status = batchv1.JobStatus{
		Succeeded: 1,
		Conditions: []batchv1.JobCondition{
			{
				Type:               batchv1.JobComplete,
				Status:             corev1.ConditionTrue,
				LastTransitionTime: metav1.Now(),
				Reason:             "Completed",
				Message:            "Job completed successfully",
			},
		},
		CompletionTime: &metav1.Time{Time: time.Now()},
	}

	_, err = fakeClientset.BatchV1().Jobs(namespace).Update(ctx, completedJob, metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("Failed to update Job with completion status: %v", err)
	}

	// Watch the Job
	watchCfg := WatchJobConfig{
		Namespace: namespace,
		JobName:   jobName,
		Timeout:   5 * time.Second,
		LogFunc: func(message string) {
			t.Logf("Job watcher: %s", message)
		},
	}

	watchResult, err := k8sClient.WatchJob(ctx, watchCfg)
	if err != nil {
		t.Fatalf("Failed to watch Job: %v", err)
	}

	// Verify Job succeeded
	if watchResult.Status != JobStatusSucceeded {
		t.Errorf("Expected Job status %s, got %s", JobStatusSucceeded, watchResult.Status)
	}
	t.Logf("Job completed with status: %s", watchResult.Status)

	// PHASE 3: Retrieve results from Object Store
	// ===========================================
	t.Logf("Phase 3: Retrieving results from Object Store")

	// Create mock results that would have been uploaded by the Job
	mockResults := createMockResults(incidentID)

	// Retrieve results from mock object store
	retrieveCfg := RetrieveResultsConfig{
		IncidentID:            incidentID,
		ObjectStore:           mockResults,
		IncludeSessionArchive: true,
	}

	retrievedResults, err := RetrieveResults(ctx, retrieveCfg)
	if err != nil {
		t.Fatalf("Failed to retrieve results: %v", err)
	}
	t.Logf("Retrieved results from Object Store")

	// Verify all results were retrieved
	if retrievedResults.ResultJSON == nil {
		t.Fatal("Expected ResultJSON to be present")
	}
	if retrievedResults.ResultJSON.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", retrievedResults.ResultJSON.ExitCode)
	}

	if len(retrievedResults.ReportMD) == 0 {
		t.Error("Expected report.md to be present")
	}
	if len(retrievedResults.AgentLog) == 0 {
		t.Error("Expected agent.log to be present")
	}
	if len(retrievedResults.CommandsExecuted) == 0 {
		t.Error("Expected commands-executed.log to be present")
	}
	if len(retrievedResults.SessionArchive) == 0 {
		t.Error("Expected session.tar.gz to be present")
	}

	// Verify no artifacts are marked as missing
	if len(retrievedResults.Missing) > 0 {
		t.Errorf("Expected no missing artifacts, got: %v", retrievedResults.Missing)
	}
	t.Logf("All artifacts retrieved successfully: report, log, commands, session archive")

	// PHASE 3 (continued): Cleanup ConfigMap
	// ======================================
	t.Logf("Phase 3: Cleaning up ConfigMap")

	// Delete the ConfigMap
	err = k8sClient.DeleteConfigMap(ctx, namespace, configMapName)
	if err != nil {
		t.Fatalf("Failed to delete ConfigMap: %v", err)
	}

	// Verify ConfigMap was deleted
	_, err = fakeClientset.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err == nil {
		t.Error("ConfigMap should not exist after deletion")
	}
	t.Logf("ConfigMap deleted successfully")

	// WORKFLOW COMPLETE
	// =================
	t.Logf("Full workflow (Phase 1-3) completed successfully!")
	t.Logf("Summary:")
	t.Logf("  - Phase 1: Generated presigned URLs, created ConfigMap with incident data")
	t.Logf("  - Phase 2: Created and watched Job for completion")
	t.Logf("  - Phase 3: Retrieved all results and cleaned up ConfigMap")
}

// TestPartialWorkflowMissingArtifacts tests the workflow when some artifacts
// fail to upload (simulating a Job that partially completes).
func TestPartialWorkflowMissingArtifacts(t *testing.T) {
	const (
		namespace   = "nightcrier"
		incidentID  = "partial-workflow-456"
		clusterName = "test-cluster"
	)

	fakeClientset := fake.NewSimpleClientset()
	k8sClient := &Client{
		clientset: fakeClientset,
	}

	ctx := context.Background()

	// Create presigned URLs
	outputURLs, err := generateMockPresignedURLs(incidentID)
	if err != nil {
		t.Fatalf("Failed to generate presigned URLs: %v", err)
	}

	// Create ConfigMap
	configMapData := ConfigMapData{
		IncidentJSON:    `{"incidentId": "partial-workflow-456"}`,
		PermissionsJSON: `{"cluster_name": "test-cluster"}`,
		BaseTriagePrompt:    "Test prompt",
	}

	configMapCfg := ConfigMapConfig{
		Namespace:   namespace,
		IncidentID:  incidentID,
		ClusterName: clusterName,
	}

	configMapName, err := k8sClient.CreateIncidentConfigMap(ctx, configMapCfg, configMapData)
	if err != nil {
		t.Fatalf("Failed to create ConfigMap: %v", err)
	}

	// Create Job
	jobCfg := JobConfig{
		Namespace:       namespace,
		IncidentID:      incidentID,
		ClusterName:     clusterName,
		AgentCLI:        "claude",
		LLMModel:        "claude-opus-4-5-20251101",
		Prompt:          "Test prompt",
		ConfigMapName:   configMapName,
		SecretName:      "triage-kubeconfig-test-cluster",
		PresignedURLs:   outputURLs.ToPresignedURLs(),
	}

	jobName, err := k8sClient.CreateJob(ctx, jobCfg)
	if err != nil {
		t.Fatalf("Failed to create Job: %v", err)
	}

	// Simulate Job completion (but with missing artifacts)
	job, err := fakeClientset.BatchV1().Jobs(namespace).Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get Job: %v", err)
	}

	job.Status = batchv1.JobStatus{
		Succeeded: 1,
		Conditions: []batchv1.JobCondition{
			{
				Type:               batchv1.JobComplete,
				Status:             corev1.ConditionTrue,
				LastTransitionTime: metav1.Now(),
			},
		},
	}

	_, err = fakeClientset.BatchV1().Jobs(namespace).Update(ctx, job, metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("Failed to update Job: %v", err)
	}

	// Watch Job completion
	watchCfg := WatchJobConfig{
		Namespace: namespace,
		JobName:   jobName,
		Timeout:   5 * time.Second,
	}

	watchResult, err := k8sClient.WatchJob(ctx, watchCfg)
	if err != nil {
		t.Fatalf("Failed to watch Job: %v", err)
	}

	if watchResult.Status != JobStatusSucceeded {
		t.Errorf("Expected Job status %s, got %s", JobStatusSucceeded, watchResult.Status)
	}

	// Create mock results with only result.json present
	// (simulating incomplete upload due to Job failure/timeout)
	resultJSON := ResultJSON{
		ExitCode: 1,
		Message:  "Partial failure - some commands timed out",
	}
	resultData, _ := json.Marshal(resultJSON)

	partialMockStore := &MockObjectStoreReader{
		Data: map[string][]byte{
			"incidents/partial-workflow-456/results/result.json": resultData,
			// Missing: report.md, agent.log, commands-executed.log, session.tar.gz
		},
		Errors: map[string]error{},
	}

	// Retrieve partial results
	retrieveCfg := RetrieveResultsConfig{
		IncidentID:            incidentID,
		ObjectStore:           partialMockStore,
		IncludeSessionArchive: true,
	}

	retrievedResults, err := RetrieveResults(ctx, retrieveCfg)
	if err != nil {
		t.Fatalf("Failed to retrieve results: %v", err)
	}

	// Verify result.json is present but other artifacts are missing
	if retrievedResults.ResultJSON == nil {
		t.Fatal("Expected ResultJSON to be present")
	}

	expectedMissing := []string{"report.md", "agent.log", "commands-executed.log", "prompt-sent.md", "session.tar.gz"}
	if len(retrievedResults.Missing) != len(expectedMissing) {
		t.Errorf("Expected %d missing artifacts, got %d", len(expectedMissing), len(retrievedResults.Missing))
	}

	t.Logf("Partial workflow completed: result.json present, %d artifacts missing", len(retrievedResults.Missing))

	// Cleanup
	err = k8sClient.DeleteConfigMap(ctx, namespace, configMapName)
	if err != nil {
		t.Fatalf("Failed to delete ConfigMap: %v", err)
	}
}

// TestJobFailureScenario tests the workflow when a Job fails.
func TestJobFailureScenario(t *testing.T) {
	const (
		namespace   = "nightcrier"
		incidentID  = "failure-scenario-789"
		clusterName = "test-cluster"
	)

	fakeClientset := fake.NewSimpleClientset()
	k8sClient := &Client{
		clientset: fakeClientset,
	}

	ctx := context.Background()

	// Create presigned URLs
	outputURLs, err := generateMockPresignedURLs(incidentID)
	if err != nil {
		t.Fatalf("Failed to generate presigned URLs: %v", err)
	}

	// Create ConfigMap
	configMapData := ConfigMapData{
		IncidentJSON:    `{"incidentId": "failure-scenario-789"}`,
		PermissionsJSON: `{"cluster_name": "test-cluster"}`,
		BaseTriagePrompt:    "Test prompt",
	}

	configMapCfg := ConfigMapConfig{
		Namespace:   namespace,
		IncidentID:  incidentID,
		ClusterName: clusterName,
	}

	configMapName, err := k8sClient.CreateIncidentConfigMap(ctx, configMapCfg, configMapData)
	if err != nil {
		t.Fatalf("Failed to create ConfigMap: %v", err)
	}

	// Create Job
	jobCfg := JobConfig{
		Namespace:       namespace,
		IncidentID:      incidentID,
		ClusterName:     clusterName,
		AgentCLI:        "claude",
		LLMModel:        "claude-opus-4-5-20251101",
		Prompt:          "Test prompt",
		ConfigMapName:   configMapName,
		SecretName:      "triage-kubeconfig-test-cluster",
		PresignedURLs:   outputURLs.ToPresignedURLs(),
	}

	jobName, err := k8sClient.CreateJob(ctx, jobCfg)
	if err != nil {
		t.Fatalf("Failed to create Job: %v", err)
	}

	// Simulate Job failure
	job, err := fakeClientset.BatchV1().Jobs(namespace).Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get Job: %v", err)
	}

	job.Status = batchv1.JobStatus{
		Failed: 1,
		Conditions: []batchv1.JobCondition{
			{
				Type:               batchv1.JobFailed,
				Status:             corev1.ConditionTrue,
				LastTransitionTime: metav1.Now(),
				Reason:             "BackoffLimitExceeded",
				Message:            "Job has reached the specified backoff limit",
			},
		},
	}

	_, err = fakeClientset.BatchV1().Jobs(namespace).Update(ctx, job, metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("Failed to update Job: %v", err)
	}

	// Watch Job for failure
	watchCfg := WatchJobConfig{
		Namespace: namespace,
		JobName:   jobName,
		Timeout:   5 * time.Second,
	}

	watchResult, err := k8sClient.WatchJob(ctx, watchCfg)
	if err != nil {
		t.Fatalf("Failed to watch Job: %v", err)
	}

	// Verify Job failed
	if watchResult.Status != JobStatusFailed {
		t.Errorf("Expected Job status %s, got %s", JobStatusFailed, watchResult.Status)
	}

	t.Logf("Job failure detected: %s", watchResult.Message)

	// Attempt to retrieve results (will be all missing since Job failed before upload)
	emptyMockStore := &MockObjectStoreReader{
		Data:   map[string][]byte{},
		Errors: map[string]error{},
	}

	retrieveCfg := RetrieveResultsConfig{
		IncidentID:  incidentID,
		ObjectStore: emptyMockStore,
	}

	retrievedResults, err := RetrieveResults(ctx, retrieveCfg)
	if err != nil {
		t.Fatalf("Failed to retrieve results: %v", err)
	}

	// Verify all artifacts are missing
	if retrievedResults.ResultJSON != nil {
		t.Error("Expected ResultJSON to be nil when no artifacts available")
	}

	expectedMissing := []string{"result.json", "report.md", "agent.log", "commands-executed.log", "prompt-sent.md"}
	if len(retrievedResults.Missing) != len(expectedMissing) {
		t.Errorf("Expected %d missing artifacts, got %d", len(expectedMissing), len(retrievedResults.Missing))
	}

	t.Logf("Job failure scenario verified: all artifacts missing as expected")

	// Cleanup
	err = k8sClient.DeleteConfigMap(ctx, namespace, configMapName)
	if err != nil {
		t.Fatalf("Failed to delete ConfigMap: %v", err)
	}
}

// Mock implementations for testing

// MockObjectStore is a mock implementation of object storage for generating presigned URLs.
type MockObjectStore struct{}

// generateMockPresignedURLs creates mock presigned URLs for testing.
func generateMockPresignedURLs(incidentID string) (*OutputURLs, error) {
	baseURL := "https://mock-storage.example.com/incidents/" + incidentID + "/results"
	return &OutputURLs{
		Report:         baseURL + "/report.md?token=abc123",
		Log:            baseURL + "/agent.log?token=abc123",
		Session:        baseURL + "/session.tar.gz?token=abc123",
		Result:         baseURL + "/result.json?token=abc123",
		Commands:       baseURL + "/commands-executed.log?token=abc123",
		ReportExpiry:   time.Now().Add(30 * time.Minute),
		LogExpiry:      time.Now().Add(30 * time.Minute),
		SessionExpiry:  time.Now().Add(30 * time.Minute),
		ResultExpiry:   time.Now().Add(30 * time.Minute),
		CommandsExpiry: time.Now().Add(30 * time.Minute),
	}, nil
}

// createMockResults creates a mock object store with all expected results.
func createMockResults(incidentID string) *MockObjectStoreReader {
	resultJSON := ResultJSON{
		ExitCode: 0,
		Message:  "Investigation completed successfully",
	}
	resultData, _ := json.Marshal(resultJSON)

	reportMD := []byte(`# Investigation Report

## Summary
The incident has been investigated successfully.

## Findings
- Pod is in CrashLoopBackOff state
- Image pull failed: ImagePullBackOff
- Recommendation: Check image registry credentials

## Resolution
Update the image pull secret in the cluster.
`)

	agentLog := []byte(`Starting agent execution...
Retrieved incident configuration from ConfigMap
Initialized cluster connection
Executing investigation commands
Analyzing pod logs
Gathering resource metrics
Generating report
Investigation complete.
Exit code: 0
`)

	commandsLog := []byte(`$ kubectl get pod test-pod -n default
$ kubectl describe pod test-pod -n default
$ kubectl logs test-pod -n default --all-containers=true
$ kubectl get events -n default --sort-by='.lastTimestamp'
$ kubectl top pod test-pod -n default
`)

	sessionArchive := []byte("fake tar.gz archive containing session data and transcripts")

	promptSent := []byte(`# System Prompt

You are a Kubernetes troubleshooting expert. Investigate the incident using kubectl commands.`)

	return &MockObjectStoreReader{
		Data: map[string][]byte{
			"incidents/" + incidentID + "/results/result.json":             resultData,
			"incidents/" + incidentID + "/results/report.md":               reportMD,
			"incidents/" + incidentID + "/results/agent.log":               agentLog,
			"incidents/" + incidentID + "/results/commands-executed.log":   commandsLog,
			"incidents/" + incidentID + "/results/prompt-sent.md":          promptSent,
			"incidents/" + incidentID + "/results/session.tar.gz":          sessionArchive,
		},
		Errors: map[string]error{},
	}
}
