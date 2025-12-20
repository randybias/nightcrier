package events

import "time"

// FaultEvent represents a fault event received from kubernetes-mcp-server
type FaultEvent struct {
	// From kubernetes-mcp-server - stable identifier for the fault condition
	FaultID    string    `json:"faultId"`  // Stable identifier from kubernetes-mcp-server (hex hash, not UUID)
	ReceivedAt time.Time `json:"-"`        // Time fault was received locally (not serialized)

	// From kubernetes-mcp-server
	SubscriptionID string        `json:"subscriptionId"`
	Cluster        string        `json:"cluster"`
	Resource       *ResourceInfo `json:"resource"`
	FaultType      string        `json:"faultType"`
	Severity       string        `json:"severity"`
	Context        string        `json:"context"`             // Human-readable fault description
	Timestamp      string        `json:"timestamp"`           // When fault occurred in K8s
}

// ResourceInfo represents the Kubernetes resource involved in the fault
type ResourceInfo struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace,omitempty"`
	UID        string `json:"uid,omitempty"` // Kubernetes resource UID (used in FaultID hash upstream)
}

// Helper methods for convenient access

// GetResourceName returns the resource name
func (f *FaultEvent) GetResourceName() string {
	if f.Resource != nil {
		return f.Resource.Name
	}
	return ""
}

// GetResourceKind returns the resource kind
func (f *FaultEvent) GetResourceKind() string {
	if f.Resource != nil {
		return f.Resource.Kind
	}
	return ""
}

// GetNamespace returns the resource namespace
func (f *FaultEvent) GetNamespace() string {
	if f.Resource != nil {
		return f.Resource.Namespace
	}
	return ""
}

// GetSeverity returns the fault severity
func (f *FaultEvent) GetSeverity() string {
	return f.Severity
}

// GetFaultType returns the fault type
func (f *FaultEvent) GetFaultType() string {
	return f.FaultType
}

// GetContext returns the fault context description
func (f *FaultEvent) GetContext() string {
	return f.Context
}

// GetTimestamp returns the fault timestamp
func (f *FaultEvent) GetTimestamp() string {
	return f.Timestamp
}

// GetReason returns the fault type (alias for GetFaultType)
func (f *FaultEvent) GetReason() string {
	return f.FaultType
}
