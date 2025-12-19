package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/rbias/kubernetes-mcp-alerts-event-runner/internal/agent"
	"github.com/rbias/kubernetes-mcp-alerts-event-runner/internal/config"
	"github.com/rbias/kubernetes-mcp-alerts-event-runner/internal/events"
	"github.com/rbias/kubernetes-mcp-alerts-event-runner/internal/incident"
	"github.com/rbias/kubernetes-mcp-alerts-event-runner/internal/reporting"
	"github.com/rbias/kubernetes-mcp-alerts-event-runner/internal/storage"
	"github.com/spf13/cobra"
)

var (
	configFile    string
	mcpEndpoint   string
	workspaceRoot string
	scriptPath    string
	logLevel      string
	agentTimeout  int
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "runner",
	Short: "Kubernetes MCP Alerts Event Runner",
	Long:  "MCP client that listens for fault events from kubernetes-mcp-server and spawns AI agents to triage them",
	RunE:  run,
}

func init() {
	// Configuration file flag
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to config file (default: searches for config.yaml in ., ./configs, /etc/runner)")

	// Override flags (take precedence over config file and env vars)
	rootCmd.Flags().StringVar(&mcpEndpoint, "mcp-endpoint", "", "MCP server endpoint URL (overrides config file and K8S_CLUSTER_MCP_ENDPOINT env var)")
	rootCmd.Flags().StringVar(&workspaceRoot, "workspace-root", "", "Workspace root directory (overrides config file and WORKSPACE_ROOT env var)")
	rootCmd.Flags().StringVar(&scriptPath, "script-path", "", "Path to agent script")
	rootCmd.Flags().StringVar(&logLevel, "log-level", "", "Log level: debug, info, warn, error (overrides config file and LOG_LEVEL env var)")
	rootCmd.Flags().IntVar(&agentTimeout, "agent-timeout", 0, "Agent execution timeout in seconds (overrides config file and AGENT_TIMEOUT env var)")

	// Bind flags to viper for precedence handling
	config.BindFlags(rootCmd.Flags())
}

func run(cmd *cobra.Command, args []string) error {
	// Load configuration with precedence: flags > env vars > config file > defaults
	cfg, err := config.LoadWithConfigFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Setup structured logging
	setupLogging(cfg.LogLevel)

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

	// Create components
	mcpClient := events.NewClient(cfg.MCPEndpoint, cfg.SubscribeMode)
	workspaceMgr := agent.NewWorkspaceManager(cfg.WorkspaceRoot)
	executor := agent.NewExecutorWithConfig(agent.ExecutorConfig{
		ScriptPath:       agentScript,
		SystemPromptFile: cfg.AgentSystemPromptFile,
		AllowedTools:     cfg.AgentAllowedTools,
		Model:            cfg.AgentModel,
		Timeout:          cfg.AgentTimeout,
		AgentCLI:         cfg.AgentCLI,
		Prompt:           cfg.AgentPrompt,
	})

	// Create Slack notifier (optional - only if webhook URL configured)
	var slackNotifier *reporting.SlackNotifier
	if cfg.SlackWebhookURL != "" {
		slackNotifier = reporting.NewSlackNotifier(cfg.SlackWebhookURL)
		slog.Info("slack notifications enabled")
	}

	// Create circuit breaker with configured threshold
	circuitBreaker := reporting.NewCircuitBreaker(cfg.FailureThresholdForAlert)
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

	// Subscribe to fault events via MCP
	eventChan, err := mcpClient.Subscribe(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to MCP events: %w", err)
	}

	// Process events
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
			if err := processEvent(ctx, event, workspaceMgr, executor, slackNotifier, storageBackend, circuitBreaker, cfg); err != nil {
				slog.Error("failed to process event", "error", err)
			}
		}
	}
}

func processEvent(ctx context.Context, event *events.FaultEvent, workspaceMgr *agent.WorkspaceManager, executor *agent.Executor, slackNotifier *reporting.SlackNotifier, storageBackend storage.Storage, circuitBreaker *reporting.CircuitBreaker, cfg *config.Config) error {
	// Create incident from event
	incidentID := uuid.New().String()
	inc := incident.NewFromEvent(incidentID, event)

	slog.Info("processing fault event",
		"incident_id", incidentID,
		"event_id", event.EventID,
		"cluster", event.Cluster,
		"namespace", event.GetNamespace(),
		"resource", fmt.Sprintf("%s/%s", event.GetResourceKind(), event.GetResourceName()),
		"reason", event.GetReason(),
		"severity", event.GetSeverity())

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

	// Mark agent start time
	startedAt := time.Now()
	inc.StartedAt = &startedAt

	// Execute agent
	exitCode, execErr := executor.Execute(ctx, workspacePath, incidentID)

	// Update incident with completion info
	inc.MarkCompleted(exitCode, execErr)

	// Detect agent failures (exit code 0 but missing or invalid output)
	agentFailed, failureReason := detectAgentFailure(workspacePath, exitCode, execErr)
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
			artifacts, err := readIncidentArtifacts(workspacePath, incidentID)
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
						"report_url", reportURL)
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
// 3. investigation.md file size is > 100 bytes
//
// Returns (failed bool, reason string)
func detectAgentFailure(workspacePath string, exitCode int, err error) (bool, string) {
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

	// Check file size
	if info.Size() <= 100 {
		return true, fmt.Sprintf("investigation.md too small: %d bytes (expected > 100)", info.Size())
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
func readIncidentArtifacts(workspacePath, incidentID string) (*storage.IncidentArtifacts, error) {
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

	return &storage.IncidentArtifacts{
		IncidentJSON:      incidentJSON,
		InvestigationMD:   investigationMD,
		InvestigationHTML: investigationHTML,
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
	fmt.Println("║         Kubernetes MCP Alerts Event Runner                    ║")
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Config File:    %-45s ║\n", truncateString(configSource, 45))
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  MCP Endpoint:   %-45s ║\n", truncateString(cfg.MCPEndpoint, 45))
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
