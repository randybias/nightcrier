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

func TestLoadWithDefaults(t *testing.T) {
	resetViper()

	// Set required env vars
	os.Setenv("K8S_CLUSTER_MCP_ENDPOINT", "http://localhost:8080/mcp")
	defer os.Unsetenv("K8S_CLUSTER_MCP_ENDPOINT")
	defer setTestAPIKey(t)()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Check defaults are applied
	if cfg.WorkspaceRoot != "./incidents" {
		t.Errorf("WorkspaceRoot = %q, want %q", cfg.WorkspaceRoot, "./incidents")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
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

	// Set env vars
	os.Setenv("K8S_CLUSTER_MCP_ENDPOINT", "http://mcp-server:8080/mcp")
	os.Setenv("WORKSPACE_ROOT", "/var/incidents")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("AGENT_MODEL", "opus")
	os.Setenv("AGENT_TIMEOUT", "600")
	os.Setenv("SEVERITY_THRESHOLD", "WARNING")
	os.Setenv("MAX_CONCURRENT_AGENTS", "10")
	defer setTestAPIKey(t)()

	defer func() {
		os.Unsetenv("K8S_CLUSTER_MCP_ENDPOINT")
		os.Unsetenv("WORKSPACE_ROOT")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("AGENT_MODEL")
		os.Unsetenv("AGENT_TIMEOUT")
		os.Unsetenv("SEVERITY_THRESHOLD")
		os.Unsetenv("MAX_CONCURRENT_AGENTS")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.MCPEndpoint != "http://mcp-server:8080/mcp" {
		t.Errorf("MCPEndpoint = %q, want %q", cfg.MCPEndpoint, "http://mcp-server:8080/mcp")
	}
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
mcp_endpoint: "http://config-file-server:8080/mcp"
workspace_root: "/config/incidents"
log_level: "warn"
agent_model: "haiku"
agent_timeout: 120
severity_threshold: "CRITICAL"
max_concurrent_agents: 3
global_queue_size: 50
cluster_queue_size: 5
dedup_window_seconds: 600
queue_overflow_policy: "reject"
anthropic_api_key: "test-key"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadWithConfigFile(configPath)
	if err != nil {
		t.Fatalf("LoadWithConfigFile() failed: %v", err)
	}

	if cfg.MCPEndpoint != "http://config-file-server:8080/mcp" {
		t.Errorf("MCPEndpoint = %q, want %q", cfg.MCPEndpoint, "http://config-file-server:8080/mcp")
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

	// Create temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
mcp_endpoint: "http://config-file-server:8080/mcp"
workspace_root: "/config/incidents"
log_level: "warn"
agent_timeout: 120
anthropic_api_key: "test-key"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Set env vars that should override config file
	os.Setenv("K8S_CLUSTER_MCP_ENDPOINT", "http://env-override:8080/mcp")
	os.Setenv("LOG_LEVEL", "error")
	defer func() {
		os.Unsetenv("K8S_CLUSTER_MCP_ENDPOINT")
		os.Unsetenv("LOG_LEVEL")
	}()

	cfg, err := LoadWithConfigFile(configPath)
	if err != nil {
		t.Fatalf("LoadWithConfigFile() failed: %v", err)
	}

	// Env vars should override config file
	if cfg.MCPEndpoint != "http://env-override:8080/mcp" {
		t.Errorf("MCPEndpoint = %q, want %q (env var should override)", cfg.MCPEndpoint, "http://env-override:8080/mcp")
	}
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

func TestValidation_MissingMCPEndpoint(t *testing.T) {
	resetViper()

	// Don't set required K8S_CLUSTER_MCP_ENDPOINT
	os.Unsetenv("K8S_CLUSTER_MCP_ENDPOINT")

	_, err := Load()
	if err == nil {
		t.Error("Load() should fail when MCP endpoint is missing")
	}
}

func TestValidation_InvalidSeverityThreshold(t *testing.T) {
	resetViper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
mcp_endpoint: "http://localhost:8080/mcp"
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

	tests := []struct {
		name    string
		config  string
		wantErr bool
	}{
		{
			name: "max_concurrent_agents < 1",
			config: `
mcp_endpoint: "http://localhost:8080/mcp"
max_concurrent_agents: 0
anthropic_api_key: "test-key"
`,
			wantErr: true,
		},
		{
			name: "global_queue_size < 1",
			config: `
mcp_endpoint: "http://localhost:8080/mcp"
global_queue_size: 0
anthropic_api_key: "test-key"
`,
			wantErr: true,
		},
		{
			name: "cluster_queue_size < 1",
			config: `
mcp_endpoint: "http://localhost:8080/mcp"
cluster_queue_size: 0
anthropic_api_key: "test-key"
`,
			wantErr: true,
		},
		{
			name: "dedup_window_seconds negative",
			config: `
mcp_endpoint: "http://localhost:8080/mcp"
dedup_window_seconds: -1
anthropic_api_key: "test-key"
`,
			wantErr: true,
		},
		{
			name: "agent_timeout < 1",
			config: `
mcp_endpoint: "http://localhost:8080/mcp"
agent_timeout: 0
anthropic_api_key: "test-key"
`,
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
mcp_endpoint: "http://localhost:8080/mcp"
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

	tests := []struct {
		name    string
		config  string
		wantErr bool
	}{
		{
			name: "initial backoff < 1",
			config: `
mcp_endpoint: "http://localhost:8080/mcp"
sse_reconnect_initial_backoff: 0
anthropic_api_key: "test-key"
`,
			wantErr: true,
		},
		{
			name: "max backoff < initial backoff",
			config: `
mcp_endpoint: "http://localhost:8080/mcp"
sse_reconnect_initial_backoff: 10
sse_reconnect_max_backoff: 5
anthropic_api_key: "test-key"
`,
			wantErr: true,
		},
		{
			name: "read timeout < 1",
			config: `
mcp_endpoint: "http://localhost:8080/mcp"
sse_read_timeout: 0
anthropic_api_key: "test-key"
`,
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
mcp_endpoint: "http://localhost:8080/mcp"
severity_threshold: "` + severity + `"
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
			configContent := `
mcp_endpoint: "http://localhost:8080/mcp"
queue_overflow_policy: "` + policy + `"
anthropic_api_key: "test-key"
`
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

	// Set required env vars
	os.Setenv("K8S_CLUSTER_MCP_ENDPOINT", "http://localhost:8080/mcp")
	defer os.Unsetenv("K8S_CLUSTER_MCP_ENDPOINT")
	defer setTestAPIKey(t)()

	// Should not fail even if no config file exists
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() should not fail when config file is not found: %v", err)
	}

	if cfg.MCPEndpoint != "http://localhost:8080/mcp" {
		t.Errorf("MCPEndpoint = %q, want %q", cfg.MCPEndpoint, "http://localhost:8080/mcp")
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
			name: "disabled when no Azure config",
			config: `
mcp_endpoint: "http://localhost:8080/mcp"
anthropic_api_key: "test-key"
`,
			enabled: false,
		},
		{
			name: "enabled with connection string",
			config: `
mcp_endpoint: "http://localhost:8080/mcp"
azure_storage_connection_string: "AccountName=test;AccountKey=key123"
azure_storage_container: "incidents"
anthropic_api_key: "test-key"
`,
			enabled: true,
		},
		{
			name: "enabled with account and key",
			config: `
mcp_endpoint: "http://localhost:8080/mcp"
azure_storage_account: "teststorage"
azure_storage_key: "key123"
azure_storage_container: "incidents"
anthropic_api_key: "test-key"
`,
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
	configContent := `
mcp_endpoint: "http://localhost:8080/mcp"
`
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
			name: "anthropic key",
			config: `
mcp_endpoint: "http://localhost:8080/mcp"
anthropic_api_key: "sk-ant-test"
`,
		},
		{
			name: "openai key",
			config: `
mcp_endpoint: "http://localhost:8080/mcp"
openai_api_key: "sk-test"
`,
		},
		{
			name: "gemini key",
			config: `
mcp_endpoint: "http://localhost:8080/mcp"
gemini_api_key: "test-key"
`,
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

	tests := []struct {
		name    string
		config  string
		wantCfg func(*Config) bool
	}{
		{
			name: "default values",
			config: `
mcp_endpoint: "http://localhost:8080/mcp"
anthropic_api_key: "test-key"
`,
			wantCfg: func(cfg *Config) bool {
				return cfg.NotifyOnAgentFailure == true &&
					cfg.FailureThresholdForAlert == 3 &&
					cfg.UploadFailedInvestigations == false
			},
		},
		{
			name: "custom values",
			config: `
mcp_endpoint: "http://localhost:8080/mcp"
anthropic_api_key: "test-key"
notify_on_agent_failure: false
failure_threshold_for_alert: 5
upload_failed_investigations: true
`,
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

	// Set env vars
	os.Setenv("K8S_CLUSTER_MCP_ENDPOINT", "http://localhost:8080/mcp")
	os.Setenv("NOTIFY_ON_AGENT_FAILURE", "false")
	os.Setenv("FAILURE_THRESHOLD_FOR_ALERT", "10")
	os.Setenv("UPLOAD_FAILED_INVESTIGATIONS", "true")
	defer setTestAPIKey(t)()

	defer func() {
		os.Unsetenv("K8S_CLUSTER_MCP_ENDPOINT")
		os.Unsetenv("NOTIFY_ON_AGENT_FAILURE")
		os.Unsetenv("FAILURE_THRESHOLD_FOR_ALERT")
		os.Unsetenv("UPLOAD_FAILED_INVESTIGATIONS")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
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
			configContent := fmt.Sprintf(`
mcp_endpoint: "http://localhost:8080/mcp"
failure_threshold_for_alert: %d
anthropic_api_key: "test-key"
`, tt.threshold)
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
	configContent := `
mcp_endpoint: "http://localhost:8080/mcp"
workspace_root: "/tmp/incidents"
log_level: "debug"
agent_timeout: 600
severity_threshold: "WARNING"
max_concurrent_agents: 10

# Circuit breaker settings
notify_on_agent_failure: false
failure_threshold_for_alert: 5
upload_failed_investigations: true

# Azure storage
azure_storage_account: "teststorage"
azure_storage_key: "testkey"
azure_storage_container: "incidents"

anthropic_api_key: "test-key"
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
	if cfg.MCPEndpoint != "http://localhost:8080/mcp" {
		t.Errorf("MCPEndpoint = %q, want %q", cfg.MCPEndpoint, "http://localhost:8080/mcp")
	}
	if cfg.MaxConcurrentAgents != 10 {
		t.Errorf("MaxConcurrentAgents = %d, want 10", cfg.MaxConcurrentAgents)
	}
	if cfg.AzureStorageAccount != "teststorage" {
		t.Errorf("AzureStorageAccount = %q, want %q", cfg.AzureStorageAccount, "teststorage")
	}
}
