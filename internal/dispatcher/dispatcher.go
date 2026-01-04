// Package dispatcher provides event dispatching with concurrency control.
package dispatcher

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/randybias/nightcrier/internal/config"
	"github.com/randybias/nightcrier/internal/events"
)

// EventHandler is the callback type for processing events.
// It receives the context, the fault event, and the cluster name.
type EventHandler func(ctx context.Context, event *events.FaultEvent, cluster string) error

// ClusterState tracks the state of event processing for a single cluster.
type ClusterState struct {
	mu           sync.Mutex
	running      bool
	queue        *EventQueue
	droppedCount int64 // Number of events dropped while busy (observability)
}

// Dispatcher manages concurrent event processing with per-cluster serialization
// and global concurrency limits.
type Dispatcher struct {
	// Configuration
	eventTTL    time.Duration
	dedupWindow time.Duration

	// Global semaphore (buffered channel) limits total concurrent agents
	globalSem chan struct{}

	// Per-cluster state map (protected by RWMutex)
	clusterStates   map[string]*ClusterState
	clusterStatesMu sync.RWMutex

	// Fault deduplication: tracks seen fault_ids to prevent duplicate processing
	// Key is "cluster:fault_id", value is first-seen timestamp
	seenFaults   map[string]time.Time
	seenFaultsMu sync.RWMutex

	// Cluster queue configuration
	dropEventsWhileBusy bool
	clusterQueueSize    int

	// Shutdown coordination
	closed   atomic.Bool
	inFlight atomic.Int64

	// Context for shutdown
	shutdownCtx    context.Context
	shutdownCancel context.CancelFunc

	// Event handler callback
	handler EventHandler
}

// NewDispatcher creates a new Dispatcher with the given configuration.
func NewDispatcher(cfg *config.Config, handler EventHandler) *Dispatcher {
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())

	// Get dropEventsWhileBusy value (default true if nil)
	dropEvents := true
	if cfg.DropEventsWhileBusy != nil {
		dropEvents = *cfg.DropEventsWhileBusy
	}

	d := &Dispatcher{
		eventTTL:            time.Duration(cfg.EventTTLSeconds) * time.Second,
		dedupWindow:         time.Duration(cfg.DedupWindowSeconds) * time.Second,
		globalSem:           make(chan struct{}, cfg.MaxConcurrentAgents),
		clusterStates:       make(map[string]*ClusterState),
		seenFaults:          make(map[string]time.Time),
		dropEventsWhileBusy: dropEvents,
		clusterQueueSize:    cfg.ClusterFailureEventQueueSize,
		shutdownCtx:         shutdownCtx,
		shutdownCancel:      shutdownCancel,
		handler:             handler,
	}

	// Start cleanup goroutine for expired dedup entries
	if d.dedupWindow > 0 {
		go d.cleanupSeenFaults()
	}

	return d
}

// Dispatch handles an incoming event. This method is non-blocking - it either
// queues the event or launches a goroutine and returns immediately.
func (d *Dispatcher) Dispatch(ctx context.Context, event *events.FaultEvent, cluster string) {
	// Check if dispatcher is closed
	if d.closed.Load() {
		slog.Debug("dispatcher closed, dropping event",
			"fault_id", event.FaultID,
			"cluster", cluster)
		return
	}

	// Check event TTL - use the event's Timestamp field (RFC3339 format)
	if d.isExpired(event) {
		slog.Debug("dropping expired event before dispatch",
			"fault_id", event.FaultID,
			"cluster", cluster,
			"timestamp", event.Timestamp,
			"ttl", d.eventTTL)
		return
	}

	// Check for duplicate fault_id within dedup window
	if d.isDuplicate(event.FaultID, cluster) {
		slog.Info("dropping duplicate event",
			"fault_id", event.FaultID,
			"cluster", cluster,
			"dedup_window", d.dedupWindow)
		return
	}

	// Get or create cluster state
	cs := d.getOrCreateClusterState(cluster)

	cs.mu.Lock()
	if cs.running {
		// Cluster busy - check if we should drop or queue
		if d.dropEventsWhileBusy {
			cs.droppedCount++
			cs.mu.Unlock()
			slog.Info("dropping event, cluster busy with active triage",
				"fault_id", event.FaultID,
				"cluster", cluster,
				"dropped_count", cs.droppedCount,
				"reason", "drop_events_while_busy enabled")
			return
		}
		// Queue the event (drop_events_while_busy is false)
		cs.queue.Enqueue(&QueuedEvent{
			Event:      event,
			EnqueuedAt: time.Now(),
		})
		cs.mu.Unlock()
		slog.Debug("event queued, cluster busy",
			"fault_id", event.FaultID,
			"cluster", cluster,
			"queue_depth", cs.queue.Len())
		return
	}

	// Mark cluster as running and launch execution
	cs.running = true
	cs.mu.Unlock()

	// Launch execution in background
	d.inFlight.Add(1)
	go d.executeWithLock(ctx, event, cluster)
}

// isExpired checks if the event has exceeded its TTL based on the event's Timestamp.
func (d *Dispatcher) isExpired(event *events.FaultEvent) bool {
	if event.Timestamp == "" {
		// If no timestamp, use ReceivedAt if available
		if !event.ReceivedAt.IsZero() {
			return time.Since(event.ReceivedAt) > d.eventTTL
		}
		// No timestamp info available, assume not expired
		return false
	}

	// Parse RFC3339 timestamp
	eventTime, err := time.Parse(time.RFC3339, event.Timestamp)
	if err != nil {
		slog.Warn("failed to parse event timestamp, assuming not expired",
			"fault_id", event.FaultID,
			"timestamp", event.Timestamp,
			"error", err)
		return false
	}

	return time.Since(eventTime) > d.eventTTL
}

// getOrCreateClusterState returns the ClusterState for the given cluster,
// creating one if it doesn't exist.
func (d *Dispatcher) getOrCreateClusterState(cluster string) *ClusterState {
	// Fast path: read lock
	d.clusterStatesMu.RLock()
	cs, exists := d.clusterStates[cluster]
	d.clusterStatesMu.RUnlock()

	if exists {
		return cs
	}

	// Slow path: write lock
	d.clusterStatesMu.Lock()
	defer d.clusterStatesMu.Unlock()

	// Double-check after acquiring write lock
	if cs, exists := d.clusterStates[cluster]; exists {
		return cs
	}

	cs = &ClusterState{
		running: false,
		queue:   NewEventQueue(cluster, d.clusterQueueSize, d.eventTTL),
	}
	d.clusterStates[cluster] = cs
	return cs
}

// executeWithLock executes the event handler with proper semaphore and lock management.
func (d *Dispatcher) executeWithLock(ctx context.Context, event *events.FaultEvent, cluster string) {
	defer d.onComplete(ctx, cluster)
	defer d.inFlight.Add(-1)

	// Acquire global semaphore slot (blocking)
	if err := d.acquireSlot(ctx); err != nil {
		slog.Error("failed to acquire global semaphore slot",
			"fault_id", event.FaultID,
			"cluster", cluster,
			"error", err)
		return
	}
	defer d.releaseSlot()

	// Log agent start
	slog.Info("agent started",
		"fault_id", event.FaultID,
		"cluster", cluster,
		"slot_usage", len(d.globalSem),
		"max_slots", cap(d.globalSem))

	startTime := time.Now()

	// Execute the handler
	if d.handler != nil {
		if err := d.handler(ctx, event, cluster); err != nil {
			slog.Error("agent execution failed",
				"fault_id", event.FaultID,
				"cluster", cluster,
				"duration", time.Since(startTime),
				"error", err)
			return
		}
	}

	slog.Info("agent completed",
		"fault_id", event.FaultID,
		"cluster", cluster,
		"duration", time.Since(startTime))
}

// acquireSlot acquires a slot from the global semaphore.
// It blocks until a slot is available or the context is cancelled.
func (d *Dispatcher) acquireSlot(ctx context.Context) error {
	select {
	case d.globalSem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-d.shutdownCtx.Done():
		return d.shutdownCtx.Err()
	}
}

// releaseSlot releases a slot back to the global semaphore.
func (d *Dispatcher) releaseSlot() {
	<-d.globalSem
}

// onComplete is called when event processing completes. It processes the next
// queued event for the cluster or marks the cluster as not running.
func (d *Dispatcher) onComplete(ctx context.Context, cluster string) {
	// Check if dispatcher is shutting down
	if d.closed.Load() {
		// During shutdown, just release the cluster lock without processing more
		d.clusterStatesMu.RLock()
		cs, exists := d.clusterStates[cluster]
		d.clusterStatesMu.RUnlock()

		if exists {
			cs.mu.Lock()
			cs.running = false
			cs.mu.Unlock()
		}
		return
	}

	d.clusterStatesMu.RLock()
	cs, exists := d.clusterStates[cluster]
	d.clusterStatesMu.RUnlock()

	if !exists {
		slog.Error("cluster state not found in onComplete",
			"cluster", cluster)
		return
	}

	cs.mu.Lock()

	// Get next valid (non-expired) event from queue
	next := cs.queue.PopFrontIfValid()
	if next != nil {
		// Launch new goroutine for the next event
		// Keep running=true, just process the next event
		cs.mu.Unlock()

		d.inFlight.Add(1)
		go d.executeWithLock(ctx, next.Event, cluster)
		return
	}

	// No more events, mark cluster as not running
	cs.running = false
	cs.mu.Unlock()
}

// Shutdown gracefully shuts down the dispatcher.
// It stops accepting new events and waits for in-flight agents to complete
// up to the context deadline.
func (d *Dispatcher) Shutdown(ctx context.Context) error {
	// Set closed flag to stop accepting new dispatches
	d.closed.Store(true)

	// Cancel the shutdown context to unblock any waiting acquireSlot calls
	d.shutdownCancel()

	slog.Info("dispatcher shutdown initiated",
		"in_flight", d.inFlight.Load())

	// Wait for all in-flight agents with timeout
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Timeout reached, log remaining state and exit
			remaining := d.inFlight.Load()
			slog.Warn("shutdown timeout, forcing exit",
				"in_flight", remaining)
			d.logQueueContents()
			return ctx.Err()
		case <-ticker.C:
			if d.inFlight.Load() == 0 {
				slog.Info("all agents completed, shutdown clean")
				d.logQueueContents()
				return nil
			}
		}
	}
}

// logQueueContents logs the contents of all cluster queues.
// Called during shutdown for observability.
func (d *Dispatcher) logQueueContents() {
	d.clusterStatesMu.RLock()
	defer d.clusterStatesMu.RUnlock()

	for cluster, cs := range d.clusterStates {
		queueLen := cs.queue.Len()
		if queueLen > 0 {
			cs.mu.Lock()
			running := cs.running
			cs.mu.Unlock()
			slog.Info("cluster queue contents at shutdown",
				"cluster", cluster,
				"queue_depth", queueLen,
				"running", running)
		}
	}
}

// InFlightCount returns the number of currently executing agents.
// Useful for monitoring and testing.
func (d *Dispatcher) InFlightCount() int64 {
	return d.inFlight.Load()
}

// IsClosed returns true if the dispatcher has been closed.
func (d *Dispatcher) IsClosed() bool {
	return d.closed.Load()
}

// QueueDepth returns the queue depth for a specific cluster.
// Returns 0 if the cluster doesn't exist.
func (d *Dispatcher) QueueDepth(cluster string) int {
	d.clusterStatesMu.RLock()
	cs, exists := d.clusterStates[cluster]
	d.clusterStatesMu.RUnlock()

	if !exists {
		return 0
	}

	return cs.queue.Len()
}

// isDuplicate checks if a fault_id has been seen within the dedup window.
// If not seen, it records the fault_id and returns false.
// If seen within the window, it returns true (duplicate).
func (d *Dispatcher) isDuplicate(faultID, cluster string) bool {
	// Skip dedup if window is 0 (disabled)
	if d.dedupWindow == 0 {
		return false
	}

	// Use cluster:fault_id as key to allow same fault on different clusters
	key := cluster + ":" + faultID

	d.seenFaultsMu.Lock()
	defer d.seenFaultsMu.Unlock()

	if firstSeen, exists := d.seenFaults[key]; exists {
		// Check if still within dedup window
		if time.Since(firstSeen) < d.dedupWindow {
			return true // Duplicate
		}
		// Expired, update timestamp
		d.seenFaults[key] = time.Now()
		return false
	}

	// Not seen before, record it
	d.seenFaults[key] = time.Now()
	return false
}

// cleanupSeenFaults periodically removes expired entries from the seenFaults map.
// Runs every dedupWindow/2 or every minute, whichever is shorter.
func (d *Dispatcher) cleanupSeenFaults() {
	// Cleanup interval: half the dedup window, min 30s, max 1m
	interval := d.dedupWindow / 2
	if interval < 30*time.Second {
		interval = 30 * time.Second
	}
	if interval > time.Minute {
		interval = time.Minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-d.shutdownCtx.Done():
			return
		case <-ticker.C:
			d.seenFaultsMu.Lock()
			now := time.Now()
			for key, firstSeen := range d.seenFaults {
				if now.Sub(firstSeen) > d.dedupWindow {
					delete(d.seenFaults, key)
				}
			}
			d.seenFaultsMu.Unlock()
		}
	}
}

// TotalQueueDepth returns the total queue depth across all clusters.
func (d *Dispatcher) TotalQueueDepth() int {
	d.clusterStatesMu.RLock()
	defer d.clusterStatesMu.RUnlock()

	total := 0
	for _, cs := range d.clusterStates {
		total += cs.queue.Len()
	}
	return total
}
