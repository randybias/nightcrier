package config

import (
	"fmt"
	"os"
)

// Config holds the application configuration.
type Config struct {
	SSEEndpoint   string
	WorkspaceRoot string
	LogLevel      string
}

// Load creates a Config by loading values from environment variables with defaults.
func Load() (*Config, error) {
	cfg := &Config{
		SSEEndpoint:   os.Getenv("SSE_ENDPOINT"),
		WorkspaceRoot: getEnvOrDefault("WORKSPACE_ROOT", "./incidents"),
		LogLevel:      getEnvOrDefault("LOG_LEVEL", "info"),
	}

	if cfg.SSEEndpoint == "" {
		return nil, fmt.Errorf("SSE_ENDPOINT is required")
	}

	return cfg, nil
}

// getEnvOrDefault returns the environment variable value or a default if not set.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
