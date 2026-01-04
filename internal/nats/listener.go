package nats

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/randybias/nightcrier/internal/storage"
)

// Listener subscribes to NATS progress events and updates the StateStore.
// It listens to all triage events using the "triage.>" wildcard subject.
type Listener struct {
	client     *Client
	store      storage.StateStore
	sub        *nats.Subscription
	cancelFunc context.CancelFunc
}

// NewListener creates a new NATS listener with a reference to the StateStore.
// The listener will subscribe to all triage events and update execution activity.
func NewListener(client *Client, store storage.StateStore) *Listener {
	return &Listener{
		client: client,
		store:  store,
	}
}

// Start begins listening for progress events on the "triage.>" subject.
// It creates a buffered subscription (1000 messages) and processes events asynchronously.
// The listener runs until the context is cancelled.
func (l *Listener) Start(ctx context.Context) error {
	if l.client == nil || !l.client.IsConnected() {
		return fmt.Errorf("NATS client is not connected")
	}

	// Create a cancellable context for internal goroutines
	ctx, l.cancelFunc = context.WithCancel(ctx)

	// Subscribe to all triage events with a buffered channel
	subject := SubjectWildcard()
	slog.Info("subscribing to NATS events", "subject", subject)

	// Create a buffered channel for incoming messages
	msgChan := make(chan *nats.Msg, 1000)

	// Subscribe using ChanSubscribe for better control
	sub, err := l.client.conn.ChanSubscribe(subject, msgChan)
	if err != nil {
		return fmt.Errorf("failed to subscribe to NATS: %w", err)
	}
	l.sub = sub

	slog.Info("NATS listener started", "subject", subject)

	// Start message processing goroutine
	go l.processMessages(ctx, msgChan)

	// Wait for context cancellation
	<-ctx.Done()
	return l.shutdown()
}

// processMessages handles incoming NATS messages from the channel
func (l *Listener) processMessages(ctx context.Context, msgChan chan *nats.Msg) {
	for {
		select {
		case <-ctx.Done():
			slog.Info("NATS listener context cancelled, stopping message processing")
			return
		case msg := <-msgChan:
			if msg == nil {
				// Channel closed
				return
			}
			l.handleProgressEvent(msg)
		}
	}
}

// handleProgressEvent processes a single progress event message.
// It parses the JSON payload and updates the StateStore for "executing" events.
// Errors are logged but don't stop the listener.
func (l *Listener) handleProgressEvent(msg *nats.Msg) {
	// Parse the progress event from JSON
	event, err := FromJSON(msg.Data)
	if err != nil {
		slog.Warn("failed to parse progress event", "error", err, "subject", msg.Subject)
		return
	}

	slog.Debug("received progress event",
		"incident_id", event.IncidentID,
		"event_type", event.EventType,
		"activity", event.Activity,
		"subject", msg.Subject)

	// If no store is configured, just log events without persisting
	if l.store == nil {
		slog.Debug("no state store configured, skipping persistence",
			"incident_id", event.IncidentID,
			"event_type", event.EventType)
		return
	}

	// Only update StateStore for "executing" events that have activity
	// The StateStore.UpdateExecutionActivity method will be added in Phase 1.2
	if event.EventType == string(EventTypeExecuting) && event.Activity != "" {
		// Parse timestamp from event
		activityTime, err := time.Parse(time.RFC3339, event.Timestamp)
		if err != nil {
			slog.Warn("failed to parse event timestamp, using current time",
				"timestamp", event.Timestamp,
				"error", err)
			activityTime = time.Now()
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = l.store.UpdateExecutionActivity(ctx, event.IncidentID, event.Activity, activityTime)
		if err != nil {
			slog.Warn("failed to update execution activity",
				"incident_id", event.IncidentID,
				"error", err)
		} else {
			slog.Debug("updated execution activity",
				"incident_id", event.IncidentID,
				"activity", event.Activity)
		}
	}

	// Handle run lifecycle events
	switch EventType(event.EventType) {
	case EventTypeRunStarted:
		// Parse timestamp from event
		runStartedAt, err := time.Parse(time.RFC3339, event.Timestamp)
		if err != nil {
			slog.Warn("failed to parse run.started timestamp, using current time",
				"timestamp", event.Timestamp,
				"error", err)
			runStartedAt = time.Now().UTC()
		}

		// Update run_started_at in database
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = l.store.UpdateRunStarted(ctx, event.IncidentID, runStartedAt)
		if err != nil {
			slog.Warn("failed to update run_started_at",
				"incident_id", event.IncidentID,
				"error", err)
		} else {
			slog.Info("triage run started",
				"incident_id", event.IncidentID,
				"cluster", event.Cluster,
				"agent_cli", event.AgentCLI,
				"model", event.Model)
		}

	case EventTypeRunCompleted:
		// Parse timestamp from event
		runCompletedAt, err := time.Parse(time.RFC3339, event.Timestamp)
		if err != nil {
			slog.Warn("failed to parse run.completed timestamp, using current time",
				"timestamp", event.Timestamp,
				"error", err)
			runCompletedAt = time.Now()
		}

		// Extract exit code (default to 1 if not provided)
		runExitCode := 1
		if event.ExitCode != nil {
			runExitCode = *event.ExitCode
		}

		// Update run_completed_at and run_exit_code in database
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = l.store.UpdateRunCompleted(ctx, event.IncidentID, runCompletedAt, runExitCode)
		if err != nil {
			slog.Warn("failed to update run_completed_at",
				"incident_id", event.IncidentID,
				"error", err)
		} else {
			slog.Info("triage run completed",
				"incident_id", event.IncidentID,
				"exit_code", runExitCode)
		}

	case EventTypeError:
		slog.Warn("triage error event",
			"incident_id", event.IncidentID,
			"error_message", event.ErrorMessage)
	}
}

// shutdown gracefully stops the listener and cleans up resources
func (l *Listener) shutdown() error {
	slog.Info("shutting down NATS listener")

	// Unsubscribe from NATS
	if l.sub != nil {
		if err := l.sub.Drain(); err != nil {
			slog.Warn("error draining NATS subscription", "error", err)
		}
		l.sub = nil
	}

	slog.Info("NATS listener stopped")
	return nil
}

// Stop stops the listener (can be called externally)
func (l *Listener) Stop() error {
	if l.cancelFunc != nil {
		l.cancelFunc()
	}
	return nil
}
