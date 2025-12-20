// Package cluster provides cluster configuration and connection management.
// It defines the structures and validation logic for managing multiple
// Kubernetes cluster connections and their associated triage configurations.
package cluster

import (
	"fmt"
	"os"
	"strings"
)

// ClusterConfig defines a single cluster's connection and triage configuration.
// Each cluster has its own MCP server endpoint for receiving fault events
// and optional triage configuration for investigating incidents.
type ClusterConfig struct {
	// Name is a unique identifier for this cluster (required).
	Name string `mapstructure:"name" validate:"required"`

	// Environment describes the cluster's deployment environment (e.g., "production", "staging").
	// This is optional and used for organizational purposes.
	Environment string `mapstructure:"environment"`

	// Labels are arbitrary key-value pairs for filtering and routing.
	// These are included in ClusterEvent metadata for downstream processing.
	Labels map[string]string `mapstructure:"labels"`

	// MCP defines the MCP server connection settings for this cluster.
	MCP MCPConfig `mapstructure:"mcp"`

	// Triage defines the triage agent settings for investigating incidents.
	Triage TriageConfig `mapstructure:"triage"`
}

// MCPConfig defines the MCP server connection settings.
// The MCP server sends fault events via Server-Sent Events (SSE).
type MCPConfig struct {
	// Endpoint is the MCP server URL (required, must be a valid URL).
	// Example: "http://localhost:8080"
	Endpoint string `mapstructure:"endpoint" validate:"required,url"`

	// APIKey is a placeholder for future MCP server authentication.
	// Currently ignored but documented in config for forward compatibility.
	// When MCP servers support authentication, this field will be used.
	APIKey string `mapstructure:"api_key"`
}

// TriageConfig defines the triage agent settings for a cluster.
// Triage can be enabled/disabled per cluster. When enabled, agents
// investigate incidents using kubectl access to the cluster.
type TriageConfig struct {
	// Enabled controls whether triage agents are spawned for this cluster.
	// Default: false (triage disabled)
	Enabled bool `mapstructure:"enabled"`

	// Kubeconfig is the path to the kubeconfig file for cluster access.
	// Required when Enabled=true. The kubeconfig must exist and be readable.
	// Validation: required_if=Enabled true, file exists
	Kubeconfig string `mapstructure:"kubeconfig" validate:"required_if=Enabled true,file"`

	// AllowSecretsAccess controls whether the triage agent can read secrets/configmaps.
	// Default: false (disabled for security)
	//
	// When enabled, the agent can access Helm release data and other secrets.
	// This requires the kubeconfig ServiceAccount to have secrets read permissions.
	//
	// Security note: This is a conscious trade-off. Secrets may contain sensitive
	// data (credentials, keys). The agent prompt instructs read-only behavior,
	// but cannot technically prevent the LLM from seeing secret contents.
	//
	// Future consideration: kubernetes-mcp-server could add restricted queries
	// that expose Helm metadata without revealing secret values, or support
	// dynamic permission escalation with operator approval.
	AllowSecretsAccess bool `mapstructure:"allow_secrets_access"`
}

// Validate checks the ClusterConfig for required fields and valid values.
// It performs comprehensive validation including:
// - Name uniqueness (handled by caller)
// - MCP endpoint presence
// - Triage kubeconfig existence (if triage enabled)
// - Label key/value validity
func (c *ClusterConfig) Validate() error {
	// Validate name
	if c.Name == "" {
		return fmt.Errorf("cluster name is required")
	}

	// Validate name format (alphanumeric, hyphens, underscores)
	if !isValidClusterName(c.Name) {
		return fmt.Errorf("cluster name %q is invalid: must contain only alphanumeric characters, hyphens, and underscores", c.Name)
	}

	// Validate MCP endpoint
	if c.MCP.Endpoint == "" {
		return fmt.Errorf("cluster %s: mcp.endpoint is required", c.Name)
	}

	// Basic URL format validation
	if !strings.HasPrefix(c.MCP.Endpoint, "http://") && !strings.HasPrefix(c.MCP.Endpoint, "https://") {
		return fmt.Errorf("cluster %s: mcp.endpoint must start with http:// or https://, got %q", c.Name, c.MCP.Endpoint)
	}

	// Validate triage configuration
	if c.Triage.Enabled {
		if c.Triage.Kubeconfig == "" {
			return fmt.Errorf("cluster %s: triage.kubeconfig is required when triage.enabled=true", c.Name)
		}

		// Check if kubeconfig file exists
		if _, err := os.Stat(c.Triage.Kubeconfig); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("cluster %s: kubeconfig file not found at %q", c.Name, c.Triage.Kubeconfig)
			}
			return fmt.Errorf("cluster %s: cannot access kubeconfig at %q: %w", c.Name, c.Triage.Kubeconfig, err)
		}
	}

	// Validate labels (keys and values)
	for key, value := range c.Labels {
		if key == "" {
			return fmt.Errorf("cluster %s: label key cannot be empty", c.Name)
		}
		if !isValidLabelKey(key) {
			return fmt.Errorf("cluster %s: invalid label key %q: must contain only alphanumeric characters, hyphens, underscores, dots, and slashes", c.Name, key)
		}
		if !isValidLabelValue(value) {
			return fmt.Errorf("cluster %s: invalid label value for key %q: must contain only alphanumeric characters, hyphens, underscores, and dots", c.Name, key)
		}
	}

	return nil
}

// isValidClusterName checks if a cluster name follows naming conventions.
// Valid names contain only alphanumeric characters, hyphens, and underscores.
func isValidClusterName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

// isValidLabelKey checks if a label key follows Kubernetes label key conventions.
// Valid keys contain alphanumeric characters, hyphens, underscores, dots, and slashes.
func isValidLabelKey(key string) bool {
	if key == "" {
		return false
	}
	for _, r := range key {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' || r == '/') {
			return false
		}
	}
	return true
}

// isValidLabelValue checks if a label value follows Kubernetes label value conventions.
// Valid values contain alphanumeric characters, hyphens, underscores, and dots.
func isValidLabelValue(value string) bool {
	// Empty values are allowed
	if value == "" {
		return true
	}
	for _, r := range value {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.') {
			return false
		}
	}
	return true
}
