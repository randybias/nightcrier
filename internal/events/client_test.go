package events

import (
	"testing"

	"github.com/rbias/nightcrier/internal/config"
)

// TestNewClient_UsesConfigurableBufferSize verifies that the event channel
// buffer size is configured from TuningConfig rather than hardcoded.
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

			client := NewClient("http://localhost:8383/mcp", "faults", tuningConfig)

			if client == nil {
				t.Fatal("expected client to be non-nil")
			}

			if client.eventChan == nil {
				t.Fatal("expected eventChan to be initialized")
			}

			// Verify channel capacity matches configured buffer size
			actualCapacity := cap(client.eventChan)
			if actualCapacity != tt.bufferSize {
				t.Errorf("expected channel buffer size %d, got %d", tt.bufferSize, actualCapacity)
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

	client := NewClient("http://localhost:8383/mcp", "events", tuningConfig)

	if client == nil {
		t.Fatal("expected client to be non-nil")
	}

	// Verify the custom buffer size is used
	actualCapacity := cap(client.eventChan)
	if actualCapacity != customBufferSize {
		t.Errorf("expected channel capacity %d, got %d (should not be hardcoded 100)", customBufferSize, actualCapacity)
	}
}

// TestNewClient_InitializesFields verifies that NewClient properly initializes
// all client fields including the event channel with configured size.
func TestNewClient_InitializesFields(t *testing.T) {
	endpoint := "http://test.local:8383/mcp"
	mode := "events"
	bufferSize := 250

	tuningConfig := &config.TuningConfig{
		Events: config.EventsTuning{
			ChannelBufferSize: bufferSize,
		},
	}

	client := NewClient(endpoint, mode, tuningConfig)

	if client.endpoint != endpoint {
		t.Errorf("expected endpoint %s, got %s", endpoint, client.endpoint)
	}

	if client.subscribeMode != mode {
		t.Errorf("expected subscribe mode %s, got %s", mode, client.subscribeMode)
	}

	if client.eventChan == nil {
		t.Fatal("expected eventChan to be initialized")
	}

	if cap(client.eventChan) != bufferSize {
		t.Errorf("expected channel capacity %d, got %d", bufferSize, cap(client.eventChan))
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

	client := NewClient("http://localhost:8383/mcp", "", tuningConfig)

	if client.subscribeMode != "faults" {
		t.Errorf("expected default subscribe mode 'faults', got %s", client.subscribeMode)
	}

	if cap(client.eventChan) != bufferSize {
		t.Errorf("expected channel capacity %d, got %d", bufferSize, cap(client.eventChan))
	}
}
