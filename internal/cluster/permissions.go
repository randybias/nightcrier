package cluster

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"
)

// ClusterPermissions captures the validated permissions for a cluster.
// This is computed at startup and written to incident workspaces to inform
// the triage agent about what it can and cannot do.
//
// Expected RBAC on the target cluster:
//   - ClusterRole "view" (built-in): pods, deployments, services, events, etc.
//   - ClusterRole "helm-readonly": secrets, configmaps (for Helm release data) - optional
//   - ClusterRole "nodes-readonly": nodes
type ClusterPermissions struct {
	ClusterName string    `json:"cluster_name"`
	ValidatedAt time.Time `json:"validated_at"`

	// Core triage permissions (from view ClusterRole)
	CanGetPods        bool `json:"can_get_pods"`
	CanGetLogs        bool `json:"can_get_logs"`        // pods/log subresource
	CanGetEvents      bool `json:"can_get_events"`
	CanGetDeployments bool `json:"can_get_deployments"`
	CanGetServices    bool `json:"can_get_services"`

	// Secrets/ConfigMaps access (from helm-readonly ClusterRole)
	// Only checked/enabled if triage.allow_secrets_access=true in config
	SecretsAccessAllowed bool `json:"secrets_access_allowed"` // Config setting
	CanGetSecrets        bool `json:"can_get_secrets"`        // Actual RBAC check
	CanGetConfigMaps     bool `json:"can_get_configmaps"`     // Actual RBAC check

	// Node permissions (from nodes-readonly ClusterRole)
	CanGetNodes bool `json:"can_get_nodes"`

	// Validation metadata
	RawOutput string   `json:"raw_output,omitempty"` // kubectl auth can-i --list output
	Warnings  []string `json:"warnings,omitempty"`
}

// MinimumPermissionsMet returns true if minimum triage permissions are available.
// Minimum set: pods, logs, events (core incident investigation).
func (p *ClusterPermissions) MinimumPermissionsMet() bool {
	return p.CanGetPods && p.CanGetLogs && p.CanGetEvents
}

// HelmAccessAvailable returns true if Helm release debugging is possible.
// Requires both config allowance AND actual RBAC permissions.
func (p *ClusterPermissions) HelmAccessAvailable() bool {
	return p.SecretsAccessAllowed && p.CanGetSecrets
}

// validateClusterPermissions validates cluster access permissions using kubectl.
// It runs kubectl auth can-i checks for various resources to determine what
// the triage agent will be able to access.
//
// Design reference: design.md lines 305-383
//
// Parameters:
//   - ctx: Context for command execution (with timeout)
//   - cfg: Cluster configuration containing kubeconfig path
//
// Returns ClusterPermissions struct with validation results, or error if kubectl fails.
func validateClusterPermissions(ctx context.Context, cfg *ClusterConfig) (*ClusterPermissions, error) {
	perms := &ClusterPermissions{
		ClusterName:          cfg.Name,
		ValidatedAt:          time.Now(),
		SecretsAccessAllowed: cfg.Triage.AllowSecretsAccess,
	}

	// Run kubectl auth can-i --list (for raw output reference)
	slog.Debug("running kubectl auth can-i --list",
		"cluster", cfg.Name,
		"kubeconfig", cfg.Triage.Kubeconfig)

	cmd := exec.CommandContext(ctx, "kubectl",
		"--kubeconfig", cfg.Triage.Kubeconfig,
		"auth", "can-i", "--list")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("kubectl auth can-i --list failed: %w (output: %s)", err, string(output))
	}

	perms.RawOutput = string(output)

	// Check specific permissions using targeted can-i queries
	// This is more reliable than parsing the --list output
	checks := []struct {
		resource string
		verb     string
		target   *bool
	}{
		{"pods", "get", &perms.CanGetPods},
		{"pods/log", "get", &perms.CanGetLogs},
		{"events", "get", &perms.CanGetEvents},
		{"deployments", "get", &perms.CanGetDeployments},
		{"services", "get", &perms.CanGetServices},
		{"nodes", "get", &perms.CanGetNodes},
	}

	// Only check secrets/configmaps if allowed by config
	if cfg.Triage.AllowSecretsAccess {
		checks = append(checks,
			struct {
				resource string
				verb     string
				target   *bool
			}{"secrets", "get", &perms.CanGetSecrets},
			struct {
				resource string
				verb     string
				target   *bool
			}{"configmaps", "get", &perms.CanGetConfigMaps},
		)
	}

	// Run each permission check
	for _, check := range checks {
		cmd := exec.CommandContext(ctx, "kubectl",
			"--kubeconfig", cfg.Triage.Kubeconfig,
			"auth", "can-i", check.verb, check.resource)

		out, err := cmd.Output()
		if err != nil {
			// can-i returns non-zero exit code for "no", which is fine
			*check.target = false
			slog.Debug("permission check returned no",
				"cluster", cfg.Name,
				"verb", check.verb,
				"resource", check.resource)
		} else {
			*check.target = strings.TrimSpace(string(out)) == "yes"
			if *check.target {
				slog.Debug("permission check passed",
					"cluster", cfg.Name,
					"verb", check.verb,
					"resource", check.resource)
			}
		}
	}

	// Build warnings for missing permissions
	if !perms.CanGetPods {
		perms.Warnings = append(perms.Warnings, "cannot get pods")
	}
	if !perms.CanGetLogs {
		perms.Warnings = append(perms.Warnings, "cannot get pod logs")
	}
	if !perms.CanGetEvents {
		perms.Warnings = append(perms.Warnings, "cannot get events")
	}
	if !perms.CanGetNodes {
		perms.Warnings = append(perms.Warnings, "cannot get nodes (cluster-wide visibility limited)")
	}

	// Secrets access warnings (only if enabled but not available)
	if cfg.Triage.AllowSecretsAccess && !perms.CanGetSecrets {
		perms.Warnings = append(perms.Warnings,
			"secrets access enabled but RBAC denies it (Helm data unavailable)")
	}

	// Info message when secrets access is disabled
	if !cfg.Triage.AllowSecretsAccess {
		perms.Warnings = append(perms.Warnings,
			"secrets access disabled by config (set triage.allow_secrets_access=true for Helm debugging)")
	}

	// Log summary
	if len(perms.Warnings) > 0 {
		slog.Warn("cluster has permission warnings",
			"cluster", cfg.Name,
			"warnings", strings.Join(perms.Warnings, "; "))
	}

	if !perms.MinimumPermissionsMet() {
		slog.Error("cluster does not meet minimum permissions for triage",
			"cluster", cfg.Name,
			"missing", strings.Join(perms.Warnings, "; "))
	} else {
		slog.Info("cluster permissions validated successfully",
			"cluster", cfg.Name,
			"pods", perms.CanGetPods,
			"logs", perms.CanGetLogs,
			"events", perms.CanGetEvents)
	}

	return perms, nil
}
