package nats

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
)

// Client wraps a NATS connection and provides methods for publishing progress events.
// This is a fire-and-forget client - errors are logged but don't fail operations.
type Client struct {
	conn *nats.Conn
}

// Option is a functional option for configuring the NATS client
type Option func(*nats.Options)

// WithName sets the connection name for the NATS client
func WithName(name string) Option {
	return func(opts *nats.Options) {
		opts.Name = name
	}
}

// WithTimeout sets the connection timeout for the NATS client
func WithTimeout(timeout time.Duration) Option {
	return func(opts *nats.Options) {
		opts.Timeout = timeout
	}
}

// WithReconnectWait sets the reconnect wait time
func WithReconnectWait(wait time.Duration) Option {
	return func(opts *nats.Options) {
		opts.ReconnectWait = wait
	}
}

// WithMaxReconnects sets the maximum number of reconnect attempts
func WithMaxReconnects(max int) Option {
	return func(opts *nats.Options) {
		opts.MaxReconnect = max
	}
}

// Connect creates a new NATS client and connects to the server.
// The token parameter is used for authentication. If empty, no auth is used.
// This is a fire-and-forget client - connection errors are handled gracefully.
func Connect(server, token string, options ...Option) (*Client, error) {
	if server == "" {
		return nil, fmt.Errorf("NATS server address is required")
	}

	// Build NATS options
	opts := []nats.Option{
		nats.Name("nightcrier-nats-client"),
		nats.Timeout(5 * time.Second),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(5),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				slog.Warn("NATS disconnected", "error", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			slog.Info("NATS reconnected", "server", nc.ConnectedUrl())
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			slog.Warn("NATS error", "error", err, "subject", sub.Subject)
		}),
	}

	// Add token authentication if provided
	if token != "" {
		opts = append(opts, nats.Token(token))
	}

	// Apply custom options
	natsOpts := nats.GetDefaultOptions()
	for _, opt := range options {
		opt(&natsOpts)
	}

	// Convert functional options to nats.Option
	for _, opt := range opts {
		opt(&natsOpts)
	}

	// Connect to NATS server
	slog.Info("connecting to NATS server", "server", server)
	conn, err := nats.Connect(server, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS server: %w", err)
	}

	slog.Info("connected to NATS server", "server", conn.ConnectedUrl())

	return &Client{conn: conn}, nil
}

// Publish publishes a progress event to the specified NATS subject.
// This is a fire-and-forget operation with a 3-second timeout.
// Errors are logged but don't fail the operation.
func (c *Client) Publish(subject string, data []byte) error {
	if c.conn == nil {
		slog.Warn("NATS client not connected, skipping publish", "subject", subject)
		return fmt.Errorf("NATS client not connected")
	}

	// Create a context with 3-second timeout for publish
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Use NATS request pattern with timeout instead of basic publish
	// This ensures we don't block if NATS is slow
	done := make(chan error, 1)
	go func() {
		err := c.conn.Publish(subject, data)
		if err != nil {
			done <- err
			return
		}
		// Ensure the message is sent by flushing with timeout
		err = c.conn.FlushWithContext(ctx)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			slog.Warn("failed to publish to NATS", "subject", subject, "error", err)
			return fmt.Errorf("publish failed: %w", err)
		}
		slog.Debug("published to NATS", "subject", subject, "bytes", len(data))
		return nil
	case <-ctx.Done():
		slog.Warn("NATS publish timeout", "subject", subject)
		return fmt.Errorf("publish timeout after 3 seconds")
	}
}

// PublishEvent is a convenience method that marshals a ProgressEvent and publishes it
func (c *Client) PublishEvent(event *ProgressEvent) error {
	// Marshal event to JSON
	data, err := event.ToJSON()
	if err != nil {
		slog.Warn("failed to marshal progress event", "error", err)
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Construct subject from event
	subject := SubjectForEvent(event.IncidentID, EventType(event.EventType))

	// Publish with fire-and-forget semantics
	return c.Publish(subject, data)
}

// Close gracefully closes the NATS connection
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
		slog.Info("NATS client closed")
	}
}

// IsConnected returns true if the client is currently connected to NATS
func (c *Client) IsConnected() bool {
	return c.conn != nil && c.conn.IsConnected()
}

// Stats returns connection statistics for monitoring
func (c *Client) Stats() nats.Statistics {
	if c.conn == nil {
		return nats.Statistics{}
	}
	return c.conn.Stats()
}
