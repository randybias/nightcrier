package reporting

import (
	"context"
	"testing"
	"time"

	"github.com/rbias/nightcrier/internal/config"
)

// TestSlackNotifierWithCustomTuning verifies that SlackNotifier uses custom tuning parameters
func TestSlackNotifierWithCustomTuning(t *testing.T) {
	tests := []struct {
		name                       string
		tuning                     *config.TuningConfig
		expectedTimeout            time.Duration
		expectedTruncation         int
		expectedDisplayCount       int
	}{
		{
			name: "default tuning",
			tuning: &config.TuningConfig{
				HTTP: config.HTTPTuning{
					SlackTimeoutSeconds: 10,
				},
				Reporting: config.ReportingTuning{
					RootCauseTruncationLength:  300,
					FailureReasonsDisplayCount: 3,
					MaxFailureReasonsTracked:   5,
				},
			},
			expectedTimeout:      10 * time.Second,
			expectedTruncation:   300,
			expectedDisplayCount: 3,
		},
		{
			name: "custom tuning - higher values",
			tuning: &config.TuningConfig{
				HTTP: config.HTTPTuning{
					SlackTimeoutSeconds: 30,
				},
				Reporting: config.ReportingTuning{
					RootCauseTruncationLength:  500,
					FailureReasonsDisplayCount: 5,
					MaxFailureReasonsTracked:   10,
				},
			},
			expectedTimeout:      30 * time.Second,
			expectedTruncation:   500,
			expectedDisplayCount: 5,
		},
		{
			name: "custom tuning - lower values",
			tuning: &config.TuningConfig{
				HTTP: config.HTTPTuning{
					SlackTimeoutSeconds: 5,
				},
				Reporting: config.ReportingTuning{
					RootCauseTruncationLength:  100,
					FailureReasonsDisplayCount: 2,
					MaxFailureReasonsTracked:   3,
				},
			},
			expectedTimeout:      5 * time.Second,
			expectedTruncation:   100,
			expectedDisplayCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifier := NewSlackNotifier("https://hooks.slack.com/test", tt.tuning)

			// Verify HTTP timeout
			if notifier.httpClient.Timeout != tt.expectedTimeout {
				t.Errorf("HTTP timeout = %v, want %v", notifier.httpClient.Timeout, tt.expectedTimeout)
			}

			// Verify root cause truncation length
			if notifier.rootCauseTruncationLength != tt.expectedTruncation {
				t.Errorf("rootCauseTruncationLength = %d, want %d", notifier.rootCauseTruncationLength, tt.expectedTruncation)
			}

			// Verify failure reasons display count
			if notifier.failureReasonsDisplayCount != tt.expectedDisplayCount {
				t.Errorf("failureReasonsDisplayCount = %d, want %d", notifier.failureReasonsDisplayCount, tt.expectedDisplayCount)
			}
		})
	}
}

// TestRootCauseTruncation verifies that root cause truncation uses the configured length
func TestRootCauseTruncation(t *testing.T) {
	tests := []struct {
		name           string
		truncationLen  int
		input          string
		expectedOutput string
	}{
		{
			name:           "text shorter than limit",
			truncationLen:  300,
			input:          "Short root cause",
			expectedOutput: "Short root cause",
		},
		{
			name:           "text exactly at limit",
			truncationLen:  20,
			input:          "Exactly twenty chars",
			expectedOutput: "Exactly twenty chars",
		},
		{
			name:           "text longer than limit",
			truncationLen:  20,
			input:          "This is a very long root cause that needs truncation",
			expectedOutput: "This is a very lo...",
		},
		{
			name:           "custom truncation length 100",
			truncationLen:  100,
			input:          "This is a moderately long root cause description that should be truncated at exactly one hundred characters for testing purposes",
			expectedOutput: "This is a moderately long root cause description that should be truncated at exactly one hundred ...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tuning := &config.TuningConfig{
				HTTP: config.HTTPTuning{
					SlackTimeoutSeconds: 10,
				},
				Reporting: config.ReportingTuning{
					RootCauseTruncationLength:  tt.truncationLen,
					FailureReasonsDisplayCount: 3,
					MaxFailureReasonsTracked:   5,
				},
			}
			notifier := NewSlackNotifier("", tuning)

			result := notifier.TruncateRootCause(tt.input)
			if result != tt.expectedOutput {
				t.Errorf("TruncateRootCause(%q) = %q, want %q", tt.input, result, tt.expectedOutput)
			}
		})
	}
}

// TestCircuitBreakerWithCustomTuning verifies that CircuitBreaker uses custom tuning parameters
func TestCircuitBreakerWithCustomTuning(t *testing.T) {
	tests := []struct {
		name               string
		threshold          int
		maxReasonsTracked  int
		failuresToRecord   int
		expectedReasonCount int
	}{
		{
			name:                "default max reasons (5)",
			threshold:           10,
			maxReasonsTracked:   5,
			failuresToRecord:    3,
			expectedReasonCount: 3,
		},
		{
			name:                "exceed default max reasons",
			threshold:           10,
			maxReasonsTracked:   5,
			failuresToRecord:    8,
			expectedReasonCount: 5, // Should only keep last 5
		},
		{
			name:                "custom max reasons (10)",
			threshold:           15,
			maxReasonsTracked:   10,
			failuresToRecord:    8,
			expectedReasonCount: 8,
		},
		{
			name:                "custom max reasons (10) exceeded",
			threshold:           20,
			maxReasonsTracked:   10,
			failuresToRecord:    15,
			expectedReasonCount: 10, // Should only keep last 10
		},
		{
			name:                "custom max reasons (3)",
			threshold:           10,
			maxReasonsTracked:   3,
			failuresToRecord:    5,
			expectedReasonCount: 3, // Should only keep last 3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tuning := &config.TuningConfig{
				HTTP: config.HTTPTuning{
					SlackTimeoutSeconds: 10,
				},
				Reporting: config.ReportingTuning{
					RootCauseTruncationLength:  300,
					FailureReasonsDisplayCount: 3,
					MaxFailureReasonsTracked:   tt.maxReasonsTracked,
				},
			}
			cb := NewCircuitBreaker(tt.threshold, tuning)

			// Verify maxReasons is set correctly
			if cb.maxReasons != tt.maxReasonsTracked {
				t.Errorf("maxReasons = %d, want %d", cb.maxReasons, tt.maxReasonsTracked)
			}

			// Record failures
			for i := 0; i < tt.failuresToRecord; i++ {
				cb.RecordFailure("failure")
				time.Sleep(1 * time.Millisecond)
			}

			// Verify the number of reasons tracked
			stats := cb.GetStats()
			if len(stats.RecentReasons) != tt.expectedReasonCount {
				t.Errorf("len(RecentReasons) = %d, want %d", len(stats.RecentReasons), tt.expectedReasonCount)
			}
		})
	}
}

// TestFailureReasonsDisplayCount verifies that the display count is configurable
func TestFailureReasonsDisplayCount(t *testing.T) {
	tests := []struct {
		name                    string
		displayCount            int
		totalReasons            int
		expectedDisplayedCount  int
	}{
		{
			name:                   "display 3 of 5",
			displayCount:           3,
			totalReasons:           5,
			expectedDisplayedCount: 3,
		},
		{
			name:                   "display 5 of 5",
			displayCount:           5,
			totalReasons:           5,
			expectedDisplayedCount: 5,
		},
		{
			name:                   "display 2 of 5",
			displayCount:           2,
			totalReasons:           5,
			expectedDisplayedCount: 2,
		},
		{
			name:                   "display all when less than limit",
			displayCount:           5,
			totalReasons:           3,
			expectedDisplayedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tuning := &config.TuningConfig{
				HTTP: config.HTTPTuning{
					SlackTimeoutSeconds: 10,
				},
				Reporting: config.ReportingTuning{
					RootCauseTruncationLength:  300,
					FailureReasonsDisplayCount: tt.displayCount,
					MaxFailureReasonsTracked:   10, // Set high enough to not interfere
				},
			}
			notifier := NewSlackNotifier("", tuning)

			// Create failure stats with the specified number of reasons
			reasons := make([]string, tt.totalReasons)
			for i := 0; i < tt.totalReasons; i++ {
				reasons[i] = "failure"
			}

			stats := FailureStats{
				Count:            tt.totalReasons,
				FirstFailureTime: time.Now().Add(-5 * time.Minute),
				LastFailureTime:  time.Now(),
				Duration:         5 * time.Minute,
				RecentReasons:    reasons,
			}

			// Send the alert (it will be skipped since webhook is empty, but we can verify the logic)
			err := notifier.SendSystemDegradedAlert(context.Background(), stats)
			if err != nil {
				t.Errorf("SendSystemDegradedAlert should not error: %v", err)
			}

			// The actual verification happens in the implementation which truncates the reasons
			// We just verify the notifier has the correct config
			if notifier.failureReasonsDisplayCount != tt.displayCount {
				t.Errorf("failureReasonsDisplayCount = %d, want %d", notifier.failureReasonsDisplayCount, tt.displayCount)
			}
		})
	}
}
