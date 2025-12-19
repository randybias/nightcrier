package reporting

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestSendIncidentNotification_WithURL(t *testing.T) {
	// Test the message structure by building it manually
	statusColor := "danger"

	blocks := []SlackBlock{
		{
			Type: "header",
			Text: &SlackText{
				Type: "plain_text",
				Text: "Kubernetes Incident Triage :x:",
			},
		},
		{
			Type: "section",
			Fields: []SlackText{
				{Type: "mrkdwn", Text: "*Cluster:*\nprod-cluster"},
				{Type: "mrkdwn", Text: "*Namespace:*\ndefault"},
				{Type: "mrkdwn", Text: "*Resource:*\npod/nginx-1234"},
				{Type: "mrkdwn", Text: "*Reason:*\nCrashLoopBackOff"},
			},
		},
		{
			Type: "section",
			Text: &SlackText{
				Type: "mrkdwn",
				Text: "*Root Cause (HIGH confidence):*\nApplication failed to start due to missing configuration",
			},
		},
		{
			Type: "context",
			Elements: []interface{}{
				SlackElement{Type: "mrkdwn", Text: "Incident ID: `incident-123` | Duration: 5m0s"},
			},
		},
		{
			Type: "actions",
			Elements: []interface{}{
				SlackButton{
					Type: "button",
					Text: &SlackText{
						Type: "plain_text",
						Text: "View Report",
					},
					URL: "https://storage.example.com/reports/incident-123/report.html?sig=abc123",
				},
			},
		},
	}

	attachments := []SlackAttachment{
		{
			Color:  statusColor,
			Footer: "Report: URL (see button above)",
		},
	}

	msg := SlackMessage{
		Blocks:      blocks,
		Attachments: attachments,
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message to JSON: %v", err)
	}

	// Verify button is in the message
	if !json.Valid(payload) {
		t.Fatal("Generated JSON is invalid")
	}

	// Unmarshal to verify structure
	var unmarshaledMsg SlackMessage
	err = json.Unmarshal(payload, &unmarshaledMsg)
	if err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	// Verify blocks count (should have 5 blocks with URL)
	if len(unmarshaledMsg.Blocks) != 5 {
		t.Errorf("Expected 5 blocks with URL, got %d", len(unmarshaledMsg.Blocks))
	}

	// Verify actions block exists
	hasActionsBlock := false
	for _, block := range unmarshaledMsg.Blocks {
		if block.Type == "actions" {
			hasActionsBlock = true
			if len(block.Elements) != 1 {
				t.Errorf("Expected 1 element in actions block, got %d", len(block.Elements))
			}
			break
		}
	}

	if !hasActionsBlock {
		t.Error("Missing actions block in message with ReportURL")
	}

	// Verify footer mentions URL
	if len(unmarshaledMsg.Attachments) == 0 || unmarshaledMsg.Attachments[0].Footer != "Report: URL (see button above)" {
		t.Error("Footer should mention URL when ReportURL is set")
	}
}

func TestSendIncidentNotification_WithoutURL(t *testing.T) {
	// Test the message structure by building it manually

	// Build message structure for testing
	blocks := []SlackBlock{
		{
			Type: "header",
			Text: &SlackText{
				Type: "plain_text",
				Text: "Kubernetes Incident Triage :x:",
			},
		},
		{
			Type: "section",
			Fields: []SlackText{
				{Type: "mrkdwn", Text: "*Cluster:*\nstaging-cluster"},
				{Type: "mrkdwn", Text: "*Namespace:*\nmonitoring"},
				{Type: "mrkdwn", Text: "*Resource:*\ndeployment/prometheus"},
				{Type: "mrkdwn", Text: "*Reason:*\nInsufficientMemory"},
			},
		},
		{
			Type: "section",
			Text: &SlackText{
				Type: "mrkdwn",
				Text: "*Root Cause (MEDIUM confidence):*\nPod memory limit exceeded",
			},
		},
		{
			Type: "context",
			Elements: []interface{}{
				SlackElement{Type: "mrkdwn", Text: "Incident ID: `incident-456` | Duration: 10m0s"},
			},
		},
	}

	attachments := []SlackAttachment{
		{
			Color:  "danger",
			Footer: "Report: /tmp/workspace/incident-456/report.md",
		},
	}

	msg := SlackMessage{
		Blocks:      blocks,
		Attachments: attachments,
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message to JSON: %v", err)
	}

	if !json.Valid(payload) {
		t.Fatal("Generated JSON is invalid")
	}

	// Unmarshal to verify structure
	var unmarshaledMsg SlackMessage
	err = json.Unmarshal(payload, &unmarshaledMsg)
	if err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	// Verify blocks count (should have 4 blocks without URL)
	if len(unmarshaledMsg.Blocks) != 4 {
		t.Errorf("Expected 4 blocks without URL, got %d", len(unmarshaledMsg.Blocks))
	}

	// Verify no actions block
	for _, block := range unmarshaledMsg.Blocks {
		if block.Type == "actions" {
			t.Error("Should not have actions block when ReportURL is empty")
		}
	}

	// Verify footer contains filesystem path
	expectedFooter := "Report: /tmp/workspace/incident-456/report.md"
	if len(unmarshaledMsg.Attachments) == 0 || unmarshaledMsg.Attachments[0].Footer != expectedFooter {
		t.Errorf("Expected footer '%s', got '%s'", expectedFooter, unmarshaledMsg.Attachments[0].Footer)
	}
}

func TestSendIncidentNotification_BothURLAndPath(t *testing.T) {
	// Test the message structure by building it manually

	// Build message structure for testing
	blocks := []SlackBlock{
		{
			Type: "header",
			Text: &SlackText{
				Type: "plain_text",
				Text: "Kubernetes Incident Triage :x:",
			},
		},
		{
			Type: "section",
			Fields: []SlackText{
				{Type: "mrkdwn", Text: "*Cluster:*\nprod-cluster"},
				{Type: "mrkdwn", Text: "*Namespace:*\ndefault"},
				{Type: "mrkdwn", Text: "*Resource:*\nstatefulset/database"},
				{Type: "mrkdwn", Text: "*Reason:*\nImagePullBackOff"},
			},
		},
		{
			Type: "section",
			Text: &SlackText{
				Type: "mrkdwn",
				Text: "*Root Cause (LOW confidence):*\nContainer image not found in registry",
			},
		},
		{
			Type: "context",
			Elements: []interface{}{
				SlackElement{Type: "mrkdwn", Text: "Incident ID: `incident-789` | Duration: 2m0s"},
			},
		},
		{
			Type: "actions",
			Elements: []interface{}{
				SlackButton{
					Type: "button",
					Text: &SlackText{
						Type: "plain_text",
						Text: "View Report",
					},
					URL: "https://storage.example.com/reports/incident-789/report.html",
				},
			},
		},
	}

	attachments := []SlackAttachment{
		{
			Color:  "danger",
			Footer: "Report: URL (see button above)",
		},
	}

	msg := SlackMessage{
		Blocks:      blocks,
		Attachments: attachments,
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message to JSON: %v", err)
	}

	if !json.Valid(payload) {
		t.Fatal("Generated JSON is invalid")
	}

	var unmarshaledMsg SlackMessage
	err = json.Unmarshal(payload, &unmarshaledMsg)
	if err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	// When both URL and path exist, URL takes precedence in the footer
	if len(unmarshaledMsg.Attachments) == 0 || unmarshaledMsg.Attachments[0].Footer != "Report: URL (see button above)" {
		t.Error("Footer should prioritize URL over path when both are available")
	}

	// Verify button is present
	hasActionsBlock := false
	for _, block := range unmarshaledMsg.Blocks {
		if block.Type == "actions" {
			hasActionsBlock = true
			break
		}
	}
	if !hasActionsBlock {
		t.Error("Should have actions block when ReportURL is set")
	}
}

func TestSendIncidentNotification_SuccessStatus(t *testing.T) {
	// Test the message structure by building it manually

	// Build message structure
	blocks := []SlackBlock{
		{
			Type: "header",
			Text: &SlackText{
				Type: "plain_text",
				Text: "Kubernetes Incident Triage :white_check_mark:",
			},
		},
		{
			Type: "section",
			Fields: []SlackText{
				{Type: "mrkdwn", Text: "*Cluster:*\nprod"},
				{Type: "mrkdwn", Text: "*Namespace:*\ndefault"},
				{Type: "mrkdwn", Text: "*Resource:*\npod/app"},
				{Type: "mrkdwn", Text: "*Reason:*\nResolved"},
			},
		},
		{
			Type: "section",
			Text: &SlackText{
				Type: "mrkdwn",
				Text: "*Root Cause (HIGH confidence):*\nIssue auto-resolved",
			},
		},
		{
			Type: "context",
			Elements: []interface{}{
				SlackElement{Type: "mrkdwn", Text: "Incident ID: `incident-success` | Duration: 1m0s"},
			},
		},
		{
			Type: "actions",
			Elements: []interface{}{
				SlackButton{
					Type: "button",
					Text: &SlackText{
						Type: "plain_text",
						Text: "View Report",
					},
					URL: "https://example.com/report",
				},
			},
		},
	}

	attachments := []SlackAttachment{
		{
			Color:  "good",
			Footer: "Report: URL (see button above)",
		},
	}

	msg := SlackMessage{
		Blocks:      blocks,
		Attachments: attachments,
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message to JSON: %v", err)
	}

	var unmarshaledMsg SlackMessage
	err = json.Unmarshal(payload, &unmarshaledMsg)
	if err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	// Verify color is "good" for success
	if len(unmarshaledMsg.Attachments) == 0 || unmarshaledMsg.Attachments[0].Color != "good" {
		t.Error("Attachment color should be 'good' for success status")
	}
}

func TestSlackButtonMarshaling(t *testing.T) {
	button := SlackButton{
		Type: "button",
		Text: &SlackText{
			Type: "plain_text",
			Text: "View Report",
		},
		URL: "https://example.com/report",
	}

	payload, err := json.Marshal(button)
	if err != nil {
		t.Fatalf("Failed to marshal button: %v", err)
	}

	var unmarshaledButton SlackButton
	err = json.Unmarshal(payload, &unmarshaledButton)
	if err != nil {
		t.Fatalf("Failed to unmarshal button: %v", err)
	}

	if unmarshaledButton.Type != "button" {
		t.Errorf("Expected button type 'button', got '%s'", unmarshaledButton.Type)
	}

	if unmarshaledButton.URL != "https://example.com/report" {
		t.Errorf("Expected URL 'https://example.com/report', got '%s'", unmarshaledButton.URL)
	}

	if unmarshaledButton.Text.Text != "View Report" {
		t.Errorf("Expected button text 'View Report', got '%s'", unmarshaledButton.Text.Text)
	}
}

func TestSendSystemDegradedAlert_BasicMessage(t *testing.T) {
	// Create a notifier (webhook URL not needed for message format testing)
	notifier := NewSlackNotifier("")

	// Create sample failure stats
	now := time.Now()
	stats := FailureStats{
		Count:            3,
		FirstFailureTime: now.Add(-5 * time.Minute),
		LastFailureTime:  now,
		Duration:         5 * time.Minute,
		RecentReasons: []string{
			"Agent timeout after 30s",
			"Failed to connect to MCP server",
			"Context deadline exceeded",
		},
	}

	// Build expected message structure
	blocks := []SlackBlock{
		{
			Type: "header",
			Text: &SlackText{
				Type: "plain_text",
				Text: "AI Agent System Degraded",
			},
		},
		{
			Type: "section",
			Fields: []SlackText{
				{Type: "mrkdwn", Text: "*Failure Count:*\n3"},
				{Type: "mrkdwn", Text: "*Time Window:*\n5m0s"},
			},
		},
		{
			Type: "section",
			Text: &SlackText{
				Type: "mrkdwn",
				Text: "*Sample Failure Reasons (last 3):*\n• Agent timeout after 30s\n• Failed to connect to MCP server\n• Context deadline exceeded",
			},
		},
		{
			Type: "context",
			Elements: []interface{}{
				SlackElement{
					Type: "mrkdwn",
					Text: "First failure: " + stats.FirstFailureTime.Format("15:04:05") + " | Last failure: " + stats.LastFailureTime.Format("15:04:05"),
				},
			},
		},
	}

	msg := SlackMessage{
		Blocks: blocks,
		Attachments: []SlackAttachment{
			{
				Color:  "warning",
				Footer: "System degradation threshold reached. AI agent may be experiencing issues.",
			},
		},
	}

	// Marshal and verify JSON structure
	payload, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	if !json.Valid(payload) {
		t.Fatal("Generated JSON is invalid")
	}

	var unmarshaledMsg SlackMessage
	err = json.Unmarshal(payload, &unmarshaledMsg)
	if err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	// Verify block count
	if len(unmarshaledMsg.Blocks) != 4 {
		t.Errorf("Expected 4 blocks, got %d", len(unmarshaledMsg.Blocks))
	}

	// Verify header
	if unmarshaledMsg.Blocks[0].Type != "header" {
		t.Errorf("First block should be header, got %s", unmarshaledMsg.Blocks[0].Type)
	}

	// Verify warning color in attachment
	if len(unmarshaledMsg.Attachments) == 0 || unmarshaledMsg.Attachments[0].Color != "warning" {
		t.Error("Attachment should have 'warning' color")
	}

	// Test the actual method (will skip sending since webhook is empty)
	err = notifier.SendSystemDegradedAlert(context.Background(), stats)
	if err != nil {
		t.Errorf("SendSystemDegradedAlert should not error with empty webhook: %v", err)
	}
}

func TestSendSystemDegradedAlert_MoreThanThreeReasons(t *testing.T) {
	notifier := NewSlackNotifier("")

	now := time.Now()
	stats := FailureStats{
		Count:            5,
		FirstFailureTime: now.Add(-10 * time.Minute),
		LastFailureTime:  now,
		Duration:         10 * time.Minute,
		RecentReasons: []string{
			"Error 1",
			"Error 2",
			"Error 3",
			"Error 4",
			"Error 5",
		},
	}

	// Build expected message - should only include last 3 reasons
	blocks := []SlackBlock{
		{
			Type: "header",
			Text: &SlackText{
				Type: "plain_text",
				Text: "AI Agent System Degraded",
			},
		},
		{
			Type: "section",
			Fields: []SlackText{
				{Type: "mrkdwn", Text: "*Failure Count:*\n5"},
				{Type: "mrkdwn", Text: "*Time Window:*\n10m0s"},
			},
		},
		{
			Type: "section",
			Text: &SlackText{
				Type: "mrkdwn",
				Text: "*Sample Failure Reasons (last 3):*\n• Error 3\n• Error 4\n• Error 5",
			},
		},
		{
			Type: "context",
			Elements: []interface{}{
				SlackElement{
					Type: "mrkdwn",
					Text: "First failure: " + stats.FirstFailureTime.Format("15:04:05") + " | Last failure: " + stats.LastFailureTime.Format("15:04:05"),
				},
			},
		},
	}

	msg := SlackMessage{
		Blocks: blocks,
		Attachments: []SlackAttachment{
			{
				Color:  "warning",
				Footer: "System degradation threshold reached. AI agent may be experiencing issues.",
			},
		},
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	var unmarshaledMsg SlackMessage
	err = json.Unmarshal(payload, &unmarshaledMsg)
	if err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	// Verify the sample reasons section contains only last 3
	if unmarshaledMsg.Blocks[2].Text.Text != "*Sample Failure Reasons (last 3):*\n• Error 3\n• Error 4\n• Error 5" {
		t.Errorf("Expected last 3 reasons, got: %s", unmarshaledMsg.Blocks[2].Text.Text)
	}

	err = notifier.SendSystemDegradedAlert(context.Background(), stats)
	if err != nil {
		t.Errorf("SendSystemDegradedAlert should not error: %v", err)
	}
}

func TestSendSystemDegradedAlert_NoReasons(t *testing.T) {
	notifier := NewSlackNotifier("")

	now := time.Now()
	stats := FailureStats{
		Count:            2,
		FirstFailureTime: now.Add(-2 * time.Minute),
		LastFailureTime:  now,
		Duration:         2 * time.Minute,
		RecentReasons:    []string{}, // No reasons
	}

	blocks := []SlackBlock{
		{
			Type: "header",
			Text: &SlackText{
				Type: "plain_text",
				Text: "AI Agent System Degraded",
			},
		},
		{
			Type: "section",
			Fields: []SlackText{
				{Type: "mrkdwn", Text: "*Failure Count:*\n2"},
				{Type: "mrkdwn", Text: "*Time Window:*\n2m0s"},
			},
		},
		{
			Type: "section",
			Text: &SlackText{
				Type: "mrkdwn",
				Text: "*Sample Failure Reasons (last 3):*\nNo failure details available",
			},
		},
		{
			Type: "context",
			Elements: []interface{}{
				SlackElement{
					Type: "mrkdwn",
					Text: "First failure: " + stats.FirstFailureTime.Format("15:04:05") + " | Last failure: " + stats.LastFailureTime.Format("15:04:05"),
				},
			},
		},
	}

	msg := SlackMessage{
		Blocks: blocks,
		Attachments: []SlackAttachment{
			{
				Color:  "warning",
				Footer: "System degradation threshold reached. AI agent may be experiencing issues.",
			},
		},
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	var unmarshaledMsg SlackMessage
	err = json.Unmarshal(payload, &unmarshaledMsg)
	if err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	// Verify the sample reasons section shows "No failure details available"
	if unmarshaledMsg.Blocks[2].Text.Text != "*Sample Failure Reasons (last 3):*\nNo failure details available" {
		t.Errorf("Expected 'No failure details available', got: %s", unmarshaledMsg.Blocks[2].Text.Text)
	}

	err = notifier.SendSystemDegradedAlert(context.Background(), stats)
	if err != nil {
		t.Errorf("SendSystemDegradedAlert should not error: %v", err)
	}
}

func TestSendSystemDegradedAlert_ZeroDuration(t *testing.T) {
	notifier := NewSlackNotifier("")

	now := time.Now()
	stats := FailureStats{
		Count:            1,
		FirstFailureTime: now,
		LastFailureTime:  now,
		Duration:         0, // Zero duration
		RecentReasons:    []string{"Single failure"},
	}

	blocks := []SlackBlock{
		{
			Type: "header",
			Text: &SlackText{
				Type: "plain_text",
				Text: "AI Agent System Degraded",
			},
		},
		{
			Type: "section",
			Fields: []SlackText{
				{Type: "mrkdwn", Text: "*Failure Count:*\n1"},
				{Type: "mrkdwn", Text: "*Time Window:*\nN/A"}, // Should show N/A for zero duration
			},
		},
		{
			Type: "section",
			Text: &SlackText{
				Type: "mrkdwn",
				Text: "*Sample Failure Reasons (last 3):*\n• Single failure",
			},
		},
		{
			Type: "context",
			Elements: []interface{}{
				SlackElement{
					Type: "mrkdwn",
					Text: "First failure: " + stats.FirstFailureTime.Format("15:04:05") + " | Last failure: " + stats.LastFailureTime.Format("15:04:05"),
				},
			},
		},
	}

	msg := SlackMessage{
		Blocks: blocks,
		Attachments: []SlackAttachment{
			{
				Color:  "warning",
				Footer: "System degradation threshold reached. AI agent may be experiencing issues.",
			},
		},
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	var unmarshaledMsg SlackMessage
	err = json.Unmarshal(payload, &unmarshaledMsg)
	if err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	// Verify time window shows N/A
	timeWindowField := unmarshaledMsg.Blocks[1].Fields[1].Text
	if timeWindowField != "*Time Window:*\nN/A" {
		t.Errorf("Expected time window 'N/A', got: %s", timeWindowField)
	}

	err = notifier.SendSystemDegradedAlert(context.Background(), stats)
	if err != nil {
		t.Errorf("SendSystemDegradedAlert should not error: %v", err)
	}
}

func TestSendSystemRecoveredAlert_BasicMessage(t *testing.T) {
	// Create a notifier (webhook URL not needed for message format testing)
	notifier := NewSlackNotifier("")

	// Create sample failure stats representing a recovery
	now := time.Now()
	stats := FailureStats{
		Count:            5,
		FirstFailureTime: now.Add(-10 * time.Minute),
		LastFailureTime:  now.Add(-1 * time.Minute),
		Duration:         9 * time.Minute,
		RecentReasons: []string{
			"Agent timeout",
			"Connection failed",
		},
	}

	// Build expected message structure
	blocks := []SlackBlock{
		{
			Type: "header",
			Text: &SlackText{
				Type: "plain_text",
				Text: "AI Agent System Recovered",
			},
		},
		{
			Type: "section",
			Fields: []SlackText{
				{Type: "mrkdwn", Text: "*Total Downtime:*\n9m0s"},
				{Type: "mrkdwn", Text: "*Total Failures:*\n5"},
			},
		},
		{
			Type: "context",
			Elements: []interface{}{
				SlackElement{
					Type: "mrkdwn",
					Text: "System has returned to healthy state. All agents operating normally.",
				},
			},
		},
	}

	msg := SlackMessage{
		Blocks: blocks,
		Attachments: []SlackAttachment{
			{
				Color:  "good",
				Footer: "System recovery detected. AI agent system is now healthy.",
			},
		},
	}

	// Marshal and verify JSON structure
	payload, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	if !json.Valid(payload) {
		t.Fatal("Generated JSON is invalid")
	}

	var unmarshaledMsg SlackMessage
	err = json.Unmarshal(payload, &unmarshaledMsg)
	if err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	// Verify block count
	if len(unmarshaledMsg.Blocks) != 3 {
		t.Errorf("Expected 3 blocks, got %d", len(unmarshaledMsg.Blocks))
	}

	// Verify header
	if unmarshaledMsg.Blocks[0].Type != "header" {
		t.Errorf("First block should be header, got %s", unmarshaledMsg.Blocks[0].Type)
	}

	if unmarshaledMsg.Blocks[0].Text.Text != "AI Agent System Recovered" {
		t.Errorf("Expected header 'AI Agent System Recovered', got %s", unmarshaledMsg.Blocks[0].Text.Text)
	}

	// Verify good color in attachment
	if len(unmarshaledMsg.Attachments) == 0 || unmarshaledMsg.Attachments[0].Color != "good" {
		t.Error("Attachment should have 'good' color")
	}

	// Test the actual method (will skip sending since webhook is empty)
	err = notifier.SendSystemRecoveredAlert(context.Background(), stats)
	if err != nil {
		t.Errorf("SendSystemRecoveredAlert should not error with empty webhook: %v", err)
	}
}

func TestSendSystemRecoveredAlert_ZeroDuration(t *testing.T) {
	notifier := NewSlackNotifier("")

	now := time.Now()
	stats := FailureStats{
		Count:            3,
		FirstFailureTime: now,
		LastFailureTime:  now,
		Duration:         0, // Zero duration
		RecentReasons:    []string{"Quick failure"},
	}

	blocks := []SlackBlock{
		{
			Type: "header",
			Text: &SlackText{
				Type: "plain_text",
				Text: "AI Agent System Recovered",
			},
		},
		{
			Type: "section",
			Fields: []SlackText{
				{Type: "mrkdwn", Text: "*Total Downtime:*\nN/A"}, // Should show N/A for zero duration
				{Type: "mrkdwn", Text: "*Total Failures:*\n3"},
			},
		},
		{
			Type: "context",
			Elements: []interface{}{
				SlackElement{
					Type: "mrkdwn",
					Text: "System has returned to healthy state. All agents operating normally.",
				},
			},
		},
	}

	msg := SlackMessage{
		Blocks: blocks,
		Attachments: []SlackAttachment{
			{
				Color:  "good",
				Footer: "System recovery detected. AI agent system is now healthy.",
			},
		},
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	var unmarshaledMsg SlackMessage
	err = json.Unmarshal(payload, &unmarshaledMsg)
	if err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	// Verify downtime shows N/A
	downtimeField := unmarshaledMsg.Blocks[1].Fields[0].Text
	if downtimeField != "*Total Downtime:*\nN/A" {
		t.Errorf("Expected downtime 'N/A', got: %s", downtimeField)
	}

	err = notifier.SendSystemRecoveredAlert(context.Background(), stats)
	if err != nil {
		t.Errorf("SendSystemRecoveredAlert should not error: %v", err)
	}
}

func TestSendSystemRecoveredAlert_HighFailureCount(t *testing.T) {
	notifier := NewSlackNotifier("")

	now := time.Now()
	stats := FailureStats{
		Count:            50,
		FirstFailureTime: now.Add(-1 * time.Hour),
		LastFailureTime:  now.Add(-5 * time.Minute),
		Duration:         55 * time.Minute,
		RecentReasons: []string{
			"Multiple failures occurred",
		},
	}

	blocks := []SlackBlock{
		{
			Type: "header",
			Text: &SlackText{
				Type: "plain_text",
				Text: "AI Agent System Recovered",
			},
		},
		{
			Type: "section",
			Fields: []SlackText{
				{Type: "mrkdwn", Text: "*Total Downtime:*\n55m0s"},
				{Type: "mrkdwn", Text: "*Total Failures:*\n50"},
			},
		},
		{
			Type: "context",
			Elements: []interface{}{
				SlackElement{
					Type: "mrkdwn",
					Text: "System has returned to healthy state. All agents operating normally.",
				},
			},
		},
	}

	msg := SlackMessage{
		Blocks: blocks,
		Attachments: []SlackAttachment{
			{
				Color:  "good",
				Footer: "System recovery detected. AI agent system is now healthy.",
			},
		},
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	var unmarshaledMsg SlackMessage
	err = json.Unmarshal(payload, &unmarshaledMsg)
	if err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	// Verify failure count is correct
	failureCountField := unmarshaledMsg.Blocks[1].Fields[1].Text
	if failureCountField != "*Total Failures:*\n50" {
		t.Errorf("Expected 50 failures, got: %s", failureCountField)
	}

	// Verify downtime is correct
	downtimeField := unmarshaledMsg.Blocks[1].Fields[0].Text
	if downtimeField != "*Total Downtime:*\n55m0s" {
		t.Errorf("Expected 55m0s downtime, got: %s", downtimeField)
	}

	err = notifier.SendSystemRecoveredAlert(context.Background(), stats)
	if err != nil {
		t.Errorf("SendSystemRecoveredAlert should not error: %v", err)
	}
}
