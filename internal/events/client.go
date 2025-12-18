package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	// LoggerFaults is the logger name for fault events from kubernetes-mcp-server
	LoggerFaults = "kubernetes/faults"
)

// Client handles MCP connections to receive fault events from kubernetes-mcp-server
type Client struct {
	endpoint       string
	mcpClient      *mcp.Client
	session        *mcp.ClientSession
	eventChan      chan *FaultEvent
	subscriptionID string
	mu             sync.Mutex
}

// NewClient creates a new MCP client for the given endpoint
// endpoint should be the base URL of the MCP server (e.g., "http://localhost:8383")
func NewClient(endpoint string) *Client {
	eventChan := make(chan *FaultEvent, 100)

	c := &Client{
		endpoint:  endpoint,
		eventChan: eventChan,
	}

	// Create MCP client with logging message handler to receive fault notifications
	c.mcpClient = mcp.NewClient(
		&mcp.Implementation{
			Name:    "kubernetes-mcp-alerts-event-runner",
			Version: "0.1.0",
		},
		&mcp.ClientOptions{
			LoggingMessageHandler: c.handleLoggingMessage,
		},
	)

	return c
}

// handleLoggingMessage processes MCP log notifications
// Fault events come as log messages with logger="kubernetes/faults"
func (c *Client) handleLoggingMessage(ctx context.Context, req *mcp.LoggingMessageRequest) {
	params := req.Params

	// Only process fault events
	if params.Logger != LoggerFaults {
		slog.Debug("ignoring non-fault log message", "logger", params.Logger)
		return
	}

	slog.Debug("received fault notification", "level", params.Level, "logger", params.Logger)

	// Parse the fault event from the log data
	faultEvent, err := parseFaultEvent(params.Data)
	if err != nil {
		slog.Error("failed to parse fault event", "error", err)
		return
	}

	slog.Info("received fault event",
		"cluster", faultEvent.Cluster,
		"namespace", faultEvent.GetNamespace(),
		"resource", fmt.Sprintf("%s/%s", faultEvent.GetResourceKind(), faultEvent.GetResourceName()),
		"reason", faultEvent.Event.Reason,
		"message", faultEvent.Event.Message)

	// Send to channel (non-blocking)
	select {
	case c.eventChan <- faultEvent:
	default:
		slog.Warn("event channel full, dropping event",
			"cluster", faultEvent.Cluster,
			"resource", faultEvent.GetResourceName())
	}
}

// parseFaultEvent converts the log data to a FaultEvent
func parseFaultEvent(data any) (*FaultEvent, error) {
	// The data comes as a map or can be marshaled to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal log data: %w", err)
	}

	var faultEvent FaultEvent
	if err := json.Unmarshal(jsonData, &faultEvent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal fault event: %w", err)
	}

	return &faultEvent, nil
}

// Subscribe connects to the MCP server, sets logging level, subscribes to faults,
// and returns a channel of FaultEvents
func (c *Client) Subscribe(ctx context.Context) (<-chan *FaultEvent, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Create Streamable HTTP transport - connect to /mcp endpoint
	mcpEndpoint := c.endpoint + "/mcp"
	transport := &mcp.StreamableClientTransport{
		Endpoint:   mcpEndpoint,
		HTTPClient: &http.Client{},
	}

	slog.Info("connecting to MCP server", "endpoint", mcpEndpoint)

	// Connect to server
	session, err := c.mcpClient.Connect(ctx, transport, nil)
	if err != nil {
		close(c.eventChan)
		return nil, fmt.Errorf("failed to connect to MCP server: %w", err)
	}
	c.session = session

	slog.Info("connected to MCP server", "session_id", session.ID())

	// Set logging level to receive notifications
	err = session.SetLoggingLevel(ctx, &mcp.SetLoggingLevelParams{
		Level: mcp.LoggingLevel("info"),
	})
	if err != nil {
		c.session.Close()
		close(c.eventChan)
		return nil, fmt.Errorf("failed to set logging level: %w", err)
	}

	slog.Info("subscribing to fault events")

	// Subscribe to fault events using the events_subscribe tool
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "events_subscribe",
		Arguments: map[string]any{
			"mode": "faults",
		},
	})
	if err != nil {
		c.session.Close()
		close(c.eventChan)
		return nil, fmt.Errorf("failed to subscribe to events: %w", err)
	}

	// Log the subscription result and check for errors
	var responseText string
	for _, content := range result.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			responseText = textContent.Text
			slog.Info("subscription response", "text", textContent.Text)
		}
	}

	// Extract subscription ID from result
	if result.IsError {
		c.session.Close()
		close(c.eventChan)
		return nil, fmt.Errorf("events_subscribe returned error: %s", responseText)
	}

	slog.Info("subscribed to fault events, waiting for notifications...")

	// Keep the session alive to receive notifications
	// Wait() blocks until the session is closed
	go func() {
		if err := c.session.Wait(); err != nil {
			slog.Error("MCP session error", "error", err)
		}
		slog.Info("MCP session ended")
		c.Close()
	}()

	// Handle context cancellation
	go func() {
		<-ctx.Done()
		slog.Info("context cancelled, closing MCP session")
		c.Close()
	}()

	return c.eventChan, nil
}

// Close closes the MCP session and event channel
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.session != nil {
		c.session.Close()
		c.session = nil
	}

	// Close channel if not already closed
	select {
	case _, ok := <-c.eventChan:
		if ok {
			close(c.eventChan)
		}
	default:
		close(c.eventChan)
	}
}
