// Package k8s provides Kubernetes client initialization and utilities for the Nightcrier agent.
package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/randybias/nightcrier/internal/storage"
	"gocloud.dev/blob"
)

// ResultJSON represents the structure of result.json uploaded by the agent container.
type ResultJSON struct {
	// ExitCode is the exit code of the agent execution
	ExitCode int `json:"exit_code"`
	// Message provides additional context about the execution result
	Message string `json:"message,omitempty"`
}

// JobResults contains all artifacts retrieved from Object Storage after Job completion.
type JobResults struct {
	// ResultJSON contains the parsed result.json with exit code
	ResultJSON *ResultJSON
	// ReportMD is the investigation report markdown (report.md)
	ReportMD []byte
	// AgentLog is the agent execution log (agent.log)
	AgentLog []byte
	// CommandsExecuted is the log of commands executed by the agent (commands-executed.log)
	CommandsExecuted []byte
	// SessionArchive is the session.tar.gz archive (optional, for debugging)
	SessionArchive []byte
	// PromptSent is the prompt-sent.md file (what was actually sent to the agent)
	PromptSent []byte

	// Missing tracks which artifacts were missing (Job failed before upload)
	Missing []string
}

// ObjectStoreReader defines the interface for reading from object storage.
// This interface allows for testing with mock implementations.
type ObjectStoreReader interface {
	// Download reads data from the object store at the specified key
	Download(ctx context.Context, key string) ([]byte, error)
}

// RetrieveResultsConfig holds configuration for retrieving Job results.
type RetrieveResultsConfig struct {
	// IncidentID is the unique identifier for the incident
	IncidentID string
	// ObjectStore is the object store to read from
	ObjectStore ObjectStoreReader
	// IncludeSessionArchive controls whether to download session.tar.gz
	// This is a large file and only needed for debugging
	IncludeSessionArchive bool
}

// RetrieveResults downloads all artifacts from Object Storage after Job completion.
//
// It attempts to download the following files:
//   - result.json (required for exit code)
//   - report.md (investigation markdown)
//   - agent.log (agent execution log)
//   - commands-executed.log (commands executed by the agent)
//   - prompt-sent.md (the exact prompt sent to the agent)
//   - session.tar.gz (optional, only if IncludeSessionArchive=true)
//
// Missing files are tracked in the Missing slice. This is expected if the Job
// failed before completing all uploads (e.g., timeout, OOM, crash).
//
// Storage paths follow the pattern: incidents/{incident-id}/results/{filename}
func RetrieveResults(ctx context.Context, cfg RetrieveResultsConfig) (*JobResults, error) {
	if cfg.IncidentID == "" {
		return nil, fmt.Errorf("incident ID is required")
	}
	if cfg.ObjectStore == nil {
		return nil, fmt.Errorf("object store is required")
	}

	results := &JobResults{
		Missing: []string{},
	}

	// Storage path prefix for this incident
	pathPrefix := fmt.Sprintf("incidents/%s/results", cfg.IncidentID)

	// Download result.json (required for exit code)
	resultKey := fmt.Sprintf("%s/result.json", pathPrefix)
	resultData, err := cfg.ObjectStore.Download(ctx, resultKey)
	if err != nil {
		results.Missing = append(results.Missing, "result.json")
	} else {
		// Parse result.json
		var resultJSON ResultJSON
		if err := json.Unmarshal(resultData, &resultJSON); err != nil {
			return nil, fmt.Errorf("failed to parse result.json: %w", err)
		}
		results.ResultJSON = &resultJSON
	}

	// Download report.md
	reportKey := fmt.Sprintf("%s/report.md", pathPrefix)
	reportData, err := cfg.ObjectStore.Download(ctx, reportKey)
	if err != nil {
		results.Missing = append(results.Missing, "report.md")
	} else {
		results.ReportMD = reportData
	}

	// Download agent.log
	logKey := fmt.Sprintf("%s/agent.log", pathPrefix)
	logData, err := cfg.ObjectStore.Download(ctx, logKey)
	if err != nil {
		results.Missing = append(results.Missing, "agent.log")
	} else {
		results.AgentLog = logData
	}

	// Download commands-executed.log
	commandsKey := fmt.Sprintf("%s/commands-executed.log", pathPrefix)
	commandsData, err := cfg.ObjectStore.Download(ctx, commandsKey)
	if err != nil {
		results.Missing = append(results.Missing, "commands-executed.log")
	} else {
		results.CommandsExecuted = commandsData
	}

	// Download prompt-sent.md (what was actually sent to the agent)
	promptKey := fmt.Sprintf("%s/prompt-sent.md", pathPrefix)
	promptData, err := cfg.ObjectStore.Download(ctx, promptKey)
	if err != nil {
		results.Missing = append(results.Missing, "prompt-sent.md")
	} else {
		results.PromptSent = promptData
	}

	// Optionally download session.tar.gz (large file, only for debugging)
	if cfg.IncludeSessionArchive {
		sessionKey := fmt.Sprintf("%s/session.tar.gz", pathPrefix)
		sessionData, err := cfg.ObjectStore.Download(ctx, sessionKey)
		if err != nil {
			results.Missing = append(results.Missing, "session.tar.gz")
		} else {
			results.SessionArchive = sessionData
		}
	}

	return results, nil
}

// BlobObjectStoreAdapter adapts a blob.Bucket to the ObjectStoreReader interface.
// This allows using the existing ObjectStore with the RetrieveResults function.
type BlobObjectStoreAdapter struct {
	bucket *blob.Bucket
}

// NewBlobObjectStoreAdapter creates a new adapter from a blob.Bucket.
func NewBlobObjectStoreAdapter(bucket *blob.Bucket) *BlobObjectStoreAdapter {
	return &BlobObjectStoreAdapter{
		bucket: bucket,
	}
}

// Download reads data from the blob storage at the specified key.
func (a *BlobObjectStoreAdapter) Download(ctx context.Context, key string) ([]byte, error) {
	reader, err := a.bucket.NewReader(ctx, key, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create reader for key %s: %w", key, err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read data for key %s: %w", key, err)
	}

	return data, nil
}

// ObjectStoreAdapter adapts the existing storage.ObjectStore to the ObjectStoreReader interface.
// This provides a convenient way to use the ObjectStore with RetrieveResults.
type ObjectStoreAdapter struct {
	store *storage.ObjectStore
}

// NewObjectStoreAdapter creates a new adapter from a storage.ObjectStore.
func NewObjectStoreAdapter(store *storage.ObjectStore) *ObjectStoreAdapter {
	return &ObjectStoreAdapter{
		store: store,
	}
}

// Download reads data from the object store at the specified key.
// This delegates to the ObjectStore's Download method.
func (a *ObjectStoreAdapter) Download(ctx context.Context, key string) ([]byte, error) {
	return a.store.Download(ctx, key)
}
