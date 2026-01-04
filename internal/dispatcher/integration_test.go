//go:build integration

package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/randybias/nightcrier/internal/config"
	"github.com/randybias/nightcrier/internal/events"
)

// -----------------------------------------------------------------------------
// Integration Test Helpers
// -----------------------------------------------------------------------------

// integrationConfig creates a config suitable for integration testing with
// higher limits and longer timeouts than unit tests.
func integrationConfig(opts ...func(*config.Config)) *config.Config {
	// Default DropEventsWhileBusy to false for tests that rely on queueing behavior
	defaultDropEvents := false
	cfg := &config.Config{
		MaxConcurrentAgents:          20,
		ClusterFailureEventQueueSize: 50,
		EventTTLSeconds:              600, // 10 minutes
		DropEventsWhileBusy:          &defaultDropEvents,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// integrationEvent creates a FaultEvent for integration testing.
func integrationEvent(faultID, cluster string) *events.FaultEvent {
	return &events.FaultEvent{
		FaultID:   faultID,
		Cluster:   cluster,
		FaultType: "IntegrationTestFault",
		Severity:  "warning",
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// orderTracker tracks execution order with timestamps for validation.
type orderTracker struct {
	mu      sync.Mutex
	records []orderRecord
}

type orderRecord struct {
	faultID   string
	cluster   string
	startTime time.Time
	endTime   time.Time
	seq       int64 // global sequence number when handler started
}

var globalSeq atomic.Int64

func newOrderTracker() *orderTracker {
	return &orderTracker{
		records: make([]orderRecord, 0),
	}
}

func (ot *orderTracker) recordExecution(faultID, cluster string, duration time.Duration) {
	start := time.Now()
	seq := globalSeq.Add(1)

	if duration > 0 {
		time.Sleep(duration)
	}

	end := time.Now()

	ot.mu.Lock()
	ot.records = append(ot.records, orderRecord{
		faultID:   faultID,
		cluster:   cluster,
		startTime: start,
		endTime:   end,
		seq:       seq,
	})
	ot.mu.Unlock()
}

func (ot *orderTracker) getRecordsForCluster(cluster string) []orderRecord {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	var result []orderRecord
	for _, r := range ot.records {
		if r.cluster == cluster {
			result = append(result, r)
		}
	}
	return result
}

func (ot *orderTracker) totalRecords() int {
	ot.mu.Lock()
	defer ot.mu.Unlock()
	return len(ot.records)
}

// verifyNoOverlapForCluster checks that events for a cluster never overlapped.
func (ot *orderTracker) verifyNoOverlapForCluster(t *testing.T, cluster string) {
	t.Helper()

	records := ot.getRecordsForCluster(cluster)
	if len(records) < 2 {
		return
	}

	// Sort by start time (they should already be in order due to serialization)
	for i := 1; i < len(records); i++ {
		prev := records[i-1]
		curr := records[i]

		// Check that current started after previous ended
		if curr.startTime.Before(prev.endTime) {
			t.Errorf("cluster %s: event %s started at %v before event %s ended at %v (overlap detected)",
				cluster, curr.faultID, curr.startTime, prev.faultID, prev.endTime)
		}
	}
}

// -----------------------------------------------------------------------------
// Test 6.1: Concurrent events across clusters
// Fire 100 events across 10 clusters, verify all complete and cluster-level
// ordering is preserved
// -----------------------------------------------------------------------------

func TestIntegration_ConcurrentEventsAcrossClusters(t *testing.T) {
	const (
		numClusters        = 10
		eventsPerCluster   = 10
		totalEvents        = numClusters * eventsPerCluster
		maxConcurrentSlots = 15
		execDuration       = 5 * time.Millisecond
	)

	cfg := integrationConfig(func(c *config.Config) {
		c.MaxConcurrentAgents = maxConcurrentSlots
		c.ClusterFailureEventQueueSize = eventsPerCluster + 5 // Room for all events
	})

	tracker := newOrderTracker()
	var completedCount atomic.Int64
	var maxConcurrent atomic.Int32
	var currentConcurrent atomic.Int32

	handler := func(ctx context.Context, event *events.FaultEvent, cluster string) error {
		// Track concurrent executions
		current := currentConcurrent.Add(1)
		defer currentConcurrent.Add(-1)

		// Update max concurrent
		for {
			max := maxConcurrent.Load()
			if current <= max {
				break
			}
			if maxConcurrent.CompareAndSwap(max, current) {
				break
			}
		}

		tracker.recordExecution(event.FaultID, cluster, execDuration)
		completedCount.Add(1)
		return nil
	}

	d := NewDispatcher(cfg, handler)
	ctx := context.Background()

	// Dispatch events concurrently from multiple goroutines
	var wg sync.WaitGroup
	dispatchStart := time.Now()

	for c := 0; c < numClusters; c++ {
		cluster := fmt.Sprintf("cluster-%02d", c)
		wg.Add(1)
		go func(cluster string, clusterIdx int) {
			defer wg.Done()
			for i := 0; i < eventsPerCluster; i++ {
				faultID := fmt.Sprintf("%s-fault-%02d", cluster, i)
				event := integrationEvent(faultID, cluster)
				d.Dispatch(ctx, event, cluster)
			}
		}(cluster, c)
	}

	// Wait for all dispatches to complete
	wg.Wait()
	dispatchDuration := time.Since(dispatchStart)
	t.Logf("All %d events dispatched in %v", totalEvents, dispatchDuration)

	// Wait for all events to complete with a generous timeout
	deadline := time.After(60 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatalf("timeout waiting for events to complete: got %d/%d",
				completedCount.Load(), totalEvents)
		case <-ticker.C:
			if completedCount.Load() >= int64(totalEvents) {
				goto done
			}
		}
	}
done:

	totalDuration := time.Since(dispatchStart)
	t.Logf("All %d events completed in %v", totalEvents, totalDuration)
	t.Logf("Max concurrent executions observed: %d (limit: %d)", maxConcurrent.Load(), maxConcurrentSlots)

	// Verify all events completed
	if completedCount.Load() != int64(totalEvents) {
		t.Errorf("expected %d events completed, got %d", totalEvents, completedCount.Load())
	}

	// Verify we actually used concurrency (different clusters ran in parallel)
	if maxConcurrent.Load() < 2 {
		t.Errorf("expected multiple concurrent executions, got max %d", maxConcurrent.Load())
	}

	// Verify global concurrency limit was respected
	if maxConcurrent.Load() > int32(maxConcurrentSlots) {
		t.Errorf("exceeded max concurrent agents: got %d, limit %d", maxConcurrent.Load(), maxConcurrentSlots)
	}

	// Verify cluster-level ordering: within each cluster, events should not overlap
	for c := 0; c < numClusters; c++ {
		cluster := fmt.Sprintf("cluster-%02d", c)
		tracker.verifyNoOverlapForCluster(t, cluster)
	}

	// Verify each cluster processed the expected number of events
	for c := 0; c < numClusters; c++ {
		cluster := fmt.Sprintf("cluster-%02d", c)
		records := tracker.getRecordsForCluster(cluster)
		if len(records) != eventsPerCluster {
			t.Errorf("cluster %s: expected %d events, got %d", cluster, eventsPerCluster, len(records))
		}
	}

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := d.Shutdown(shutdownCtx); err != nil {
		t.Errorf("shutdown error: %v", err)
	}
}

// -----------------------------------------------------------------------------
// Test 6.2: Serialized events for same cluster
// Fire 20 events to a single cluster, verify strict FIFO ordering with no overlap
// -----------------------------------------------------------------------------

func TestIntegration_SerializedEventsForSameCluster(t *testing.T) {
	const (
		numEvents        = 20
		execDuration     = 10 * time.Millisecond
		targetCluster    = "single-cluster"
	)

	cfg := integrationConfig(func(c *config.Config) {
		c.MaxConcurrentAgents = 10 // Enough slots to show serialization is enforced
		c.ClusterFailureEventQueueSize = numEvents + 5
	})

	type execRecord struct {
		faultID   string
		startTime time.Time
		endTime   time.Time
		seq       int
	}

	var mu sync.Mutex
	var records []execRecord
	var seqCounter int

	handler := func(ctx context.Context, event *events.FaultEvent, cluster string) error {
		mu.Lock()
		seq := seqCounter
		seqCounter++
		mu.Unlock()

		start := time.Now()
		time.Sleep(execDuration)
		end := time.Now()

		mu.Lock()
		records = append(records, execRecord{
			faultID:   event.FaultID,
			startTime: start,
			endTime:   end,
			seq:       seq,
		})
		mu.Unlock()

		return nil
	}

	d := NewDispatcher(cfg, handler)
	ctx := context.Background()

	// Dispatch all events rapidly
	dispatchStart := time.Now()
	for i := 0; i < numEvents; i++ {
		faultID := fmt.Sprintf("fault-%02d", i)
		event := integrationEvent(faultID, targetCluster)
		d.Dispatch(ctx, event, targetCluster)
	}
	t.Logf("All %d events dispatched in %v", numEvents, time.Since(dispatchStart))

	// Wait for all events to complete
	deadline := time.After(30 * time.Second)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			mu.Lock()
			count := len(records)
			mu.Unlock()
			t.Fatalf("timeout waiting for events to complete: got %d/%d", count, numEvents)
		case <-ticker.C:
			mu.Lock()
			count := len(records)
			mu.Unlock()
			if count >= numEvents {
				goto done
			}
		}
	}
done:

	totalDuration := time.Since(dispatchStart)
	t.Logf("All %d events completed in %v", numEvents, totalDuration)

	mu.Lock()
	defer mu.Unlock()

	// Verify all events completed
	if len(records) != numEvents {
		t.Errorf("expected %d events completed, got %d", numEvents, len(records))
	}

	// Verify FIFO ordering: events should be processed in dispatch order
	for i, rec := range records {
		expectedFaultID := fmt.Sprintf("fault-%02d", i)
		if rec.faultID != expectedFaultID {
			t.Errorf("position %d: expected %q, got %q (FIFO order violated)", i, expectedFaultID, rec.faultID)
		}
	}

	// Verify no overlap: each event should start after the previous ended
	for i := 1; i < len(records); i++ {
		prev := records[i-1]
		curr := records[i]

		if curr.startTime.Before(prev.endTime) {
			t.Errorf("event %d (%s) started at %v before event %d (%s) ended at %v",
				i, curr.faultID, curr.startTime, i-1, prev.faultID, prev.endTime)
			t.Logf("  gap between events: %v", curr.startTime.Sub(prev.endTime))
		}
	}

	// Verify minimum duration (serialized events should take at least numEvents * execDuration)
	expectedMinDuration := time.Duration(numEvents) * execDuration
	if totalDuration < expectedMinDuration {
		t.Errorf("events completed too quickly (%v < %v), serialization may not be working",
			totalDuration, expectedMinDuration)
	}

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := d.Shutdown(shutdownCtx); err != nil {
		t.Errorf("shutdown error: %v", err)
	}
}

// -----------------------------------------------------------------------------
// Test 6.3: Shutdown with in-flight agents
// Start many in-flight agents, initiate shutdown, verify it waits appropriately
// -----------------------------------------------------------------------------

func TestIntegration_ShutdownWithInFlightAgents(t *testing.T) {
	const (
		numClusters      = 10
		maxConcurrent    = 10
		agentDuration    = 200 * time.Millisecond
		shutdownTimeout  = 5 * time.Second
	)

	cfg := integrationConfig(func(c *config.Config) {
		c.MaxConcurrentAgents = maxConcurrent
		c.ClusterFailureEventQueueSize = 10
	})

	var startedCount atomic.Int64
	var completedCount atomic.Int64
	handlerStarted := make(chan struct{}, numClusters)

	handler := func(ctx context.Context, event *events.FaultEvent, cluster string) error {
		startedCount.Add(1)
		handlerStarted <- struct{}{}

		// Simulate agent work
		select {
		case <-time.After(agentDuration):
			completedCount.Add(1)
			return nil
		case <-ctx.Done():
			// Still count as completed even if cancelled
			completedCount.Add(1)
			return ctx.Err()
		}
	}

	d := NewDispatcher(cfg, handler)
	ctx := context.Background()

	// Dispatch one event per cluster to maximize concurrency
	for c := 0; c < numClusters; c++ {
		cluster := fmt.Sprintf("cluster-%02d", c)
		event := integrationEvent(fmt.Sprintf("fault-%02d", c), cluster)
		d.Dispatch(ctx, event, cluster)
	}

	// Wait for all handlers to start
	for i := 0; i < numClusters; i++ {
		select {
		case <-handlerStarted:
		case <-time.After(5 * time.Second):
			t.Fatalf("timeout waiting for handlers to start: got %d/%d", startedCount.Load(), numClusters)
		}
	}

	t.Logf("All %d handlers started, in-flight count: %d", numClusters, d.InFlightCount())

	// Verify all agents are in-flight
	inFlight := d.InFlightCount()
	if inFlight != int64(numClusters) {
		t.Errorf("expected %d in-flight, got %d", numClusters, inFlight)
	}

	// Initiate shutdown while agents are still running
	shutdownStart := time.Now()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	err := d.Shutdown(shutdownCtx)
	shutdownDuration := time.Since(shutdownStart)

	t.Logf("Shutdown completed in %v (agent duration: %v)", shutdownDuration, agentDuration)
	t.Logf("Started: %d, Completed: %d", startedCount.Load(), completedCount.Load())

	// Shutdown should complete without error (within timeout)
	if err != nil {
		t.Errorf("shutdown error: %v", err)
	}

	// Shutdown should have waited for agents (took at least agent duration)
	// Allow some tolerance for test timing variations
	expectedMinDuration := agentDuration - 50*time.Millisecond
	if shutdownDuration < expectedMinDuration {
		t.Errorf("shutdown completed too quickly (%v < %v), may not have waited for agents",
			shutdownDuration, expectedMinDuration)
	}

	// All agents should have completed
	if completedCount.Load() != int64(numClusters) {
		t.Errorf("expected %d agents completed, got %d", numClusters, completedCount.Load())
	}

	// In-flight should be 0 after shutdown
	if d.InFlightCount() != 0 {
		t.Errorf("expected 0 in-flight after shutdown, got %d", d.InFlightCount())
	}

	// Dispatcher should be closed
	if !d.IsClosed() {
		t.Error("expected dispatcher to be closed after shutdown")
	}
}

func TestIntegration_ShutdownTimeoutWithBlockedAgents(t *testing.T) {
	const (
		numAgents       = 5
		shortTimeout    = 200 * time.Millisecond
	)

	cfg := integrationConfig(func(c *config.Config) {
		c.MaxConcurrentAgents = numAgents
		c.ClusterFailureEventQueueSize = 10
	})

	blockCh := make(chan struct{})
	var startedCount atomic.Int64
	handlerStarted := make(chan struct{}, numAgents)

	handler := func(ctx context.Context, event *events.FaultEvent, cluster string) error {
		startedCount.Add(1)
		handlerStarted <- struct{}{}

		// Block until channel is closed or context cancelled
		select {
		case <-blockCh:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	d := NewDispatcher(cfg, handler)
	ctx := context.Background()

	// Dispatch events to different clusters
	for c := 0; c < numAgents; c++ {
		cluster := fmt.Sprintf("cluster-%02d", c)
		event := integrationEvent(fmt.Sprintf("fault-%02d", c), cluster)
		d.Dispatch(ctx, event, cluster)
	}

	// Wait for all handlers to start
	for i := 0; i < numAgents; i++ {
		select {
		case <-handlerStarted:
		case <-time.After(5 * time.Second):
			t.Fatalf("timeout waiting for handlers to start")
		}
	}

	t.Logf("All %d handlers started and blocked", numAgents)

	// Initiate shutdown with short timeout (agents are blocked)
	shutdownStart := time.Now()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shortTimeout)
	defer cancel()

	err := d.Shutdown(shutdownCtx)
	shutdownDuration := time.Since(shutdownStart)

	t.Logf("Shutdown returned after %v with error: %v", shutdownDuration, err)

	// Should return context deadline exceeded
	if err == nil {
		t.Error("expected shutdown to timeout")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}

	// Shutdown duration should be approximately the timeout
	if shutdownDuration < shortTimeout-50*time.Millisecond {
		t.Errorf("shutdown returned too quickly: %v < %v", shutdownDuration, shortTimeout)
	}

	// Release blocked handlers to clean up
	close(blockCh)

	// Wait for handlers to finish
	time.Sleep(100 * time.Millisecond)
}

func TestIntegration_ShutdownRejectsNewEvents(t *testing.T) {
	cfg := integrationConfig(func(c *config.Config) {
		c.MaxConcurrentAgents = 5
		c.ClusterFailureEventQueueSize = 10
	})

	var processedBefore atomic.Int64
	var processedAfter atomic.Int64

	handler := func(ctx context.Context, event *events.FaultEvent, cluster string) error {
		if event.FaultID == "before-shutdown" {
			processedBefore.Add(1)
		} else if event.FaultID == "after-shutdown" {
			processedAfter.Add(1)
		}
		return nil
	}

	d := NewDispatcher(cfg, handler)
	ctx := context.Background()

	// Dispatch event before shutdown
	d.Dispatch(ctx, integrationEvent("before-shutdown", "cluster-1"), "cluster-1")

	// Wait for it to process
	deadline := time.After(5 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for pre-shutdown event")
		case <-ticker.C:
			if processedBefore.Load() > 0 {
				goto done
			}
		}
	}
done:

	// Initiate shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := d.Shutdown(shutdownCtx); err != nil {
		t.Errorf("shutdown error: %v", err)
	}

	// Verify closed
	if !d.IsClosed() {
		t.Error("expected dispatcher to be closed")
	}

	// Try to dispatch after shutdown
	d.Dispatch(ctx, integrationEvent("after-shutdown", "cluster-1"), "cluster-1")

	// Give time for any potential processing
	time.Sleep(100 * time.Millisecond)

	// Verify post-shutdown event was not processed
	if processedAfter.Load() > 0 {
		t.Errorf("event dispatched after shutdown was processed (should be rejected)")
	}

	t.Logf("Pre-shutdown events: %d, Post-shutdown events: %d",
		processedBefore.Load(), processedAfter.Load())
}

// -----------------------------------------------------------------------------
// Additional Integration Tests: Error Recovery
// -----------------------------------------------------------------------------

func TestIntegration_ErrorRecoveryAcrossClusters(t *testing.T) {
	const (
		numClusters       = 5
		eventsPerCluster  = 5
		failingCluster    = "cluster-02"
		failingEventIndex = 2
	)

	cfg := integrationConfig(func(c *config.Config) {
		c.MaxConcurrentAgents = 10
		c.ClusterFailureEventQueueSize = 20
	})

	var successCount atomic.Int64
	var failureCount atomic.Int64

	handler := func(ctx context.Context, event *events.FaultEvent, cluster string) error {
		// Simulate random processing time
		time.Sleep(5 * time.Millisecond)

		// Fail specific event
		if cluster == failingCluster && event.FaultID == fmt.Sprintf("%s-fault-%02d", failingCluster, failingEventIndex) {
			failureCount.Add(1)
			return errors.New("simulated failure")
		}

		successCount.Add(1)
		return nil
	}

	d := NewDispatcher(cfg, handler)
	ctx := context.Background()

	// Dispatch events
	totalEvents := numClusters * eventsPerCluster
	for c := 0; c < numClusters; c++ {
		cluster := fmt.Sprintf("cluster-%02d", c)
		for i := 0; i < eventsPerCluster; i++ {
			faultID := fmt.Sprintf("%s-fault-%02d", cluster, i)
			d.Dispatch(ctx, integrationEvent(faultID, cluster), cluster)
		}
	}

	// Wait for all events to complete
	deadline := time.After(30 * time.Second)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatalf("timeout: success=%d, failure=%d, total=%d",
				successCount.Load(), failureCount.Load(), totalEvents)
		case <-ticker.C:
			total := successCount.Load() + failureCount.Load()
			if total >= int64(totalEvents) {
				goto done
			}
		}
	}
done:

	t.Logf("Completed: success=%d, failure=%d", successCount.Load(), failureCount.Load())

	// Verify exactly one failure occurred
	if failureCount.Load() != 1 {
		t.Errorf("expected 1 failure, got %d", failureCount.Load())
	}

	// Verify all other events succeeded
	expectedSuccess := int64(totalEvents - 1)
	if successCount.Load() != expectedSuccess {
		t.Errorf("expected %d successes, got %d", expectedSuccess, successCount.Load())
	}

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := d.Shutdown(shutdownCtx); err != nil {
		t.Errorf("shutdown error: %v", err)
	}
}

func TestIntegration_ErrorRecoveryWithinCluster(t *testing.T) {
	const (
		numEvents         = 10
		failingEventIndex = 3
		targetCluster     = "test-cluster"
	)

	cfg := integrationConfig(func(c *config.Config) {
		c.MaxConcurrentAgents = 5
		c.ClusterFailureEventQueueSize = 20
	})

	var mu sync.Mutex
	var processedOrder []string
	var failedEvents []string

	handler := func(ctx context.Context, event *events.FaultEvent, cluster string) error {
		time.Sleep(10 * time.Millisecond)

		mu.Lock()
		processedOrder = append(processedOrder, event.FaultID)
		mu.Unlock()

		// Fail specific event
		if event.FaultID == fmt.Sprintf("fault-%02d", failingEventIndex) {
			mu.Lock()
			failedEvents = append(failedEvents, event.FaultID)
			mu.Unlock()
			return errors.New("simulated failure")
		}

		return nil
	}

	d := NewDispatcher(cfg, handler)
	ctx := context.Background()

	// Dispatch events
	for i := 0; i < numEvents; i++ {
		faultID := fmt.Sprintf("fault-%02d", i)
		d.Dispatch(ctx, integrationEvent(faultID, targetCluster), targetCluster)
	}

	// Wait for all events to complete
	deadline := time.After(30 * time.Second)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			mu.Lock()
			count := len(processedOrder)
			mu.Unlock()
			t.Fatalf("timeout: processed %d/%d", count, numEvents)
		case <-ticker.C:
			mu.Lock()
			count := len(processedOrder)
			mu.Unlock()
			if count >= numEvents {
				goto done
			}
		}
	}
done:

	mu.Lock()
	defer mu.Unlock()

	t.Logf("Processed order: %v", processedOrder)
	t.Logf("Failed events: %v", failedEvents)

	// Verify all events were processed
	if len(processedOrder) != numEvents {
		t.Errorf("expected %d events processed, got %d", numEvents, len(processedOrder))
	}

	// Verify FIFO ordering was maintained despite failure
	for i, faultID := range processedOrder {
		expected := fmt.Sprintf("fault-%02d", i)
		if faultID != expected {
			t.Errorf("position %d: expected %q, got %q", i, expected, faultID)
		}
	}

	// Verify failure was recorded
	if len(failedEvents) != 1 {
		t.Errorf("expected 1 failed event, got %d", len(failedEvents))
	}

	// Events after failure should still process
	expectedAfterFailure := numEvents - failingEventIndex - 1
	afterFailure := 0
	for _, faultID := range processedOrder {
		var idx int
		fmt.Sscanf(faultID, "fault-%d", &idx)
		if idx > failingEventIndex {
			afterFailure++
		}
	}
	if afterFailure != expectedAfterFailure {
		t.Errorf("expected %d events after failure, got %d", expectedAfterFailure, afterFailure)
	}

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := d.Shutdown(shutdownCtx); err != nil {
		t.Errorf("shutdown error: %v", err)
	}
}

// -----------------------------------------------------------------------------
// Stress Test: High Load
// -----------------------------------------------------------------------------

func TestIntegration_HighLoadStress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	const (
		numClusters       = 20
		eventsPerCluster  = 50
		totalEvents       = numClusters * eventsPerCluster
		maxConcurrentSlots = 25
	)

	cfg := integrationConfig(func(c *config.Config) {
		c.MaxConcurrentAgents = maxConcurrentSlots
		c.ClusterFailureEventQueueSize = eventsPerCluster + 10
	})

	var completedCount atomic.Int64
	var maxConcurrent atomic.Int32
	var currentConcurrent atomic.Int32

	handler := func(ctx context.Context, event *events.FaultEvent, cluster string) error {
		current := currentConcurrent.Add(1)
		defer currentConcurrent.Add(-1)

		// Update max concurrent
		for {
			max := maxConcurrent.Load()
			if current <= max {
				break
			}
			if maxConcurrent.CompareAndSwap(max, current) {
				break
			}
		}

		// Variable processing time
		time.Sleep(time.Duration(1+completedCount.Load()%5) * time.Millisecond)
		completedCount.Add(1)
		return nil
	}

	d := NewDispatcher(cfg, handler)
	ctx := context.Background()

	startTime := time.Now()

	// Dispatch all events as fast as possible
	var wg sync.WaitGroup
	for c := 0; c < numClusters; c++ {
		cluster := fmt.Sprintf("cluster-%02d", c)
		wg.Add(1)
		go func(cluster string) {
			defer wg.Done()
			for i := 0; i < eventsPerCluster; i++ {
				faultID := fmt.Sprintf("%s-fault-%02d", cluster, i)
				d.Dispatch(ctx, integrationEvent(faultID, cluster), cluster)
			}
		}(cluster)
	}
	wg.Wait()

	dispatchDuration := time.Since(startTime)
	t.Logf("Dispatched %d events in %v", totalEvents, dispatchDuration)

	// Wait for completion
	deadline := time.After(120 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	lastCount := int64(0)
	stuckCounter := 0

	for {
		select {
		case <-deadline:
			t.Fatalf("timeout: completed %d/%d events", completedCount.Load(), totalEvents)
		case <-ticker.C:
			current := completedCount.Load()
			if current >= int64(totalEvents) {
				goto done
			}
			// Check for stuck processing
			if current == lastCount {
				stuckCounter++
				if stuckCounter > 50 { // 5 seconds stuck
					t.Logf("Progress stalled at %d/%d for 5 seconds", current, totalEvents)
				}
			} else {
				stuckCounter = 0
				lastCount = current
			}
		}
	}
done:

	totalDuration := time.Since(startTime)
	t.Logf("Completed %d events in %v", totalEvents, totalDuration)
	t.Logf("Max concurrent: %d (limit: %d)", maxConcurrent.Load(), maxConcurrentSlots)
	t.Logf("Throughput: %.2f events/second", float64(totalEvents)/totalDuration.Seconds())

	// Verify all completed
	if completedCount.Load() != int64(totalEvents) {
		t.Errorf("expected %d completed, got %d", totalEvents, completedCount.Load())
	}

	// Verify concurrency limit respected
	if maxConcurrent.Load() > int32(maxConcurrentSlots) {
		t.Errorf("exceeded max concurrent: %d > %d", maxConcurrent.Load(), maxConcurrentSlots)
	}

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := d.Shutdown(shutdownCtx); err != nil {
		t.Errorf("shutdown error: %v", err)
	}
}
