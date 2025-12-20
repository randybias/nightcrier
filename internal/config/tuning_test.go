package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTuning_WithDefaults(t *testing.T) {
	// Load tuning config when no file exists - should return defaults
	tuning, err := LoadTuning()
	if err != nil {
		t.Fatalf("LoadTuning() failed: %v", err)
	}

	// Verify HTTP defaults
	if tuning.HTTP.SlackTimeoutSeconds != 10 {
		t.Errorf("HTTP.SlackTimeoutSeconds = %d, want 10", tuning.HTTP.SlackTimeoutSeconds)
	}

	// Verify Agent defaults
	if tuning.Agent.TimeoutBufferSeconds != 60 {
		t.Errorf("Agent.TimeoutBufferSeconds = %d, want 60", tuning.Agent.TimeoutBufferSeconds)
	}
	if tuning.Agent.InvestigationMinSizeBytes != 100 {
		t.Errorf("Agent.InvestigationMinSizeBytes = %d, want 100", tuning.Agent.InvestigationMinSizeBytes)
	}

	// Verify Reporting defaults
	if tuning.Reporting.RootCauseTruncationLength != 300 {
		t.Errorf("Reporting.RootCauseTruncationLength = %d, want 300", tuning.Reporting.RootCauseTruncationLength)
	}
	if tuning.Reporting.FailureReasonsDisplayCount != 3 {
		t.Errorf("Reporting.FailureReasonsDisplayCount = %d, want 3", tuning.Reporting.FailureReasonsDisplayCount)
	}
	if tuning.Reporting.MaxFailureReasonsTracked != 5 {
		t.Errorf("Reporting.MaxFailureReasonsTracked = %d, want 5", tuning.Reporting.MaxFailureReasonsTracked)
	}

	// Verify Events defaults
	if tuning.Events.ChannelBufferSize != 100 {
		t.Errorf("Events.ChannelBufferSize = %d, want 100", tuning.Events.ChannelBufferSize)
	}

	// Verify IO defaults
	if tuning.IO.StdoutBufferSize != 1024 {
		t.Errorf("IO.StdoutBufferSize = %d, want 1024", tuning.IO.StdoutBufferSize)
	}
	if tuning.IO.StderrBufferSize != 1024 {
		t.Errorf("IO.StderrBufferSize = %d, want 1024", tuning.IO.StderrBufferSize)
	}
}

func TestLoadTuningWithFile_ValidConfig(t *testing.T) {
	// Create temp tuning file with custom values
	tmpDir := t.TempDir()
	tuningPath := filepath.Join(tmpDir, "tuning.yaml")
	tuningContent := `
http:
  slack_timeout_seconds: 20

agent:
  timeout_buffer_seconds: 120
  investigation_min_size_bytes: 200

reporting:
  root_cause_truncation_length: 500
  failure_reasons_display_count: 5
  max_failure_reasons_tracked: 10

events:
  channel_buffer_size: 200

io:
  stdout_buffer_size: 2048
  stderr_buffer_size: 2048
`
	if err := os.WriteFile(tuningPath, []byte(tuningContent), 0644); err != nil {
		t.Fatalf("failed to write tuning file: %v", err)
	}

	tuning, err := LoadTuningWithFile(tuningPath)
	if err != nil {
		t.Fatalf("LoadTuningWithFile() failed: %v", err)
	}

	// Verify HTTP values
	if tuning.HTTP.SlackTimeoutSeconds != 20 {
		t.Errorf("HTTP.SlackTimeoutSeconds = %d, want 20", tuning.HTTP.SlackTimeoutSeconds)
	}

	// Verify Agent values
	if tuning.Agent.TimeoutBufferSeconds != 120 {
		t.Errorf("Agent.TimeoutBufferSeconds = %d, want 120", tuning.Agent.TimeoutBufferSeconds)
	}
	if tuning.Agent.InvestigationMinSizeBytes != 200 {
		t.Errorf("Agent.InvestigationMinSizeBytes = %d, want 200", tuning.Agent.InvestigationMinSizeBytes)
	}

	// Verify Reporting values
	if tuning.Reporting.RootCauseTruncationLength != 500 {
		t.Errorf("Reporting.RootCauseTruncationLength = %d, want 500", tuning.Reporting.RootCauseTruncationLength)
	}
	if tuning.Reporting.FailureReasonsDisplayCount != 5 {
		t.Errorf("Reporting.FailureReasonsDisplayCount = %d, want 5", tuning.Reporting.FailureReasonsDisplayCount)
	}
	if tuning.Reporting.MaxFailureReasonsTracked != 10 {
		t.Errorf("Reporting.MaxFailureReasonsTracked = %d, want 10", tuning.Reporting.MaxFailureReasonsTracked)
	}

	// Verify Events values
	if tuning.Events.ChannelBufferSize != 200 {
		t.Errorf("Events.ChannelBufferSize = %d, want 200", tuning.Events.ChannelBufferSize)
	}

	// Verify IO values
	if tuning.IO.StdoutBufferSize != 2048 {
		t.Errorf("IO.StdoutBufferSize = %d, want 2048", tuning.IO.StdoutBufferSize)
	}
	if tuning.IO.StderrBufferSize != 2048 {
		t.Errorf("IO.StderrBufferSize = %d, want 2048", tuning.IO.StderrBufferSize)
	}
}

func TestLoadTuningWithFile_PartialConfig(t *testing.T) {
	// Create tuning file with only some values specified
	tmpDir := t.TempDir()
	tuningPath := filepath.Join(tmpDir, "tuning.yaml")
	tuningContent := `
http:
  slack_timeout_seconds: 15

reporting:
  root_cause_truncation_length: 400
`
	if err := os.WriteFile(tuningPath, []byte(tuningContent), 0644); err != nil {
		t.Fatalf("failed to write tuning file: %v", err)
	}

	tuning, err := LoadTuningWithFile(tuningPath)
	if err != nil {
		t.Fatalf("LoadTuningWithFile() failed: %v", err)
	}

	// Verify specified values are loaded
	if tuning.HTTP.SlackTimeoutSeconds != 15 {
		t.Errorf("HTTP.SlackTimeoutSeconds = %d, want 15", tuning.HTTP.SlackTimeoutSeconds)
	}
	if tuning.Reporting.RootCauseTruncationLength != 400 {
		t.Errorf("Reporting.RootCauseTruncationLength = %d, want 400", tuning.Reporting.RootCauseTruncationLength)
	}

	// Verify unspecified values use defaults
	if tuning.Agent.TimeoutBufferSeconds != 60 {
		t.Errorf("Agent.TimeoutBufferSeconds = %d, want 60 (default)", tuning.Agent.TimeoutBufferSeconds)
	}
	if tuning.Events.ChannelBufferSize != 100 {
		t.Errorf("Events.ChannelBufferSize = %d, want 100 (default)", tuning.Events.ChannelBufferSize)
	}
	if tuning.IO.StdoutBufferSize != 1024 {
		t.Errorf("IO.StdoutBufferSize = %d, want 1024 (default)", tuning.IO.StdoutBufferSize)
	}
}

func TestLoadTuningWithFile_FileNotFound(t *testing.T) {
	// Try to load from non-existent file - should return defaults without error
	tuning, err := LoadTuningWithFile("/nonexistent/path/tuning.yaml")
	if err != nil {
		t.Fatalf("LoadTuningWithFile() should not error on missing file: %v", err)
	}

	// Should return defaults
	if tuning.HTTP.SlackTimeoutSeconds != 10 {
		t.Errorf("HTTP.SlackTimeoutSeconds = %d, want 10 (default)", tuning.HTTP.SlackTimeoutSeconds)
	}
	if tuning.Agent.TimeoutBufferSeconds != 60 {
		t.Errorf("Agent.TimeoutBufferSeconds = %d, want 60 (default)", tuning.Agent.TimeoutBufferSeconds)
	}
}

func TestLoadTuningWithFile_InvalidYAML(t *testing.T) {
	// Create tuning file with invalid YAML
	tmpDir := t.TempDir()
	tuningPath := filepath.Join(tmpDir, "tuning.yaml")
	tuningContent := `
http:
  slack_timeout_seconds: [this is not valid
`
	if err := os.WriteFile(tuningPath, []byte(tuningContent), 0644); err != nil {
		t.Fatalf("failed to write tuning file: %v", err)
	}

	_, err := LoadTuningWithFile(tuningPath)
	if err == nil {
		t.Error("LoadTuningWithFile() should fail with invalid YAML")
	}
}

func TestValidate_HTTPSlackTimeout(t *testing.T) {
	tests := []struct {
		name    string
		value   int
		wantErr bool
	}{
		{"valid: 1", 1, false},
		{"valid: 10", 10, false},
		{"valid: 100", 100, false},
		{"invalid: 0", 0, true},
		{"invalid: -1", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tuning := defaultTuning()
			tuning.HTTP.SlackTimeoutSeconds = tt.value

			err := tuning.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_AgentTimeoutBuffer(t *testing.T) {
	tests := []struct {
		name    string
		value   int
		wantErr bool
	}{
		{"valid: 0", 0, false},
		{"valid: 60", 60, false},
		{"valid: 120", 120, false},
		{"invalid: -1", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tuning := defaultTuning()
			tuning.Agent.TimeoutBufferSeconds = tt.value

			err := tuning.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_AgentInvestigationMinSize(t *testing.T) {
	tests := []struct {
		name    string
		value   int
		wantErr bool
	}{
		{"valid: 0", 0, false},
		{"valid: 100", 100, false},
		{"valid: 1000", 1000, false},
		{"invalid: -1", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tuning := defaultTuning()
			tuning.Agent.InvestigationMinSizeBytes = tt.value

			err := tuning.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_ReportingTruncationLength(t *testing.T) {
	tests := []struct {
		name    string
		value   int
		wantErr bool
	}{
		{"valid: 1", 1, false},
		{"valid: 300", 300, false},
		{"valid: 1000", 1000, false},
		{"invalid: 0", 0, true},
		{"invalid: -1", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tuning := defaultTuning()
			tuning.Reporting.RootCauseTruncationLength = tt.value

			err := tuning.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_ReportingDisplayCount(t *testing.T) {
	tests := []struct {
		name    string
		value   int
		wantErr bool
	}{
		{"valid: 1", 1, false},
		{"valid: 3", 3, false},
		{"valid: 10", 10, false},
		{"invalid: 0", 0, true},
		{"invalid: -1", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tuning := defaultTuning()
			tuning.Reporting.FailureReasonsDisplayCount = tt.value
			// Ensure tracked >= display for valid test cases
			if tt.value > 0 {
				tuning.Reporting.MaxFailureReasonsTracked = tt.value
			}

			err := tuning.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_ReportingTrackedVsDisplay(t *testing.T) {
	tests := []struct {
		name       string
		display    int
		tracked    int
		wantErr    bool
		errMessage string
	}{
		{"valid: tracked = display", 3, 3, false, ""},
		{"valid: tracked > display", 3, 5, false, ""},
		{"invalid: tracked < display", 5, 3, true, "must be >= failure_reasons_display_count"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tuning := defaultTuning()
			tuning.Reporting.FailureReasonsDisplayCount = tt.display
			tuning.Reporting.MaxFailureReasonsTracked = tt.tracked

			err := tuning.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMessage != "" {
				if !contains(err.Error(), tt.errMessage) {
					t.Errorf("error message should contain %q, got: %v", tt.errMessage, err)
				}
			}
		})
	}
}

func TestValidate_EventsChannelBufferSize(t *testing.T) {
	tests := []struct {
		name    string
		value   int
		wantErr bool
	}{
		{"valid: 1", 1, false},
		{"valid: 100", 100, false},
		{"valid: 1000", 1000, false},
		{"invalid: 0", 0, true},
		{"invalid: -1", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tuning := defaultTuning()
			tuning.Events.ChannelBufferSize = tt.value

			err := tuning.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_IOBufferSizes(t *testing.T) {
	tests := []struct {
		name       string
		bufferType string
		value      int
		wantErr    bool
	}{
		{"valid stdout: 1", "stdout", 1, false},
		{"valid stdout: 1024", "stdout", 1024, false},
		{"valid stdout: 8192", "stdout", 8192, false},
		{"invalid stdout: 0", "stdout", 0, true},
		{"invalid stdout: -1", "stdout", -1, true},
		{"valid stderr: 1", "stderr", 1, false},
		{"valid stderr: 1024", "stderr", 1024, false},
		{"valid stderr: 8192", "stderr", 8192, false},
		{"invalid stderr: 0", "stderr", 0, true},
		{"invalid stderr: -1", "stderr", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tuning := defaultTuning()
			if tt.bufferType == "stdout" {
				tuning.IO.StdoutBufferSize = tt.value
			} else {
				tuning.IO.StderrBufferSize = tt.value
			}

			err := tuning.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadTuningWithFile_ValidationFailures(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		errContains string
	}{
		{
			name: "negative slack timeout",
			config: `
http:
  slack_timeout_seconds: -1
`,
			errContains: "slack_timeout_seconds must be >= 1",
		},
		{
			name: "negative timeout buffer",
			config: `
agent:
  timeout_buffer_seconds: -1
`,
			errContains: "timeout_buffer_seconds must be >= 0",
		},
		{
			name: "zero truncation length",
			config: `
reporting:
  root_cause_truncation_length: 0
`,
			errContains: "root_cause_truncation_length must be >= 1",
		},
		{
			name: "tracked < display",
			config: `
reporting:
  failure_reasons_display_count: 5
  max_failure_reasons_tracked: 3
`,
			errContains: "must be >= failure_reasons_display_count",
		},
		{
			name: "zero channel buffer",
			config: `
events:
  channel_buffer_size: 0
`,
			errContains: "channel_buffer_size must be >= 1",
		},
		{
			name: "zero stdout buffer",
			config: `
io:
  stdout_buffer_size: 0
`,
			errContains: "stdout_buffer_size must be >= 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tuningPath := filepath.Join(tmpDir, "tuning.yaml")
			if err := os.WriteFile(tuningPath, []byte(tt.config), 0644); err != nil {
				t.Fatalf("failed to write tuning file: %v", err)
			}

			_, err := LoadTuningWithFile(tuningPath)
			if err == nil {
				t.Error("LoadTuningWithFile() should fail validation")
			}
			if !contains(err.Error(), tt.errContains) {
				t.Errorf("error should contain %q, got: %v", tt.errContains, err)
			}
		})
	}
}

func TestDefaultTuning(t *testing.T) {
	defaults := defaultTuning()

	// Verify all default values are set correctly
	if defaults.HTTP.SlackTimeoutSeconds != 10 {
		t.Errorf("HTTP.SlackTimeoutSeconds = %d, want 10", defaults.HTTP.SlackTimeoutSeconds)
	}
	if defaults.Agent.TimeoutBufferSeconds != 60 {
		t.Errorf("Agent.TimeoutBufferSeconds = %d, want 60", defaults.Agent.TimeoutBufferSeconds)
	}
	if defaults.Agent.InvestigationMinSizeBytes != 100 {
		t.Errorf("Agent.InvestigationMinSizeBytes = %d, want 100", defaults.Agent.InvestigationMinSizeBytes)
	}
	if defaults.Reporting.RootCauseTruncationLength != 300 {
		t.Errorf("Reporting.RootCauseTruncationLength = %d, want 300", defaults.Reporting.RootCauseTruncationLength)
	}
	if defaults.Reporting.FailureReasonsDisplayCount != 3 {
		t.Errorf("Reporting.FailureReasonsDisplayCount = %d, want 3", defaults.Reporting.FailureReasonsDisplayCount)
	}
	if defaults.Reporting.MaxFailureReasonsTracked != 5 {
		t.Errorf("Reporting.MaxFailureReasonsTracked = %d, want 5", defaults.Reporting.MaxFailureReasonsTracked)
	}
	if defaults.Events.ChannelBufferSize != 100 {
		t.Errorf("Events.ChannelBufferSize = %d, want 100", defaults.Events.ChannelBufferSize)
	}
	if defaults.IO.StdoutBufferSize != 1024 {
		t.Errorf("IO.StdoutBufferSize = %d, want 1024", defaults.IO.StdoutBufferSize)
	}
	if defaults.IO.StderrBufferSize != 1024 {
		t.Errorf("IO.StderrBufferSize = %d, want 1024", defaults.IO.StderrBufferSize)
	}

	// Verify defaults pass validation
	if err := defaults.Validate(); err != nil {
		t.Errorf("defaultTuning() should produce valid config, got error: %v", err)
	}
}

func TestLoadTuning_AllCategories(t *testing.T) {
	// Create a comprehensive tuning file covering all categories
	tmpDir := t.TempDir()
	tuningPath := filepath.Join(tmpDir, "tuning.yaml")
	tuningContent := `
http:
  slack_timeout_seconds: 30

agent:
  timeout_buffer_seconds: 90
  investigation_min_size_bytes: 250

reporting:
  root_cause_truncation_length: 600
  failure_reasons_display_count: 7
  max_failure_reasons_tracked: 15

events:
  channel_buffer_size: 500

io:
  stdout_buffer_size: 4096
  stderr_buffer_size: 4096
`
	if err := os.WriteFile(tuningPath, []byte(tuningContent), 0644); err != nil {
		t.Fatalf("failed to write tuning file: %v", err)
	}

	tuning, err := LoadTuningWithFile(tuningPath)
	if err != nil {
		t.Fatalf("LoadTuningWithFile() failed: %v", err)
	}

	// Verify all categories are loaded correctly
	if tuning.HTTP.SlackTimeoutSeconds != 30 {
		t.Errorf("HTTP.SlackTimeoutSeconds = %d, want 30", tuning.HTTP.SlackTimeoutSeconds)
	}
	if tuning.Agent.TimeoutBufferSeconds != 90 {
		t.Errorf("Agent.TimeoutBufferSeconds = %d, want 90", tuning.Agent.TimeoutBufferSeconds)
	}
	if tuning.Agent.InvestigationMinSizeBytes != 250 {
		t.Errorf("Agent.InvestigationMinSizeBytes = %d, want 250", tuning.Agent.InvestigationMinSizeBytes)
	}
	if tuning.Reporting.RootCauseTruncationLength != 600 {
		t.Errorf("Reporting.RootCauseTruncationLength = %d, want 600", tuning.Reporting.RootCauseTruncationLength)
	}
	if tuning.Reporting.FailureReasonsDisplayCount != 7 {
		t.Errorf("Reporting.FailureReasonsDisplayCount = %d, want 7", tuning.Reporting.FailureReasonsDisplayCount)
	}
	if tuning.Reporting.MaxFailureReasonsTracked != 15 {
		t.Errorf("Reporting.MaxFailureReasonsTracked = %d, want 15", tuning.Reporting.MaxFailureReasonsTracked)
	}
	if tuning.Events.ChannelBufferSize != 500 {
		t.Errorf("Events.ChannelBufferSize = %d, want 500", tuning.Events.ChannelBufferSize)
	}
	if tuning.IO.StdoutBufferSize != 4096 {
		t.Errorf("IO.StdoutBufferSize = %d, want 4096", tuning.IO.StdoutBufferSize)
	}
	if tuning.IO.StderrBufferSize != 4096 {
		t.Errorf("IO.StderrBufferSize = %d, want 4096", tuning.IO.StderrBufferSize)
	}
}
