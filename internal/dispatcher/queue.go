// Package dispatcher provides event dispatching with concurrency control.
package dispatcher

import (
	"log/slog"
	"sync"
	"time"

	"github.com/randybias/nightcrier/internal/events"
)

// QueuedEvent wraps a FaultEvent with metadata for queue management.
type QueuedEvent struct {
	Event      *events.FaultEvent
	EnqueuedAt time.Time
}

// IsExpired returns true if the event has exceeded its TTL.
func (qe *QueuedEvent) IsExpired(ttl time.Duration) bool {
	return time.Now().After(qe.EnqueuedAt.Add(ttl))
}

// EventQueue is a thread-safe queue for FaultEvents with TTL expiration
// and drop-oldest overflow policy.
type EventQueue struct {
	cluster string
	maxSize int
	ttl     time.Duration
	events  []*QueuedEvent
	mu      sync.Mutex
}

// NewEventQueue creates a new EventQueue for the given cluster.
func NewEventQueue(cluster string, maxSize int, ttl time.Duration) *EventQueue {
	return &EventQueue{
		cluster: cluster,
		maxSize: maxSize,
		ttl:     ttl,
		events:  make([]*QueuedEvent, 0, maxSize),
	}
}

// Enqueue adds an event to the queue. It first prunes expired events,
// then drops the oldest event if the queue is full.
func (q *EventQueue) Enqueue(event *QueuedEvent) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// 1. Remove expired events from queue
	q.pruneExpiredLocked()

	// 2. If full, drop oldest (it's the most stale)
	if q.maxSize > 0 && len(q.events) >= q.maxSize {
		dropped := q.events[0]
		q.events = q.events[1:]
		slog.Warn("queue full, dropped oldest event",
			"cluster", q.cluster,
			"dropped_fault_id", dropped.Event.FaultID,
			"dropped_age", time.Since(dropped.EnqueuedAt))
	}

	// 3. Add new event
	q.events = append(q.events, event)
	slog.Debug("event queued",
		"cluster", q.cluster,
		"fault_id", event.Event.FaultID,
		"queue_depth", len(q.events))
}

// PopFrontIfValid removes and returns the first non-expired event.
// It prunes expired events before returning. Returns nil if queue is empty
// or all events are expired.
func (q *EventQueue) PopFrontIfValid() *QueuedEvent {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.pruneExpiredLocked()

	if len(q.events) == 0 {
		return nil
	}

	event := q.events[0]
	q.events = q.events[1:]
	return event
}

// Len returns the current number of events in the queue.
func (q *EventQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.events)
}

// IsEmpty returns true if the queue has no events.
func (q *EventQueue) IsEmpty() bool {
	return q.Len() == 0
}

// pruneExpiredLocked removes expired events from the queue.
// Caller must hold q.mu.
func (q *EventQueue) pruneExpiredLocked() {
	now := time.Now()
	valid := q.events[:0]
	for _, e := range q.events {
		if now.Before(e.EnqueuedAt.Add(q.ttl)) {
			valid = append(valid, e)
		} else {
			slog.Debug("pruned expired event",
				"cluster", q.cluster,
				"fault_id", e.Event.FaultID,
				"age", now.Sub(e.EnqueuedAt))
		}
	}
	q.events = valid
}
