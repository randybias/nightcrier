package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

// MockObjectStoreReader is a mock implementation of ObjectStoreReader for testing.
type MockObjectStoreReader struct {
	// Data maps keys to their content
	Data map[string][]byte
	// Errors maps keys to errors to return
	Errors map[string]error
}

// Download implements the ObjectStoreReader interface.
func (m *MockObjectStoreReader) Download(ctx context.Context, key string) ([]byte, error) {
	if err, ok := m.Errors[key]; ok {
		return nil, err
	}
	if data, ok := m.Data[key]; ok {
		return data, nil
	}
	return nil, fmt.Errorf("key not found: %s", key)
}

func TestRetrieveResults_AllArtifactsPresent(t *testing.T) {
	// Create mock data
	resultJSON := ResultJSON{
		ExitCode: 0,
		Message:  "Success",
	}
	resultData, _ := json.Marshal(resultJSON)

	reportMD := []byte("# Investigation Report\n\nAll systems operational.")
	agentLog := []byte("Starting agent...\nCompleted investigation.\n")
	commandsLog := []byte("$ kubectl get pods\n$ kubectl describe pod nginx\n")
	sessionArchive := []byte("fake tar.gz data")

	// Create mock object store
	mockStore := &MockObjectStoreReader{
		Data: map[string][]byte{
			"incidents/test-123/results/result.json":             resultData,
			"incidents/test-123/results/report.md":               reportMD,
			"incidents/test-123/results/agent.log":               agentLog,
			"incidents/test-123/results/commands-executed.log":   commandsLog,
			"incidents/test-123/results/session.tar.gz":          sessionArchive,
		},
		Errors: map[string]error{},
	}

	// Test RetrieveResults with all artifacts present
	ctx := context.Background()
	cfg := RetrieveResultsConfig{
		IncidentID:            "test-123",
		ObjectStore:           mockStore,
		IncludeSessionArchive: true,
	}

	results, err := RetrieveResults(ctx, cfg)
	if err != nil {
		t.Fatalf("RetrieveResults() failed: %v", err)
	}

	// Verify all artifacts were retrieved
	if results.ResultJSON == nil {
		t.Fatal("Expected ResultJSON to be non-nil")
	}
	if results.ResultJSON.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", results.ResultJSON.ExitCode)
	}
	if results.ResultJSON.Message != "Success" {
		t.Errorf("Expected message 'Success', got '%s'", results.ResultJSON.Message)
	}

	if string(results.ReportMD) != string(reportMD) {
		t.Error("ReportMD doesn't match expected data")
	}

	if string(results.AgentLog) != string(agentLog) {
		t.Error("AgentLog doesn't match expected data")
	}

	if string(results.CommandsExecuted) != string(commandsLog) {
		t.Error("CommandsExecuted doesn't match expected data")
	}

	if string(results.SessionArchive) != string(sessionArchive) {
		t.Error("SessionArchive doesn't match expected data")
	}

	// Verify no missing artifacts
	if len(results.Missing) != 0 {
		t.Errorf("Expected no missing artifacts, got %v", results.Missing)
	}
}

func TestRetrieveResults_MissingArtifacts(t *testing.T) {
	// Create mock data with only result.json present
	resultJSON := ResultJSON{
		ExitCode: 1,
		Message:  "Failed",
	}
	resultData, _ := json.Marshal(resultJSON)

	// Create mock object store with only result.json
	mockStore := &MockObjectStoreReader{
		Data: map[string][]byte{
			"incidents/test-456/results/result.json": resultData,
		},
		Errors: map[string]error{},
	}

	// Test RetrieveResults with missing artifacts
	ctx := context.Background()
	cfg := RetrieveResultsConfig{
		IncidentID:            "test-456",
		ObjectStore:           mockStore,
		IncludeSessionArchive: false,
	}

	results, err := RetrieveResults(ctx, cfg)
	if err != nil {
		t.Fatalf("RetrieveResults() failed: %v", err)
	}

	// Verify result.json was retrieved
	if results.ResultJSON == nil {
		t.Fatal("Expected ResultJSON to be non-nil")
	}
	if results.ResultJSON.ExitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", results.ResultJSON.ExitCode)
	}

	// Verify missing artifacts are tracked
	expectedMissing := []string{"report.md", "agent.log", "commands-executed.log"}
	if len(results.Missing) != len(expectedMissing) {
		t.Errorf("Expected %d missing artifacts, got %d", len(expectedMissing), len(results.Missing))
	}

	// Verify each expected missing artifact is in the list
	missingMap := make(map[string]bool)
	for _, missing := range results.Missing {
		missingMap[missing] = true
	}
	for _, expected := range expectedMissing {
		if !missingMap[expected] {
			t.Errorf("Expected '%s' to be in missing list", expected)
		}
	}
}

func TestRetrieveResults_AllMissing(t *testing.T) {
	// Create mock object store with no data (Job failed before upload)
	mockStore := &MockObjectStoreReader{
		Data:   map[string][]byte{},
		Errors: map[string]error{},
	}

	// Test RetrieveResults with all artifacts missing
	ctx := context.Background()
	cfg := RetrieveResultsConfig{
		IncidentID:            "test-789",
		ObjectStore:           mockStore,
		IncludeSessionArchive: false,
	}

	results, err := RetrieveResults(ctx, cfg)
	if err != nil {
		t.Fatalf("RetrieveResults() failed: %v", err)
	}

	// Verify all artifacts are marked as missing
	expectedMissing := []string{"result.json", "report.md", "agent.log", "commands-executed.log"}
	if len(results.Missing) != len(expectedMissing) {
		t.Errorf("Expected %d missing artifacts, got %d", len(expectedMissing), len(results.Missing))
	}

	// Verify ResultJSON is nil when result.json is missing
	if results.ResultJSON != nil {
		t.Error("Expected ResultJSON to be nil when result.json is missing")
	}
}

func TestRetrieveResults_InvalidResultJSON(t *testing.T) {
	// Create mock data with invalid JSON
	invalidJSON := []byte("{ invalid json }")

	// Create mock object store
	mockStore := &MockObjectStoreReader{
		Data: map[string][]byte{
			"incidents/test-invalid/results/result.json": invalidJSON,
		},
		Errors: map[string]error{},
	}

	// Test RetrieveResults with invalid result.json
	ctx := context.Background()
	cfg := RetrieveResultsConfig{
		IncidentID:  "test-invalid",
		ObjectStore: mockStore,
	}

	_, err := RetrieveResults(ctx, cfg)
	if err == nil {
		t.Fatal("Expected error when result.json is invalid JSON, got nil")
	}

	// Verify error message mentions parsing
	expectedErrMsg := "failed to parse result.json"
	if len(err.Error()) < len(expectedErrMsg) || err.Error()[:len(expectedErrMsg)] != expectedErrMsg {
		t.Errorf("Expected error to start with '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

func TestRetrieveResults_SessionArchiveOptional(t *testing.T) {
	// Create mock data without session archive
	resultJSON := ResultJSON{
		ExitCode: 0,
		Message:  "Success",
	}
	resultData, _ := json.Marshal(resultJSON)

	reportMD := []byte("# Report")
	agentLog := []byte("Logs")
	commandsLog := []byte("$ commands")

	// Create mock object store without session archive
	mockStore := &MockObjectStoreReader{
		Data: map[string][]byte{
			"incidents/test-session/results/result.json":           resultData,
			"incidents/test-session/results/report.md":             reportMD,
			"incidents/test-session/results/agent.log":             agentLog,
			"incidents/test-session/results/commands-executed.log": commandsLog,
		},
		Errors: map[string]error{},
	}

	// Test with IncludeSessionArchive=false
	ctx := context.Background()
	cfg := RetrieveResultsConfig{
		IncidentID:            "test-session",
		ObjectStore:           mockStore,
		IncludeSessionArchive: false,
	}

	results, err := RetrieveResults(ctx, cfg)
	if err != nil {
		t.Fatalf("RetrieveResults() failed: %v", err)
	}

	// Verify session archive was not retrieved
	if results.SessionArchive != nil {
		t.Error("Expected SessionArchive to be nil when not included")
	}

	// Verify session.tar.gz is not in missing list (since we didn't request it)
	for _, missing := range results.Missing {
		if missing == "session.tar.gz" {
			t.Error("session.tar.gz should not be in missing list when not requested")
		}
	}

	// Test with IncludeSessionArchive=true
	cfg.IncludeSessionArchive = true
	results, err = RetrieveResults(ctx, cfg)
	if err != nil {
		t.Fatalf("RetrieveResults() failed: %v", err)
	}

	// Verify session.tar.gz is in missing list now
	foundMissing := false
	for _, missing := range results.Missing {
		if missing == "session.tar.gz" {
			foundMissing = true
			break
		}
	}
	if !foundMissing {
		t.Error("Expected session.tar.gz to be in missing list when requested but not present")
	}
}

func TestRetrieveResults_EmptyIncidentID(t *testing.T) {
	mockStore := &MockObjectStoreReader{
		Data:   map[string][]byte{},
		Errors: map[string]error{},
	}

	ctx := context.Background()
	cfg := RetrieveResultsConfig{
		IncidentID:  "", // Empty incident ID
		ObjectStore: mockStore,
	}

	_, err := RetrieveResults(ctx, cfg)
	if err == nil {
		t.Fatal("Expected error when incident ID is empty, got nil")
	}

	expectedErrMsg := "incident ID is required"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

func TestRetrieveResults_NilObjectStore(t *testing.T) {
	ctx := context.Background()
	cfg := RetrieveResultsConfig{
		IncidentID:  "test-123",
		ObjectStore: nil, // Nil object store
	}

	_, err := RetrieveResults(ctx, cfg)
	if err == nil {
		t.Fatal("Expected error when object store is nil, got nil")
	}

	expectedErrMsg := "object store is required"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

func TestRetrieveResults_StorageError(t *testing.T) {
	// Create mock object store that returns errors
	mockStore := &MockObjectStoreReader{
		Data: map[string][]byte{},
		Errors: map[string]error{
			"incidents/test-error/results/result.json": fmt.Errorf("storage unavailable"),
		},
	}

	ctx := context.Background()
	cfg := RetrieveResultsConfig{
		IncidentID:  "test-error",
		ObjectStore: mockStore,
	}

	results, err := RetrieveResults(ctx, cfg)
	if err != nil {
		t.Fatalf("RetrieveResults() failed: %v", err)
	}

	// Storage errors should result in missing artifacts
	if len(results.Missing) == 0 {
		t.Error("Expected missing artifacts when storage returns errors")
	}

	// Verify result.json is in missing list
	foundMissing := false
	for _, missing := range results.Missing {
		if missing == "result.json" {
			foundMissing = true
			break
		}
	}
	if !foundMissing {
		t.Error("Expected result.json to be in missing list when storage error occurs")
	}
}

func TestObjectStoreAdapter(t *testing.T) {
	// This test verifies the adapter compiles and has the right interface
	// In a real scenario, you'd test with a real ObjectStore
	var _ ObjectStoreReader = (*ObjectStoreAdapter)(nil)

	// Test that nil ObjectStore doesn't panic on creation
	adapter := NewObjectStoreAdapter(nil)
	if adapter == nil {
		t.Error("NewObjectStoreAdapter should not return nil")
	}
}

func TestBlobObjectStoreAdapter(t *testing.T) {
	// This test verifies the adapter compiles and has the right interface
	var _ ObjectStoreReader = (*BlobObjectStoreAdapter)(nil)

	// Test that nil bucket doesn't panic on creation
	adapter := NewBlobObjectStoreAdapter(nil)
	if adapter == nil {
		t.Error("NewBlobObjectStoreAdapter should not return nil")
	}
}

func TestResultJSON_Marshal(t *testing.T) {
	// Test marshaling ResultJSON
	result := ResultJSON{
		ExitCode: 42,
		Message:  "Test message",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal ResultJSON: %v", err)
	}

	// Test unmarshaling
	var unmarshaled ResultJSON
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal ResultJSON: %v", err)
	}

	if unmarshaled.ExitCode != result.ExitCode {
		t.Errorf("Expected exit code %d, got %d", result.ExitCode, unmarshaled.ExitCode)
	}

	if unmarshaled.Message != result.Message {
		t.Errorf("Expected message '%s', got '%s'", result.Message, unmarshaled.Message)
	}
}

func TestJobResults_Structure(t *testing.T) {
	// Test that JobResults structure can hold all expected data
	results := &JobResults{
		ResultJSON: &ResultJSON{
			ExitCode: 0,
			Message:  "Success",
		},
		ReportMD:         []byte("# Report"),
		AgentLog:         []byte("Logs"),
		CommandsExecuted: []byte("Commands"),
		SessionArchive:   []byte("Archive"),
		Missing:          []string{},
	}

	// Verify all fields are accessible
	if results.ResultJSON == nil {
		t.Error("ResultJSON should not be nil")
	}
	if len(results.ReportMD) == 0 {
		t.Error("ReportMD should not be empty")
	}
	if len(results.AgentLog) == 0 {
		t.Error("AgentLog should not be empty")
	}
	if len(results.CommandsExecuted) == 0 {
		t.Error("CommandsExecuted should not be empty")
	}
	if len(results.SessionArchive) == 0 {
		t.Error("SessionArchive should not be empty")
	}
	if results.Missing == nil {
		t.Error("Missing slice should be initialized")
	}
}

func TestRetrieveResults_PartialSuccess(t *testing.T) {
	// Test scenario where some artifacts are present and some are missing
	resultJSON := ResultJSON{
		ExitCode: 0,
		Message:  "Partial success",
	}
	resultData, _ := json.Marshal(resultJSON)
	reportMD := []byte("# Partial Report")

	mockStore := &MockObjectStoreReader{
		Data: map[string][]byte{
			"incidents/test-partial/results/result.json": resultData,
			"incidents/test-partial/results/report.md":   reportMD,
			// Missing: agent.log, commands-executed.log
		},
		Errors: map[string]error{},
	}

	ctx := context.Background()
	cfg := RetrieveResultsConfig{
		IncidentID:            "test-partial",
		ObjectStore:           mockStore,
		IncludeSessionArchive: false,
	}

	results, err := RetrieveResults(ctx, cfg)
	if err != nil {
		t.Fatalf("RetrieveResults() failed: %v", err)
	}

	// Verify partial success
	if results.ResultJSON == nil {
		t.Error("Expected ResultJSON to be present")
	}
	if len(results.ReportMD) == 0 {
		t.Error("Expected ReportMD to be present")
	}
	if len(results.AgentLog) != 0 {
		t.Error("Expected AgentLog to be empty (missing)")
	}
	if len(results.CommandsExecuted) != 0 {
		t.Error("Expected CommandsExecuted to be empty (missing)")
	}

	// Verify missing list
	expectedMissing := 2
	if len(results.Missing) != expectedMissing {
		t.Errorf("Expected %d missing artifacts, got %d", expectedMissing, len(results.Missing))
	}
}
