package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/r3labs/sse/v2"
)

// Client handles SSE connections to receive fault events
type Client struct {
	endpoint string
	client   *sse.Client
}

// NewClient creates a new SSE client for the given endpoint
func NewClient(endpoint string) *Client {
	sseClient := sse.NewClient(endpoint)
	return &Client{
		endpoint: endpoint,
		client:   sseClient,
	}
}

// Subscribe connects to the SSE endpoint and returns a channel of FaultEvents
func (c *Client) Subscribe(ctx context.Context) (<-chan *FaultEvent, error) {
	eventChan := make(chan *FaultEvent, 10)

	// Subscribe to SSE events
	events := make(chan *sse.Event)
	err := c.client.SubscribeChanWithContext(ctx, "", events)
	if err != nil {
		close(eventChan)
		slog.Error("failed to subscribe to SSE endpoint",
			"endpoint", c.endpoint,
			"error", err)
		return nil, fmt.Errorf("failed to subscribe to SSE endpoint: %w", err)
	}

	// Process SSE events in a goroutine
	go func() {
		defer close(eventChan)

		for {
			select {
			case <-ctx.Done():
				slog.Info("SSE subscription context cancelled")
				return
			case event, ok := <-events:
				if !ok {
					slog.Info("SSE event channel closed")
					return
				}

				// Parse the SSE data field as JSON
				var faultEvent FaultEvent
				if err := json.Unmarshal(event.Data, &faultEvent); err != nil {
					slog.Error("failed to parse fault event",
						"error", err,
						"data", string(event.Data))
					continue
				}

				// Log the received event
				slog.Info("received fault event",
					"cluster_id", faultEvent.ClusterID,
					"namespace", faultEvent.Namespace,
					"resource_type", faultEvent.ResourceType,
					"resource_name", faultEvent.ResourceName,
					"severity", faultEvent.Severity,
					"message", faultEvent.Message,
					"timestamp", faultEvent.Timestamp)

				// Send the event to the channel
				select {
				case eventChan <- &faultEvent:
				case <-ctx.Done():
					slog.Info("context cancelled while sending event")
					return
				}
			}
		}
	}()

	return eventChan, nil
}
