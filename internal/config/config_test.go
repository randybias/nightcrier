package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
monitored_clusters:
  - name: test-cluster
    mcp:
      endpoint: "http://localhost:8080/mcp"
subscribe_mode: "faults"
workspace_root: "./incidents"
agent:
  cli: "claude"
  model: "sonnet"
  timeout: 300
  system_prompt_file: "./prompts/system.md"
  allowed_tools: "all"
execution_defaults:
  namespace: "nightcrier"
  runner_image: "nc-agent-runner:latest"
  image_pull_policy: "IfNotPresent"
  timeout: 600
  memory_limit: "2Gi"
  cpu_limit: "1"
  cleanup_ttl: 3600
severity_threshold: "ERROR"
max_concurrent_agents: 5
global_queue_size: 100
cluster_failure_event_queue_size: 3
dedup_window_seconds: 300
queue_overflow_policy: "drop"
shutdown_timeout: 30
mcp_reconnect_initial_backoff: 1
mcp_reconnect_max_backoff: 60
mcp_read_timeout: 120
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
monitored_clusters:
  - name: test-cluster
    mcp:
      endpoint: "http://localhost:8080/mcp"
subscribe_mode: "faults"
workspace_root: "./incidents"
agent:
  cli: "claude"
  model: "sonnet"
  timeout: 300
  system_prompt_file: "./prompts/system.md"
  allowed_tools: "all"
execution_defaults:
  namespace: "nightcrier"
  runner_image: "nc-agent-runner:latest"
  image_pull_policy: "IfNotPresent"
  timeout: 600
  memory_limit: "2Gi"
  cpu_limit: "1"
  cleanup_ttl: 3600
severity_threshold: "ERROR"
max_concurrent_agents: 5
global_queue_size: 100
cluster_failure_event_queue_size: 3
dedup_window_seconds: 300
queue_overflow_policy: "drop"
shutdown_timeout: 30
mcp_reconnect_initial_backoff: 1
mcp_reconnect_max_backoff: 60
mcp_read_timeout: 120
failure_threshold_for_alert: 3
`
}

// buildTestConfig creates a complete config with specific field values
func buildTestConfig(overrides map[string]interface{}) string {
	// Default values
	values := map[string]interface{}{
		"subscribe_mode":                "faults",
		"workspace_root":                "./incidents",
		"severity_threshold":            "ERROR",
		"max_concurrent_agents":         5,
		"global_queue_size":             100,
		"cluster_failure_event_queue_size": 3,
		"dedup_window_seconds":          300,
		"queue_overflow_policy":         "drop",
		"shutdown_timeout":              30,
		"mcp_reconnect_initial_backoff": 1,
		"mcp_reconnect_max_backoff":     60,
		"mcp_read_timeout":              120,
		"failure_threshold_for_alert":   3,
		"anthropic_api_key":             "test-key",
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
monitored_clusters:
  - name: test-cluster
    mcp:
      endpoint: "http://localhost:8080/mcp"
subscribe_mode: "` + values["subscribe_mode"].(string) + `"
workspace_root: "` + values["workspace_root"].(string) + `"
agent:
  cli: "claude"
  model: "sonnet"
  timeout: 300
  system_prompt_file: "./prompts/system.md"
  allowed_tools: "all"
execution_defaults:
  namespace: "nightcrier"
  runner_image: "nc-agent-runner:latest"
  image_pull_policy: "IfNotPresent"
  timeout: 600
  memory_limit: "2Gi"
  cpu_limit: "1"
  cleanup_ttl: 3600
`
	// Add remaining values
	for k, v := range values {
		if k == "subscribe_mode" || k == "workspace_root" {
			continue // Already added
		}
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
	if cfg.Agent.Model != "sonnet" {
		t.Errorf("Agent.Model = %q, want %q", cfg.Agent.Model, "sonnet")
	}
	if cfg.Agent.Timeout != 300 {
		t.Errorf("Agent.Timeout = %d, want %d", cfg.Agent.Timeout, 300)
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
	if cfg.ClusterFailureEventQueueSize != 3 {
		t.Errorf("ClusterFailureEventQueueSize = %d, want %d", cfg.ClusterFailureEventQueueSize, 3)
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
	if cfg.Agent.Model != "opus" {
		t.Errorf("Agent.Model = %q, want %q", cfg.Agent.Model, "opus")
	}
	if cfg.Agent.Timeout != 600 {
		t.Errorf("Agent.Timeout = %d, want %d", cfg.Agent.Timeout, 600)
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
monitored_clusters:
  - name: config-file-cluster
    mcp:
      endpoint: "http://config-file-server:8080/mcp"
subscribe_mode: "faults"
workspace_root: "/config/incidents"
log_level: "warn"
agent:
  cli: "claude"
  model: "haiku"
  timeout: 120
  system_prompt_file: "./prompts/system.md"
  allowed_tools: "all"
execution_defaults:
  namespace: "nightcrier"
  runner_image: "nc-agent-runner:latest"
  image_pull_policy: "IfNotPresent"
  timeout: 600
  memory_limit: "2Gi"
  cpu_limit: "1"
  cleanup_ttl: 3600
severity_threshold: "CRITICAL"
max_concurrent_agents: 3
global_queue_size: 50
cluster_failure_event_queue_size: 5
dedup_window_seconds: 600
queue_overflow_policy: "reject"
shutdown_timeout: 30
mcp_reconnect_initial_backoff: 1
mcp_reconnect_max_backoff: 60
mcp_read_timeout: 120
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

	if len(cfg.MonitoredClusters) != 1 || cfg.MonitoredClusters[0].MCP.Endpoint != "http://config-file-server:8080/mcp" {
		t.Errorf("Clusters[0].MCP.Endpoint = %q, want %q", cfg.MonitoredClusters[0].MCP.Endpoint, "http://config-file-server:8080/mcp")
	}
	if cfg.WorkspaceRoot != "/config/incidents" {
		t.Errorf("WorkspaceRoot = %q, want %q", cfg.WorkspaceRoot, "/config/incidents")
	}
	if cfg.LogLevel != "warn" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "warn")
	}
	if cfg.Agent.Model != "haiku" {
		t.Errorf("Agent.Model = %q, want %q", cfg.Agent.Model, "haiku")
	}
	if cfg.Agent.Timeout != 120 {
		t.Errorf("Agent.Timeout = %d, want %d", cfg.Agent.Timeout, 120)
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
	if cfg.ClusterFailureEventQueueSize != 5 {
		t.Errorf("ClusterFailureEventQueueSize = %d, want %d", cfg.ClusterFailureEventQueueSize, 5)
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
monitored_clusters:
  - name: config-file-cluster
    mcp:
      endpoint: "http://config-file-server:8080/mcp"
subscribe_mode: "faults"
workspace_root: "/config/incidents"
log_level: "warn"
agent:
  cli: "claude"
  model: "sonnet"
  timeout: 120
  system_prompt_file: "./prompts/system.md"
  allowed_tools: "all"
execution_defaults:
  namespace: "nightcrier"
  runner_image: "nc-agent-runner:latest"
  image_pull_policy: "IfNotPresent"
  timeout: 600
  memory_limit: "2Gi"
  cpu_limit: "1"
  cleanup_ttl: 3600
severity_threshold: "ERROR"
max_concurrent_agents: 5
global_queue_size: 100
cluster_failure_event_queue_size: 3
dedup_window_seconds: 300
queue_overflow_policy: "drop"
shutdown_timeout: 30
mcp_reconnect_initial_backoff: 1
mcp_reconnect_max_backoff: 60
mcp_read_timeout: 120
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
	if cfg.Agent.Timeout != 120 {
		t.Errorf("Agent.Timeout = %d, want %d (from config file)", cfg.Agent.Timeout, 120)
	}
}

func TestValidation_ZeroClustersAllowed(t *testing.T) {
	resetViper()

	// Config without monitored_clusters should now be valid (zero clusters at startup allowed)
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
subscribe_mode: "faults"
workspace_root: "./incidents"
agent:
  cli: "claude"
  model: "sonnet"
  timeout: 300
  system_prompt_file: "./prompts/system.md"
  allowed_tools: "all"
execution_defaults:
  namespace: "nightcrier"
  runner_image: "nc-agent-runner:latest"
  image_pull_policy: "IfNotPresent"
  timeout: 600
  memory_limit: "2Gi"
  cpu_limit: "1"
  cleanup_ttl: 3600
severity_threshold: "ERROR"
max_concurrent_agents: 5
global_queue_size: 100
cluster_failure_event_queue_size: 3
dedup_window_seconds: 300
queue_overflow_policy: "drop"
shutdown_timeout: 30
mcp_reconnect_initial_backoff: 1
mcp_reconnect_max_backoff: 60
mcp_read_timeout: 120
failure_threshold_for_alert: 3
anthropic_api_key: "test-key"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadWithConfigFile(configPath)
	if err != nil {
		t.Errorf("LoadWithConfigFile() should succeed with zero monitored_clusters, got error: %v", err)
	}
	if len(cfg.MonitoredClusters) != 0 {
		t.Errorf("MonitoredClusters should be empty, got %d clusters", len(cfg.MonitoredClusters))
	}
}

func TestValidation_InvalidSeverityThreshold(t *testing.T) {
	resetViper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
monitored_clusters:
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
monitored_clusters:
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
			name:    "cluster_failure_event_queue_size < 1",
			config:  clusterPrefix + "cluster_failure_event_queue_size: 0\nanthropic_api_key: \"test-key\"\n",
			wantErr: true,
		},
		{
			name:    "dedup_window_seconds negative",
			config:  clusterPrefix + "dedup_window_seconds: -1\nanthropic_api_key: \"test-key\"\n",
			wantErr: true,
		},
		{
			name:    "agent_timeout < 1",
			config:  clusterPrefix + "agent:\n  timeout: 0\nanthropic_api_key: \"test-key\"\n",
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
monitored_clusters:
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

func TestValidation_MCPReconnectSettings(t *testing.T) {
	resetViper()

	clusterPrefix := `
monitored_clusters:
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
			config:  clusterPrefix + "mcp_reconnect_initial_backoff: 0\nanthropic_api_key: \"test-key\"\n",
			wantErr: true,
		},
		{
			name:    "max backoff < initial backoff",
			config:  clusterPrefix + "mcp_reconnect_initial_backoff: 10\nmcp_reconnect_max_backoff: 5\nanthropic_api_key: \"test-key\"\n",
			wantErr: true,
		},
		{
			name:    "read timeout < 1",
			config:  clusterPrefix + "mcp_read_timeout: 0\nanthropic_api_key: \"test-key\"\n",
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
			configContent := buildTestConfig(map[string]interface{}{
				"severity_threshold": severity,
			})
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
			name: "enabled with Azure storage URL",
			config: completeTestConfigWith(`
object_storage:
  url: "azblob://incidents"
  azure_storage_account: "test"
  azure_storage_key: "key123"
`),
			enabled: true,
		},
		{
			name: "enabled with account and key",
			config: completeTestConfigWith(`
object_storage:
  url: "azblob://incidents"
  azure_storage_account: "teststorage"
  azure_storage_key: "key123"
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

func TestGetObjectStorageType(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "empty URL",
			url:      "",
			expected: "not_configured",
		},
		{
			name:     "memory storage",
			url:      "mem://",
			expected: "memory",
		},
		{
			name:     "azure blob storage",
			url:      "azblob://incidents",
			expected: "azure_blob",
		},
		{
			name:     "s3 storage",
			url:      "s3://my-bucket/path",
			expected: "s3",
		},
		{
			name:     "gcs storage",
			url:      "gs://my-bucket",
			expected: "gcs",
		},
		{
			name:     "file storage",
			url:      "file:///var/data/incidents",
			expected: "local_filesystem",
		},
		{
			name:     "unknown scheme",
			url:      "custom://something",
			expected: "custom",
		},
		{
			name:     "malformed URL without scheme",
			url:      "no-scheme-here",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				ObjectStorage: ObjectStorage{
					URL: tt.url,
				},
			}

			result := cfg.GetObjectStorageType()
			if result != tt.expected {
				t.Errorf("GetObjectStorageType() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestGetAzureSASExpiry has been removed as the GetAzureSASExpiry method
// is deprecated and will be removed when the storage layer is refactored.
// Signed URL expiry is now configured via ObjectStorage.SignedURLExpiry
// and validated in ValidateObjectStorageConfig().

func TestValidation_RequiresLLMAPIKey(t *testing.T) {
	resetViper()

	// Ensure no API keys are set in environment
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Unsetenv("OPENAI_API_KEY")
	os.Unsetenv("GEMINI_API_KEY")

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

	// Base config without circuit breaker settings (uses defaults)
	baseConfigNoCircuitBreaker := completeTestConfig()

	// Custom config with custom circuit breaker settings
	customConfig := buildTestConfig(map[string]interface{}{
		"notify_on_agent_failure":      false,
		"failure_threshold_for_alert":  5,
		"upload_failed_investigations": true,
	})

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
		"workspace_root":               "/tmp/incidents",
		"log_level":                    "debug",
		"severity_threshold":           "WARNING",
		"max_concurrent_agents":        10,
		"notify_on_agent_failure":      false,
		"failure_threshold_for_alert":  5,
		"upload_failed_investigations": true,
	}) + `
object_storage:
  url: "azblob://incidents"
  azure_storage_account: "teststorage"
  azure_storage_key: "testkey"
`
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
	if len(cfg.MonitoredClusters) != 1 || cfg.MonitoredClusters[0].MCP.Endpoint != "http://localhost:8080/mcp" {
		t.Errorf("Clusters[0].MCP.Endpoint = %q, want %q", cfg.MonitoredClusters[0].MCP.Endpoint, "http://localhost:8080/mcp")
	}
	if cfg.MaxConcurrentAgents != 10 {
		t.Errorf("MaxConcurrentAgents = %d, want 10", cfg.MaxConcurrentAgents)
	}
	// Note: Azure storage account is parsed from URL and azure_storage_account field
	// The actual parsing behavior may combine or override values
	if cfg.ObjectStorage.AzureStorageAccount == "" {
		t.Errorf("ObjectStorage.AzureStorageAccount should not be empty")
	}
}

// TestValidation_MissingRequiredFields tests that all required fields generate helpful error messages
func TestValidation_MissingRequiredFields(t *testing.T) {
	// Cluster config prefix for tests that need clusters defined
	clusterPrefix := `
monitored_clusters:
  - name: test-cluster
    mcp:
      endpoint: "http://localhost:8080/mcp"
`

	// Base agent config (nested structure)
	baseAgentConfig := `agent:
  cli: "claude"
  model: "sonnet"
  timeout: 300
  system_prompt_file: "./prompts/system.md"
  allowed_tools: "all"
`

	// Base execution defaults config (nested structure)
	baseExecutionConfig := `execution_defaults:
  namespace: "nightcrier"
  runner_image: "nc-agent-runner:latest"
  image_pull_policy: "IfNotPresent"
  timeout: 600
  memory_limit: "2Gi"
  cpu_limit: "1"
  cleanup_ttl: 3600
`

	tests := []struct {
		name              string
		config            string
		expectedFieldName string
		expectedEnvVar    string
		skipDetailedCheck bool // Skip env var and config.example.yaml check for nested configs
	}{
		// NOTE: "missing clusters" test removed - monitored_clusters is now optional (zero clusters allowed at startup)
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
			name: "missing agent.timeout",
			config: clusterPrefix + `subscribe_mode: "faults"
workspace_root: "./incidents"
agent:
  cli: "claude"
  model: "sonnet"
anthropic_api_key: "test-key"`,
			expectedFieldName: "agent.timeout",
			expectedEnvVar:    "AGENT_TIMEOUT",
			skipDetailedCheck: true,
		},
		{
			name: "missing agent.model",
			config: clusterPrefix + `subscribe_mode: "faults"
workspace_root: "./incidents"
agent:
  cli: "claude"
  timeout: 300
anthropic_api_key: "test-key"`,
			expectedFieldName: "agent.model",
			expectedEnvVar:    "AGENT_MODEL",
			skipDetailedCheck: true,
		},
		{
			name: "missing agent.cli",
			config: clusterPrefix + `subscribe_mode: "faults"
workspace_root: "./incidents"
agent:
  model: "sonnet"
  timeout: 300
anthropic_api_key: "test-key"`,
			expectedFieldName: "agent.cli",
			expectedEnvVar:    "AGENT_CLI",
			skipDetailedCheck: true,
		},
		{
			name: "missing severity_threshold",
			config: clusterPrefix + `subscribe_mode: "faults"
workspace_root: "./incidents"
` + baseAgentConfig + baseExecutionConfig + `anthropic_api_key: "test-key"`,
			expectedFieldName: "severity_threshold",
			expectedEnvVar:    "SEVERITY_THRESHOLD",
		},
		{
			name: "missing max_concurrent_agents",
			config: clusterPrefix + `subscribe_mode: "faults"
workspace_root: "./incidents"
` + baseAgentConfig + baseExecutionConfig + `severity_threshold: "ERROR"
anthropic_api_key: "test-key"`,
			expectedFieldName: "max_concurrent_agents",
			expectedEnvVar:    "MAX_CONCURRENT_AGENTS",
		},
		{
			name: "missing global_queue_size",
			config: clusterPrefix + `subscribe_mode: "faults"
workspace_root: "./incidents"
` + baseAgentConfig + baseExecutionConfig + `severity_threshold: "ERROR"
max_concurrent_agents: 5
anthropic_api_key: "test-key"`,
			expectedFieldName: "global_queue_size",
			expectedEnvVar:    "GLOBAL_QUEUE_SIZE",
		},
		{
			name: "missing queue_overflow_policy",
			config: clusterPrefix + `subscribe_mode: "faults"
workspace_root: "./incidents"
` + baseAgentConfig + baseExecutionConfig + `severity_threshold: "ERROR"
max_concurrent_agents: 5
global_queue_size: 100
cluster_failure_event_queue_size: 3
dedup_window_seconds: 300
anthropic_api_key: "test-key"`,
			expectedFieldName: "queue_overflow_policy",
			expectedEnvVar:    "QUEUE_OVERFLOW_POLICY",
		},
		{
			name: "missing shutdown_timeout",
			config: clusterPrefix + `subscribe_mode: "faults"
workspace_root: "./incidents"
` + baseAgentConfig + baseExecutionConfig + `severity_threshold: "ERROR"
max_concurrent_agents: 5
global_queue_size: 100
cluster_failure_event_queue_size: 3
dedup_window_seconds: 300
queue_overflow_policy: "drop"
anthropic_api_key: "test-key"`,
			expectedFieldName: "shutdown_timeout",
			expectedEnvVar:    "SHUTDOWN_TIMEOUT_SECONDS",
		},
		{
			name: "missing mcp_reconnect_initial_backoff",
			config: clusterPrefix + `subscribe_mode: "faults"
workspace_root: "./incidents"
` + baseAgentConfig + baseExecutionConfig + `severity_threshold: "ERROR"
max_concurrent_agents: 5
global_queue_size: 100
cluster_failure_event_queue_size: 3
dedup_window_seconds: 300
queue_overflow_policy: "drop"
shutdown_timeout: 30
anthropic_api_key: "test-key"`,
			expectedFieldName: "mcp_reconnect_initial_backoff",
			expectedEnvVar:    "MCP_RECONNECT_INITIAL_BACKOFF",
		},
		{
			name: "missing mcp_reconnect_max_backoff",
			config: clusterPrefix + `subscribe_mode: "faults"
workspace_root: "./incidents"
` + baseAgentConfig + baseExecutionConfig + `severity_threshold: "ERROR"
max_concurrent_agents: 5
global_queue_size: 100
cluster_failure_event_queue_size: 3
dedup_window_seconds: 300
queue_overflow_policy: "drop"
shutdown_timeout: 30
mcp_reconnect_initial_backoff: 1
anthropic_api_key: "test-key"`,
			expectedFieldName: "mcp_reconnect_max_backoff",
			expectedEnvVar:    "MCP_RECONNECT_MAX_BACKOFF",
		},
		{
			name: "missing mcp_read_timeout",
			config: clusterPrefix + `subscribe_mode: "faults"
workspace_root: "./incidents"
` + baseAgentConfig + baseExecutionConfig + `severity_threshold: "ERROR"
max_concurrent_agents: 5
global_queue_size: 100
cluster_failure_event_queue_size: 3
dedup_window_seconds: 300
queue_overflow_policy: "drop"
shutdown_timeout: 30
mcp_reconnect_initial_backoff: 1
mcp_reconnect_max_backoff: 60
anthropic_api_key: "test-key"`,
			expectedFieldName: "mcp_read_timeout",
			expectedEnvVar:    "MCP_READ_TIMEOUT_SECONDS",
		},
		{
			name: "missing failure_threshold_for_alert",
			config: clusterPrefix + `subscribe_mode: "faults"
workspace_root: "./incidents"
` + baseAgentConfig + baseExecutionConfig + `severity_threshold: "ERROR"
max_concurrent_agents: 5
global_queue_size: 100
cluster_failure_event_queue_size: 3
dedup_window_seconds: 300
queue_overflow_policy: "drop"
shutdown_timeout: 30
mcp_reconnect_initial_backoff: 1
mcp_reconnect_max_backoff: 60
mcp_read_timeout: 120
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

			// Skip detailed checks for nested configs which have different validation patterns
			if tt.skipDetailedCheck {
				return
			}

			// Verify error message contains the environment variable name (if applicable)
			if tt.expectedEnvVar != "" && !contains(err.Error(), tt.expectedEnvVar) {
				t.Errorf("error message should contain environment variable %q, got: %v", tt.expectedEnvVar, err)
			}

			// Verify error message references config.example.yaml
			if !contains(err.Error(), "config.example.yaml") {
				t.Errorf("error message should reference config.example.yaml, got: %v", err)
			}
		})
	}
}

// TestStateStorage_DefaultToFilesystem tests that state storage defaults to filesystem for backward compatibility
func TestStateStorage_DefaultToFilesystem(t *testing.T) {
	resetViper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := completeTestConfig() // No state_storage section
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadWithConfigFile(configPath)
	if err != nil {
		t.Fatalf("LoadWithConfigFile() failed: %v", err)
	}

	if cfg.StateStorage.Type != "filesystem" {
		t.Errorf("StateStorage.Type = %q, want %q (default)", cfg.StateStorage.Type, "filesystem")
	}

	if cfg.IsSQLStorageEnabled() {
		t.Error("IsSQLStorageEnabled() = true, want false for default filesystem storage")
	}

	if cfg.GetStateStorageType() != "filesystem" {
		t.Errorf("GetStateStorageType() = %q, want %q", cfg.GetStateStorageType(), "filesystem")
	}
}

// TestStateStorage_SQLiteConfiguration tests SQLite storage configuration
func TestStateStorage_SQLiteConfiguration(t *testing.T) {
	resetViper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := completeTestConfigWith(`
state_storage:
  type: "sqlite"
  sqlite_path: "/custom/path/nightcrier.db"
  migrations_path: "./custom/migrations"
`)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadWithConfigFile(configPath)
	if err != nil {
		t.Fatalf("LoadWithConfigFile() failed: %v", err)
	}

	if cfg.StateStorage.Type != "sqlite" {
		t.Errorf("StateStorage.Type = %q, want %q", cfg.StateStorage.Type, "sqlite")
	}

	if cfg.StateStorage.SQLitePath != "/custom/path/nightcrier.db" {
		t.Errorf("StateStorage.SQLitePath = %q, want %q", cfg.StateStorage.SQLitePath, "/custom/path/nightcrier.db")
	}

	if cfg.StateStorage.MigrationsPath != "./custom/migrations" {
		t.Errorf("StateStorage.MigrationsPath = %q, want %q", cfg.StateStorage.MigrationsPath, "./custom/migrations")
	}

	if !cfg.IsSQLStorageEnabled() {
		t.Error("IsSQLStorageEnabled() = false, want true for sqlite storage")
	}
}

// TestStateStorage_SQLiteDefaultPath tests that SQLite uses default path when not specified
func TestStateStorage_SQLiteDefaultPath(t *testing.T) {
	resetViper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := completeTestConfigWith(`
state_storage:
  type: "sqlite"
`)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadWithConfigFile(configPath)
	if err != nil {
		t.Fatalf("LoadWithConfigFile() failed: %v", err)
	}

	expectedPath := filepath.Join("./incidents", "nightcrier.db")
	if cfg.StateStorage.SQLitePath != expectedPath {
		t.Errorf("StateStorage.SQLitePath = %q, want %q (default)", cfg.StateStorage.SQLitePath, expectedPath)
	}

	if cfg.StateStorage.MigrationsPath != "./migrations" {
		t.Errorf("StateStorage.MigrationsPath = %q, want %q (default)", cfg.StateStorage.MigrationsPath, "./migrations")
	}
}

// TestStateStorage_PostgresWithConnectionString tests PostgreSQL configuration with connection string
func TestStateStorage_PostgresWithConnectionString(t *testing.T) {
	resetViper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := completeTestConfigWith(`
state_storage:
  type: "postgres"
  postgres_connection_string: "postgres://user:pass@localhost:5432/nightcrier?sslmode=disable"
  migrations_path: "./migrations"
`)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadWithConfigFile(configPath)
	if err != nil {
		t.Fatalf("LoadWithConfigFile() failed: %v", err)
	}

	if cfg.StateStorage.Type != "postgres" {
		t.Errorf("StateStorage.Type = %q, want %q", cfg.StateStorage.Type, "postgres")
	}

	if cfg.StateStorage.PostgresConnectionString != "postgres://user:pass@localhost:5432/nightcrier?sslmode=disable" {
		t.Errorf("StateStorage.PostgresConnectionString = %q, unexpected value", cfg.StateStorage.PostgresConnectionString)
	}

	if !cfg.IsSQLStorageEnabled() {
		t.Error("IsSQLStorageEnabled() = false, want true for postgres storage")
	}
}

// TestStateStorage_PostgresWithIndividualFields tests PostgreSQL configuration with individual fields
func TestStateStorage_PostgresWithIndividualFields(t *testing.T) {
	resetViper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := completeTestConfigWith(`
state_storage:
  type: "postgres"
  postgres_host: "db.example.com"
  postgres_port: 5433
  postgres_database: "nightcrier_prod"
  postgres_user: "nightcrier_user"
  postgres_password: "secret"
`)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadWithConfigFile(configPath)
	if err != nil {
		t.Fatalf("LoadWithConfigFile() failed: %v", err)
	}

	if cfg.StateStorage.PostgresHost != "db.example.com" {
		t.Errorf("StateStorage.PostgresHost = %q, want %q", cfg.StateStorage.PostgresHost, "db.example.com")
	}

	if cfg.StateStorage.PostgresPort != 5433 {
		t.Errorf("StateStorage.PostgresPort = %d, want %d", cfg.StateStorage.PostgresPort, 5433)
	}

	if cfg.StateStorage.PostgresDatabase != "nightcrier_prod" {
		t.Errorf("StateStorage.PostgresDatabase = %q, want %q", cfg.StateStorage.PostgresDatabase, "nightcrier_prod")
	}

	if cfg.StateStorage.PostgresUser != "nightcrier_user" {
		t.Errorf("StateStorage.PostgresUser = %q, want %q", cfg.StateStorage.PostgresUser, "nightcrier_user")
	}

	if cfg.StateStorage.PostgresPassword != "secret" {
		t.Errorf("StateStorage.PostgresPassword = %q, want %q", cfg.StateStorage.PostgresPassword, "secret")
	}
}

// TestStateStorage_PostgresDefaultPort tests that PostgreSQL uses default port when not specified
func TestStateStorage_PostgresDefaultPort(t *testing.T) {
	resetViper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := completeTestConfigWith(`
state_storage:
  type: "postgres"
  postgres_host: "localhost"
  postgres_database: "nightcrier"
  postgres_user: "postgres"
`)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadWithConfigFile(configPath)
	if err != nil {
		t.Fatalf("LoadWithConfigFile() failed: %v", err)
	}

	if cfg.StateStorage.PostgresPort != 5432 {
		t.Errorf("StateStorage.PostgresPort = %d, want %d (default)", cfg.StateStorage.PostgresPort, 5432)
	}
}

// TestStateStorage_InvalidType tests validation of invalid storage type
func TestStateStorage_InvalidType(t *testing.T) {
	resetViper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := completeTestConfigWith(`
state_storage:
  type: "invalid"
`)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := LoadWithConfigFile(configPath)
	if err == nil {
		t.Error("LoadWithConfigFile() should fail with invalid storage type")
	}

	if !contains(err.Error(), "invalid state_storage.type") {
		t.Errorf("error should mention invalid type, got: %v", err)
	}
}

// TestStateStorage_PostgresMissingHost tests validation when postgres host is missing
func TestStateStorage_PostgresMissingHost(t *testing.T) {
	resetViper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := completeTestConfigWith(`
state_storage:
  type: "postgres"
  postgres_database: "nightcrier"
  postgres_user: "postgres"
`)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := LoadWithConfigFile(configPath)
	if err == nil {
		t.Error("LoadWithConfigFile() should fail when postgres host is missing")
	}

	if !contains(err.Error(), "STATE_STORAGE_POSTGRES_HOST") {
		t.Errorf("error should mention missing host, got: %v", err)
	}
}

// TestStateStorage_PostgresMissingDatabase tests validation when postgres database is missing
func TestStateStorage_PostgresMissingDatabase(t *testing.T) {
	resetViper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := completeTestConfigWith(`
state_storage:
  type: "postgres"
  postgres_host: "localhost"
  postgres_user: "postgres"
`)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := LoadWithConfigFile(configPath)
	if err == nil {
		t.Error("LoadWithConfigFile() should fail when postgres database is missing")
	}

	if !contains(err.Error(), "STATE_STORAGE_POSTGRES_DATABASE") {
		t.Errorf("error should mention missing database, got: %v", err)
	}
}

// TestStateStorage_PostgresMissingUser tests validation when postgres user is missing
func TestStateStorage_PostgresMissingUser(t *testing.T) {
	resetViper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := completeTestConfigWith(`
state_storage:
  type: "postgres"
  postgres_host: "localhost"
  postgres_database: "nightcrier"
`)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := LoadWithConfigFile(configPath)
	if err == nil {
		t.Error("LoadWithConfigFile() should fail when postgres user is missing")
	}

	if !contains(err.Error(), "STATE_STORAGE_POSTGRES_USER") {
		t.Errorf("error should mention missing user, got: %v", err)
	}
}

// TestStateStorage_InvalidConnectionString tests validation of invalid postgres connection strings
func TestStateStorage_InvalidConnectionString(t *testing.T) {
	tests := []struct {
		name    string
		connStr string
	}{
		{"empty", ""},
		{"no protocol", "user:pass@localhost:5432/dbname"},
		{"no @", "postgres://userpass:localhost:5432/dbname"},
		{"wrong protocol", "mysql://user:pass@localhost:5432/dbname"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper()

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			configContent := completeTestConfigWith(fmt.Sprintf(`
state_storage:
  type: "postgres"
  postgres_connection_string: "%s"
`, tt.connStr))
			if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			_, err := LoadWithConfigFile(configPath)
			if err == nil {
				t.Errorf("LoadWithConfigFile() should fail with invalid connection string: %q", tt.connStr)
			}
		})
	}
}

// TestStateStorage_FromEnvVars tests loading state storage config from environment variables
func TestStateStorage_FromEnvVars(t *testing.T) {
	resetViper()

	// Create a minimal config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := completeTestConfig()
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Set env vars for SQLite storage
	os.Setenv("STATE_STORAGE_TYPE", "sqlite")
	os.Setenv("STATE_STORAGE_SQLITE_PATH", "/env/path/nightcrier.db")
	os.Setenv("STATE_STORAGE_MIGRATIONS_PATH", "/env/migrations")

	defer func() {
		os.Unsetenv("STATE_STORAGE_TYPE")
		os.Unsetenv("STATE_STORAGE_SQLITE_PATH")
		os.Unsetenv("STATE_STORAGE_MIGRATIONS_PATH")
	}()

	cfg, err := LoadWithConfigFile(configPath)
	if err != nil {
		t.Fatalf("LoadWithConfigFile() failed: %v", err)
	}

	if cfg.StateStorage.Type != "sqlite" {
		t.Errorf("StateStorage.Type = %q, want %q", cfg.StateStorage.Type, "sqlite")
	}

	if cfg.StateStorage.SQLitePath != "/env/path/nightcrier.db" {
		t.Errorf("StateStorage.SQLitePath = %q, want %q", cfg.StateStorage.SQLitePath, "/env/path/nightcrier.db")
	}

	if cfg.StateStorage.MigrationsPath != "/env/migrations" {
		t.Errorf("StateStorage.MigrationsPath = %q, want %q", cfg.StateStorage.MigrationsPath, "/env/migrations")
	}
}

// TestStateStorage_PostgresFromEnvVars tests loading postgres config from environment variables
func TestStateStorage_PostgresFromEnvVars(t *testing.T) {
	resetViper()

	// Create a minimal config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := completeTestConfig()
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Set env vars for PostgreSQL storage
	os.Setenv("STATE_STORAGE_TYPE", "postgres")
	os.Setenv("STATE_STORAGE_POSTGRES_HOST", "env.db.example.com")
	os.Setenv("STATE_STORAGE_POSTGRES_PORT", "5433")
	os.Setenv("STATE_STORAGE_POSTGRES_DATABASE", "env_nightcrier")
	os.Setenv("STATE_STORAGE_POSTGRES_USER", "env_user")
	os.Setenv("STATE_STORAGE_POSTGRES_PASSWORD", "env_pass")

	defer func() {
		os.Unsetenv("STATE_STORAGE_TYPE")
		os.Unsetenv("STATE_STORAGE_POSTGRES_HOST")
		os.Unsetenv("STATE_STORAGE_POSTGRES_PORT")
		os.Unsetenv("STATE_STORAGE_POSTGRES_DATABASE")
		os.Unsetenv("STATE_STORAGE_POSTGRES_USER")
		os.Unsetenv("STATE_STORAGE_POSTGRES_PASSWORD")
	}()

	cfg, err := LoadWithConfigFile(configPath)
	if err != nil {
		t.Fatalf("LoadWithConfigFile() failed: %v", err)
	}

	if cfg.StateStorage.Type != "postgres" {
		t.Errorf("StateStorage.Type = %q, want %q", cfg.StateStorage.Type, "postgres")
	}

	if cfg.StateStorage.PostgresHost != "env.db.example.com" {
		t.Errorf("StateStorage.PostgresHost = %q, want %q", cfg.StateStorage.PostgresHost, "env.db.example.com")
	}

	if cfg.StateStorage.PostgresPort != 5433 {
		t.Errorf("StateStorage.PostgresPort = %d, want %d", cfg.StateStorage.PostgresPort, 5433)
	}

	if cfg.StateStorage.PostgresDatabase != "env_nightcrier" {
		t.Errorf("StateStorage.PostgresDatabase = %q, want %q", cfg.StateStorage.PostgresDatabase, "env_nightcrier")
	}

	if cfg.StateStorage.PostgresUser != "env_user" {
		t.Errorf("StateStorage.PostgresUser = %q, want %q", cfg.StateStorage.PostgresUser, "env_user")
	}

	if cfg.StateStorage.PostgresPassword != "env_pass" {
		t.Errorf("StateStorage.PostgresPassword = %q, want %q", cfg.StateStorage.PostgresPassword, "env_pass")
	}
}

// TestStateStorage_CaseInsensitiveType tests that storage type is case-insensitive
func TestStateStorage_CaseInsensitiveType(t *testing.T) {
	cases := []string{"SQLITE", "SQLite", "SqLiTe", "sqlite"}

	for _, tc := range cases {
		t.Run(tc, func(t *testing.T) {
			resetViper()

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			configContent := completeTestConfigWith(fmt.Sprintf(`
state_storage:
  type: "%s"
`, tc))
			if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			cfg, err := LoadWithConfigFile(configPath)
			if err != nil {
				t.Fatalf("LoadWithConfigFile() failed: %v", err)
			}

			if cfg.StateStorage.Type != "sqlite" {
				t.Errorf("StateStorage.Type = %q, want %q (normalized)", cfg.StateStorage.Type, "sqlite")
			}
		})
	}
}

// TestAgentConfig_Validate tests validation of AgentConfig
func TestAgentConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  AgentConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: AgentConfig{
				CLI:              "claude",
				Model:            "sonnet",
				Timeout:          300,
				SystemPromptFile: "./test.md",
				AllowedTools:     "all",
			},
			wantErr: false,
		},
		{
			name: "empty CLI",
			config: AgentConfig{
				CLI:     "",
				Model:   "sonnet",
				Timeout: 300,
			},
			wantErr: true,
			errMsg:  "agent.cli is required",
		},
		{
			name: "invalid CLI",
			config: AgentConfig{
				CLI:     "invalid",
				Model:   "sonnet",
				Timeout: 300,
			},
			wantErr: true,
			errMsg:  "invalid agent.cli",
		},
		{
			name: "empty Model",
			config: AgentConfig{
				CLI:     "claude",
				Model:   "",
				Timeout: 300,
			},
			wantErr: true,
			errMsg:  "agent.model is required",
		},
		{
			name: "zero Timeout",
			config: AgentConfig{
				CLI:     "claude",
				Model:   "sonnet",
				Timeout: 0,
			},
			wantErr: true,
			errMsg:  "agent.timeout must be >= 1",
		},
		{
			name: "negative Timeout",
			config: AgentConfig{
				CLI:     "claude",
				Model:   "sonnet",
				Timeout: -1,
			},
			wantErr: true,
			errMsg:  "agent.timeout must be >= 1",
		},
		{
			name: "valid CLI codex",
			config: AgentConfig{
				CLI:     "codex",
				Model:   "sonnet",
				Timeout: 300,
			},
			wantErr: false,
		},
		{
			name: "valid CLI gemini",
			config: AgentConfig{
				CLI:     "gemini",
				Model:   "pro",
				Timeout: 300,
			},
			wantErr: false,
		},
		{
			name: "valid CLI goose",
			config: AgentConfig{
				CLI:     "goose",
				Model:   "sonnet",
				Timeout: 300,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("AgentConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("AgentConfig.Validate() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

// TestExecutionDefaults_Validate tests validation of ExecutionDefaults
func TestExecutionDefaults_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  ExecutionDefaults
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: ExecutionDefaults{
				Namespace:       "nightcrier",
				RunnerImage:     "nc-agent-runner:latest",
				ImagePullPolicy: "IfNotPresent",
				Timeout:         600,
				MemoryLimit:     "2Gi",
				CPULimit:        "1",
				CleanupTTL:      3600,
			},
			wantErr: false,
		},
		{
			name: "valid with Always pull policy",
			config: ExecutionDefaults{
				Namespace:       "nightcrier",
				RunnerImage:     "nc-agent-runner:latest",
				ImagePullPolicy: "Always",
				Timeout:         600,
			},
			wantErr: false,
		},
		{
			name: "valid with Never pull policy",
			config: ExecutionDefaults{
				Namespace:       "nightcrier",
				RunnerImage:     "nc-agent-runner:latest",
				ImagePullPolicy: "Never",
				Timeout:         600,
			},
			wantErr: false,
		},
		{
			name: "invalid ImagePullPolicy",
			config: ExecutionDefaults{
				Namespace:       "nightcrier",
				RunnerImage:     "nc-agent-runner:latest",
				ImagePullPolicy: "Sometimes",
				Timeout:         600,
			},
			wantErr: true,
			errMsg:  "invalid execution_defaults.image_pull_policy",
		},
		{
			name: "empty ImagePullPolicy (valid - will use default)",
			config: ExecutionDefaults{
				Namespace:       "nightcrier",
				RunnerImage:     "nc-agent-runner:latest",
				ImagePullPolicy: "",
				Timeout:         600,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecutionDefaults.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ExecutionDefaults.Validate() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

// TestExecutionDefaults_ApplyDefaults tests that ApplyDefaults sets all default values correctly
func TestExecutionDefaults_ApplyDefaults(t *testing.T) {
	config := ExecutionDefaults{}
	config.ApplyDefaults()

	tests := []struct {
		field string
		got   interface{}
		want  interface{}
	}{
		{"Namespace", config.Namespace, "nightcrier"},
		{"RunnerImage", config.RunnerImage, "nc-agent-runner:latest"},
		{"ImagePullPolicy", config.ImagePullPolicy, "IfNotPresent"},
		{"Timeout", config.Timeout, 600},
		{"MemoryLimit", config.MemoryLimit, "2Gi"},
		{"CPULimit", config.CPULimit, "1"},
		{"CleanupTTL", config.CleanupTTL, 3600},
		{"MaxConcurrentAgents", config.MaxConcurrentAgents, 10},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("ExecutionDefaults.%s = %v, want %v", tt.field, tt.got, tt.want)
			}
		})
	}
}

// TestNestedConfigLoadFromYAML tests that nested YAML is properly loaded into nested struct fields
func TestNestedConfigLoadFromYAML(t *testing.T) {
	resetViper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
monitored_clusters:
  - name: test-cluster
    mcp:
      endpoint: "http://localhost:8080/mcp"
subscribe_mode: "faults"
workspace_root: "./incidents"
agent:
  cli: "gemini"
  model: "flash"
  timeout: 456
  system_prompt_file: "./custom.md"
  allowed_tools: "restricted"
  additional_prompt: "Be helpful"
execution_defaults:
  namespace: "custom-ns"
  runner_image: "custom-image:v1"
  image_pull_policy: "Always"
  timeout: 999
  memory_limit: "4Gi"
  cpu_limit: "2"
  cleanup_ttl: 7200
severity_threshold: "WARNING"
max_concurrent_agents: 8
global_queue_size: 200
cluster_failure_event_queue_size: 20
dedup_window_seconds: 600
queue_overflow_policy: "reject"
shutdown_timeout: 60
mcp_reconnect_initial_backoff: 2
mcp_reconnect_max_backoff: 120
mcp_read_timeout: 240
failure_threshold_for_alert: 5
anthropic_api_key: "test-key"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadWithConfigFile(configPath)
	if err != nil {
		t.Fatalf("LoadWithConfigFile() failed: %v", err)
	}

	// Test Agent config
	if cfg.Agent.CLI != "gemini" {
		t.Errorf("Agent.CLI = %q, want %q", cfg.Agent.CLI, "gemini")
	}
	if cfg.Agent.Model != "flash" {
		t.Errorf("Agent.Model = %q, want %q", cfg.Agent.Model, "flash")
	}
	if cfg.Agent.Timeout != 456 {
		t.Errorf("Agent.Timeout = %d, want %d", cfg.Agent.Timeout, 456)
	}
	if cfg.Agent.SystemPromptFile != "./custom.md" {
		t.Errorf("Agent.SystemPromptFile = %q, want %q", cfg.Agent.SystemPromptFile, "./custom.md")
	}
	if cfg.Agent.AllowedTools != "restricted" {
		t.Errorf("Agent.AllowedTools = %q, want %q", cfg.Agent.AllowedTools, "restricted")
	}
	if cfg.Agent.AdditionalPrompt != "Be helpful" {
		t.Errorf("Agent.AdditionalPrompt = %q, want %q", cfg.Agent.AdditionalPrompt, "Be helpful")
	}

	// Test ExecutionDefaults config
	if cfg.ExecutionDefaults.Namespace != "custom-ns" {
		t.Errorf("ExecutionDefaults.Namespace = %q, want %q", cfg.ExecutionDefaults.Namespace, "custom-ns")
	}
	if cfg.ExecutionDefaults.RunnerImage != "custom-image:v1" {
		t.Errorf("ExecutionDefaults.RunnerImage = %q, want %q", cfg.ExecutionDefaults.RunnerImage, "custom-image:v1")
	}
	if cfg.ExecutionDefaults.ImagePullPolicy != "Always" {
		t.Errorf("ExecutionDefaults.ImagePullPolicy = %q, want %q", cfg.ExecutionDefaults.ImagePullPolicy, "Always")
	}
	if cfg.ExecutionDefaults.Timeout != 999 {
		t.Errorf("ExecutionDefaults.Timeout = %d, want %d", cfg.ExecutionDefaults.Timeout, 999)
	}
	if cfg.ExecutionDefaults.MemoryLimit != "4Gi" {
		t.Errorf("ExecutionDefaults.MemoryLimit = %q, want %q", cfg.ExecutionDefaults.MemoryLimit, "4Gi")
	}
	if cfg.ExecutionDefaults.CPULimit != "2" {
		t.Errorf("ExecutionDefaults.CPULimit = %q, want %q", cfg.ExecutionDefaults.CPULimit, "2")
	}
	if cfg.ExecutionDefaults.CleanupTTL != 7200 {
		t.Errorf("ExecutionDefaults.CleanupTTL = %d, want %d", cfg.ExecutionDefaults.CleanupTTL, 7200)
	}
}

// TestNATSConfig_Validate tests validation of NATSConfig
func TestNATSConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  NATSConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "disabled config is valid",
			config: NATSConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "enabled with all fields valid",
			config: NATSConfig{
				Enabled:        true,
				Server:         "nats://localhost:4222",
				Token:          "test-token",
				ConnectTimeout: 5 * time.Second,
				ReconnectWait:  2 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "enabled without server",
			config: NATSConfig{
				Enabled:        true,
				Token:          "test-token",
				ConnectTimeout: 5 * time.Second,
				ReconnectWait:  2 * time.Second,
			},
			wantErr: true,
			errMsg:  "nats.server is required",
		},
		{
			name: "enabled without token",
			config: NATSConfig{
				Enabled:        true,
				Server:         "nats://localhost:4222",
				ConnectTimeout: 5 * time.Second,
				ReconnectWait:  2 * time.Second,
			},
			wantErr: true,
			errMsg:  "nats.token is required",
		},
		{
			name: "enabled with zero connect timeout",
			config: NATSConfig{
				Enabled:        true,
				Server:         "nats://localhost:4222",
				Token:          "test-token",
				ConnectTimeout: 0,
				ReconnectWait:  2 * time.Second,
			},
			wantErr: true,
			errMsg:  "nats.connect_timeout must be positive",
		},
		{
			name: "enabled with negative connect timeout",
			config: NATSConfig{
				Enabled:        true,
				Server:         "nats://localhost:4222",
				Token:          "test-token",
				ConnectTimeout: -1 * time.Second,
				ReconnectWait:  2 * time.Second,
			},
			wantErr: true,
			errMsg:  "nats.connect_timeout must be positive",
		},
		{
			name: "enabled with zero reconnect wait",
			config: NATSConfig{
				Enabled:        true,
				Server:         "nats://localhost:4222",
				Token:          "test-token",
				ConnectTimeout: 5 * time.Second,
				ReconnectWait:  0,
			},
			wantErr: true,
			errMsg:  "nats.reconnect_wait must be positive",
		},
		{
			name: "enabled with negative reconnect wait",
			config: NATSConfig{
				Enabled:        true,
				Server:         "nats://localhost:4222",
				Token:          "test-token",
				ConnectTimeout: 5 * time.Second,
				ReconnectWait:  -1 * time.Second,
			},
			wantErr: true,
			errMsg:  "nats.reconnect_wait must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("NATSConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("NATSConfig.Validate() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

// TestNATSConfig_ApplyDefaults tests that ApplyDefaults sets correct default values
func TestNATSConfig_ApplyDefaults(t *testing.T) {
	config := NATSConfig{}
	config.ApplyDefaults()

	tests := []struct {
		field string
		got   interface{}
		want  interface{}
	}{
		{"ConnectTimeout", config.ConnectTimeout, 5 * time.Second},
		{"ReconnectWait", config.ReconnectWait, 2 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("NATSConfig.%s = %v, want %v", tt.field, tt.got, tt.want)
			}
		})
	}
}

// TestNATSConfig_ApplyDefaultsPreservesExisting tests that defaults don't override existing values
func TestNATSConfig_ApplyDefaultsPreservesExisting(t *testing.T) {
	config := NATSConfig{
		ConnectTimeout: 10 * time.Second,
		ReconnectWait:  5 * time.Second,
	}
	config.ApplyDefaults()

	if config.ConnectTimeout != 10*time.Second {
		t.Errorf("ConnectTimeout = %v, want %v (should preserve existing)", config.ConnectTimeout, 10*time.Second)
	}
	if config.ReconnectWait != 5*time.Second {
		t.Errorf("ReconnectWait = %v, want %v (should preserve existing)", config.ReconnectWait, 5*time.Second)
	}
}

// TestNATSConfig_LoadFromYAML tests loading NATS config from YAML
func TestNATSConfig_LoadFromYAML(t *testing.T) {
	resetViper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := completeTestConfigWith(`
nats:
  enabled: true
  server: "nats://test-server:4222"
  token: "test-token-123"
  connect_timeout: "10s"
  reconnect_wait: "3s"
`)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadWithConfigFile(configPath)
	if err != nil {
		t.Fatalf("LoadWithConfigFile() failed: %v", err)
	}

	if !cfg.NATS.Enabled {
		t.Errorf("NATS.Enabled = %v, want true", cfg.NATS.Enabled)
	}
	if cfg.NATS.Server != "nats://test-server:4222" {
		t.Errorf("NATS.Server = %q, want %q", cfg.NATS.Server, "nats://test-server:4222")
	}
	if cfg.NATS.Token != "test-token-123" {
		t.Errorf("NATS.Token = %q, want %q", cfg.NATS.Token, "test-token-123")
	}
	if cfg.NATS.ConnectTimeout != 10*time.Second {
		t.Errorf("NATS.ConnectTimeout = %v, want %v", cfg.NATS.ConnectTimeout, 10*time.Second)
	}
	if cfg.NATS.ReconnectWait != 3*time.Second {
		t.Errorf("NATS.ReconnectWait = %v, want %v", cfg.NATS.ReconnectWait, 3*time.Second)
	}
}

// TestNATSConfig_LoadFromEnvVars tests loading NATS config from environment variables
func TestNATSConfig_LoadFromEnvVars(t *testing.T) {
	resetViper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := completeTestConfig()
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Set NATS env vars
	os.Setenv("NATS_ENABLED", "true")
	os.Setenv("NATS_SERVER", "nats://env-server:4222")
	os.Setenv("NATS_TOKEN", "env-token-456")
	os.Setenv("NATS_CONNECT_TIMEOUT", "15s")
	os.Setenv("NATS_RECONNECT_WAIT", "4s")

	defer func() {
		os.Unsetenv("NATS_ENABLED")
		os.Unsetenv("NATS_SERVER")
		os.Unsetenv("NATS_TOKEN")
		os.Unsetenv("NATS_CONNECT_TIMEOUT")
		os.Unsetenv("NATS_RECONNECT_WAIT")
	}()

	cfg, err := LoadWithConfigFile(configPath)
	if err != nil {
		t.Fatalf("LoadWithConfigFile() failed: %v", err)
	}

	if !cfg.NATS.Enabled {
		t.Errorf("NATS.Enabled = %v, want true", cfg.NATS.Enabled)
	}
	if cfg.NATS.Server != "nats://env-server:4222" {
		t.Errorf("NATS.Server = %q, want %q", cfg.NATS.Server, "nats://env-server:4222")
	}
	if cfg.NATS.Token != "env-token-456" {
		t.Errorf("NATS.Token = %q, want %q", cfg.NATS.Token, "env-token-456")
	}
	if cfg.NATS.ConnectTimeout != 15*time.Second {
		t.Errorf("NATS.ConnectTimeout = %v, want %v", cfg.NATS.ConnectTimeout, 15*time.Second)
	}
	if cfg.NATS.ReconnectWait != 4*time.Second {
		t.Errorf("NATS.ReconnectWait = %v, want %v", cfg.NATS.ReconnectWait, 4*time.Second)
	}
}

// TestNATSConfig_DisabledByDefault tests that NATS is disabled by default
func TestNATSConfig_DisabledByDefault(t *testing.T) {
	resetViper()

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

	if cfg.NATS.Enabled {
		t.Errorf("NATS.Enabled = true, want false (disabled by default)")
	}

	// Defaults should still be applied
	if cfg.NATS.ConnectTimeout != 5*time.Second {
		t.Errorf("NATS.ConnectTimeout = %v, want %v (default)", cfg.NATS.ConnectTimeout, 5*time.Second)
	}
	if cfg.NATS.ReconnectWait != 2*time.Second {
		t.Errorf("NATS.ReconnectWait = %v, want %v (default)", cfg.NATS.ReconnectWait, 2*time.Second)
	}
}

// TestNATSConfig_ValidationFailsWhenEnabledWithMissingFields tests validation failures
func TestNATSConfig_ValidationFailsWhenEnabledWithMissingFields(t *testing.T) {
	resetViper()

	tests := []struct {
		name       string
		yamlConfig string
		wantErrMsg string
	}{
		{
			name: "missing server",
			yamlConfig: `
nats:
  enabled: true
  token: "test-token"
`,
			wantErrMsg: "nats.server is required",
		},
		{
			name: "missing token",
			yamlConfig: `
nats:
  enabled: true
  server: "nats://localhost:4222"
`,
			wantErrMsg: "nats.token is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper()

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			configContent := completeTestConfigWith(tt.yamlConfig)
			if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			_, err := LoadWithConfigFile(configPath)
			if err == nil {
				t.Errorf("LoadWithConfigFile() should fail when %s", tt.name)
				return
			}

			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Errorf("error = %v, want error containing %q", err, tt.wantErrMsg)
			}
		})
	}
}

// TestNestedConfigEnvVarOverride tests that environment variables properly override nested YAML values
func TestNestedConfigEnvVarOverride(t *testing.T) {
	resetViper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
monitored_clusters:
  - name: test-cluster
    mcp:
      endpoint: "http://localhost:8080/mcp"
subscribe_mode: "faults"
workspace_root: "./incidents"
agent:
  cli: "claude"
  model: "sonnet"
  timeout: 300
  system_prompt_file: "./prompts/system.md"
  allowed_tools: "all"
execution_defaults:
  namespace: "nightcrier"
  runner_image: "nc-agent-runner:latest"
  image_pull_policy: "IfNotPresent"
  timeout: 600
  memory_limit: "2Gi"
  cpu_limit: "1"
  cleanup_ttl: 3600
severity_threshold: "ERROR"
max_concurrent_agents: 5
global_queue_size: 100
cluster_failure_event_queue_size: 3
dedup_window_seconds: 300
queue_overflow_policy: "drop"
shutdown_timeout: 30
mcp_reconnect_initial_backoff: 1
mcp_reconnect_max_backoff: 60
mcp_read_timeout: 120
failure_threshold_for_alert: 3
anthropic_api_key: "test-key"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Set environment variables to override nested YAML values
	os.Setenv("AGENT_CLI", "goose")
	os.Setenv("AGENT_MODEL", "opus")
	os.Setenv("AGENT_TIMEOUT", "999")
	os.Setenv("AGENT_SYSTEM_PROMPT_FILE", "/env/prompt.md")
	os.Setenv("AGENT_ALLOWED_TOOLS", "restricted")
	os.Setenv("ADDITIONAL_AGENT_PROMPT", "Env prompt")
	os.Setenv("EXECUTION_DEFAULTS_NAMESPACE", "env-namespace")
	os.Setenv("EXECUTION_DEFAULTS_RUNNER_IMAGE", "env-image:latest")
	os.Setenv("EXECUTION_DEFAULTS_IMAGE_PULL_POLICY", "Never")
	os.Setenv("EXECUTION_DEFAULTS_TIMEOUT", "888")
	os.Setenv("EXECUTION_DEFAULTS_MEMORY_LIMIT", "8Gi")
	os.Setenv("EXECUTION_DEFAULTS_CPU_LIMIT", "4")
	os.Setenv("EXECUTION_DEFAULTS_CLEANUP_TTL", "9999")

	defer func() {
		os.Unsetenv("AGENT_CLI")
		os.Unsetenv("AGENT_MODEL")
		os.Unsetenv("AGENT_TIMEOUT")
		os.Unsetenv("AGENT_SYSTEM_PROMPT_FILE")
		os.Unsetenv("AGENT_ALLOWED_TOOLS")
		os.Unsetenv("ADDITIONAL_AGENT_PROMPT")
		os.Unsetenv("EXECUTION_DEFAULTS_NAMESPACE")
		os.Unsetenv("EXECUTION_DEFAULTS_RUNNER_IMAGE")
		os.Unsetenv("EXECUTION_DEFAULTS_IMAGE_PULL_POLICY")
		os.Unsetenv("EXECUTION_DEFAULTS_TIMEOUT")
		os.Unsetenv("EXECUTION_DEFAULTS_MEMORY_LIMIT")
		os.Unsetenv("EXECUTION_DEFAULTS_CPU_LIMIT")
		os.Unsetenv("EXECUTION_DEFAULTS_CLEANUP_TTL")
	}()

	cfg, err := LoadWithConfigFile(configPath)
	if err != nil {
		t.Fatalf("LoadWithConfigFile() failed: %v", err)
	}

	// Verify Agent config overrides
	if cfg.Agent.CLI != "goose" {
		t.Errorf("Agent.CLI = %q, want %q (env var override)", cfg.Agent.CLI, "goose")
	}
	if cfg.Agent.Model != "opus" {
		t.Errorf("Agent.Model = %q, want %q (env var override)", cfg.Agent.Model, "opus")
	}
	if cfg.Agent.Timeout != 999 {
		t.Errorf("Agent.Timeout = %d, want %d (env var override)", cfg.Agent.Timeout, 999)
	}
	if cfg.Agent.SystemPromptFile != "/env/prompt.md" {
		t.Errorf("Agent.SystemPromptFile = %q, want %q (env var override)", cfg.Agent.SystemPromptFile, "/env/prompt.md")
	}
	if cfg.Agent.AllowedTools != "restricted" {
		t.Errorf("Agent.AllowedTools = %q, want %q (env var override)", cfg.Agent.AllowedTools, "restricted")
	}
	if cfg.Agent.AdditionalPrompt != "Env prompt" {
		t.Errorf("Agent.AdditionalPrompt = %q, want %q (env var override)", cfg.Agent.AdditionalPrompt, "Env prompt")
	}

	// Verify ExecutionDefaults config overrides
	if cfg.ExecutionDefaults.Namespace != "env-namespace" {
		t.Errorf("ExecutionDefaults.Namespace = %q, want %q (env var override)", cfg.ExecutionDefaults.Namespace, "env-namespace")
	}
	if cfg.ExecutionDefaults.RunnerImage != "env-image:latest" {
		t.Errorf("ExecutionDefaults.RunnerImage = %q, want %q (env var override)", cfg.ExecutionDefaults.RunnerImage, "env-image:latest")
	}
	if cfg.ExecutionDefaults.ImagePullPolicy != "Never" {
		t.Errorf("ExecutionDefaults.ImagePullPolicy = %q, want %q (env var override)", cfg.ExecutionDefaults.ImagePullPolicy, "Never")
	}
	if cfg.ExecutionDefaults.Timeout != 888 {
		t.Errorf("ExecutionDefaults.Timeout = %d, want %d (env var override)", cfg.ExecutionDefaults.Timeout, 888)
	}
	if cfg.ExecutionDefaults.MemoryLimit != "8Gi" {
		t.Errorf("ExecutionDefaults.MemoryLimit = %q, want %q (env var override)", cfg.ExecutionDefaults.MemoryLimit, "8Gi")
	}
	if cfg.ExecutionDefaults.CPULimit != "4" {
		t.Errorf("ExecutionDefaults.CPULimit = %q, want %q (env var override)", cfg.ExecutionDefaults.CPULimit, "4")
	}
	if cfg.ExecutionDefaults.CleanupTTL != 9999 {
		t.Errorf("ExecutionDefaults.CleanupTTL = %d, want %d (env var override)", cfg.ExecutionDefaults.CleanupTTL, 9999)
	}
}
