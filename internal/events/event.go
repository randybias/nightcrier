package events

// FaultEvent represents a fault event received from kubernetes-mcp-server
// This matches the structure in pkg/events/faults.go of kubernetes-mcp-server
type FaultEvent struct {
	SubscriptionID string         `json:"subscriptionId"`
	Cluster        string         `json:"cluster"`
	Event          EventData      `json:"event"`
	Logs           []ContainerLog `json:"logs,omitempty"`
}

// EventData contains the core event information
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

// InvolvedObject represents the Kubernetes object involved in the event
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

// GetResourceName returns the involved object name
func (f *FaultEvent) GetResourceName() string {
	if f.Event.InvolvedObject != nil {
		return f.Event.InvolvedObject.Name
	}
	return ""
}

// GetResourceKind returns the involved object kind
func (f *FaultEvent) GetResourceKind() string {
	if f.Event.InvolvedObject != nil {
		return f.Event.InvolvedObject.Kind
	}
	return ""
}

// GetNamespace returns the event namespace
func (f *FaultEvent) GetNamespace() string {
	return f.Event.Namespace
}

// GetSeverity returns the event type (Normal/Warning) as severity
func (f *FaultEvent) GetSeverity() string {
	return f.Event.Type
}
