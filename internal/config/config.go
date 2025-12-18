package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Config holds the application configuration.
type Config struct {
	// MCP Connection
	MCPEndpoint string `mapstructure:"mcp_endpoint"`

	// Workspace
	WorkspaceRoot string `mapstructure:"workspace_root"`

	// Logging
	LogLevel string `mapstructure:"log_level"`

	// Slack Integration
	SlackWebhookURL string `mapstructure:"slack_webhook_url"`

	// Agent Configuration
	AgentScriptPath       string `mapstructure:"agent_script_path"`
	AgentSystemPromptFile string `mapstructure:"agent_system_prompt_file"`
	AgentAllowedTools     string `mapstructure:"agent_allowed_tools"`
	AgentModel            string `mapstructure:"agent_model"`
	AgentTimeout          int    `mapstructure:"agent_timeout"` // seconds

	// Event Processing (Phase 1 additions)
	SeverityThreshold   string `mapstructure:"severity_threshold"`
	MaxConcurrentAgents int    `mapstructure:"max_concurrent_agents"`
	GlobalQueueSize     int    `mapstructure:"global_queue_size"`
	ClusterQueueSize    int    `mapstructure:"cluster_queue_size"`
	DedupWindowSeconds  int    `mapstructure:"dedup_window_seconds"`
	QueueOverflowPolicy string `mapstructure:"queue_overflow_policy"`
	ShutdownTimeout     int    `mapstructure:"shutdown_timeout"` // seconds

	// SSE/MCP Reconnection
	SSEReconnectInitialBackoff int `mapstructure:"sse_reconnect_initial_backoff"` // seconds
	SSEReconnectMaxBackoff     int `mapstructure:"sse_reconnect_max_backoff"`     // seconds
	SSEReadTimeout             int `mapstructure:"sse_read_timeout"`              // seconds

	// Azure Storage Configuration (optional)
	AzureStorageConnectionString string `mapstructure:"azure_storage_connection_string"`
	AzureStorageAccount          string `mapstructure:"azure_storage_account"`
	AzureStorageKey              string `mapstructure:"azure_storage_key"`
	AzureStorageContainer        string `mapstructure:"azure_storage_container"`
	AzureSASExpiry               string `mapstructure:"azure_sas_expiry"`
}

// setDefaults configures default values for all configuration options.
func setDefaults() {
	// Workspace defaults
	viper.SetDefault("workspace_root", "./incidents")

	// Logging defaults
	viper.SetDefault("log_level", "info")

	// Agent defaults
	viper.SetDefault("agent_script_path", "./agent-container/run-agent.sh")
	viper.SetDefault("agent_system_prompt_file", "./configs/triage-system-prompt.md")
	viper.SetDefault("agent_allowed_tools", "Read,Write,Grep,Glob,Bash,Skill")
	viper.SetDefault("agent_model", "sonnet")
	viper.SetDefault("agent_timeout", 300)

	// Event processing defaults (from design.md)
	viper.SetDefault("severity_threshold", "ERROR")
	viper.SetDefault("max_concurrent_agents", 5)
	viper.SetDefault("global_queue_size", 100)
	viper.SetDefault("cluster_queue_size", 10)
	viper.SetDefault("dedup_window_seconds", 300)
	viper.SetDefault("queue_overflow_policy", "drop")
	viper.SetDefault("shutdown_timeout", 30)

	// SSE/MCP reconnection defaults
	viper.SetDefault("sse_reconnect_initial_backoff", 1)
	viper.SetDefault("sse_reconnect_max_backoff", 60)
	viper.SetDefault("sse_read_timeout", 120)

	// Azure defaults
	viper.SetDefault("azure_sas_expiry", "168h")
}

// bindEnvVars binds environment variables to viper keys.
// Environment variables use uppercase with underscores (e.g., K8S_CLUSTER_MCP_ENDPOINT).
func bindEnvVars() {
	// Map config keys to environment variable names
	envBindings := map[string]string{
		"mcp_endpoint":                    "K8S_CLUSTER_MCP_ENDPOINT",
		"workspace_root":                  "WORKSPACE_ROOT",
		"log_level":                       "LOG_LEVEL",
		"slack_webhook_url":               "SLACK_WEBHOOK_URL",
		"agent_script_path":               "AGENT_SCRIPT_PATH",
		"agent_system_prompt_file":        "AGENT_SYSTEM_PROMPT_FILE",
		"agent_allowed_tools":             "AGENT_ALLOWED_TOOLS",
		"agent_model":                     "AGENT_MODEL",
		"agent_timeout":                   "AGENT_TIMEOUT",
		"severity_threshold":              "SEVERITY_THRESHOLD",
		"max_concurrent_agents":           "MAX_CONCURRENT_AGENTS",
		"global_queue_size":               "GLOBAL_QUEUE_SIZE",
		"cluster_queue_size":              "CLUSTER_QUEUE_SIZE",
		"dedup_window_seconds":            "DEDUP_WINDOW_SECONDS",
		"queue_overflow_policy":           "QUEUE_OVERFLOW_POLICY",
		"shutdown_timeout":                "SHUTDOWN_TIMEOUT_SECONDS",
		"sse_reconnect_initial_backoff":   "SSE_RECONNECT_INITIAL_BACKOFF",
		"sse_reconnect_max_backoff":       "SSE_RECONNECT_MAX_BACKOFF",
		"sse_read_timeout":                "SSE_READ_TIMEOUT_SECONDS",
		"azure_storage_connection_string": "AZURE_STORAGE_CONNECTION_STRING",
		"azure_storage_account":           "AZURE_STORAGE_ACCOUNT",
		"azure_storage_key":               "AZURE_STORAGE_KEY",
		"azure_storage_container":         "AZURE_STORAGE_CONTAINER",
		"azure_sas_expiry":                "AZURE_SAS_EXPIRY",
	}

	for key, envVar := range envBindings {
		_ = viper.BindEnv(key, envVar)
	}
}

// BindFlags binds cobra/pflag flags to viper configuration.
// This should be called after flag definitions but before config loading.
func BindFlags(flags *pflag.FlagSet) {
	// Bind flags that match config keys
	flagBindings := map[string]string{
		"mcp-endpoint":          "mcp_endpoint",
		"workspace-root":        "workspace_root",
		"log-level":             "log_level",
		"config":                "config_file",
		"agent-timeout":         "agent_timeout",
		"severity-threshold":    "severity_threshold",
		"max-concurrent-agents": "max_concurrent_agents",
		"shutdown-timeout":      "shutdown_timeout",
	}

	for flagName, configKey := range flagBindings {
		if flag := flags.Lookup(flagName); flag != nil {
			_ = viper.BindPFlag(configKey, flag)
		}
	}
}

// Load creates a Config by loading values with the following precedence:
// 1. Command-line flags (highest priority)
// 2. Environment variables
// 3. Configuration file
// 4. Default values (lowest priority)
func Load() (*Config, error) {
	return LoadWithConfigFile("")
}

// LoadWithConfigFile creates a Config, optionally loading from a specific config file.
// If configFile is empty, it searches for config.yaml in standard locations.
func LoadWithConfigFile(configFile string) (*Config, error) {
	// Set defaults first (lowest priority)
	setDefaults()

	// Bind environment variables (overrides defaults)
	bindEnvVars()

	// Load config file if specified or found (overrides env vars but under flags)
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		// Search for config file in standard locations
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")           // Current directory
		viper.AddConfigPath("./configs")   // configs subdirectory
		viper.AddConfigPath("/etc/runner") // System-wide config
	}

	// Read config file (ignore "not found" errors - file is optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Only return error if it's not a "file not found" error
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Unmarshal into Config struct
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate checks the configuration for required fields and valid values.
func (c *Config) Validate() error {
	// Required field
	if c.MCPEndpoint == "" {
		return fmt.Errorf("mcp_endpoint (K8S_CLUSTER_MCP_ENDPOINT) is required")
	}

	// Validate severity threshold
	validSeverities := map[string]bool{
		"DEBUG": true, "INFO": true, "WARNING": true, "ERROR": true, "CRITICAL": true,
	}
	if !validSeverities[strings.ToUpper(c.SeverityThreshold)] {
		return fmt.Errorf("invalid severity_threshold '%s': must be one of DEBUG, INFO, WARNING, ERROR, CRITICAL", c.SeverityThreshold)
	}

	// Validate numeric ranges
	if c.MaxConcurrentAgents < 1 {
		return fmt.Errorf("max_concurrent_agents must be >= 1, got %d", c.MaxConcurrentAgents)
	}
	if c.GlobalQueueSize < 1 {
		return fmt.Errorf("global_queue_size must be >= 1, got %d", c.GlobalQueueSize)
	}
	if c.ClusterQueueSize < 1 {
		return fmt.Errorf("cluster_queue_size must be >= 1, got %d", c.ClusterQueueSize)
	}
	if c.DedupWindowSeconds < 0 {
		return fmt.Errorf("dedup_window_seconds must be >= 0, got %d", c.DedupWindowSeconds)
	}
	if c.AgentTimeout < 1 {
		return fmt.Errorf("agent_timeout must be >= 1, got %d", c.AgentTimeout)
	}
	if c.ShutdownTimeout < 1 {
		return fmt.Errorf("shutdown_timeout must be >= 1, got %d", c.ShutdownTimeout)
	}

	// Validate queue overflow policy
	validPolicies := map[string]bool{"drop": true, "reject": true}
	if !validPolicies[strings.ToLower(c.QueueOverflowPolicy)] {
		return fmt.Errorf("invalid queue_overflow_policy '%s': must be 'drop' or 'reject'", c.QueueOverflowPolicy)
	}

	// Validate SSE reconnection settings
	if c.SSEReconnectInitialBackoff < 1 {
		return fmt.Errorf("sse_reconnect_initial_backoff must be >= 1, got %d", c.SSEReconnectInitialBackoff)
	}
	if c.SSEReconnectMaxBackoff < c.SSEReconnectInitialBackoff {
		return fmt.Errorf("sse_reconnect_max_backoff (%d) must be >= sse_reconnect_initial_backoff (%d)",
			c.SSEReconnectMaxBackoff, c.SSEReconnectInitialBackoff)
	}
	if c.SSEReadTimeout < 1 {
		return fmt.Errorf("sse_read_timeout must be >= 1, got %d", c.SSEReadTimeout)
	}

	// Validate Azure configuration if enabled
	if err := c.ValidateAzureConfig(); err != nil {
		return err
	}

	return nil
}

// GetConfigFile returns the config file that was used, if any.
func GetConfigFile() string {
	return viper.ConfigFileUsed()
}

// IsAzureStorageEnabled detects if Azure storage is configured.
func (c *Config) IsAzureStorageEnabled() bool {
	return c.AzureStorageAccount != "" || c.AzureStorageConnectionString != ""
}

// GetWorkspaceRoot returns the configured workspace root directory.
func (c *Config) GetWorkspaceRoot() string {
	return c.WorkspaceRoot
}

// GetAzureConnectionString returns the Azure connection string.
func (c *Config) GetAzureConnectionString() string {
	return c.AzureStorageConnectionString
}

// GetAzureAccount returns the Azure storage account name.
func (c *Config) GetAzureAccount() string {
	return c.AzureStorageAccount
}

// GetAzureKey returns the Azure storage account access key.
func (c *Config) GetAzureKey() string {
	return c.AzureStorageKey
}

// GetAzureContainer returns the Azure storage container name.
func (c *Config) GetAzureContainer() string {
	return c.AzureStorageContainer
}

// GetAzureSASExpiry returns the SAS token expiration duration.
func (c *Config) GetAzureSASExpiry() time.Duration {
	duration, err := time.ParseDuration(c.AzureSASExpiry)
	if err != nil {
		return 168 * time.Hour // Default 7 days
	}
	return duration
}

// ValidateAzureConfig validates Azure storage configuration if enabled.
func (c *Config) ValidateAzureConfig() error {
	if !c.IsAzureStorageEnabled() {
		return nil
	}

	if c.AzureStorageContainer == "" {
		return fmt.Errorf("azure_storage_container is required when Azure storage is enabled")
	}

	hasConnectionString := c.AzureStorageConnectionString != ""
	hasAccountAndKey := c.AzureStorageAccount != "" && c.AzureStorageKey != ""

	if !hasConnectionString && !hasAccountAndKey {
		return fmt.Errorf("Azure storage requires either azure_storage_connection_string or both azure_storage_account and azure_storage_key")
	}

	if hasConnectionString {
		if err := validateConnectionString(c.AzureStorageConnectionString); err != nil {
			return fmt.Errorf("invalid azure_storage_connection_string: %w", err)
		}
	}

	if c.AzureSASExpiry != "" {
		if _, err := time.ParseDuration(c.AzureSASExpiry); err != nil {
			return fmt.Errorf("invalid azure_sas_expiry duration '%s': %w", c.AzureSASExpiry, err)
		}
	}

	return nil
}

// validateConnectionString performs basic validation on Azure connection string format.
func validateConnectionString(connStr string) error {
	if connStr == "" {
		return fmt.Errorf("connection string is empty")
	}

	parts := strings.Split(connStr, ";")
	if len(parts) < 2 {
		return fmt.Errorf("connection string must contain at least 2 key-value pairs")
	}

	hasAccountName := false
	hasAuth := false

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		kvPair := strings.SplitN(part, "=", 2)
		if len(kvPair) != 2 {
			return fmt.Errorf("invalid key-value pair in connection string: %s", part)
		}

		key := strings.TrimSpace(kvPair[0])
		switch key {
		case "AccountName":
			hasAccountName = true
		case "AccountKey", "SharedAccessSignature":
			hasAuth = true
		}
	}

	if !hasAccountName {
		return fmt.Errorf("connection string must contain AccountName")
	}

	if !hasAuth {
		return fmt.Errorf("connection string must contain either AccountKey or SharedAccessSignature")
	}

	return nil
}
