package reporting

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Result represents the outcome of an agent execution
type Result struct {
	IncidentID  string    `json:"incident_id"`
	ExitCode    int       `json:"exit_code"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	Status      string    `json:"status"` // "success", "failed", "error"

	// Cloud storage fields (populated when cloud storage is used)
	PresignedURLs         map[string]string `json:"presigned_urls,omitempty"`           // URLs to access artifacts (e.g., "investigation.md" -> URL)
	PresignedURLsExpireAt *time.Time        `json:"presigned_urls_expire_at,omitempty"` // When the presigned URLs expire
}

// WriteResult writes the execution result to the workspace as JSON
func WriteResult(workspacePath string, result *Result) error {
	resultPath := filepath.Join(workspacePath, "result.json")

	// Marshal result to indented JSON
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	// Write to file with 0600 permissions
	if err := os.WriteFile(resultPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write result file: %w", err)
	}

	return nil
}
