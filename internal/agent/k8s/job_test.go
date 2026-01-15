package k8s

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateJob(t *testing.T) {
	// Create a fake clientset
	fakeClientset := fake.NewSimpleClientset()

	// Create a client with the fake clientset
	client := &Client{
		clientset: fakeClientset,
	}

	// Prepare test configuration
	testIncidentID := "test-incident-123"
	testClusterName := "test-cluster"
	testNamespace := "nightcrier"
	testImage := "nc-agent-runner:test"

	cfg := JobConfig{
		Namespace:     testNamespace,
		IncidentID:    testIncidentID,
		ClusterName:   testClusterName,
		Image:         testImage,
		AgentCLI:      "claude",
		LLMModel:      "claude-opus-4-5-20251101",
		Prompt:        "Investigate the pod failure",
		ConfigMapName: "incident-test-incident-123",
		SecretName:    "triage-kubeconfig-test-cluster",
		PresignedURLs: PresignedURLs{
			Report:   "https://storage.example.com/report.md",
			Log:      "https://storage.example.com/agent.log",
			Session:  "https://storage.example.com/session.tar.gz",
			Result:   "https://storage.example.com/result.json",
			Commands: "https://storage.example.com/commands-executed.log",
		},
		Resources: ResourceConfig{
			MemoryLimit:   "2Gi",
			CPULimit:      "1",
			MemoryRequest: "512Mi",
			CPURequest:    "250m",
		},
		TTLSecondsAfterFinished: 3600,
		ActiveDeadlineSeconds:   600,
		BackoffLimit:            0,
		Labels: map[string]string{
			"custom-label": "custom-value",
		},
	}

	// Test Job creation
	ctx := context.Background()
	jobName, err := client.CreateJob(ctx, cfg)
	if err != nil {
		t.Fatalf("CreateJob() failed: %v", err)
	}

	expectedName := "triage-test-incident-123"
	if jobName != expectedName {
		t.Errorf("CreateJob() returned name %s, want %s", jobName, expectedName)
	}

	// Verify Job was created
	createdJob, err := fakeClientset.BatchV1().Jobs(testNamespace).Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get created Job: %v", err)
	}

	// Verify name
	if createdJob.Name != expectedName {
		t.Errorf("Job name = %s, want %s", createdJob.Name, expectedName)
	}

	// Verify namespace
	if createdJob.Namespace != testNamespace {
		t.Errorf("Job namespace = %s, want %s", createdJob.Namespace, testNamespace)
	}

	// Verify labels
	expectedLabels := map[string]string{
		"app":          "nc-agent-runner",
		"incident-id":  testIncidentID,
		"cluster":      testClusterName,
		"custom-label": "custom-value",
	}
	for k, expectedValue := range expectedLabels {
		if actualValue, ok := createdJob.Labels[k]; !ok {
			t.Errorf("Job missing label %s", k)
		} else if actualValue != expectedValue {
			t.Errorf("Job label %s = %s, want %s", k, actualValue, expectedValue)
		}
	}

	// Verify TTLSecondsAfterFinished
	if createdJob.Spec.TTLSecondsAfterFinished == nil {
		t.Error("Job TTLSecondsAfterFinished is nil")
	} else if *createdJob.Spec.TTLSecondsAfterFinished != cfg.TTLSecondsAfterFinished {
		t.Errorf("Job TTLSecondsAfterFinished = %d, want %d", *createdJob.Spec.TTLSecondsAfterFinished, cfg.TTLSecondsAfterFinished)
	}

	// Verify ActiveDeadlineSeconds
	if createdJob.Spec.ActiveDeadlineSeconds == nil {
		t.Error("Job ActiveDeadlineSeconds is nil")
	} else if *createdJob.Spec.ActiveDeadlineSeconds != cfg.ActiveDeadlineSeconds {
		t.Errorf("Job ActiveDeadlineSeconds = %d, want %d", *createdJob.Spec.ActiveDeadlineSeconds, cfg.ActiveDeadlineSeconds)
	}

	// Verify BackoffLimit
	if createdJob.Spec.BackoffLimit == nil {
		t.Error("Job BackoffLimit is nil")
	} else if *createdJob.Spec.BackoffLimit != cfg.BackoffLimit {
		t.Errorf("Job BackoffLimit = %d, want %d", *createdJob.Spec.BackoffLimit, cfg.BackoffLimit)
	}

	// Verify RestartPolicy
	if createdJob.Spec.Template.Spec.RestartPolicy != "Never" {
		t.Errorf("Job RestartPolicy = %s, want Never", createdJob.Spec.Template.Spec.RestartPolicy)
	}

	// Verify container spec
	if len(createdJob.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("Job has %d containers, want 1", len(createdJob.Spec.Template.Spec.Containers))
	}
	container := createdJob.Spec.Template.Spec.Containers[0]

	// Verify container name
	if container.Name != "agent" {
		t.Errorf("Container name = %s, want agent", container.Name)
	}

	// Verify image
	if container.Image != testImage {
		t.Errorf("Container image = %s, want %s", container.Image, testImage)
	}

	// Verify environment variables
	expectedEnvVars := map[string]string{
		"AGENT_CLI":           "claude",
		"LLM_MODEL":           "claude-opus-4-5-20251101",
		"INCIDENT_ID":         testIncidentID,
		"PROMPT":              "Investigate the pod failure",
		"OUTPUT_URL_REPORT":   "https://storage.example.com/report.md",
		"OUTPUT_URL_LOG":      "https://storage.example.com/agent.log",
		"OUTPUT_URL_SESSION":  "https://storage.example.com/session.tar.gz",
		"OUTPUT_URL_RESULT":   "https://storage.example.com/result.json",
		"OUTPUT_URL_COMMANDS": "https://storage.example.com/commands-executed.log",
	}
	envMap := make(map[string]string)
	for _, env := range container.Env {
		if env.Value != "" {
			envMap[env.Name] = env.Value
		}
	}
	for k, expectedValue := range expectedEnvVars {
		if actualValue, ok := envMap[k]; !ok {
			t.Errorf("Container missing environment variable %s", k)
		} else if actualValue != expectedValue {
			t.Errorf("Container env %s = %s, want %s", k, actualValue, expectedValue)
		}
	}

	// Verify API key environment variables from Secret
	expectedSecretEnvVars := map[string]string{
		"ANTHROPIC_API_KEY": "anthropic",
		"OPENAI_API_KEY":    "openai",
		"GEMINI_API_KEY":    "gemini",
	}
	for _, env := range container.Env {
		if expectedKey, ok := expectedSecretEnvVars[env.Name]; ok {
			if env.ValueFrom == nil || env.ValueFrom.SecretKeyRef == nil {
				t.Errorf("Environment variable %s should reference a Secret", env.Name)
			} else {
				if env.ValueFrom.SecretKeyRef.Name != "ai-api-keys" {
					t.Errorf("Environment variable %s references Secret %s, want ai-api-keys", env.Name, env.ValueFrom.SecretKeyRef.Name)
				}
				if env.ValueFrom.SecretKeyRef.Key != expectedKey {
					t.Errorf("Environment variable %s references key %s, want %s", env.Name, env.ValueFrom.SecretKeyRef.Key, expectedKey)
				}
				if env.ValueFrom.SecretKeyRef.Optional == nil || !*env.ValueFrom.SecretKeyRef.Optional {
					t.Errorf("Environment variable %s should be optional", env.Name)
				}
			}
		}
	}

	// Verify volume mounts
	if len(container.VolumeMounts) != 5 {
		t.Errorf("Container has %d volume mounts, want 5", len(container.VolumeMounts))
	}
	expectedVolumeMounts := map[string]struct {
		volumeName string
		mountPath  string
		subPath    string
		readOnly   bool
	}{
		"incident.json": {
			volumeName: "incident-data",
			mountPath:  "/home/agent/incident.json",
			subPath:    "incident.json",
			readOnly:   true,
		},
		"permissions.json": {
			volumeName: "incident-data",
			mountPath:  "/home/agent/incident_cluster_permissions.json",
			subPath:    "permissions.json",
			readOnly:   true,
		},
		"base-triage-prompt.md": {
			volumeName: "incident-data",
			mountPath:  "/home/agent/base-triage-prompt.md",
			subPath:    "base-triage-prompt.md",
			readOnly:   true,
		},
		"additional-prompt.md": {
			volumeName: "incident-data",
			mountPath:  "/home/agent/additional-prompt.md",
			subPath:    "additional-prompt.md",
			readOnly:   true,
		},
		"config": {
			volumeName: "kubeconfig",
			mountPath:  "/home/agent/.kube/config",
			subPath:    "config",
			readOnly:   true,
		},
	}
	for _, mount := range container.VolumeMounts {
		key := mount.SubPath
		if expected, ok := expectedVolumeMounts[key]; ok {
			if mount.Name != expected.volumeName {
				t.Errorf("VolumeMount %s has name %s, want %s", key, mount.Name, expected.volumeName)
			}
			if mount.MountPath != expected.mountPath {
				t.Errorf("VolumeMount %s has mountPath %s, want %s", key, mount.MountPath, expected.mountPath)
			}
			if mount.SubPath != expected.subPath {
				t.Errorf("VolumeMount %s has subPath %s, want %s", key, mount.SubPath, expected.subPath)
			}
			if mount.ReadOnly != expected.readOnly {
				t.Errorf("VolumeMount %s has readOnly %v, want %v", key, mount.ReadOnly, expected.readOnly)
			}
		}
	}

	// Verify volumes
	if len(createdJob.Spec.Template.Spec.Volumes) != 2 {
		t.Errorf("Job has %d volumes, want 2", len(createdJob.Spec.Template.Spec.Volumes))
	}
	volumeMap := make(map[string]string)
	for _, vol := range createdJob.Spec.Template.Spec.Volumes {
		if vol.ConfigMap != nil {
			volumeMap[vol.Name] = vol.ConfigMap.Name
		} else if vol.Secret != nil {
			volumeMap[vol.Name] = vol.Secret.SecretName
		}
	}
	if volumeMap["incident-data"] != cfg.ConfigMapName {
		t.Errorf("Volume incident-data references ConfigMap %s, want %s", volumeMap["incident-data"], cfg.ConfigMapName)
	}
	if volumeMap["kubeconfig"] != cfg.SecretName {
		t.Errorf("Volume kubeconfig references Secret %s, want %s", volumeMap["kubeconfig"], cfg.SecretName)
	}

	// Verify resource limits and requests
	if container.Resources.Limits.Memory().String() != "2Gi" {
		t.Errorf("Container memory limit = %s, want 2Gi", container.Resources.Limits.Memory().String())
	}
	if container.Resources.Limits.Cpu().String() != "1" {
		t.Errorf("Container CPU limit = %s, want 1", container.Resources.Limits.Cpu().String())
	}
	if container.Resources.Requests.Memory().String() != "512Mi" {
		t.Errorf("Container memory request = %s, want 512Mi", container.Resources.Requests.Memory().String())
	}
	if container.Resources.Requests.Cpu().String() != "250m" {
		t.Errorf("Container CPU request = %s, want 250m", container.Resources.Requests.Cpu().String())
	}
}

func TestCreateJob_WithDefaults(t *testing.T) {
	// Create a fake clientset
	fakeClientset := fake.NewSimpleClientset()

	client := &Client{
		clientset: fakeClientset,
	}

	// Minimal configuration (testing defaults)
	cfg := JobConfig{
		Namespace:     "nightcrier",
		IncidentID:    "test-incident-456",
		ClusterName:   "test-cluster",
		AgentCLI:      "codex",
		LLMModel:      "gpt-4",
		Prompt:        "Test prompt",
		ConfigMapName: "incident-test-incident-456",
		SecretName:    "triage-kubeconfig-test-cluster",
		PresignedURLs: PresignedURLs{
			Report:   "https://storage.example.com/report.md",
			Log:      "https://storage.example.com/agent.log",
			Session:  "https://storage.example.com/session.tar.gz",
			Result:   "https://storage.example.com/result.json",
			Commands: "https://storage.example.com/commands-executed.log",
		},
		// No Image, Resources, TTL, etc. - should use defaults
	}

	ctx := context.Background()
	jobName, err := client.CreateJob(ctx, cfg)
	if err != nil {
		t.Fatalf("CreateJob() failed: %v", err)
	}

	// Verify Job was created
	createdJob, err := fakeClientset.BatchV1().Jobs("nightcrier").Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get created Job: %v", err)
	}

	// Verify default image
	if len(createdJob.Spec.Template.Spec.Containers) < 1 {
		t.Fatal("Job has no containers")
	}
	container := createdJob.Spec.Template.Spec.Containers[0]
	if container.Image != "nc-agent-runner:latest" {
		t.Errorf("Container image = %s, want nc-agent-runner:latest", container.Image)
	}

	// Verify default TTLSecondsAfterFinished
	if createdJob.Spec.TTLSecondsAfterFinished == nil {
		t.Error("Job TTLSecondsAfterFinished is nil")
	} else if *createdJob.Spec.TTLSecondsAfterFinished != 3600 {
		t.Errorf("Job TTLSecondsAfterFinished = %d, want 3600", *createdJob.Spec.TTLSecondsAfterFinished)
	}

	// Verify default ActiveDeadlineSeconds
	if createdJob.Spec.ActiveDeadlineSeconds == nil {
		t.Error("Job ActiveDeadlineSeconds is nil")
	} else if *createdJob.Spec.ActiveDeadlineSeconds != 600 {
		t.Errorf("Job ActiveDeadlineSeconds = %d, want 600", *createdJob.Spec.ActiveDeadlineSeconds)
	}

	// Verify default BackoffLimit
	if createdJob.Spec.BackoffLimit == nil {
		t.Error("Job BackoffLimit is nil")
	} else if *createdJob.Spec.BackoffLimit != 0 {
		t.Errorf("Job BackoffLimit = %d, want 0", *createdJob.Spec.BackoffLimit)
	}

	// Verify default resource limits and requests
	if container.Resources.Limits.Memory().String() != "2Gi" {
		t.Errorf("Container memory limit = %s, want 2Gi", container.Resources.Limits.Memory().String())
	}
	if container.Resources.Limits.Cpu().String() != "1" {
		t.Errorf("Container CPU limit = %s, want 1", container.Resources.Limits.Cpu().String())
	}
	if container.Resources.Requests.Memory().String() != "512Mi" {
		t.Errorf("Container memory request = %s, want 512Mi", container.Resources.Requests.Memory().String())
	}
	if container.Resources.Requests.Cpu().String() != "250m" {
		t.Errorf("Container CPU request = %s, want 250m", container.Resources.Requests.Cpu().String())
	}
}

func TestCreateJob_NoAdditionalLabels(t *testing.T) {
	// Create a fake clientset
	fakeClientset := fake.NewSimpleClientset()

	client := &Client{
		clientset: fakeClientset,
	}

	cfg := JobConfig{
		Namespace:     "nightcrier",
		IncidentID:    "test-incident-789",
		ClusterName:   "test-cluster-2",
		AgentCLI:      "gemini",
		LLMModel:      "gemini-pro",
		Prompt:        "Test prompt",
		ConfigMapName: "incident-test-incident-789",
		SecretName:    "triage-kubeconfig-test-cluster-2",
		PresignedURLs: PresignedURLs{
			Report:   "https://storage.example.com/report.md",
			Log:      "https://storage.example.com/agent.log",
			Session:  "https://storage.example.com/session.tar.gz",
			Result:   "https://storage.example.com/result.json",
			Commands: "https://storage.example.com/commands-executed.log",
		},
		// No additional labels
	}

	ctx := context.Background()
	jobName, err := client.CreateJob(ctx, cfg)
	if err != nil {
		t.Fatalf("CreateJob() failed: %v", err)
	}

	// Verify Job was created
	createdJob, err := fakeClientset.BatchV1().Jobs("nightcrier").Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get created Job: %v", err)
	}

	// Verify only default labels are present
	expectedLabels := map[string]string{
		"app":         "nc-agent-runner",
		"incident-id": "test-incident-789",
		"cluster":     "test-cluster-2",
	}
	if len(createdJob.Labels) != len(expectedLabels) {
		t.Errorf("Job has %d labels, want %d", len(createdJob.Labels), len(expectedLabels))
	}
	for k, expectedValue := range expectedLabels {
		if actualValue, ok := createdJob.Labels[k]; !ok {
			t.Errorf("Job missing label %s", k)
		} else if actualValue != expectedValue {
			t.Errorf("Job label %s = %s, want %s", k, actualValue, expectedValue)
		}
	}
}

func TestCreateJob_InvalidResourceQuantity(t *testing.T) {
	// Create a fake clientset
	fakeClientset := fake.NewSimpleClientset()

	client := &Client{
		clientset: fakeClientset,
	}

	cfg := JobConfig{
		Namespace:     "nightcrier",
		IncidentID:    "test-incident-invalid",
		ClusterName:   "test-cluster",
		AgentCLI:      "claude",
		LLMModel:      "claude-opus-4-5-20251101",
		Prompt:        "Test prompt",
		ConfigMapName: "incident-test-incident-invalid",
		SecretName:    "triage-kubeconfig-test-cluster",
		PresignedURLs: PresignedURLs{
			Report:   "https://storage.example.com/report.md",
			Log:      "https://storage.example.com/agent.log",
			Session:  "https://storage.example.com/session.tar.gz",
			Result:   "https://storage.example.com/result.json",
			Commands: "https://storage.example.com/commands-executed.log",
		},
		Resources: ResourceConfig{
			MemoryLimit:   "invalid-memory",
			CPULimit:      "1",
			MemoryRequest: "512Mi",
			CPURequest:    "250m",
		},
	}

	ctx := context.Background()
	_, err := client.CreateJob(ctx, cfg)
	if err == nil {
		t.Error("CreateJob() should fail with invalid memory limit")
	}
}

func TestCreateJob_AllAgentTypes(t *testing.T) {
	agentTypes := []struct {
		agentCLI string
		model    string
	}{
		{"claude", "claude-opus-4-5-20251101"},
		{"codex", "gpt-4"},
		{"gemini", "gemini-pro"},
		{"goose", "gpt-4"},
	}

	for _, tt := range agentTypes {
		t.Run(tt.agentCLI, func(t *testing.T) {
			// Create a fake clientset
			fakeClientset := fake.NewSimpleClientset()

			client := &Client{
				clientset: fakeClientset,
			}

			cfg := JobConfig{
				Namespace:     "nightcrier",
				IncidentID:    "test-incident-" + tt.agentCLI,
				ClusterName:   "test-cluster",
				AgentCLI:      tt.agentCLI,
				LLMModel:      tt.model,
				Prompt:        "Test prompt for " + tt.agentCLI,
				ConfigMapName: "incident-test-incident-" + tt.agentCLI,
				SecretName:    "triage-kubeconfig-test-cluster",
				PresignedURLs: PresignedURLs{
					Report:   "https://storage.example.com/report.md",
					Log:      "https://storage.example.com/agent.log",
					Session:  "https://storage.example.com/session.tar.gz",
					Result:   "https://storage.example.com/result.json",
					Commands: "https://storage.example.com/commands-executed.log",
				},
			}

			ctx := context.Background()
			jobName, err := client.CreateJob(ctx, cfg)
			if err != nil {
				t.Fatalf("CreateJob() failed for %s: %v", tt.agentCLI, err)
			}

			// Verify Job was created
			createdJob, err := fakeClientset.BatchV1().Jobs("nightcrier").Get(ctx, jobName, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Failed to get created Job for %s: %v", tt.agentCLI, err)
			}

			// Verify AGENT_CLI and LLM_MODEL environment variables
			container := createdJob.Spec.Template.Spec.Containers[0]
			envMap := make(map[string]string)
			for _, env := range container.Env {
				if env.Value != "" {
					envMap[env.Name] = env.Value
				}
			}

			if envMap["AGENT_CLI"] != tt.agentCLI {
				t.Errorf("AGENT_CLI = %s, want %s", envMap["AGENT_CLI"], tt.agentCLI)
			}
			if envMap["LLM_MODEL"] != tt.model {
				t.Errorf("LLM_MODEL = %s, want %s", envMap["LLM_MODEL"], tt.model)
			}
		})
	}
}
