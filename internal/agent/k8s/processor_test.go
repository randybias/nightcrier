package k8s

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/randybias/nightcrier/internal/events"
	"github.com/randybias/nightcrier/internal/incident"
	"github.com/randybias/nightcrier/internal/storage"
)

// mockObjectStore implements storage.ObjectStore for testing
type mockObjectStore struct {
	uploads      map[string][]byte
	uploadErrors map[string]error
}

func newMockObjectStore() *mockObjectStore {
	return &mockObjectStore{
		uploads:      make(map[string][]byte),
		uploadErrors: make(map[string]error),
	}
}

func (m *mockObjectStore) Upload(ctx context.Context, key string, data []byte, contentType string) error {
	if err, exists := m.uploadErrors[key]; exists {
		return err
	}
	m.uploads[key] = data
	return nil
}

func (m *mockObjectStore) SignedURL(ctx context.Context, key string) (string, time.Time, error) {
	return "https://signed.url/" + key, time.Now().Add(1 * time.Hour), nil
}

func (m *mockObjectStore) CanonicalURL(key string) string {
	return "https://canonical.url/" + key
}

// mockStateStore implements storage.StateStore for testing
type mockStateStore struct {
	incidents       map[string]*incident.Incident
	executions      map[string]*storage.AgentExecution
	reports         map[string]*storage.TriageReport
	completionCalls []completionCall
}

type completionCall struct {
	incidentID    string
	exitCode      int
	failureReason string
}

func newMockStateStore() *mockStateStore {
	return &mockStateStore{
		incidents:       make(map[string]*incident.Incident),
		executions:      make(map[string]*storage.AgentExecution),
		reports:         make(map[string]*storage.TriageReport),
		completionCalls: []completionCall{},
	}
}

func (m *mockStateStore) CreateIncident(ctx context.Context, inc *incident.Incident, event *events.FaultEvent) error {
	m.incidents[inc.IncidentID] = inc
	return nil
}

func (m *mockStateStore) UpdateIncidentStatus(ctx context.Context, incidentID string, status string, startedAt *time.Time) error {
	if inc, exists := m.incidents[incidentID]; exists {
		inc.Status = status
		inc.StartedAt = startedAt
		return nil
	}
	return errors.New("incident not found")
}

func (m *mockStateStore) CompleteIncident(ctx context.Context, incidentID string, exitCode int, failureReason string) error {
	m.completionCalls = append(m.completionCalls, completionCall{
		incidentID:    incidentID,
		exitCode:      exitCode,
		failureReason: failureReason,
	})
	if inc, exists := m.incidents[incidentID]; exists {
		if failureReason != "" || exitCode != 0 {
			inc.Status = incident.StatusFailed
		} else {
			inc.Status = incident.StatusResolved
		}
		now := time.Now()
		inc.CompletedAt = &now
		exitCodePtr := exitCode
		inc.ExitCode = &exitCodePtr
		inc.FailureReason = failureReason
		return nil
	}
	return errors.New("incident not found")
}

func (m *mockStateStore) RecordAgentExecution(ctx context.Context, exec *storage.AgentExecution) error {
	m.executions[exec.ExecutionID] = exec
	return nil
}

func (m *mockStateStore) RecordTriageReport(ctx context.Context, report *storage.TriageReport) error {
	m.reports[report.ReportID] = report
	return nil
}

func (m *mockStateStore) GetIncident(ctx context.Context, incidentID string) (*incident.Incident, error) {
	if inc, exists := m.incidents[incidentID]; exists {
		return inc, nil
	}
	return nil, errors.New("incident not found")
}

func (m *mockStateStore) ListIncidents(ctx context.Context, filters *storage.IncidentFilters) ([]*incident.Incident, error) {
	return nil, errors.New("not implemented")
}

func (m *mockStateStore) Close() error {
	return nil
}

// mockStorage implements storage.Storage for testing
type mockStorage struct {
	savedIncidents map[string]*storage.IncidentArtifacts
	saveError      error
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		savedIncidents: make(map[string]*storage.IncidentArtifacts),
	}
}

func (m *mockStorage) SaveIncident(ctx context.Context, incidentID string, artifacts *storage.IncidentArtifacts) (*storage.SaveResult, error) {
	if m.saveError != nil {
		return nil, m.saveError
	}

	m.savedIncidents[incidentID] = artifacts

	return &storage.SaveResult{
		ReportURL: "https://signed.url/investigation.html",
		ArtifactURLs: map[string]string{
			"investigation.md":   "https://signed.url/investigation.md",
			"investigation.html": "https://signed.url/investigation.html",
		},
		CanonicalURLs: map[string]string{
			"investigation.md":   "https://canonical.url/investigation.md",
			"investigation.html": "https://canonical.url/investigation.html",
		},
		LogURLs: map[string]string{
			"agent-full.log":            "https://signed.url/logs/agent-full.log",
			"agent-commands-executed.log": "https://signed.url/logs/commands.log",
		},
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}, nil
}

func TestProcessJobResults_Success(t *testing.T) {
	ctx := context.Background()
	stateStore := newMockStateStore()
	storageBackend := newMockStorage()

	processor := NewArtifactProcessor(&storage.ObjectStore{}, stateStore, storageBackend)

	incidentID := uuid.New().String()
	executionID := uuid.New().String()

	// Create a test incident in the state store
	testIncident := &incident.Incident{
		IncidentID: incidentID,
		Status:     incident.StatusInvestigating,
	}
	stateStore.incidents[incidentID] = testIncident

	// Test successful job with all artifacts
	cfg := ProcessJobResultsConfig{
		IncidentID:  incidentID,
		ExecutionID: executionID,
		StartedAt:   time.Now().Add(-10 * time.Minute),
		JobResults: &JobResults{
			ResultJSON: &ResultJSON{
				ExitCode: 0,
				Message:  "Investigation completed successfully",
			},
			ReportMD: []byte("# Investigation Report\n\nTest report content"),
			AgentLog: []byte("Agent log content"),
			CommandsExecuted: []byte("$ kubectl get pods\n$ kubectl describe pod foo"),
		},
		IncidentJSON:    []byte(`{"incident_id":"test"}`),
		PermissionsJSON: []byte(`{"permissions":["get","list"]}`),
		PromptSent:      []byte("System prompt content"),
	}

	err := processor.ProcessJobResults(ctx, cfg)
	if err != nil {
		t.Fatalf("ProcessJobResults failed: %v", err)
	}

	// Verify incident was completed with correct status
	if testIncident.Status != incident.StatusResolved {
		t.Errorf("Expected incident status %s, got %s", incident.StatusResolved, testIncident.Status)
	}

	// Verify agent execution was recorded
	if len(stateStore.executions) != 1 {
		t.Fatalf("Expected 1 agent execution, got %d", len(stateStore.executions))
	}
	exec := stateStore.executions[executionID]
	if exec == nil {
		t.Fatal("Agent execution not found")
	}
	if *exec.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", *exec.ExitCode)
	}

	// Verify triage report was recorded
	if len(stateStore.reports) != 1 {
		t.Fatalf("Expected 1 triage report, got %d", len(stateStore.reports))
	}
	for _, report := range stateStore.reports {
		if report.IncidentID != incidentID {
			t.Errorf("Expected incident ID %s, got %s", incidentID, report.IncidentID)
		}
		if report.ExecutionID != executionID {
			t.Errorf("Expected execution ID %s, got %s", executionID, report.ExecutionID)
		}
		if !strings.Contains(report.ReportMarkdown, "Test report content") {
			t.Error("Report markdown does not contain expected content")
		}
		if !strings.Contains(report.ReportHTML, "Test report content") {
			t.Error("Report HTML does not contain expected content")
		}
		if !strings.Contains(report.ReportHTML, incidentID) {
			t.Error("Report HTML does not contain incident ID")
		}
	}

	// Verify artifacts were saved
	if len(storageBackend.savedIncidents) != 1 {
		t.Fatalf("Expected 1 saved incident, got %d", len(storageBackend.savedIncidents))
	}
	artifacts := storageBackend.savedIncidents[incidentID]
	if artifacts == nil {
		t.Fatal("Saved artifacts not found")
	}
	if len(artifacts.InvestigationMD) == 0 {
		t.Error("InvestigationMD is empty")
	}
	if len(artifacts.InvestigationHTML) == 0 {
		t.Error("InvestigationHTML is empty")
	}
	if len(artifacts.AgentLogs.Combined) == 0 {
		t.Error("Agent logs are empty")
	}

	// Verify incident completion was called
	if len(stateStore.completionCalls) != 1 {
		t.Fatalf("Expected 1 completion call, got %d", len(stateStore.completionCalls))
	}
	call := stateStore.completionCalls[0]
	if call.incidentID != incidentID {
		t.Errorf("Expected incident ID %s, got %s", incidentID, call.incidentID)
	}
	if call.exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", call.exitCode)
	}
	if call.failureReason != "" {
		t.Errorf("Expected empty failure reason, got %s", call.failureReason)
	}
}

func TestProcessJobResults_NonZeroExitCode(t *testing.T) {
	ctx := context.Background()
	stateStore := newMockStateStore()
	storageBackend := newMockStorage()

	processor := NewArtifactProcessor(&storage.ObjectStore{}, stateStore, storageBackend)

	incidentID := uuid.New().String()
	executionID := uuid.New().String()

	// Create a test incident
	testIncident := &incident.Incident{
		IncidentID: incidentID,
		Status:     incident.StatusInvestigating,
	}
	stateStore.incidents[incidentID] = testIncident

	cfg := ProcessJobResultsConfig{
		IncidentID:  incidentID,
		ExecutionID: executionID,
		StartedAt:   time.Now().Add(-10 * time.Minute),
		JobResults: &JobResults{
			ResultJSON: &ResultJSON{
				ExitCode: 1,
				Message:  "Agent encountered errors",
			},
			ReportMD: []byte("# Investigation Report\n\nPartial investigation"),
		},
	}

	err := processor.ProcessJobResults(ctx, cfg)
	if err != nil {
		t.Fatalf("ProcessJobResults failed: %v", err)
	}

	// Verify incident was marked as failed
	if testIncident.Status != incident.StatusFailed {
		t.Errorf("Expected incident status %s, got %s", incident.StatusFailed, testIncident.Status)
	}

	// Verify completion call has failure reason
	if len(stateStore.completionCalls) != 1 {
		t.Fatalf("Expected 1 completion call, got %d", len(stateStore.completionCalls))
	}
	call := stateStore.completionCalls[0]
	if call.exitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", call.exitCode)
	}
	if !strings.Contains(call.failureReason, "exited with code 1") {
		t.Errorf("Expected failure reason to mention exit code, got: %s", call.failureReason)
	}
}

func TestProcessJobResults_MissingResultJSON(t *testing.T) {
	ctx := context.Background()
	stateStore := newMockStateStore()
	storageBackend := newMockStorage()

	processor := NewArtifactProcessor(&storage.ObjectStore{}, stateStore, storageBackend)

	incidentID := uuid.New().String()
	executionID := uuid.New().String()

	// Create test incident so it can be completed
	testIncident := &incident.Incident{
		IncidentID: incidentID,
		Status:     incident.StatusInvestigating,
	}
	stateStore.incidents[incidentID] = testIncident

	cfg := ProcessJobResultsConfig{
		IncidentID:  incidentID,
		ExecutionID: executionID,
		StartedAt:   time.Now().Add(-10 * time.Minute),
		JobResults: &JobResults{
			ResultJSON: nil, // Missing result.json
			Missing:    []string{"result.json"},
		},
	}

	err := processor.ProcessJobResults(ctx, cfg)
	if err == nil {
		t.Fatal("Expected error for missing result.json")
	}

	if !strings.Contains(err.Error(), "result.json not found") {
		t.Errorf("Expected error about missing result.json, got: %v", err)
	}

	// Verify agent execution was recorded with failure
	if len(stateStore.executions) != 1 {
		t.Fatalf("Expected 1 agent execution, got %d", len(stateStore.executions))
	}

	// Verify incident was completed with failure
	if len(stateStore.completionCalls) != 1 {
		t.Fatalf("Expected 1 completion call, got %d", len(stateStore.completionCalls))
	}
	call := stateStore.completionCalls[0]
	if call.exitCode != -1 {
		t.Errorf("Expected exit code -1, got %d", call.exitCode)
	}
}

func TestProcessJobResults_MissingReport(t *testing.T) {
	ctx := context.Background()
	stateStore := newMockStateStore()
	storageBackend := newMockStorage()

	processor := NewArtifactProcessor(&storage.ObjectStore{}, stateStore, storageBackend)

	incidentID := uuid.New().String()
	executionID := uuid.New().String()

	// Create test incident so it can be completed
	testIncident := &incident.Incident{
		IncidentID: incidentID,
		Status:     incident.StatusInvestigating,
	}
	stateStore.incidents[incidentID] = testIncident

	cfg := ProcessJobResultsConfig{
		IncidentID:  incidentID,
		ExecutionID: executionID,
		StartedAt:   time.Now().Add(-10 * time.Minute),
		JobResults: &JobResults{
			ResultJSON: &ResultJSON{
				ExitCode: 0,
			},
			ReportMD: nil, // Missing report.md
			Missing:  []string{"report.md"},
		},
	}

	err := processor.ProcessJobResults(ctx, cfg)
	if err == nil {
		t.Fatal("Expected error for missing report")
	}

	if !strings.Contains(err.Error(), "failed to generate investigation report") {
		t.Errorf("Expected error about missing report, got: %v", err)
	}

	// Verify incident was completed with failure
	if len(stateStore.completionCalls) != 1 {
		t.Fatalf("Expected 1 completion call, got %d", len(stateStore.completionCalls))
	}
	call := stateStore.completionCalls[0]
	if !strings.Contains(call.failureReason, "report") {
		t.Errorf("Expected failure reason to mention report, got: %s", call.failureReason)
	}
}

func TestProcessJobResults_StorageFailure(t *testing.T) {
	ctx := context.Background()
	stateStore := newMockStateStore()
	storageBackend := newMockStorage()
	storageBackend.saveError = errors.New("storage unavailable")

	processor := NewArtifactProcessor(&storage.ObjectStore{}, stateStore, storageBackend)

	incidentID := uuid.New().String()
	executionID := uuid.New().String()

	cfg := ProcessJobResultsConfig{
		IncidentID:  incidentID,
		ExecutionID: executionID,
		StartedAt:   time.Now().Add(-10 * time.Minute),
		JobResults: &JobResults{
			ResultJSON: &ResultJSON{
				ExitCode: 0,
			},
			ReportMD: []byte("# Report"),
		},
	}

	err := processor.ProcessJobResults(ctx, cfg)
	if err == nil {
		t.Fatal("Expected error for storage failure")
	}

	if !strings.Contains(err.Error(), "failed to save incident artifacts") {
		t.Errorf("Expected error about storage failure, got: %v", err)
	}
}

func TestSerializeIncidentToJSON(t *testing.T) {
	testIncident := &incident.Incident{
		IncidentID: "test-123",
		FaultID:    "fault-456",
		Status:     incident.StatusInvestigating,
		Cluster:    "prod-us-east-1",
		FaultType:  "PodCrashLoop",
		Severity:   "critical",
	}

	data, err := SerializeIncidentToJSON(testIncident)
	if err != nil {
		t.Fatalf("SerializeIncidentToJSON failed: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("Serialized data is empty")
	}

	// Verify JSON contains expected fields
	jsonStr := string(data)
	if !strings.Contains(jsonStr, "test-123") {
		t.Error("JSON does not contain incident ID")
	}
	if !strings.Contains(jsonStr, "PodCrashLoop") {
		t.Error("JSON does not contain fault type")
	}
}

func TestGetContentType(t *testing.T) {
	tests := []struct {
		filename    string
		expectedCT  string
	}{
		{"report.md", "text/markdown; charset=utf-8"},
		{"investigation.html", "text/html; charset=utf-8"},
		{"incident.json", "application/json; charset=utf-8"},
		{"agent.log", "text/plain; charset=utf-8"},
		{"session.tar.gz", "application/gzip"},
		{"archive.tgz", "application/gzip"},
		{"unknown.xyz", "application/octet-stream"},
		{"", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := getContentType(tt.filename)
			if result != tt.expectedCT {
				t.Errorf("getContentType(%q) = %q, want %q", tt.filename, result, tt.expectedCT)
			}
		})
	}
}

func TestProcessJobResults_RequiredFields(t *testing.T) {
	ctx := context.Background()
	stateStore := newMockStateStore()
	storageBackend := newMockStorage()
	processor := NewArtifactProcessor(&storage.ObjectStore{}, stateStore, storageBackend)

	tests := []struct {
		name        string
		cfg         ProcessJobResultsConfig
		expectedErr string
	}{
		{
			name: "missing incident ID",
			cfg: ProcessJobResultsConfig{
				ExecutionID: "exec-123",
				JobResults:  &JobResults{},
			},
			expectedErr: "incident ID is required",
		},
		{
			name: "missing execution ID",
			cfg: ProcessJobResultsConfig{
				IncidentID: "inc-123",
				JobResults: &JobResults{},
			},
			expectedErr: "execution ID is required",
		},
		{
			name: "missing job results",
			cfg: ProcessJobResultsConfig{
				IncidentID:  "inc-123",
				ExecutionID: "exec-123",
				JobResults:  nil,
			},
			expectedErr: "job results are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := processor.ProcessJobResults(ctx, tt.cfg)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("Expected error containing %q, got: %v", tt.expectedErr, err)
			}
		})
	}
}
