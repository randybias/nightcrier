package dispatcher

import (
	"sync"
	"testing"
	"time"

	"github.com/rbias/nightcrier/internal/events"
)

// Helper to create a test FaultEvent
func newTestFaultEvent(faultID string) *events.FaultEvent {
	return &events.FaultEvent{
		FaultID:   faultID,
		Cluster:   "test-cluster",
		FaultType: "TestFault",
		Severity:  "warning",
	}
}

// Helper to create a QueuedEvent with a specific enqueue time
func newQueuedEvent(faultID string, enqueuedAt time.Time) *QueuedEvent {
	return &QueuedEvent{
		Event:      newTestFaultEvent(faultID),
		EnqueuedAt: enqueuedAt,
	}
}

func TestNewEventQueue(t *testing.T) {
	q := NewEventQueue("test-cluster", 10, 5*time.Minute)

	if q.cluster != "test-cluster" {
		t.Errorf("expected cluster 'test-cluster', got %q", q.cluster)
	}
	if q.maxSize != 10 {
		t.Errorf("expected maxSize 10, got %d", q.maxSize)
	}
	if q.ttl != 5*time.Minute {
		t.Errorf("expected ttl 5m, got %v", q.ttl)
	}
	if q.Len() != 0 {
		t.Errorf("expected empty queue, got len %d", q.Len())
	}
}

func TestQueuedEvent_IsExpired(t *testing.T) {
	tests := []struct {
		name       string
		enqueuedAt time.Time
		ttl        time.Duration
		wantExpired bool
	}{
		{
			name:       "not expired - recent event",
			enqueuedAt: time.Now(),
			ttl:        5 * time.Minute,
			wantExpired: false,
		},
		{
			name:       "not expired - just within TTL",
			enqueuedAt: time.Now().Add(-4 * time.Minute),
			ttl:        5 * time.Minute,
			wantExpired: false,
		},
		{
			name:       "expired - past TTL",
			enqueuedAt: time.Now().Add(-6 * time.Minute),
			ttl:        5 * time.Minute,
			wantExpired: true,
		},
		{
			name:       "expired - exactly at TTL boundary (past)",
			enqueuedAt: time.Now().Add(-5*time.Minute - time.Millisecond),
			ttl:        5 * time.Minute,
			wantExpired: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qe := &QueuedEvent{
				Event:      newTestFaultEvent("test"),
				EnqueuedAt: tt.enqueuedAt,
			}
			if got := qe.IsExpired(tt.ttl); got != tt.wantExpired {
				t.Errorf("IsExpired() = %v, want %v", got, tt.wantExpired)
			}
		})
	}
}

func TestEventQueue_EnqueueAndLen(t *testing.T) {
	q := NewEventQueue("test-cluster", 5, 5*time.Minute)

	// Test initial state
	if !q.IsEmpty() {
		t.Error("expected queue to be empty initially")
	}

	// Enqueue some events
	for i := 0; i < 3; i++ {
		q.Enqueue(&QueuedEvent{
			Event:      newTestFaultEvent("fault-" + string(rune('a'+i))),
			EnqueuedAt: time.Now(),
		})
	}

	if q.Len() != 3 {
		t.Errorf("expected len 3, got %d", q.Len())
	}
	if q.IsEmpty() {
		t.Error("expected queue to not be empty")
	}
}

func TestEventQueue_TTLExpiration(t *testing.T) {
	ttl := 100 * time.Millisecond
	q := NewEventQueue("test-cluster", 10, ttl)

	// Enqueue an event that will expire
	q.Enqueue(&QueuedEvent{
		Event:      newTestFaultEvent("old-fault"),
		EnqueuedAt: time.Now().Add(-200 * time.Millisecond), // Already expired
	})

	// Enqueue a fresh event
	q.Enqueue(&QueuedEvent{
		Event:      newTestFaultEvent("fresh-fault"),
		EnqueuedAt: time.Now(),
	})

	// PopFrontIfValid should skip the expired one
	event := q.PopFrontIfValid()
	if event == nil {
		t.Fatal("expected to get an event")
	}
	if event.Event.FaultID != "fresh-fault" {
		t.Errorf("expected 'fresh-fault', got %q", event.Event.FaultID)
	}

	// Queue should now be empty
	if !q.IsEmpty() {
		t.Errorf("expected empty queue, got len %d", q.Len())
	}
}

func TestEventQueue_TTLExpirationDuringEnqueue(t *testing.T) {
	ttl := 100 * time.Millisecond
	q := NewEventQueue("test-cluster", 10, ttl)

	// Enqueue events that are already expired
	for i := 0; i < 3; i++ {
		q.Enqueue(&QueuedEvent{
			Event:      newTestFaultEvent("expired-" + string(rune('a'+i))),
			EnqueuedAt: time.Now().Add(-200 * time.Millisecond),
		})
	}

	// When we enqueue a new event, pruning happens first
	q.Enqueue(&QueuedEvent{
		Event:      newTestFaultEvent("fresh"),
		EnqueuedAt: time.Now(),
	})

	// Should only have the fresh event
	if q.Len() != 1 {
		t.Errorf("expected len 1 after pruning, got %d", q.Len())
	}
}

func TestEventQueue_DropOldestWhenFull(t *testing.T) {
	q := NewEventQueue("test-cluster", 3, 5*time.Minute)

	// Fill the queue
	for i := 0; i < 3; i++ {
		q.Enqueue(&QueuedEvent{
			Event:      newTestFaultEvent("fault-" + string(rune('a'+i))),
			EnqueuedAt: time.Now(),
		})
	}

	if q.Len() != 3 {
		t.Errorf("expected len 3, got %d", q.Len())
	}

	// Enqueue one more - should drop the oldest
	q.Enqueue(&QueuedEvent{
		Event:      newTestFaultEvent("fault-d"),
		EnqueuedAt: time.Now(),
	})

	// Still at max size
	if q.Len() != 3 {
		t.Errorf("expected len 3 after overflow, got %d", q.Len())
	}

	// Pop all and verify oldest was dropped
	var faultIDs []string
	for {
		event := q.PopFrontIfValid()
		if event == nil {
			break
		}
		faultIDs = append(faultIDs, event.Event.FaultID)
	}

	// fault-a should have been dropped
	expected := []string{"fault-b", "fault-c", "fault-d"}
	if len(faultIDs) != len(expected) {
		t.Fatalf("expected %d events, got %d: %v", len(expected), len(faultIDs), faultIDs)
	}
	for i, id := range expected {
		if faultIDs[i] != id {
			t.Errorf("position %d: expected %q, got %q", i, id, faultIDs[i])
		}
	}
}

func TestEventQueue_DropOldestMultipleTimes(t *testing.T) {
	q := NewEventQueue("test-cluster", 2, 5*time.Minute)

	// Enqueue 5 events into a queue of size 2
	for i := 0; i < 5; i++ {
		q.Enqueue(&QueuedEvent{
			Event:      newTestFaultEvent("fault-" + string(rune('a'+i))),
			EnqueuedAt: time.Now(),
		})
	}

	// Should only have the last 2
	if q.Len() != 2 {
		t.Errorf("expected len 2, got %d", q.Len())
	}

	event1 := q.PopFrontIfValid()
	event2 := q.PopFrontIfValid()

	if event1.Event.FaultID != "fault-d" {
		t.Errorf("expected fault-d, got %q", event1.Event.FaultID)
	}
	if event2.Event.FaultID != "fault-e" {
		t.Errorf("expected fault-e, got %q", event2.Event.FaultID)
	}
}

func TestEventQueue_PopFrontIfValid_EmptyQueue(t *testing.T) {
	q := NewEventQueue("test-cluster", 10, 5*time.Minute)

	event := q.PopFrontIfValid()
	if event != nil {
		t.Errorf("expected nil from empty queue, got %v", event)
	}
}

func TestEventQueue_PopFrontIfValid_AllExpired(t *testing.T) {
	ttl := 100 * time.Millisecond
	q := NewEventQueue("test-cluster", 10, ttl)

	// Enqueue expired events
	for i := 0; i < 3; i++ {
		q.Enqueue(&QueuedEvent{
			Event:      newTestFaultEvent("expired-" + string(rune('a'+i))),
			EnqueuedAt: time.Now().Add(-200 * time.Millisecond),
		})
	}

	// All are expired, should return nil
	event := q.PopFrontIfValid()
	if event != nil {
		t.Errorf("expected nil when all expired, got %v", event)
	}

	// Queue should be empty after pruning
	if !q.IsEmpty() {
		t.Errorf("expected empty queue after pruning, got len %d", q.Len())
	}
}

func TestEventQueue_ConcurrentAccess(t *testing.T) {
	q := NewEventQueue("test-cluster", 100, 5*time.Minute)
	var wg sync.WaitGroup
	numGoroutines := 10
	eventsPerGoroutine := 50

	// Concurrent enqueuers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				q.Enqueue(&QueuedEvent{
					Event:      newTestFaultEvent("fault"),
					EnqueuedAt: time.Now(),
				})
			}
		}(i)
	}

	// Concurrent readers (Len and IsEmpty)
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				_ = q.Len()
				_ = q.IsEmpty()
			}
		}()
	}

	// Concurrent poppers
	var popped int
	var poppedMu sync.Mutex
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				if q.PopFrontIfValid() != nil {
					poppedMu.Lock()
					popped++
					poppedMu.Unlock()
				}
			}
		}()
	}

	wg.Wait()

	// Verify no panic and queue is in a consistent state
	finalLen := q.Len()
	t.Logf("Final queue length: %d, popped: %d", finalLen, popped)

	// Total enqueued = numGoroutines * eventsPerGoroutine = 500
	// But queue is capped at 100, and we had poppers running concurrently
	// Just verify we didn't crash and length is within bounds
	if finalLen < 0 || finalLen > 100 {
		t.Errorf("queue length %d is out of bounds [0, 100]", finalLen)
	}
}

func TestEventQueue_ConcurrentEnqueueAndPop(t *testing.T) {
	q := NewEventQueue("test-cluster", 10, 5*time.Minute)
	var wg sync.WaitGroup

	// Producer
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			q.Enqueue(&QueuedEvent{
				Event:      newTestFaultEvent("fault-" + string(rune(i))),
				EnqueuedAt: time.Now(),
			})
			time.Sleep(time.Microsecond)
		}
	}()

	// Consumer
	var consumed int
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			if q.PopFrontIfValid() != nil {
				consumed++
			}
			time.Sleep(time.Microsecond)
		}
	}()

	wg.Wait()
	t.Logf("Consumed %d events, remaining in queue: %d", consumed, q.Len())
}

func TestEventQueue_TTLExpirationRace(t *testing.T) {
	// Use a very short TTL to test race conditions around expiration
	ttl := 10 * time.Millisecond
	q := NewEventQueue("test-cluster", 50, ttl)
	var wg sync.WaitGroup

	// Enqueue events that will expire quickly
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				q.Enqueue(&QueuedEvent{
					Event:      newTestFaultEvent("fault"),
					EnqueuedAt: time.Now(),
				})
				time.Sleep(time.Millisecond)
			}
		}()
	}

	// Pop events, some may be expired by the time we get to them
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 30; j++ {
				q.PopFrontIfValid()
				time.Sleep(2 * time.Millisecond)
			}
		}()
	}

	wg.Wait()
	// Test passes if no race conditions detected (run with -race flag)
}

func TestEventQueue_FIFO_Order(t *testing.T) {
	q := NewEventQueue("test-cluster", 10, 5*time.Minute)

	// Enqueue in order
	for i := 0; i < 5; i++ {
		q.Enqueue(&QueuedEvent{
			Event:      newTestFaultEvent("fault-" + string(rune('a'+i))),
			EnqueuedAt: time.Now(),
		})
	}

	// Pop and verify FIFO order
	expected := []string{"fault-a", "fault-b", "fault-c", "fault-d", "fault-e"}
	for i, exp := range expected {
		event := q.PopFrontIfValid()
		if event == nil {
			t.Fatalf("expected event at position %d, got nil", i)
		}
		if event.Event.FaultID != exp {
			t.Errorf("position %d: expected %q, got %q", i, exp, event.Event.FaultID)
		}
	}
}

func TestEventQueue_ZeroMaxSize(t *testing.T) {
	// Edge case: queue with max size 0 means unbounded (no drop-oldest)
	// This is handled by checking maxSize > 0 before dropping
	q := NewEventQueue("test-cluster", 0, 5*time.Minute)

	// Enqueue multiple events
	for i := 0; i < 5; i++ {
		q.Enqueue(&QueuedEvent{
			Event:      newTestFaultEvent("fault-" + string(rune('a'+i))),
			EnqueuedAt: time.Now(),
		})
	}

	// With maxSize 0, nothing is dropped - queue grows unbounded
	if q.Len() != 5 {
		t.Errorf("expected len 5 (unbounded queue), got %d", q.Len())
	}

	// Verify all events are present in FIFO order
	expected := []string{"fault-a", "fault-b", "fault-c", "fault-d", "fault-e"}
	for i, exp := range expected {
		event := q.PopFrontIfValid()
		if event == nil {
			t.Fatalf("expected event at position %d, got nil", i)
		}
		if event.Event.FaultID != exp {
			t.Errorf("position %d: expected %q, got %q", i, exp, event.Event.FaultID)
		}
	}
}

func TestEventQueue_SingleElement(t *testing.T) {
	q := NewEventQueue("test-cluster", 1, 5*time.Minute)

	q.Enqueue(&QueuedEvent{
		Event:      newTestFaultEvent("fault-a"),
		EnqueuedAt: time.Now(),
	})

	if q.Len() != 1 {
		t.Errorf("expected len 1, got %d", q.Len())
	}

	// Enqueue another - should replace
	q.Enqueue(&QueuedEvent{
		Event:      newTestFaultEvent("fault-b"),
		EnqueuedAt: time.Now(),
	})

	if q.Len() != 1 {
		t.Errorf("expected len 1 after overflow, got %d", q.Len())
	}

	event := q.PopFrontIfValid()
	if event.Event.FaultID != "fault-b" {
		t.Errorf("expected fault-b (newest), got %q", event.Event.FaultID)
	}
}
