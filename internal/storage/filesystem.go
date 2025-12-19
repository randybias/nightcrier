package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// FilesystemStorage implements the Storage interface by persisting incident artifacts to the local filesystem.
type FilesystemStorage struct {
	workspaceRoot string
}

// NewFilesystemStorage creates a new FilesystemStorage instance with the given workspace root directory.
func NewFilesystemStorage(workspaceRoot string) *FilesystemStorage {
	return &FilesystemStorage{
		workspaceRoot: workspaceRoot,
	}
}

// SaveIncident persists all incident artifacts to the local filesystem.
// It creates a directory structure: <workspace-root>/<incident-id>/ containing incident.json and investigation files
// For filesystem storage, it returns filesystem paths (not URLs) and a zero ExpiresAt time.
func (fs *FilesystemStorage) SaveIncident(ctx context.Context, incidentID string, artifacts *IncidentArtifacts) (*SaveResult, error) {
	if artifacts == nil {
		return nil, fmt.Errorf("artifacts cannot be nil")
	}

	incidentDir := filepath.Join(fs.workspaceRoot, incidentID)

	// Create incident directory with secure permissions (owner read/write/execute only)
	if err := os.MkdirAll(incidentDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create incident directory: %w", err)
	}

	// Write incident.json
	incidentPath := filepath.Join(incidentDir, "incident.json")
	if err := os.WriteFile(incidentPath, artifacts.IncidentJSON, 0600); err != nil {
		return nil, fmt.Errorf("failed to write incident.json: %w", err)
	}

	// Write investigation.md
	investigationPath := filepath.Join(incidentDir, "investigation.md")
	if err := os.WriteFile(investigationPath, artifacts.InvestigationMD, 0600); err != nil {
		return nil, fmt.Errorf("failed to write investigation.md: %w", err)
	}

	// Write investigation.html
	investigationHTMLPath := filepath.Join(incidentDir, "investigation.html")
	if err := os.WriteFile(investigationHTMLPath, artifacts.InvestigationHTML, 0600); err != nil {
		return nil, fmt.Errorf("failed to write investigation.html: %w", err)
	}

	// Return filesystem paths and zero ExpiresAt (filesystem paths don't expire)
	return &SaveResult{
		ReportURL: investigationHTMLPath,
		ArtifactURLs: map[string]string{
			"incident.json":       incidentPath,
			"investigation.md":    investigationPath,
			"investigation.html":  investigationHTMLPath,
		},
		ExpiresAt: time.Time{},
	}, nil
}
