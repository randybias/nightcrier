package bootstrap

import (
	"time"
)

// RetryConfig holds configuration for exponential backoff retry.
type RetryConfig struct {
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	Multiplier     float64
}

// DefaultRetryConfig returns a RetryConfig with sensible defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		InitialBackoff: 5 * time.Second,
		MaxBackoff:     300 * time.Second,
		Multiplier:     2.0,
	}
}

// Backoff calculates the next backoff duration using exponential backoff.
// It returns the new backoff duration capped at MaxBackoff.
func (c RetryConfig) Backoff(current time.Duration) time.Duration {
	if current == 0 {
		return c.InitialBackoff
	}

	next := time.Duration(float64(current) * c.Multiplier)
	if next > c.MaxBackoff {
		next = c.MaxBackoff
	}
	return next
}
