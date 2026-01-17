package bootstrap

import (
	"testing"
	"time"
)

func TestRetryConfig_Backoff(t *testing.T) {
	tests := []struct {
		name     string
		config   RetryConfig
		current  time.Duration
		expected time.Duration
	}{
		{
			name:     "zero current returns initial backoff",
			config:   DefaultRetryConfig(),
			current:  0,
			expected: 5 * time.Second,
		},
		{
			name:     "first backoff doubles",
			config:   DefaultRetryConfig(),
			current:  5 * time.Second,
			expected: 10 * time.Second,
		},
		{
			name:     "second backoff doubles again",
			config:   DefaultRetryConfig(),
			current:  10 * time.Second,
			expected: 20 * time.Second,
		},
		{
			name:     "caps at max backoff",
			config:   DefaultRetryConfig(),
			current:  200 * time.Second,
			expected: 300 * time.Second, // 400 would exceed max of 300
		},
		{
			name:     "already at max stays at max",
			config:   DefaultRetryConfig(),
			current:  300 * time.Second,
			expected: 300 * time.Second,
		},
		{
			name: "custom config initial",
			config: RetryConfig{
				InitialBackoff: 1 * time.Second,
				MaxBackoff:     60 * time.Second,
				Multiplier:     3.0,
			},
			current:  0,
			expected: 1 * time.Second,
		},
		{
			name: "custom config multiplier",
			config: RetryConfig{
				InitialBackoff: 1 * time.Second,
				MaxBackoff:     60 * time.Second,
				Multiplier:     3.0,
			},
			current:  1 * time.Second,
			expected: 3 * time.Second,
		},
		{
			name: "custom config caps at max",
			config: RetryConfig{
				InitialBackoff: 1 * time.Second,
				MaxBackoff:     60 * time.Second,
				Multiplier:     3.0,
			},
			current:  30 * time.Second,
			expected: 60 * time.Second, // 90 would exceed max
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.Backoff(tt.current)
			if got != tt.expected {
				t.Errorf("Backoff(%v) = %v, want %v", tt.current, got, tt.expected)
			}
		})
	}
}

func TestRetryConfig_BackoffSequence(t *testing.T) {
	// Test a full sequence of backoffs with default config
	config := DefaultRetryConfig()

	// Expected sequence: 5s, 10s, 20s, 40s, 80s, 160s, 300s (capped), 300s (stays capped)
	expected := []time.Duration{
		5 * time.Second,
		10 * time.Second,
		20 * time.Second,
		40 * time.Second,
		80 * time.Second,
		160 * time.Second,
		300 * time.Second, // capped
		300 * time.Second, // stays capped
	}

	current := time.Duration(0)
	for i, want := range expected {
		got := config.Backoff(current)
		if got != want {
			t.Errorf("iteration %d: Backoff(%v) = %v, want %v", i, current, got, want)
		}
		current = got
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.InitialBackoff != 5*time.Second {
		t.Errorf("InitialBackoff = %v, want 5s", config.InitialBackoff)
	}
	if config.MaxBackoff != 300*time.Second {
		t.Errorf("MaxBackoff = %v, want 300s", config.MaxBackoff)
	}
	if config.Multiplier != 2.0 {
		t.Errorf("Multiplier = %v, want 2.0", config.Multiplier)
	}
}
