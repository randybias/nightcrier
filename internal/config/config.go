package config

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/rbias/nightcrier/internal/cluster"
)

// Config holds the application configuration.
type Config struct {
	// Cluster Configuration
	Clusters      []cluster.ClusterConfig `mapstructure:"clusters"`
	SubscribeMode string                  `mapstructure:"subscribe_mode"` // events, faults

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
	AgentCLI              string `mapstructure:"agent_cli"`     // claude, codex, goose, gemini
	AgentImage            string `mapstructure:"agent_image"`              // Docker image for agent container
	AgentVerbose          bool   `mapstructure:"agent_verbose"`           // Enable verbose agent output
	AdditionalAgentPrompt string `mapstructure:"additional_agent_prompt"` // Optional additional context for agent (cluster-specific SLOs, escalation info)

	// LLM API Keys (optional - can also be set via environment)
	AnthropicAPIKey string `mapstructure:"anthropic_api_key"`
	OpenAIAPIKey    string `mapstructure:"openai_api_key"`
	GeminiAPIKey    string `mapstructure:"gemini_api_key"`

	// Kubernetes Configuration
	KubeconfigPath    string `mapstructure:"kubeconfig_path"`
	KubernetesContext string `mapstructure:"kubernetes_context"`

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

	// Azure Storage Configuration (optional - used when cloud storage is enabled)
	AzureStorageConnectionString string `mapstructure:"azure_storage_connection_string"`
	AzureStorageAccount          string `mapstructure:"azure_storage_account"`
	AzureStorageKey              string `mapstructure:"azure_storage_key"`
	AzureStorageContainer        string `mapstructure:"azure_storage_container"`
	AzureSASExpiry               string `mapstructure:"azure_sas_expiry"`

	// Circuit Breaker and Notification Configuration (Phase 2)
	NotifyOnAgentFailure        bool `mapstructure:"notify_on_agent_failure"`
	FailureThresholdForAlert    int  `mapstructure:"failure_threshold_for_alert"`
	UploadFailedInvestigations  bool `mapstructure:"upload_failed_investigations"`

	// State Storage Configuration (SQL Support)
	// Configures where incident state is persisted. Supports filesystem (backward compatible),
	// SQLite (embedded), and PostgreSQL (centralized). Default: filesystem
	StateStorage StateStorage `mapstructure:"state_storage"`

	// Skills Configuration
	// Configures where downloaded skills (like k8s4agents) are cached and
	// whether to preload triage scripts
	Skills SkillsConfig `mapstructure:"skills"`
}

// StateStorage configures persistent state storage for incidents, agent executions, and triage reports.
// Supports three storage backends:
//   - filesystem: Legacy filesystem-based storage (default for backward compatibility)
//   - sqlite: Embedded SQLite database (single-node, file-based)
//   - postgres: PostgreSQL database (multi-node, centralized)
type StateStorage struct {
	// Type specifies the storage backend: "filesystem", "sqlite", or "postgres"
	// Default: "filesystem" (maintains backward compatibility)
	// Environment variable: STATE_STORAGE_TYPE
	Type string `mapstructure:"type"`

	// SQLitePath specifies the path to the SQLite database file
	// Only used when Type is "sqlite"
	// Default: "{workspace_root}/nightcrier.db"
	// Environment variable: STATE_STORAGE_SQLITE_PATH
	SQLitePath string `mapstructure:"sqlite_path"`

	// PostgresConnectionString is a complete PostgreSQL connection string
	// Format: "postgres://user:password@host:port/dbname?sslmode=disable"
	// Only used when Type is "postgres"
	// Takes precedence over individual Postgres* fields if provided
	// Environment variable: STATE_STORAGE_POSTGRES_CONNECTION_STRING
	PostgresConnectionString string `mapstructure:"postgres_connection_string"`

	// PostgresHost is the PostgreSQL server hostname
	// Only used when Type is "postgres" and PostgresConnectionString is not provided
	// Environment variable: STATE_STORAGE_POSTGRES_HOST
	PostgresHost string `mapstructure:"postgres_host"`

	// PostgresPort is the PostgreSQL server port
	// Default: 5432
	// Only used when Type is "postgres" and PostgresConnectionString is not provided
	// Environment variable: STATE_STORAGE_POSTGRES_PORT
	PostgresPort int `mapstructure:"postgres_port"`

	// PostgresDatabase is the PostgreSQL database name
	// Only used when Type is "postgres" and PostgresConnectionString is not provided
	// Environment variable: STATE_STORAGE_POSTGRES_DATABASE
	PostgresDatabase string `mapstructure:"postgres_database"`

	// PostgresUser is the PostgreSQL username
	// Only used when Type is "postgres" and PostgresConnectionString is not provided
	// Environment variable: STATE_STORAGE_POSTGRES_USER
	PostgresUser string `mapstructure:"postgres_user"`

	// PostgresPassword is the PostgreSQL password
	// Only used when Type is "postgres" and PostgresConnectionString is not provided
	// Environment variable: STATE_STORAGE_POSTGRES_PASSWORD
	PostgresPassword string `mapstructure:"postgres_password"`

	// MigrationsPath is the path to the directory containing SQL migration files
	// Default: "./migrations"
	// Environment variable: STATE_STORAGE_MIGRATIONS_PATH
	MigrationsPath string `mapstructure:"migrations_path"`
}

// SkillsConfig configures the skills subsystem for the agent.
// Skills are external tools and utilities that extend agent capabilities,
// such as downloaded triage scripts (k8s4agents).
type SkillsConfig struct {
	// CacheDir is the directory where downloaded skills are cached
	// Default: "{workspace_root}/agent-home/skills"
	// Environment variable: SKILLS_CACHE_DIR
	CacheDir string `mapstructure:"cache_dir"`

	// DisableTriagePreload controls whether triage scripts should be preloaded
	// When false (default), the system preloads triage scripts from the cache
	// When true, the agent runs triage scripts itself
	// Default: false
	// Environment variable: SKILLS_DISABLE_TRIAGE_PRELOAD
	DisableTriagePreload bool `mapstructure:"disable_triage_preload"`
}

// bindEnvVars binds environment variables to viper keys.
// Environment variables use uppercase with underscores (e.g., WORKSPACE_ROOT).
func bindEnvVars() {
	// Map config keys to environment variable names
	envBindings := map[string]string{
		"subscribe_mode":                  "SUBSCRIBE_MODE",
		"workspace_root":                  "WORKSPACE_ROOT",
		"log_level":                       "LOG_LEVEL",
		"slack_webhook_url":               "SLACK_WEBHOOK_URL",
		"agent_script_path":               "AGENT_SCRIPT_PATH",
		"agent_system_prompt_file":        "AGENT_SYSTEM_PROMPT_FILE",
		"agent_allowed_tools":             "AGENT_ALLOWED_TOOLS",
		"agent_model":                     "AGENT_MODEL",
		"agent_timeout":                   "AGENT_TIMEOUT",
		"agent_cli":                       "AGENT_CLI",
		"agent_image":                     "AGENT_IMAGE",
		"agent_verbose":                   "AGENT_VERBOSE",
		"additional_agent_prompt":         "ADDITIONAL_AGENT_PROMPT",
		"anthropic_api_key":               "ANTHROPIC_API_KEY",
		"openai_api_key":                  "OPENAI_API_KEY",
		"gemini_api_key":                  "GEMINI_API_KEY",
		"kubeconfig_path":                 "KUBECONFIG_PATH",
		"kubernetes_context":              "KUBERNETES_CONTEXT",
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
		"notify_on_agent_failure":         "NOTIFY_ON_AGENT_FAILURE",
		"failure_threshold_for_alert":     "FAILURE_THRESHOLD_FOR_ALERT",
		"upload_failed_investigations":    "UPLOAD_FAILED_INVESTIGATIONS",
		"state_storage.type":                                "STATE_STORAGE_TYPE",
		"state_storage.sqlite_path":                         "STATE_STORAGE_SQLITE_PATH",
		"state_storage.postgres_connection_string":          "STATE_STORAGE_POSTGRES_CONNECTION_STRING",
		"state_storage.postgres_host":                       "STATE_STORAGE_POSTGRES_HOST",
		"state_storage.postgres_port":                       "STATE_STORAGE_POSTGRES_PORT",
		"state_storage.postgres_database":                   "STATE_STORAGE_POSTGRES_DATABASE",
		"state_storage.postgres_user":                       "STATE_STORAGE_POSTGRES_USER",
		"state_storage.postgres_password":                   "STATE_STORAGE_POSTGRES_PASSWORD",
		"state_storage.migrations_path":                     "STATE_STORAGE_MIGRATIONS_PATH",
		"skills.cache_dir":                                  "SKILLS_CACHE_DIR",
		"skills.disable_triage_preload":                     "SKILLS_DISABLE_TRIAGE_PRELOAD",
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
		"workspace-root":                "workspace_root",
		"log-level":                     "log_level",
		"config":                        "config_file",
		"agent-timeout":                 "agent_timeout",
		"severity-threshold":            "severity_threshold",
		"max-concurrent-agents":         "max_concurrent_agents",
		"shutdown-timeout":              "shutdown_timeout",
		"notify-on-agent-failure":       "notify_on_agent_failure",
		"failure-threshold-for-alert":   "failure_threshold_for_alert",
		"upload-failed-investigations":  "upload_failed_investigations",
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
// 3. Configuration file (lowest priority)
// All required fields must be provided through one of these sources.
func Load() (*Config, error) {
	return LoadWithConfigFile("")
}

// LoadWithConfigFile creates a Config, optionally loading from a specific config file.
// If configFile is empty, it searches for config.yaml in standard locations.
func LoadWithConfigFile(configFile string) (*Config, error) {
	// Bind environment variables
	bindEnvVars()

	// Load config file if specified or found (overrides env vars but under flags)
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		// Search for config file in standard locations
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")               // Current directory
		viper.AddConfigPath("./configs")       // configs subdirectory
		viper.AddConfigPath("/etc/nightcrier") // System-wide config
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
	// Helper function to format missing field errors
	missingFieldError := func(fieldName, envVar string) error {
		return fmt.Errorf("required field %q is missing (environment variable: %s). Please set it via environment variable, config file, or command-line flag. See configs/config.example.yaml for details", fieldName, envVar)
	}

	// Required: Clusters
	if len(c.Clusters) == 0 {
		return fmt.Errorf("at least one cluster must be configured in the 'clusters' array")
	}

	// Validate cluster name uniqueness and individual cluster configs
	clusterNames := make(map[string]bool)
	for i, cluster := range c.Clusters {
		if cluster.Name == "" {
			return fmt.Errorf("cluster[%d]: name is required", i)
		}

		if clusterNames[cluster.Name] {
			return fmt.Errorf("duplicate cluster name: %s", cluster.Name)
		}
		clusterNames[cluster.Name] = true

		// Validate individual cluster config
		if err := cluster.Validate(); err != nil {
			return fmt.Errorf("cluster[%d] (%s): %w", i, cluster.Name, err)
		}
	}

	if c.SubscribeMode == "" {
		return missingFieldError("subscribe_mode", "SUBSCRIBE_MODE")
	}

	// Required: Workspace
	if c.WorkspaceRoot == "" {
		return missingFieldError("workspace_root", "WORKSPACE_ROOT")
	}

	// Required: Agent Configuration
	if c.AgentScriptPath == "" {
		return missingFieldError("agent_script_path", "AGENT_SCRIPT_PATH")
	}

	if c.AgentTimeout == 0 {
		return missingFieldError("agent_timeout", "AGENT_TIMEOUT")
	}

	if c.AgentModel == "" {
		return missingFieldError("agent_model", "AGENT_MODEL")
	}

	if c.AgentCLI == "" {
		return missingFieldError("agent_cli", "AGENT_CLI")
	}

	if c.AgentImage == "" {
		return missingFieldError("agent_image", "AGENT_IMAGE")
	}

	// Note: AdditionalAgentPrompt is optional - system prompt drives investigation

	// Required: Event Processing
	if c.SeverityThreshold == "" {
		return missingFieldError("severity_threshold", "SEVERITY_THRESHOLD")
	}

	if c.MaxConcurrentAgents == 0 {
		return missingFieldError("max_concurrent_agents", "MAX_CONCURRENT_AGENTS")
	}

	if c.GlobalQueueSize == 0 {
		return missingFieldError("global_queue_size", "GLOBAL_QUEUE_SIZE")
	}

	if c.ClusterQueueSize == 0 {
		return missingFieldError("cluster_queue_size", "CLUSTER_QUEUE_SIZE")
	}

	if c.DedupWindowSeconds < 0 {
		return missingFieldError("dedup_window_seconds", "DEDUP_WINDOW_SECONDS")
	}

	if c.QueueOverflowPolicy == "" {
		return missingFieldError("queue_overflow_policy", "QUEUE_OVERFLOW_POLICY")
	}

	if c.ShutdownTimeout == 0 {
		return missingFieldError("shutdown_timeout", "SHUTDOWN_TIMEOUT_SECONDS")
	}

	// Required: SSE/MCP Reconnection
	if c.SSEReconnectInitialBackoff == 0 {
		return missingFieldError("sse_reconnect_initial_backoff", "SSE_RECONNECT_INITIAL_BACKOFF")
	}

	if c.SSEReconnectMaxBackoff == 0 {
		return missingFieldError("sse_reconnect_max_backoff", "SSE_RECONNECT_MAX_BACKOFF")
	}

	if c.SSEReadTimeout == 0 {
		return missingFieldError("sse_read_timeout", "SSE_READ_TIMEOUT_SECONDS")
	}

	// Required: Circuit Breaker
	if c.FailureThresholdForAlert == 0 {
		return missingFieldError("failure_threshold_for_alert", "FAILURE_THRESHOLD_FOR_ALERT")
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
		return fmt.Errorf("max_concurrent_agents must be >= 1, got %d. Set via MAX_CONCURRENT_AGENTS environment variable or config file", c.MaxConcurrentAgents)
	}
	if c.GlobalQueueSize < 1 {
		return fmt.Errorf("global_queue_size must be >= 1, got %d. Set via GLOBAL_QUEUE_SIZE environment variable or config file", c.GlobalQueueSize)
	}
	if c.ClusterQueueSize < 1 {
		return fmt.Errorf("cluster_queue_size must be >= 1, got %d. Set via CLUSTER_QUEUE_SIZE environment variable or config file", c.ClusterQueueSize)
	}
	if c.DedupWindowSeconds < 0 {
		return fmt.Errorf("dedup_window_seconds must be >= 0, got %d. Set via DEDUP_WINDOW_SECONDS environment variable or config file", c.DedupWindowSeconds)
	}
	if c.AgentTimeout < 1 {
		return fmt.Errorf("agent_timeout must be >= 1, got %d. Set via AGENT_TIMEOUT environment variable or config file", c.AgentTimeout)
	}
	if c.ShutdownTimeout < 1 {
		return fmt.Errorf("shutdown_timeout must be >= 1, got %d. Set via SHUTDOWN_TIMEOUT_SECONDS environment variable or config file", c.ShutdownTimeout)
	}

	// Validate queue overflow policy
	validPolicies := map[string]bool{"drop": true, "reject": true}
	if !validPolicies[strings.ToLower(c.QueueOverflowPolicy)] {
		return fmt.Errorf("invalid queue_overflow_policy '%s': must be 'drop' or 'reject'. Set via QUEUE_OVERFLOW_POLICY environment variable or config file", c.QueueOverflowPolicy)
	}

	// Validate SSE reconnection settings
	if c.SSEReconnectInitialBackoff < 1 {
		return fmt.Errorf("sse_reconnect_initial_backoff must be >= 1, got %d. Set via SSE_RECONNECT_INITIAL_BACKOFF environment variable or config file", c.SSEReconnectInitialBackoff)
	}
	if c.SSEReconnectMaxBackoff < c.SSEReconnectInitialBackoff {
		return fmt.Errorf("sse_reconnect_max_backoff (%d) must be >= sse_reconnect_initial_backoff (%d). Set via SSE_RECONNECT_MAX_BACKOFF environment variable or config file",
			c.SSEReconnectMaxBackoff, c.SSEReconnectInitialBackoff)
	}
	if c.SSEReadTimeout < 1 {
		return fmt.Errorf("sse_read_timeout must be >= 1, got %d. Set via SSE_READ_TIMEOUT_SECONDS environment variable or config file", c.SSEReadTimeout)
	}

	// Validate circuit breaker settings
	if c.FailureThresholdForAlert < 1 {
		return fmt.Errorf("failure_threshold_for_alert must be >= 1, got %d. Set via FAILURE_THRESHOLD_FOR_ALERT environment variable or config file", c.FailureThresholdForAlert)
	}

	// Require at least one LLM API key
	if err := c.ValidateLLMAPIKeys(); err != nil {
		return err
	}

	// Validate Azure configuration if enabled
	if err := c.ValidateAzureConfig(); err != nil {
		return err
	}

	// Validate state storage configuration
	if err := c.ValidateStateStorage(); err != nil {
		return err
	}

	return nil
}

// ValidateLLMAPIKeys ensures at least one LLM API key is configured.
// Returns an error if no API keys are found.
func (c *Config) ValidateLLMAPIKeys() error {
	if c.AnthropicAPIKey != "" {
		return nil
	}
	if c.OpenAIAPIKey != "" {
		return nil
	}
	if c.GeminiAPIKey != "" {
		return nil
	}

	return fmt.Errorf("at least one LLM API key is required: set ANTHROPIC_API_KEY, OPENAI_API_KEY, or GEMINI_API_KEY (via environment variable, config file, or command-line)")
}

// GetConfigFile returns the config file that was used, if any.
func GetConfigFile() string {
	return viper.ConfigFileUsed()
}

// IsAzureStorageEnabled detects if Azure storage is configured.
// Returns true if AZURE_STORAGE_ACCOUNT or AZURE_STORAGE_CONNECTION_STRING is set.
func (c *Config) IsAzureStorageEnabled() bool {
	return c.AzureStorageAccount != "" || c.AzureStorageConnectionString != ""
}

// GetWorkspaceRoot returns the configured workspace root directory.
// This method is part of the StorageConfig interface.
func (c *Config) GetWorkspaceRoot() string {
	return c.WorkspaceRoot
}

// GetAzureConnectionString returns the Azure connection string.
// This method is part of the AzureConfig interface.
func (c *Config) GetAzureConnectionString() string {
	return c.AzureStorageConnectionString
}

// GetAzureAccount returns the Azure storage account name.
// This method is part of the AzureConfig interface.
func (c *Config) GetAzureAccount() string {
	return c.AzureStorageAccount
}

// GetAzureKey returns the Azure storage account access key.
// This method is part of the AzureConfig interface.
func (c *Config) GetAzureKey() string {
	return c.AzureStorageKey
}

// GetAzureContainer returns the Azure storage container name.
// This method is part of the AzureConfig interface.
func (c *Config) GetAzureContainer() string {
	return c.AzureStorageContainer
}

// GetAzureSASExpiry returns the SAS token expiration duration.
// This method is part of the AzureConfig interface.
func (c *Config) GetAzureSASExpiry() time.Duration {
	duration, err := time.ParseDuration(c.AzureSASExpiry)
	if err != nil {
		// Fall back to default (7 days) if parsing fails
		return 168 * time.Hour
	}
	return duration
}

// ValidateAzureConfig validates Azure storage configuration if Azure storage is enabled.
// Returns an error if Azure is enabled but required fields are missing or invalid.
func (c *Config) ValidateAzureConfig() error {
	// If Azure storage is not enabled, no validation needed
	if !c.IsAzureStorageEnabled() {
		return nil
	}

	// Validate container is provided (required for Azure storage)
	if c.AzureStorageContainer == "" {
		return fmt.Errorf("AZURE_STORAGE_CONTAINER is required when Azure storage is enabled")
	}

	// Validate authentication: either connection string OR account+key must be provided
	hasConnectionString := c.AzureStorageConnectionString != ""
	hasAccountAndKey := c.AzureStorageAccount != "" && c.AzureStorageKey != ""

	if !hasConnectionString && !hasAccountAndKey {
		return fmt.Errorf("Azure storage requires either AZURE_STORAGE_CONNECTION_STRING or both AZURE_STORAGE_ACCOUNT and AZURE_STORAGE_KEY")
	}

	// If connection string is provided, validate it's parseable
	if hasConnectionString {
		if err := validateConnectionString(c.AzureStorageConnectionString); err != nil {
			return fmt.Errorf("invalid AZURE_STORAGE_CONNECTION_STRING: %w", err)
		}
	}

	// Validate SAS expiry is a valid duration
	if c.AzureSASExpiry != "" {
		if _, err := time.ParseDuration(c.AzureSASExpiry); err != nil {
			return fmt.Errorf("invalid AZURE_SAS_EXPIRY duration '%s': %w", c.AzureSASExpiry, err)
		}
	}

	return nil
}

// validateConnectionString performs basic validation on Azure connection string format.
// It checks for the presence of required key-value pairs but doesn't validate their actual values.
func validateConnectionString(connStr string) error {
	if connStr == "" {
		return fmt.Errorf("connection string is empty")
	}

	// Connection string should contain key=value pairs separated by semicolons
	// Required fields: AccountName and either AccountKey or SharedAccessSignature
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

// ValidateStateStorage validates state storage configuration based on the selected storage type.
// Returns an error if the configuration is invalid or missing required fields.
func (c *Config) ValidateStateStorage() error {
	// Default to filesystem if not specified (backward compatibility)
	if c.StateStorage.Type == "" {
		c.StateStorage.Type = "filesystem"
	}

	// Normalize type to lowercase
	c.StateStorage.Type = strings.ToLower(c.StateStorage.Type)

	// Validate storage type
	validTypes := map[string]bool{"filesystem": true, "sqlite": true, "postgres": true}
	if !validTypes[c.StateStorage.Type] {
		return fmt.Errorf("invalid state_storage.type '%s': must be 'filesystem', 'sqlite', or 'postgres'", c.StateStorage.Type)
	}

	// Set default migrations path if not specified
	if c.StateStorage.MigrationsPath == "" {
		c.StateStorage.MigrationsPath = "./migrations"
	}

	// Validate SQLite configuration
	if c.StateStorage.Type == "sqlite" {
		// Set default SQLite path if not specified
		if c.StateStorage.SQLitePath == "" {
			c.StateStorage.SQLitePath = filepath.Join(c.WorkspaceRoot, "nightcrier.db")
		}
	}

	// Validate PostgreSQL configuration
	if c.StateStorage.Type == "postgres" {
		// If connection string is provided, validate it
		if c.StateStorage.PostgresConnectionString != "" {
			if err := validatePostgresConnectionString(c.StateStorage.PostgresConnectionString); err != nil {
				return fmt.Errorf("invalid STATE_STORAGE_POSTGRES_CONNECTION_STRING: %w", err)
			}
		} else {
			// Validate individual connection parameters
			if c.StateStorage.PostgresHost == "" {
				return fmt.Errorf("STATE_STORAGE_POSTGRES_HOST is required when state_storage.type is 'postgres' and connection string is not provided")
			}
			if c.StateStorage.PostgresDatabase == "" {
				return fmt.Errorf("STATE_STORAGE_POSTGRES_DATABASE is required when state_storage.type is 'postgres' and connection string is not provided")
			}
			if c.StateStorage.PostgresUser == "" {
				return fmt.Errorf("STATE_STORAGE_POSTGRES_USER is required when state_storage.type is 'postgres' and connection string is not provided")
			}
			// Password is optional (could use peer auth, SSL certs, etc.)

			// Set default port if not specified
			if c.StateStorage.PostgresPort == 0 {
				c.StateStorage.PostgresPort = 5432
			}
		}
	}

	return nil
}

// validatePostgresConnectionString performs basic validation on PostgreSQL connection string format.
// It checks for the presence of required components but doesn't validate their actual values.
func validatePostgresConnectionString(connStr string) error {
	if connStr == "" {
		return fmt.Errorf("connection string is empty")
	}

	// Basic validation: should start with postgres:// or postgresql://
	if !strings.HasPrefix(connStr, "postgres://") && !strings.HasPrefix(connStr, "postgresql://") {
		return fmt.Errorf("connection string must start with 'postgres://' or 'postgresql://'")
	}

	// Should contain @ symbol (separating user info from host)
	if !strings.Contains(connStr, "@") {
		return fmt.Errorf("connection string must contain '@' to separate credentials from host")
	}

	return nil
}

// IsSQLStorageEnabled returns true if SQL-based storage (SQLite or PostgreSQL) is configured.
// Returns false if using filesystem storage (default).
func (c *Config) IsSQLStorageEnabled() bool {
	return c.StateStorage.Type == "sqlite" || c.StateStorage.Type == "postgres"
}

// GetStateStorageType returns the configured state storage type.
// Defaults to "filesystem" if not configured.
func (c *Config) GetStateStorageType() string {
	if c.StateStorage.Type == "" {
		return "filesystem"
	}
	return c.StateStorage.Type
}
