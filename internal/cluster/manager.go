package cluster

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"sync"
	"time"
)

// ConnectionManager orchestrates multiple cluster connections.
// It manages the lifecycle of all MCP connections, fans in events from
// all clusters into a single channel, and provides health monitoring.
//
// Phase 2 implementation: No permission validation yet (that's Phase 3).
//
// To avoid circular imports with the events and config packages, this manager:
// 1. Accepts pre-created event clients via SetClusterClient()
// 2. Returns events as interface{} (caller casts to *events.ClusterEvent)
type ConnectionManager struct {
	// connections maps cluster name to its connection instance
	connections map[string]*ClusterConnection

	// eventChan is the global fan-in channel for all cluster events
	// Element type is interface{} to avoid circular import
	// (actual type will be *events.ClusterEvent)
	eventChan chan interface{}

	// transport is the shared HTTP transport used by all MCP clients
	transport *http.Transport

	// Global configuration values
	subscribeMode              string
	globalQueueSize            int
	queueOverflowPolicy        string
	sseReconnectInitialBackoff int // seconds

	// mu protects access to the connections map
	mu sync.RWMutex

	// wg tracks running connection goroutines for graceful shutdown
	wg sync.WaitGroup

	// ctx is the manager's context for shutdown coordination
	ctx context.Context

	// cancel cancels the manager's context
	cancel context.CancelFunc
}

// ManagerConfig holds configuration values for the ConnectionManager.
// This struct exists to avoid directly passing *config.Config and creating
// a circular dependency (config -> cluster -> events -> config).
type ManagerConfig struct {
	Clusters                   []ClusterConfig
	SubscribeMode              string
	GlobalQueueSize            int
	QueueOverflowPolicy        string
	SSEReconnectInitialBackoff int // seconds
}

// NewConnectionManager creates a new ConnectionManager with the given configuration.
// It initializes the shared HTTP transport and creates connection instances for
// each cluster.
//
// After creation, the caller must:
// 1. Call SetClusterClient() for each cluster to provide the events.Client
// 2. Call Start() to begin event processing
//
// The manager is created in a stopped state.
//
// Design reference: lines 239-263 (HTTP transport pooling settings)
//
// Parameters:
//   - cfg: Manager configuration containing cluster definitions
//
// Returns a new ConnectionManager ready to have clients set and be started.
func NewConnectionManager(cfg *ManagerConfig) (*ConnectionManager, error) {
	// Create shared HTTP transport with connection pooling
	// Design reference: lines 240-256
	transport := &http.Transport{
		// Connection pool settings
		MaxIdleConns:        200, // Total idle connections across all hosts
		MaxIdleConnsPerHost: 2,   // Idle connections per MCP server
		MaxConnsPerHost:     10,  // Max connections per MCP server
		IdleConnTimeout:     90 * time.Second,

		// Timeouts
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,

		// Keep-alive for persistent connections
		DisableKeepAlives: false,

		// Force HTTP/2 for multiplexing (if server supports)
		ForceAttemptHTTP2: true,
	}

	// Create manager instance
	ctx, cancel := context.WithCancel(context.Background())
	mgr := &ConnectionManager{
		connections:                make(map[string]*ClusterConnection),
		eventChan:                  make(chan interface{}, cfg.GlobalQueueSize),
		transport:                  transport,
		subscribeMode:              cfg.SubscribeMode,
		globalQueueSize:            cfg.GlobalQueueSize,
		queueOverflowPolicy:        cfg.QueueOverflowPolicy,
		sseReconnectInitialBackoff: cfg.SSEReconnectInitialBackoff,
		ctx:                        ctx,
		cancel:                     cancel,
	}

	// Create connections for each cluster
	for i := range cfg.Clusters {
		cluster := &cfg.Clusters[i]

		// Create cluster connection (no client yet)
		conn := NewClusterConnection(cluster)

		// Store in connections map
		mgr.connections[cluster.Name] = conn

		slog.Info("cluster connection created",
			"cluster", cluster.Name,
			"endpoint", cluster.MCP.Endpoint,
			"triage_enabled", cluster.Triage.Enabled)
	}

	return mgr, nil
}

// SetClusterClient sets the event client for a specific cluster.
// This must be called for each cluster before calling Start().
//
// The client parameter should be an *events.Client instance that implements
// the Subscribe(ctx) method returning a channel of events.
//
// Returns an error if the cluster is not found.
func (cm *ConnectionManager) SetClusterClient(clusterName string, client interface{}) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	conn, ok := cm.connections[clusterName]
	if !ok {
		return fmt.Errorf("cluster %q not found", clusterName)
	}

	conn.SetClient(client)
	return nil
}

// Initialize validates cluster permissions for all clusters with triage enabled.
// This should be called after all clients are set but before Start().
//
// For each cluster with triage.enabled=true:
//   - Validates kubeconfig file exists
//   - Runs kubectl auth can-i checks
//   - Sets permissions on the ClusterConnection
//   - Logs warnings if minimum permissions not met
//
// Clusters with triage.enabled=false are skipped.
//
// Phase 3: Added for permission validation (design.md lines 269-304)
//
// Parameters:
//   - ctx: Context for kubectl command execution (with timeout)
//
// Returns error if validation fails for any cluster with triage enabled.
func (cm *ConnectionManager) Initialize(ctx context.Context) error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	slog.Info("initializing connection manager - validating cluster permissions",
		"cluster_count", len(cm.connections))

	for clusterName, conn := range cm.connections {
		clusterConfig := conn.config

		// Skip validation if triage is disabled
		if !clusterConfig.Triage.Enabled {
			slog.Info("triage disabled for cluster",
				"cluster", clusterName,
				"reason", "triage.enabled=false")
			continue
		}

		// Validate permissions
		slog.Info("validating cluster permissions",
			"cluster", clusterName,
			"kubeconfig", clusterConfig.Triage.Kubeconfig)

		perms, err := validateClusterPermissions(ctx, clusterConfig)
		if err != nil {
			return fmt.Errorf("cluster %s: permission validation failed: %w",
				clusterName, err)
		}

		// Set permissions on connection
		conn.SetPermissions(perms)

		// Warn if minimum permissions not met (but don't fail)
		if !perms.MinimumPermissionsMet() {
			slog.Warn("cluster has insufficient permissions for full triage",
				"cluster", clusterName,
				"warnings", perms.Warnings)
		} else {
			slog.Info("cluster permissions validated successfully",
				"cluster", clusterName,
				"minimum_met", true,
				"helm_access", perms.HelmAccessAvailable())
		}
	}

	slog.Info("connection manager initialization complete")
	return nil
}

// Start begins managing all cluster connections and returns a read-only
// channel for receiving cluster events. It spawns a goroutine for each
// cluster connection to subscribe to its MCP server and fan events into
// the global event channel.
//
// The fan-in pattern: each cluster goroutine subscribes to its MCP server
// and wraps incoming FaultEvents in ClusterEvent wrappers that include
// cluster context (name, kubeconfig, labels).
//
// Design reference: lines 384-430 (fan-in architecture)
//
// Parameters:
//   - ctx: Context for controlling connection lifecycle
//
// Returns a read-only channel that emits ClusterEvent instances (as interface{}).
// The caller should type assert each event to *events.ClusterEvent.
func (cm *ConnectionManager) Start(ctx context.Context) <-chan interface{} {
	slog.Info("starting connection manager",
		"cluster_count", len(cm.connections),
		"global_queue_size", cap(cm.eventChan))

	// Start a goroutine for each cluster connection
	for clusterName, conn := range cm.connections {
		cm.wg.Add(1)
		go cm.runConnection(ctx, clusterName, conn)
	}

	return cm.eventChan
}

// runConnection manages the lifecycle of a single cluster connection.
// It subscribes to the MCP server, receives events, and fans them into
// the global event channel. On disconnect, it implements reconnection
// logic with backoff.
//
// This is the core of the fan-in architecture: each connection runs
// independently and pushes ClusterEvent wrappers to the shared channel.
func (cm *ConnectionManager) runConnection(ctx context.Context, clusterName string, conn *ClusterConnection) {
	defer cm.wg.Done()

	// Get cluster config
	clusterConfig := conn.config

	slog.Info("starting cluster connection",
		"cluster", clusterName,
		"endpoint", clusterConfig.MCP.Endpoint)

	// Main connection loop with reconnection
	for {
		select {
		case <-ctx.Done():
			slog.Info("stopping cluster connection", "cluster", clusterName)
			return
		default:
			// Attempt to subscribe to events
			if err := cm.subscribeAndFanIn(ctx, clusterName, conn); err != nil {
				slog.Error("cluster connection failed",
					"cluster", clusterName,
					"error", err)

				// Update connection status
				cm.updateConnectionStatus(conn, StatusFailed, err)

				// Wait before reconnecting (simple backoff)
				// TODO: Implement exponential backoff with jitter in future phase
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Duration(cm.sseReconnectInitialBackoff) * time.Second):
					slog.Info("reconnecting to cluster",
						"cluster", clusterName)
				}
			}
		}
	}
}

// subscribeAndFanIn subscribes to a cluster's MCP server and fans events
// into the global channel. It returns when the subscription ends (either
// due to disconnect or context cancellation).
//
// To avoid circular imports, this uses reflection to call Subscribe() on the
// client and handle the returned channel. The actual types are:
// - client: *events.Client
// - Subscribe returns: (<-chan *events.FaultEvent, error)
func (cm *ConnectionManager) subscribeAndFanIn(ctx context.Context, clusterName string, conn *ClusterConnection) error {
	// Update status to connecting
	cm.updateConnectionStatus(conn, StatusConnecting, nil)

	// Get the event client (stored as interface{})
	eventClient := conn.client
	if eventClient == nil {
		return fmt.Errorf("no event client set for cluster %s", clusterName)
	}

	// Use reflection to call Subscribe(ctx) method
	cm.updateConnectionStatus(conn, StatusSubscribing, nil)

	clientValue := reflect.ValueOf(eventClient)
	subscribeMethod := clientValue.MethodByName("Subscribe")
	if !subscribeMethod.IsValid() {
		return fmt.Errorf("client for cluster %s does not have Subscribe method", clusterName)
	}

	// Call Subscribe(ctx) using reflection
	results := subscribeMethod.Call([]reflect.Value{reflect.ValueOf(ctx)})
	if len(results) != 2 {
		return fmt.Errorf("Subscribe method for cluster %s returned unexpected number of results", clusterName)
	}

	// Check for error (second return value)
	if !results[1].IsNil() {
		err := results[1].Interface().(error)
		return fmt.Errorf("subscribe failed: %w", err)
	}

	// Get the event channel (first return value)
	// The actual type is <-chan *events.FaultEvent
	eventChanValue := results[0]

	// Mark as active
	cm.updateConnectionStatus(conn, StatusActive, nil)
	slog.Info("cluster connection active",
		"cluster", clusterName)

	// Get cluster config for event wrapping
	clusterConfig := conn.config

	// Fan-in events to global channel using reflection to receive from the channel
	// Events come in as *events.FaultEvent, we wrap them in a map structure
	// that matches events.ClusterEvent fields to avoid importing events package
	for {
		// Use reflection to receive from the channel
		chosen, recv, recvOK := reflect.Select([]reflect.SelectCase{
			{Dir: reflect.SelectRecv, Chan: eventChanValue},
			{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ctx.Done())},
		})

		if chosen == 1 {
			// Context cancelled
			return ctx.Err()
		}

		if !recvOK {
			// Channel closed
			cm.updateConnectionStatus(conn, StatusDisconnected, nil)
			return fmt.Errorf("event stream closed")
		}

		// Extract the event (it's interface{} but actually *events.FaultEvent)
		event := recv.Interface()
		// Create ClusterEvent wrapper as a map
		// This matches the structure of events.ClusterEvent:
		//   ClusterName string
		//   Kubeconfig  string
		//   Permissions *ClusterPermissions  (Phase 3: added)
		//   Labels      map[string]string
		//   Event       *FaultEvent
		clusterEvent := map[string]interface{}{
			"ClusterName": clusterConfig.Name,
			"Kubeconfig":  clusterConfig.Triage.Kubeconfig,
			"Permissions": conn.GetPermissions(), // Phase 3: include permissions
			"Labels":      clusterConfig.Labels,
			"Event":       event,
		}

		// Try to send to global channel
		select {
		case cm.eventChan <- clusterEvent:
			// Event sent successfully
			cm.updateLastEvent(conn)

			slog.Debug("event received and forwarded",
				"cluster", clusterName)

		case <-ctx.Done():
			// Context cancelled, stop processing
			return ctx.Err()

		default:
			// Queue full, apply overflow policy
			if cm.queueOverflowPolicy == "drop" {
				slog.Warn("event queue full, dropping event",
					"cluster", clusterName,
					"policy", "drop")
			} else {
				// Reject policy - log and continue (can't block here)
				slog.Warn("event queue full, event rejected",
					"cluster", clusterName,
					"policy", "reject")
			}
		}
	}

	// Event stream closed (likely disconnected)
	cm.updateConnectionStatus(conn, StatusDisconnected, nil)
	return fmt.Errorf("event stream closed")
}

// updateConnectionStatus updates a connection's status and error state.
func (cm *ConnectionManager) updateConnectionStatus(conn *ClusterConnection, status ConnectionStatus, err error) {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	conn.status = status
	conn.lastError = err

	if err != nil {
		conn.retryCount++
	} else if status == StatusActive {
		conn.retryCount = 0
	}
}

// updateLastEvent updates the last event timestamp and increments the event counter for a connection.
func (cm *ConnectionManager) updateLastEvent(conn *ClusterConnection) {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	conn.lastEvent = time.Now()
	conn.eventCount++
}

// Stop gracefully shuts down the connection manager.
// It cancels all connection contexts, waits for goroutines to complete,
// and closes the event channel.
func (cm *ConnectionManager) Stop() {
	slog.Info("stopping connection manager",
		"cluster_count", len(cm.connections))

	// Cancel the manager's context to signal all connections to stop
	cm.cancel()

	// Wait for all connection goroutines to finish
	cm.wg.Wait()

	// Close the event channel
	close(cm.eventChan)

	slog.Info("connection manager stopped")
}

// GetConnectionStatus returns the current status of a specific cluster connection.
// Returns nil if the cluster is not found.
func (cm *ConnectionManager) GetConnectionStatus(clusterName string) *ClusterConnection {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.connections[clusterName]
}

// GetAllConnectionStatuses returns status information for all cluster connections.
// This is useful for health monitoring and debugging.
func (cm *ConnectionManager) GetAllConnectionStatuses() map[string]ConnectionStatus {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	statuses := make(map[string]ConnectionStatus)
	for name, conn := range cm.connections {
		conn.mu.RLock()
		statuses[name] = conn.status
		conn.mu.RUnlock()
	}

	return statuses
}

// GetHealth returns a complete health summary for all cluster connections.
// This method is used by the health monitoring HTTP endpoint to provide
// detailed status information including per-cluster health and aggregate statistics.
//
// The returned summary includes:
//   - Per-cluster health: status, last event time, error messages, event counts,
//     triage configuration, permissions, and labels
//   - Aggregate statistics: total clusters, active connections, unhealthy connections,
//     and triage-enabled count
//
// This method is thread-safe and acquires read locks on both the manager and
// individual connections.
//
// Phase 4: Added for health monitoring endpoint (design.md lines 547-572)
//
// Returns a HealthSummary structure (defined in internal/health package).
// Note: We return interface{} here to avoid importing internal/health and creating
// a circular dependency. The actual return type is *health.HealthSummary.
func (cm *ConnectionManager) GetHealth() interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Import the health package types at runtime via reflection would be complex,
	// so we'll construct a compatible map structure instead
	clusters := make([]map[string]interface{}, 0, len(cm.connections))

	// Counters for summary statistics
	totalCount := len(cm.connections)
	activeCount := 0
	unhealthyCount := 0
	triageEnabledCount := 0

	// Collect health data for each cluster
	for _, conn := range cm.connections {
		conn.mu.RLock()

		// Determine if this cluster is healthy
		isHealthy := conn.status == StatusActive
		if isHealthy {
			activeCount++
		} else if conn.status == StatusFailed || conn.status == StatusDisconnected {
			unhealthyCount++
		}

		// Check if triage is enabled
		triageEnabled := conn.config.Triage.Enabled
		if triageEnabled {
			triageEnabledCount++
		}

		// Build cluster health data
		clusterHealth := map[string]interface{}{
			"name":           conn.config.Name,
			"status":         conn.status,
			"event_count":    conn.eventCount,
			"triage_enabled": triageEnabled,
		}

		// Add optional fields
		if !conn.lastEvent.IsZero() {
			lastEvent := conn.lastEvent
			clusterHealth["last_event"] = &lastEvent
		}

		if conn.lastError != nil {
			clusterHealth["error"] = conn.lastError.Error()
		}

		if conn.permissions != nil {
			clusterHealth["permissions"] = conn.permissions
		}

		if len(conn.config.Labels) > 0 {
			clusterHealth["labels"] = conn.config.Labels
		}

		conn.mu.RUnlock()

		clusters = append(clusters, clusterHealth)
	}

	// Build summary structure
	summary := map[string]interface{}{
		"clusters": clusters,
		"summary": map[string]interface{}{
			"total":          totalCount,
			"active":         activeCount,
			"unhealthy":      unhealthyCount,
			"triage_enabled": triageEnabledCount,
		},
	}

	return summary
}
