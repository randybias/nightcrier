package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// resetViper clears viper state between tests
func resetViper() {
	viper.Reset()
}

// setTestAPIKey sets a dummy API key for tests that need to pass LLM validation
func setTestAPIKey(t *testing.T) func() {
	os.Setenv("ANTHROPIC_API_KEY", "test-key-for-unit-tests")
	return func() {
		os.Unsetenv("ANTHROPIC_API_KEY")
	}
}

// testConfigWithAPIKey returns config content with a test API key included
func testConfigWithAPIKey(baseConfig string) string {
	return baseConfig + "\nanthropic_api_key: \"test-key-for-unit-tests\"\n"
}

// completeTestConfig returns a complete config with all required fields for testing
func completeTestConfig() string {
	return `
clusters:
  - name: test-cluster
    mcp:
      endpoint: "http://localhost:8080/mcp"
subscribe_mode: "faults"
workspace_root: "./incidents"
agent_script_path: "./agent-container/run-agent.sh"
agent_timeout: 300
agent_model: "sonnet"
agent_cli: "claude"
agent_image: "nightcrier-agent:latest"
severity_threshold: "ERROR"
max_concurrent_agents: 5
global_queue_size: 100
cluster_queue_size: 10
dedup_window_seconds: 300
queue_overflow_policy: "drop"
shutdown_timeout: 30
sse_reconnect_initial_backoff: 1
sse_reconnect_max_backoff: 60
sse_read_timeout: 120
failure_threshold_for_alert: 3
anthropic_api_key: "test-key"
`
}

// completeTestConfigWith overrides specific fields in completeTestConfig
func completeTestConfigWith(overrides string) string {
	return completeTestConfig() + overrides
}

// completeTestConfigWithoutAPIKey returns a complete config without any API key for validation testing
func completeTestConfigWithoutAPIKey() string {
	return `
clusters:
  - name: test-cluster
    mcp:
      endpoint: "http://localhost:8080/mcp"
subscribe_mode: "faults"
workspace_root: "./incidents"
agent_script_path: "./agent-container/run-agent.sh"
agent_timeout: 300
agent_model: "sonnet"
agent_cli: "claude"
agent_image: "nightcrier-agent:latest"
severity_threshold: "ERROR"
max_concurrent_agents: 5
global_queue_size: 100
cluster_queue_size: 10
dedup_window_seconds: 300
queue_overflow_policy: "drop"
shutdown_timeout: 30
sse_reconnect_initial_backoff: 1
sse_reconnect_max_backoff: 60
sse_read_timeout: 120
failure_threshold_for_alert: 3
`
}

// buildTestConfig creates a complete config with specific field values
func buildTestConfig(overrides map[string]interface{}) string {
	// Default values
	values := map[string]interface{}{
		"subscribe_mode":                  "faults",
		"workspace_root":                  "./incidents",
		"agent_script_path":               "./agent-container/run-agent.sh",
		"agent_timeout":                   300,
		"agent_model":                     "sonnet",
		"agent_cli":                       "claude",
		"agent_image":                     "nightcrier-agent:latest",
		"severity_threshold":              "ERROR",
		"max_concurrent_agents":           5,
		"global_queue_size":               100,
		"cluster_queue_size":              10,
		"dedup_window_seconds":            300,
		"queue_overflow_policy":           "drop",
		"shutdown_timeout":                30,
		"sse_reconnect_initial_backoff":   1,
		"sse_reconnect_max_backoff":       60,
		"sse_read_timeout":                120,
		"failure_threshold_for_alert":     3,
		"anthropic_api_key":               "test-key",
	}

	// Apply overrides
	for k, v := range overrides {
		if v == nil {
			delete(values, k)
		} else {
			values[k] = v
		}
	}

	// Build YAML string - start with clusters section
	config := `
clusters:
  - name: test-cluster
    mcp:
      endpoint: "http://localhost:8080/mcp"
`
	for k, v := range values {
		switch val := v.(type) {
		case string:
			config += fmt.Sprintf("%s: \"%s\"\n", k, val)
		case int:
			config += fmt.Sprintf("%s: %d\n", k, val)
		case bool:
			config += fmt.Sprintf("%s: %v\n", k, val)
		}
	}
	return config
}

func TestLoadWithAllRequiredFields(t *testing.T) {
	resetViper()

	// Create config file with all required fields including clusters
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := completeTestConfig()
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadWithConfigFile(configPath)
	if err != nil {
		t.Fatalf("LoadWithConfigFile() failed: %v", err)
	}

	// Check all values are set correctly
	if cfg.WorkspaceRoot != "./incidents" {
		t.Errorf("WorkspaceRoot = %q, want %q", cfg.WorkspaceRoot, "./incidents")
	}
	if cfg.AgentModel != "sonnet" {
		t.Errorf("AgentModel = %q, want %q", cfg.AgentModel, "sonnet")
	}
	if cfg.AgentTimeout != 300 {
		t.Errorf("AgentTimeout = %d, want %d", cfg.AgentTimeout, 300)
	}
	if cfg.SeverityThreshold != "ERROR" {
		t.Errorf("SeverityThreshold = %q, want %q", cfg.SeverityThreshold, "ERROR")
	}
	if cfg.MaxConcurrentAgents != 5 {
		t.Errorf("MaxConcurrentAgents = %d, want %d", cfg.MaxConcurrentAgents, 5)
	}
	if cfg.GlobalQueueSize != 100 {
		t.Errorf("GlobalQueueSize = %d, want %d", cfg.GlobalQueueSize, 100)
	}
	if cfg.ClusterQueueSize != 10 {
		t.Errorf("ClusterQueueSize = %d, want %d", cfg.ClusterQueueSize, 10)
	}
	if cfg.DedupWindowSeconds != 300 {
		t.Errorf("DedupWindowSeconds = %d, want %d", cfg.DedupWindowSeconds, 300)
	}
	if cfg.QueueOverflowPolicy != "drop" {
		t.Errorf("QueueOverflowPolicy = %q, want %q", cfg.QueueOverflowPolicy, "drop")
	}
}

func TestLoadFromEnvVars(t *testing.T) {
	resetViper()

	// Create a minimal config file with clusters (required in config file)
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := completeTestConfig() // Provides clusters and defaults
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Set env vars to override config file values
	os.Setenv("WORKSPACE_ROOT", "/var/incidents")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("AGENT_MODEL", "opus")
	os.Setenv("AGENT_TIMEOUT", "600")
	os.Setenv("SEVERITY_THRESHOLD", "WARNING")
	os.Setenv("MAX_CONCURRENT_AGENTS", "10")

	defer func() {
		os.Unsetenv("WORKSPACE_ROOT")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("AGENT_MODEL")
		os.Unsetenv("AGENT_TIMEOUT")
		os.Unsetenv("SEVERITY_THRESHOLD")
		os.Unsetenv("MAX_CONCURRENT_AGENTS")
	}()

	cfg, err := LoadWithConfigFile(configPath)
	if err != nil {
		t.Fatalf("LoadWithConfigFile() failed: %v", err)
	}

	// Env vars should override config file values
	if cfg.WorkspaceRoot != "/var/incidents" {
		t.Errorf("WorkspaceRoot = %q, want %q", cfg.WorkspaceRoot, "/var/incidents")
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
	if cfg.AgentModel != "opus" {
		t.Errorf("AgentModel = %q, want %q", cfg.AgentModel, "opus")
	}
	if cfg.AgentTimeout != 600 {
		t.Errorf("AgentTimeout = %d, want %d", cfg.AgentTimeout, 600)
	}
	if cfg.SeverityThreshold != "WARNING" {
		t.Errorf("SeverityThreshold = %q, want %q", cfg.SeverityThreshold, "WARNING")
	}
	if cfg.MaxConcurrentAgents != 10 {
		t.Errorf("MaxConcurrentAgents = %d, want %d", cfg.MaxConcurrentAgents, 10)
	}
}

func TestLoadFromConfigFile(t *testing.T) {
	resetViper()

	// Create temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
clusters:
  - name: config-file-cluster
    mcp:
      endpoint: "http://config-file-server:8080/mcp"
subscribe_mode: "faults"
workspace_root: "/config/incidents"
log_level: "warn"
agent_script_path: "./agent-container/run-agent.sh"
agent_model: "haiku"
agent_timeout: 120
agent_cli: "claude"
agent_image: "nightcrier-agent:latest"
severity_threshold: "CRITICAL"
max_concurrent_agents: 3
global_queue_size: 50
cluster_queue_size: 5
dedup_window_seconds: 600
queue_overflow_policy: "reject"
shutdown_timeout: 30
sse_reconnect_initial_backoff: 1
sse_reconnect_max_backoff: 60
sse_read_timeout: 120
failure_threshold_for_alert: 3
anthropic_api_key: "test-key"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadWithConfigFile(configPath)
	if err != nil {
		t.Fatalf("LoadWithConfigFile() failed: %v", err)
	}

	if len(cfg.Clusters) != 1 || cfg.Clusters[0].MCP.Endpoint != "http://config-file-server:8080/mcp" {
		t.Errorf("Clusters[0].MCP.Endpoint = %q, want %q", cfg.Clusters[0].MCP.Endpoint, "http://config-file-server:8080/mcp")
	}
	if cfg.WorkspaceRoot != "/config/incidents" {
		t.Errorf("WorkspaceRoot = %q, want %q", cfg.WorkspaceRoot, "/config/incidents")
	}
	if cfg.LogLevel != "warn" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "warn")
	}
	if cfg.AgentModel != "haiku" {
		t.Errorf("AgentModel = %q, want %q", cfg.AgentModel, "haiku")
	}
	if cfg.AgentTimeout != 120 {
		t.Errorf("AgentTimeout = %d, want %d", cfg.AgentTimeout, 120)
	}
	if cfg.SeverityThreshold != "CRITICAL" {
		t.Errorf("SeverityThreshold = %q, want %q", cfg.SeverityThreshold, "CRITICAL")
	}
	if cfg.MaxConcurrentAgents != 3 {
		t.Errorf("MaxConcurrentAgents = %d, want %d", cfg.MaxConcurrentAgents, 3)
	}
	if cfg.GlobalQueueSize != 50 {
		t.Errorf("GlobalQueueSize = %d, want %d", cfg.GlobalQueueSize, 50)
	}
	if cfg.ClusterQueueSize != 5 {
		t.Errorf("ClusterQueueSize = %d, want %d", cfg.ClusterQueueSize, 5)
	}
	if cfg.DedupWindowSeconds != 600 {
		t.Errorf("DedupWindowSeconds = %d, want %d", cfg.DedupWindowSeconds, 600)
	}
	if cfg.QueueOverflowPolicy != "reject" {
		t.Errorf("QueueOverflowPolicy = %q, want %q", cfg.QueueOverflowPolicy, "reject")
	}
}

func TestEnvVarsOverrideConfigFile(t *testing.T) {
	resetViper()

	// Create temp config file with all required fields
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
clusters:
  - name: config-file-cluster
    mcp:
      endpoint: "http://config-file-server:8080/mcp"
subscribe_mode: "faults"
workspace_root: "/config/incidents"
log_level: "warn"
agent_script_path: "./agent-container/run-agent.sh"
agent_timeout: 120
agent_model: "sonnet"
agent_cli: "claude"
agent_image: "nightcrier-agent:latest"
severity_threshold: "ERROR"
max_concurrent_agents: 5
global_queue_size: 100
cluster_queue_size: 10
dedup_window_seconds: 300
queue_overflow_policy: "drop"
shutdown_timeout: 30
sse_reconnect_initial_backoff: 1
sse_reconnect_max_backoff: 60
sse_read_timeout: 120
failure_threshold_for_alert: 3
anthropic_api_key: "test-key"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Set env vars that should override config file
	// Note: Cluster MCP endpoint is no longer overridable via env var (multi-cluster config)
	os.Setenv("LOG_LEVEL", "error")
	defer func() {
		os.Unsetenv("LOG_LEVEL")
	}()

	cfg, err := LoadWithConfigFile(configPath)
	if err != nil {
		t.Fatalf("LoadWithConfigFile() failed: %v", err)
	}

	// Env vars should override config file for supported fields
	if cfg.LogLevel != "error" {
		t.Errorf("LogLevel = %q, want %q (env var should override)", cfg.LogLevel, "error")
	}

	// Config file values should still apply where not overridden
	if cfg.WorkspaceRoot != "/config/incidents" {
		t.Errorf("WorkspaceRoot = %q, want %q (from config file)", cfg.WorkspaceRoot, "/config/incidents")
	}
	if cfg.AgentTimeout != 120 {
		t.Errorf("AgentTimeout = %d, want %d (from config file)", cfg.AgentTimeout, 120)
	}
}

func TestValidation_MissingClusters(t *testing.T) {
	resetViper()

	// Config without clusters should fail
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
subscribe_mode: "faults"
workspace_root: "./incidents"
agent_script_path: "./agent-container/run-agent.sh"
agent_timeout: 300
agent_model: "sonnet"
agent_cli: "claude"
agent_image: "nightcrier-agent:latest"
severity_threshold: "ERROR"
max_concurrent_agents: 5
global_queue_size: 100
cluster_queue_size: 10
dedup_window_seconds: 300
queue_overflow_policy: "drop"
shutdown_timeout: 30
sse_reconnect_initial_backoff: 1
sse_reconnect_max_backoff: 60
sse_read_timeout: 120
failure_threshold_for_alert: 3
anthropic_api_key: "test-key"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := LoadWithConfigFile(configPath)
	if err == nil {
		t.Error("LoadWithConfigFile() should fail when clusters is missing")
	}
}

func TestValidation_InvalidSeverityThreshold(t *testing.T) {
	resetViper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
clusters:
  - name: test-cluster
    mcp:
      endpoint: "http://localhost:8080/mcp"
severity_threshold: "INVALID"
anthropic_api_key: "test-key"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := LoadWithConfigFile(configPath)
	if err == nil {
		t.Error("LoadWithConfigFile() should fail with invalid severity threshold")
	}
}

func TestValidation_InvalidNumericRanges(t *testing.T) {
	resetViper()

	clusterPrefix := `
clusters:
  - name: test-cluster
    mcp:
      endpoint: "http://localhost:8080/mcp"
`

	tests := []struct {
		name    string
		config  string
		wantErr bool
	}{
		{
			name:    "max_concurrent_agents < 1",
			config:  clusterPrefix + "max_concurrent_agents: 0\nanthropic_api_key: \"test-key\"\n",
			wantErr: true,
		},
		{
			name:    "global_queue_size < 1",
			config:  clusterPrefix + "global_queue_size: 0\nanthropic_api_key: \"test-key\"\n",
			wantErr: true,
		},
		{
			name:    "cluster_queue_size < 1",
			config:  clusterPrefix + "cluster_queue_size: 0\nanthropic_api_key: \"test-key\"\n",
			wantErr: true,
		},
		{
			name:    "dedup_window_seconds negative",
			config:  clusterPrefix + "dedup_window_seconds: -1\nanthropic_api_key: \"test-key\"\n",
			wantErr: true,
		},
		{
			name:    "agent_timeout < 1",
			config:  clusterPrefix + "agent_timeout: 0\nanthropic_api_key: \"test-key\"\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper()

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			if err := os.WriteFile(configPath, []byte(tt.config), 0644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			_, err := LoadWithConfigFile(configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadWithConfigFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidation_InvalidQueueOverflowPolicy(t *testing.T) {
	resetViper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
clusters:
  - name: test-cluster
    mcp:
      endpoint: "http://localhost:8080/mcp"
queue_overflow_policy: "invalid"
anthropic_api_key: "test-key"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := LoadWithConfigFile(configPath)
	if err == nil {
		t.Error("LoadWithConfigFile() should fail with invalid queue overflow policy")
	}
}

func TestValidation_SSEReconnectSettings(t *testing.T) {
	resetViper()

	clusterPrefix := `
clusters:
  - name: test-cluster
    mcp:
      endpoint: "http://localhost:8080/mcp"
`

	tests := []struct {
		name    string
		config  string
		wantErr bool
	}{
		{
			name:    "initial backoff < 1",
			config:  clusterPrefix + "sse_reconnect_initial_backoff: 0\nanthropic_api_key: \"test-key\"\n",
			wantErr: true,
		},
		{
			name:    "max backoff < initial backoff",
			config:  clusterPrefix + "sse_reconnect_initial_backoff: 10\nsse_reconnect_max_backoff: 5\nanthropic_api_key: \"test-key\"\n",
			wantErr: true,
		},
		{
			name:    "read timeout < 1",
			config:  clusterPrefix + "sse_read_timeout: 0\nanthropic_api_key: \"test-key\"\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper()

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			if err := os.WriteFile(configPath, []byte(tt.config), 0644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			_, err := LoadWithConfigFile(configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadWithConfigFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidation_ValidSeverityLevels(t *testing.T) {
	resetViper()

	validSeverities := []string{"DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL", "debug", "info", "warning", "error", "critical"}

	for _, severity := range validSeverities {
		t.Run(severity, func(t *testing.T) {
			resetViper()

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			configContent := `
clusters:
  - name: test-cluster
    mcp:
      endpoint: "http://localhost:8080/mcp"
subscribe_mode: "faults"
workspace_root: "./incidents"
agent_script_path: "./agent-container/run-agent.sh"
agent_timeout: 300
agent_model: "sonnet"
agent_cli: "claude"
agent_image: "nightcrier-agent:latest"
severity_threshold: "` + severity + `"
max_concurrent_agents: 5
global_queue_size: 100
cluster_queue_size: 10
dedup_window_seconds: 300
queue_overflow_policy: "drop"
shutdown_timeout: 30
sse_reconnect_initial_backoff: 1
sse_reconnect_max_backoff: 60
sse_read_timeout: 120
failure_threshold_for_alert: 3
anthropic_api_key: "test-key"
`
			if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			_, err := LoadWithConfigFile(configPath)
			if err != nil {
				t.Errorf("LoadWithConfigFile() failed for valid severity %q: %v", severity, err)
			}
		})
	}
}

func TestValidation_ValidQueueOverflowPolicies(t *testing.T) {
	resetViper()

	validPolicies := []string{"drop", "reject", "DROP", "REJECT"}

	for _, policy := range validPolicies {
		t.Run(policy, func(t *testing.T) {
			resetViper()

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			configContent := buildTestConfig(map[string]interface{}{
				"queue_overflow_policy": policy,
			})
			if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			_, err := LoadWithConfigFile(configPath)
			if err != nil {
				t.Errorf("LoadWithConfigFile() failed for valid policy %q: %v", policy, err)
			}
		})
	}
}

func TestConfigFileNotFound(t *testing.T) {
	resetViper()

	// With multi-cluster config, clusters must be defined in a config file.
	// Load() without a config file should fail when no clusters are defined.
	_, err := Load()
	if err == nil {
		t.Error("Load() should fail when no config file exists and clusters are not defined")
	}
}

func TestInvalidConfigFilePath(t *testing.T) {
	resetViper()

	_, err := LoadWithConfigFile("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("LoadWithConfigFile() should fail with invalid config file path")
	}
}

func TestAzureStorageEnabled(t *testing.T) {
	resetViper()

	tests := []struct {
		name    string
		config  string
		enabled bool
	}{
		{
			name:    "disabled when no Azure config",
			config:  completeTestConfig(),
			enabled: false,
		},
		{
			name: "enabled with connection string",
			config: completeTestConfigWith(`
azure_storage_connection_string: "AccountName=test;AccountKey=key123"
azure_storage_container: "incidents"
`),
			enabled: true,
		},
		{
			name: "enabled with account and key",
			config: completeTestConfigWith(`
azure_storage_account: "teststorage"
azure_storage_key: "key123"
azure_storage_container: "incidents"
`),
			enabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper()

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			if err := os.WriteFile(configPath, []byte(tt.config), 0644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			cfg, err := LoadWithConfigFile(configPath)
			if err != nil {
				t.Fatalf("LoadWithConfigFile() failed: %v", err)
			}

			if cfg.IsAzureStorageEnabled() != tt.enabled {
				t.Errorf("IsAzureStorageEnabled() = %v, want %v", cfg.IsAzureStorageEnabled(), tt.enabled)
			}
		})
	}
}

func TestGetAzureSASExpiry(t *testing.T) {
	resetViper()

	tests := []struct {
		name    string
		expiry  string
		wantHrs int
	}{
		{"default", "", 168}, // 7 days
		{"24 hours", "24h", 24},
		{"48 hours", "48h", 48},
		{"invalid", "invalid", 168}, // Falls back to default
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{AzureSASExpiry: tt.expiry}
			got := cfg.GetAzureSASExpiry()
			wantDur := float64(tt.wantHrs)
			gotHrs := got.Hours()
			if gotHrs != wantDur {
				t.Errorf("GetAzureSASExpiry() = %v hours, want %v hours", gotHrs, wantDur)
			}
		})
	}
}

func TestValidation_RequiresLLMAPIKey(t *testing.T) {
	resetViper()

	// Config without any API key should fail
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := completeTestConfigWithoutAPIKey()
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := LoadWithConfigFile(configPath)
	if err == nil {
		t.Error("LoadWithConfigFile() should fail when no LLM API key is configured")
	}

	// Verify error message is helpful
	expectedMsg := "at least one LLM API key is required"
	if err != nil && !contains(err.Error(), expectedMsg) {
		t.Errorf("error message should contain %q, got: %v", expectedMsg, err)
	}
}

func TestValidation_AcceptsAnyLLMAPIKey(t *testing.T) {
	resetViper()

	tests := []struct {
		name   string
		config string
	}{
		{
			name:   "anthropic key",
			config: buildTestConfig(map[string]interface{}{"anthropic_api_key": "sk-ant-test"}),
		},
		{
			name:   "openai key",
			config: buildTestConfig(map[string]interface{}{"anthropic_api_key": nil, "openai_api_key": "sk-test"}),
		},
		{
			name:   "gemini key",
			config: buildTestConfig(map[string]interface{}{"anthropic_api_key": nil, "gemini_api_key": "test-key"}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper()

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			if err := os.WriteFile(configPath, []byte(tt.config), 0644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			_, err := LoadWithConfigFile(configPath)
			if err != nil {
				t.Errorf("LoadWithConfigFile() should succeed with %s: %v", tt.name, err)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestCircuitBreakerConfig tests circuit breaker configuration options
func TestCircuitBreakerConfig(t *testing.T) {
	resetViper()

	clusterPrefix := `
clusters:
  - name: test-cluster
    mcp:
      endpoint: "http://localhost:8080/mcp"
`

	// Base config without circuit breaker settings (uses defaults)
	baseConfigNoCircuitBreaker := clusterPrefix + `subscribe_mode: "faults"
workspace_root: "./incidents"
agent_script_path: "./run-agent.sh"
agent_timeout: 300
agent_model: "sonnet"
agent_cli: "claude"
agent_image: "nightcrier-agent:latest"
severity_threshold: "ERROR"
max_concurrent_agents: 5
global_queue_size: 100
cluster_queue_size: 10
dedup_window_seconds: 300
queue_overflow_policy: "drop"
shutdown_timeout: 30
sse_reconnect_initial_backoff: 1
sse_reconnect_max_backoff: 60
sse_read_timeout: 120
failure_threshold_for_alert: 3
anthropic_api_key: "test-key"
`

	// Custom config with custom circuit breaker settings
	customConfig := clusterPrefix + `subscribe_mode: "faults"
workspace_root: "./incidents"
agent_script_path: "./run-agent.sh"
agent_timeout: 300
agent_model: "sonnet"
agent_cli: "claude"
agent_image: "nightcrier-agent:latest"
severity_threshold: "ERROR"
max_concurrent_agents: 5
global_queue_size: 100
cluster_queue_size: 10
dedup_window_seconds: 300
queue_overflow_policy: "drop"
shutdown_timeout: 30
sse_reconnect_initial_backoff: 1
sse_reconnect_max_backoff: 60
sse_read_timeout: 120
anthropic_api_key: "test-key"
notify_on_agent_failure: false
failure_threshold_for_alert: 5
upload_failed_investigations: true
`

	tests := []struct {
		name    string
		config  string
		wantCfg func(*Config) bool
	}{
		{
			name:   "uses optional defaults when not specified",
			config: baseConfigNoCircuitBreaker,
			wantCfg: func(cfg *Config) bool {
				return cfg.NotifyOnAgentFailure == false &&
					cfg.FailureThresholdForAlert == 3 &&
					cfg.UploadFailedInvestigations == false
			},
		},
		{
			name:   "custom values",
			config: customConfig,
			wantCfg: func(cfg *Config) bool {
				return cfg.NotifyOnAgentFailure == false &&
					cfg.FailureThresholdForAlert == 5 &&
					cfg.UploadFailedInvestigations == true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper()

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			if err := os.WriteFile(configPath, []byte(tt.config), 0644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			cfg, err := LoadWithConfigFile(configPath)
			if err != nil {
				t.Fatalf("LoadWithConfigFile() failed: %v", err)
			}

			if !tt.wantCfg(cfg) {
				t.Errorf("config values mismatch: NotifyOnAgentFailure=%v, FailureThresholdForAlert=%d, UploadFailedInvestigations=%v",
					cfg.NotifyOnAgentFailure, cfg.FailureThresholdForAlert, cfg.UploadFailedInvestigations)
			}
		})
	}
}

// TestCircuitBreakerConfigFromEnv tests circuit breaker configuration from environment variables
func TestCircuitBreakerConfigFromEnv(t *testing.T) {
	resetViper()

	// Create a config file with clusters (required) and default circuit breaker settings
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := completeTestConfig()
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Set env vars to override circuit breaker settings
	os.Setenv("NOTIFY_ON_AGENT_FAILURE", "false")
	os.Setenv("FAILURE_THRESHOLD_FOR_ALERT", "10")
	os.Setenv("UPLOAD_FAILED_INVESTIGATIONS", "true")

	defer func() {
		os.Unsetenv("NOTIFY_ON_AGENT_FAILURE")
		os.Unsetenv("FAILURE_THRESHOLD_FOR_ALERT")
		os.Unsetenv("UPLOAD_FAILED_INVESTIGATIONS")
	}()

	cfg, err := LoadWithConfigFile(configPath)
	if err != nil {
		t.Fatalf("LoadWithConfigFile() failed: %v", err)
	}

	if cfg.NotifyOnAgentFailure != false {
		t.Errorf("NotifyOnAgentFailure = %v, want false", cfg.NotifyOnAgentFailure)
	}
	if cfg.FailureThresholdForAlert != 10 {
		t.Errorf("FailureThresholdForAlert = %d, want 10", cfg.FailureThresholdForAlert)
	}
	if cfg.UploadFailedInvestigations != true {
		t.Errorf("UploadFailedInvestigations = %v, want true", cfg.UploadFailedInvestigations)
	}
}

// TestValidation_FailureThresholdRange tests failure threshold validation
func TestValidation_FailureThresholdRange(t *testing.T) {
	resetViper()

	tests := []struct {
		name      string
		threshold int
		wantErr   bool
	}{
		{"valid threshold 1", 1, false},
		{"valid threshold 3", 3, false},
		{"valid threshold 10", 10, false},
		{"invalid threshold 0", 0, true},
		{"invalid threshold -1", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper()

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			configContent := buildTestConfig(map[string]interface{}{
				"failure_threshold_for_alert": tt.threshold,
			})
			if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			_, err := LoadWithConfigFile(configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadWithConfigFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestCircuitBreakerConfig_IntegrationTest tests that circuit breaker config works with other config options
func TestCircuitBreakerConfig_IntegrationTest(t *testing.T) {
	resetViper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := buildTestConfig(map[string]interface{}{
		"workspace_root":                "/tmp/incidents",
		"log_level":                     "debug",
		"agent_timeout":                 600,
		"severity_threshold":            "WARNING",
		"max_concurrent_agents":         10,
		"notify_on_agent_failure":       false,
		"failure_threshold_for_alert":   5,
		"upload_failed_investigations":  true,
		"azure_storage_account":         "teststorage",
		"azure_storage_key":             "testkey",
		"azure_storage_container":       "incidents",
	})
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadWithConfigFile(configPath)
	if err != nil {
		t.Fatalf("LoadWithConfigFile() failed: %v", err)
	}

	// Verify circuit breaker settings
	if cfg.NotifyOnAgentFailure != false {
		t.Errorf("NotifyOnAgentFailure = %v, want false", cfg.NotifyOnAgentFailure)
	}
	if cfg.FailureThresholdForAlert != 5 {
		t.Errorf("FailureThresholdForAlert = %d, want 5", cfg.FailureThresholdForAlert)
	}
	if cfg.UploadFailedInvestigations != true {
		t.Errorf("UploadFailedInvestigations = %v, want true", cfg.UploadFailedInvestigations)
	}

	// Verify other settings still work
	if len(cfg.Clusters) != 1 || cfg.Clusters[0].MCP.Endpoint != "http://localhost:8080/mcp" {
		t.Errorf("Clusters[0].MCP.Endpoint = %q, want %q", cfg.Clusters[0].MCP.Endpoint, "http://localhost:8080/mcp")
	}
	if cfg.MaxConcurrentAgents != 10 {
		t.Errorf("MaxConcurrentAgents = %d, want 10", cfg.MaxConcurrentAgents)
	}
	if cfg.AzureStorageAccount != "teststorage" {
		t.Errorf("AzureStorageAccount = %q, want %q", cfg.AzureStorageAccount, "teststorage")
	}
}

// TestValidation_MissingRequiredFields tests that all required fields generate helpful error messages
func TestValidation_MissingRequiredFields(t *testing.T) {
	// Cluster config prefix for tests that need clusters defined
	clusterPrefix := `
clusters:
  - name: test-cluster
    mcp:
      endpoint: "http://localhost:8080/mcp"
`

	tests := []struct {
		name              string
		config            string
		expectedFieldName string
		expectedEnvVar    string
	}{
		{
			name:              "missing clusters",
			config:            `anthropic_api_key: "test-key"`,
			expectedFieldName: "clusters",
			expectedEnvVar:    "",
		},
		{
			name:              "missing subscribe_mode",
			config:            clusterPrefix + `anthropic_api_key: "test-key"`,
			expectedFieldName: "subscribe_mode",
			expectedEnvVar:    "SUBSCRIBE_MODE",
		},
		{
			name: "missing workspace_root",
			config: clusterPrefix + `subscribe_mode: "faults"
anthropic_api_key: "test-key"`,
			expectedFieldName: "workspace_root",
			expectedEnvVar:    "WORKSPACE_ROOT",
		},
		{
			name: "missing agent_script_path",
			config: clusterPrefix + `subscribe_mode: "faults"
workspace_root: "./incidents"
anthropic_api_key: "test-key"`,
			expectedFieldName: "agent_script_path",
			expectedEnvVar:    "AGENT_SCRIPT_PATH",
		},
		{
			name: "missing agent_timeout",
			config: clusterPrefix + `subscribe_mode: "faults"
workspace_root: "./incidents"
agent_script_path: "./run-agent.sh"
anthropic_api_key: "test-key"`,
			expectedFieldName: "agent_timeout",
			expectedEnvVar:    "AGENT_TIMEOUT",
		},
		{
			name: "missing agent_model",
			config: clusterPrefix + `subscribe_mode: "faults"
workspace_root: "./incidents"
agent_script_path: "./run-agent.sh"
agent_timeout: 300
anthropic_api_key: "test-key"`,
			expectedFieldName: "agent_model",
			expectedEnvVar:    "AGENT_MODEL",
		},
		{
			name: "missing agent_cli",
			config: clusterPrefix + `subscribe_mode: "faults"
workspace_root: "./incidents"
agent_script_path: "./run-agent.sh"
agent_timeout: 300
agent_model: "sonnet"
anthropic_api_key: "test-key"`,
			expectedFieldName: "agent_cli",
			expectedEnvVar:    "AGENT_CLI",
		},
		{
			name: "missing agent_image",
			config: clusterPrefix + `subscribe_mode: "faults"
workspace_root: "./incidents"
agent_script_path: "./run-agent.sh"
agent_timeout: 300
agent_model: "sonnet"
agent_cli: "claude"
anthropic_api_key: "test-key"`,
			expectedFieldName: "agent_image",
			expectedEnvVar:    "AGENT_IMAGE",
		},
		{
			name: "missing severity_threshold",
			config: clusterPrefix + `subscribe_mode: "faults"
workspace_root: "./incidents"
agent_script_path: "./run-agent.sh"
agent_timeout: 300
agent_model: "sonnet"
agent_cli: "claude"
agent_image: "nightcrier-agent:latest"
anthropic_api_key: "test-key"`,
			expectedFieldName: "severity_threshold",
			expectedEnvVar:    "SEVERITY_THRESHOLD",
		},
		{
			name: "missing max_concurrent_agents",
			config: clusterPrefix + `subscribe_mode: "faults"
workspace_root: "./incidents"
agent_script_path: "./run-agent.sh"
agent_timeout: 300
agent_model: "sonnet"
agent_cli: "claude"
agent_image: "nightcrier-agent:latest"
severity_threshold: "ERROR"
anthropic_api_key: "test-key"`,
			expectedFieldName: "max_concurrent_agents",
			expectedEnvVar:    "MAX_CONCURRENT_AGENTS",
		},
		{
			name: "missing global_queue_size",
			config: clusterPrefix + `subscribe_mode: "faults"
workspace_root: "./incidents"
agent_script_path: "./run-agent.sh"
agent_timeout: 300
agent_model: "sonnet"
agent_cli: "claude"
agent_image: "nightcrier-agent:latest"
severity_threshold: "ERROR"
max_concurrent_agents: 5
anthropic_api_key: "test-key"`,
			expectedFieldName: "global_queue_size",
			expectedEnvVar:    "GLOBAL_QUEUE_SIZE",
		},
		{
			name: "missing cluster_queue_size",
			config: clusterPrefix + `subscribe_mode: "faults"
workspace_root: "./incidents"
agent_script_path: "./run-agent.sh"
agent_timeout: 300
agent_model: "sonnet"
agent_cli: "claude"
agent_image: "nightcrier-agent:latest"
severity_threshold: "ERROR"
max_concurrent_agents: 5
global_queue_size: 100
anthropic_api_key: "test-key"`,
			expectedFieldName: "cluster_queue_size",
			expectedEnvVar:    "CLUSTER_QUEUE_SIZE",
		},
		{
			name: "missing queue_overflow_policy",
			config: clusterPrefix + `subscribe_mode: "faults"
workspace_root: "./incidents"
agent_script_path: "./run-agent.sh"
agent_timeout: 300
agent_model: "sonnet"
agent_cli: "claude"
agent_image: "nightcrier-agent:latest"
severity_threshold: "ERROR"
max_concurrent_agents: 5
global_queue_size: 100
cluster_queue_size: 10
dedup_window_seconds: 300
anthropic_api_key: "test-key"`,
			expectedFieldName: "queue_overflow_policy",
			expectedEnvVar:    "QUEUE_OVERFLOW_POLICY",
		},
		{
			name: "missing shutdown_timeout",
			config: clusterPrefix + `subscribe_mode: "faults"
workspace_root: "./incidents"
agent_script_path: "./run-agent.sh"
agent_timeout: 300
agent_model: "sonnet"
agent_cli: "claude"
agent_image: "nightcrier-agent:latest"
severity_threshold: "ERROR"
max_concurrent_agents: 5
global_queue_size: 100
cluster_queue_size: 10
dedup_window_seconds: 300
queue_overflow_policy: "drop"
anthropic_api_key: "test-key"`,
			expectedFieldName: "shutdown_timeout",
			expectedEnvVar:    "SHUTDOWN_TIMEOUT_SECONDS",
		},
		{
			name: "missing sse_reconnect_initial_backoff",
			config: clusterPrefix + `subscribe_mode: "faults"
workspace_root: "./incidents"
agent_script_path: "./run-agent.sh"
agent_timeout: 300
agent_model: "sonnet"
agent_cli: "claude"
agent_image: "nightcrier-agent:latest"
severity_threshold: "ERROR"
max_concurrent_agents: 5
global_queue_size: 100
cluster_queue_size: 10
dedup_window_seconds: 300
queue_overflow_policy: "drop"
shutdown_timeout: 30
anthropic_api_key: "test-key"`,
			expectedFieldName: "sse_reconnect_initial_backoff",
			expectedEnvVar:    "SSE_RECONNECT_INITIAL_BACKOFF",
		},
		{
			name: "missing sse_reconnect_max_backoff",
			config: clusterPrefix + `subscribe_mode: "faults"
workspace_root: "./incidents"
agent_script_path: "./run-agent.sh"
agent_timeout: 300
agent_model: "sonnet"
agent_cli: "claude"
agent_image: "nightcrier-agent:latest"
severity_threshold: "ERROR"
max_concurrent_agents: 5
global_queue_size: 100
cluster_queue_size: 10
dedup_window_seconds: 300
queue_overflow_policy: "drop"
shutdown_timeout: 30
sse_reconnect_initial_backoff: 1
anthropic_api_key: "test-key"`,
			expectedFieldName: "sse_reconnect_max_backoff",
			expectedEnvVar:    "SSE_RECONNECT_MAX_BACKOFF",
		},
		{
			name: "missing sse_read_timeout",
			config: clusterPrefix + `subscribe_mode: "faults"
workspace_root: "./incidents"
agent_script_path: "./run-agent.sh"
agent_timeout: 300
agent_model: "sonnet"
agent_cli: "claude"
agent_image: "nightcrier-agent:latest"
severity_threshold: "ERROR"
max_concurrent_agents: 5
global_queue_size: 100
cluster_queue_size: 10
dedup_window_seconds: 300
queue_overflow_policy: "drop"
shutdown_timeout: 30
sse_reconnect_initial_backoff: 1
sse_reconnect_max_backoff: 60
anthropic_api_key: "test-key"`,
			expectedFieldName: "sse_read_timeout",
			expectedEnvVar:    "SSE_READ_TIMEOUT_SECONDS",
		},
		{
			name: "missing failure_threshold_for_alert",
			config: clusterPrefix + `subscribe_mode: "faults"
workspace_root: "./incidents"
agent_script_path: "./run-agent.sh"
agent_timeout: 300
agent_model: "sonnet"
agent_cli: "claude"
agent_image: "nightcrier-agent:latest"
severity_threshold: "ERROR"
max_concurrent_agents: 5
global_queue_size: 100
cluster_queue_size: 10
dedup_window_seconds: 300
queue_overflow_policy: "drop"
shutdown_timeout: 30
sse_reconnect_initial_backoff: 1
sse_reconnect_max_backoff: 60
sse_read_timeout: 120
anthropic_api_key: "test-key"`,
			expectedFieldName: "failure_threshold_for_alert",
			expectedEnvVar:    "FAILURE_THRESHOLD_FOR_ALERT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper()

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			if err := os.WriteFile(configPath, []byte(tt.config), 0644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			_, err := LoadWithConfigFile(configPath)
			if err == nil {
				t.Errorf("LoadWithConfigFile() should fail when %s is missing", tt.expectedFieldName)
				return
			}

			// Verify error message contains the field name
			if !contains(err.Error(), tt.expectedFieldName) {
				t.Errorf("error message should contain field name %q, got: %v", tt.expectedFieldName, err)
			}

			// Verify error message contains the environment variable name (if applicable)
			if tt.expectedEnvVar != "" && !contains(err.Error(), tt.expectedEnvVar) {
				t.Errorf("error message should contain environment variable %q, got: %v", tt.expectedEnvVar, err)
			}

			// Verify error message references config.example.yaml (skip for clusters which has different error)
			if tt.expectedFieldName != "clusters" && !contains(err.Error(), "config.example.yaml") {
				t.Errorf("error message should reference config.example.yaml, got: %v", err)
			}
		})
	}
}
