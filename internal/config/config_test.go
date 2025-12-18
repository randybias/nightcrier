package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// resetViper clears viper state between tests
func resetViper() {
	viper.Reset()
}

func TestLoadWithDefaults(t *testing.T) {
	resetViper()

	// Set required env var
	os.Setenv("K8S_CLUSTER_MCP_ENDPOINT", "http://localhost:8080/mcp")
	defer os.Unsetenv("K8S_CLUSTER_MCP_ENDPOINT")

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
`,
			wantErr: true,
		},
		{
			name: "global_queue_size < 1",
			config: `
mcp_endpoint: "http://localhost:8080/mcp"
global_queue_size: 0
`,
			wantErr: true,
		},
		{
			name: "cluster_queue_size < 1",
			config: `
mcp_endpoint: "http://localhost:8080/mcp"
cluster_queue_size: 0
`,
			wantErr: true,
		},
		{
			name: "dedup_window_seconds negative",
			config: `
mcp_endpoint: "http://localhost:8080/mcp"
dedup_window_seconds: -1
`,
			wantErr: true,
		},
		{
			name: "agent_timeout < 1",
			config: `
mcp_endpoint: "http://localhost:8080/mcp"
agent_timeout: 0
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
`,
			wantErr: true,
		},
		{
			name: "max backoff < initial backoff",
			config: `
mcp_endpoint: "http://localhost:8080/mcp"
sse_reconnect_initial_backoff: 10
sse_reconnect_max_backoff: 5
`,
			wantErr: true,
		},
		{
			name: "read timeout < 1",
			config: `
mcp_endpoint: "http://localhost:8080/mcp"
sse_read_timeout: 0
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

	// Set required env var
	os.Setenv("K8S_CLUSTER_MCP_ENDPOINT", "http://localhost:8080/mcp")
	defer os.Unsetenv("K8S_CLUSTER_MCP_ENDPOINT")

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
`,
			enabled: false,
		},
		{
			name: "enabled with connection string",
			config: `
mcp_endpoint: "http://localhost:8080/mcp"
azure_storage_connection_string: "AccountName=test;AccountKey=key123"
azure_storage_container: "incidents"
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
