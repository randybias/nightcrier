// Package k8s provides Kubernetes client initialization and utilities for the Nightcrier agent.
package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/randybias/nightcrier/internal/cluster"
	"github.com/randybias/nightcrier/internal/incident"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigMapData holds the data to be stored in the ConfigMap for an incident.
// This includes all input files needed by the agent container.
type ConfigMapData struct {
	// IncidentJSON is the marshaled incident data (incident.json)
	IncidentJSON string

	// PermissionsJSON is the marshaled cluster permissions data (permissions.json)
	PermissionsJSON string

	// BaseTriagePrompt is the base triage prompt content (base-triage-prompt.md)
	BaseTriagePrompt string
}

// ConfigMapConfig holds configuration for ConfigMap creation.
type ConfigMapConfig struct {
	// Namespace is the Kubernetes namespace where the ConfigMap will be created
	Namespace string

	// IncidentID is the unique identifier for the incident
	IncidentID string

	// ClusterName is the name of the cluster being investigated
	ClusterName string

	// Labels are additional labels to apply to the ConfigMap
	Labels map[string]string
}

// CreateIncidentConfigMap creates a ConfigMap containing incident data for agent execution.
// The ConfigMap includes:
//   - incident.json: Full incident context (resource, fault type, severity, etc.)
//   - permissions.json: Validated cluster permissions for the triage agent
//   - base-triage-prompt.md: Base triage prompt content for the agent
//
// Labels applied:
//   - app=nc-agent-runner: Identifies ConfigMaps managed by the agent executor
//   - incident-id={incidentID}: Links to specific incident for cleanup
//   - cluster={clusterName}: Links to target cluster
//
// Returns the created ConfigMap name on success.
func (c *Client) CreateIncidentConfigMap(ctx context.Context, cfg ConfigMapConfig, data ConfigMapData) (string, error) {
	// Generate ConfigMap name based on incident ID
	configMapName := fmt.Sprintf("incident-%s", cfg.IncidentID)

	// Build labels
	labels := map[string]string{
		"app":         "nc-agent-runner",
		"incident-id": cfg.IncidentID,
		"cluster":     cfg.ClusterName,
	}
	// Merge additional labels if provided
	for k, v := range cfg.Labels {
		labels[k] = v
	}

	// Create ConfigMap
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: cfg.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"incident.json":          data.IncidentJSON,
			"permissions.json":       data.PermissionsJSON,
			"base-triage-prompt.md":  data.BaseTriagePrompt,
		},
	}

	createdConfigMap, err := c.clientset.CoreV1().ConfigMaps(cfg.Namespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create ConfigMap %s: %w", configMapName, err)
	}

	return createdConfigMap.Name, nil
}

// DeleteConfigMap deletes a ConfigMap by name.
// This is used for cleanup after Job completion.
func (c *Client) DeleteConfigMap(ctx context.Context, namespace, name string) error {
	err := c.clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete ConfigMap %s: %w", name, err)
	}
	return nil
}

// MarshalIncidentToJSON marshals an Incident to JSON string for ConfigMap storage.
func MarshalIncidentToJSON(inc *incident.Incident) (string, error) {
	data, err := json.MarshalIndent(inc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal incident: %w", err)
	}
	return string(data), nil
}

// MarshalPermissionsToJSON marshals ClusterPermissions to JSON string for ConfigMap storage.
func MarshalPermissionsToJSON(perms *cluster.ClusterPermissions) (string, error) {
	data, err := json.MarshalIndent(perms, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal permissions: %w", err)
	}
	return string(data), nil
}

// CleanupOrphanedConfigMaps deletes orphaned ConfigMaps that are older than the specified age.
// Orphaned ConfigMaps are those with the label app=nc-agent-runner that were created
// more than maxAge ago. This cleanup is typically run on startup to remove ConfigMaps
// left behind by failed or interrupted Jobs.
//
// The default recommendation is to delete ConfigMaps older than 24 hours.
func (c *Client) CleanupOrphanedConfigMaps(ctx context.Context, namespace string, maxAge string) ([]string, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	// Parse max age duration
	maxAgeDuration, err := parseAge(maxAge)
	if err != nil {
		return nil, fmt.Errorf("failed to parse max age: %w", err)
	}

	// List ConfigMaps with the nc-agent-runner label
	listOpts := metav1.ListOptions{
		LabelSelector: "app=nc-agent-runner",
	}

	configMaps, err := c.clientset.CoreV1().ConfigMaps(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list ConfigMaps: %w", err)
	}

	var deleted []string
	cutoffTime := metav1.Now().Add(-maxAgeDuration)

	for _, cm := range configMaps.Items {
		// Check if ConfigMap is older than maxAge
		if cm.CreationTimestamp.Time.Before(cutoffTime) {
			// Delete the ConfigMap
			err := c.DeleteConfigMap(ctx, namespace, cm.Name)
			if err != nil {
				// Log error but continue with other ConfigMaps
				continue
			}
			deleted = append(deleted, cm.Name)
		}
	}

	return deleted, nil
}

// parseAge parses a duration string like "24h", "1h30m", "30m".
// This is a helper for CleanupOrphanedConfigMaps.
func parseAge(age string) (time.Duration, error) {
	d, err := time.ParseDuration(age)
	if err != nil {
		return 0, fmt.Errorf("invalid duration format: %w", err)
	}
	if d < 0 {
		return 0, fmt.Errorf("duration cannot be negative")
	}
	return d, nil
}
