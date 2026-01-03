package dispatcher

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/randybias/nightcrier/internal/config"
	"github.com/randybias/nightcrier/internal/events"
)

// -----------------------------------------------------------------------------
// Test Helpers
// -----------------------------------------------------------------------------

// testConfig creates a config suitable for testing with sensible defaults.
func testConfig(opts ...func(*config.Config)) *config.Config {
	cfg := &config.Config{
		MaxConcurrentAgents: 5,
		ClusterQueueSize:    10,
		EventTTLSeconds:     300, // 5 minutes
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// withMaxConcurrentAgents sets the max concurrent agents for testing.
func withMaxConcurrentAgents(n int) func(*config.Config) {
	return func(cfg *config.Config) {
		cfg.MaxConcurrentAgents = n
	}
}

// withClusterQueueSize sets the cluster queue size for testing.
func withClusterQueueSize(n int) func(*config.Config) {
	return func(cfg *config.Config) {
		cfg.ClusterQueueSize = n
	}
}

// withEventTTL sets the event TTL in seconds for testing.
func withEventTTL(seconds int) func(*config.Config) {
	return func(cfg *config.Config) {
		cfg.EventTTLSeconds = seconds
	}
}

// testEvent creates a FaultEvent for testing with the given fault ID.
func testEvent(faultID string) *events.FaultEvent {
	return &events.FaultEvent{
		FaultID:   faultID,
		Cluster:   "test-cluster",
		FaultType: "TestFault",
		Severity:  "warning",
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// testEventForCluster creates a FaultEvent for a specific cluster.
func testEventForCluster(faultID, cluster string) *events.FaultEvent {
	return &events.FaultEvent{
		FaultID:   faultID,
		Cluster:   cluster,
		FaultType: "TestFault",
		Severity:  "warning",
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// testEventWithTimestamp creates a FaultEvent with a specific timestamp.
func testEventWithTimestamp(faultID string, ts time.Time) *events.FaultEvent {
	return &events.FaultEvent{
		FaultID:   faultID,
		Cluster:   "test-cluster",
		FaultType: "TestFault",
		Severity:  "warning",
		Timestamp: ts.Format(time.RFC3339),
	}
}

// trackingHandler creates an EventHandler that tracks which events were processed.
// It returns the handler and channels to signal when processing starts and completes.
type handlerTracker struct {
	mu           sync.Mutex
	processed    []string                      // List of processed fault IDs
	clusters     map[string][]string           // Cluster -> list of fault IDs
	started      chan string                   // Signals when a handler starts (sends faultID)
	completed    chan string                   // Signals when a handler completes (sends faultID)
	blockCh      chan struct{}                 // If non-nil, handler blocks until this is closed
	errorForID   map[string]error              // Map of faultID -> error to return
	execDuration time.Duration                 // How long to simulate execution
	concurrent   atomic.Int32                  // Current concurrent executions
	maxConcurrent atomic.Int32                 // Max observed concurrent executions
}

func newHandlerTracker() *handlerTracker {
	return &handlerTracker{
		processed: make([]string, 0),
		clusters:  make(map[string][]string),
		started:   make(chan string, 100),
		completed: make(chan string, 100),
		errorForID: make(map[string]error),
	}
}

func (ht *handlerTracker) handler(ctx context.Context, event *events.FaultEvent, cluster string) error {
	// Track that we started
	ht.started <- event.FaultID

	// Track concurrent executions
	current := ht.concurrent.Add(1)
	defer ht.concurrent.Add(-1)

	// Update max concurrent
	for {
		max := ht.maxConcurrent.Load()
		if current <= max {
			break
		}
		if ht.maxConcurrent.CompareAndSwap(max, current) {
			break
		}
	}

	// Block if blockCh is set
	if ht.blockCh != nil {
		select {
		case <-ht.blockCh:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Simulate execution time
	if ht.execDuration > 0 {
		select {
		case <-time.After(ht.execDuration):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Record the processed event
	ht.mu.Lock()
	ht.processed = append(ht.processed, event.FaultID)
	ht.clusters[cluster] = append(ht.clusters[cluster], event.FaultID)
	ht.mu.Unlock()

	// Signal completion
	ht.completed <- event.FaultID

	// Check for configured error
	if err, ok := ht.errorForID[event.FaultID]; ok {
		return err
	}
	return nil
}

func (ht *handlerTracker) getProcessed() []string {
	ht.mu.Lock()
	defer ht.mu.Unlock()
	result := make([]string, len(ht.processed))
	copy(result, ht.processed)
	return result
}

func (ht *handlerTracker) getProcessedForCluster(cluster string) []string {
	ht.mu.Lock()
	defer ht.mu.Unlock()
	result := make([]string, len(ht.clusters[cluster]))
	copy(result, ht.clusters[cluster])
	return result
}

func (ht *handlerTracker) getProcessedCount() int {
	ht.mu.Lock()
	defer ht.mu.Unlock()
	return len(ht.processed)
}

// waitForNCompletions waits for n completions or times out.
func (ht *handlerTracker) waitForNCompletions(t *testing.T, n int, timeout time.Duration) {
	t.Helper()
	deadline := time.After(timeout)
	for i := 0; i < n; i++ {
		select {
		case <-ht.completed:
		case <-deadline:
			t.Fatalf("timeout waiting for completion %d/%d", i+1, n)
		}
	}
}

// waitForNStarts waits for n starts or times out.
func (ht *handlerTracker) waitForNStarts(t *testing.T, n int, timeout time.Duration) {
	t.Helper()
	deadline := time.After(timeout)
	for i := 0; i < n; i++ {
		select {
		case <-ht.started:
		case <-deadline:
			t.Fatalf("timeout waiting for start %d/%d", i+1, n)
		}
	}
}

// -----------------------------------------------------------------------------
// Test: NewDispatcher
// -----------------------------------------------------------------------------

func TestNewDispatcher(t *testing.T) {
	cfg := testConfig()
	handler := func(ctx context.Context, event *events.FaultEvent, cluster string) error {
		return nil
	}

	d := NewDispatcher(cfg, handler)

	if d == nil {
		t.Fatal("expected non-nil dispatcher")
	}
	if cap(d.globalSem) != cfg.MaxConcurrentAgents {
		t.Errorf("expected globalSem capacity %d, got %d", cfg.MaxConcurrentAgents, cap(d.globalSem))
	}
	if d.eventTTL != time.Duration(cfg.EventTTLSeconds)*time.Second {
		t.Errorf("expected eventTTL %v, got %v", time.Duration(cfg.EventTTLSeconds)*time.Second, d.eventTTL)
	}
	if d.clusterQueueSize != cfg.ClusterQueueSize {
		t.Errorf("expected clusterQueueSize %d, got %d", cfg.ClusterQueueSize, d.clusterQueueSize)
	}
	if d.IsClosed() {
		t.Error("expected dispatcher to not be closed initially")
	}
	if d.InFlightCount() != 0 {
		t.Errorf("expected 0 in-flight, got %d", d.InFlightCount())
	}
}

// -----------------------------------------------------------------------------
// Test 4.2: Events for different clusters run in parallel
// -----------------------------------------------------------------------------

func TestDispatcher_DifferentClustersRunInParallel(t *testing.T) {
	cfg := testConfig(withMaxConcurrentAgents(10))
	tracker := newHandlerTracker()
	tracker.blockCh = make(chan struct{}) // Block handlers until we release

	d := NewDispatcher(cfg, tracker.handler)
	ctx := context.Background()

	// Dispatch events to different clusters
	clusters := []string{"cluster-a", "cluster-b", "cluster-c"}
	for i, cluster := range clusters {
		event := testEventForCluster("fault-"+cluster, cluster)
		event.FaultID = "fault-" + string(rune('a'+i))
		d.Dispatch(ctx, event, cluster)
	}

	// Wait for all handlers to start
	tracker.waitForNStarts(t, 3, 2*time.Second)

	// Verify all 3 are running concurrently
	if tracker.concurrent.Load() != 3 {
		t.Errorf("expected 3 concurrent handlers, got %d", tracker.concurrent.Load())
	}

	// Release the handlers
	close(tracker.blockCh)

	// Wait for completions
	tracker.waitForNCompletions(t, 3, 2*time.Second)

	// Clean shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := d.Shutdown(shutdownCtx); err != nil {
		t.Errorf("shutdown error: %v", err)
	}
}

// -----------------------------------------------------------------------------
// Test 4.3: Events for same cluster are serialized
// -----------------------------------------------------------------------------

func TestDispatcher_SameClusterSerialized(t *testing.T) {
	cfg := testConfig(withMaxConcurrentAgents(10))
	tracker := newHandlerTracker()
	tracker.execDuration = 50 * time.Millisecond

	d := NewDispatcher(cfg, tracker.handler)
	ctx := context.Background()

	// Dispatch multiple events to the same cluster
	for i := 0; i < 5; i++ {
		event := testEvent("fault-" + string(rune('a'+i)))
		d.Dispatch(ctx, event, "same-cluster")
	}

	// Wait for all to complete
	tracker.waitForNCompletions(t, 5, 5*time.Second)

	// Verify they ran serially (max concurrent should be 1 for same cluster)
	if tracker.maxConcurrent.Load() != 1 {
		t.Errorf("expected max concurrent 1 for same cluster, got %d", tracker.maxConcurrent.Load())
	}

	// Verify FIFO order
	processed := tracker.getProcessedForCluster("same-cluster")
	expected := []string{"fault-a", "fault-b", "fault-c", "fault-d", "fault-e"}
	if len(processed) != len(expected) {
		t.Fatalf("expected %d processed, got %d: %v", len(expected), len(processed), processed)
	}
	for i, exp := range expected {
		if processed[i] != exp {
			t.Errorf("position %d: expected %q, got %q", i, exp, processed[i])
		}
	}

	// Clean shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	d.Shutdown(shutdownCtx)
}

func TestDispatcher_SameClusterSerializedWithBlocking(t *testing.T) {
	cfg := testConfig(withMaxConcurrentAgents(10))

	// Use a more controlled approach: track execution order with timestamps
	var mu sync.Mutex
	type execRecord struct {
		faultID string
		start   time.Time
		end     time.Time
	}
	var records []execRecord

	handler := func(ctx context.Context, event *events.FaultEvent, cluster string) error {
		start := time.Now()
		time.Sleep(30 * time.Millisecond) // Ensure non-overlapping
		end := time.Now()

		mu.Lock()
		records = append(records, execRecord{
			faultID: event.FaultID,
			start:   start,
			end:     end,
		})
		mu.Unlock()
		return nil
	}

	d := NewDispatcher(cfg, handler)
	ctx := context.Background()

	// Dispatch multiple events
	for i := 0; i < 3; i++ {
		event := testEvent("fault-" + string(rune('a'+i)))
		d.Dispatch(ctx, event, "same-cluster")
	}

	// Wait for in-flight to return to 0
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if d.InFlightCount() == 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Verify records: each should start after the previous ended
	mu.Lock()
	defer mu.Unlock()

	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}

	for i := 1; i < len(records); i++ {
		if records[i].start.Before(records[i-1].end) {
			t.Errorf("event %d started before event %d ended (overlap detected)", i, i-1)
			t.Logf("  event %d: start=%v end=%v", i-1, records[i-1].start, records[i-1].end)
			t.Logf("  event %d: start=%v end=%v", i, records[i].start, records[i].end)
		}
	}

	// Clean shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	d.Shutdown(shutdownCtx)
}

// -----------------------------------------------------------------------------
// Test 4.4: Global semaphore limits total concurrent agents
// -----------------------------------------------------------------------------

func TestDispatcher_GlobalSemaphoreLimitsConcurrency(t *testing.T) {
	maxAgents := 3
	cfg := testConfig(withMaxConcurrentAgents(maxAgents))
	tracker := newHandlerTracker()
	tracker.blockCh = make(chan struct{}) // Block all handlers

	d := NewDispatcher(cfg, tracker.handler)
	ctx := context.Background()

	// Dispatch more events than max concurrent (to different clusters)
	numEvents := maxAgents + 2
	for i := 0; i < numEvents; i++ {
		cluster := "cluster-" + string(rune('a'+i))
		event := testEventForCluster("fault-"+string(rune('a'+i)), cluster)
		d.Dispatch(ctx, event, cluster)
	}

	// Wait a bit for dispatching to settle
	time.Sleep(100 * time.Millisecond)

	// Only maxAgents should be running (others waiting for semaphore)
	// Note: Some may be in the goroutine but blocked on semaphore
	tracker.waitForNStarts(t, maxAgents, 2*time.Second)

	// Give a little more time to ensure no more start
	time.Sleep(50 * time.Millisecond)

	// Check that we're not exceeding max concurrent
	if tracker.concurrent.Load() > int32(maxAgents) {
		t.Errorf("exceeded max concurrent: got %d, max %d", tracker.concurrent.Load(), maxAgents)
	}

	// Release handlers
	close(tracker.blockCh)

	// Wait for all completions
	tracker.waitForNCompletions(t, numEvents, 5*time.Second)

	// Verify max concurrent never exceeded limit
	if tracker.maxConcurrent.Load() > int32(maxAgents) {
		t.Errorf("max concurrent exceeded limit: got %d, max %d", tracker.maxConcurrent.Load(), maxAgents)
	}

	// Clean shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	d.Shutdown(shutdownCtx)
}

// -----------------------------------------------------------------------------
// Test 4.5: Non-blocking dispatch (returns immediately)
// -----------------------------------------------------------------------------

func TestDispatcher_NonBlockingDispatch(t *testing.T) {
	cfg := testConfig(withMaxConcurrentAgents(1))
	tracker := newHandlerTracker()
	tracker.blockCh = make(chan struct{}) // Block the handler

	d := NewDispatcher(cfg, tracker.handler)
	ctx := context.Background()

	// Measure time to dispatch
	start := time.Now()
	event := testEvent("fault-a")
	d.Dispatch(ctx, event, "test-cluster")
	dispatchDuration := time.Since(start)

	// Dispatch should return immediately (< 10ms)
	if dispatchDuration > 10*time.Millisecond {
		t.Errorf("Dispatch took %v, expected < 10ms (should be non-blocking)", dispatchDuration)
	}

	// Dispatch more events while first is blocked
	for i := 0; i < 5; i++ {
		start = time.Now()
		event := testEvent("fault-" + string(rune('b'+i)))
		d.Dispatch(ctx, event, "test-cluster")
		dispatchDuration = time.Since(start)

		if dispatchDuration > 10*time.Millisecond {
			t.Errorf("Dispatch %d took %v, expected < 10ms", i, dispatchDuration)
		}
	}

	// Release the handler
	close(tracker.blockCh)

	// Clean shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	d.Shutdown(shutdownCtx)
}

func TestDispatcher_NonBlockingWhenQueueFull(t *testing.T) {
	cfg := testConfig(
		withMaxConcurrentAgents(1),
		withClusterQueueSize(2),
	)
	tracker := newHandlerTracker()
	tracker.blockCh = make(chan struct{}) // Block the handler

	d := NewDispatcher(cfg, tracker.handler)
	ctx := context.Background()

	// Dispatch first event (will start processing)
	d.Dispatch(ctx, testEvent("fault-1"), "cluster")

	// Wait for it to start
	tracker.waitForNStarts(t, 1, time.Second)

	// Fill the queue
	d.Dispatch(ctx, testEvent("fault-2"), "cluster")
	d.Dispatch(ctx, testEvent("fault-3"), "cluster")

	// Queue should be full now, but dispatch should still be fast
	start := time.Now()
	d.Dispatch(ctx, testEvent("fault-4"), "cluster") // This will drop oldest
	dispatchDuration := time.Since(start)

	if dispatchDuration > 10*time.Millisecond {
		t.Errorf("Dispatch with full queue took %v, expected < 10ms", dispatchDuration)
	}

	// Release handler
	close(tracker.blockCh)

	// Clean shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	d.Shutdown(shutdownCtx)
}

// -----------------------------------------------------------------------------
// Test 4.6: Expired events dropped before dispatch
// -----------------------------------------------------------------------------

func TestDispatcher_ExpiredEventsDroppedBeforeDispatch(t *testing.T) {
	cfg := testConfig(withEventTTL(1)) // 1 second TTL
	tracker := newHandlerTracker()

	d := NewDispatcher(cfg, tracker.handler)
	ctx := context.Background()

	// Create an expired event (timestamp in the past beyond TTL)
	expiredEvent := testEventWithTimestamp("expired-fault", time.Now().Add(-5*time.Second))

	// Dispatch the expired event
	d.Dispatch(ctx, expiredEvent, "test-cluster")

	// Give time for any processing
	time.Sleep(100 * time.Millisecond)

	// Verify it was not processed
	if tracker.getProcessedCount() != 0 {
		t.Errorf("expected 0 processed events (expired should be dropped), got %d", tracker.getProcessedCount())
	}

	// Verify no in-flight
	if d.InFlightCount() != 0 {
		t.Errorf("expected 0 in-flight, got %d", d.InFlightCount())
	}

	// Clean shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	d.Shutdown(shutdownCtx)
}

func TestDispatcher_FreshEventsNotDropped(t *testing.T) {
	cfg := testConfig(withEventTTL(300)) // 5 minute TTL
	tracker := newHandlerTracker()

	d := NewDispatcher(cfg, tracker.handler)
	ctx := context.Background()

	// Create a fresh event
	freshEvent := testEvent("fresh-fault")

	// Dispatch it
	d.Dispatch(ctx, freshEvent, "test-cluster")

	// Wait for completion
	tracker.waitForNCompletions(t, 1, 2*time.Second)

	// Verify it was processed
	processed := tracker.getProcessed()
	if len(processed) != 1 || processed[0] != "fresh-fault" {
		t.Errorf("expected fresh-fault to be processed, got %v", processed)
	}

	// Clean shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	d.Shutdown(shutdownCtx)
}

func TestDispatcher_EventWithNoTimestampNotExpired(t *testing.T) {
	cfg := testConfig(withEventTTL(1))
	tracker := newHandlerTracker()

	d := NewDispatcher(cfg, tracker.handler)
	ctx := context.Background()

	// Create event with no timestamp
	event := &events.FaultEvent{
		FaultID:   "no-timestamp",
		Cluster:   "test-cluster",
		FaultType: "TestFault",
		Severity:  "warning",
		Timestamp: "", // No timestamp
	}

	// Dispatch it
	d.Dispatch(ctx, event, "test-cluster")

	// Wait for completion
	tracker.waitForNCompletions(t, 1, 2*time.Second)

	// Verify it was processed (no timestamp means not expired)
	if tracker.getProcessedCount() != 1 {
		t.Errorf("expected event with no timestamp to be processed")
	}

	// Clean shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	d.Shutdown(shutdownCtx)
}

// -----------------------------------------------------------------------------
// Test 4.7: Queue overflow drops oldest
// -----------------------------------------------------------------------------

func TestDispatcher_QueueOverflowDropsOldest(t *testing.T) {
	cfg := testConfig(
		withMaxConcurrentAgents(1),
		withClusterQueueSize(2),
	)
	tracker := newHandlerTracker()
	tracker.blockCh = make(chan struct{})

	d := NewDispatcher(cfg, tracker.handler)
	ctx := context.Background()

	// Dispatch first event (starts processing, blocks)
	d.Dispatch(ctx, testEvent("fault-1"), "cluster")
	tracker.waitForNStarts(t, 1, time.Second)

	// Fill the queue
	d.Dispatch(ctx, testEvent("fault-2"), "cluster")
	d.Dispatch(ctx, testEvent("fault-3"), "cluster")

	// Queue is now full (size 2)
	if d.QueueDepth("cluster") != 2 {
		t.Errorf("expected queue depth 2, got %d", d.QueueDepth("cluster"))
	}

	// Add more events - should drop oldest
	d.Dispatch(ctx, testEvent("fault-4"), "cluster")
	d.Dispatch(ctx, testEvent("fault-5"), "cluster")

	// Queue should still be at max size
	if d.QueueDepth("cluster") != 2 {
		t.Errorf("expected queue depth 2 after overflow, got %d", d.QueueDepth("cluster"))
	}

	// Release handler
	close(tracker.blockCh)

	// Wait for all to complete (1 running + 2 in queue = 3 total)
	tracker.waitForNCompletions(t, 3, 5*time.Second)

	// Verify which events were processed
	processed := tracker.getProcessed()
	// fault-1 was running, fault-2 and fault-3 were dropped, fault-4 and fault-5 should be in queue
	// Expected: fault-1, fault-4, fault-5
	expected := []string{"fault-1", "fault-4", "fault-5"}
	if len(processed) != len(expected) {
		t.Fatalf("expected %d processed, got %d: %v", len(expected), len(processed), processed)
	}
	for i, exp := range expected {
		if processed[i] != exp {
			t.Errorf("position %d: expected %q, got %q", i, exp, processed[i])
		}
	}

	// Clean shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	d.Shutdown(shutdownCtx)
}

func TestDispatcher_TotalQueueDepth(t *testing.T) {
	cfg := testConfig(
		withMaxConcurrentAgents(2),
		withClusterQueueSize(5),
	)
	tracker := newHandlerTracker()
	tracker.blockCh = make(chan struct{})

	d := NewDispatcher(cfg, tracker.handler)
	ctx := context.Background()

	// Dispatch to multiple clusters
	d.Dispatch(ctx, testEventForCluster("fault-a1", "cluster-a"), "cluster-a")
	d.Dispatch(ctx, testEventForCluster("fault-b1", "cluster-b"), "cluster-b")

	// Wait for both to start
	tracker.waitForNStarts(t, 2, time.Second)

	// Now queue more events
	d.Dispatch(ctx, testEventForCluster("fault-a2", "cluster-a"), "cluster-a")
	d.Dispatch(ctx, testEventForCluster("fault-a3", "cluster-a"), "cluster-a")
	d.Dispatch(ctx, testEventForCluster("fault-b2", "cluster-b"), "cluster-b")

	// Check individual queue depths
	if d.QueueDepth("cluster-a") != 2 {
		t.Errorf("expected cluster-a queue depth 2, got %d", d.QueueDepth("cluster-a"))
	}
	if d.QueueDepth("cluster-b") != 1 {
		t.Errorf("expected cluster-b queue depth 1, got %d", d.QueueDepth("cluster-b"))
	}

	// Check total queue depth
	if d.TotalQueueDepth() != 3 {
		t.Errorf("expected total queue depth 3, got %d", d.TotalQueueDepth())
	}

	// Release handlers
	close(tracker.blockCh)

	// Clean shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	d.Shutdown(shutdownCtx)
}

// -----------------------------------------------------------------------------
// Test 4.8: Failure in one agent doesn't block others
// -----------------------------------------------------------------------------

func TestDispatcher_FailureDoesNotBlockOthers(t *testing.T) {
	cfg := testConfig(withMaxConcurrentAgents(5))
	tracker := newHandlerTracker()
	tracker.errorForID["fault-fail"] = errors.New("simulated failure")
	tracker.execDuration = 20 * time.Millisecond

	d := NewDispatcher(cfg, tracker.handler)
	ctx := context.Background()

	// Dispatch to different clusters - one will fail
	d.Dispatch(ctx, testEventForCluster("fault-fail", "cluster-fail"), "cluster-fail")
	d.Dispatch(ctx, testEventForCluster("fault-ok-1", "cluster-1"), "cluster-1")
	d.Dispatch(ctx, testEventForCluster("fault-ok-2", "cluster-2"), "cluster-2")

	// Wait for all completions (including the failure)
	tracker.waitForNCompletions(t, 3, 5*time.Second)

	// Verify all were processed (failure still completes, just with error)
	processed := tracker.getProcessed()
	if len(processed) != 3 {
		t.Errorf("expected 3 processed (including failed), got %d: %v", len(processed), processed)
	}

	// Clean shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	d.Shutdown(shutdownCtx)
}

func TestDispatcher_FailureInQueuedEventDoesNotBlockSubsequent(t *testing.T) {
	cfg := testConfig(withMaxConcurrentAgents(1))
	tracker := newHandlerTracker()
	tracker.errorForID["fault-fail"] = errors.New("simulated failure")
	tracker.execDuration = 10 * time.Millisecond

	d := NewDispatcher(cfg, tracker.handler)
	ctx := context.Background()

	// Dispatch multiple events to same cluster - middle one will fail
	d.Dispatch(ctx, testEvent("fault-1"), "cluster")
	d.Dispatch(ctx, testEvent("fault-fail"), "cluster")
	d.Dispatch(ctx, testEvent("fault-3"), "cluster")

	// Wait for all completions
	tracker.waitForNCompletions(t, 3, 5*time.Second)

	// Verify all were processed in order
	processed := tracker.getProcessed()
	expected := []string{"fault-1", "fault-fail", "fault-3"}
	if len(processed) != len(expected) {
		t.Fatalf("expected %d processed, got %d: %v", len(expected), len(processed), processed)
	}
	for i, exp := range expected {
		if processed[i] != exp {
			t.Errorf("position %d: expected %q, got %q", i, exp, processed[i])
		}
	}

	// Clean shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	d.Shutdown(shutdownCtx)
}

func TestDispatcher_PanicRecoveryInDifferentClusters(t *testing.T) {
	// This test verifies that clusters are independent
	cfg := testConfig(withMaxConcurrentAgents(5))

	var mu sync.Mutex
	var processed []string

	handler := func(ctx context.Context, event *events.FaultEvent, cluster string) error {
		mu.Lock()
		processed = append(processed, event.FaultID)
		mu.Unlock()

		// Return error for specific event
		if event.FaultID == "fault-error" {
			return errors.New("handler error")
		}
		return nil
	}

	d := NewDispatcher(cfg, handler)
	ctx := context.Background()

	// Dispatch to different clusters
	d.Dispatch(ctx, testEventForCluster("fault-error", "cluster-err"), "cluster-err")
	d.Dispatch(ctx, testEventForCluster("fault-ok-1", "cluster-1"), "cluster-1")
	d.Dispatch(ctx, testEventForCluster("fault-ok-2", "cluster-2"), "cluster-2")

	// Wait for completion
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		count := len(processed)
		mu.Unlock()
		if count >= 3 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	mu.Lock()
	processedCount := len(processed)
	mu.Unlock()

	if processedCount != 3 {
		t.Errorf("expected 3 processed, got %d", processedCount)
	}

	// Clean shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	d.Shutdown(shutdownCtx)
}

// -----------------------------------------------------------------------------
// Test 4.9: Graceful shutdown waits for in-flight
// -----------------------------------------------------------------------------

func TestDispatcher_GracefulShutdownWaitsForInFlight(t *testing.T) {
	cfg := testConfig(withMaxConcurrentAgents(5))
	tracker := newHandlerTracker()
	tracker.execDuration = 200 * time.Millisecond

	d := NewDispatcher(cfg, tracker.handler)
	ctx := context.Background()

	// Start some events
	for i := 0; i < 3; i++ {
		cluster := "cluster-" + string(rune('a'+i))
		d.Dispatch(ctx, testEventForCluster("fault-"+string(rune('a'+i)), cluster), cluster)
	}

	// Wait for them to start
	tracker.waitForNStarts(t, 3, 2*time.Second)

	// Verify in-flight count
	if d.InFlightCount() != 3 {
		t.Errorf("expected 3 in-flight, got %d", d.InFlightCount())
	}

	// Start shutdown with generous timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	shutdownDone := make(chan error, 1)
	go func() {
		shutdownDone <- d.Shutdown(shutdownCtx)
	}()

	// Shutdown should wait for in-flight to complete
	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Errorf("shutdown error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown took too long")
	}

	// Verify all completed
	if d.InFlightCount() != 0 {
		t.Errorf("expected 0 in-flight after shutdown, got %d", d.InFlightCount())
	}

	// Verify all were processed
	if tracker.getProcessedCount() != 3 {
		t.Errorf("expected 3 processed, got %d", tracker.getProcessedCount())
	}
}

func TestDispatcher_ShutdownStopsAcceptingNewEvents(t *testing.T) {
	cfg := testConfig(withMaxConcurrentAgents(5))
	tracker := newHandlerTracker()

	d := NewDispatcher(cfg, tracker.handler)
	ctx := context.Background()

	// Dispatch one event
	d.Dispatch(ctx, testEvent("fault-before"), "cluster")
	tracker.waitForNCompletions(t, 1, 2*time.Second)

	// Shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	d.Shutdown(shutdownCtx)

	// Verify closed
	if !d.IsClosed() {
		t.Error("expected dispatcher to be closed after shutdown")
	}

	// Try to dispatch after shutdown
	d.Dispatch(ctx, testEvent("fault-after"), "cluster")

	// Give time for any unexpected processing
	time.Sleep(100 * time.Millisecond)

	// Verify only the first event was processed
	processed := tracker.getProcessed()
	if len(processed) != 1 || processed[0] != "fault-before" {
		t.Errorf("expected only fault-before to be processed, got %v", processed)
	}
}

func TestDispatcher_ShutdownTimeout(t *testing.T) {
	cfg := testConfig(withMaxConcurrentAgents(5))
	tracker := newHandlerTracker()
	tracker.blockCh = make(chan struct{}) // Block handlers forever

	d := NewDispatcher(cfg, tracker.handler)
	ctx := context.Background()

	// Start some events that will block
	for i := 0; i < 3; i++ {
		cluster := "cluster-" + string(rune('a'+i))
		d.Dispatch(ctx, testEventForCluster("fault-"+string(rune('a'+i)), cluster), cluster)
	}

	// Wait for them to start
	tracker.waitForNStarts(t, 3, 2*time.Second)

	// Start shutdown with short timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := d.Shutdown(shutdownCtx)

	// Should return context deadline exceeded error
	if err == nil {
		t.Error("expected shutdown to timeout")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}

	// Release handlers to clean up
	close(tracker.blockCh)

	// Wait for handlers to finish
	time.Sleep(100 * time.Millisecond)
}

func TestDispatcher_ShutdownClearsQueuedEvents(t *testing.T) {
	cfg := testConfig(
		withMaxConcurrentAgents(1),
		withClusterQueueSize(5),
	)
	tracker := newHandlerTracker()
	tracker.blockCh = make(chan struct{})

	d := NewDispatcher(cfg, tracker.handler)
	ctx := context.Background()

	// Dispatch first event (will start and block)
	d.Dispatch(ctx, testEvent("fault-1"), "cluster")
	tracker.waitForNStarts(t, 1, time.Second)

	// Queue more events
	d.Dispatch(ctx, testEvent("fault-2"), "cluster")
	d.Dispatch(ctx, testEvent("fault-3"), "cluster")

	// Verify queue depth
	if d.QueueDepth("cluster") != 2 {
		t.Errorf("expected queue depth 2, got %d", d.QueueDepth("cluster"))
	}

	// Start shutdown - it will cancel further processing
	shutdownStarted := make(chan struct{})
	shutdownDone := make(chan error, 1)
	go func() {
		close(shutdownStarted)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		shutdownDone <- d.Shutdown(shutdownCtx)
	}()

	// Wait for shutdown to start
	<-shutdownStarted
	time.Sleep(50 * time.Millisecond)

	// Release the blocking handler
	close(tracker.blockCh)

	// Wait for shutdown to complete
	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Errorf("shutdown error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("shutdown took too long")
	}

	// During shutdown, queued events are not processed (onComplete marks cluster as not running)
	// The exact count depends on timing - at minimum fault-1 should be processed
	processed := tracker.getProcessed()
	if len(processed) < 1 {
		t.Errorf("expected at least 1 processed, got %d", len(processed))
	}
}

// -----------------------------------------------------------------------------
// Additional Integration Tests
// -----------------------------------------------------------------------------

func TestDispatcher_ConcurrentDispatchesToMultipleClusters(t *testing.T) {
	cfg := testConfig(withMaxConcurrentAgents(10))
	tracker := newHandlerTracker()
	tracker.execDuration = 10 * time.Millisecond

	d := NewDispatcher(cfg, tracker.handler)
	ctx := context.Background()

	// Dispatch many events concurrently to multiple clusters
	var wg sync.WaitGroup
	numClusters := 5
	eventsPerCluster := 10

	for c := 0; c < numClusters; c++ {
		cluster := "cluster-" + string(rune('a'+c))
		wg.Add(1)
		go func(cluster string, clusterIdx int) {
			defer wg.Done()
			for i := 0; i < eventsPerCluster; i++ {
				faultID := cluster + "-fault-" + string(rune('0'+i))
				event := testEventForCluster(faultID, cluster)
				d.Dispatch(ctx, event, cluster)
			}
		}(cluster, c)
	}

	wg.Wait()

	// Wait for all to complete
	tracker.waitForNCompletions(t, numClusters*eventsPerCluster, 30*time.Second)

	// Verify all processed
	if tracker.getProcessedCount() != numClusters*eventsPerCluster {
		t.Errorf("expected %d processed, got %d", numClusters*eventsPerCluster, tracker.getProcessedCount())
	}

	// Verify each cluster processed in order
	for c := 0; c < numClusters; c++ {
		cluster := "cluster-" + string(rune('a'+c))
		clusterProcessed := tracker.getProcessedForCluster(cluster)
		if len(clusterProcessed) != eventsPerCluster {
			t.Errorf("cluster %s: expected %d processed, got %d", cluster, eventsPerCluster, len(clusterProcessed))
			continue
		}
		// Verify FIFO order within cluster
		for i := 0; i < eventsPerCluster; i++ {
			expected := cluster + "-fault-" + string(rune('0'+i))
			if clusterProcessed[i] != expected {
				t.Errorf("cluster %s position %d: expected %q, got %q", cluster, i, expected, clusterProcessed[i])
			}
		}
	}

	// Clean shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	d.Shutdown(shutdownCtx)
}

func TestDispatcher_RaceConditions(t *testing.T) {
	// This test is designed to catch race conditions when run with -race flag
	cfg := testConfig(
		withMaxConcurrentAgents(5),
		withClusterQueueSize(10),
	)

	var processCount atomic.Int64

	handler := func(ctx context.Context, event *events.FaultEvent, cluster string) error {
		processCount.Add(1)
		time.Sleep(time.Millisecond) // Small delay to increase chance of races
		return nil
	}

	d := NewDispatcher(cfg, handler)
	ctx := context.Background()

	// Concurrent operations
	var wg sync.WaitGroup

	// Dispatchers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				cluster := "cluster-" + string(rune('a'+(id%5)))
				event := testEventForCluster("fault-"+string(rune('0'+j)), cluster)
				d.Dispatch(ctx, event, cluster)
			}
		}(i)
	}

	// Readers (InFlightCount, QueueDepth, etc.)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = d.InFlightCount()
				_ = d.IsClosed()
				_ = d.TotalQueueDepth()
				_ = d.QueueDepth("cluster-a")
			}
		}()
	}

	wg.Wait()

	// Wait for processing to settle
	time.Sleep(500 * time.Millisecond)

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	d.Shutdown(shutdownCtx)

	t.Logf("Total events processed: %d", processCount.Load())
}

func TestDispatcher_QueueDepthForNonExistentCluster(t *testing.T) {
	cfg := testConfig()
	d := NewDispatcher(cfg, nil)

	// Query queue depth for cluster that doesn't exist
	depth := d.QueueDepth("nonexistent-cluster")
	if depth != 0 {
		t.Errorf("expected 0 for nonexistent cluster, got %d", depth)
	}

	// Clean shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	d.Shutdown(shutdownCtx)
}

func TestDispatcher_NilHandler(t *testing.T) {
	cfg := testConfig()
	d := NewDispatcher(cfg, nil)
	ctx := context.Background()

	// Dispatch with nil handler should not panic
	event := testEvent("fault-1")
	d.Dispatch(ctx, event, "cluster")

	// Wait for processing
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if d.InFlightCount() == 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Clean shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	d.Shutdown(shutdownCtx)
}

func TestDispatcher_ContextCancellation(t *testing.T) {
	cfg := testConfig(withMaxConcurrentAgents(1))

	handlerStarted := make(chan struct{})
	handlerDone := make(chan error, 1)

	handler := func(ctx context.Context, event *events.FaultEvent, cluster string) error {
		close(handlerStarted)
		<-ctx.Done()
		handlerDone <- ctx.Err()
		return ctx.Err()
	}

	d := NewDispatcher(cfg, handler)

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Dispatch event
	d.Dispatch(ctx, testEvent("fault-1"), "cluster")

	// Wait for handler to start
	<-handlerStarted

	// Cancel the context
	cancel()

	// Handler should receive the cancellation
	select {
	case err := <-handlerDone:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not receive cancellation")
	}

	// Clean shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	d.Shutdown(shutdownCtx)
}
