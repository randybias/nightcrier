package events

import "time"

// FaultEvent represents a fault event received from the SSE stream
type FaultEvent struct {
	ClusterID    string    `json:"cluster_id"`
	Namespace    string    `json:"namespace"`
	ResourceType string    `json:"resource_type"`
	ResourceName string    `json:"resource_name"`
	Severity     string    `json:"severity"`
	Message      string    `json:"message"`
	Timestamp    time.Time `json:"timestamp"`
	Logs         string    `json:"logs,omitempty"`
}
