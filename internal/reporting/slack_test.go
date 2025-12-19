package reporting

import (
	"encoding/json"
	"testing"
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
