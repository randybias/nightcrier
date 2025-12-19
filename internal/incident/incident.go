package incident

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/rbias/kubernetes-mcp-alerts-event-runner/internal/events"
)

// Status constants for incident lifecycle
const (
	StatusPending       = "pending"
	StatusInvestigating = "investigating"
	StatusResolved      = "resolved"
	StatusFailed        = "failed"
	StatusAgentFailed   = "agent_failed"
)

// Incident represents our investigation of a fault
type Incident struct {
	// Identity
	IncidentID string `json:"incidentId"`

	// Lifecycle
	Status      string     `json:"status"`      // pending, investigating, resolved, failed, agent_failed
	CreatedAt   time.Time  `json:"createdAt"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`

	// Result (populated after agent runs)
	ExitCode      *int   `json:"exitCode,omitempty"`
	FailureReason string `json:"failureReason,omitempty"`

	// Context (flattened from triggering event)
	Cluster   string        `json:"cluster"`
	Namespace string        `json:"namespace"`
	Resource  *ResourceInfo `json:"resource"`
	FaultType string        `json:"faultType"`
	Severity  string        `json:"severity"`
	Context   string        `json:"context"`   // Human-readable description
	Timestamp string        `json:"timestamp"` // When fault occurred in K8s

	// Traceability (internal, not for agent)
	TriggeringEventID string `json:"triggeringEventId,omitempty"`
}

// ResourceInfo represents the Kubernetes resource involved in the incident
type ResourceInfo struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace,omitempty"`
}

// NewFromEvent creates a new Incident from a FaultEvent, flattening event data
// Handles both "faults" and "resource-faults" subscription modes
func NewFromEvent(incidentID string, event *events.FaultEvent) *Incident {
	now := time.Now()

	incident := &Incident{
		IncidentID:        incidentID,
		Status:            StatusInvestigating,
		CreatedAt:         now,
		Cluster:           event.Cluster,
		Namespace:         event.GetNamespace(),
		FaultType:         event.GetFaultType(),
		Severity:          event.GetSeverity(),
		Context:           event.GetContext(),
		Timestamp:         event.GetTimestamp(),
		TriggeringEventID: event.SubscriptionID,
	}

	// Flatten resource information from event
	incident.Resource = extractResourceInfo(event)

	return incident
}

// extractResourceInfo extracts ResourceInfo from a FaultEvent
// Handles both "faults" and "resource-faults" modes
func extractResourceInfo(event *events.FaultEvent) *ResourceInfo {
	// Prefer resource-faults mode (flat structure)
	if event.Resource != nil {
		return &ResourceInfo{
			APIVersion: event.Resource.APIVersion,
			Kind:       event.Resource.Kind,
			Name:       event.Resource.Name,
			Namespace:  event.Resource.Namespace,
		}
	}

	// Fall back to faults mode (nested structure)
	if event.Event != nil && event.Event.InvolvedObject != nil {
		obj := event.Event.InvolvedObject
		return &ResourceInfo{
			APIVersion: obj.APIVersion,
			Kind:       obj.Kind,
			Name:       obj.Name,
			Namespace:  obj.Namespace,
		}
	}

	// Minimal resource info if neither mode has valid data
	return &ResourceInfo{
		Kind: "Unknown",
		Name: "Unknown",
	}
}

// WriteToFile writes the incident to a JSON file with proper formatting
func (i *Incident) WriteToFile(path string) error {
	// Marshal incident to indented JSON
	data, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal incident: %w", err)
	}

	// Write to file with 0600 permissions
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write incident file: %w", err)
	}

	return nil
}

// UpdateFromFile reads an existing incident.json and unmarshals it into this incident
func (i *Incident) UpdateFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read incident file: %w", err)
	}

	if err := json.Unmarshal(data, i); err != nil {
		return fmt.Errorf("failed to unmarshal incident: %w", err)
	}

	return nil
}

// MarkCompleted updates the incident status and completion information
func (i *Incident) MarkCompleted(exitCode int, err error) {
	now := time.Now()
	i.CompletedAt = &now
	i.ExitCode = &exitCode

	if err != nil {
		i.Status = StatusFailed
		i.FailureReason = err.Error()
	} else if exitCode == 0 {
		i.Status = StatusResolved
	} else {
		i.Status = StatusFailed
		i.FailureReason = fmt.Sprintf("agent exited with code %d", exitCode)
	}
}
