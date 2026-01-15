package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/randybias/nightcrier/internal/cluster"
)

// AgentConfig holds AI agent runtime configuration.
type AgentConfig struct {
	// CLI is the AI CLI to use: claude, codex, goose, gemini
	CLI string `mapstructure:"cli"`

	// Model is the LLM model to use (model names depend on CLI)
	Model string `mapstructure:"model"`

	// Timeout is the maximum execution time in seconds
	Timeout int `mapstructure:"timeout"`

	// SystemPromptFile is the path to the system prompt file
	SystemPromptFile string `mapstructure:"system_prompt_file"`

	// AllowedTools is a comma-separated list of allowed tools
	AllowedTools string `mapstructure:"allowed_tools"`

	// AdditionalPrompt is optional cluster-specific context
	AdditionalPrompt string `mapstructure:"additional_prompt"`
}

// Validate validates the AgentConfig fields
func (a *AgentConfig) Validate() error {
	if a.CLI == "" {
		return fmt.Errorf("agent.cli is required")
	}
	validCLIs := map[string]bool{"claude": true, "codex": true, "goose": true, "gemini": true}
	if !validCLIs[a.CLI] {
		return fmt.Errorf("invalid agent.cli '%s', must be one of: claude, codex, goose, gemini", a.CLI)
	}
	if a.Model == "" {
		return fmt.Errorf("agent.model is required")
	}
	if a.Timeout < 1 {
		return fmt.Errorf("agent.timeout must be >= 1")
	}
	return nil
}

// ExecutionDefaults provides default values for all execution clusters.
// Individual execution clusters can override any of these fields.
type ExecutionDefaults struct {
	// Namespace where Jobs and ConfigMaps are created (default: "nightcrier")
	Namespace string `mapstructure:"namespace"`

	// RunnerImage is the container image for the agent runner (default: "nc-agent-runner:latest")
	RunnerImage string `mapstructure:"runner_image"`

	// ImagePullPolicy: Always, Never, IfNotPresent (default: "IfNotPresent")
	ImagePullPolicy string `mapstructure:"image_pull_policy"`

	// Timeout is the Job timeout in seconds (default: 600)
	Timeout int `mapstructure:"timeout"`

	// MemoryLimit for Job containers (default: "2Gi")
	MemoryLimit string `mapstructure:"memory_limit"`

	// CPULimit for Job containers (default: "1")
	CPULimit string `mapstructure:"cpu_limit"`

	// CleanupTTL is the TTL for Job cleanup after completion in seconds (default: 3600)
	CleanupTTL int `mapstructure:"cleanup_ttl"`

	// MaxConcurrentAgents is the maximum number of concurrent agent Jobs per cluster (default: 10)
	MaxConcurrentAgents int `mapstructure:"max_concurrent_agents"`
}

// ExecutionClusterConfig defines a Kubernetes cluster where agent Jobs run.
// Each execution cluster can override the global execution_defaults.
type ExecutionClusterConfig struct {
	// Name is a unique identifier for this execution cluster (required)
	Name string `mapstructure:"name"`

	// KubeconfigPath is the path to the kubeconfig file for this cluster (required)
	KubeconfigPath string `mapstructure:"kubeconfig_path"`

	// Namespace where Jobs and ConfigMaps are created (optional, uses execution_defaults)
	Namespace string `mapstructure:"namespace"`

	// RunnerImage is the container image for the agent runner (optional, uses execution_defaults)
	RunnerImage string `mapstructure:"runner_image"`

	// ImagePullPolicy: Always, Never, IfNotPresent (optional, uses execution_defaults)
	ImagePullPolicy string `mapstructure:"image_pull_policy"`

	// Timeout is the Job timeout in seconds (optional, uses execution_defaults)
	Timeout int `mapstructure:"timeout"`

	// MemoryLimit for Job containers (optional, uses execution_defaults)
	MemoryLimit string `mapstructure:"memory_limit"`

	// CPULimit for Job containers (optional, uses execution_defaults)
	CPULimit string `mapstructure:"cpu_limit"`

	// CleanupTTL is the TTL for Job cleanup after completion in seconds (optional, uses execution_defaults)
	CleanupTTL int `mapstructure:"cleanup_ttl"`

	// MaxConcurrentAgents is the maximum number of concurrent agent Jobs (optional, uses execution_defaults)
	MaxConcurrentAgents int `mapstructure:"max_concurrent_agents"`
}

// NATSConfig holds NATS progress tracking configuration.
type NATSConfig struct {
	// Enabled controls whether NATS progress tracking is active
	Enabled bool `mapstructure:"enabled"`

	// Server is the NATS server URL (e.g., "nats://localhost:4222")
	Server string `mapstructure:"server"`

	// Token is the authentication token for NATS server
	Token string `mapstructure:"token"`

	// ConnectTimeout is the timeout for initial connection to NATS server
	ConnectTimeout time.Duration `mapstructure:"connect_timeout"`

	// ReconnectWait is the delay between reconnection attempts
	ReconnectWait time.Duration `mapstructure:"reconnect_wait"`
}

// StartupConfig holds configuration for resilient startup behavior.
// This controls how nightcrier handles missing credentials or unavailable
// resources at startup, allowing it to start in degraded mode and recover
// automatically when resources become available.
type StartupConfig struct {
	// CredentialRetryInitial is the initial backoff duration for retrying
	// failed credential/resource bootstrap operations.
	// Default: 5s
	// Environment variable: STARTUP_CREDENTIAL_RETRY_INITIAL
	CredentialRetryInitial time.Duration `mapstructure:"credential_retry_initial"`

	// CredentialRetryMax is the maximum backoff duration for retrying
	// failed credential/resource bootstrap operations.
	// Default: 300s (5 minutes)
	// Environment variable: STARTUP_CREDENTIAL_RETRY_MAX
	CredentialRetryMax time.Duration `mapstructure:"credential_retry_max"`

	// CredentialRetryMultiplier is the multiplier for exponential backoff.
	// Default: 2.0
	// Environment variable: STARTUP_CREDENTIAL_RETRY_MULTIPLIER
	CredentialRetryMultiplier float64 `mapstructure:"credential_retry_multiplier"`
}

// ApplyDefaults sets default values for StartupConfig
func (s *StartupConfig) ApplyDefaults() {
	if s.CredentialRetryInitial == 0 {
		s.CredentialRetryInitial = 5 * time.Second
	}
	if s.CredentialRetryMax == 0 {
		s.CredentialRetryMax = 300 * time.Second
	}
	if s.CredentialRetryMultiplier == 0 {
		s.CredentialRetryMultiplier = 2.0
	}
}

// Validate validates the StartupConfig fields
func (s *StartupConfig) Validate() error {
	if s.CredentialRetryInitial <= 0 {
		return fmt.Errorf("startup.credential_retry_initial must be positive")
	}
	if s.CredentialRetryMax <= 0 {
		return fmt.Errorf("startup.credential_retry_max must be positive")
	}
	if s.CredentialRetryMax < s.CredentialRetryInitial {
		return fmt.Errorf("startup.credential_retry_max (%v) must be >= credential_retry_initial (%v)",
			s.CredentialRetryMax, s.CredentialRetryInitial)
	}
	if s.CredentialRetryMultiplier < 1.0 {
		return fmt.Errorf("startup.credential_retry_multiplier must be >= 1.0")
	}
	return nil
}

// Validate validates the ExecutionDefaults fields
func (e *ExecutionDefaults) Validate() error {
	if e.ImagePullPolicy != "" {
		validPolicies := map[string]bool{"Always": true, "Never": true, "IfNotPresent": true}
		if !validPolicies[e.ImagePullPolicy] {
			return fmt.Errorf("invalid execution_defaults.image_pull_policy '%s', must be one of: Always, Never, IfNotPresent", e.ImagePullPolicy)
		}
	}
	return nil
}

// ApplyDefaults sets default values for ExecutionDefaults
func (e *ExecutionDefaults) ApplyDefaults() {
	if e.Namespace == "" {
		e.Namespace = "nightcrier"
	}
	if e.RunnerImage == "" {
		e.RunnerImage = "nc-agent-runner:latest"
	}
	if e.ImagePullPolicy == "" {
		e.ImagePullPolicy = "IfNotPresent"
	}
	if e.Timeout == 0 {
		e.Timeout = 600
	}
	if e.MemoryLimit == "" {
		e.MemoryLimit = "2Gi"
	}
	if e.CPULimit == "" {
		e.CPULimit = "1"
	}
	if e.CleanupTTL == 0 {
		e.CleanupTTL = 3600
	}
	if e.MaxConcurrentAgents == 0 {
		e.MaxConcurrentAgents = 10
	}
}

// Validate validates the ExecutionClusterConfig fields
func (e *ExecutionClusterConfig) Validate() error {
	if e.Name == "" {
		return fmt.Errorf("execution cluster name is required")
	}
	if e.KubeconfigPath == "" {
		return fmt.Errorf("execution cluster %s: kubeconfig_path is required", e.Name)
	}
	// Check if kubeconfig file exists
	if _, err := os.Stat(e.KubeconfigPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("execution cluster %s: kubeconfig file not found at %q", e.Name, e.KubeconfigPath)
		}
		return fmt.Errorf("execution cluster %s: cannot access kubeconfig at %q: %w", e.Name, e.KubeconfigPath, err)
	}
	if e.ImagePullPolicy != "" {
		validPolicies := map[string]bool{"Always": true, "Never": true, "IfNotPresent": true}
		if !validPolicies[e.ImagePullPolicy] {
			return fmt.Errorf("execution cluster %s: invalid image_pull_policy '%s', must be one of: Always, Never, IfNotPresent", e.Name, e.ImagePullPolicy)
		}
	}
	return nil
}

// ApplyDefaults applies defaults from ExecutionDefaults to this ExecutionClusterConfig
func (e *ExecutionClusterConfig) ApplyDefaults(defaults *ExecutionDefaults) {
	if e.Namespace == "" {
		e.Namespace = defaults.Namespace
	}
	if e.RunnerImage == "" {
		e.RunnerImage = defaults.RunnerImage
	}
	if e.ImagePullPolicy == "" {
		e.ImagePullPolicy = defaults.ImagePullPolicy
	}
	if e.Timeout == 0 {
		e.Timeout = defaults.Timeout
	}
	if e.MemoryLimit == "" {
		e.MemoryLimit = defaults.MemoryLimit
	}
	if e.CPULimit == "" {
		e.CPULimit = defaults.CPULimit
	}
	if e.CleanupTTL == 0 {
		e.CleanupTTL = defaults.CleanupTTL
	}
	if e.MaxConcurrentAgents == 0 {
		e.MaxConcurrentAgents = defaults.MaxConcurrentAgents
	}
}

// Validate validates the NATSConfig fields.
// Returns nil if Enabled is false (validation only applies when enabled).
func (n *NATSConfig) Validate() error {
	// Skip validation if NATS is disabled
	if !n.Enabled {
		return nil
	}

	// When enabled, server and token are required
	if n.Server == "" {
		return fmt.Errorf("nats.server is required when NATS is enabled")
	}
	if n.Token == "" {
		return fmt.Errorf("nats.token is required when NATS is enabled")
	}

	// Validate timeouts are positive
	if n.ConnectTimeout <= 0 {
		return fmt.Errorf("nats.connect_timeout must be positive, got %v", n.ConnectTimeout)
	}
	if n.ReconnectWait <= 0 {
		return fmt.Errorf("nats.reconnect_wait must be positive, got %v", n.ReconnectWait)
	}

	return nil
}

// ApplyDefaults sets default values for NATSConfig
func (n *NATSConfig) ApplyDefaults() {
	if n.ConnectTimeout == 0 {
		n.ConnectTimeout = 5 * time.Second
	}
	if n.ReconnectWait == 0 {
		n.ReconnectWait = 2 * time.Second
	}
}

// Config holds the application configuration.
type Config struct {
	// Monitored Clusters - clusters where fault events are received from MCP servers
	MonitoredClusters []cluster.MonitoredClusterConfig `mapstructure:"monitored_clusters"`
	SubscribeMode     string                           `mapstructure:"subscribe_mode"` // events, faults

	// Execution Clusters - Kubernetes clusters where agent Jobs run
	ExecutionClusters []ExecutionClusterConfig `mapstructure:"execution_clusters"`

	// Execution Defaults - default values for execution clusters
	ExecutionDefaults ExecutionDefaults `mapstructure:"execution_defaults"`

	// Workspace
	WorkspaceRoot string `mapstructure:"workspace_root"`

	// Logging
	LogLevel string `mapstructure:"log_level"`

	// Slack Integration
	SlackWebhookURL string `mapstructure:"slack_webhook_url"`

	// Agent Configuration (nested)
	Agent AgentConfig `mapstructure:"agent"`

	// LLM API Keys (optional - can also be set via environment)
	AnthropicAPIKey string `mapstructure:"anthropic_api_key"`
	OpenAIAPIKey    string `mapstructure:"openai_api_key"`
	GeminiAPIKey    string `mapstructure:"gemini_api_key"`

	// NATS Configuration (optional progress tracking)
	NATS NATSConfig `mapstructure:"nats"`

	// Startup Configuration (resilient credential handling)
	Startup StartupConfig `mapstructure:"startup"`

	// Event Processing (Phase 1 additions)
	SeverityThreshold            string `mapstructure:"severity_threshold"`
	MaxConcurrentAgents          int    `mapstructure:"max_concurrent_agents"`
	GlobalQueueSize              int    `mapstructure:"global_queue_size"`
	DropEventsWhileBusy          *bool  `mapstructure:"drop_events_while_busy"`            // default: true
	ClusterFailureEventQueueSize int    `mapstructure:"cluster_failure_event_queue_size"` // default: 3
	EventTTLSeconds              int    `mapstructure:"event_ttl_seconds"`                // default: 300
	DedupWindowSeconds           int    `mapstructure:"dedup_window_seconds"`
	QueueOverflowPolicy          string `mapstructure:"queue_overflow_policy"`
	ShutdownTimeout              int    `mapstructure:"shutdown_timeout"` // seconds

	// MCP Transport Reconnection
	MCPReconnectInitialBackoff int `mapstructure:"mcp_reconnect_initial_backoff"` // seconds
	MCPReconnectMaxBackoff     int `mapstructure:"mcp_reconnect_max_backoff"`     // seconds
	MCPReadTimeout             int `mapstructure:"mcp_read_timeout"`              // seconds

	// Object Storage Configuration (optional - used when cloud storage is enabled)
	ObjectStorage ObjectStorage `mapstructure:"object_storage"`

	// Circuit Breaker and Notification Configuration (Phase 2)
	NotifyOnAgentFailure       bool `mapstructure:"notify_on_agent_failure"`
	FailureThresholdForAlert   int  `mapstructure:"failure_threshold_for_alert"`
	UploadFailedInvestigations bool `mapstructure:"upload_failed_investigations"`

	// State Storage Configuration (SQL Support)
	// Configures where incident state is persisted. Supports filesystem (backward compatible),
	// SQLite (embedded), and PostgreSQL (centralized). Default: filesystem
	StateStorage StateStorage `mapstructure:"state_storage"`
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

// ObjectStorage configures the Go CDK-based object storage for incident artifacts.
// Supports Azure Blob Storage, S3-compatible storage (AWS S3, MinIO, RustFS), and in-memory storage.
// Storage is optional - if not configured, artifacts are stored locally in the filesystem.
type ObjectStorage struct {
	// URL is the Go CDK storage URL that specifies the provider and bucket/container
	// Examples:
	//   - Azure: "azblob://mycontainer"
	//   - S3: "s3://mybucket?region=us-east-1"
	//   - MinIO/S3-compatible: "s3://mybucket?endpoint=http://minio:9000&disableSSL=true&s3ForcePathStyle=true"
	//   - In-memory (testing): "mem://"
	// Environment variable: OBJECT_STORAGE_URL
	URL string `mapstructure:"url"`

	// SignedURLExpiry is the duration for which signed URLs remain valid
	// Format: Go duration string (e.g., "168h" for 7 days)
	// Default: "168h" (7 days)
	// Environment variable: OBJECT_STORAGE_SIGNED_URL_EXPIRY
	SignedURLExpiry string `mapstructure:"signed_url_expiry"`

	// AWSAccessKeyID is the AWS access key ID for S3-compatible storage
	// Used for S3, MinIO, and other S3-compatible providers
	// Environment variable: AWS_ACCESS_KEY_ID
	AWSAccessKeyID string `mapstructure:"aws_access_key_id"`

	// AWSSecretAccessKey is the AWS secret access key for S3-compatible storage
	// Used for S3, MinIO, and other S3-compatible providers
	// Environment variable: AWS_SECRET_ACCESS_KEY
	AWSSecretAccessKey string `mapstructure:"aws_secret_access_key"`

	// AzureStorageAccount is the Azure storage account name
	// Required when using Azure Blob Storage (azblob://)
	// Environment variable: AZURE_STORAGE_ACCOUNT
	AzureStorageAccount string `mapstructure:"azure_storage_account"`

	// AzureStorageKey is the Azure storage account access key
	// Required when using Azure Blob Storage (azblob://)
	// Environment variable: AZURE_STORAGE_KEY
	AzureStorageKey string `mapstructure:"azure_storage_key"`
}

// bindEnvVars binds environment variables to viper keys.
// Environment variables use uppercase with underscores (e.g., WORKSPACE_ROOT).
func bindEnvVars() {
	// Map config keys to environment variable names
	envBindings := map[string]string{
		"subscribe_mode":    "SUBSCRIBE_MODE",
		"workspace_root":    "WORKSPACE_ROOT",
		"log_level":         "LOG_LEVEL",
		"slack_webhook_url": "SLACK_WEBHOOK_URL",
		// Agent configuration (nested)
		"agent.cli":                "AGENT_CLI",
		"agent.model":              "AGENT_MODEL",
		"agent.timeout":            "AGENT_TIMEOUT",
		"agent.system_prompt_file": "AGENT_SYSTEM_PROMPT_FILE",
		"agent.allowed_tools":      "AGENT_ALLOWED_TOOLS",
		"agent.additional_prompt":  "ADDITIONAL_AGENT_PROMPT",
		"anthropic_api_key":        "ANTHROPIC_API_KEY",
		"openai_api_key":           "OPENAI_API_KEY",
		"gemini_api_key":           "GEMINI_API_KEY",
		// Execution defaults configuration (nested)
		"execution_defaults.namespace":            "EXECUTION_DEFAULTS_NAMESPACE",
		"execution_defaults.runner_image":         "EXECUTION_DEFAULTS_RUNNER_IMAGE",
		"execution_defaults.image_pull_policy":    "EXECUTION_DEFAULTS_IMAGE_PULL_POLICY",
		"execution_defaults.timeout":              "EXECUTION_DEFAULTS_TIMEOUT",
		"execution_defaults.memory_limit":         "EXECUTION_DEFAULTS_MEMORY_LIMIT",
		"execution_defaults.cpu_limit":            "EXECUTION_DEFAULTS_CPU_LIMIT",
		"execution_defaults.cleanup_ttl":          "EXECUTION_DEFAULTS_CLEANUP_TTL",
		"execution_defaults.max_concurrent_agents": "EXECUTION_DEFAULTS_MAX_CONCURRENT_AGENTS",
		// NATS configuration (nested)
		"nats.enabled":                             "NATS_ENABLED",
		"nats.server":                              "NATS_SERVER",
		"nats.token":                               "NATS_TOKEN",
		"nats.connect_timeout":                     "NATS_CONNECT_TIMEOUT",
		"nats.reconnect_wait":                      "NATS_RECONNECT_WAIT",
		"severity_threshold":                       "SEVERITY_THRESHOLD",
		"max_concurrent_agents":                    "MAX_CONCURRENT_AGENTS",
		"global_queue_size":                        "GLOBAL_QUEUE_SIZE",
		"cluster_queue_size":                       "CLUSTER_QUEUE_SIZE",
		"event_ttl_seconds":                        "EVENT_TTL_SECONDS",
		"dedup_window_seconds":                     "DEDUP_WINDOW_SECONDS",
		"queue_overflow_policy":                    "QUEUE_OVERFLOW_POLICY",
		"shutdown_timeout":                         "SHUTDOWN_TIMEOUT_SECONDS",
		"mcp_reconnect_initial_backoff":            "MCP_RECONNECT_INITIAL_BACKOFF",
		"mcp_reconnect_max_backoff":                "MCP_RECONNECT_MAX_BACKOFF",
		"mcp_read_timeout":                         "MCP_READ_TIMEOUT_SECONDS",
		"object_storage.url":                       "OBJECT_STORAGE_URL",
		"object_storage.signed_url_expiry":         "OBJECT_STORAGE_SIGNED_URL_EXPIRY",
		"object_storage.aws_access_key_id":         "AWS_ACCESS_KEY_ID",
		"object_storage.aws_secret_access_key":     "AWS_SECRET_ACCESS_KEY",
		"object_storage.azure_storage_account":     "AZURE_STORAGE_ACCOUNT",
		"object_storage.azure_storage_key":         "AZURE_STORAGE_KEY",
		"notify_on_agent_failure":                  "NOTIFY_ON_AGENT_FAILURE",
		"failure_threshold_for_alert":              "FAILURE_THRESHOLD_FOR_ALERT",
		"upload_failed_investigations":             "UPLOAD_FAILED_INVESTIGATIONS",
		"state_storage.type":                       "STATE_STORAGE_TYPE",
		"state_storage.sqlite_path":                "STATE_STORAGE_SQLITE_PATH",
		"state_storage.postgres_connection_string": "STATE_STORAGE_POSTGRES_CONNECTION_STRING",
		"state_storage.postgres_host":              "STATE_STORAGE_POSTGRES_HOST",
		"state_storage.postgres_port":              "STATE_STORAGE_POSTGRES_PORT",
		"state_storage.postgres_database":          "STATE_STORAGE_POSTGRES_DATABASE",
		"state_storage.postgres_user":              "STATE_STORAGE_POSTGRES_USER",
		"state_storage.postgres_password":          "STATE_STORAGE_POSTGRES_PASSWORD",
		"state_storage.migrations_path":            "STATE_STORAGE_MIGRATIONS_PATH",
		"startup.credential_retry_initial":         "STARTUP_CREDENTIAL_RETRY_INITIAL",
		"startup.credential_retry_max":             "STARTUP_CREDENTIAL_RETRY_MAX",
		"startup.credential_retry_multiplier":      "STARTUP_CREDENTIAL_RETRY_MULTIPLIER",
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
		"workspace-root":               "workspace_root",
		"log-level":                    "log_level",
		"config":                       "config_file",
		"agent-timeout":                "agent_timeout",
		"severity-threshold":           "severity_threshold",
		"max-concurrent-agents":        "max_concurrent_agents",
		"shutdown-timeout":             "shutdown_timeout",
		"notify-on-agent-failure":      "notify_on_agent_failure",
		"failure-threshold-for-alert":  "failure_threshold_for_alert",
		"upload-failed-investigations": "upload_failed_investigations",
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

	// Monitored Clusters: zero clusters allowed at startup (warn only)
	if len(c.MonitoredClusters) == 0 {
		// Log warning - will be logged by caller since we don't have logger here
		// System will poll database for new clusters
	}

	// Validate monitored cluster name uniqueness and individual cluster configs
	monitoredClusterNames := make(map[string]bool)
	triageEnabledCount := 0
	for i, mc := range c.MonitoredClusters {
		if mc.Name == "" {
			return fmt.Errorf("monitored_clusters[%d]: name is required", i)
		}

		if monitoredClusterNames[mc.Name] {
			return fmt.Errorf("duplicate monitored cluster name: %s", mc.Name)
		}
		monitoredClusterNames[mc.Name] = true

		// Validate individual monitored cluster config
		if err := mc.Validate(); err != nil {
			return fmt.Errorf("monitored_clusters[%d] (%s): %w", i, mc.Name, err)
		}

		if mc.Triage.Enabled {
			triageEnabledCount++
		}
	}

	// Apply execution defaults first
	c.ExecutionDefaults.ApplyDefaults()
	if err := c.ExecutionDefaults.Validate(); err != nil {
		return fmt.Errorf("execution_defaults configuration invalid: %w", err)
	}

	// Validate execution clusters
	executionClusterNames := make(map[string]bool)
	for i := range c.ExecutionClusters {
		ec := &c.ExecutionClusters[i]
		if ec.Name == "" {
			return fmt.Errorf("execution_clusters[%d]: name is required", i)
		}

		if executionClusterNames[ec.Name] {
			return fmt.Errorf("duplicate execution cluster name: %s", ec.Name)
		}
		executionClusterNames[ec.Name] = true

		// Apply defaults from ExecutionDefaults to this cluster
		ec.ApplyDefaults(&c.ExecutionDefaults)

		// Validate individual execution cluster config
		if err := ec.Validate(); err != nil {
			return fmt.Errorf("execution_clusters[%d] (%s): %w", i, ec.Name, err)
		}
	}

	// Validate execution cluster references in monitored clusters
	for i, mc := range c.MonitoredClusters {
		if mc.Triage.Enabled && mc.Triage.ExecutionCluster != "" {
			if !executionClusterNames[mc.Triage.ExecutionCluster] {
				return fmt.Errorf("monitored_clusters[%d] (%s): triage.execution_cluster references non-existent execution cluster %q", i, mc.Name, mc.Triage.ExecutionCluster)
			}
		}
	}

	// If any monitored cluster has triage enabled but no execution clusters exist, return error
	if triageEnabledCount > 0 && len(c.ExecutionClusters) == 0 {
		return fmt.Errorf("triage is enabled for %d monitored cluster(s) but no execution_clusters are configured - agent Jobs cannot be created", triageEnabledCount)
	}

	if c.SubscribeMode == "" {
		return missingFieldError("subscribe_mode", "SUBSCRIBE_MODE")
	}

	// Required: Workspace
	if c.WorkspaceRoot == "" {
		return missingFieldError("workspace_root", "WORKSPACE_ROOT")
	}

	// Validate nested agent configuration
	if err := c.Agent.Validate(); err != nil {
		return fmt.Errorf("agent configuration invalid: %w", err)
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

	// Apply default for DropEventsWhileBusy (optional with default: true)
	if c.DropEventsWhileBusy == nil {
		defaultTrue := true
		c.DropEventsWhileBusy = &defaultTrue
	}

	// Apply default for ClusterFailureEventQueueSize (optional with default: 3)
	if c.ClusterFailureEventQueueSize == 0 {
		c.ClusterFailureEventQueueSize = 3
	}

	// Apply default for EventTTLSeconds (optional with default: 300)
	if c.EventTTLSeconds == 0 {
		c.EventTTLSeconds = 300
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

	// Required: MCP Transport Reconnection
	if c.MCPReconnectInitialBackoff == 0 {
		return missingFieldError("mcp_reconnect_initial_backoff", "MCP_RECONNECT_INITIAL_BACKOFF")
	}

	if c.MCPReconnectMaxBackoff == 0 {
		return missingFieldError("mcp_reconnect_max_backoff", "MCP_RECONNECT_MAX_BACKOFF")
	}

	if c.MCPReadTimeout == 0 {
		return missingFieldError("mcp_read_timeout", "MCP_READ_TIMEOUT_SECONDS")
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
	if c.ClusterFailureEventQueueSize < 1 {
		return fmt.Errorf("cluster_failure_event_queue_size must be >= 1, got %d. Set via CLUSTER_FAILURE_EVENT_QUEUE_SIZE environment variable or config file", c.ClusterFailureEventQueueSize)
	}
	if c.EventTTLSeconds < 1 {
		return fmt.Errorf("event_ttl_seconds must be >= 1, got %d. Set via EVENT_TTL_SECONDS environment variable or config file", c.EventTTLSeconds)
	}
	if c.DedupWindowSeconds < 0 {
		return fmt.Errorf("dedup_window_seconds must be >= 0, got %d. Set via DEDUP_WINDOW_SECONDS environment variable or config file", c.DedupWindowSeconds)
	}
	if c.ShutdownTimeout < 1 {
		return fmt.Errorf("shutdown_timeout must be >= 1, got %d. Set via SHUTDOWN_TIMEOUT_SECONDS environment variable or config file", c.ShutdownTimeout)
	}

	// Validate queue overflow policy
	validPolicies := map[string]bool{"drop": true, "reject": true}
	if !validPolicies[strings.ToLower(c.QueueOverflowPolicy)] {
		return fmt.Errorf("invalid queue_overflow_policy '%s': must be 'drop' or 'reject'. Set via QUEUE_OVERFLOW_POLICY environment variable or config file", c.QueueOverflowPolicy)
	}

	// Validate MCP transport reconnection settings
	if c.MCPReconnectInitialBackoff < 1 {
		return fmt.Errorf("mcp_reconnect_initial_backoff must be >= 1, got %d. Set via MCP_RECONNECT_INITIAL_BACKOFF environment variable or config file", c.MCPReconnectInitialBackoff)
	}
	if c.MCPReconnectMaxBackoff < c.MCPReconnectInitialBackoff {
		return fmt.Errorf("mcp_reconnect_max_backoff (%d) must be >= mcp_reconnect_initial_backoff (%d). Set via MCP_RECONNECT_MAX_BACKOFF environment variable or config file",
			c.MCPReconnectMaxBackoff, c.MCPReconnectInitialBackoff)
	}
	if c.MCPReadTimeout < 1 {
		return fmt.Errorf("mcp_read_timeout must be >= 1, got %d. Set via MCP_READ_TIMEOUT_SECONDS environment variable or config file", c.MCPReadTimeout)
	}

	// Validate circuit breaker settings
	if c.FailureThresholdForAlert < 1 {
		return fmt.Errorf("failure_threshold_for_alert must be >= 1, got %d. Set via FAILURE_THRESHOLD_FOR_ALERT environment variable or config file", c.FailureThresholdForAlert)
	}

	// Require at least one LLM API key
	if err := c.ValidateLLMAPIKeys(); err != nil {
		return err
	}

	// Validate object storage configuration if enabled
	if err := c.ValidateObjectStorageConfig(); err != nil {
		return err
	}

	// Validate state storage configuration
	if err := c.ValidateStateStorage(); err != nil {
		return err
	}

	// Apply NATS defaults and validate (NATS is optional)
	c.NATS.ApplyDefaults()
	if err := c.NATS.Validate(); err != nil {
		return fmt.Errorf("nats configuration invalid: %w", err)
	}

	// Apply startup defaults and validate
	c.Startup.ApplyDefaults()
	if err := c.Startup.Validate(); err != nil {
		return fmt.Errorf("startup configuration invalid: %w", err)
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

// GetWorkspaceRoot returns the configured workspace root directory.
// This method is part of the StorageConfig interface.
func (c *Config) GetWorkspaceRoot() string {
	return c.WorkspaceRoot
}

// GetObjectStorageURL returns the Go CDK storage URL (empty if not configured).
// This method is part of the StorageConfig interface.
func (c *Config) GetObjectStorageURL() string {
	return c.ObjectStorage.URL
}

// GetObjectStorageExpiry returns the signed URL expiry duration.
// This method is part of the StorageConfig interface.
func (c *Config) GetObjectStorageExpiry() time.Duration {
	if c.ObjectStorage.SignedURLExpiry == "" {
		return 168 * time.Hour // Default 7 days
	}
	duration, err := time.ParseDuration(c.ObjectStorage.SignedURLExpiry)
	if err != nil {
		return 168 * time.Hour // Default on parse error
	}
	return duration
}

// GetObjectStorageType returns a human-readable storage type derived from the URL scheme.
// Returns "not_configured" if the URL is empty.
func (c *Config) GetObjectStorageType() string {
	if c.ObjectStorage.URL == "" {
		return "not_configured"
	}
	parts := strings.SplitN(c.ObjectStorage.URL, "://", 2)
	if len(parts) < 2 {
		return "unknown"
	}
	switch parts[0] {
	case "mem":
		return "memory"
	case "azblob":
		return "azure_blob"
	case "s3":
		return "s3"
	case "gs":
		return "gcs"
	default:
		return parts[0]
	}
}

// GetAzureStorageAccount returns the Azure storage account name (for Azure provider).
// This method is part of the StorageConfig interface.
func (c *Config) GetAzureStorageAccount() string {
	return c.ObjectStorage.AzureStorageAccount
}

// GetAzureStorageKey returns the Azure storage account key.
func (c *Config) GetAzureStorageKey() string {
	return c.ObjectStorage.AzureStorageKey
}

// GetAWSAccessKeyID returns the AWS access key ID for S3.
func (c *Config) GetAWSAccessKeyID() string {
	return c.ObjectStorage.AWSAccessKeyID
}

// GetAWSSecretAccessKey returns the AWS secret access key for S3.
func (c *Config) GetAWSSecretAccessKey() string {
	return c.ObjectStorage.AWSSecretAccessKey
}

// IsAzureStorageEnabled detects if object storage is configured.
// This method maintains backward compatibility with the StorageConfig interface.
// It returns true if any object storage URL is configured.
// Deprecated: This method will be removed once the storage layer is refactored to use ObjectStorage.
func (c *Config) IsAzureStorageEnabled() bool {
	return c.ObjectStorage.URL != ""
}

// ValidateObjectStorageConfig validates object storage configuration if object storage is enabled.
// Returns an error if object storage is enabled but required fields are missing or invalid.
func (c *Config) ValidateObjectStorageConfig() error {
	// If object storage URL is not set, no validation needed
	if c.ObjectStorage.URL == "" {
		return nil
	}

	// Extract URL scheme to determine provider
	parts := strings.SplitN(c.ObjectStorage.URL, "://", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid OBJECT_STORAGE_URL format: must be in the form 'scheme://path'")
	}
	scheme := parts[0]

	// Validate URL scheme is supported
	supportedSchemes := map[string]bool{
		"azblob": true,
		"s3":     true,
		"mem":    true,
	}
	if !supportedSchemes[scheme] {
		return fmt.Errorf("unsupported OBJECT_STORAGE_URL scheme '%s': must be one of azblob, s3, mem", scheme)
	}

	// Skip credential validation for in-memory storage
	if scheme == "mem" {
		return nil
	}

	// Validate Azure credentials if using Azure Blob Storage
	if scheme == "azblob" {
		if c.ObjectStorage.AzureStorageAccount == "" {
			return fmt.Errorf("AZURE_STORAGE_ACCOUNT is required when using azblob:// storage")
		}
		if c.ObjectStorage.AzureStorageKey == "" {
			return fmt.Errorf("AZURE_STORAGE_KEY is required when using azblob:// storage")
		}
	}

	// Validate AWS credentials if using S3-compatible storage
	if scheme == "s3" {
		if c.ObjectStorage.AWSAccessKeyID == "" {
			return fmt.Errorf("AWS_ACCESS_KEY_ID is required when using s3:// storage")
		}
		if c.ObjectStorage.AWSSecretAccessKey == "" {
			return fmt.Errorf("AWS_SECRET_ACCESS_KEY is required when using s3:// storage")
		}
	}

	// Validate signed URL expiry if provided
	if c.ObjectStorage.SignedURLExpiry != "" {
		if _, err := time.ParseDuration(c.ObjectStorage.SignedURLExpiry); err != nil {
			return fmt.Errorf("invalid OBJECT_STORAGE_SIGNED_URL_EXPIRY duration '%s': %w", c.ObjectStorage.SignedURLExpiry, err)
		}
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
