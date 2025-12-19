package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestFilesystemStorageNewFilesystemStorage verifies the constructor creates a valid instance.
func TestFilesystemStorageNewFilesystemStorage(t *testing.T) {
	workspaceRoot := "/tmp/test-workspace"
	fs := NewFilesystemStorage(workspaceRoot)

	if fs == nil {
		t.Fatalf("NewFilesystemStorage returned nil")
	}
	if fs.workspaceRoot != workspaceRoot {
		t.Fatalf("expected workspaceRoot %q, got %q", workspaceRoot, fs.workspaceRoot)
	}
}

// TestFilesystemStorageSaveIncidentSuccess verifies successful artifact persistence.
func TestFilesystemStorageSaveIncidentSuccess(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()
	fs := NewFilesystemStorage(tmpDir)

	incidentID := "test-incident-001"
	eventJSON := []byte(`{"event":"test"}`)
	resultJSON := []byte(`{"result":"passed"}`)
	investigationMD := []byte(`# Investigation Report\nAll systems healthy.`)

	artifacts := &IncidentArtifacts{
		EventJSON:      eventJSON,
		ResultJSON:     resultJSON,
		InvestigationMD: investigationMD,
	}

	ctx := context.Background()
	result, err := fs.SaveIncident(ctx, incidentID, artifacts)

	if err != nil {
		t.Fatalf("SaveIncident failed: %v", err)
	}

	if result == nil {
		t.Fatalf("SaveIncident returned nil SaveResult")
	}

	// Verify directory structure was created
	incidentDir := filepath.Join(tmpDir, incidentID)
	if _, err := os.Stat(incidentDir); os.IsNotExist(err) {
		t.Fatalf("incident directory not created at %s", incidentDir)
	}

	// Verify files were written correctly
	eventPath := filepath.Join(incidentDir, "event.json")
	eventData, err := os.ReadFile(eventPath)
	if err != nil {
		t.Fatalf("failed to read event.json: %v", err)
	}
	if string(eventData) != string(eventJSON) {
		t.Fatalf("event.json content mismatch: expected %q, got %q", string(eventJSON), string(eventData))
	}

	resultPath := filepath.Join(incidentDir, "result.json")
	resultData, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("failed to read result.json: %v", err)
	}
	if string(resultData) != string(resultJSON) {
		t.Fatalf("result.json content mismatch: expected %q, got %q", string(resultJSON), string(resultData))
	}

	investigationPath := filepath.Join(incidentDir, "investigation.md")
	investigationData, err := os.ReadFile(investigationPath)
	if err != nil {
		t.Fatalf("failed to read investigation.md: %v", err)
	}
	if string(investigationData) != string(investigationMD) {
		t.Fatalf("investigation.md content mismatch: expected %q, got %q", string(investigationMD), string(investigationData))
	}
}

// TestFilesystemStorageSaveResultContent verifies the SaveResult contains correct paths and URLs.
func TestFilesystemStorageSaveResultContent(t *testing.T) {
	tmpDir := t.TempDir()
	fs := NewFilesystemStorage(tmpDir)

	incidentID := "test-incident-002"
	artifacts := &IncidentArtifacts{
		EventJSON:      []byte(`{}`),
		ResultJSON:     []byte(`{}`),
		InvestigationMD: []byte(`# Report`),
	}

	ctx := context.Background()
	result, err := fs.SaveIncident(ctx, incidentID, artifacts)

	if err != nil {
		t.Fatalf("SaveIncident failed: %v", err)
	}

	expectedIncidentDir := filepath.Join(tmpDir, incidentID)
	expectedInvestigationPath := filepath.Join(expectedIncidentDir, "investigation.md")

	// Verify ReportURL points to investigation.md
	if result.ReportURL != expectedInvestigationPath {
		t.Fatalf("ReportURL mismatch: expected %q, got %q", expectedInvestigationPath, result.ReportURL)
	}

	// Verify ArtifactURLs map contains all three artifacts
	expectedArtifacts := map[string]string{
		"event.json":       filepath.Join(expectedIncidentDir, "event.json"),
		"result.json":      filepath.Join(expectedIncidentDir, "result.json"),
		"investigation.md": expectedInvestigationPath,
	}

	if len(result.ArtifactURLs) != len(expectedArtifacts) {
		t.Fatalf("ArtifactURLs count mismatch: expected %d, got %d", len(expectedArtifacts), len(result.ArtifactURLs))
	}

	for key, expectedPath := range expectedArtifacts {
		actualPath, found := result.ArtifactURLs[key]
		if !found {
			t.Fatalf("ArtifactURLs missing key %q", key)
		}
		if actualPath != expectedPath {
			t.Fatalf("ArtifactURLs[%q] mismatch: expected %q, got %q", key, expectedPath, actualPath)
		}
	}

	// Verify ExpiresAt is zero (filesystem paths don't expire)
	if !result.ExpiresAt.IsZero() {
		t.Fatalf("ExpiresAt should be zero time, got %v", result.ExpiresAt)
	}
}

// TestFilesystemStorageSaveIncidentNilArtifacts verifies error handling for nil artifacts.
func TestFilesystemStorageSaveIncidentNilArtifacts(t *testing.T) {
	tmpDir := t.TempDir()
	fs := NewFilesystemStorage(tmpDir)

	ctx := context.Background()
	_, err := fs.SaveIncident(ctx, "test-incident", nil)

	if err == nil {
		t.Fatalf("expected error for nil artifacts, got nil")
	}
}

// TestFilesystemStorageSaveIncidentMultipleIncidents verifies handling of multiple incidents.
func TestFilesystemStorageSaveIncidentMultipleIncidents(t *testing.T) {
	tmpDir := t.TempDir()
	fs := NewFilesystemStorage(tmpDir)

	incidents := []string{"incident-1", "incident-2", "incident-3"}
	ctx := context.Background()

	for _, incidentID := range incidents {
		artifacts := &IncidentArtifacts{
			EventJSON:      []byte(`{"incident":"` + incidentID + `"}`),
			ResultJSON:     []byte(`{"status":"ok"}`),
			InvestigationMD: []byte(`# Report for ` + incidentID),
		}

		_, err := fs.SaveIncident(ctx, incidentID, artifacts)
		if err != nil {
			t.Fatalf("SaveIncident failed for %s: %v", incidentID, err)
		}
	}

	// Verify all incident directories were created
	for _, incidentID := range incidents {
		incidentDir := filepath.Join(tmpDir, incidentID)
		if _, err := os.Stat(incidentDir); os.IsNotExist(err) {
			t.Fatalf("incident directory not created for %s", incidentID)
		}
	}
}

// TestFilesystemStorageSaveIncidentFilePermissions verifies files are created with correct permissions.
func TestFilesystemStorageSaveIncidentFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	fs := NewFilesystemStorage(tmpDir)

	incidentID := "test-incident-perms"
	artifacts := &IncidentArtifacts{
		EventJSON:      []byte(`{}`),
		ResultJSON:     []byte(`{}`),
		InvestigationMD: []byte(`# Report`),
	}

	ctx := context.Background()
	_, err := fs.SaveIncident(ctx, incidentID, artifacts)

	if err != nil {
		t.Fatalf("SaveIncident failed: %v", err)
	}

	incidentDir := filepath.Join(tmpDir, incidentID)
	eventPath := filepath.Join(incidentDir, "event.json")
	resultPath := filepath.Join(incidentDir, "result.json")
	investigationPath := filepath.Join(incidentDir, "investigation.md")

	// Check file permissions are 0600 (owner read/write only)
	for _, path := range []string{eventPath, resultPath, investigationPath} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("failed to stat %s: %v", path, err)
		}

		mode := info.Mode().Perm()
		expectedMode := os.FileMode(0600)
		if mode != expectedMode {
			t.Fatalf("file %s has incorrect permissions: expected %o, got %o", path, expectedMode, mode)
		}
	}

	// Check directory permissions are 0700 (owner read/write/execute only)
	for _, dir := range []string{incidentDir} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("failed to stat directory %s: %v", dir, err)
		}

		mode := info.Mode().Perm()
		expectedMode := os.FileMode(0700)
		if mode != expectedMode {
			t.Fatalf("directory %s has incorrect permissions: expected %o, got %o", dir, expectedMode, mode)
		}
	}
}

// TestFilesystemStorageSaveIncidentBinaryContent verifies handling of binary content in artifacts.
func TestFilesystemStorageSaveIncidentBinaryContent(t *testing.T) {
	tmpDir := t.TempDir()
	fs := NewFilesystemStorage(tmpDir)

	incidentID := "test-incident-binary"
	// Include various byte values including null bytes and high bytes
	binaryContent := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}

	artifacts := &IncidentArtifacts{
		EventJSON:      binaryContent,
		ResultJSON:     binaryContent,
		InvestigationMD: binaryContent,
	}

	ctx := context.Background()
	result, err := fs.SaveIncident(ctx, incidentID, artifacts)

	if err != nil {
		t.Fatalf("SaveIncident failed: %v", err)
	}

	// Verify binary content was written correctly
	eventPath := result.ArtifactURLs["event.json"]
	readContent, err := os.ReadFile(eventPath)
	if err != nil {
		t.Fatalf("failed to read event.json: %v", err)
	}

	if len(readContent) != len(binaryContent) {
		t.Fatalf("binary content length mismatch: expected %d, got %d", len(binaryContent), len(readContent))
	}

	for i, b := range readContent {
		if b != binaryContent[i] {
			t.Fatalf("binary content mismatch at byte %d: expected %d, got %d", i, binaryContent[i], b)
		}
	}
}

// TestFilesystemStorageSaveIncidentExistingDirectory verifies behavior when incident directory already exists.
func TestFilesystemStorageSaveIncidentExistingDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	fs := NewFilesystemStorage(tmpDir)

	incidentID := "test-incident-existing"

	// Create incident directory first
	incidentDir := filepath.Join(tmpDir, incidentID)
	if err := os.MkdirAll(incidentDir, 0700); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Save incident should succeed even if directory already exists
	artifacts := &IncidentArtifacts{
		EventJSON:      []byte(`{}`),
		ResultJSON:     []byte(`{}`),
		InvestigationMD: []byte(`# Report`),
	}

	ctx := context.Background()
	_, err := fs.SaveIncident(ctx, incidentID, artifacts)

	if err != nil {
		t.Fatalf("SaveIncident failed with existing directory: %v", err)
	}

	// Verify files were written
	eventPath := filepath.Join(incidentDir, "event.json")
	if _, err := os.Stat(eventPath); os.IsNotExist(err) {
		t.Fatalf("event.json not created")
	}
}

// TestFilesystemStorageSaveIncidentLargeContent verifies handling of large artifact content.
func TestFilesystemStorageSaveIncidentLargeContent(t *testing.T) {
	tmpDir := t.TempDir()
	fs := NewFilesystemStorage(tmpDir)

	incidentID := "test-incident-large"

	// Create large content (1MB)
	largeContent := make([]byte, 1024*1024)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	artifacts := &IncidentArtifacts{
		EventJSON:      largeContent,
		ResultJSON:     largeContent,
		InvestigationMD: largeContent,
	}

	ctx := context.Background()
	result, err := fs.SaveIncident(ctx, incidentID, artifacts)

	if err != nil {
		t.Fatalf("SaveIncident failed with large content: %v", err)
	}

	// Verify large content was written correctly
	eventPath := result.ArtifactURLs["event.json"]
	readContent, err := os.ReadFile(eventPath)
	if err != nil {
		t.Fatalf("failed to read large event.json: %v", err)
	}

	if len(readContent) != len(largeContent) {
		t.Fatalf("large content length mismatch: expected %d, got %d", len(largeContent), len(readContent))
	}
}

// TestFilesystemStorageSaveIncidentContextCancellation verifies behavior with cancelled context.
// Note: Current implementation doesn't check context during execution, but this test documents expected behavior.
func TestFilesystemStorageSaveIncidentContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	fs := NewFilesystemStorage(tmpDir)

	incidentID := "test-incident-cancel"
	artifacts := &IncidentArtifacts{
		EventJSON:      []byte(`{}`),
		ResultJSON:     []byte(`{}`),
		InvestigationMD: []byte(`# Report`),
	}

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Current implementation doesn't check context during execution,
	// so it will still succeed. This is acceptable for synchronous file operations.
	_, err := fs.SaveIncident(ctx, incidentID, artifacts)

	if err != nil {
		t.Fatalf("SaveIncident with cancelled context failed: %v", err)
	}
}

// TestFilesystemStorageSaveIncidentEmptyArtifacts verifies handling of empty artifact content.
func TestFilesystemStorageSaveIncidentEmptyArtifacts(t *testing.T) {
	tmpDir := t.TempDir()
	fs := NewFilesystemStorage(tmpDir)

	incidentID := "test-incident-empty"
	artifacts := &IncidentArtifacts{
		EventJSON:      []byte{},
		ResultJSON:     []byte{},
		InvestigationMD: []byte{},
	}

	ctx := context.Background()
	result, err := fs.SaveIncident(ctx, incidentID, artifacts)

	if err != nil {
		t.Fatalf("SaveIncident failed with empty artifacts: %v", err)
	}

	// Verify files were created, even if empty
	for _, path := range result.ArtifactURLs {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("failed to stat artifact: %v", err)
		}
		if info.Size() != 0 {
			t.Fatalf("expected empty file, got size %d", info.Size())
		}
	}
}

// TestFilesystemStorageSaveIncidentZeroExpiresAt verifies ExpiresAt is always zero.
func TestFilesystemStorageSaveIncidentZeroExpiresAt(t *testing.T) {
	tmpDir := t.TempDir()
	fs := NewFilesystemStorage(tmpDir)

	// Run multiple saves and verify ExpiresAt is always zero
	for i := 0; i < 5; i++ {
		incidentID := filepath.Join("test-incident", "zero-expires", "incident-"+string(rune('0'+i)))
		artifacts := &IncidentArtifacts{
			EventJSON:      []byte(`{}`),
			ResultJSON:     []byte(`{}`),
			InvestigationMD: []byte(`# Report`),
		}

		ctx := context.Background()
		result, err := fs.SaveIncident(ctx, incidentID, artifacts)

		if err != nil {
			t.Fatalf("SaveIncident failed: %v", err)
		}

		if !result.ExpiresAt.IsZero() {
			t.Fatalf("ExpiresAt should be zero time, got %v", result.ExpiresAt)
		}

		// Compare with a near-zero time to ensure it's the zero value, not just "old"
		expectedZero := time.Time{}
		if result.ExpiresAt != expectedZero {
			t.Fatalf("ExpiresAt mismatch: expected %v, got %v", expectedZero, result.ExpiresAt)
		}
	}
}
