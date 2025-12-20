package events

// ClusterEvent wraps a FaultEvent with cluster context metadata.
// This allows the event processing system to know which cluster
// generated the event and what credentials to use for triage.
type ClusterEvent struct {
	// ClusterName identifies which cluster generated this event
	ClusterName string

	// Kubeconfig is the path to the kubeconfig file for cluster access
	Kubeconfig string

	// Labels are arbitrary key-value pairs from cluster configuration
	Labels map[string]string

	// Event is the underlying fault event from the MCP server
	Event *FaultEvent
}
