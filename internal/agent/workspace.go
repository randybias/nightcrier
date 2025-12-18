package agent

import (
	"fmt"
	"os"
	"path/filepath"
)

// WorkspaceManager manages incident workspace directories
type WorkspaceManager struct {
	root string
}

// NewWorkspaceManager creates a new workspace manager with the given root directory
func NewWorkspaceManager(root string) *WorkspaceManager {
	return &WorkspaceManager{
		root: root,
	}
}

// Create creates a workspace directory for the given incident ID
// Returns the absolute path to the created workspace
func (w *WorkspaceManager) Create(incidentID string) (string, error) {
	workspacePath := filepath.Join(w.root, incidentID)

	// Create the directory with 0700 permissions (owner read/write/execute only)
	if err := os.MkdirAll(workspacePath, 0700); err != nil {
		return "", fmt.Errorf("failed to create workspace directory: %w", err)
	}

	return workspacePath, nil
}
