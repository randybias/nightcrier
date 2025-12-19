package events

// FaultEvent represents a fault event received from kubernetes-mcp-server
// Supports BOTH subscription modes:
// - "faults" mode: uses nested Event struct with InvolvedObject
// - "resource-faults" mode: uses flat Resource struct with context/faultType/severity
type FaultEvent struct {
	// IncidentID is set by the runner when processing begins (not from MCP)
	IncidentID     string         `json:"incidentId,omitempty"`
	SubscriptionID string         `json:"subscriptionId"`
	Cluster        string         `json:"cluster"`
	Event          *EventData     `json:"event,omitempty"`          // Used by "faults" mode (pointer so omitempty works)
	Logs           []ContainerLog `json:"logs,omitempty"`           // Used by "faults" mode
	Resource       *ResourceInfo  `json:"resource,omitempty"`       // Used by "resource-faults" mode
	Context        string         `json:"context,omitempty"`        // Used by "resource-faults" mode
	FaultType      string         `json:"faultType,omitempty"`      // Used by "resource-faults" mode
	Severity       string         `json:"severity,omitempty"`       // Used by "resource-faults" mode
	Timestamp      string         `json:"timestamp,omitempty"`      // Used by "resource-faults" mode
}

// ResourceInfo represents the Kubernetes resource in resource-faults mode
type ResourceInfo struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace,omitempty"`
}

// EventData contains the core event information (faults mode)
type EventData struct {
	Namespace      string            `json:"namespace"`
	Timestamp      string            `json:"timestamp"`
	Type           string            `json:"type"`
	Reason         string            `json:"reason"`
	Message        string            `json:"message"`
	Labels         map[string]string `json:"labels,omitempty"`
	InvolvedObject *InvolvedObject   `json:"involvedObject"`
	Count          int32             `json:"count"`
}

// InvolvedObject represents the Kubernetes object involved in the event (faults mode)
type InvolvedObject struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace,omitempty"`
	UID        string `json:"uid,omitempty"`
}

// ContainerLog represents captured container logs
type ContainerLog struct {
	ContainerName string `json:"containerName"`
	Log           string `json:"log"`
	HasPanic      bool   `json:"hasPanic,omitempty"`
}

// Helper methods for convenient access
// These methods work with both "faults" and "resource-faults" modes

// GetResourceName returns the involved object name
func (f *FaultEvent) GetResourceName() string {
	// Check resource-faults mode first (flat structure)
	if f.Resource != nil && f.Resource.Name != "" {
		return f.Resource.Name
	}
	// Fall back to faults mode (nested structure)
	if f.Event != nil && f.Event.InvolvedObject != nil {
		return f.Event.InvolvedObject.Name
	}
	return ""
}

// GetResourceKind returns the involved object kind
func (f *FaultEvent) GetResourceKind() string {
	// Check resource-faults mode first (flat structure)
	if f.Resource != nil && f.Resource.Kind != "" {
		return f.Resource.Kind
	}
	// Fall back to faults mode (nested structure)
	if f.Event != nil && f.Event.InvolvedObject != nil {
		return f.Event.InvolvedObject.Kind
	}
	return ""
}

// GetNamespace returns the event namespace
func (f *FaultEvent) GetNamespace() string {
	// Check resource-faults mode first (flat structure)
	if f.Resource != nil && f.Resource.Namespace != "" {
		return f.Resource.Namespace
	}
	// Fall back to faults mode (nested structure)
	if f.Event != nil {
		return f.Event.Namespace
	}
	return ""
}

// GetSeverity returns severity/faultType
func (f *FaultEvent) GetSeverity() string {
	// Check resource-faults mode first (flat structure)
	if f.Severity != "" {
		return f.Severity
	}
	// Fall back to faults mode (nested structure - event type)
	if f.Event != nil {
		return f.Event.Type
	}
	return ""
}

// GetFaultType returns the fault type (resource-faults mode only)
func (f *FaultEvent) GetFaultType() string {
	return f.FaultType
}

// GetContext returns the context message (resource-faults mode only)
func (f *FaultEvent) GetContext() string {
	// Check resource-faults mode first
	if f.Context != "" {
		return f.Context
	}
	// Fall back to faults mode message
	if f.Event != nil {
		return f.Event.Message
	}
	return ""
}

// GetTimestamp returns the event timestamp
func (f *FaultEvent) GetTimestamp() string {
	// Check resource-faults mode first (flat structure)
	if f.Timestamp != "" {
		return f.Timestamp
	}
	// Fall back to faults mode (nested structure)
	if f.Event != nil {
		return f.Event.Timestamp
	}
	return ""
}

// GetReason returns the event reason (faults mode) or fault type (resource-faults mode)
func (f *FaultEvent) GetReason() string {
	// Check resource-faults mode first
	if f.FaultType != "" {
		return f.FaultType
	}
	// Fall back to faults mode
	if f.Event != nil {
		return f.Event.Reason
	}
	return ""
}

// IsResourceFaultsMode returns true if this event came from resource-faults subscription
func (f *FaultEvent) IsResourceFaultsMode() bool {
	return f.Resource != nil
}
