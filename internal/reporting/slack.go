package reporting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SlackNotifier sends incident notifications to Slack
type SlackNotifier struct {
	WebhookURL string
	httpClient *http.Client
}

// SlackMessage represents a Slack webhook message
type SlackMessage struct {
	Text        string            `json:"text,omitempty"`
	Blocks      []SlackBlock      `json:"blocks,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

// SlackBlock represents a Slack block element
type SlackBlock struct {
	Type     string        `json:"type"`
	Text     *SlackText    `json:"text,omitempty"`
	Fields   []SlackText   `json:"fields,omitempty"`
	Elements []interface{} `json:"elements,omitempty"`
}

// SlackText represents text content in Slack
type SlackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// SlackElement represents an element in a context block
type SlackElement struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// SlackButton represents a button element in an actions block
type SlackButton struct {
	Type string     `json:"type"`
	Text *SlackText `json:"text"`
	URL  string     `json:"url"`
}

// SlackAttachment represents a Slack attachment
type SlackAttachment struct {
	Color  string `json:"color"`
	Text   string `json:"text,omitempty"`
	Footer string `json:"footer,omitempty"`
}

// IncidentSummary contains the key information for a Slack notification
type IncidentSummary struct {
	IncidentID string
	Cluster    string
	Namespace  string
	Resource   string
	Reason     string
	Status     string
	RootCause  string
	Confidence string
	Duration   time.Duration
	ReportPath string
	ReportURL  string
}

// NewSlackNotifier creates a new Slack notifier
func NewSlackNotifier(webhookURL string) *SlackNotifier {
	return &SlackNotifier{
		WebhookURL: webhookURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendIncidentNotification sends a formatted incident notification to Slack
func (s *SlackNotifier) SendIncidentNotification(summary *IncidentSummary) error {
	if s.WebhookURL == "" {
		return nil // No webhook configured, skip silently
	}

	// Determine status emoji and color based on incident status
	statusEmoji := ":white_check_mark:"
	statusColor := "good"
	// Check for resolved status (successful completion)
	if summary.Status != "resolved" {
		statusEmoji = ":x:"
		statusColor = "danger"
	}

	// Build the blocks
	blocks := []SlackBlock{
		{
			Type: "header",
			Text: &SlackText{
				Type: "plain_text",
				Text: fmt.Sprintf("Kubernetes Incident Triage %s", statusEmoji),
			},
		},
		{
			Type: "section",
			Fields: []SlackText{
				{Type: "mrkdwn", Text: fmt.Sprintf("*Cluster:*\n%s", summary.Cluster)},
				{Type: "mrkdwn", Text: fmt.Sprintf("*Namespace:*\n%s", summary.Namespace)},
				{Type: "mrkdwn", Text: fmt.Sprintf("*Resource:*\n%s", summary.Resource)},
				{Type: "mrkdwn", Text: fmt.Sprintf("*Reason:*\n%s", summary.Reason)},
			},
		},
		{
			Type: "section",
			Text: &SlackText{
				Type: "mrkdwn",
				Text: fmt.Sprintf("*Root Cause (%s confidence):*\n%s", summary.Confidence, summary.RootCause),
			},
		},
		{
			Type: "context",
			Elements: []interface{}{
				SlackElement{Type: "mrkdwn", Text: fmt.Sprintf("Incident ID: `%s` | Duration: %s", summary.IncidentID, summary.Duration.Round(time.Second))},
			},
		},
	}

	// Add "View Report" button if URL is available
	if summary.ReportURL != "" {
		blocks = append(blocks, SlackBlock{
			Type: "actions",
			Elements: []interface{}{
				SlackButton{
					Type: "button",
					Text: &SlackText{
						Type: "plain_text",
						Text: "View Report",
					},
					URL: summary.ReportURL,
				},
			},
		})
	}

	// Determine footer text based on available data
	var footer string
	if summary.ReportURL != "" {
		footer = "Report: URL (see button above)"
	} else if summary.ReportPath != "" {
		footer = fmt.Sprintf("Report: %s", summary.ReportPath)
	}

	// Build the message
	msg := SlackMessage{
		Blocks: blocks,
		Attachments: []SlackAttachment{
			{
				Color:  statusColor,
				Footer: footer,
			},
		},
	}

	return s.send(msg)
}

// SendSystemDegradedAlert sends a system-level degradation alert to Slack
func (s *SlackNotifier) SendSystemDegradedAlert(ctx context.Context, stats FailureStats) error {
	if s.WebhookURL == "" {
		return nil // No webhook configured, skip silently
	}

	// Format the time window
	timeWindow := "N/A"
	if stats.Duration > 0 {
		timeWindow = stats.Duration.Round(time.Second).String()
	}

	// Format the failure count
	failureCount := fmt.Sprintf("%d", stats.Count)

	// Get the last 3 failure reasons
	sampleReasons := stats.RecentReasons
	if len(sampleReasons) > 3 {
		sampleReasons = sampleReasons[len(sampleReasons)-3:]
	}

	// Format sample reasons as a bullet list
	reasonsText := ""
	if len(sampleReasons) > 0 {
		var reasonsList []string
		for _, reason := range sampleReasons {
			reasonsList = append(reasonsList, fmt.Sprintf("• %s", reason))
		}
		reasonsText = strings.Join(reasonsList, "\n")
	} else {
		reasonsText = "No failure details available"
	}

	// Build the blocks
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
				{Type: "mrkdwn", Text: fmt.Sprintf("*Failure Count:*\n%s", failureCount)},
				{Type: "mrkdwn", Text: fmt.Sprintf("*Time Window:*\n%s", timeWindow)},
			},
		},
		{
			Type: "section",
			Text: &SlackText{
				Type: "mrkdwn",
				Text: fmt.Sprintf("*Sample Failure Reasons (last 3):*\n%s", reasonsText),
			},
		},
		{
			Type: "context",
			Elements: []interface{}{
				SlackElement{Type: "mrkdwn", Text: fmt.Sprintf("First failure: %s | Last failure: %s",
					stats.FirstFailureTime.Format("15:04:05"),
					stats.LastFailureTime.Format("15:04:05"))},
			},
		},
	}

	// Build the message with warning color
	msg := SlackMessage{
		Blocks: blocks,
		Attachments: []SlackAttachment{
			{
				Color:  "warning", // Yellow/orange color for warning level
				Footer: "System degradation threshold reached. AI agent may be experiencing issues.",
			},
		},
	}

	return s.send(msg)
}

// SendSystemRecoveredAlert sends a system recovery alert to Slack
func (s *SlackNotifier) SendSystemRecoveredAlert(ctx context.Context, stats FailureStats) error {
	if s.WebhookURL == "" {
		return nil // No webhook configured, skip silently
	}

	// Format the downtime duration
	downtime := "N/A"
	if stats.Duration > 0 {
		downtime = stats.Duration.Round(time.Second).String()
	}

	// Format the failure count
	failureCount := fmt.Sprintf("%d", stats.Count)

	// Build the blocks
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
				{Type: "mrkdwn", Text: fmt.Sprintf("*Total Downtime:*\n%s", downtime)},
				{Type: "mrkdwn", Text: fmt.Sprintf("*Total Failures:*\n%s", failureCount)},
			},
		},
		{
			Type: "context",
			Elements: []interface{}{
				SlackElement{Type: "mrkdwn", Text: "System has returned to healthy state. All agents operating normally."},
			},
		},
	}

	// Build the message with success color (green)
	msg := SlackMessage{
		Blocks: blocks,
		Attachments: []SlackAttachment{
			{
				Color:  "good", // Green color for success
				Footer: "System recovery detected. AI agent system is now healthy.",
			},
		},
	}

	return s.send(msg)
}

// send sends a message to the Slack webhook
func (s *SlackNotifier) send(msg SlackMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal slack message: %w", err)
	}

	resp, err := s.httpClient.Post(s.WebhookURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to send slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("slack webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ExtractSummaryFromReport reads an investigation report and extracts key information
func ExtractSummaryFromReport(workspacePath string) (rootCause, confidence string, err error) {
	reportPath := filepath.Join(workspacePath, "output", "investigation.md")

	content, err := os.ReadFile(reportPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read investigation report: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	// Extract root cause and confidence from the report
	inRootCause := false
	var rootCauseLines []string

	for _, line := range lines {
		// Look for confidence level (handles markdown bold ** markers)
		lineLower := strings.ToLower(line)
		if strings.Contains(lineLower, "confidence level") || strings.Contains(lineLower, "confidence:") || strings.Contains(line, "confidence)") {
			lineUpper := strings.ToUpper(line)
			if strings.Contains(lineUpper, "HIGH") {
				confidence = "HIGH"
			} else if strings.Contains(lineUpper, "MEDIUM") {
				confidence = "MEDIUM"
			} else if strings.Contains(lineUpper, "LOW") {
				confidence = "LOW"
			}
		}

		// Look for root cause section
		if strings.HasPrefix(line, "## Root Cause") {
			inRootCause = true
			continue
		}

		// End of root cause section
		if inRootCause && strings.HasPrefix(line, "## ") {
			inRootCause = false
		}

		// Capture root cause content (first substantive paragraph)
		if inRootCause && strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "**Confidence") {
			rootCauseLines = append(rootCauseLines, strings.TrimSpace(line))
			if len(rootCauseLines) >= 2 {
				break // Just get first couple lines
			}
		}
	}

	if len(rootCauseLines) > 0 {
		rootCause = strings.Join(rootCauseLines, " ")
		// Truncate if too long
		if len(rootCause) > 300 {
			rootCause = rootCause[:297] + "..."
		}
	} else {
		rootCause = "See investigation report for details"
	}

	if confidence == "" {
		confidence = "UNKNOWN"
	}

	return rootCause, confidence, nil
}
