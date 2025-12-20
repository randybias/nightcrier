package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// TuningConfig holds tunable operational parameters that control system behavior.
// These parameters can be adjusted without changing core application configuration.
type TuningConfig struct {
	HTTP     HTTPTuning     `mapstructure:"http"`
	Agent    AgentTuning    `mapstructure:"agent"`
	Reporting ReportingTuning `mapstructure:"reporting"`
	Events   EventsTuning   `mapstructure:"events"`
	IO       IOTuning       `mapstructure:"io"`
}

// HTTPTuning contains HTTP client tuning parameters.
type HTTPTuning struct {
	// SlackTimeoutSeconds is the timeout for Slack webhook HTTP requests.
	SlackTimeoutSeconds int `mapstructure:"slack_timeout_seconds"`
}

// AgentTuning contains agent runtime tuning parameters.
type AgentTuning struct {
	// TimeoutBufferSeconds is additional buffer time beyond the configured agent timeout
	// to allow for graceful shutdown and cleanup.
	TimeoutBufferSeconds int `mapstructure:"timeout_buffer_seconds"`

	// InvestigationMinSizeBytes is the minimum size threshold for investigation output.
	// Investigations smaller than this are considered potentially incomplete.
	InvestigationMinSizeBytes int `mapstructure:"investigation_min_size_bytes"`
}

// ReportingTuning contains reporting and notification tuning parameters.
type ReportingTuning struct {
	// RootCauseTruncationLength is the maximum length of root cause text in Slack notifications.
	RootCauseTruncationLength int `mapstructure:"root_cause_truncation_length"`

	// FailureReasonsDisplayCount is the number of failure reasons to display in reports.
	FailureReasonsDisplayCount int `mapstructure:"failure_reasons_display_count"`

	// MaxFailureReasonsTracked is the maximum number of failure reasons to track internally.
	MaxFailureReasonsTracked int `mapstructure:"max_failure_reasons_tracked"`
}

// EventsTuning contains event processing tuning parameters.
type EventsTuning struct {
	// ChannelBufferSize is the buffer size for event processing channels.
	ChannelBufferSize int `mapstructure:"channel_buffer_size"`
}

// IOTuning contains I/O tuning parameters for agent output capture.
type IOTuning struct {
	// StdoutBufferSize is the buffer size for capturing agent stdout.
	StdoutBufferSize int `mapstructure:"stdout_buffer_size"`

	// StderrBufferSize is the buffer size for capturing agent stderr.
	StderrBufferSize int `mapstructure:"stderr_buffer_size"`
}

// defaultTuning returns a TuningConfig with sensible defaults.
// These defaults are used when tuning.yaml is not found or values are missing.
func defaultTuning() *TuningConfig {
	return &TuningConfig{
		HTTP: HTTPTuning{
			SlackTimeoutSeconds: 10,
		},
		Agent: AgentTuning{
			TimeoutBufferSeconds:      60,
			InvestigationMinSizeBytes: 100,
		},
		Reporting: ReportingTuning{
			RootCauseTruncationLength:  300,
			FailureReasonsDisplayCount: 3,
			MaxFailureReasonsTracked:   5,
		},
		Events: EventsTuning{
			ChannelBufferSize: 100,
		},
		IO: IOTuning{
			StdoutBufferSize: 1024,
			StderrBufferSize: 1024,
		},
	}
}

// setTuningDefaults configures default values for tuning parameters in viper.
func setTuningDefaults() {
	defaults := defaultTuning()

	// HTTP defaults
	viper.SetDefault("http.slack_timeout_seconds", defaults.HTTP.SlackTimeoutSeconds)

	// Agent defaults
	viper.SetDefault("agent.timeout_buffer_seconds", defaults.Agent.TimeoutBufferSeconds)
	viper.SetDefault("agent.investigation_min_size_bytes", defaults.Agent.InvestigationMinSizeBytes)

	// Reporting defaults
	viper.SetDefault("reporting.root_cause_truncation_length", defaults.Reporting.RootCauseTruncationLength)
	viper.SetDefault("reporting.failure_reasons_display_count", defaults.Reporting.FailureReasonsDisplayCount)
	viper.SetDefault("reporting.max_failure_reasons_tracked", defaults.Reporting.MaxFailureReasonsTracked)

	// Events defaults
	viper.SetDefault("events.channel_buffer_size", defaults.Events.ChannelBufferSize)

	// IO defaults
	viper.SetDefault("io.stdout_buffer_size", defaults.IO.StdoutBufferSize)
	viper.SetDefault("io.stderr_buffer_size", defaults.IO.StderrBufferSize)
}

// LoadTuning loads tuning configuration from configs/tuning.yaml.
// If the file is not found, it returns a TuningConfig with default values.
// This function creates a separate viper instance to avoid interfering with
// the main application configuration.
func LoadTuning() (*TuningConfig, error) {
	return LoadTuningWithFile("")
}

// LoadTuningWithFile loads tuning configuration from a specific file path.
// If tuningFile is empty, it searches for tuning.yaml in standard locations.
// If the file is not found, it returns a TuningConfig with default values.
func LoadTuningWithFile(tuningFile string) (*TuningConfig, error) {
	// Create a separate viper instance for tuning config
	v := viper.New()

	// Set defaults first
	defaults := defaultTuning()
	v.SetDefault("http.slack_timeout_seconds", defaults.HTTP.SlackTimeoutSeconds)
	v.SetDefault("agent.timeout_buffer_seconds", defaults.Agent.TimeoutBufferSeconds)
	v.SetDefault("agent.investigation_min_size_bytes", defaults.Agent.InvestigationMinSizeBytes)
	v.SetDefault("reporting.root_cause_truncation_length", defaults.Reporting.RootCauseTruncationLength)
	v.SetDefault("reporting.failure_reasons_display_count", defaults.Reporting.FailureReasonsDisplayCount)
	v.SetDefault("reporting.max_failure_reasons_tracked", defaults.Reporting.MaxFailureReasonsTracked)
	v.SetDefault("events.channel_buffer_size", defaults.Events.ChannelBufferSize)
	v.SetDefault("io.stdout_buffer_size", defaults.IO.StdoutBufferSize)
	v.SetDefault("io.stderr_buffer_size", defaults.IO.StderrBufferSize)

	// Configure file location
	if tuningFile != "" {
		v.SetConfigFile(tuningFile)
	} else {
		// Search for tuning.yaml in standard locations
		v.SetConfigName("tuning")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")               // Current directory
		v.AddConfigPath("./configs")       // configs subdirectory
		v.AddConfigPath("/etc/nightcrier") // System-wide config
	}

	// Try to read the config file
	if err := v.ReadInConfig(); err != nil {
		// If file not found, return defaults without error
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return defaults, nil
		}
		// Check if it's an os.PathError (file doesn't exist)
		if _, ok := err.(*os.PathError); ok {
			return defaults, nil
		}
		// For other errors (e.g., parse errors), return the error
		return nil, fmt.Errorf("failed to read tuning config: %w", err)
	}

	// Unmarshal into TuningConfig struct
	var tuning TuningConfig
	if err := v.Unmarshal(&tuning); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tuning config: %w", err)
	}

	// Validate tuning parameters
	if err := tuning.Validate(); err != nil {
		return nil, err
	}

	return &tuning, nil
}

// Validate checks tuning parameters for valid ranges.
func (t *TuningConfig) Validate() error {
	// HTTP validations
	if t.HTTP.SlackTimeoutSeconds < 1 {
		return fmt.Errorf("http.slack_timeout_seconds must be >= 1, got %d", t.HTTP.SlackTimeoutSeconds)
	}

	// Agent validations
	if t.Agent.TimeoutBufferSeconds < 0 {
		return fmt.Errorf("agent.timeout_buffer_seconds must be >= 0, got %d", t.Agent.TimeoutBufferSeconds)
	}
	if t.Agent.InvestigationMinSizeBytes < 0 {
		return fmt.Errorf("agent.investigation_min_size_bytes must be >= 0, got %d", t.Agent.InvestigationMinSizeBytes)
	}

	// Reporting validations
	if t.Reporting.RootCauseTruncationLength < 1 {
		return fmt.Errorf("reporting.root_cause_truncation_length must be >= 1, got %d", t.Reporting.RootCauseTruncationLength)
	}
	if t.Reporting.FailureReasonsDisplayCount < 1 {
		return fmt.Errorf("reporting.failure_reasons_display_count must be >= 1, got %d", t.Reporting.FailureReasonsDisplayCount)
	}
	if t.Reporting.MaxFailureReasonsTracked < t.Reporting.FailureReasonsDisplayCount {
		return fmt.Errorf("reporting.max_failure_reasons_tracked (%d) must be >= failure_reasons_display_count (%d)",
			t.Reporting.MaxFailureReasonsTracked, t.Reporting.FailureReasonsDisplayCount)
	}

	// Events validations
	if t.Events.ChannelBufferSize < 1 {
		return fmt.Errorf("events.channel_buffer_size must be >= 1, got %d", t.Events.ChannelBufferSize)
	}

	// IO validations
	if t.IO.StdoutBufferSize < 1 {
		return fmt.Errorf("io.stdout_buffer_size must be >= 1, got %d", t.IO.StdoutBufferSize)
	}
	if t.IO.StderrBufferSize < 1 {
		return fmt.Errorf("io.stderr_buffer_size must be >= 1, got %d", t.IO.StderrBufferSize)
	}

	return nil
}

// GetTuningFile returns the tuning config file that was used, if any.
// This is useful for debugging configuration issues.
func GetTuningFile(v *viper.Viper) string {
	if v != nil {
		return v.ConfigFileUsed()
	}
	return ""
}
