// Package k8s provides artifact processing and database integration for K8s agent execution.
package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/randybias/nightcrier/internal/incident"
	"github.com/randybias/nightcrier/internal/reporting"
	"github.com/randybias/nightcrier/internal/storage"
)

// ArtifactProcessor handles processing and storage of Job artifacts after completion.
type ArtifactProcessor struct {
	objectStore *storage.ObjectStore
	stateStore  storage.StateStore
	storage     storage.Storage
}

// NewArtifactProcessor creates a new ArtifactProcessor with the provided dependencies.
func NewArtifactProcessor(objectStore *storage.ObjectStore, stateStore storage.StateStore, storageBackend storage.Storage) *ArtifactProcessor {
	return &ArtifactProcessor{
		objectStore: objectStore,
		stateStore:  stateStore,
		storage:     storageBackend,
	}
}

// ProcessJobResults processes all artifacts from a completed Job and updates the database.
// This implements Phase 4 of the K8s executor tasks:
//   - Convert markdown to HTML (Phase 4.1)
//   - Upload processed artifacts to Object Store (Phase 4.2)
//   - Update database with report content (Phase 4.3)
//   - Complete incident with final status (Phase 4.4)
//
// The function handles missing artifacts gracefully - if critical artifacts are missing,
// the incident is marked as failed with an appropriate failure reason.
//
// Returns ProcessJobResultsOutput with artifact information (reportURL, rootCause, confidence)
// for use in notifications and reporting.
func (p *ArtifactProcessor) ProcessJobResults(ctx context.Context, cfg ProcessJobResultsConfig) (*ProcessJobResultsOutput, error) {
	if cfg.IncidentID == "" {
		return nil, fmt.Errorf("incident ID is required")
	}
	if cfg.ExecutionID == "" {
		return nil, fmt.Errorf("execution ID is required")
	}
	if cfg.JobResults == nil {
		return nil, fmt.Errorf("job results are required")
	}

	// Check for critical failures - missing result.json means the Job never completed properly
	if cfg.JobResults.ResultJSON == nil {
		err := p.handleJobFailure(ctx, cfg.IncidentID, cfg.ExecutionID, "Job failed: result.json not found", cfg.JobResults.Missing)
		return nil, err
	}

	// Phase 4.1: Convert markdown to HTML (if report exists)
	var reportHTML []byte
	var reportMD []byte
	if len(cfg.JobResults.ReportMD) > 0 {
		reportMD = cfg.JobResults.ReportMD
		reportHTML = reporting.ConvertMarkdownToHTML(reportMD, cfg.IncidentID)
	} else {
		// Missing report is a critical failure - the agent didn't complete investigation
		err := p.handleJobFailure(ctx, cfg.IncidentID, cfg.ExecutionID, "Agent failed to generate investigation report", cfg.JobResults.Missing)
		return nil, err
	}

	// Phase 4.2: Upload processed artifacts to Object Store
	// Build artifact bundle for storage
	artifacts := &storage.IncidentArtifacts{
		InvestigationMD:   reportMD,
		InvestigationHTML: reportHTML,
		// Add other artifacts if provided in config
		IncidentJSON:           cfg.IncidentJSON,
		ClusterPermissionsJSON: cfg.PermissionsJSON,
		PromptSent:             cfg.JobResults.PromptSent,
	}

	// Add agent logs if available (these come from K8s Job results)
	if len(cfg.JobResults.AgentLog) > 0 {
		// Always provide stdout (the combined log contains stdout)
		artifacts.AgentLogs.Stdout = cfg.JobResults.AgentLog
		artifacts.AgentLogs.Combined = cfg.JobResults.AgentLog

		// In debug mode, also provide stderr (use combined log as it contains both)
		if cfg.Debug {
			artifacts.AgentLogs.Stderr = cfg.JobResults.AgentLog
		}
	}
	if len(cfg.JobResults.CommandsExecuted) > 0 {
		artifacts.AgentLogs.CommandsExecuted = cfg.JobResults.CommandsExecuted
	}

	// Include session archive in debug mode
	if cfg.Debug && len(cfg.JobResults.SessionArchive) > 0 {
		artifacts.AgentSessionArchive = cfg.JobResults.SessionArchive
	}

	// Upload all artifacts and get URLs
	saveResult, err := p.storage.SaveIncident(ctx, cfg.IncidentID, artifacts)
	if err != nil {
		return nil, fmt.Errorf("failed to save incident artifacts: %w", err)
	}

	// Extract root cause and confidence from report for notifications
	rootCause, confidence, err := reporting.ExtractSummaryFromContent(reportMD)
	if err != nil {
		// Log warning but don't fail - use defaults
		rootCause = "See investigation report"
		confidence = "UNKNOWN"
	}

	// Phase 4.3: Update database with report content
	// Record the agent execution with log URLs (not file paths)
	logPaths := make(map[string]string)
	for filename, url := range saveResult.LogURLs {
		logPaths[filename] = url
	}
	// Also include canonical URLs for long-term storage
	for filename, url := range saveResult.CanonicalURLs {
		if _, exists := logPaths[filename]; !exists {
			logPaths[filename+"_canonical"] = url
		}
	}

	now := time.Now()
	exitCode := cfg.JobResults.ResultJSON.ExitCode
	agentExecution := &storage.AgentExecution{
		ExecutionID:  cfg.ExecutionID,
		IncidentID:   cfg.IncidentID,
		StartedAt:    cfg.StartedAt,
		CompletedAt:  &now,
		ExitCode:     &exitCode,
		ErrorMessage: cfg.JobResults.ResultJSON.Message,
		LogPaths:     logPaths,
	}

	if err := p.stateStore.RecordAgentExecution(ctx, agentExecution); err != nil {
		return nil, fmt.Errorf("failed to record agent execution: %w", err)
	}

	// Record the triage report with both markdown and HTML
	reportID := uuid.New().String()
	triageReport := &storage.TriageReport{
		ReportID:       reportID,
		IncidentID:     cfg.IncidentID,
		ExecutionID:    cfg.ExecutionID,
		GeneratedAt:    now,
		ReportMarkdown: string(reportMD),
		ReportHTML:     string(reportHTML),
	}

	if err := p.stateStore.RecordTriageReport(ctx, triageReport); err != nil {
		return nil, fmt.Errorf("failed to record triage report: %w", err)
	}

	// Phase 4.4: Complete incident with exit code
	failureReason := ""
	if exitCode != 0 {
		failureReason = fmt.Sprintf("Agent exited with code %d", exitCode)
		if cfg.JobResults.ResultJSON.Message != "" {
			failureReason += ": " + cfg.JobResults.ResultJSON.Message
		}
	}

	if err := p.stateStore.CompleteIncident(ctx, cfg.IncidentID, exitCode, failureReason); err != nil {
		return nil, fmt.Errorf("failed to complete incident: %w", err)
	}

	// Return artifact information for notifications
	output := &ProcessJobResultsOutput{
		ReportURL:  saveResult.ReportURL,
		RootCause:  rootCause,
		Confidence: confidence,
	}

	return output, nil
}

// ProcessJobResultsConfig holds configuration for processing Job results.
type ProcessJobResultsConfig struct {
	// IncidentID is the unique identifier for the incident
	IncidentID string
	// ExecutionID is the unique identifier for this execution attempt
	ExecutionID string
	// JobResults contains the artifacts retrieved from Object Store
	JobResults *JobResults
	// StartedAt is when the agent execution began
	StartedAt time.Time

	// Additional artifacts to upload (these were created by the orchestrator)
	IncidentJSON    []byte
	PermissionsJSON []byte

	// Debug mode - includes session archive and separate stderr
	Debug bool
}

// ProcessJobResultsOutput holds the results of processing Job artifacts.
type ProcessJobResultsOutput struct {
	// ReportURL is the URL to the investigation report (HTML)
	ReportURL string
	// RootCause is the root cause extracted from the report
	RootCause string
	// Confidence is the confidence level (HIGH/MEDIUM/LOW/UNKNOWN)
	Confidence string
}

// handleJobFailure records a Job failure in the database and completes the incident.
// This is called when critical artifacts are missing or the Job failed before producing results.
func (p *ArtifactProcessor) handleJobFailure(ctx context.Context, incidentID, executionID, failureReason string, missingArtifacts []string) error {
	// Build detailed failure message
	detailedReason := failureReason
	if len(missingArtifacts) > 0 {
		detailedReason += fmt.Sprintf(" (missing artifacts: %v)", missingArtifacts)
	}

	// Record agent execution with failure
	now := time.Now()
	exitCode := -1
	agentExecution := &storage.AgentExecution{
		ExecutionID:  executionID,
		IncidentID:   incidentID,
		StartedAt:    time.Now().Add(-5 * time.Minute), // Approximate - we don't know exact start time
		CompletedAt:  &now,
		ExitCode:     &exitCode,
		ErrorMessage: detailedReason,
		LogPaths:     make(map[string]string),
	}

	if err := p.stateStore.RecordAgentExecution(ctx, agentExecution); err != nil {
		return fmt.Errorf("failed to record failed agent execution: %w", err)
	}

	// Complete incident with failure
	if err := p.stateStore.CompleteIncident(ctx, incidentID, exitCode, detailedReason); err != nil {
		return fmt.Errorf("failed to complete failed incident: %w", err)
	}

	return fmt.Errorf("%s", detailedReason)
}

// UploadProcessedArtifacts uploads additional processed artifacts to Object Store.
// This is used for artifacts created by the processor (like HTML reports) that need
// to be stored alongside the Job outputs.
func (p *ArtifactProcessor) UploadProcessedArtifacts(ctx context.Context, incidentID string, artifacts map[string][]byte) error {
	for filename, data := range artifacts {
		if len(data) == 0 {
			continue
		}

		key := fmt.Sprintf("incidents/%s/results/%s", incidentID, filename)
		contentType := getContentType(filename)

		if err := p.objectStore.Upload(ctx, key, data, contentType); err != nil {
			return fmt.Errorf("failed to upload %s: %w", filename, err)
		}
	}

	return nil
}

// getContentType returns the appropriate MIME type for a filename.
func getContentType(filename string) string {
	// Check for .tar.gz first (compound extension)
	if len(filename) > 7 && filename[len(filename)-7:] == ".tar.gz" {
		return "application/gzip"
	}

	// Extract single extension
	ext := ""
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			ext = filename[i:]
			break
		}
	}

	switch ext {
	case ".md":
		return "text/markdown; charset=utf-8"
	case ".html":
		return "text/html; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".log":
		return "text/plain; charset=utf-8"
	case ".tgz", ".gz":
		return "application/gzip"
	default:
		return "application/octet-stream"
	}
}

// GetIncidentStatus retrieves the current status of an incident from the database.
// This is useful for checking the outcome after processing completes.
func (p *ArtifactProcessor) GetIncidentStatus(ctx context.Context, incidentID string) (*incident.Incident, error) {
	return p.stateStore.GetIncident(ctx, incidentID)
}

// SerializeIncidentToJSON converts an incident to JSON for storage.
// This is a helper function for creating the incident.json artifact.
func SerializeIncidentToJSON(inc *incident.Incident) ([]byte, error) {
	data, err := json.MarshalIndent(inc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal incident to JSON: %w", err)
	}
	return data, nil
}
