package reporting

import (
	"sync"
	"time"
)

// CircuitBreakerState represents the current state of the circuit breaker
type CircuitBreakerState int

const (
	// StateClosed indicates the circuit is closed (normal operation)
	StateClosed CircuitBreakerState = iota
	// StateOpen indicates the circuit is open (threshold reached, alert sent)
	StateOpen
)

// CircuitBreaker tracks agent failures and determines when to send alerts
type CircuitBreaker struct {
	mu                sync.RWMutex
	threshold         int
	failureCount      int
	firstFailureTime  time.Time
	lastFailureTime   time.Time
	state             CircuitBreakerState
	alerted           bool
	failureReasons    []string
	maxReasons        int
}

// FailureStats contains statistics about failures for alert messages
type FailureStats struct {
	Count            int
	FirstFailureTime time.Time
	LastFailureTime  time.Time
	Duration         time.Duration
	RecentReasons    []string
}

// NewCircuitBreaker creates a new circuit breaker with the specified failure threshold
func NewCircuitBreaker(threshold int) *CircuitBreaker {
	if threshold <= 0 {
		threshold = 3 // Default threshold
	}
	return &CircuitBreaker{
		threshold:      threshold,
		state:          StateClosed,
		maxReasons:     5, // Keep last 5 failure reasons
		failureReasons: make([]string, 0, 5),
	}
}

// RecordFailure records an agent failure and updates the circuit breaker state
func (cb *CircuitBreaker) RecordFailure(reason string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	// First failure
	if cb.failureCount == 0 {
		cb.firstFailureTime = now
	}

	cb.failureCount++
	cb.lastFailureTime = now

	// Store failure reason (keep only most recent ones)
	cb.failureReasons = append(cb.failureReasons, reason)
	if len(cb.failureReasons) > cb.maxReasons {
		cb.failureReasons = cb.failureReasons[1:]
	}

	// Open circuit if threshold reached
	if cb.failureCount >= cb.threshold && cb.state == StateClosed {
		cb.state = StateOpen
	}
}

// RecordSuccess records a successful agent execution and returns whether a recovery alert is needed
func (cb *CircuitBreaker) RecordSuccess() (needsRecoveryAlert bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// If we were in an open state with failures, we need a recovery alert
	needsRecoveryAlert = cb.state == StateOpen && cb.failureCount > 0 && cb.alerted

	// Reset all state
	cb.failureCount = 0
	cb.firstFailureTime = time.Time{}
	cb.lastFailureTime = time.Time{}
	cb.state = StateClosed
	cb.alerted = false
	cb.failureReasons = cb.failureReasons[:0]

	return needsRecoveryAlert
}

// ShouldAlert returns true if an alert should be sent (threshold reached and not yet alerted)
func (cb *CircuitBreaker) ShouldAlert() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateOpen && !cb.alerted {
		cb.alerted = true
		return true
	}

	return false
}

// GetStats returns current failure statistics for alert messages
func (cb *CircuitBreaker) GetStats() FailureStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	duration := time.Duration(0)
	if !cb.firstFailureTime.IsZero() && !cb.lastFailureTime.IsZero() {
		duration = cb.lastFailureTime.Sub(cb.firstFailureTime)
	}

	// Copy reasons to avoid race conditions
	reasons := make([]string, len(cb.failureReasons))
	copy(reasons, cb.failureReasons)

	return FailureStats{
		Count:            cb.failureCount,
		FirstFailureTime: cb.firstFailureTime,
		LastFailureTime:  cb.lastFailureTime,
		Duration:         duration,
		RecentReasons:    reasons,
	}
}

// GetState returns the current circuit breaker state (for testing/monitoring)
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetFailureCount returns the current failure count (for testing/monitoring)
func (cb *CircuitBreaker) GetFailureCount() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failureCount
}

// Reset resets the circuit breaker to initial state (primarily for testing)
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount = 0
	cb.firstFailureTime = time.Time{}
	cb.lastFailureTime = time.Time{}
	cb.state = StateClosed
	cb.alerted = false
	cb.failureReasons = cb.failureReasons[:0]
}
