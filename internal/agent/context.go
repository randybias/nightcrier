package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rbias/kubernetes-mcp-alerts-event-runner/internal/events"
)

// WriteEventContext writes a FaultEvent to the workspace as JSON
func WriteEventContext(workspacePath string, event *events.FaultEvent) error {
	eventPath := filepath.Join(workspacePath, "event.json")

	// Marshal event to indented JSON
	data, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Write to file with 0600 permissions (owner read/write only)
	if err := os.WriteFile(eventPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write event file: %w", err)
	}

	return nil
}
