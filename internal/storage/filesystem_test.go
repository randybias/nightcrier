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
	incidentJSON := []byte(`{"incidentId":"test-001","status":"resolved"}`)
	investigationMD := []byte(`# Investigation Report\nAll systems healthy.`)
	investigationHTML := []byte(`<h1>Investigation Report</h1><p>All systems healthy.</p>`)

	artifacts := &IncidentArtifacts{
		IncidentJSON:      incidentJSON,
		InvestigationMD:   investigationMD,
		InvestigationHTML: investigationHTML,
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
	incidentPath := filepath.Join(incidentDir, "incident.json")
	incidentData, err := os.ReadFile(incidentPath)
	if err != nil {
		t.Fatalf("failed to read incident.json: %v", err)
	}
	if string(incidentData) != string(incidentJSON) {
		t.Fatalf("incident.json content mismatch: expected %q, got %q", string(incidentJSON), string(incidentData))
	}

	investigationPath := filepath.Join(incidentDir, "investigation.md")
	investigationData, err := os.ReadFile(investigationPath)
	if err != nil {
		t.Fatalf("failed to read investigation.md: %v", err)
	}
	if string(investigationData) != string(investigationMD) {
		t.Fatalf("investigation.md content mismatch: expected %q, got %q", string(investigationMD), string(investigationData))
	}

	investigationHTMLPath := filepath.Join(incidentDir, "investigation.html")
	investigationHTMLData, err := os.ReadFile(investigationHTMLPath)
	if err != nil {
		t.Fatalf("failed to read investigation.html: %v", err)
	}
	if string(investigationHTMLData) != string(investigationHTML) {
		t.Fatalf("investigation.html content mismatch: expected %q, got %q", string(investigationHTML), string(investigationHTMLData))
	}
}

// TestFilesystemStorageSaveResultContent verifies the SaveResult contains correct paths and URLs.
func TestFilesystemStorageSaveResultContent(t *testing.T) {
	tmpDir := t.TempDir()
	fs := NewFilesystemStorage(tmpDir)

	incidentID := "test-incident-002"
	artifacts := &IncidentArtifacts{
		IncidentJSON:      []byte(`{}`),
		InvestigationHTML:     []byte(`{}`),
		InvestigationMD: []byte(`# Report`),
	}

	ctx := context.Background()
	result, err := fs.SaveIncident(ctx, incidentID, artifacts)

	if err != nil {
		t.Fatalf("SaveIncident failed: %v", err)
	}

	expectedIncidentDir := filepath.Join(tmpDir, incidentID)
	expectedReportPath := filepath.Join(expectedIncidentDir, "investigation.html")

	// Verify ReportURL points to investigation.html
	if result.ReportURL != expectedReportPath {
		t.Fatalf("ReportURL mismatch: expected %q, got %q", expectedReportPath, result.ReportURL)
	}

	// Verify ArtifactURLs map contains all three artifacts
	expectedArtifacts := map[string]string{
		"incident.json":       filepath.Join(expectedIncidentDir, "incident.json"),
		"investigation.html":  filepath.Join(expectedIncidentDir, "investigation.html"),
		"investigation.md":    filepath.Join(expectedIncidentDir, "investigation.md"),
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
			IncidentJSON:      []byte(`{"incident":"` + incidentID + `"}`),
			InvestigationHTML:     []byte(`{"status":"ok"}`),
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
		IncidentJSON:      []byte(`{}`),
		InvestigationHTML:     []byte(`{}`),
		InvestigationMD: []byte(`# Report`),
	}

	ctx := context.Background()
	_, err := fs.SaveIncident(ctx, incidentID, artifacts)

	if err != nil {
		t.Fatalf("SaveIncident failed: %v", err)
	}

	incidentDir := filepath.Join(tmpDir, incidentID)
	eventPath := filepath.Join(incidentDir, "incident.json")
	resultPath := filepath.Join(incidentDir, "investigation.html")
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
		IncidentJSON:      binaryContent,
		InvestigationHTML:     binaryContent,
		InvestigationMD: binaryContent,
	}

	ctx := context.Background()
	result, err := fs.SaveIncident(ctx, incidentID, artifacts)

	if err != nil {
		t.Fatalf("SaveIncident failed: %v", err)
	}

	// Verify binary content was written correctly
	eventPath := result.ArtifactURLs["incident.json"]
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
		IncidentJSON:      []byte(`{}`),
		InvestigationHTML:     []byte(`{}`),
		InvestigationMD: []byte(`# Report`),
	}

	ctx := context.Background()
	_, err := fs.SaveIncident(ctx, incidentID, artifacts)

	if err != nil {
		t.Fatalf("SaveIncident failed with existing directory: %v", err)
	}

	// Verify files were written
	eventPath := filepath.Join(incidentDir, "incident.json")
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
		IncidentJSON:      largeContent,
		InvestigationHTML:     largeContent,
		InvestigationMD: largeContent,
	}

	ctx := context.Background()
	result, err := fs.SaveIncident(ctx, incidentID, artifacts)

	if err != nil {
		t.Fatalf("SaveIncident failed with large content: %v", err)
	}

	// Verify large content was written correctly
	eventPath := result.ArtifactURLs["incident.json"]
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
		IncidentJSON:      []byte(`{}`),
		InvestigationHTML:     []byte(`{}`),
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
		IncidentJSON:      []byte{},
		InvestigationHTML:     []byte{},
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
			IncidentJSON:      []byte(`{}`),
			InvestigationHTML:     []byte(`{}`),
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
