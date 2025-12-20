package reporting

import (
	"sync"
	"testing"
	"time"

	"github.com/rbias/nightcrier/internal/config"
)

func defaultTestTuning() *config.TuningConfig {
	return &config.TuningConfig{
		HTTP: config.HTTPTuning{
			SlackTimeoutSeconds: 10,
		},
		Reporting: config.ReportingTuning{
			RootCauseTruncationLength:  300,
			FailureReasonsDisplayCount: 3,
			MaxFailureReasonsTracked:   5,
		},
	}
}

func TestNewCircuitBreaker(t *testing.T) {
	tests := []struct {
		name              string
		threshold         int
		expectedThreshold int
	}{
		{
			name:              "positive threshold",
			threshold:         5,
			expectedThreshold: 5,
		},
		{
			name:              "zero threshold defaults to 3",
			threshold:         0,
			expectedThreshold: 3,
		},
		{
			name:              "negative threshold defaults to 3",
			threshold:         -1,
			expectedThreshold: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := NewCircuitBreaker(tt.threshold, defaultTestTuning())
			if cb == nil {
				t.Fatal("NewCircuitBreaker returned nil")
			}
			if cb.threshold != tt.expectedThreshold {
				t.Errorf("threshold = %d, want %d", cb.threshold, tt.expectedThreshold)
			}
			if cb.state != StateClosed {
				t.Errorf("initial state = %d, want StateClosed (%d)", cb.state, StateClosed)
			}
			if cb.failureCount != 0 {
				t.Errorf("initial failureCount = %d, want 0", cb.failureCount)
			}
		})
	}
}

func TestRecordFailure(t *testing.T) {
	cb := NewCircuitBreaker(3, defaultTestTuning())

	// Record first failure
	cb.RecordFailure("API connection failed")
	if cb.GetFailureCount() != 1 {
		t.Errorf("failureCount after 1 failure = %d, want 1", cb.GetFailureCount())
	}
	if cb.GetState() != StateClosed {
		t.Errorf("state after 1 failure = %d, want StateClosed (%d)", cb.GetState(), StateClosed)
	}

	// Record second failure
	cb.RecordFailure("API timeout")
	if cb.GetFailureCount() != 2 {
		t.Errorf("failureCount after 2 failures = %d, want 2", cb.GetFailureCount())
	}
	if cb.GetState() != StateClosed {
		t.Errorf("state after 2 failures = %d, want StateClosed (%d)", cb.GetState(), StateClosed)
	}

	// Record third failure - should open circuit
	cb.RecordFailure("API unavailable")
	if cb.GetFailureCount() != 3 {
		t.Errorf("failureCount after 3 failures = %d, want 3", cb.GetFailureCount())
	}
	if cb.GetState() != StateOpen {
		t.Errorf("state after 3 failures = %d, want StateOpen (%d)", cb.GetState(), StateOpen)
	}

	// Verify stats
	stats := cb.GetStats()
	if stats.Count != 3 {
		t.Errorf("stats.Count = %d, want 3", stats.Count)
	}
	if len(stats.RecentReasons) != 3 {
		t.Errorf("len(stats.RecentReasons) = %d, want 3", len(stats.RecentReasons))
	}
}

func TestRecordSuccess(t *testing.T) {
	cb := NewCircuitBreaker(2, defaultTestTuning())

	// Record failures to open circuit
	cb.RecordFailure("failure 1")
	cb.RecordFailure("failure 2")

	// Mark as alerted
	shouldAlert := cb.ShouldAlert()
	if !shouldAlert {
		t.Error("ShouldAlert() = false after reaching threshold, want true")
	}

	if cb.GetState() != StateOpen {
		t.Errorf("state before success = %d, want StateOpen (%d)", cb.GetState(), StateOpen)
	}

	// Record success - should return true for recovery alert
	needsRecoveryAlert := cb.RecordSuccess()
	if !needsRecoveryAlert {
		t.Error("RecordSuccess() = false when recovering from open state, want true")
	}

	// Verify reset
	if cb.GetFailureCount() != 0 {
		t.Errorf("failureCount after success = %d, want 0", cb.GetFailureCount())
	}
	if cb.GetState() != StateClosed {
		t.Errorf("state after success = %d, want StateClosed (%d)", cb.GetState(), StateClosed)
	}

	stats := cb.GetStats()
	if stats.Count != 0 {
		t.Errorf("stats.Count after reset = %d, want 0", stats.Count)
	}
	if len(stats.RecentReasons) != 0 {
		t.Errorf("len(stats.RecentReasons) after reset = %d, want 0", len(stats.RecentReasons))
	}
}

func TestRecordSuccess_NoRecoveryAlertIfNotAlerted(t *testing.T) {
	cb := NewCircuitBreaker(2, defaultTestTuning())

	// Record failures but don't call ShouldAlert
	cb.RecordFailure("failure 1")
	cb.RecordFailure("failure 2")

	// Record success - should NOT need recovery alert since we never sent an alert
	needsRecoveryAlert := cb.RecordSuccess()
	if needsRecoveryAlert {
		t.Error("RecordSuccess() = true when never alerted, want false")
	}
}

func TestShouldAlert(t *testing.T) {
	cb := NewCircuitBreaker(3, defaultTestTuning())

	// Should not alert before threshold
	cb.RecordFailure("failure 1")
	if cb.ShouldAlert() {
		t.Error("ShouldAlert() = true before threshold, want false")
	}

	cb.RecordFailure("failure 2")
	if cb.ShouldAlert() {
		t.Error("ShouldAlert() = true before threshold, want false")
	}

	// Should alert when threshold reached
	cb.RecordFailure("failure 3")
	if !cb.ShouldAlert() {
		t.Error("ShouldAlert() = false at threshold, want true")
	}

	// Should not alert again (already alerted)
	if cb.ShouldAlert() {
		t.Error("ShouldAlert() = true on second call, want false (already alerted)")
	}
}

func TestGetStats(t *testing.T) {
	cb := NewCircuitBreaker(5, defaultTestTuning())

	// Record multiple failures with small delays
	reasons := []string{"failure 1", "failure 2", "failure 3"}
	for _, reason := range reasons {
		cb.RecordFailure(reason)
		time.Sleep(10 * time.Millisecond)
	}

	stats := cb.GetStats()

	if stats.Count != 3 {
		t.Errorf("stats.Count = %d, want 3", stats.Count)
	}

	if stats.FirstFailureTime.IsZero() {
		t.Error("stats.FirstFailureTime is zero, want valid timestamp")
	}

	if stats.LastFailureTime.IsZero() {
		t.Error("stats.LastFailureTime is zero, want valid timestamp")
	}

	if stats.Duration <= 0 {
		t.Errorf("stats.Duration = %v, want > 0", stats.Duration)
	}

	if len(stats.RecentReasons) != 3 {
		t.Errorf("len(stats.RecentReasons) = %d, want 3", len(stats.RecentReasons))
	}

	for i, reason := range reasons {
		if stats.RecentReasons[i] != reason {
			t.Errorf("stats.RecentReasons[%d] = %q, want %q", i, stats.RecentReasons[i], reason)
		}
	}
}

func TestMaxReasons(t *testing.T) {
	cb := NewCircuitBreaker(10, defaultTestTuning())

	// Record more failures than maxReasons (5)
	for i := 0; i < 8; i++ {
		cb.RecordFailure("failure")
	}

	stats := cb.GetStats()
	if len(stats.RecentReasons) > 5 {
		t.Errorf("len(stats.RecentReasons) = %d, want <= 5", len(stats.RecentReasons))
	}
}

func TestThreadSafety(t *testing.T) {
	cb := NewCircuitBreaker(100, defaultTestTuning())
	var wg sync.WaitGroup
	numGoroutines := 50
	failuresPerGoroutine := 10

	// Concurrent failures
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < failuresPerGoroutine; j++ {
				cb.RecordFailure("concurrent failure")
				time.Sleep(1 * time.Millisecond)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				_ = cb.GetStats()
				_ = cb.GetState()
				_ = cb.GetFailureCount()
				time.Sleep(1 * time.Millisecond)
			}
		}()
	}

	wg.Wait()

	expectedCount := numGoroutines * failuresPerGoroutine
	actualCount := cb.GetFailureCount()
	if actualCount != expectedCount {
		t.Errorf("failureCount after concurrent operations = %d, want %d", actualCount, expectedCount)
	}
}

func TestStateTransitions(t *testing.T) {
	cb := NewCircuitBreaker(2, defaultTestTuning())

	// Initial state: Closed
	if cb.GetState() != StateClosed {
		t.Errorf("initial state = %d, want StateClosed (%d)", cb.GetState(), StateClosed)
	}

	// After 1 failure: Still closed
	cb.RecordFailure("failure 1")
	if cb.GetState() != StateClosed {
		t.Errorf("state after 1 failure = %d, want StateClosed (%d)", cb.GetState(), StateClosed)
	}

	// After 2 failures: Open
	cb.RecordFailure("failure 2")
	if cb.GetState() != StateOpen {
		t.Errorf("state after 2 failures = %d, want StateOpen (%d)", cb.GetState(), StateOpen)
	}

	// After success: Closed
	cb.RecordSuccess()
	if cb.GetState() != StateClosed {
		t.Errorf("state after success = %d, want StateClosed (%d)", cb.GetState(), StateClosed)
	}
}

func TestReset(t *testing.T) {
	cb := NewCircuitBreaker(2, defaultTestTuning())

	// Record failures and open circuit
	cb.RecordFailure("failure 1")
	cb.RecordFailure("failure 2")
	cb.ShouldAlert()

	// Reset
	cb.Reset()

	// Verify everything is cleared
	if cb.GetFailureCount() != 0 {
		t.Errorf("failureCount after reset = %d, want 0", cb.GetFailureCount())
	}
	if cb.GetState() != StateClosed {
		t.Errorf("state after reset = %d, want StateClosed (%d)", cb.GetState(), StateClosed)
	}
	if cb.ShouldAlert() {
		t.Error("ShouldAlert() = true after reset, want false")
	}

	stats := cb.GetStats()
	if stats.Count != 0 {
		t.Errorf("stats.Count after reset = %d, want 0", stats.Count)
	}
	if len(stats.RecentReasons) != 0 {
		t.Errorf("len(stats.RecentReasons) after reset = %d, want 0", len(stats.RecentReasons))
	}
}

func TestMultipleCycles(t *testing.T) {
	cb := NewCircuitBreaker(2, defaultTestTuning())

	// First cycle: fail -> recover
	cb.RecordFailure("cycle 1 failure 1")
	cb.RecordFailure("cycle 1 failure 2")
	if !cb.ShouldAlert() {
		t.Error("ShouldAlert() = false in first cycle, want true")
	}
	if !cb.RecordSuccess() {
		t.Error("RecordSuccess() = false in first cycle, want true (recovery alert)")
	}

	// Second cycle: fail -> recover
	cb.RecordFailure("cycle 2 failure 1")
	cb.RecordFailure("cycle 2 failure 2")
	if !cb.ShouldAlert() {
		t.Error("ShouldAlert() = false in second cycle, want true")
	}
	if !cb.RecordSuccess() {
		t.Error("RecordSuccess() = false in second cycle, want true (recovery alert)")
	}

	// Verify clean state after multiple cycles
	if cb.GetFailureCount() != 0 {
		t.Errorf("failureCount after cycles = %d, want 0", cb.GetFailureCount())
	}
	if cb.GetState() != StateClosed {
		t.Errorf("state after cycles = %d, want StateClosed (%d)", cb.GetState(), StateClosed)
	}
}
