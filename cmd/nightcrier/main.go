package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/rbias/nightcrier/internal/agent"
	"github.com/rbias/nightcrier/internal/cluster"
	"github.com/rbias/nightcrier/internal/config"
	"github.com/rbias/nightcrier/internal/events"
	"github.com/rbias/nightcrier/internal/health"
	"github.com/rbias/nightcrier/internal/incident"
	"github.com/rbias/nightcrier/internal/reporting"
	"github.com/rbias/nightcrier/internal/storage"
	"github.com/spf13/cobra"
)

var (
	// Version information (set via ldflags at build time)
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"

	// Command-line flags
	configFile    string
	mcpEndpoint   string
	workspaceRoot string
	scriptPath    string
	logLevel      string
	agentTimeout  int
	healthPort    int
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "nightcrier",
	Short: "Nightcrier - Kubernetes Incident Triage",
	Long:  "MCP client that listens for fault events from kubernetes-mcp-server and spawns AI agents to triage them",
	RunE:  run,
}

func init() {
	// Version flag
	rootCmd.Flags().BoolP("version", "v", false, "Print version information and exit")

	// Configuration file flag
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to config file (default: searches for config.yaml in ., ./configs, /etc/nightcrier)")

	// Override flags (take precedence over config file and env vars)
	rootCmd.Flags().StringVar(&mcpEndpoint, "mcp-endpoint", "", "MCP server endpoint URL (overrides config file and K8S_CLUSTER_MCP_ENDPOINT env var)")
	rootCmd.Flags().StringVar(&workspaceRoot, "workspace-root", "", "Workspace root directory (overrides config file and WORKSPACE_ROOT env var)")
	rootCmd.Flags().StringVar(&scriptPath, "script-path", "", "Path to agent script")
	rootCmd.Flags().StringVar(&logLevel, "log-level", "", "Log level: debug, info, warn, error (overrides config file and LOG_LEVEL env var)")
	rootCmd.Flags().IntVar(&agentTimeout, "agent-timeout", 0, "Agent execution timeout in seconds (overrides config file and AGENT_TIMEOUT env var)")

	// Health monitoring flags
	rootCmd.Flags().IntVar(&healthPort, "health-port", 8080, "Port for health monitoring HTTP endpoint (0 to disable)")

	// Bind flags to viper for precedence handling
	config.BindFlags(rootCmd.Flags())
}

func run(cmd *cobra.Command, args []string) error {
	// Handle --version flag
	versionFlag, _ := cmd.Flags().GetBool("version")
	if versionFlag {
		fmt.Printf("nightcrier version %s\n", Version)
		fmt.Printf("  Build Time: %s\n", BuildTime)
		fmt.Printf("  Git Commit: %s\n", GitCommit)
		return nil
	}

	// Load configuration with precedence: flags > env vars > config file > defaults
	cfg, err := config.LoadWithConfigFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Load tuning configuration (optional - uses defaults if not found)
	tuning, err := config.LoadTuning()
	if err != nil {
		return fmt.Errorf("failed to load tuning configuration: %w", err)
	}

	// Setup structured logging
	setupLogging(cfg.LogLevel)
	slog.Info("tuning configuration loaded")

	// Print startup banner
	printStartupBanner(cfg, config.GetConfigFile())

	// Determine script path (CLI flag overrides config)
	agentScript := scriptPath
	if agentScript == "" {
		agentScript = cfg.AgentScriptPath
	}

	// Verify script exists
	if _, err := os.Stat(agentScript); os.IsNotExist(err) {
		return fmt.Errorf("agent script not found: %s", agentScript)
	}

	// Verify system prompt file exists
	if _, err := os.Stat(cfg.AgentSystemPromptFile); os.IsNotExist(err) {
		slog.Warn("system prompt file not found, will run without it", "path", cfg.AgentSystemPromptFile)
		cfg.AgentSystemPromptFile = ""
	}

	// Create ConnectionManager for multi-cluster support
	mgrConfig := &cluster.ManagerConfig{
		Clusters:                   cfg.Clusters,
		SubscribeMode:              cfg.SubscribeMode,
		GlobalQueueSize:            cfg.GlobalQueueSize,
		QueueOverflowPolicy:        cfg.QueueOverflowPolicy,
		SSEReconnectInitialBackoff: cfg.SSEReconnectInitialBackoff,
	}
	connectionMgr, err := cluster.NewConnectionManager(mgrConfig)
	if err != nil {
		return fmt.Errorf("failed to create connection manager: %w", err)
	}

	// Create and inject MCP clients for each cluster
	for _, clusterCfg := range cfg.Clusters {
		mcpClient := events.NewClient(clusterCfg.MCP.Endpoint, cfg.SubscribeMode, tuning)
		if err := connectionMgr.SetClusterClient(clusterCfg.Name, mcpClient); err != nil {
			return fmt.Errorf("failed to set client for cluster %s: %w", clusterCfg.Name, err)
		}
		slog.Info("mcp client created for cluster",
			"cluster", clusterCfg.Name,
			"endpoint", clusterCfg.MCP.Endpoint)
	}

	workspaceMgr := agent.NewWorkspaceManager(cfg.WorkspaceRoot)

	// Create executors per cluster (each cluster has its own kubeconfig)
	executors := make(map[string]*agent.Executor)
	for _, clusterCfg := range cfg.Clusters {
		executors[clusterCfg.Name] = agent.NewExecutorWithConfig(agent.ExecutorConfig{
			ScriptPath:       agentScript,
			SystemPromptFile: cfg.AgentSystemPromptFile,
			AllowedTools:     cfg.AgentAllowedTools,
			Model:            cfg.AgentModel,
			Timeout:          cfg.AgentTimeout,
			AgentCLI:         cfg.AgentCLI,
			AgentImage:       cfg.AgentImage,
			AdditionalPrompt: cfg.AdditionalAgentPrompt,
			Debug:            cfg.LogLevel == "debug",
			Verbose:          cfg.AgentVerbose || cfg.LogLevel == "debug",
			Kubeconfig:       clusterCfg.Triage.Kubeconfig,
		}, tuning)
		slog.Info("executor created for cluster",
			"cluster", clusterCfg.Name,
			"kubeconfig", clusterCfg.Triage.Kubeconfig)
	}

	// Create Slack notifier (optional - only if webhook URL configured)
	var slackNotifier *reporting.SlackNotifier
	if cfg.SlackWebhookURL != "" {
		slackNotifier = reporting.NewSlackNotifier(cfg.SlackWebhookURL, tuning)
		slog.Info("slack notifications enabled")
	}

	// Create circuit breaker with configured threshold
	circuitBreaker := reporting.NewCircuitBreaker(cfg.FailureThresholdForAlert, tuning)
	slog.Info("circuit breaker initialized", "threshold", cfg.FailureThresholdForAlert)

	// Initialize storage backend based on configuration
	storageBackend, err := storage.NewStorage(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize storage backend: %w", err)
	}
	storageMode := "filesystem"
	if cfg.IsAzureStorageEnabled() {
		storageMode = "azure"
	}
	slog.Info("storage backend initialized", "mode", storageMode)

	// Setup context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		slog.Info("received shutdown signal", "signal", sig)
		cancel()
	}()

	// Phase 3: Initialize connection manager (validates cluster permissions)
	// This runs kubectl auth can-i checks for all clusters with triage enabled
	slog.Info("initializing connection manager - validating permissions")
	initCtx, initCancel := context.WithTimeout(ctx, 30*time.Second)
	defer initCancel()
	if err := connectionMgr.Initialize(initCtx); err != nil {
		return fmt.Errorf("failed to initialize connection manager: %w", err)
	}

	// Phase 4: Start health monitoring server if enabled
	if healthPort > 0 {
		healthServer := health.NewServer(connectionMgr, healthPort)
		go func() {
			slog.Info("starting health monitoring server",
				"port", healthPort,
				"endpoint", fmt.Sprintf("http://localhost:%d/health/clusters", healthPort))
			if err := healthServer.Start(); err != nil && err != http.ErrServerClosed {
				slog.Error("health server failed", "error", err)
			}
		}()
	} else {
		slog.Info("health monitoring server disabled", "reason", "health-port=0")
	}

	// Start the ConnectionManager and get event channel
	eventChan := connectionMgr.Start(ctx)
	defer connectionMgr.Stop()

	slog.Info("connection manager started, processing events",
		"cluster_count", len(cfg.Clusters))

	// Event processing loop
	for {
		select {
		case <-ctx.Done():
			slog.Info("shutting down...")
			return nil

		case event, ok := <-eventChan:
			if !ok {
				slog.Info("event channel closed")
				return nil
			}

			// Type assert event from interface{} to map[string]interface{}
			clusterEvent, ok := event.(map[string]interface{})
			if !ok {
				slog.Error("invalid event type received", "type", fmt.Sprintf("%T", event))
				continue
			}

			// Extract cluster context
			clusterName, ok := clusterEvent["ClusterName"].(string)
			if !ok {
				slog.Error("missing or invalid ClusterName in event")
				continue
			}

			kubeconfig, ok := clusterEvent["Kubeconfig"].(string)
			if !ok {
				slog.Error("missing or invalid Kubeconfig in event", "cluster", clusterName)
				continue
			}

			// Phase 3: Extract cluster permissions (may be nil if triage disabled)
			permissions, _ := clusterEvent["Permissions"].(*cluster.ClusterPermissions)

			// Extract the FaultEvent
			faultEvent, ok := clusterEvent["Event"].(*events.FaultEvent)
			if !ok {
				slog.Error("missing or invalid Event in cluster event",
					"cluster", clusterName,
					"type", fmt.Sprintf("%T", clusterEvent["Event"]))
				continue
			}

			// Get the executor for this cluster
			executor, ok := executors[clusterName]
			if !ok {
				slog.Error("no executor found for cluster", "cluster", clusterName)
				continue
			}

			// Process the event with cluster context (including permissions)
			if err := processEvent(ctx, faultEvent, clusterName, kubeconfig, permissions, workspaceMgr, executor, slackNotifier, storageBackend, circuitBreaker, cfg, tuning); err != nil {
				slog.Error("failed to process event",
					"cluster", clusterName,
					"fault_id", faultEvent.FaultID,
					"error", err)
			}
		}
	}

	return nil
}

func processEvent(ctx context.Context, event *events.FaultEvent, clusterName string, kubeconfig string, permissions *cluster.ClusterPermissions, workspaceMgr *agent.WorkspaceManager, executor *agent.Executor, slackNotifier *reporting.SlackNotifier, storageBackend storage.Storage, circuitBreaker *reporting.CircuitBreaker, cfg *config.Config, tuning *config.TuningConfig) error {
	// Create incident from event
	incidentID := uuid.New().String()
	inc := incident.NewFromEvent(incidentID, event)

	// Override cluster name with the one from ClusterEvent (Phase 2: multi-cluster support)
	inc.Cluster = clusterName

	slog.Info("processing fault event",
		"incident_id", incidentID,
		"fault_id", event.FaultID,
		"cluster", clusterName,
		"namespace", event.GetNamespace(),
		"resource", fmt.Sprintf("%s/%s", event.GetResourceKind(), event.GetResourceName()),
		"reason", event.GetReason(),
		"severity", event.GetSeverity())

	// Phase 3: Check if triage is enabled for this cluster
	// If permissions are nil, triage is disabled (triage.enabled=false in config)
	if permissions == nil {
		slog.Info("triage disabled for cluster - skipping agent execution",
			"incident_id", incidentID,
			"cluster", clusterName,
			"reason", "triage.enabled=false or no kubeconfig")
		// Event is logged but no investigation is performed
		return nil
	}

	// Phase 3: Check if cluster has minimum permissions for triage
	if !permissions.MinimumPermissionsMet() {
		slog.Warn("cluster has insufficient permissions for triage - proceeding anyway",
			"incident_id", incidentID,
			"cluster", clusterName,
			"warnings", permissions.Warnings)
		// We log a warning but still attempt triage - agent will see limited permissions
	}

	// Create workspace
	workspacePath, err := workspaceMgr.Create(incidentID)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}
	slog.Info("created workspace", "path", workspacePath)

	// Write incident.json with investigating status
	incidentPath := filepath.Join(workspacePath, "incident.json")
	if err := inc.WriteToFile(incidentPath); err != nil {
		return fmt.Errorf("failed to write incident context: %w", err)
	}

	// Phase 3: Write incident_cluster_permissions.json if permissions are available
	// This informs the agent about what cluster access it has
	if permissions != nil {
		permsPath := filepath.Join(workspacePath, "incident_cluster_permissions.json")
		permsFile, err := os.Create(permsPath)
		if err != nil {
			return fmt.Errorf("failed to create permissions file: %w", err)
		}
		defer permsFile.Close()

		encoder := json.NewEncoder(permsFile)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(permissions); err != nil {
			return fmt.Errorf("failed to write permissions file: %w", err)
		}
		slog.Info("wrote cluster permissions to workspace",
			"path", permsPath,
			"cluster", clusterName,
			"minimum_met", permissions.MinimumPermissionsMet())
	} else {
		slog.Info("no cluster permissions available (triage may be disabled)",
			"cluster", clusterName)
	}

	// Mark agent start time
	startedAt := time.Now()
	inc.StartedAt = &startedAt

	// Execute agent
	exitCode, logPaths, execErr := executor.Execute(ctx, workspacePath, incidentID)

	// Update incident with completion info
	inc.MarkCompleted(exitCode, execErr)

	// Populate log paths in incident for local reference
	inc.LogPaths = map[string]string{
		"agent-stdout.log": logPaths.Stdout,
		"agent-stderr.log": logPaths.Stderr,
		"agent-full.log":   logPaths.Combined,
	}

	// Detect agent failures (exit code 0 but missing or invalid output)
	agentFailed, failureReason := detectAgentFailure(workspacePath, exitCode, execErr, tuning)
	if agentFailed {
		inc.Status = incident.StatusAgentFailed
		inc.FailureReason = failureReason
		slog.Warn("agent execution failed validation",
			"incident_id", incidentID,
			"reason", failureReason)

		// Record failure in circuit breaker
		circuitBreaker.RecordFailure(failureReason)
		slog.Debug("circuit breaker: recorded failure",
			"failure_count", circuitBreaker.GetFailureCount(),
			"state", circuitBreaker.GetState())

		// Check if we should send a system degraded alert
		if circuitBreaker.ShouldAlert() {
			stats := circuitBreaker.GetStats()
			slog.Warn("circuit breaker threshold reached, system degraded",
				"failure_count", stats.Count,
				"duration", stats.Duration,
				"recent_reasons", stats.RecentReasons)

			// Send system degraded alert to Slack if configured and enabled
			if slackNotifier != nil && cfg.NotifyOnAgentFailure {
				if err := slackNotifier.SendSystemDegradedAlert(ctx, stats); err != nil {
					slog.Error("failed to send system degraded alert", "error", err)
				} else {
					slog.Info("system degraded alert sent to slack",
						"failure_count", stats.Count,
						"duration", stats.Duration)
				}
			} else {
				if slackNotifier == nil {
					slog.Debug("slack not configured, skipping system degraded alert")
				} else {
					slog.Debug("system degraded alert disabled by configuration",
						"config", "notify_on_agent_failure=false")
				}
			}
		}
	} else {
		// Record success in circuit breaker and get stats before reset
		stats := circuitBreaker.GetStats()
		needsRecoveryAlert := circuitBreaker.RecordSuccess()
		slog.Debug("circuit breaker: recorded success",
			"needs_recovery_alert", needsRecoveryAlert)

		// Send recovery alert if needed
		if needsRecoveryAlert {
			slog.Info("circuit breaker recovered, system returned to healthy state",
				"total_failures", stats.Count,
				"total_downtime", stats.Duration)

			// Send system recovered alert to Slack if configured and enabled
			if slackNotifier != nil && cfg.NotifyOnAgentFailure {
				if err := slackNotifier.SendSystemRecoveredAlert(ctx, stats); err != nil {
					slog.Error("failed to send system recovered alert", "error", err)
				} else {
					slog.Info("system recovered alert sent to slack",
						"total_failures", stats.Count,
						"total_downtime", stats.Duration)
				}
			} else {
				if slackNotifier == nil {
					slog.Debug("slack not configured, skipping system recovered alert")
				} else {
					slog.Debug("system recovered alert disabled by configuration",
						"config", "notify_on_agent_failure=false")
				}
			}
		}
	}

	// Write updated incident.json with completion info
	if err := inc.WriteToFile(incidentPath); err != nil {
		return fmt.Errorf("failed to update incident: %w", err)
	}

	// Calculate duration
	duration := inc.CompletedAt.Sub(startedAt)

	// Save incident artifacts to storage
	var reportURL string
	if storageBackend != nil {
		// Skip storage upload for agent failures (missing/invalid output) unless configured otherwise
		if inc.Status == incident.StatusAgentFailed && !cfg.UploadFailedInvestigations {
			slog.Info("skipping storage upload due to agent failure",
				"incident_id", incidentID,
				"reason", inc.FailureReason,
				"config", "upload_failed_investigations=false")
		} else {
			// Read the generated artifacts and convert markdown to HTML
			artifacts, err := readIncidentArtifacts(workspacePath, incidentID, logPaths)
			if err != nil {
				slog.Warn("failed to read incident artifacts for storage", "error", err)
			} else {
				// Upload artifacts to storage (Azure or filesystem)
				saveResult, err := storageBackend.SaveIncident(ctx, incidentID, artifacts)
				if err != nil {
					slog.Error("failed to save incident to storage", "error", err)
				} else {
					reportURL = saveResult.ReportURL
					slog.Info("incident artifacts saved to storage",
						"incident_id", incidentID,
						"artifact_count", len(saveResult.ArtifactURLs),
						"log_url_count", len(saveResult.LogURLs),
						"report_url", reportURL)

					// Populate log URLs in incident from storage result
					inc.LogURLs = saveResult.LogURLs

					// Update incident.json with log URLs
					if err := inc.WriteToFile(incidentPath); err != nil {
						slog.Warn("failed to update incident.json with log URLs", "error", err)
					}
				}
			}
		}
	}

	slog.Info("event processed",
		"incident_id", incidentID,
		"status", inc.Status,
		"exit_code", exitCode,
		"duration", duration)

	// Send Slack notification if configured
	if slackNotifier != nil {
		// Always skip individual notifications for agent failures to prevent spam
		// Circuit breaker will send aggregated alerts if configured
		if inc.Status == incident.StatusAgentFailed {
			slog.Info("skipping slack notification due to agent failure",
				"incident_id", incidentID,
				"reason", inc.FailureReason,
				"note", "circuit breaker will send aggregated alert if threshold reached")
		} else {
			rootCause, confidence, err := reporting.ExtractSummaryFromReport(workspacePath)
			if err != nil {
				slog.Warn("failed to extract report summary for slack", "error", err)
				rootCause = "See investigation report"
				confidence = "UNKNOWN"
			}

			summary := &reporting.IncidentSummary{
				IncidentID: incidentID,
				Cluster:    inc.Cluster,
				Namespace:  inc.Namespace,
				Resource:   fmt.Sprintf("%s/%s", inc.Resource.Kind, inc.Resource.Name),
				Reason:     inc.FaultType,
				Status:     inc.Status,
				RootCause:  rootCause,
				Confidence: confidence,
				Duration:   duration,
				ReportPath: filepath.Join(workspacePath, "output", "investigation.md"),
				ReportURL:  reportURL,
			}

			slog.Info("sending slack notification",
				"incident_id", incidentID,
				"report_url", reportURL,
				"has_url", reportURL != "")

			if err := slackNotifier.SendIncidentNotification(summary); err != nil {
				slog.Error("failed to send slack notification", "error", err)
			} else {
				slog.Info("slack notification sent", "incident_id", incidentID)
			}
		}
	}

	return nil
}

// detectAgentFailure validates agent execution and returns whether the agent failed and a reason string.
// It checks:
// 1. Exit code is 0
// 2. output/investigation.md file exists
// 3. investigation.md file size meets minimum threshold from tuning config
//
// Returns (failed bool, reason string)
func detectAgentFailure(workspacePath string, exitCode int, err error, tuning *config.TuningConfig) (bool, string) {
	// Check if there was an execution error
	if err != nil {
		return true, fmt.Sprintf("agent execution error: %v", err)
	}

	// Check exit code
	if exitCode != 0 {
		return true, fmt.Sprintf("agent exited with non-zero code: %d", exitCode)
	}

	// Check if investigation.md exists
	investigationPath := filepath.Join(workspacePath, "output", "investigation.md")
	info, err := os.Stat(investigationPath)
	if err != nil {
		if os.IsNotExist(err) {
			return true, "investigation.md file not found"
		}
		return true, fmt.Sprintf("error checking investigation.md: %v", err)
	}

	// Check file size against tuning threshold
	minSize := int64(tuning.Agent.InvestigationMinSizeBytes)
	if info.Size() < minSize {
		return true, fmt.Sprintf("investigation.md too small: %d bytes (expected >= %d)", info.Size(), minSize)
	}

	// All checks passed
	return false, ""
}

func setupLogging(level string) {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})
	slog.SetDefault(slog.New(handler))
}

// readIncidentArtifacts reads the generated artifacts from the workspace for storage upload.
// It also converts the markdown report to HTML for better browser rendering.
// It reads agent logs if they exist.
func readIncidentArtifacts(workspacePath, incidentID string, logPaths agent.LogPaths) (*storage.IncidentArtifacts, error) {
	// Read incident.json
	incidentPath := filepath.Join(workspacePath, "incident.json")
	incidentJSON, err := os.ReadFile(incidentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read incident.json: %w", err)
	}

	// Read investigation.md
	investigationPath := filepath.Join(workspacePath, "output", "investigation.md")
	investigationMD, err := os.ReadFile(investigationPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read investigation.md: %w", err)
	}

	// Convert markdown to HTML for better browser rendering
	investigationHTML := reporting.ConvertMarkdownToHTML(investigationMD, incidentID)

	// Read agent logs if they exist (logs are optional)
	var agentLogs storage.AgentLogs

	// Read stdout log
	if logPaths.Stdout != "" {
		stdout, err := os.ReadFile(logPaths.Stdout)
		if err != nil {
			slog.Debug("failed to read agent stdout log (this is normal if logging disabled)",
				"path", logPaths.Stdout,
				"error", err)
		} else {
			agentLogs.Stdout = stdout
			slog.Debug("read agent stdout log",
				"path", logPaths.Stdout,
				"size", len(stdout))
		}
	}

	// Read stderr log
	if logPaths.Stderr != "" {
		stderr, err := os.ReadFile(logPaths.Stderr)
		if err != nil {
			slog.Debug("failed to read agent stderr log (this is normal if logging disabled)",
				"path", logPaths.Stderr,
				"error", err)
		} else {
			agentLogs.Stderr = stderr
			slog.Debug("read agent stderr log",
				"path", logPaths.Stderr,
				"size", len(stderr))
		}
	}

	// Read combined log
	if logPaths.Combined != "" {
		combined, err := os.ReadFile(logPaths.Combined)
		if err != nil {
			slog.Debug("failed to read agent combined log (this is normal if logging disabled)",
				"path", logPaths.Combined,
				"error", err)
		} else {
			agentLogs.Combined = combined
			slog.Debug("read agent combined log",
				"path", logPaths.Combined,
				"size", len(combined))
		}
	}

	// Read cluster permissions file (optional - only present if triage was enabled)
	var clusterPermissionsJSON []byte
	permissionsPath := filepath.Join(workspacePath, "incident_cluster_permissions.json")
	if permsData, err := os.ReadFile(permissionsPath); err != nil {
		slog.Debug("cluster permissions file not found (this is normal if triage disabled)",
			"path", permissionsPath,
			"error", err)
	} else {
		clusterPermissionsJSON = permsData
		slog.Debug("read cluster permissions file",
			"path", permissionsPath,
			"size", len(permsData))
	}

	// Read Claude Code session archive if present (DEBUG mode only)
	var claudeSessionArchive []byte
	sessionArchivePath := filepath.Join(workspacePath, "logs", "claude-session.tar.gz")
	if sessionData, err := os.ReadFile(sessionArchivePath); err != nil {
		slog.Debug("claude session archive not found (this is normal in production mode)",
			"path", sessionArchivePath,
			"error", err)
	} else {
		claudeSessionArchive = sessionData
		slog.Debug("read claude session archive",
			"path", sessionArchivePath,
			"size", len(sessionData))
	}

	// Read prompt-sent.md (optional - may not exist for older incidents)
	promptSentPath := filepath.Join(workspacePath, "prompt-sent.md")
	promptSent, err := os.ReadFile(promptSentPath)
	if err != nil {
		// prompt-sent.md is optional, log but don't fail
		slog.Debug("prompt-sent.md not found (optional artifact)", "path", promptSentPath)
		promptSent = nil
	}

	return &storage.IncidentArtifacts{
		IncidentJSON:           incidentJSON,
		InvestigationMD:        investigationMD,
		InvestigationHTML:      investigationHTML,
		ClusterPermissionsJSON: clusterPermissionsJSON,
		AgentLogs:              agentLogs,
		ClaudeSessionArchive:   claudeSessionArchive,
		PromptSent:             promptSent,
	}, nil
}

// printStartupBanner displays configuration summary at startup
func printStartupBanner(cfg *config.Config, configFile string) {
	// Determine storage mode
	storageMode := "filesystem"
	if cfg.IsAzureStorageEnabled() {
		storageMode = "azure"
	}

	// Determine slack status
	slackStatus := "disabled"
	if cfg.SlackWebhookURL != "" {
		slackStatus = "enabled"
	}

	// Mask sensitive values
	configSource := configFile
	if configSource == "" {
		configSource = "(defaults only)"
	}

	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║         Nightcrier - Kubernetes Incident Triage              ║")
	fmt.Printf("║         Version: %-45s║\n", truncateString(Version, 45))
	fmt.Printf("║         Built:   %-45s║\n", truncateString(BuildTime, 45))
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Config File:    %-45s ║\n", truncateString(configSource, 45))
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Clusters:       %-45s ║\n", fmt.Sprintf("%d configured", len(cfg.Clusters)))
	fmt.Printf("║  Subscribe Mode: %-45s ║\n", cfg.SubscribeMode)
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Agent CLI:      %-45s ║\n", cfg.AgentCLI)
	fmt.Printf("║  Agent Model:    %-45s ║\n", cfg.AgentModel)
	fmt.Printf("║  Agent Timeout:  %-45s ║\n", fmt.Sprintf("%ds", cfg.AgentTimeout))
	fmt.Printf("║  Allowed Tools:  %-45s ║\n", truncateString(cfg.AgentAllowedTools, 45))
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Workspace Root: %-45s ║\n", truncateString(cfg.WorkspaceRoot, 45))
	fmt.Printf("║  Storage Mode:   %-45s ║\n", storageMode)
	fmt.Printf("║  Slack:          %-45s ║\n", slackStatus)
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Log Level:      %-45s ║\n", cfg.LogLevel)
	fmt.Printf("║  Max Concurrent: %-45s ║\n", fmt.Sprintf("%d agents", cfg.MaxConcurrentAgents))
	fmt.Printf("║  Severity:       %-45s ║\n", cfg.SeverityThreshold)
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

// truncateString truncates a string to maxLen, adding "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
