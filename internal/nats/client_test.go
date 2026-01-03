package nats

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	natstest "github.com/nats-io/nats-server/v2/test"
)

// startTestServer starts an embedded NATS server for testing
func startTestServer(t *testing.T) (*natsserver.Server, string) {
	t.Helper()

	opts := natstest.DefaultTestOptions
	opts.Port = -1 // Use random port

	s := natstest.RunServer(&opts)
	if s == nil {
		t.Fatal("failed to start test NATS server")
	}

	return s, s.ClientURL()
}

// startTestServerWithAuth starts an embedded NATS server with token authentication
func startTestServerWithAuth(t *testing.T, token string) (*natsserver.Server, string) {
	t.Helper()

	opts := natstest.DefaultTestOptions
	opts.Port = -1 // Use random port
	opts.Authorization = token

	s := natstest.RunServer(&opts)
	if s == nil {
		t.Fatal("failed to start test NATS server with auth")
	}

	return s, s.ClientURL()
}

// TestConnect_Success verifies successful connection to NATS server
func TestConnect_Success(t *testing.T) {
	s, url := startTestServer(t)
	defer s.Shutdown()

	client, err := Connect(url, "")
	if err != nil {
		t.Fatalf("expected successful connection, got error: %v", err)
	}
	defer client.Close()

	if !client.IsConnected() {
		t.Error("expected client to be connected")
	}
}

// TestConnect_WithAuth verifies connection with token authentication
func TestConnect_WithAuth(t *testing.T) {
	token := "test-token-123"
	s, url := startTestServerWithAuth(t, token)
	defer s.Shutdown()

	// Test with correct token
	client, err := Connect(url, token)
	if err != nil {
		t.Fatalf("expected successful connection with auth, got error: %v", err)
	}
	defer client.Close()

	if !client.IsConnected() {
		t.Error("expected client to be connected")
	}
}

// TestConnect_WithAuthFailure verifies connection fails with wrong token
func TestConnect_WithAuthFailure(t *testing.T) {
	token := "correct-token"
	s, url := startTestServerWithAuth(t, token)
	defer s.Shutdown()

	// Test with wrong token
	_, err := Connect(url, "wrong-token")
	if err == nil {
		t.Fatal("expected connection to fail with wrong token")
	}
}

// TestConnect_EmptyServer verifies error when server address is empty
func TestConnect_EmptyServer(t *testing.T) {
	_, err := Connect("", "")
	if err == nil {
		t.Fatal("expected error for empty server address")
	}

	expectedMsg := "NATS server address is required"
	if err.Error() != expectedMsg {
		t.Errorf("expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}

// TestPublish_Success verifies successful message publishing
func TestPublish_Success(t *testing.T) {
	s, url := startTestServer(t)
	defer s.Shutdown()

	client, err := Connect(url, "")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Subscribe to verify message was published
	sub, err := client.conn.SubscribeSync("test.subject")
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	// Publish a message
	testData := []byte(`{"test": "data"}`)
	err = client.Publish("test.subject", testData)
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	// Verify message received
	msg, err := sub.NextMsg(2 * time.Second)
	if err != nil {
		t.Fatalf("failed to receive message: %v", err)
	}

	if string(msg.Data) != string(testData) {
		t.Errorf("expected data '%s', got '%s'", testData, msg.Data)
	}
}

// TestPublishEvent_Success verifies successful event publishing
func TestPublishEvent_Success(t *testing.T) {
	s, url := startTestServer(t)
	defer s.Shutdown()

	client, err := Connect(url, "")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Subscribe to verify event was published
	incidentID := "inc-test-123"
	subject := SubjectForEvent(incidentID, EventTypeRunStarted)
	sub, err := client.conn.SubscribeSync(subject)
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	// Create and publish an event
	event := NewProgressEvent(incidentID, "test-cluster", EventTypeRunStarted).
		WithAgentInfo("claude", "sonnet")

	err = client.PublishEvent(event)
	if err != nil {
		t.Fatalf("failed to publish event: %v", err)
	}

	// Verify event received
	msg, err := sub.NextMsg(2 * time.Second)
	if err != nil {
		t.Fatalf("failed to receive message: %v", err)
	}

	// Parse received event
	var receivedEvent ProgressEvent
	err = json.Unmarshal(msg.Data, &receivedEvent)
	if err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if receivedEvent.IncidentID != incidentID {
		t.Errorf("expected incident_id '%s', got '%s'", incidentID, receivedEvent.IncidentID)
	}
	if receivedEvent.Cluster != "test-cluster" {
		t.Errorf("expected cluster 'test-cluster', got '%s'", receivedEvent.Cluster)
	}
	if receivedEvent.EventType != string(EventTypeRunStarted) {
		t.Errorf("expected event_type '%s', got '%s'", EventTypeRunStarted, receivedEvent.EventType)
	}
}

// TestListener_ReceivesEvents verifies listener receives and processes events
func TestListener_ReceivesEvents(t *testing.T) {
	s, url := startTestServer(t)
	defer s.Shutdown()

	client, err := Connect(url, "")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Create a mock StateStore (nil is acceptable since we're not calling UpdateExecutionActivity yet)
	listener := NewListener(client, nil)

	// Start listener in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- listener.Start(ctx)
	}()

	// Wait a bit for subscription to be active
	time.Sleep(100 * time.Millisecond)

	// Publish a test event
	incidentID := "inc-test-456"
	event := NewProgressEvent(incidentID, "test-cluster", EventTypeExecuting).
		WithActivity("kubectl get pods")

	err = client.PublishEvent(event)
	if err != nil {
		t.Fatalf("failed to publish event: %v", err)
	}

	// Wait a bit for event to be processed
	time.Sleep(100 * time.Millisecond)

	// Stop listener
	cancel()

	// Wait for listener to stop
	select {
	case err := <-errChan:
		if err != nil && err != context.Canceled {
			t.Errorf("listener returned unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("listener did not stop within timeout")
	}
}

// TestListener_HandlesMultipleEvents verifies listener handles concurrent events
func TestListener_HandlesMultipleEvents(t *testing.T) {
	s, url := startTestServer(t)
	defer s.Shutdown()

	client, err := Connect(url, "")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	listener := NewListener(client, nil)

	// Start listener
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go listener.Start(ctx)

	// Wait for subscription to be active
	time.Sleep(100 * time.Millisecond)

	// Publish multiple events
	eventTypes := []EventType{
		EventTypeRunStarted,
		EventTypeExecuting,
		EventTypeExecuting,
		EventTypeRunCompleted,
	}

	for i, eventType := range eventTypes {
		incidentID := "inc-multi-test"
		event := NewProgressEvent(incidentID, "test-cluster", eventType)

		if eventType == EventTypeExecuting {
			event.WithActivity("test activity")
		} else if eventType == EventTypeRunCompleted {
			exitCode := 0
			event.WithExitCode(exitCode)
		}

		err = client.PublishEvent(event)
		if err != nil {
			t.Fatalf("failed to publish event %d: %v", i, err)
		}
	}

	// Wait for events to be processed
	time.Sleep(200 * time.Millisecond)

	// Verify stats
	stats := client.Stats()
	if stats.OutMsgs < uint64(len(eventTypes)) {
		t.Errorf("expected at least %d outgoing messages, got %d", len(eventTypes), stats.OutMsgs)
	}
}

// TestListener_HandlesMalformedJSON verifies listener handles malformed events gracefully
func TestListener_HandlesMalformedJSON(t *testing.T) {
	s, url := startTestServer(t)
	defer s.Shutdown()

	client, err := Connect(url, "")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	listener := NewListener(client, nil)

	// Start listener
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go listener.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// Publish malformed JSON
	malformedData := []byte(`{"invalid": json}`)
	err = client.Publish("triage.inc-test.executing", malformedData)
	if err != nil {
		t.Fatalf("failed to publish malformed data: %v", err)
	}

	// Publish a valid event after to ensure listener still works
	event := NewProgressEvent("inc-test", "test-cluster", EventTypeRunStarted)
	err = client.PublishEvent(event)
	if err != nil {
		t.Fatalf("failed to publish valid event: %v", err)
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Listener should still be running (no panic or crash)
	if !client.IsConnected() {
		t.Error("client disconnected after malformed JSON")
	}
}

// TestProgressEvent_WithMethods verifies event builder methods
func TestProgressEvent_WithMethods(t *testing.T) {
	incidentID := "inc-test"
	cluster := "prod-cluster"

	event := NewProgressEvent(incidentID, cluster, EventTypeExecuting).
		WithAgentInfo("claude", "sonnet").
		WithActivity("kubectl get pods -n kube-system")

	if event.IncidentID != incidentID {
		t.Errorf("expected incident_id '%s', got '%s'", incidentID, event.IncidentID)
	}
	if event.Cluster != cluster {
		t.Errorf("expected cluster '%s', got '%s'", cluster, event.Cluster)
	}
	if event.EventType != string(EventTypeExecuting) {
		t.Errorf("expected event_type '%s', got '%s'", EventTypeExecuting, event.EventType)
	}
	if event.AgentCLI != "claude" {
		t.Errorf("expected agent_cli 'claude', got '%s'", event.AgentCLI)
	}
	if event.Model != "sonnet" {
		t.Errorf("expected model 'sonnet', got '%s'", event.Model)
	}
	if event.Activity != "kubectl get pods -n kube-system" {
		t.Errorf("expected activity 'kubectl get pods -n kube-system', got '%s'", event.Activity)
	}
}

// TestProgressEvent_ActivityTruncation verifies activity is truncated to 100 chars
func TestProgressEvent_ActivityTruncation(t *testing.T) {
	longActivity := "kubectl get pods -n kube-system && kubectl get nodes && kubectl get services && kubectl describe pod test-pod"

	event := NewProgressEvent("inc-test", "cluster", EventTypeExecuting).
		WithActivity(longActivity)

	if len(event.Activity) > 100 {
		t.Errorf("expected activity to be truncated to 100 chars, got %d", len(event.Activity))
	}
	if event.Activity != longActivity[:100] {
		t.Errorf("activity not truncated correctly")
	}
}

// TestSubjectForEvent verifies subject construction
func TestSubjectForEvent(t *testing.T) {
	tests := []struct {
		name       string
		incidentID string
		eventType  EventType
		expected   string
	}{
		{
			name:       "run started",
			incidentID: "inc-123",
			eventType:  EventTypeRunStarted,
			expected:   "triage.inc-123.run.started",
		},
		{
			name:       "executing",
			incidentID: "inc-456",
			eventType:  EventTypeExecuting,
			expected:   "triage.inc-456.executing",
		},
		{
			name:       "error",
			incidentID: "inc-789",
			eventType:  EventTypeError,
			expected:   "triage.inc-789.error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subject := SubjectForEvent(tt.incidentID, tt.eventType)
			if subject != tt.expected {
				t.Errorf("expected subject '%s', got '%s'", tt.expected, subject)
			}
		})
	}
}

// TestSubjectWildcard verifies wildcard subject
func TestSubjectWildcard(t *testing.T) {
	expected := "triage.>"
	subject := SubjectWildcard()
	if subject != expected {
		t.Errorf("expected subject '%s', got '%s'", expected, subject)
	}
}

// TestFromJSON_ValidEvent verifies event deserialization
func TestFromJSON_ValidEvent(t *testing.T) {
	jsonData := []byte(`{
		"incident_id": "inc-test",
		"cluster": "prod",
		"timestamp": "2025-01-03T12:00:00Z",
		"event_type": "executing",
		"agent_cli": "claude",
		"model": "sonnet",
		"activity": "kubectl get pods"
	}`)

	event, err := FromJSON(jsonData)
	if err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if event.IncidentID != "inc-test" {
		t.Errorf("expected incident_id 'inc-test', got '%s'", event.IncidentID)
	}
	if event.EventType != "executing" {
		t.Errorf("expected event_type 'executing', got '%s'", event.EventType)
	}
	if event.Activity != "kubectl get pods" {
		t.Errorf("expected activity 'kubectl get pods', got '%s'", event.Activity)
	}
}

// TestFromJSON_MalformedEvent verifies error handling for invalid JSON
func TestFromJSON_MalformedEvent(t *testing.T) {
	jsonData := []byte(`{"invalid": json}`)

	_, err := FromJSON(jsonData)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

// TestClient_Stats verifies client statistics
func TestClient_Stats(t *testing.T) {
	s, url := startTestServer(t)
	defer s.Shutdown()

	client, err := Connect(url, "")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	stats := client.Stats()
	if stats.OutMsgs != 0 {
		t.Errorf("expected 0 outgoing messages initially, got %d", stats.OutMsgs)
	}

	// Publish a message
	event := NewProgressEvent("inc-test", "cluster", EventTypeRunStarted)
	err = client.PublishEvent(event)
	if err != nil {
		t.Fatalf("failed to publish event: %v", err)
	}

	// Check stats again
	stats = client.Stats()
	if stats.OutMsgs < 1 {
		t.Errorf("expected at least 1 outgoing message, got %d", stats.OutMsgs)
	}
}

// TestClient_CloseIdempotent verifies Close can be called multiple times
func TestClient_CloseIdempotent(t *testing.T) {
	s, url := startTestServer(t)
	defer s.Shutdown()

	client, err := Connect(url, "")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	// Close multiple times should not panic
	client.Close()
	client.Close()
	client.Close()
}
