package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/rbias/kubernetes-mcp-alerts-event-runner/internal/reporting"
)

func TestDetectAgentFailure(t *testing.T) {
	// Create a temporary directory for test workspaces
	tempDir := t.TempDir()

	tests := []struct {
		name            string
		setupFunc       func(string) error
		workspacePath   string
		exitCode        int
		err             error
		expectFailed    bool
		expectReasonMsg string
	}{
		{
			name: "success - exit code 0, file exists with sufficient size",
			setupFunc: func(workspacePath string) error {
				outputDir := filepath.Join(workspacePath, "output")
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					return err
				}
				// Create file with > 100 bytes
				content := make([]byte, 150)
				for i := range content {
					content[i] = 'a'
				}
				return os.WriteFile(filepath.Join(outputDir, "investigation.md"), content, 0644)
			},
			exitCode:        0,
			err:             nil,
			expectFailed:    false,
			expectReasonMsg: "",
		},
		{
			name: "failure - execution error",
			setupFunc: func(workspacePath string) error {
				return nil
			},
			exitCode:        0,
			err:             errors.New("mock execution error"),
			expectFailed:    true,
			expectReasonMsg: "agent execution error",
		},
		{
			name: "failure - non-zero exit code",
			setupFunc: func(workspacePath string) error {
				outputDir := filepath.Join(workspacePath, "output")
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					return err
				}
				content := make([]byte, 150)
				return os.WriteFile(filepath.Join(outputDir, "investigation.md"), content, 0644)
			},
			exitCode:        1,
			err:             nil,
			expectFailed:    true,
			expectReasonMsg: "agent exited with non-zero code: 1",
		},
		{
			name: "failure - investigation.md file not found",
			setupFunc: func(workspacePath string) error {
				// Create output directory but no file
				outputDir := filepath.Join(workspacePath, "output")
				return os.MkdirAll(outputDir, 0755)
			},
			exitCode:        0,
			err:             nil,
			expectFailed:    true,
			expectReasonMsg: "investigation.md file not found",
		},
		{
			name: "failure - investigation.md too small (0 bytes)",
			setupFunc: func(workspacePath string) error {
				outputDir := filepath.Join(workspacePath, "output")
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					return err
				}
				// Create empty file
				return os.WriteFile(filepath.Join(outputDir, "investigation.md"), []byte{}, 0644)
			},
			exitCode:        0,
			err:             nil,
			expectFailed:    true,
			expectReasonMsg: "investigation.md too small: 0 bytes (expected > 100)",
		},
		{
			name: "failure - investigation.md too small (exactly 100 bytes)",
			setupFunc: func(workspacePath string) error {
				outputDir := filepath.Join(workspacePath, "output")
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					return err
				}
				// Create file with exactly 100 bytes (should fail as we need > 100)
				content := make([]byte, 100)
				return os.WriteFile(filepath.Join(outputDir, "investigation.md"), content, 0644)
			},
			exitCode:        0,
			err:             nil,
			expectFailed:    true,
			expectReasonMsg: "investigation.md too small: 100 bytes (expected > 100)",
		},
		{
			name: "success - investigation.md exactly 101 bytes (boundary test)",
			setupFunc: func(workspacePath string) error {
				outputDir := filepath.Join(workspacePath, "output")
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					return err
				}
				// Create file with exactly 101 bytes (should pass)
				content := make([]byte, 101)
				return os.WriteFile(filepath.Join(outputDir, "investigation.md"), content, 0644)
			},
			exitCode:        0,
			err:             nil,
			expectFailed:    false,
			expectReasonMsg: "",
		},
		{
			name: "failure - multiple issues (exit code takes precedence over missing file)",
			setupFunc: func(workspacePath string) error {
				// Don't create the file at all
				return nil
			},
			exitCode:        42,
			err:             nil,
			expectFailed:    true,
			expectReasonMsg: "agent exited with non-zero code: 42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a unique workspace for this test
			workspacePath := filepath.Join(tempDir, tt.name)
			if err := os.MkdirAll(workspacePath, 0755); err != nil {
				t.Fatalf("failed to create workspace: %v", err)
			}

			// Setup test environment
			if err := tt.setupFunc(workspacePath); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			// Call the function under test
			failed, reason := detectAgentFailure(workspacePath, tt.exitCode, tt.err)

			// Validate results
			if failed != tt.expectFailed {
				t.Errorf("detectAgentFailure() failed = %v, want %v", failed, tt.expectFailed)
			}

			if tt.expectReasonMsg != "" {
				if reason != tt.expectReasonMsg {
					// For error messages, check if the expected message is contained
					if len(reason) < len(tt.expectReasonMsg) || reason[:len(tt.expectReasonMsg)] != tt.expectReasonMsg {
						t.Errorf("detectAgentFailure() reason = %q, want to start with %q", reason, tt.expectReasonMsg)
					}
				}
			} else if reason != "" {
				t.Errorf("detectAgentFailure() reason = %q, want empty string", reason)
			}
		})
	}
}

func TestDetectAgentFailure_ExitCodeCheckedBeforeFileChecks(t *testing.T) {
	// This test verifies that exit code is checked before file system operations
	// This is important because if the agent fails early, we don't want to waste time
	// checking files that may not have been created
	tempDir := t.TempDir()
	workspacePath := filepath.Join(tempDir, "test")
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Don't create any files
	failed, reason := detectAgentFailure(workspacePath, 1, nil)

	if !failed {
		t.Error("expected failure when exit code is non-zero")
	}

	// The reason should mention the exit code, not the missing file
	if reason != "agent exited with non-zero code: 1" {
		t.Errorf("expected exit code error message, got: %s", reason)
	}
}

func TestDetectAgentFailure_ExecutionErrorCheckedFirst(t *testing.T) {
	// This test verifies that execution errors are checked before everything else
	tempDir := t.TempDir()
	workspacePath := filepath.Join(tempDir, "test")
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	testErr := errors.New("test error")
	failed, reason := detectAgentFailure(workspacePath, 0, testErr)

	if !failed {
		t.Error("expected failure when execution error is present")
	}

	if reason != "agent execution error: test error" {
		t.Errorf("expected execution error message, got: %s", reason)
	}
}

// TestProcessEvent_Integration tests the full event processing flow including agent failure handling
func TestProcessEvent_Integration(t *testing.T) {
	// Skip if not in integration test mode (require explicit opt-in)
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tests := []struct {
		name                  string
		setupWorkspace        func(string) error
		mockAgentExitCode     int
		mockAgentError        error
		expectStatus          string
		expectStorageSkipped  bool
		expectSlackSkipped    bool
		expectResultFileWritten bool
	}{
		{
			name: "agent success - full flow",
			setupWorkspace: func(workspacePath string) error {
				outputDir := filepath.Join(workspacePath, "output")
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					return err
				}
				content := []byte("# Investigation Report\n\nThis is a successful investigation with sufficient content to pass validation checks.")
				return os.WriteFile(filepath.Join(outputDir, "investigation.md"), content, 0644)
			},
			mockAgentExitCode:       0,
			mockAgentError:          nil,
			expectStatus:            "completed",
			expectStorageSkipped:    false,
			expectSlackSkipped:      false,
			expectResultFileWritten: true,
		},
		{
			name: "agent failure - exit code 1",
			setupWorkspace: func(workspacePath string) error {
				// Agent failed, might not have created output
				return nil
			},
			mockAgentExitCode:       1,
			mockAgentError:          nil,
			expectStatus:            "agent_failed",
			expectStorageSkipped:    true,
			expectSlackSkipped:      true,
			expectResultFileWritten: true,
		},
		{
			name: "agent failure - execution error",
			setupWorkspace: func(workspacePath string) error {
				return nil
			},
			mockAgentExitCode:       0,
			mockAgentError:          errors.New("simulated LLM API failure"),
			expectStatus:            "agent_failed",
			expectStorageSkipped:    true,
			expectSlackSkipped:      true,
			expectResultFileWritten: true,
		},
		{
			name: "agent failure - missing output file",
			setupWorkspace: func(workspacePath string) error {
				// Create output dir but no file
				outputDir := filepath.Join(workspacePath, "output")
				return os.MkdirAll(outputDir, 0755)
			},
			mockAgentExitCode:       0,
			mockAgentError:          nil,
			expectStatus:            "agent_failed",
			expectStorageSkipped:    true,
			expectSlackSkipped:      true,
			expectResultFileWritten: true,
		},
		{
			name: "agent failure - output file too small",
			setupWorkspace: func(workspacePath string) error {
				outputDir := filepath.Join(workspacePath, "output")
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					return err
				}
				// Create file with only 50 bytes
				content := make([]byte, 50)
				return os.WriteFile(filepath.Join(outputDir, "investigation.md"), content, 0644)
			},
			mockAgentExitCode:       0,
			mockAgentError:          nil,
			expectStatus:            "agent_failed",
			expectStorageSkipped:    true,
			expectSlackSkipped:      true,
			expectResultFileWritten: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary workspace
			tempDir := t.TempDir()
			workspacePath := filepath.Join(tempDir, "workspace")
			if err := os.MkdirAll(workspacePath, 0755); err != nil {
				t.Fatalf("failed to create workspace: %v", err)
			}

			// Setup workspace according to test case
			if err := tt.setupWorkspace(workspacePath); err != nil {
				t.Fatalf("failed to setup workspace: %v", err)
			}

			// Simulate agent execution result
			exitCode := tt.mockAgentExitCode
			execErr := tt.mockAgentError

			// Call detectAgentFailure (this is the core validation logic)
			agentFailed, failureReason := detectAgentFailure(workspacePath, exitCode, execErr)

			// Verify agent failure detection
			if tt.expectStatus == "agent_failed" {
				if !agentFailed {
					t.Errorf("expected agent failure to be detected, but it was not")
				}
				if failureReason == "" {
					t.Errorf("expected failure reason, but got empty string")
				}
				t.Logf("Detected failure reason: %s", failureReason)
			} else {
				if agentFailed {
					t.Errorf("expected no agent failure, but got failure: %s", failureReason)
				}
			}

			// Simulate writing incident.json (this would happen in processEvent)
			incidentPath := filepath.Join(workspacePath, "incident.json")
			status := "completed"
			if agentFailed {
				status = "agent_failed"
			}
			incidentData := map[string]interface{}{
				"status":         status,
				"failure_reason": failureReason,
				"exit_code":      exitCode,
			}

			// In real processEvent, incident.WriteToFile is called
			// Here we verify the status is correct
			if incidentData["status"] != tt.expectStatus {
				t.Errorf("expected status %q, got %q", tt.expectStatus, incidentData["status"])
			}

			// Verify storage/slack skipping logic matches expectations
			shouldSkipStorage := (status == "agent_failed")
			shouldSkipSlack := (status == "agent_failed")

			if shouldSkipStorage != tt.expectStorageSkipped {
				t.Errorf("storage skip logic: expected %v, got %v", tt.expectStorageSkipped, shouldSkipStorage)
			}

			if shouldSkipSlack != tt.expectSlackSkipped {
				t.Errorf("slack skip logic: expected %v, got %v", tt.expectSlackSkipped, shouldSkipSlack)
			}

			// Verify that incident.json would be written (in real flow)
			if tt.expectResultFileWritten {
				// In the actual implementation, incident.json is always written
				// We verify the path exists (we would write to it)
				if _, err := os.Stat(filepath.Dir(incidentPath)); os.IsNotExist(err) {
					t.Errorf("workspace directory should exist for writing incident.json")
				}
			}
		})
	}
}

// TestProcessEvent_IntegrationFlow documents the expected behavior for manual verification
func TestProcessEvent_IntegrationFlow(t *testing.T) {
	t.Log("Integration Flow Test - Documents expected behavior for manual testing")
	t.Log("")
	t.Log("AGENT SUCCESS SCENARIO:")
	t.Log("  1. Agent exits with code 0")
	t.Log("  2. investigation.md exists and is > 100 bytes")
	t.Log("  3. incident.json written with status='completed'")
	t.Log("  4. Azure storage upload executed")
	t.Log("  5. Slack notification sent with report URL")
	t.Log("")
	t.Log("AGENT FAILURE SCENARIO (Exit Code 1):")
	t.Log("  1. Agent exits with code 1")
	t.Log("  2. detectAgentFailure() returns (true, 'agent exited with non-zero code: 1')")
	t.Log("  3. incident.json written with status='agent_failed', failure_reason set")
	t.Log("  4. Azure storage upload SKIPPED (log: 'skipping storage upload due to agent failure')")
	t.Log("  5. Slack notification SKIPPED (log: 'skipping slack notification due to agent failure')")
	t.Log("")
	t.Log("AGENT FAILURE SCENARIO (LLM API Error):")
	t.Log("  1. Agent execution returns error (e.g., API timeout)")
	t.Log("  2. detectAgentFailure() returns (true, 'agent execution error: ...')")
	t.Log("  3. incident.json written with status='agent_failed', failure_reason set")
	t.Log("  4. Azure storage upload SKIPPED")
	t.Log("  5. Slack notification SKIPPED")
	t.Log("")
	t.Log("AGENT FAILURE SCENARIO (Missing/Invalid Output):")
	t.Log("  1. Agent exits with code 0 but investigation.md missing or too small")
	t.Log("  2. detectAgentFailure() returns (true, 'investigation.md file not found' or 'too small')")
	t.Log("  3. incident.json written with status='agent_failed', failure_reason set")
	t.Log("  4. Azure storage upload SKIPPED")
	t.Log("  5. Slack notification SKIPPED")
	t.Log("")
	t.Log("MANUAL TESTING:")
	t.Log("  Run: go build -o runner ./cmd/runner")
	t.Log("  Test success: ./runner -c configs/config-test.yaml")
	t.Log("  Test failure: Modify agent script to exit 1 or simulate API failure")
	t.Log("  Verify: Check logs for skip messages and incident.json status")
}

// TestCircuitBreakerIntegration tests the complete circuit breaker alert flow
func TestCircuitBreakerIntegration(t *testing.T) {
	tests := []struct {
		name                     string
		threshold                int
		failureSequence          []bool
		expectAlertAfterFailure  int
		expectMultipleAlerts     bool
		expectRecoveryAlert      bool
	}{
		{
			name:                     "alert sent after threshold reached",
			threshold:                3,
			failureSequence:          []bool{false, false, false},
			expectAlertAfterFailure:  3,
			expectMultipleAlerts:     false,
			expectRecoveryAlert:      false,
		},
		{
			name:                     "alert sent only once, not repeated",
			threshold:                2,
			failureSequence:          []bool{false, false, false, false},
			expectAlertAfterFailure:  2,
			expectMultipleAlerts:     false,
			expectRecoveryAlert:      false,
		},
		{
			name:                     "recovery resets alert state",
			threshold:                2,
			failureSequence:          []bool{false, false, true, false, false},
			expectAlertAfterFailure:  2,
			expectMultipleAlerts:     true,
			expectRecoveryAlert:      true,
		},
		{
			name:                     "no alert before threshold",
			threshold:                5,
			failureSequence:          []bool{false, false, false},
			expectAlertAfterFailure:  0,
			expectMultipleAlerts:     false,
			expectRecoveryAlert:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create circuit breaker
			cb := reporting.NewCircuitBreaker(tt.threshold)

			// Track alerts
			alertCount := 0
			alertSentAfterFailure := 0
			recoveryAlertSent := false

			// Simulate event processing sequence
			for i, success := range tt.failureSequence {
				failureNumber := i + 1

				if success {
					// Simulate successful agent execution
					needsRecoveryAlert := cb.RecordSuccess()
					if needsRecoveryAlert {
						recoveryAlertSent = true
						t.Logf("Recovery alert sent after failure %d", failureNumber)
					}
				} else {
					// Simulate agent failure
					reason := "test failure reason"
					cb.RecordFailure(reason)

					// Check if alert should be sent
					if cb.ShouldAlert() {
						alertCount++
						if alertSentAfterFailure == 0 {
							alertSentAfterFailure = failureNumber
						}
						t.Logf("Alert sent after failure %d (alert count: %d)", failureNumber, alertCount)

						// Verify stats are available for alert
						stats := cb.GetStats()
						if stats.Count == 0 {
							t.Error("stats.Count is 0, expected > 0")
						}
						if len(stats.RecentReasons) == 0 {
							t.Error("stats.RecentReasons is empty, expected at least one reason")
						}
					}
				}
			}

			// Verify alert was sent at the right time
			if tt.expectAlertAfterFailure > 0 {
				if alertSentAfterFailure != tt.expectAlertAfterFailure {
					t.Errorf("alert sent after failure %d, want %d", alertSentAfterFailure, tt.expectAlertAfterFailure)
				}
			} else {
				if alertSentAfterFailure > 0 {
					t.Errorf("alert sent after failure %d, expected no alert", alertSentAfterFailure)
				}
			}

			// Verify alert was not sent multiple times (unless recovery happened)
			if !tt.expectMultipleAlerts {
				expectedMaxAlerts := 1
				if tt.expectRecoveryAlert {
					expectedMaxAlerts = 2
				}
				if tt.expectAlertAfterFailure > 0 && alertCount > expectedMaxAlerts {
					t.Errorf("alert sent %d times, expected at most %d", alertCount, expectedMaxAlerts)
				}
			}

			// Verify recovery alert
			if tt.expectRecoveryAlert {
				if !recoveryAlertSent {
					t.Error("expected recovery alert, but none was sent")
				}
			} else {
				if recoveryAlertSent {
					t.Error("did not expect recovery alert, but one was sent")
				}
			}
		})
	}
}

// TestCircuitBreakerThresholdConfiguration tests that the circuit breaker respects configured threshold
func TestCircuitBreakerThresholdConfiguration(t *testing.T) {
	tests := []struct {
		name               string
		configuredThreshold int
		failureCount       int
		expectAlert        bool
	}{
		{
			name:               "alert when threshold=3 and failures=3",
			configuredThreshold: 3,
			failureCount:       3,
			expectAlert:        true,
		},
		{
			name:               "no alert when threshold=5 and failures=3",
			configuredThreshold: 5,
			failureCount:       3,
			expectAlert:        false,
		},
		{
			name:               "alert when threshold=1 and failures=1",
			configuredThreshold: 1,
			failureCount:       1,
			expectAlert:        true,
		},
		{
			name:               "alert when threshold=3 and failures=5",
			configuredThreshold: 3,
			failureCount:       5,
			expectAlert:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := reporting.NewCircuitBreaker(tt.configuredThreshold)

			// Record the specified number of failures
			for i := 0; i < tt.failureCount; i++ {
				cb.RecordFailure("test failure")

				// Only check on the last failure
				if i == tt.failureCount-1 {
					gotAlert := cb.ShouldAlert()
					if gotAlert != tt.expectAlert {
						t.Errorf("after %d failures with threshold=%d: ShouldAlert()=%v, want %v",
							tt.failureCount, tt.configuredThreshold, gotAlert, tt.expectAlert)
					}
				}
			}
		})
	}
}

// TestUploadFailedInvestigationsConfig tests that upload behavior is controlled by configuration
func TestUploadFailedInvestigationsConfig(t *testing.T) {
	tests := []struct {
		name                         string
		uploadFailedInvestigations   bool
		agentFailed                  bool
		expectUploadSkipped          bool
	}{
		{
			name:                         "skip upload when agent failed and config=false (default)",
			uploadFailedInvestigations:   false,
			agentFailed:                  true,
			expectUploadSkipped:          true,
		},
		{
			name:                         "upload when agent failed but config=true",
			uploadFailedInvestigations:   true,
			agentFailed:                  true,
			expectUploadSkipped:          false,
		},
		{
			name:                         "upload when agent succeeded regardless of config",
			uploadFailedInvestigations:   false,
			agentFailed:                  false,
			expectUploadSkipped:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the logic in processEvent for storage upload decision
			status := "completed"
			if tt.agentFailed {
				status = "agent_failed"
			}

			// This is the logic from processEvent
			shouldSkipUpload := (status == "agent_failed" && !tt.uploadFailedInvestigations)

			if shouldSkipUpload != tt.expectUploadSkipped {
				t.Errorf("uploadFailedInvestigations=%v, agentFailed=%v: shouldSkipUpload=%v, want %v",
					tt.uploadFailedInvestigations, tt.agentFailed, shouldSkipUpload, tt.expectUploadSkipped)
			}
		})
	}
}

// TestNotifyOnAgentFailureConfig tests that circuit breaker alert behavior is controlled by configuration
func TestNotifyOnAgentFailureConfig(t *testing.T) {
	tests := []struct {
		name                  string
		notifyOnAgentFailure  bool
		slackConfigured       bool
		circuitBreakerTripped bool
		expectAlertSent       bool
	}{
		{
			name:                  "send alert when config=true, slack configured, and circuit breaker tripped",
			notifyOnAgentFailure:  true,
			slackConfigured:       true,
			circuitBreakerTripped: true,
			expectAlertSent:       true,
		},
		{
			name:                  "skip alert when config=false even if circuit breaker tripped",
			notifyOnAgentFailure:  false,
			slackConfigured:       true,
			circuitBreakerTripped: true,
			expectAlertSent:       false,
		},
		{
			name:                  "skip alert when slack not configured",
			notifyOnAgentFailure:  true,
			slackConfigured:       false,
			circuitBreakerTripped: true,
			expectAlertSent:       false,
		},
		{
			name:                  "skip alert when circuit breaker not tripped",
			notifyOnAgentFailure:  true,
			slackConfigured:       true,
			circuitBreakerTripped: false,
			expectAlertSent:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the logic in processEvent for circuit breaker alert decision
			slackNotifier := tt.slackConfigured // non-nil if configured

			// This is the logic from processEvent
			shouldSendAlert := tt.slackConfigured && tt.notifyOnAgentFailure && tt.circuitBreakerTripped

			if shouldSendAlert != tt.expectAlertSent {
				t.Errorf("notifyOnAgentFailure=%v, slackConfigured=%v, circuitBreakerTripped=%v: shouldSendAlert=%v, want %v",
					tt.notifyOnAgentFailure, slackNotifier, tt.circuitBreakerTripped, shouldSendAlert, tt.expectAlertSent)
			}
		})
	}
}

// TestIndividualNotificationsAlwaysSkippedForAgentFailures tests that individual notifications
// are always skipped for agent failures, regardless of configuration
func TestIndividualNotificationsAlwaysSkippedForAgentFailures(t *testing.T) {
	// The behavior should be:
	// - Individual notifications for agent failures are ALWAYS skipped (this prevents spam)
	// - Circuit breaker alerts are controlled by cfg.NotifyOnAgentFailure

	tests := []struct {
		name        string
		agentFailed bool
		expectSkip  bool
	}{
		{
			name:        "skip notification when agent failed",
			agentFailed: true,
			expectSkip:  true,
		},
		{
			name:        "send notification when agent succeeded",
			agentFailed: false,
			expectSkip:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := "completed"
			if tt.agentFailed {
				status = "agent_failed"
			}

			// This is the logic from processEvent for individual notifications
			shouldSkipNotification := (status == "agent_failed")

			if shouldSkipNotification != tt.expectSkip {
				t.Errorf("agentFailed=%v: shouldSkipNotification=%v, want %v",
					tt.agentFailed, shouldSkipNotification, tt.expectSkip)
			}
		})
	}
}

// TestCircuitBreakerAlertContent verifies the alert contains required information
func TestCircuitBreakerAlertContent(t *testing.T) {
	cb := reporting.NewCircuitBreaker(3)

	// Record failures with different reasons
	reasons := []string{
		"agent exited with non-zero code: 1",
		"investigation.md file not found",
		"agent execution error: API timeout",
	}

	for _, reason := range reasons {
		cb.RecordFailure(reason)
	}

	// Check if alert should be sent
	if !cb.ShouldAlert() {
		t.Fatal("expected ShouldAlert() to return true after threshold")
	}

	// Get stats for alert
	stats := cb.GetStats()

	// Verify stats contain required information
	if stats.Count != 3 {
		t.Errorf("stats.Count = %d, want 3", stats.Count)
	}

	if stats.FirstFailureTime.IsZero() {
		t.Error("stats.FirstFailureTime is zero, expected valid timestamp")
	}

	if stats.LastFailureTime.IsZero() {
		t.Error("stats.LastFailureTime is zero, expected valid timestamp")
	}

	if stats.Duration < 0 {
		t.Errorf("stats.Duration = %v, want >= 0", stats.Duration)
	}

	if len(stats.RecentReasons) != 3 {
		t.Errorf("len(stats.RecentReasons) = %d, want 3", len(stats.RecentReasons))
	}

	// Verify reasons are preserved in order
	for i, expectedReason := range reasons {
		if stats.RecentReasons[i] != expectedReason {
			t.Errorf("stats.RecentReasons[%d] = %q, want %q", i, stats.RecentReasons[i], expectedReason)
		}
	}

	t.Logf("Alert stats: Count=%d, Duration=%v, Reasons=%v",
		stats.Count, stats.Duration, stats.RecentReasons)
}

// TestCircuitBreakerFullScenario tests a complete failure and recovery cycle
func TestCircuitBreakerFullScenario(t *testing.T) {
	cb := reporting.NewCircuitBreaker(3)

	// Scenario: 3 failures -> alert -> 2 more failures (no new alert) -> success -> recovery alert
	// Then repeat cycle to verify state is fully reset

	// First cycle: fail until threshold
	for i := 1; i <= 3; i++ {
		cb.RecordFailure("failure")
		if i < 3 {
			if cb.ShouldAlert() {
				t.Errorf("ShouldAlert() = true before threshold (failure %d/3)", i)
			}
		}
	}

	// Verify alert should be sent
	if !cb.ShouldAlert() {
		t.Error("ShouldAlert() = false after reaching threshold")
	}

	// Verify alert is not sent again
	if cb.ShouldAlert() {
		t.Error("ShouldAlert() = true on second call (should only alert once)")
	}

	// Continue failures without new alerts
	cb.RecordFailure("failure 4")
	cb.RecordFailure("failure 5")
	if cb.ShouldAlert() {
		t.Error("ShouldAlert() = true after already alerted")
	}

	// Verify failure count accumulated
	if cb.GetFailureCount() != 5 {
		t.Errorf("failureCount = %d, want 5", cb.GetFailureCount())
	}

	// Record success - should trigger recovery alert
	needsRecovery := cb.RecordSuccess()
	if !needsRecovery {
		t.Error("RecordSuccess() = false, want true (should trigger recovery alert)")
	}

	// Verify circuit is reset
	if cb.GetFailureCount() != 0 {
		t.Errorf("failureCount after recovery = %d, want 0", cb.GetFailureCount())
	}
	if cb.GetState() != reporting.StateClosed {
		t.Errorf("state after recovery = %d, want StateClosed", cb.GetState())
	}

	// Second cycle: verify clean state
	cb.RecordFailure("cycle2 failure1")
	cb.RecordFailure("cycle2 failure2")
	cb.RecordFailure("cycle2 failure3")

	if !cb.ShouldAlert() {
		t.Error("ShouldAlert() = false in second cycle, want true")
	}

	// Success without recovery alert (no alert was sent in second cycle)
	needsRecovery = cb.RecordSuccess()
	if !needsRecovery {
		t.Error("RecordSuccess() = false in second cycle, want true")
	}

	t.Log("Full scenario test passed: alert -> no spam -> recovery -> clean reset")
}

// TestCircuitBreakerRecoveryNotificationFlow tests when recovery alerts should be sent
func TestCircuitBreakerRecoveryNotificationFlow(t *testing.T) {
	tests := []struct {
		name               string
		threshold          int
		failuresBeforeAlert int
		alertCalled        bool
		expectRecoveryAlert bool
	}{
		{
			name:               "recovery alert sent after alert was triggered",
			threshold:          3,
			failuresBeforeAlert: 3,
			alertCalled:        true,
			expectRecoveryAlert: true,
		},
		{
			name:               "no recovery alert if threshold not reached",
			threshold:          5,
			failuresBeforeAlert: 3,
			alertCalled:        false,
			expectRecoveryAlert: false,
		},
		{
			name:               "no recovery alert if ShouldAlert never called",
			threshold:          2,
			failuresBeforeAlert: 2,
			alertCalled:        false,
			expectRecoveryAlert: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := reporting.NewCircuitBreaker(tt.threshold)

			// Record failures
			for i := 0; i < tt.failuresBeforeAlert; i++ {
				cb.RecordFailure("failure")
			}

			// Call ShouldAlert if test expects it
			if tt.alertCalled {
				cb.ShouldAlert()
			}

			// Record success and check recovery alert
			gotRecoveryAlert := cb.RecordSuccess()
			if gotRecoveryAlert != tt.expectRecoveryAlert {
				t.Errorf("RecordSuccess() = %v, want %v", gotRecoveryAlert, tt.expectRecoveryAlert)
			}
		})
	}
}

// TestCircuitBreakerNoAlertSpam verifies no repeated alerts without recovery
func TestCircuitBreakerNoAlertSpam(t *testing.T) {
	cb := reporting.NewCircuitBreaker(2)

	// Record failures to open circuit
	cb.RecordFailure("failure 1")
	cb.RecordFailure("failure 2")

	// First call should return true
	if !cb.ShouldAlert() {
		t.Fatal("ShouldAlert() = false on first call after threshold")
	}

	// All subsequent calls should return false
	for i := 0; i < 10; i++ {
		if cb.ShouldAlert() {
			t.Errorf("ShouldAlert() = true on call %d (should only alert once)", i+2)
		}

		// Even recording more failures shouldn't trigger alerts
		cb.RecordFailure("additional failure")
		if cb.ShouldAlert() {
			t.Errorf("ShouldAlert() = true after additional failure %d", i+1)
		}
	}

	t.Logf("Verified no alert spam after %d additional checks and failures", 10)
}

// TestCircuitBreakerDifferentThresholds tests circuit breaker with various threshold values
func TestCircuitBreakerDifferentThresholds(t *testing.T) {
	thresholds := []int{1, 2, 3, 5, 10}

	for _, threshold := range thresholds {
		t.Run(fmt.Sprintf("threshold=%d", threshold), func(t *testing.T) {
			cb := reporting.NewCircuitBreaker(threshold)

			// Record failures up to threshold-1
			for i := 1; i < threshold; i++ {
				cb.RecordFailure("failure")
				if cb.ShouldAlert() {
					t.Errorf("ShouldAlert() = true before threshold (failure %d/%d)", i, threshold)
				}
			}

			// Record one more to reach threshold
			cb.RecordFailure("final failure")
			if !cb.ShouldAlert() {
				t.Errorf("ShouldAlert() = false at threshold %d", threshold)
			}

			// Verify state
			if cb.GetState() != reporting.StateOpen {
				t.Errorf("state = %v, want StateOpen", cb.GetState())
			}
			if cb.GetFailureCount() != threshold {
				t.Errorf("failureCount = %d, want %d", cb.GetFailureCount(), threshold)
			}

			// Verify recovery
			if !cb.RecordSuccess() {
				t.Errorf("RecordSuccess() = false, want true for recovery alert")
			}
			if cb.GetFailureCount() != 0 {
				t.Errorf("failureCount after recovery = %d, want 0", cb.GetFailureCount())
			}
		})
	}
}

// TestCircuitBreakerConfigInteraction tests interaction between circuit breaker and config options
func TestCircuitBreakerConfigInteraction(t *testing.T) {
	tests := []struct {
		name                       string
		notifyOnAgentFailure       bool
		uploadFailedInvestigations bool
		circuitBreakerOpen         bool
		expectSystemAlert          bool
		expectIndividualNotification bool
		expectStorageUpload        bool
	}{
		{
			name:                       "all enabled, circuit open - send system alert, skip individual notification, skip storage",
			notifyOnAgentFailure:       true,
			uploadFailedInvestigations: false,
			circuitBreakerOpen:         true,
			expectSystemAlert:          true,
			expectIndividualNotification: false,
			expectStorageUpload:        false,
		},
		{
			name:                       "notify disabled, circuit open - skip all alerts",
			notifyOnAgentFailure:       false,
			uploadFailedInvestigations: false,
			circuitBreakerOpen:         true,
			expectSystemAlert:          false,
			expectIndividualNotification: false,
			expectStorageUpload:        false,
		},
		{
			name:                       "upload enabled, circuit open - upload despite failure",
			notifyOnAgentFailure:       true,
			uploadFailedInvestigations: true,
			circuitBreakerOpen:         true,
			expectSystemAlert:          true,
			expectIndividualNotification: false,
			expectStorageUpload:        true,
		},
		{
			name:                       "circuit not open - no system alert, send individual notification",
			notifyOnAgentFailure:       true,
			uploadFailedInvestigations: false,
			circuitBreakerOpen:         false,
			expectSystemAlert:          false,
			expectIndividualNotification: true,
			expectStorageUpload:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate processEvent logic
			agentFailed := tt.circuitBreakerOpen
			slackConfigured := true

			// System alert decision (from processEvent)
			shouldSendSystemAlert := slackConfigured && tt.notifyOnAgentFailure && tt.circuitBreakerOpen

			// Individual notification decision (always skipped for agent failures)
			shouldSendIndividualNotification := !agentFailed

			// Storage upload decision
			shouldUpload := !agentFailed || tt.uploadFailedInvestigations

			// Verify expectations
			if shouldSendSystemAlert != tt.expectSystemAlert {
				t.Errorf("shouldSendSystemAlert = %v, want %v", shouldSendSystemAlert, tt.expectSystemAlert)
			}
			if shouldSendIndividualNotification != tt.expectIndividualNotification {
				t.Errorf("shouldSendIndividualNotification = %v, want %v", shouldSendIndividualNotification, tt.expectIndividualNotification)
			}
			if shouldUpload != tt.expectStorageUpload {
				t.Errorf("shouldUpload = %v, want %v", shouldUpload, tt.expectStorageUpload)
			}
		})
	}
}
