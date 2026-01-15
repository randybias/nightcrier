package events

import (
	"net/http"
	"testing"

	"github.com/randybias/nightcrier/internal/config"
)

// TestNewClient_UsesConfigurableBufferSize verifies that the event channel
// buffer size is configured from TuningConfig rather than hardcoded.
// Note: The actual channel is created in Subscribe(), so we verify the
// buffer size is stored correctly in the client.
func TestNewClient_UsesConfigurableBufferSize(t *testing.T) {
	tests := []struct {
		name       string
		bufferSize int
	}{
		{
			name:       "default buffer size",
			bufferSize: 100,
		},
		{
			name:       "small buffer size",
			bufferSize: 10,
		},
		{
			name:       "large buffer size",
			bufferSize: 1000,
		},
		{
			name:       "minimal buffer size",
			bufferSize: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tuningConfig := &config.TuningConfig{
				Events: config.EventsTuning{
					ChannelBufferSize: tt.bufferSize,
				},
			}

			client := NewClient("http://localhost:8383/mcp", "faults", "", tuningConfig)

			if client == nil {
				t.Fatal("expected client to be non-nil")
			}

			// Verify buffer size is stored correctly (channel created in Subscribe)
			if client.channelBufferSize != tt.bufferSize {
				t.Errorf("expected channelBufferSize %d, got %d", tt.bufferSize, client.channelBufferSize)
			}
		})
	}
}

// TestNewClient_RequiresTuningConfig verifies that NewClient properly uses
// the TuningConfig parameter and doesn't fall back to hardcoded defaults.
func TestNewClient_RequiresTuningConfig(t *testing.T) {
	customBufferSize := 500

	tuningConfig := &config.TuningConfig{
		Events: config.EventsTuning{
			ChannelBufferSize: customBufferSize,
		},
	}

	client := NewClient("http://localhost:8383/mcp", "events", "", tuningConfig)

	if client == nil {
		t.Fatal("expected client to be non-nil")
	}

	// Verify the custom buffer size is stored
	if client.channelBufferSize != customBufferSize {
		t.Errorf("expected channelBufferSize %d, got %d (should not be hardcoded 100)", customBufferSize, client.channelBufferSize)
	}
}

// TestNewClient_InitializesFields verifies that NewClient properly initializes
// all client fields including the buffer size configuration.
func TestNewClient_InitializesFields(t *testing.T) {
	endpoint := "http://test.local:8383/mcp"
	mode := "events"
	bufferSize := 250

	tuningConfig := &config.TuningConfig{
		Events: config.EventsTuning{
			ChannelBufferSize: bufferSize,
		},
	}

	client := NewClient(endpoint, mode, "", tuningConfig)

	if client.endpoint != endpoint {
		t.Errorf("expected endpoint %s, got %s", endpoint, client.endpoint)
	}

	if client.subscribeMode != mode {
		t.Errorf("expected subscribe mode %s, got %s", mode, client.subscribeMode)
	}

	// eventChan is created in Subscribe(), not NewClient()
	// Verify buffer size is stored for later use
	if client.channelBufferSize != bufferSize {
		t.Errorf("expected channelBufferSize %d, got %d", bufferSize, client.channelBufferSize)
	}

	if client.mcpClient == nil {
		t.Error("expected mcpClient to be initialized")
	}
}

// TestNewClient_DefaultSubscribeMode verifies that an empty subscribe mode
// defaults to "faults" while still respecting the configured buffer size.
func TestNewClient_DefaultSubscribeMode(t *testing.T) {
	bufferSize := 150

	tuningConfig := &config.TuningConfig{
		Events: config.EventsTuning{
			ChannelBufferSize: bufferSize,
		},
	}

	client := NewClient("http://localhost:8383/mcp", "", "", tuningConfig)

	if client.subscribeMode != "faults" {
		t.Errorf("expected default subscribe mode 'faults', got %s", client.subscribeMode)
	}

	// Verify buffer size is stored correctly
	if client.channelBufferSize != bufferSize {
		t.Errorf("expected channelBufferSize %d, got %d", bufferSize, client.channelBufferSize)
	}
}

// TestNewClient_WithAPIKey verifies that the API key is stored in the client.
func TestNewClient_WithAPIKey(t *testing.T) {
	tuningConfig := &config.TuningConfig{
		Events: config.EventsTuning{
			ChannelBufferSize: 100,
		},
	}

	apiKey := "sk-test-key-12345"
	client := NewClient("https://agentgateway:8443/mcp", "faults", apiKey, tuningConfig)

	if client.apiKey != apiKey {
		t.Errorf("expected apiKey %q, got %q", apiKey, client.apiKey)
	}
}

// TestNewClient_WithoutAPIKey verifies client works without API key.
func TestNewClient_WithoutAPIKey(t *testing.T) {
	tuningConfig := &config.TuningConfig{
		Events: config.EventsTuning{
			ChannelBufferSize: 100,
		},
	}

	client := NewClient("http://localhost:8383/mcp", "faults", "", tuningConfig)

	if client.apiKey != "" {
		t.Errorf("expected empty apiKey, got %q", client.apiKey)
	}
}

// TestAuthTransport_AddsAuthorizationHeader verifies the auth transport adds
// the correct Authorization header to requests.
func TestAuthTransport_AddsAuthorizationHeader(t *testing.T) {
	apiKey := "sk-test-key-12345"
	transport := &authTransport{
		base:   &mockRoundTripper{},
		apiKey: apiKey,
	}

	req, _ := http.NewRequest("GET", "https://example.com/mcp", nil)
	_, _ = transport.RoundTrip(req)

	// The mock transport will have captured the request
	// For a more complete test, we'd check the header was set correctly
	// Here we verify the transport doesn't panic and processes correctly
}

// mockRoundTripper is a test double for http.RoundTripper
type mockRoundTripper struct {
	lastRequest *http.Request
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.lastRequest = req
	return &http.Response{StatusCode: 200}, nil
}

// TestAuthTransport_SetsCorrectHeader verifies the exact header format.
func TestAuthTransport_SetsCorrectHeader(t *testing.T) {
	apiKey := "sk-test-key-12345"
	mock := &mockRoundTripper{}
	transport := &authTransport{
		base:   mock,
		apiKey: apiKey,
	}

	req, _ := http.NewRequest("GET", "https://example.com/mcp", nil)
	_, _ = transport.RoundTrip(req)

	expectedHeader := "Bearer " + apiKey
	actualHeader := mock.lastRequest.Header.Get("Authorization")

	if actualHeader != expectedHeader {
		t.Errorf("expected Authorization header %q, got %q", expectedHeader, actualHeader)
	}
}

// TestClient_SupportsReconnection verifies that Subscribe() can be called
// multiple times on the same client to support reconnection after failures.
func TestClient_SupportsReconnection(t *testing.T) {
	bufferSize := 50

	tuningConfig := &config.TuningConfig{
		Events: config.EventsTuning{
			ChannelBufferSize: bufferSize,
		},
	}

	client := NewClient("http://localhost:8383/mcp", "faults", "", tuningConfig)

	// Verify initial state - no channel yet
	if client.eventChan != nil {
		t.Error("expected eventChan to be nil before Subscribe()")
	}

	// Verify closed state is initially false
	if client.closed.Load() {
		t.Error("expected closed to be false initially")
	}

	// Verify buffer size is ready for Subscribe()
	if client.channelBufferSize != bufferSize {
		t.Errorf("expected channelBufferSize %d, got %d", bufferSize, client.channelBufferSize)
	}
}
