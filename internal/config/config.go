package config

import (
	"fmt"
	"os"
)

// Config holds the application configuration.
type Config struct {
	MCPEndpoint           string
	WorkspaceRoot         string
	LogLevel              string
	SlackWebhookURL       string
	AgentScriptPath       string
	AgentSystemPromptFile string
	AgentAllowedTools     string
	AgentModel            string
	AgentTimeout          int // seconds
}

// Load creates a Config by loading values from environment variables with defaults.
func Load() (*Config, error) {
	cfg := &Config{
		MCPEndpoint:           os.Getenv("K8S_CLUSTER_MCP_ENDPOINT"),
		WorkspaceRoot:         getEnvOrDefault("WORKSPACE_ROOT", "./incidents"),
		LogLevel:              getEnvOrDefault("LOG_LEVEL", "info"),
		SlackWebhookURL:       os.Getenv("SLACK_WEBHOOK_URL"),
		AgentScriptPath:       getEnvOrDefault("AGENT_SCRIPT_PATH", "./agent-container/run-agent.sh"),
		AgentSystemPromptFile: getEnvOrDefault("AGENT_SYSTEM_PROMPT_FILE", "./configs/triage-system-prompt.md"),
		AgentAllowedTools:     getEnvOrDefault("AGENT_ALLOWED_TOOLS", "Read,Write,Grep,Glob,Bash,Skill"),
		AgentModel:            getEnvOrDefault("AGENT_MODEL", "sonnet"),
		AgentTimeout:          getEnvOrDefaultInt("AGENT_TIMEOUT", 300),
	}

	if cfg.MCPEndpoint == "" {
		return nil, fmt.Errorf("K8S_CLUSTER_MCP_ENDPOINT is required")
	}

	return cfg, nil
}

// getEnvOrDefaultInt returns the environment variable value as int or a default if not set.
func getEnvOrDefaultInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var result int
		if _, err := fmt.Sscanf(value, "%d", &result); err == nil {
			return result
		}
	}
	return defaultValue
}

// getEnvOrDefault returns the environment variable value or a default if not set.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
