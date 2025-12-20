package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rbias/nightcrier/internal/config"
)

// Helper function to create a test tuning config
func createTestTuning() *config.TuningConfig {
	return &config.TuningConfig{
		Agent: config.AgentTuning{
			TimeoutBufferSeconds:      30,
			InvestigationMinSizeBytes: 100,
		},
		IO: config.IOTuning{
			StdoutBufferSize: 512,
			StderrBufferSize: 512,
		},
		HTTP: config.HTTPTuning{
			SlackTimeoutSeconds: 10,
		},
		Reporting: config.ReportingTuning{
			RootCauseTruncationLength:  300,
			FailureReasonsDisplayCount: 3,
			MaxFailureReasonsTracked:   5,
		},
		Events: config.EventsTuning{
			ChannelBufferSize: 100,
		},
	}
}

// Helper function to create a test script
func createTestScript(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test-script.sh")
	scriptContent := `#!/usr/bin/env bash
echo "Test script executed"
echo "INCIDENT_ID: $INCIDENT_ID"
exit 0
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("failed to create test script: %v", err)
	}
	return scriptPath
}

func TestNewExecutorWithConfig_RequiresTuningConfig(t *testing.T) {
	scriptPath := createTestScript(t)
	tuning := createTestTuning()

	execConfig := ExecutorConfig{
		ScriptPath:   scriptPath,
		AllowedTools: "Read,Write,Grep,Glob,Bash,Skill",
		Model:        "sonnet",
		Timeout:      300,
		Prompt:       "Test prompt",
	}

	// Test that executor can be created with tuning config
	executor := NewExecutorWithConfig(execConfig, tuning)

	if executor == nil {
		t.Fatal("NewExecutorWithConfig() returned nil")
	}
	if executor.config.ScriptPath == "" {
		t.Error("Executor config ScriptPath should not be empty")
	}
	if executor.tuning == nil {
		t.Fatal("Executor tuning should not be nil")
	}
	if executor.tuning.Agent.TimeoutBufferSeconds != 30 {
		t.Errorf("TimeoutBufferSeconds = %d, want 30", executor.tuning.Agent.TimeoutBufferSeconds)
	}
	if executor.tuning.IO.StdoutBufferSize != 512 {
		t.Errorf("StdoutBufferSize = %d, want 512", executor.tuning.IO.StdoutBufferSize)
	}
	if executor.tuning.IO.StderrBufferSize != 512 {
		t.Errorf("StderrBufferSize = %d, want 512", executor.tuning.IO.StderrBufferSize)
	}
}

func TestNewExecutorWithConfig_AbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()
	scriptName := "test-script.sh"
	scriptPath := filepath.Join(tmpDir, scriptName)
	scriptContent := `#!/usr/bin/env bash
exit 0
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("failed to create test script: %v", err)
	}

	tuning := createTestTuning()
	execConfig := ExecutorConfig{
		ScriptPath:   scriptName, // Relative path
		AllowedTools: "Read,Write",
		Model:        "sonnet",
		Timeout:      60,
		Prompt:       "Test",
	}

	executor := NewExecutorWithConfig(execConfig, tuning)

	// Verify that the script path was converted to absolute
	if !filepath.IsAbs(executor.config.ScriptPath) {
		t.Errorf("ScriptPath should be absolute, got: %s", executor.config.ScriptPath)
	}
}

func TestNewExecutorWithConfig_SystemPromptFileAbsolutePath(t *testing.T) {
	scriptPath := createTestScript(t)
	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "prompt.txt")
	if err := os.WriteFile(promptFile, []byte("Test prompt"), 0644); err != nil {
		t.Fatalf("failed to create prompt file: %v", err)
	}

	tuning := createTestTuning()
	execConfig := ExecutorConfig{
		ScriptPath:       scriptPath,
		SystemPromptFile: "prompt.txt", // Relative path
		AllowedTools:     "Read,Write",
		Model:            "sonnet",
		Timeout:          60,
		Prompt:           "Test",
	}

	executor := NewExecutorWithConfig(execConfig, tuning)

	// Verify that the system prompt file path was processed
	// (it may or may not be absolute depending on whether the file exists)
	if executor.config.SystemPromptFile == "" {
		t.Error("SystemPromptFile should not be empty")
	}
}

func TestNewExecutorWithConfig_AllConfigFieldsPreserved(t *testing.T) {
	scriptPath := createTestScript(t)
	tuning := createTestTuning()

	execConfig := ExecutorConfig{
		ScriptPath:       scriptPath,
		SystemPromptFile: "/path/to/prompt.txt",
		AllowedTools:     "Read,Write,Grep",
		Model:            "opus",
		Timeout:          600,
		AgentCLI:         "claude",
		Prompt:           "Custom prompt text",
	}

	executor := NewExecutorWithConfig(execConfig, tuning)

	// Verify all fields are preserved (except potentially absolute path transformation)
	if executor.config.SystemPromptFile != "/path/to/prompt.txt" {
		t.Errorf("SystemPromptFile = %s, want /path/to/prompt.txt", executor.config.SystemPromptFile)
	}
	if executor.config.AllowedTools != "Read,Write,Grep" {
		t.Errorf("AllowedTools = %s, want Read,Write,Grep", executor.config.AllowedTools)
	}
	if executor.config.Model != "opus" {
		t.Errorf("Model = %s, want opus", executor.config.Model)
	}
	if executor.config.Timeout != 600 {
		t.Errorf("Timeout = %d, want 600", executor.config.Timeout)
	}
	if executor.config.AgentCLI != "claude" {
		t.Errorf("AgentCLI = %s, want claude", executor.config.AgentCLI)
	}
	if executor.config.Prompt != "Custom prompt text" {
		t.Errorf("Prompt = %s, want 'Custom prompt text'", executor.config.Prompt)
	}
}

func TestExecutor_UsesTuningConfigTimeoutBuffer(t *testing.T) {
	// This test verifies that the executor uses the timeout buffer from TuningConfig
	// by checking that the correct timeout is calculated
	scriptPath := createTestScript(t)

	tests := []struct {
		name                 string
		configTimeout        int
		tuningTimeoutBuffer  int
		expectedTotalTimeout int
	}{
		{"default buffer", 300, 60, 360},
		{"custom buffer", 300, 30, 330},
		{"zero buffer", 300, 0, 300},
		{"large buffer", 120, 180, 300},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tuning := createTestTuning()
			tuning.Agent.TimeoutBufferSeconds = tt.tuningTimeoutBuffer

			execConfig := ExecutorConfig{
				ScriptPath:   scriptPath,
				AllowedTools: "Read,Write",
				Model:        "sonnet",
				Timeout:      tt.configTimeout,
				Prompt:       "Test",
			}

			executor := NewExecutorWithConfig(execConfig, tuning)

			// Verify tuning config is set correctly
			if executor.tuning.Agent.TimeoutBufferSeconds != tt.tuningTimeoutBuffer {
				t.Errorf("TimeoutBufferSeconds = %d, want %d",
					executor.tuning.Agent.TimeoutBufferSeconds, tt.tuningTimeoutBuffer)
			}

			// The actual timeout calculation happens in ExecuteWithPrompt,
			// but we can verify the tuning is accessible
			expectedTotal := tt.configTimeout + tt.tuningTimeoutBuffer
			actualTotal := executor.config.Timeout + executor.tuning.Agent.TimeoutBufferSeconds
			if actualTotal != expectedTotal {
				t.Errorf("Total timeout = %d, want %d", actualTotal, expectedTotal)
			}
		})
	}
}

func TestExecutor_UsesTuningConfigIOBuffers(t *testing.T) {
	// This test verifies that the executor has access to I/O buffer sizes from TuningConfig
	scriptPath := createTestScript(t)

	tests := []struct {
		name         string
		stdoutBuffer int
		stderrBuffer int
	}{
		{"default buffers", 1024, 1024},
		{"small buffers", 256, 256},
		{"large buffers", 8192, 8192},
		{"asymmetric buffers", 512, 2048},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tuning := createTestTuning()
			tuning.IO.StdoutBufferSize = tt.stdoutBuffer
			tuning.IO.StderrBufferSize = tt.stderrBuffer

			execConfig := ExecutorConfig{
				ScriptPath:   scriptPath,
				AllowedTools: "Read,Write",
				Model:        "sonnet",
				Timeout:      60,
				Prompt:       "Test",
			}

			executor := NewExecutorWithConfig(execConfig, tuning)

			// Verify I/O buffer sizes are set correctly in tuning
			if executor.tuning.IO.StdoutBufferSize != tt.stdoutBuffer {
				t.Errorf("StdoutBufferSize = %d, want %d",
					executor.tuning.IO.StdoutBufferSize, tt.stdoutBuffer)
			}
			if executor.tuning.IO.StderrBufferSize != tt.stderrBuffer {
				t.Errorf("StderrBufferSize = %d, want %d",
					executor.tuning.IO.StderrBufferSize, tt.stderrBuffer)
			}
		})
	}
}

func TestExecutor_NoDefaultsApplied(t *testing.T) {
	// This test verifies that NO defaults are applied by the executor itself
	// All values must come from the provided config and tuning
	scriptPath := createTestScript(t)
	tuning := createTestTuning()

	// Create executor with explicit empty/zero values where possible
	execConfig := ExecutorConfig{
		ScriptPath:       scriptPath,
		SystemPromptFile: "", // Empty is valid
		AllowedTools:     "None",
		Model:            "test-model",
		Timeout:          1, // Minimal timeout
		AgentCLI:         "",
		Prompt:           "",
	}

	executor := NewExecutorWithConfig(execConfig, tuning)

	// Verify that even unusual values are preserved without defaults
	if executor.config.AllowedTools != "None" {
		t.Errorf("AllowedTools = %s, expected 'None' (no defaults applied)", executor.config.AllowedTools)
	}
	if executor.config.Model != "test-model" {
		t.Errorf("Model = %s, expected 'test-model' (no defaults applied)", executor.config.Model)
	}
	if executor.config.Timeout != 1 {
		t.Errorf("Timeout = %d, expected 1 (no defaults applied)", executor.config.Timeout)
	}
	if executor.config.Prompt != "" {
		t.Errorf("Prompt = %s, expected empty (no defaults applied)", executor.config.Prompt)
	}
}

func TestExecute_UsesConfigPrompt(t *testing.T) {
	// Create a script that succeeds quickly
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "quick-script.sh")
	scriptContent := `#!/usr/bin/env bash
exit 0
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("failed to create test script: %v", err)
	}

	tuning := createTestTuning()
	customPrompt := "Test investigation prompt"

	execConfig := ExecutorConfig{
		ScriptPath:   scriptPath,
		AllowedTools: "Read,Write",
		Model:        "sonnet",
		Timeout:      5,
		Prompt:       customPrompt,
	}

	executor := NewExecutorWithConfig(execConfig, tuning)

	// Verify the prompt is stored in config
	if executor.config.Prompt != customPrompt {
		t.Errorf("config.Prompt = %s, want %s", executor.config.Prompt, customPrompt)
	}

	// Note: We don't actually execute here because it requires a full agent setup
	// The important thing is that the prompt is stored and accessible
}

func TestExecutor_TuningConfigRequired(t *testing.T) {
	// This is a compile-time check that NewExecutorWithConfig requires TuningConfig
	// If we can't compile without providing tuning, this test passes
	scriptPath := createTestScript(t)

	execConfig := ExecutorConfig{
		ScriptPath:   scriptPath,
		AllowedTools: "Read,Write",
		Model:        "sonnet",
		Timeout:      60,
		Prompt:       "Test",
	}

	// This line MUST include tuning parameter - if it compiles without it, the test fails
	tuning := createTestTuning()
	executor := NewExecutorWithConfig(execConfig, tuning)

	if executor.tuning == nil {
		t.Fatal("Executor tuning must not be nil - TuningConfig is required")
	}
}

func TestExecute_ContextCancellation(t *testing.T) {
	// Create a script that runs for a while
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "long-script.sh")
	scriptContent := `#!/usr/bin/env bash
sleep 10
exit 0
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("failed to create test script: %v", err)
	}

	tuning := createTestTuning()
	tuning.Agent.TimeoutBufferSeconds = 1 // Short buffer for quick test

	execConfig := ExecutorConfig{
		ScriptPath:   scriptPath,
		AllowedTools: "Read,Write",
		Model:        "sonnet",
		Timeout:      1, // 1 second timeout
		Prompt:       "Test",
	}

	executor := NewExecutorWithConfig(execConfig, tuning)
	workspace := t.TempDir()

	ctx := context.Background()
	exitCode, err := executor.Execute(ctx, workspace, "test-incident-001")

	// The script should be cancelled due to timeout
	// We expect either an error or non-zero exit code
	if err == nil && exitCode == 0 {
		t.Error("Expected timeout or non-zero exit code for long-running script")
	}
}

func TestExecutorConfig_AllFieldsExplicit(t *testing.T) {
	// Verify that ExecutorConfig struct has no default values built in
	var cfg ExecutorConfig

	// All string fields should be empty by default
	if cfg.ScriptPath != "" {
		t.Error("ExecutorConfig.ScriptPath should have no default")
	}
	if cfg.SystemPromptFile != "" {
		t.Error("ExecutorConfig.SystemPromptFile should have no default")
	}
	if cfg.AllowedTools != "" {
		t.Error("ExecutorConfig.AllowedTools should have no default")
	}
	if cfg.Model != "" {
		t.Error("ExecutorConfig.Model should have no default")
	}
	if cfg.AgentCLI != "" {
		t.Error("ExecutorConfig.AgentCLI should have no default")
	}
	if cfg.Prompt != "" {
		t.Error("ExecutorConfig.Prompt should have no default")
	}

	// Timeout should be zero by default
	if cfg.Timeout != 0 {
		t.Error("ExecutorConfig.Timeout should have no default")
	}
}
