package k8s

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rbias/nightcrier/internal/incident"
	"github.com/rbias/nightcrier/internal/storage"
)

// TestPhase1Through4Integration demonstrates the complete flow from Job creation
// through artifact processing and database updates.
// This is an example test showing how Phase 4 integrates with the rest of the system.
func TestPhase1Through4Integration(t *testing.T) {
	t.Skip("Integration example - requires full K8s setup")

	ctx := context.Background()

	// Setup mock dependencies
	stateStore := newMockStateStore()
	storageBackend := newMockStorage()

	// Create test incident
	incidentID := uuid.New().String()
	executionID := uuid.New().String()

	testIncident := &incident.Incident{
		IncidentID: incidentID,
		FaultID:    "fault-123",
		Status:     incident.StatusPending,
		Cluster:    "prod-us-east-1",
		Namespace:  "default",
		FaultType:  "PodCrashLoop",
		Severity:   "critical",
		Context:    "Pod nginx-deployment-abc123 is crash looping",
		Timestamp:  time.Now().Format(time.RFC3339),
		CreatedAt:  time.Now(),
	}
	stateStore.incidents[incidentID] = testIncident

	// Phase 1: Job Creation (simulated)
	// In real implementation:
	// 1. Generate presigned PUT URLs for outputs
	// 2. Create ConfigMap with incident data
	// 3. Create Job referencing ConfigMap and Secrets
	// 4. Watch Job for completion
	fmt.Printf("Phase 1-3: Job created and completed (simulated)\n")

	// Simulate Job execution - the agent container would:
	// 1. Read incident.json from ConfigMap
	// 2. Execute investigation
	// 3. Upload report.md, agent.log, commands-executed.log to Object Store
	// 4. Upload result.json with exit code

	// Phase 3: Result Retrieval
	// After Job completes, retrieve artifacts from Object Store
	jobResults := &JobResults{
		ResultJSON: &ResultJSON{
			ExitCode: 0,
			Message:  "Investigation completed successfully",
		},
		ReportMD: []byte(`# Investigation Report

## Summary
Pod nginx-deployment-abc123 is crash looping due to missing ConfigMap.

## Root Cause
The pod references a ConfigMap 'app-config' that does not exist in the namespace.

## Resolution
Create the missing ConfigMap or update the pod spec to remove the reference.

## Commands Executed
- kubectl get pods -n default
- kubectl describe pod nginx-deployment-abc123 -n default
- kubectl get configmaps -n default
`),
		AgentLog: []byte(`[2025-12-28 10:00:00] Agent started
[2025-12-28 10:00:05] Analyzing pod nginx-deployment-abc123
[2025-12-28 10:00:10] Found missing ConfigMap reference
[2025-12-28 10:00:15] Investigation complete
`),
		CommandsExecuted: []byte(`$ kubectl get pods -n default
$ kubectl describe pod nginx-deployment-abc123 -n default
$ kubectl get configmaps -n default
`),
		Missing: []string{}, // All artifacts present
	}

	// Phase 4: Artifact Processing
	processor := NewArtifactProcessor(&storage.ObjectStore{}, stateStore, storageBackend)

	// Prepare additional artifacts
	incidentJSON, err := SerializeIncidentToJSON(testIncident)
	if err != nil {
		t.Fatalf("Failed to serialize incident: %v", err)
	}

	permissionsJSON := []byte(`{
		"permissions": ["get", "list", "describe"],
		"resources": ["pods", "configmaps", "events"]
	}`)

	promptSent := []byte("You are a Kubernetes troubleshooting expert. Investigate the pod crash loop issue.")

	// Process Job results
	cfg := ProcessJobResultsConfig{
		IncidentID:      incidentID,
		ExecutionID:     executionID,
		JobResults:      jobResults,
		StartedAt:       time.Now().Add(-5 * time.Minute),
		IncidentJSON:    incidentJSON,
		PermissionsJSON: permissionsJSON,
		PromptSent:      promptSent,
	}

	err = processor.ProcessJobResults(ctx, cfg)
	if err != nil {
		t.Fatalf("ProcessJobResults failed: %v", err)
	}

	// Verify Phase 4 outputs
	fmt.Printf("Phase 4: Artifact processing completed\n")

	// 4.1: Markdown to HTML conversion
	savedArtifacts := storageBackend.savedIncidents[incidentID]
	if savedArtifacts == nil {
		t.Fatal("Artifacts not saved")
	}
	if len(savedArtifacts.InvestigationHTML) == 0 {
		t.Error("HTML report not generated")
	}
	fmt.Printf("  ✓ Markdown converted to HTML\n")

	// 4.2: Storage integration
	if len(storageBackend.savedIncidents) != 1 {
		t.Error("Artifacts not uploaded to storage")
	}
	fmt.Printf("  ✓ Artifacts uploaded to Object Store\n")

	// 4.3: Database updates
	if len(stateStore.executions) != 1 {
		t.Error("Agent execution not recorded")
	}
	if len(stateStore.reports) != 1 {
		t.Error("Triage report not recorded")
	}

	// Verify report contains both markdown and HTML
	var savedReport *storage.TriageReport
	for _, r := range stateStore.reports {
		savedReport = r
		break
	}
	if savedReport == nil {
		t.Fatal("Report not found")
	}
	if savedReport.ReportMarkdown == "" {
		t.Error("Report markdown not saved")
	}
	if savedReport.ReportHTML == "" {
		t.Error("Report HTML not saved")
	}
	fmt.Printf("  ✓ Report saved to database (markdown + HTML)\n")

	// Verify execution has log URLs (not file paths)
	exec := stateStore.executions[executionID]
	if exec == nil {
		t.Fatal("Execution not found")
	}
	if len(exec.LogPaths) == 0 {
		t.Error("Log URLs not recorded")
	}
	// Check that URLs are URLs, not file paths
	for name, path := range exec.LogPaths {
		if path != "" && path[0] == '/' {
			t.Errorf("Log path %s appears to be a file path, not a URL: %s", name, path)
		}
	}
	fmt.Printf("  ✓ Agent execution recorded with log URLs\n")

	// 4.4: Incident completion
	if len(stateStore.completionCalls) != 1 {
		t.Error("Incident not completed")
	}
	completionCall := stateStore.completionCalls[0]
	if completionCall.exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", completionCall.exitCode)
	}
	if completionCall.failureReason != "" {
		t.Errorf("Expected no failure reason, got: %s", completionCall.failureReason)
	}

	// Verify incident status transition
	if testIncident.Status != incident.StatusResolved {
		t.Errorf("Expected incident status %s, got %s", incident.StatusResolved, testIncident.Status)
	}
	fmt.Printf("  ✓ Incident completed with status: %s\n", testIncident.Status)

	fmt.Printf("\nIntegration test successful - all phases completed\n")
}

// TestPhase4WithFailures demonstrates error handling scenarios
func TestPhase4WithFailures(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		jobResults     *JobResults
		expectedStatus string
		expectedError  bool
	}{
		{
			name: "Job timeout - missing result.json",
			jobResults: &JobResults{
				ResultJSON: nil,
				Missing:    []string{"result.json", "report.md"},
			},
			expectedStatus: incident.StatusFailed,
			expectedError:  true,
		},
		{
			name: "Agent crashed - missing report",
			jobResults: &JobResults{
				ResultJSON: &ResultJSON{
					ExitCode: 137, // SIGKILL
					Message:  "Container killed",
				},
				ReportMD: nil,
				Missing:  []string{"report.md"},
			},
			expectedStatus: incident.StatusFailed,
			expectedError:  true,
		},
		{
			name: "Agent failed with report",
			jobResults: &JobResults{
				ResultJSON: &ResultJSON{
					ExitCode: 1,
					Message:  "Unable to access cluster",
				},
				ReportMD: []byte("# Partial Investigation\n\nCluster access denied."),
			},
			expectedStatus: incident.StatusFailed,
			expectedError:  false, // Not an error - agent ran but failed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stateStore := newMockStateStore()
			storageBackend := newMockStorage()
			processor := NewArtifactProcessor(&storage.ObjectStore{}, stateStore, storageBackend)

			incidentID := uuid.New().String()
			executionID := uuid.New().String()

			testIncident := &incident.Incident{
				IncidentID: incidentID,
				Status:     incident.StatusInvestigating,
			}
			stateStore.incidents[incidentID] = testIncident

			cfg := ProcessJobResultsConfig{
				IncidentID:  incidentID,
				ExecutionID: executionID,
				JobResults:  tt.jobResults,
				StartedAt:   time.Now().Add(-5 * time.Minute),
			}

			err := processor.ProcessJobResults(ctx, cfg)

			if tt.expectedError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Verify incident was completed even on failure
			if len(stateStore.completionCalls) != 1 {
				t.Error("Incident should be completed even on failure")
			}

			// Verify final status
			if testIncident.Status != tt.expectedStatus {
				t.Errorf("Expected status %s, got %s", tt.expectedStatus, testIncident.Status)
			}
		})
	}
}
