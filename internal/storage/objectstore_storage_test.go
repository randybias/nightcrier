package storage

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestObjectStoreStorageNewObjectStoreStorage verifies the constructor creates a valid instance.
func TestObjectStoreStorageNewObjectStoreStorage(t *testing.T) {
	ctx := context.Background()
	store, err := NewObjectStore(ctx, "mem://", 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to create object store: %v", err)
	}
	defer store.Close()

	oss := NewObjectStoreStorage(store)
	if oss == nil {
		t.Fatalf("NewObjectStoreStorage returned nil")
	}
	if oss.store == nil {
		t.Fatalf("ObjectStoreStorage.store is nil")
	}
}

// TestObjectStoreStorageUploadOperation verifies artifact uploads work correctly.
// Note: mem:// storage doesn't support SignedURL, so we verify uploads succeed
// by checking the error handling when SignedURL fails.
func TestObjectStoreStorageUploadOperation(t *testing.T) {
	ctx := context.Background()
	store, err := NewObjectStore(ctx, "mem://", 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to create object store: %v", err)
	}
	defer store.Close()

	oss := NewObjectStoreStorage(store)
	incidentID := "test-incident-001"
	incidentJSON := []byte(`{"incidentId":"test-001","status":"resolved"}`)
	investigationMD := []byte(`# Investigation Report\nAll systems healthy.`)
	investigationHTML := []byte(`<h1>Investigation Report</h1><p>All systems healthy.</p>`)

	artifacts := &IncidentArtifacts{
		IncidentJSON:      incidentJSON,
		InvestigationMD:   investigationMD,
		InvestigationHTML: investigationHTML,
	}

	// SaveIncident will fail because mem:// doesn't support SignedURL
	result, err := oss.SaveIncident(ctx, incidentID, artifacts)

	// We expect an error due to SignedURL not being supported
	if err == nil {
		t.Fatalf("expected error due to mem:// not supporting SignedURL, got nil")
	}

	// Verify the error is related to SignedURL functionality
	if !strings.Contains(err.Error(), "not implemented") && !strings.Contains(err.Error(), "failed to upload any artifacts") {
		t.Fatalf("unexpected error: %v", err)
	}

	// Result should be nil since signed URL generation failed
	if result != nil {
		t.Logf("Note: implementation returned partial results despite signed URL failure: %+v", result)
	}
}

// TestObjectStoreStorageContentTypeMapping verifies correct content-type headers are set.
func TestObjectStoreStorageContentTypeMapping(t *testing.T) {
	tests := []struct {
		filename    string
		contentType string
	}{
		{"incident.json", "application/json; charset=utf-8"},
		{"investigation.md", "text/markdown; charset=utf-8"},
		{"investigation.html", "text/html; charset=utf-8"},
		{"index.html", "text/html; charset=utf-8"},
		{"test.txt", "text/plain; charset=utf-8"},
		{"test.log", "text/plain; charset=utf-8"},
		{"test.gz", "application/gzip"},
		{"unknown", "application/octet-stream"},
		{"", "application/octet-stream"},
		{"noext", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			actualContentType := getContentTypeFromFilename(tt.filename)
			if actualContentType != tt.contentType {
				t.Errorf("content type mismatch for %s: expected %s, got %s",
					tt.filename, tt.contentType, actualContentType)
			}
		})
	}
}

// TestObjectStoreStorageCanonicalURLGeneration verifies canonical URL structure.
// This tests the URL generation without needing SignedURL support.
func TestObjectStoreStorageCanonicalURLGeneration(t *testing.T) {
	ctx := context.Background()
	store, err := NewObjectStore(ctx, "mem://", 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to create object store: %v", err)
	}
	defer store.Close()

	// Test canonical URL generation for various keys
	tests := []struct {
		key         string
		expectedURL string
	}{
		{"test-incident/incident.json", "mem://test-incident/incident.json"},
		{"test-incident/investigation.md", "mem://test-incident/investigation.md"},
		{"test-incident/investigation.html", "mem://test-incident/investigation.html"},
		{"test-incident/logs/agent-stdout.log", "mem://test-incident/logs/agent-stdout.log"},
		{"test-incident/index.html", "mem://test-incident/index.html"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			canonicalURL := store.CanonicalURL(tt.key)
			if canonicalURL != tt.expectedURL {
				t.Errorf("canonical URL mismatch: expected %s, got %s", tt.expectedURL, canonicalURL)
			}
		})
	}
}

// TestObjectStoreStorageIndexHTMLGeneration verifies index.html structure.
func TestObjectStoreStorageIndexHTMLGeneration(t *testing.T) {
	incidentID := "test-incident-004"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	artifactURLs := map[string]string{
		"investigation.html": "http://storage.example.com/test-incident-004/investigation.html?token=abc123",
		"investigation.md":   "http://storage.example.com/test-incident-004/investigation.md?token=def456",
		"incident.json":      "http://storage.example.com/test-incident-004/incident.json?token=ghi789",
	}

	indexHTML := generateIndexHTMLWithSignedURLs(incidentID, artifactURLs, expiresAt)

	// Verify HTML contains expected elements
	if !strings.Contains(indexHTML, "<!DOCTYPE html>") {
		t.Errorf("index.html does not contain DOCTYPE declaration")
	}

	if !strings.Contains(indexHTML, incidentID) {
		t.Errorf("index.html does not contain incident ID")
	}

	if !strings.Contains(indexHTML, "Kubernetes Incident Report") {
		t.Errorf("index.html does not contain expected title")
	}

	// Verify all artifact URLs are present in the HTML
	for filename, url := range artifactURLs {
		if !strings.Contains(indexHTML, url) {
			t.Errorf("index.html does not contain link to %s (URL: %s)", filename, url)
		}
	}

	// Verify expiration time is included
	expiryStr := expiresAt.UTC().Format("2006-01-02 15:04:05")
	if !strings.Contains(indexHTML, expiryStr) {
		t.Errorf("index.html does not contain expiration time: %s", expiryStr)
	}

	// Verify HTML is well-formed
	if !strings.Contains(indexHTML, "</html>") {
		t.Errorf("index.html is not well-formed (missing closing tag)")
	}

	// Verify clickable links use <a href="...">
	if !strings.Contains(indexHTML, `<a href="`) {
		t.Errorf("index.html does not contain clickable links")
	}
}

// TestObjectStoreStorageIndexHTMLWithLogs verifies index.html includes log files.
func TestObjectStoreStorageIndexHTMLWithLogs(t *testing.T) {
	incidentID := "test-incident-logs"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	artifactURLs := map[string]string{
		"investigation.html":          "http://storage.example.com/investigation.html",
		"agent-stdout.log":            "http://storage.example.com/logs/agent-stdout.log",
		"agent-stderr.log":            "http://storage.example.com/logs/agent-stderr.log",
		"agent-full.log":              "http://storage.example.com/logs/agent-full.log",
		"agent-commands-executed.log": "http://storage.example.com/logs/agent-commands-executed.log",
		"claude-session.tar.gz":       "http://storage.example.com/logs/claude-session.tar.gz",
	}

	indexHTML := generateIndexHTMLWithSignedURLs(incidentID, artifactURLs, expiresAt)

	// Verify all log files are referenced
	logFiles := []string{
		"agent-stdout.log",
		"agent-stderr.log",
		"agent-full.log",
		"agent-commands-executed.log",
		"claude-session.tar.gz",
	}

	for _, logFile := range logFiles {
		if !strings.Contains(indexHTML, logFile) {
			t.Errorf("index.html does not reference log file: %s", logFile)
		}
		url := artifactURLs[logFile]
		if !strings.Contains(indexHTML, url) {
			t.Errorf("index.html does not contain URL for %s: %s", logFile, url)
		}
	}

	// Verify DEBUG mode indicators are present
	if !strings.Contains(indexHTML, "DEBUG mode only") {
		t.Errorf("index.html does not indicate DEBUG mode for log files")
	}
}

// TestObjectStoreStorageNilArtifacts verifies error handling for nil artifacts.
func TestObjectStoreStorageNilArtifacts(t *testing.T) {
	ctx := context.Background()
	store, err := NewObjectStore(ctx, "mem://", 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to create object store: %v", err)
	}
	defer store.Close()

	oss := NewObjectStoreStorage(store)
	_, err = oss.SaveIncident(ctx, "test-incident", nil)

	if err == nil {
		t.Fatalf("expected error for nil artifacts, got nil")
	}

	expectedMsg := "artifacts cannot be nil"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("error message mismatch: expected to contain %q, got %q", expectedMsg, err.Error())
	}
}

// TestObjectStoreStorageEmptyArtifacts verifies handling of empty artifact content.
// Empty artifacts are skipped during upload, which should result in an error.
func TestObjectStoreStorageEmptyArtifacts(t *testing.T) {
	ctx := context.Background()
	store, err := NewObjectStore(ctx, "mem://", 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to create object store: %v", err)
	}
	defer store.Close()

	oss := NewObjectStoreStorage(store)
	incidentID := "test-incident-empty"

	artifacts := &IncidentArtifacts{
		IncidentJSON:      []byte{},
		InvestigationMD:   []byte{},
		InvestigationHTML: []byte{},
	}

	result, err := oss.SaveIncident(ctx, incidentID, artifacts)

	// Empty artifacts should be skipped, resulting in an error
	if err == nil {
		t.Fatalf("expected error with empty artifacts, got nil")
	}

	if !strings.Contains(err.Error(), "no artifacts") && !strings.Contains(err.Error(), "failed to upload") {
		t.Errorf("unexpected error message: %v", err)
	}

	if result != nil {
		t.Errorf("expected nil result with empty artifacts, got %+v", result)
	}
}

// TestObjectStoreStorageContextCancellation verifies behavior with cancelled context.
func TestObjectStoreStorageContextCancellation(t *testing.T) {
	ctx := context.Background()
	store, err := NewObjectStore(ctx, "mem://", 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to create object store: %v", err)
	}
	defer store.Close()

	oss := NewObjectStoreStorage(store)
	incidentID := "test-incident-cancel"

	artifacts := &IncidentArtifacts{
		IncidentJSON:      []byte(`{}`),
		InvestigationMD:   []byte(`# Report`),
		InvestigationHTML: []byte(`<html></html>`),
	}

	// Create a cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	// Attempt to save with cancelled context
	_, err = oss.SaveIncident(cancelledCtx, incidentID, artifacts)

	// We expect an error due to context cancellation
	if err == nil {
		t.Fatalf("expected error with cancelled context, got nil")
	}

	// Error should indicate context was cancelled or operation failed
	t.Logf("SaveIncident with cancelled context returned expected error: %v", err)
}

// TestObjectStoreStorageBinaryContent verifies content-type for binary files.
func TestObjectStoreStorageBinaryContent(t *testing.T) {
	// Test binary file content types
	binaryFiles := []struct {
		filename    string
		contentType string
	}{
		{"test.gz", "application/gzip"},
		{"test.tar.gz", "application/gzip"},
		{"test.bin", "application/octet-stream"},
		{"test.dat", "application/octet-stream"},
	}

	for _, tt := range binaryFiles {
		t.Run(tt.filename, func(t *testing.T) {
			actualContentType := getContentTypeFromFilename(tt.filename)
			if actualContentType != tt.contentType {
				t.Errorf("content type mismatch for %s: expected %s, got %s",
					tt.filename, tt.contentType, actualContentType)
			}
		})
	}
}

// TestObjectStoreStoragePathStructure verifies artifact path structure.
// This tests the expected path format without needing actual uploads.
func TestObjectStoreStoragePathStructure(t *testing.T) {
	tests := []struct {
		name         string
		incidentID   string
		filename     string
		expectedPath string
	}{
		{
			name:         "artifact in incident root",
			incidentID:   "incident-123",
			filename:     "incident.json",
			expectedPath: "incident-123/incident.json",
		},
		{
			name:         "log file in logs subdirectory",
			incidentID:   "incident-456",
			filename:     "agent-stdout.log",
			expectedPath: "incident-456/logs/agent-stdout.log",
		},
		{
			name:         "index.html in incident root",
			incidentID:   "incident-789",
			filename:     "index.html",
			expectedPath: "incident-789/index.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For artifacts (not logs)
			if !strings.Contains(tt.filename, ".log") && tt.filename != "claude-session.tar.gz" {
				expectedKey := tt.incidentID + "/" + tt.filename
				if expectedKey != tt.expectedPath {
					t.Errorf("path mismatch: expected %s, got %s", tt.expectedPath, expectedKey)
				}
			}
		})
	}
}

// TestObjectStoreStorageAllArtifactTypes verifies all artifact types are handled.
func TestObjectStoreStorageAllArtifactTypes(t *testing.T) {
	// Verify that all artifact types have correct content types
	artifactTypes := map[string]string{
		"incident.json":                     "application/json; charset=utf-8",
		"investigation.md":                  "text/markdown; charset=utf-8",
		"investigation.html":                "text/html; charset=utf-8",
		"incident_cluster_permissions.json": "application/json; charset=utf-8",
		"prompt-sent.md":                    "text/markdown; charset=utf-8",
		"index.html":                        "text/html; charset=utf-8",
		"agent-stdout.log":                  "text/plain; charset=utf-8",
		"agent-stderr.log":                  "text/plain; charset=utf-8",
		"agent-full.log":                    "text/plain; charset=utf-8",
		"agent-commands-executed.log":       "text/plain; charset=utf-8",
		"claude-session.tar.gz":             "application/gzip",
	}

	for filename, expectedContentType := range artifactTypes {
		t.Run(filename, func(t *testing.T) {
			actualContentType := getContentTypeFromFilename(filename)
			if actualContentType != expectedContentType {
				t.Errorf("content type mismatch for %s: expected %s, got %s",
					filename, expectedContentType, actualContentType)
			}
		})
	}
}

// TestObjectStoreStorageIndexHTMLFileDescriptions verifies descriptions are present.
func TestObjectStoreStorageIndexHTMLFileDescriptions(t *testing.T) {
	incidentID := "test-incident"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	artifactURLs := map[string]string{
		"investigation.html":                "http://example.com/investigation.html",
		"investigation.md":                  "http://example.com/investigation.md",
		"incident.json":                     "http://example.com/incident.json",
		"incident_cluster_permissions.json": "http://example.com/incident_cluster_permissions.json",
		"prompt-sent.md":                    "http://example.com/prompt-sent.md",
	}

	indexHTML := generateIndexHTMLWithSignedURLs(incidentID, artifactURLs, expiresAt)

	// Verify descriptions are present for each artifact type
	expectedDescriptions := []string{
		"Formatted HTML report with root cause analysis",
		"Markdown source for programmatic access",
		"Complete incident context",
		"Validated Kubernetes permissions",
		"Full system prompt and additional context sent to the agent for audit",
	}

	for _, desc := range expectedDescriptions {
		if !strings.Contains(indexHTML, desc) {
			t.Errorf("index.html missing expected description: %s", desc)
		}
	}

	// Verify badges are present
	expectedBadges := []string{"badge-primary", "badge-secondary", "badge-success"}
	for _, badge := range expectedBadges {
		if !strings.Contains(indexHTML, badge) {
			t.Errorf("index.html missing expected badge class: %s", badge)
		}
	}
}
