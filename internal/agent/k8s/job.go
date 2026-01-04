// Package k8s provides Kubernetes client initialization and utilities for the Nightcrier agent.
package k8s

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// JobConfig holds configuration for Job creation.
type JobConfig struct {
	// Namespace is the Kubernetes namespace where the Job will be created
	Namespace string

	// IncidentID is the unique identifier for the incident
	IncidentID string

	// ClusterName is the name of the cluster being investigated
	ClusterName string

	// Image is the container image to use (default: nc-agent-runner:latest)
	Image string

	// ImagePullPolicy is the image pull policy (default: IfNotPresent)
	// Valid values: Always, Never, IfNotPresent
	ImagePullPolicy string

	// AgentCLI is the agent CLI to use (claude/codex/gemini/goose)
	AgentCLI string

	// LLMModel is the model to use for the agent
	LLMModel string

	// Prompt is the investigation prompt for the agent
	Prompt string

	// ConfigMapName is the name of the ConfigMap containing incident data
	ConfigMapName string

	// SecretName is the name of the Secret containing the kubeconfig
	SecretName string

	// PresignedURLs contains the presigned PUT URLs for outputs
	PresignedURLs PresignedURLs

	// Resources contains resource limits and requests
	Resources ResourceConfig

	// TTLSecondsAfterFinished is the TTL for cleanup after Job completion (default: 3600)
	TTLSecondsAfterFinished int32

	// ActiveDeadlineSeconds is the Job timeout in seconds (default: 600)
	ActiveDeadlineSeconds int64

	// BackoffLimit is the number of retries (default: 0 for triage)
	BackoffLimit int32

	// Labels are additional labels to apply to the Job
	Labels map[string]string

	// NATS configuration for progress tracking
	NATSEnabled bool
	NATSServer  string
	NATSToken   string
}

// PresignedURLs contains presigned PUT URLs for agent outputs.
type PresignedURLs struct {
	// Report is the URL for uploading report.md
	Report string

	// Log is the URL for uploading agent.log
	Log string

	// Session is the URL for uploading session.tar.gz
	Session string

	// Result is the URL for uploading result.json
	Result string

	// Commands is the URL for uploading commands-executed.log
	Commands string

	// PromptSent is the URL for uploading prompt-sent.md
	PromptSent string
}

// ResourceConfig contains resource limits and requests for the Job container.
type ResourceConfig struct {
	// MemoryLimit is the memory limit (default: 2Gi)
	MemoryLimit string

	// CPULimit is the CPU limit (default: 1)
	CPULimit string

	// MemoryRequest is the memory request (default: 512Mi)
	MemoryRequest string

	// CPURequest is the CPU request (default: 250m)
	CPURequest string
}

// CreateJob creates a Kubernetes Job for agent execution.
// The Job includes:
//   - Container with nc-agent-runner image
//   - Environment variables for agent configuration and API keys
//   - Volume mounts for ConfigMap (incident data) and Secret (kubeconfig)
//   - Resource limits and requests
//   - TTL for cleanup after completion
//   - activeDeadlineSeconds for job timeout
//   - backoffLimit=0 (no retries, triage is point-in-time)
//
// Labels applied:
//   - app=nc-agent-runner: Identifies Jobs managed by the agent executor
//   - incident-id={incidentID}: Links to specific incident for cleanup
//   - cluster={clusterName}: Links to target cluster
//
// Returns the created Job name on success.
func (c *Client) CreateJob(ctx context.Context, cfg JobConfig) (string, error) {
	// Apply defaults
	if cfg.Image == "" {
		cfg.Image = "nc-agent-runner:latest"
	}
	if cfg.ImagePullPolicy == "" {
		cfg.ImagePullPolicy = "IfNotPresent"
	}
	if cfg.TTLSecondsAfterFinished == 0 {
		cfg.TTLSecondsAfterFinished = 3600
	}
	if cfg.ActiveDeadlineSeconds == 0 {
		cfg.ActiveDeadlineSeconds = 600
	}
	if cfg.BackoffLimit == 0 {
		cfg.BackoffLimit = 0
	}
	if cfg.Resources.MemoryLimit == "" {
		cfg.Resources.MemoryLimit = "2Gi"
	}
	if cfg.Resources.CPULimit == "" {
		cfg.Resources.CPULimit = "1"
	}
	if cfg.Resources.MemoryRequest == "" {
		cfg.Resources.MemoryRequest = "512Mi"
	}
	if cfg.Resources.CPURequest == "" {
		cfg.Resources.CPURequest = "250m"
	}

	// Parse image pull policy
	var imagePullPolicy corev1.PullPolicy
	switch cfg.ImagePullPolicy {
	case "Always":
		imagePullPolicy = corev1.PullAlways
	case "Never":
		imagePullPolicy = corev1.PullNever
	case "IfNotPresent":
		imagePullPolicy = corev1.PullIfNotPresent
	default:
		return "", fmt.Errorf("invalid image pull policy: %s (must be Always, Never, or IfNotPresent)", cfg.ImagePullPolicy)
	}

	// Generate Job name based on incident ID
	jobName := fmt.Sprintf("triage-%s", cfg.IncidentID)

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

	// Build environment variables
	env := []corev1.EnvVar{
		{
			Name:  "AGENT_CLI",
			Value: cfg.AgentCLI,
		},
		{
			Name:  "LLM_MODEL",
			Value: cfg.LLMModel,
		},
		{
			Name:  "INCIDENT_ID",
			Value: cfg.IncidentID,
		},
		{
			Name:  "PROMPT",
			Value: cfg.Prompt,
		},
		// API keys from Secret
		{
			Name: "ANTHROPIC_API_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "ai-api-keys",
					},
					Key:      "anthropic",
					Optional: boolPtr(true),
				},
			},
		},
		{
			Name: "OPENAI_API_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "ai-api-keys",
					},
					Key:      "openai",
					Optional: boolPtr(true),
				},
			},
		},
		{
			Name: "GEMINI_API_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "ai-api-keys",
					},
					Key:      "gemini",
					Optional: boolPtr(true),
				},
			},
		},
		// Presigned PUT URLs for outputs
		{
			Name:  "OUTPUT_URL_REPORT",
			Value: cfg.PresignedURLs.Report,
		},
		{
			Name:  "OUTPUT_URL_LOG",
			Value: cfg.PresignedURLs.Log,
		},
		{
			Name:  "OUTPUT_URL_SESSION",
			Value: cfg.PresignedURLs.Session,
		},
		{
			Name:  "OUTPUT_URL_RESULT",
			Value: cfg.PresignedURLs.Result,
		},
		{
			Name:  "OUTPUT_URL_COMMANDS",
			Value: cfg.PresignedURLs.Commands,
		},
		{
			Name:  "OUTPUT_URL_PROMPT_SENT",
			Value: cfg.PresignedURLs.PromptSent,
		},
		// Cluster name for context
		{
			Name:  "CLUSTER",
			Value: cfg.ClusterName,
		},
	}

	// Add NATS configuration if enabled
	if cfg.NATSEnabled {
		env = append(env, []corev1.EnvVar{
			{
				Name:  "NATS_ENABLED",
				Value: "true",
			},
			{
				Name:  "NATS_SERVER",
				Value: cfg.NATSServer,
			},
			{
				Name:  "NATS_TOKEN",
				Value: cfg.NATSToken,
			},
		}...)
	}

	// Build volume mounts
	volumeMounts := []corev1.VolumeMount{
		// ConfigMap mounts (incident data)
		{
			Name:      "incident-data",
			MountPath: "/home/agent/incident.json",
			SubPath:   "incident.json",
			ReadOnly:  true,
		},
		{
			Name:      "incident-data",
			MountPath: "/home/agent/incident_cluster_permissions.json",
			SubPath:   "permissions.json",
			ReadOnly:  true,
		},
		{
			Name:      "incident-data",
			MountPath: "/home/agent/base-triage-prompt.md",
			SubPath:   "base-triage-prompt.md",
			ReadOnly:  true,
		},
		// Secret mount (kubeconfig)
		{
			Name:      "kubeconfig",
			MountPath: "/home/agent/.kube/config",
			SubPath:   "config",
			ReadOnly:  true,
		},
	}

	// Build volumes
	volumes := []corev1.Volume{
		{
			Name: "incident-data",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cfg.ConfigMapName,
					},
				},
			},
		},
		{
			Name: "kubeconfig",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: cfg.SecretName,
				},
			},
		},
	}

	// Parse resource quantities
	memoryLimit, err := resource.ParseQuantity(cfg.Resources.MemoryLimit)
	if err != nil {
		return "", fmt.Errorf("failed to parse memory limit: %w", err)
	}
	cpuLimit, err := resource.ParseQuantity(cfg.Resources.CPULimit)
	if err != nil {
		return "", fmt.Errorf("failed to parse CPU limit: %w", err)
	}
	memoryRequest, err := resource.ParseQuantity(cfg.Resources.MemoryRequest)
	if err != nil {
		return "", fmt.Errorf("failed to parse memory request: %w", err)
	}
	cpuRequest, err := resource.ParseQuantity(cfg.Resources.CPURequest)
	if err != nil {
		return "", fmt.Errorf("failed to parse CPU request: %w", err)
	}

	// Build container spec
	container := corev1.Container{
		Name:            "agent",
		Image:           cfg.Image,
		ImagePullPolicy: imagePullPolicy,
		Env:             env,
		VolumeMounts:    volumeMounts,
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: memoryLimit,
				corev1.ResourceCPU:    cpuLimit,
			},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: memoryRequest,
				corev1.ResourceCPU:    cpuRequest,
			},
		},
	}

	// Build Job spec
	restartPolicy := corev1.RestartPolicyNever
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: cfg.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: &cfg.TTLSecondsAfterFinished,
			ActiveDeadlineSeconds:   &cfg.ActiveDeadlineSeconds,
			BackoffLimit:            &cfg.BackoffLimit,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: restartPolicy,
					Containers:    []corev1.Container{container},
					Volumes:       volumes,
				},
			},
		},
	}

	// Create the Job
	createdJob, err := c.clientset.BatchV1().Jobs(cfg.Namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create Job %s: %w", jobName, err)
	}

	return createdJob.Name, nil
}

// CancelJob deletes a Kubernetes Job by name in the given namespace.
// It uses the DeletePropagationBackground policy to also clean up pods.
func (c *Client) CancelJob(ctx context.Context, namespace, jobName string) error {
	propagation := metav1.DeletePropagationBackground
	err := c.clientset.BatchV1().Jobs(namespace).Delete(ctx, jobName, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})
	if err != nil {
		return fmt.Errorf("failed to delete Job %s in namespace %s: %w", jobName, namespace, err)
	}
	return nil
}

// boolPtr returns a pointer to a bool value.
func boolPtr(b bool) *bool {
	return &b
}
