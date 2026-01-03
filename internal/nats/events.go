package nats

import (
	"encoding/json"
	"fmt"
	"time"
)

// EventType represents the type of progress event
type EventType string

const (
	// EventTypeRunStarted indicates a triage run has started
	EventTypeRunStarted EventType = "run.started"
	// EventTypeRunCompleted indicates a triage run has completed
	EventTypeRunCompleted EventType = "run.completed"
	// EventTypeExecuting indicates the agent is currently executing a command
	EventTypeExecuting EventType = "executing"
	// EventTypeError indicates an error occurred during execution
	EventTypeError EventType = "error"
)

// ProgressEvent represents a triage agent progress event emitted to NATS.
// This matches the JSON schema defined in the proposal.
type ProgressEvent struct {
	// IncidentID is the unique identifier for the incident being triaged
	IncidentID string `json:"incident_id"`
	// Cluster is the name of the cluster where the incident occurred
	Cluster string `json:"cluster"`
	// Timestamp is the ISO8601 timestamp when the event occurred
	Timestamp string `json:"timestamp"`
	// EventType is the type of event (run.started, run.completed, executing, error)
	EventType string `json:"event_type"`
	// AgentCLI is the agent CLI being used (claude, codex, gemini, goose)
	AgentCLI string `json:"agent_cli,omitempty"`
	// Model is the model being used (sonnet, opus, etc.)
	Model string `json:"model,omitempty"`
	// Activity is a short description of what the agent is doing (max 100 chars)
	// Only present for "executing" events
	Activity string `json:"activity,omitempty"`
	// ExitCode is the process exit code (only for run.completed events)
	ExitCode *int `json:"exit_code,omitempty"`
	// ErrorMessage contains error details (only for error events)
	ErrorMessage string `json:"error_message,omitempty"`
}

// NewProgressEvent creates a new ProgressEvent with the current timestamp
func NewProgressEvent(incidentID, cluster string, eventType EventType) *ProgressEvent {
	return &ProgressEvent{
		IncidentID: incidentID,
		Cluster:    cluster,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		EventType:  string(eventType),
	}
}

// WithAgentInfo adds agent CLI and model information to the event
func (e *ProgressEvent) WithAgentInfo(agentCLI, model string) *ProgressEvent {
	e.AgentCLI = agentCLI
	e.Model = model
	return e
}

// WithActivity adds activity information (max 100 chars) to the event
func (e *ProgressEvent) WithActivity(activity string) *ProgressEvent {
	// Truncate activity to 100 characters if needed
	if len(activity) > 100 {
		e.Activity = activity[:100]
	} else {
		e.Activity = activity
	}
	return e
}

// WithExitCode adds exit code information to the event
func (e *ProgressEvent) WithExitCode(exitCode int) *ProgressEvent {
	e.ExitCode = &exitCode
	return e
}

// WithError adds error message information to the event
func (e *ProgressEvent) WithError(errorMessage string) *ProgressEvent {
	e.ErrorMessage = errorMessage
	return e
}

// ToJSON marshals the event to JSON bytes
func (e *ProgressEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// FromJSON unmarshals a ProgressEvent from JSON bytes
func FromJSON(data []byte) (*ProgressEvent, error) {
	var event ProgressEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal progress event: %w", err)
	}
	return &event, nil
}

// SubjectForEvent constructs the NATS subject for a given incident and event type.
// Format: triage.<incident-id>.<event-type>
// Examples:
//   - triage.inc-abc123.run.started
//   - triage.inc-abc123.executing
//   - triage.inc-abc123.error
func SubjectForEvent(incidentID string, eventType EventType) string {
	return fmt.Sprintf("triage.%s.%s", incidentID, eventType)
}

// SubjectWildcard returns the wildcard subject for subscribing to all triage events
// Returns: "triage.>"
func SubjectWildcard() string {
	return "triage.>"
}

// SubjectForIncident returns the subject pattern for all events for a specific incident
// Format: triage.<incident-id>.*
func SubjectForIncident(incidentID string) string {
	return fmt.Sprintf("triage.%s.*", incidentID)
}
