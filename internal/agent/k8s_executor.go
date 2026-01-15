package agent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/randybias/nightcrier/internal/agent/k8s"
	"github.com/randybias/nightcrier/internal/config"
	"github.com/randybias/nightcrier/internal/incident"
	"github.com/randybias/nightcrier/internal/storage"
)

// LogPaths contains artifact information from agent execution.
// For K8s executor, local file paths are empty but artifact URLs are populated.
type LogPaths struct {
	Stdout   string // Unused for K8s executor
	Stderr   string // Unused for K8s executor
	Combined string // Unused for K8s executor

	// Artifact information populated by K8s executor
	ReportURL  string // URL to the investigation report (HTML)
	RootCause  string // Root cause extracted from report
	Confidence string // Confidence level (HIGH/MEDIUM/LOW/UNKNOWN)
}

// K8sExecutorConfig holds configuration for the Kubernetes-native executor.
// This includes agent settings that are not cluster-specific.
type K8sExecutorConfig struct {
	// Agent CLI to use (claude/codex/gemini/goose)
	AgentCLI string
	// LLM model to use
	Model string
	// System prompt file path
	SystemPromptFile string
	// Additional operator-specified prompt content
	AdditionalPrompt string
	// Debug mode - includes session archive and separate stderr
	Debug bool
	// NATS configuration for progress tracking
	NATSEnabled bool
	NATSServer  string
	NATSToken   string
}

// K8sExecutor implements the executor interface using Kubernetes Jobs.
// It orchestrates the full Phase 1-5 flow:
//   - Phase 1: Generate presigned URLs and create ConfigMap/Job
//   - Phase 3: Watch Job for completion and retrieve results
//   - Phase 4: Process artifacts and update database
//   - Phase 5: Complete incident and cleanup
type K8sExecutor struct {
	config            K8sExecutorConfig
	executionClusters map[string]*config.ExecutionClusterConfig
	defaultCluster    string // Name of the default execution cluster
	k8sClient         *k8s.Client
	objectStore       *storage.ObjectStore
	stateStore        storage.StateStore
	storage           storage.Storage
	processor         *k8s.ArtifactProcessor
	tuning            *config.TuningConfig
}

// NewK8sExecutor creates a new Kubernetes-native executor.
// executionClusters is a map of cluster name to ExecutionClusterConfig.
// defaultClusterName specifies which cluster to use when no specific cluster is requested.
func NewK8sExecutor(
	cfg K8sExecutorConfig,
	executionClusters map[string]*config.ExecutionClusterConfig,
	defaultClusterName string,
	k8sClient *k8s.Client,
	objectStore *storage.ObjectStore,
	stateStore storage.StateStore,
	storageBackend storage.Storage,
	tuning *config.TuningConfig,
) *K8sExecutor {
	return &K8sExecutor{
		config:            cfg,
		executionClusters: executionClusters,
		defaultCluster:    defaultClusterName,
		k8sClient:         k8sClient,
		objectStore:       objectStore,
		stateStore:        stateStore,
		storage:           storageBackend,
		processor:         k8s.NewArtifactProcessor(objectStore, stateStore, storageBackend),
		tuning:            tuning,
	}
}

// SelectExecutionCluster returns the execution cluster configuration for the given name.
// If preferredName is empty, returns the default execution cluster.
// Returns an error if the cluster is not found or no clusters are configured.
func (e *K8sExecutor) SelectExecutionCluster(preferredName string) (*config.ExecutionClusterConfig, error) {
	if len(e.executionClusters) == 0 {
		return nil, fmt.Errorf("no execution clusters configured")
	}

	if preferredName != "" {
		if cluster, ok := e.executionClusters[preferredName]; ok {
			return cluster, nil
		}
		return nil, fmt.Errorf("execution cluster %q not found", preferredName)
	}

	// Return default cluster
	if e.defaultCluster != "" {
		if cluster, ok := e.executionClusters[e.defaultCluster]; ok {
			return cluster, nil
		}
		return nil, fmt.Errorf("default execution cluster %q not found", e.defaultCluster)
	}

	// Fallback: return first available cluster
	for _, cluster := range e.executionClusters {
		return cluster, nil
	}

	return nil, fmt.Errorf("no execution clusters configured")
}

// SetStateStore updates the stateStore reference in the executor and its processor.
// This is called after the stateStore is initialized in main.go.
func (e *K8sExecutor) SetStateStore(stateStore storage.StateStore) {
	e.stateStore = stateStore
	// Recreate processor with the new stateStore
	e.processor = k8s.NewArtifactProcessor(e.objectStore, stateStore, e.storage)
}

// Execute runs the agent in a Kubernetes Job and processes the results.
// This implements the full Phase 1-5 orchestration.
// Uses the default execution cluster.
func (e *K8sExecutor) Execute(ctx context.Context, workspacePath string, incidentID string) (int, LogPaths, error) {
	return e.ExecuteOnCluster(ctx, workspacePath, incidentID, "", "")
}

// ExecuteWithPrompt runs the agent with a custom prompt.
// Uses the default execution cluster.
func (e *K8sExecutor) ExecuteWithPrompt(ctx context.Context, workspacePath string, incidentID string, prompt string) (int, LogPaths, error) {
	return e.ExecuteOnCluster(ctx, workspacePath, incidentID, prompt, "")
}

// ExecuteOnCluster runs the agent on a specific execution cluster.
// If executionClusterName is empty, uses the default execution cluster.
// This is the main entry point that orchestrates all phases.
func (e *K8sExecutor) ExecuteOnCluster(ctx context.Context, workspacePath string, incidentID string, prompt string, executionClusterName string) (int, LogPaths, error) {
	// Select execution cluster
	execCluster, err := e.SelectExecutionCluster(executionClusterName)
	if err != nil {
		return -1, LogPaths{}, fmt.Errorf("failed to select execution cluster: %w", err)
	}

	slog.Info("executing agent in Kubernetes Job",
		"incident_id", incidentID,
		"execution_cluster", execCluster.Name,
		"namespace", execCluster.Namespace,
		"image", execCluster.RunnerImage,
		"timeout", execCluster.Timeout)

	executionID := uuid.New().String()
	startedAt := time.Now().UTC()

	// Phase 1.1: Generate presigned PUT URLs for outputs
	slog.Info("generating presigned URLs for outputs", "incident_id", incidentID)
	jobTimeout := time.Duration(execCluster.Timeout) * time.Second
	outputURLs, err := k8s.GenerateOutputURLs(ctx, e.objectStore, incidentID, jobTimeout)
	if err != nil {
		return -1, LogPaths{}, fmt.Errorf("failed to generate output URLs: %w", err)
	}
	slog.Debug("presigned URLs generated", "incident_id", incidentID, "expiry", outputURLs.ReportExpiry)

	// Phase 1.2: Load incident data and system prompt
	incidentData, err := e.loadIncidentData(workspacePath, incidentID)
	if err != nil {
		return -1, LogPaths{}, fmt.Errorf("failed to load incident data: %w", err)
	}

	// Phase 1.3: Create ConfigMap with incident data
	slog.Info("creating ConfigMap with incident data", "incident_id", incidentID)
	configMapName, err := e.createConfigMap(ctx, execCluster, incidentID, incidentData)
	if err != nil {
		return -1, LogPaths{}, fmt.Errorf("failed to create ConfigMap: %w", err)
	}
	slog.Debug("ConfigMap created", "name", configMapName)

	// Ensure ConfigMap cleanup on exit (unless Job cleanup handles it)
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := e.k8sClient.DeleteConfigMap(cleanupCtx, execCluster.Namespace, configMapName); err != nil {
			slog.Warn("failed to cleanup ConfigMap", "name", configMapName, "error", err)
		} else {
			slog.Debug("ConfigMap cleaned up", "name", configMapName)
		}
	}()

	// Phase 1.5: Create Job
	slog.Info("creating Kubernetes Job", "incident_id", incidentID)
	combinedPrompt := e.buildCombinedPrompt(prompt)
	jobName, err := e.createJob(ctx, execCluster, incidentID, incidentData.ClusterName, configMapName, outputURLs, combinedPrompt)
	if err != nil {
		return -1, LogPaths{}, fmt.Errorf("failed to create Job: %w", err)
	}
	slog.Info("Job created", "name", jobName, "incident_id", incidentID)

	// Record agent execution at Job creation time so NATS listener can update run_started_at
	// The processor will update this record with completion data when the Job finishes
	if e.stateStore != nil {
		initialExec := &storage.AgentExecution{
			ExecutionID: executionID,
			IncidentID:  incidentID,
			StartedAt:   startedAt,
			AgentCLI:    e.config.AgentCLI,
			AgentModel:  e.config.Model,
			ClusterName: incidentData.ClusterName,
		}
		if err := e.stateStore.RecordAgentExecution(ctx, initialExec); err != nil {
			slog.Warn("failed to record initial agent execution", "incident_id", incidentID, "error", err)
			// Don't fail - this is non-critical for the Job execution
		}
	}

	// Phase 3.1: Watch Job for completion
	slog.Info("watching Job for completion", "job", jobName, "timeout", execCluster.Timeout)
	watchResult, err := e.watchJob(ctx, execCluster, jobName)
	if err != nil {
		return -1, LogPaths{}, fmt.Errorf("failed to watch Job: %w", err)
	}

	slog.Info("Job completed",
		"job", jobName,
		"status", watchResult.Status,
		"message", watchResult.Message,
		"completion_time", watchResult.CompletionTime)

	// Phase 3.2: Retrieve artifacts from Object Store
	slog.Info("retrieving Job results from Object Store", "incident_id", incidentID)
	jobResults, err := e.retrieveResults(ctx, incidentID, e.config.Debug)
	if err != nil {
		return -1, LogPaths{}, fmt.Errorf("failed to retrieve Job results: %w", err)
	}

	// Check for missing artifacts
	if len(jobResults.Missing) > 0 {
		slog.Warn("some artifacts were not uploaded by the Job",
			"incident_id", incidentID,
			"missing", jobResults.Missing)
	}

	// Phase 4: Process Job results (convert markdown, save artifacts, update database)
	slog.Info("processing Job artifacts", "incident_id", incidentID)
	output, err := e.processResults(ctx, incidentID, executionID, startedAt, jobResults, incidentData)
	if err != nil {
		slog.Error("failed to process Job results", "incident_id", incidentID, "error", err)
		// Return exit code from result.json if available, otherwise -1
		exitCode := -1
		if jobResults.ResultJSON != nil {
			exitCode = jobResults.ResultJSON.ExitCode
		}
		return exitCode, LogPaths{}, fmt.Errorf("failed to process Job results: %w", err)
	}

	// Determine final exit code
	exitCode := 0
	if jobResults.ResultJSON != nil {
		exitCode = jobResults.ResultJSON.ExitCode
	}

	slog.Info("agent execution completed successfully",
		"incident_id", incidentID,
		"execution_id", executionID,
		"exit_code", exitCode,
		"report_url", output.ReportURL,
		"root_cause", output.RootCause,
		"confidence", output.Confidence)

	// Return LogPaths with artifact information for notifications
	// Log paths are empty since logs are in Object Store, not local files
	logPaths := LogPaths{
		ReportURL:  output.ReportURL,
		RootCause:  output.RootCause,
		Confidence: output.Confidence,
	}

	return exitCode, logPaths, nil
}

// IncidentData holds all data needed to create the ConfigMap and Job.
type IncidentData struct {
	IncidentJSON     string
	PermissionsJSON  string
	BaseTriagePrompt string
	AdditionalPrompt string
	ClusterName      string
}

// loadIncidentData reads incident data from the workspace.
func (e *K8sExecutor) loadIncidentData(workspacePath string, incidentID string) (*IncidentData, error) {
	data := &IncidentData{}

	// Read incident.json
	incidentPath := filepath.Join(workspacePath, "incident.json")
	incidentBytes, err := os.ReadFile(incidentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read incident.json: %w", err)
	}
	data.IncidentJSON = string(incidentBytes)

	// Parse incident to extract cluster name
	var inc incident.Incident
	if err := inc.UpdateFromFile(incidentPath); err != nil {
		return nil, fmt.Errorf("failed to parse incident.json: %w", err)
	}
	data.ClusterName = inc.Cluster

	// Read permissions.json (optional - may not exist if triage disabled)
	permsPath := filepath.Join(workspacePath, "incident_cluster_permissions.json")
	permsBytes, err := os.ReadFile(permsPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read permissions.json: %w", err)
		}
		// Create empty permissions JSON if file doesn't exist
		data.PermissionsJSON = "{}"
	} else {
		data.PermissionsJSON = string(permsBytes)
	}

	// Read base triage prompt
	if e.config.SystemPromptFile != "" {
		promptBytes, err := os.ReadFile(e.config.SystemPromptFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read base triage prompt: %w", err)
		}
		data.BaseTriagePrompt = string(promptBytes)
	}

	// Read additional prompt from config if provided
	if e.config.AdditionalPrompt != "" {
		data.AdditionalPrompt = e.config.AdditionalPrompt
	}

	// Debug: Log what data was loaded
	slog.Debug("incident data loaded",
		"incident_id", incidentID,
		"incident_json_size", len(data.IncidentJSON),
		"permissions_json_size", len(data.PermissionsJSON),
		"base_triage_prompt_size", len(data.BaseTriagePrompt),
		"additional_prompt_size", len(data.AdditionalPrompt),
		"cluster_name", data.ClusterName)

	return data, nil
}

// buildCombinedPrompt combines system prompt with additional prompt.
func (e *K8sExecutor) buildCombinedPrompt(additionalPrompt string) string {
	// The system prompt is already in the ConfigMap, so we only need to pass
	// the additional prompt to the Job. The agent container will combine them.
	return additionalPrompt
}

// createConfigMap creates a ConfigMap with incident data in the execution cluster's namespace.
func (e *K8sExecutor) createConfigMap(ctx context.Context, execCluster *config.ExecutionClusterConfig, incidentID string, data *IncidentData) (string, error) {
	cfg := k8s.ConfigMapConfig{
		Namespace:   execCluster.Namespace,
		IncidentID:  incidentID,
		ClusterName: data.ClusterName,
		Labels: map[string]string{
			"app":               "nc-agent-runner",
			"incident-id":       incidentID,
			"cluster":           data.ClusterName,
			"execution-cluster": execCluster.Name,
		},
	}

	cmData := k8s.ConfigMapData{
		IncidentJSON:     data.IncidentJSON,
		PermissionsJSON:  data.PermissionsJSON,
		BaseTriagePrompt: data.BaseTriagePrompt,
		AdditionalPrompt: data.AdditionalPrompt,
	}

	// Debug: Log ConfigMap data sizes
	slog.Debug("ConfigMap data prepared",
		"incident_id", incidentID,
		"execution_cluster", execCluster.Name,
		"incident_json_size", len(cmData.IncidentJSON),
		"permissions_json_size", len(cmData.PermissionsJSON),
		"base_triage_prompt_size", len(cmData.BaseTriagePrompt),
		"additional_prompt_size", len(cmData.AdditionalPrompt))

	return e.k8sClient.CreateIncidentConfigMap(ctx, cfg, cmData)
}

// createJob creates a Kubernetes Job for agent execution using the execution cluster's settings.
func (e *K8sExecutor) createJob(
	ctx context.Context,
	execCluster *config.ExecutionClusterConfig,
	incidentID string,
	clusterName string,
	configMapName string,
	outputURLs *k8s.OutputURLs,
	prompt string,
) (string, error) {
	cfg := k8s.JobConfig{
		Namespace:       execCluster.Namespace,
		IncidentID:      incidentID,
		ClusterName:     clusterName,
		Image:           execCluster.RunnerImage,
		ImagePullPolicy: execCluster.ImagePullPolicy,
		AgentCLI:        e.config.AgentCLI,
		LLMModel:        e.config.Model,
		Prompt:          prompt,
		ConfigMapName:   configMapName,
		SecretName:      "triage-kubeconfig-" + clusterName, // Must match bootstrap: triage-kubeconfig-{cluster-name}
		PresignedURLs:   outputURLs.ToPresignedURLs(),
		Resources: k8s.ResourceConfig{
			MemoryLimit:   execCluster.MemoryLimit,
			CPULimit:      execCluster.CPULimit,
			MemoryRequest: "512Mi", // Fixed for now
			CPURequest:    "250m",  // Fixed for now
		},
		TTLSecondsAfterFinished: int32(execCluster.CleanupTTL),
		ActiveDeadlineSeconds:   int64(execCluster.Timeout),
		BackoffLimit:            0, // No retries for triage
		Labels: map[string]string{
			"app":               "nc-agent-runner",
			"incident-id":       incidentID,
			"cluster":           clusterName,
			"execution-cluster": execCluster.Name,
		},
		NATSEnabled: e.config.NATSEnabled,
		NATSServer:  e.config.NATSServer,
		NATSToken:   e.config.NATSToken,
	}

	return e.k8sClient.CreateJob(ctx, cfg)
}

// watchJob watches a Job until completion using the execution cluster's timeout.
func (e *K8sExecutor) watchJob(ctx context.Context, execCluster *config.ExecutionClusterConfig, jobName string) (*k8s.JobWatchResult, error) {
	// Add buffer to watch timeout (should be longer than Job timeout)
	watchTimeout := time.Duration(execCluster.Timeout+e.tuning.Agent.TimeoutBufferSeconds) * time.Second

	cfg := k8s.WatchJobConfig{
		Namespace: execCluster.Namespace,
		JobName:   jobName,
		Timeout:   watchTimeout,
		LogFunc: func(message string) {
			slog.Info("job watch event", "job", jobName, "message", message)
		},
	}

	return e.k8sClient.WatchJob(ctx, cfg)
}

// retrieveResults downloads artifacts from Object Store.
func (e *K8sExecutor) retrieveResults(ctx context.Context, incidentID string, debug bool) (*k8s.JobResults, error) {
	// Use adapter to make ObjectStore compatible with RetrieveResults
	adapter := k8s.NewBlobObjectStoreAdapter(e.objectStore.Bucket())

	cfg := k8s.RetrieveResultsConfig{
		IncidentID:            incidentID,
		ObjectStore:           adapter,
		IncludeSessionArchive: debug, // Include session archive in debug mode
	}

	return k8s.RetrieveResults(ctx, cfg)
}

// processResults processes Job artifacts and updates the database.
// Returns the artifact information (reportURL, rootCause, confidence) for notifications.
func (e *K8sExecutor) processResults(
	ctx context.Context,
	incidentID string,
	executionID string,
	startedAt time.Time,
	jobResults *k8s.JobResults,
	incidentData *IncidentData,
) (*k8s.ProcessJobResultsOutput, error) {
	cfg := k8s.ProcessJobResultsConfig{
		IncidentID:      incidentID,
		ExecutionID:     executionID,
		JobResults:      jobResults,
		StartedAt:       startedAt,
		IncidentJSON:    []byte(incidentData.IncidentJSON),
		PermissionsJSON: []byte(incidentData.PermissionsJSON),
		AgentCLI:        e.config.AgentCLI,
		AgentModel:      e.config.Model,
		ClusterName:     incidentData.ClusterName,
		Debug:           e.config.Debug,
	}

	return e.processor.ProcessJobResults(ctx, cfg)
}

