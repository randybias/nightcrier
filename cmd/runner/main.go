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
	})

	// Create Slack notifier (optional - only if webhook URL configured)
	var slackNotifier *reporting.SlackNotifier
	if cfg.SlackWebhookURL != "" {
		slackNotifier = reporting.NewSlackNotifier(cfg.SlackWebhookURL)
		slog.Info("slack notifications enabled")
	}

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
			if err := processEvent(ctx, event, workspaceMgr, executor, slackNotifier, storageBackend); err != nil {
				slog.Error("failed to process event", "error", err)
			}
		}
	}
}

func processEvent(ctx context.Context, event *events.FaultEvent, workspaceMgr *agent.WorkspaceManager, executor *agent.Executor, slackNotifier *reporting.SlackNotifier, storageBackend storage.Storage) error {
	incidentID := uuid.New().String()
	startedAt := time.Now()

	// Set the incident ID on the event so it flows through to event.json and storage
	event.IncidentID = incidentID

	slog.Info("processing fault event",
		"incident_id", incidentID,
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

	// Write event context (now includes incidentId)
	if err := agent.WriteEventContext(workspacePath, event); err != nil {
		return fmt.Errorf("failed to write event context: %w", err)
	}

	// Execute agent
	exitCode, err := executor.Execute(ctx, workspacePath, incidentID)
	completedAt := time.Now()

	// Determine status
	status := "success"
	if err != nil {
		status = "error"
		slog.Error("agent execution error", "error", err)
	} else if exitCode != 0 {
		status = "failed"
	}

	// Write result
	result := &reporting.Result{
		IncidentID:  incidentID,
		ExitCode:    exitCode,
		StartedAt:   startedAt,
		CompletedAt: completedAt,
		Status:      status,
	}

	// Save incident artifacts to storage
	var reportURL string
	if storageBackend != nil {
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
				// Populate result with presigned URLs
				result.PresignedURLs = saveResult.ArtifactURLs
				if !saveResult.ExpiresAt.IsZero() {
					result.PresignedURLsExpireAt = &saveResult.ExpiresAt
				}
				reportURL = saveResult.ReportURL
				slog.Info("incident artifacts saved to storage",
					"incident_id", incidentID,
					"artifact_count", len(saveResult.ArtifactURLs),
					"report_url", reportURL)
			}
		}
	}

	if err := reporting.WriteResult(workspacePath, result); err != nil {
		return fmt.Errorf("failed to write result: %w", err)
	}

	duration := completedAt.Sub(startedAt)

	slog.Info("event processed",
		"incident_id", incidentID,
		"status", status,
		"exit_code", exitCode,
		"duration", duration)

	// Send Slack notification if configured
	if slackNotifier != nil {
		rootCause, confidence, err := reporting.ExtractSummaryFromReport(workspacePath)
		if err != nil {
			slog.Warn("failed to extract report summary for slack", "error", err)
			rootCause = "See investigation report"
			confidence = "UNKNOWN"
		}

		summary := &reporting.IncidentSummary{
			IncidentID: incidentID,
			Cluster:    event.Cluster,
			Namespace:  event.GetNamespace(),
			Resource:   fmt.Sprintf("%s/%s", event.GetResourceKind(), event.GetResourceName()),
			Reason:     event.GetReason(),
			Status:     status,
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

	return nil
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
	// Read event.json
	eventPath := filepath.Join(workspacePath, "event.json")
	eventJSON, err := os.ReadFile(eventPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read event.json: %w", err)
	}

	// Read result.json (will be updated after this function returns)
	resultPath := filepath.Join(workspacePath, "result.json")
	resultJSON, err := os.ReadFile(resultPath)
	if err != nil {
		// Result may not exist yet, use empty JSON
		resultJSON = []byte("{}")
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
		EventJSON:         eventJSON,
		ResultJSON:        resultJSON,
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
