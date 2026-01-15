package bootstrap

import (
	"context"
	"log/slog"
	"time"
)

// BackgroundRetry manages background retry loops for failed bootstrap components.
type BackgroundRetry struct {
	manager     *Manager
	retryConfig RetryConfig
	stopCh      chan struct{}
	doneCh      chan struct{}
}

// NewBackgroundRetry creates a new BackgroundRetry instance.
func NewBackgroundRetry(manager *Manager, config RetryConfig) *BackgroundRetry {
	return &BackgroundRetry{
		manager:     manager,
		retryConfig: config,
		stopCh:      make(chan struct{}),
		doneCh:      make(chan struct{}),
	}
}

// Start begins background retry loops for any failed components.
// It runs until the context is cancelled or Stop() is called.
func (b *BackgroundRetry) Start(ctx context.Context) {
	go b.run(ctx)
}

// Stop signals the background retry to stop and waits for it to finish.
func (b *BackgroundRetry) Stop() {
	close(b.stopCh)
	<-b.doneCh
}

// run is the main background retry loop.
func (b *BackgroundRetry) run(ctx context.Context) {
	defer close(b.doneCh)

	// Track backoff durations for each component
	globalBackoff := time.Duration(0)
	apiKeysBackoff := time.Duration(0)
	clusterBackoffs := make(map[string]time.Duration)

	// Create a ticker for checking status
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("background bootstrap retry stopped (context cancelled)")
			return
		case <-b.stopCh:
			slog.Info("background bootstrap retry stopped")
			return
		case <-ticker.C:
			status := b.manager.GetStatus()

			// If everything is ready, just reset backoffs and continue
			if status.IsReady() {
				// Reset all backoffs when fully recovered (no need to log every tick)
				globalBackoff = 0
				apiKeysBackoff = 0
				clusterBackoffs = make(map[string]time.Duration)
				continue
			}

			// Retry global resources if needed
			if !status.GlobalReady {
				globalBackoff = b.maybeRetryGlobal(ctx, globalBackoff)
			} else {
				globalBackoff = 0
			}

			// Retry API keys if needed (only if global is ready)
			if status.GlobalReady && !status.APIKeysReady {
				apiKeysBackoff = b.maybeRetryAPIKeys(ctx, apiKeysBackoff)
			} else if status.APIKeysReady {
				apiKeysBackoff = 0
			}

			// Retry failed clusters (only if global is ready)
			if status.GlobalReady {
				for _, cs := range status.DegradedClusters() {
					backoff := clusterBackoffs[cs.Name]
					clusterBackoffs[cs.Name] = b.maybeRetryCluster(ctx, cs.Name, backoff)
				}
				// Clear backoffs for recovered clusters
				for name := range clusterBackoffs {
					found := false
					for _, cs := range status.DegradedClusters() {
						if cs.Name == name {
							found = true
							break
						}
					}
					if !found {
						delete(clusterBackoffs, name)
					}
				}
			}
		}
	}
}

// maybeRetryGlobal retries global resources if the backoff has elapsed.
// Returns the new backoff duration.
func (b *BackgroundRetry) maybeRetryGlobal(ctx context.Context, currentBackoff time.Duration) time.Duration {
	// Calculate next backoff
	nextBackoff := b.retryConfig.Backoff(currentBackoff)

	// Check if we should retry now
	status := b.manager.GetStatus()
	if status.LastUpdated.Add(currentBackoff).After(time.Now()) {
		return currentBackoff // Still waiting
	}

	slog.Info("retrying global bootstrap",
		"backoff", nextBackoff)

	if err := b.manager.RetryGlobal(ctx); err != nil {
		slog.Warn("global bootstrap retry failed",
			"error", err,
			"next_retry", nextBackoff)
		return nextBackoff
	}

	return 0 // Success, reset backoff
}

// maybeRetryAPIKeys retries API keys bootstrap if the backoff has elapsed.
// Returns the new backoff duration.
func (b *BackgroundRetry) maybeRetryAPIKeys(ctx context.Context, currentBackoff time.Duration) time.Duration {
	// Calculate next backoff
	nextBackoff := b.retryConfig.Backoff(currentBackoff)

	// Check if we should retry now
	status := b.manager.GetStatus()
	if status.LastUpdated.Add(currentBackoff).After(time.Now()) {
		return currentBackoff // Still waiting
	}

	slog.Info("retrying API keys bootstrap",
		"backoff", nextBackoff)

	if err := b.manager.RetryAPIKeys(ctx); err != nil {
		slog.Warn("API keys bootstrap retry failed",
			"error", err,
			"next_retry", nextBackoff)
		return nextBackoff
	}

	return 0 // Success, reset backoff
}

// maybeRetryCluster retries a cluster's bootstrap if the backoff has elapsed.
// Returns the new backoff duration.
func (b *BackgroundRetry) maybeRetryCluster(ctx context.Context, clusterName string, currentBackoff time.Duration) time.Duration {
	// Calculate next backoff
	nextBackoff := b.retryConfig.Backoff(currentBackoff)

	// Check if we should retry now
	status := b.manager.GetStatus()
	cs := status.ClusterStatuses[clusterName]
	if cs == nil {
		return 0
	}

	if cs.LastRetry.Add(currentBackoff).After(time.Now()) {
		return currentBackoff // Still waiting
	}

	slog.Info("retrying cluster bootstrap",
		"cluster", clusterName,
		"backoff", nextBackoff)

	if err := b.manager.RetryCluster(ctx, clusterName); err != nil {
		slog.Warn("cluster bootstrap retry failed",
			"cluster", clusterName,
			"error", err,
			"next_retry", nextBackoff)
		return nextBackoff
	}

	return 0 // Success, reset backoff
}

// StartBackgroundRetry is a convenience function to start background retry.
// It creates a BackgroundRetry instance and starts it.
// Returns the BackgroundRetry instance for later stopping.
func StartBackgroundRetry(ctx context.Context, manager *Manager, config RetryConfig) *BackgroundRetry {
	br := NewBackgroundRetry(manager, config)
	br.Start(ctx)
	return br
}
