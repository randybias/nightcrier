package cluster

import (
	"sync"
	"time"
)

// ConnectionStatus represents the current state of a cluster connection.
type ConnectionStatus string

const (
	// StatusDisconnected indicates the connection is not established.
	StatusDisconnected ConnectionStatus = "disconnected"

	// StatusConnecting indicates the connection is being established.
	StatusConnecting ConnectionStatus = "connecting"

	// StatusConnected indicates the MCP session is established but not yet subscribed.
	StatusConnected ConnectionStatus = "connected"

	// StatusSubscribing indicates the connection is subscribing to events.
	StatusSubscribing ConnectionStatus = "subscribing"

	// StatusActive indicates the connection is fully operational and receiving events.
	StatusActive ConnectionStatus = "active"

	// StatusFailed indicates the connection has failed and may need reconnection.
	StatusFailed ConnectionStatus = "failed"
)

// ClusterConnection manages the lifecycle of a single MCP connection.
// It tracks connection state, handles reconnection logic, and routes events
// from the MCP server to the global event channel.
//
// Phase 3 addition: Added permissions field for cluster access validation.
// Note: The client field will be set by the connection manager after construction
// to avoid circular dependencies (events package imports config, which imports cluster).
type ClusterConnection struct {
	// config holds the cluster's configuration (endpoint, triage settings, labels).
	config *ClusterConfig

	// client is the MCP client instance for this cluster connection.
	// Type is interface{} to avoid circular import (actual type is *events.Client).
	// The connection manager will set this after construction.
	client interface{}

	// permissions holds validated cluster access permissions (Phase 3).
	// This is set during connection manager initialization if triage is enabled.
	// It is included in ClusterEvent so the agent knows what it can access.
	permissions *ClusterPermissions

	// status tracks the current connection state.
	status ConnectionStatus

	// lastEvent records when the most recent event was received.
	lastEvent time.Time

	// eventCount tracks the total number of events received from this cluster.
	eventCount int64

	// lastError stores the most recent connection error for diagnostics.
	lastError error

	// retryCount tracks the number of consecutive reconnection attempts.
	retryCount int

	// mu protects concurrent access to connection state.
	mu sync.RWMutex
}

// NewClusterConnection creates a new ClusterConnection with the given configuration.
// The connection is created in the StatusDisconnected state and must be started
// explicitly via the connection manager.
//
// The client field should be set via SetClient() after construction, once the
// events.Client has been created by the manager.
//
// Parameters:
//   - config: The cluster configuration (must not be nil)
//
// Returns a new ClusterConnection ready to be started.
func NewClusterConnection(config *ClusterConfig) *ClusterConnection {
	return &ClusterConnection{
		config: config,
		status: StatusDisconnected,
	}
}

// SetClient sets the MCP client for this connection.
// This is called by the connection manager after creating the events.Client.
// The client parameter should be of type *events.Client.
func (c *ClusterConnection) SetClient(client interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.client = client
}

// SetPermissions sets the validated cluster permissions for this connection.
// This is called by the connection manager during initialization after
// validating cluster access permissions with kubectl.
//
// Phase 3: Added for cluster access validation.
func (c *ClusterConnection) SetPermissions(perms *ClusterPermissions) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.permissions = perms
}

// GetPermissions returns a copy of the cluster permissions.
// Returns nil if permissions have not been validated yet.
//
// Phase 3: Added for cluster access validation.
func (c *ClusterConnection) GetPermissions() *ClusterPermissions {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.permissions
}
