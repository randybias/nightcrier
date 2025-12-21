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

	// Write cluster permissions if available
	artifactURLs := map[string]string{
		"incident.json":      incidentPath,
		"investigation.md":   investigationPath,
		"investigation.html": investigationHTMLPath,
	}
	if len(artifacts.ClusterPermissionsJSON) > 0 {
		permissionsPath := filepath.Join(incidentDir, "incident_cluster_permissions.json")
		if err := os.WriteFile(permissionsPath, artifacts.ClusterPermissionsJSON, 0600); err != nil {
			return nil, fmt.Errorf("failed to write incident_cluster_permissions.json: %w", err)
		}
		artifactURLs["incident_cluster_permissions.json"] = permissionsPath
	}

	// Create logs subdirectory and write agent logs and session archive
	logURLs := make(map[string]string)
	if artifacts.AgentLogs.Stdout != nil || artifacts.AgentLogs.Stderr != nil || artifacts.AgentLogs.Combined != nil || len(artifacts.ClaudeSessionArchive) > 0 {
		logsDir := filepath.Join(incidentDir, "logs")
		if err := os.MkdirAll(logsDir, 0700); err != nil {
			return nil, fmt.Errorf("failed to create logs directory: %w", err)
		}

		// Write stdout log if not empty
		if len(artifacts.AgentLogs.Stdout) > 0 {
			stdoutPath := filepath.Join(logsDir, "agent-stdout.log")
			if err := os.WriteFile(stdoutPath, artifacts.AgentLogs.Stdout, 0600); err != nil {
				return nil, fmt.Errorf("failed to write agent-stdout.log: %w", err)
			}
			logURLs["agent-stdout.log"] = stdoutPath
		}

		// Write stderr log if not empty
		if len(artifacts.AgentLogs.Stderr) > 0 {
			stderrPath := filepath.Join(logsDir, "agent-stderr.log")
			if err := os.WriteFile(stderrPath, artifacts.AgentLogs.Stderr, 0600); err != nil {
				return nil, fmt.Errorf("failed to write agent-stderr.log: %w", err)
			}
			logURLs["agent-stderr.log"] = stderrPath
		}

		// Write combined log if not empty
		if len(artifacts.AgentLogs.Combined) > 0 {
			combinedPath := filepath.Join(logsDir, "agent-full.log")
			if err := os.WriteFile(combinedPath, artifacts.AgentLogs.Combined, 0600); err != nil {
				return nil, fmt.Errorf("failed to write agent-full.log: %w", err)
			}
			logURLs["agent-full.log"] = combinedPath
		}

		// Write Claude session archive if not empty
		if len(artifacts.ClaudeSessionArchive) > 0 {
			sessionPath := filepath.Join(logsDir, "claude-session.tar.gz")
			if err := os.WriteFile(sessionPath, artifacts.ClaudeSessionArchive, 0600); err != nil {
				return nil, fmt.Errorf("failed to write claude-session.tar.gz: %w", err)
			}
			logURLs["claude-session.tar.gz"] = sessionPath
		}
	}

	// Return filesystem paths and zero ExpiresAt (filesystem paths don't expire)
	return &SaveResult{
		ReportURL:    investigationHTMLPath,
		ArtifactURLs: artifactURLs,
		LogURLs:      logURLs,
		ExpiresAt:    time.Time{},
	}, nil
}
