package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rbias/nightcrier/internal/config"
)

const (
	// LoggerPrefix is the prefix for all kubernetes-mcp-server loggers
	LoggerPrefix = "kubernetes/"
)

// Client handles MCP connections to receive fault events from kubernetes-mcp-server
type Client struct {
	endpoint       string
	subscribeMode  string // "events" or "faults"
	mcpClient      *mcp.Client
	session        *mcp.ClientSession
	eventChan      chan *FaultEvent
	subscriptionID string
	mu             sync.Mutex
}

// NewClient creates a new MCP client for the given endpoint
// endpoint should be the full MCP endpoint URL (e.g., "http://localhost:8383/mcp")
// subscribeMode should be "events" or "faults" (default: "faults")
// tuningConfig provides tunable operational parameters, including event channel buffer size
func NewClient(endpoint, subscribeMode string, tuningConfig *config.TuningConfig) *Client {
	if subscribeMode == "" {
		subscribeMode = "faults"
	}
	eventChan := make(chan *FaultEvent, tuningConfig.Events.ChannelBufferSize)

	c := &Client{
		endpoint:      endpoint,
		subscribeMode: subscribeMode,
		eventChan:     eventChan,
	}

	// Create MCP client with logging message handler to receive fault notifications
	c.mcpClient = mcp.NewClient(
		&mcp.Implementation{
			Name:    "nightcrier",
			Version: "1.0.0",
		},
		&mcp.ClientOptions{
			LoggingMessageHandler: c.handleLoggingMessage,
		},
	)

	return c
}

// handleLoggingMessage processes MCP log notifications
// Fault events come as log messages with logger="kubernetes/{mode}" based on subscribe mode
func (c *Client) handleLoggingMessage(ctx context.Context, req *mcp.LoggingMessageRequest) {
	params := req.Params

	// Expected logger name is "kubernetes/{subscribeMode}"
	expectedLogger := LoggerPrefix + c.subscribeMode

	// Only process events matching our subscription mode
	if params.Logger != expectedLogger {
		slog.Debug("ignoring non-matching log message", "logger", params.Logger, "expected", expectedLogger)
		return
	}

	slog.Debug("received fault notification", "level", params.Level, "logger", params.Logger)

	// Log raw data for debugging
	if rawJSON, err := json.Marshal(params.Data); err == nil {
		slog.Debug("raw MCP data", "data", string(rawJSON))
	}

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
		"reason", faultEvent.GetReason(),
		"message", faultEvent.GetContext())

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

	// Generate EventID and set ReceivedAt on receipt
	faultEvent.EventID = uuid.New().String()
	faultEvent.ReceivedAt = time.Now()

	return &faultEvent, nil
}

// Subscribe connects to the MCP server, sets logging level, subscribes to faults,
// and returns a channel of FaultEvents
func (c *Client) Subscribe(ctx context.Context) (<-chan *FaultEvent, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Create Streamable HTTP transport using the configured endpoint as-is
	transport := &mcp.StreamableClientTransport{
		Endpoint:   c.endpoint,
		HTTPClient: &http.Client{},
	}

	slog.Info("connecting to MCP server", "endpoint", c.endpoint)

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

	slog.Info("subscribing to events", "mode", c.subscribeMode)

	// Subscribe to events using the events_subscribe tool
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "events_subscribe",
		Arguments: map[string]any{
			"mode": c.subscribeMode,
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
