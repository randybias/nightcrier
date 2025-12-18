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
	"github.com/spf13/cobra"
)

var (
	mcpEndpoint   string
	workspaceRoot string
	scriptPath    string
	logLevel      string
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
	rootCmd.Flags().StringVar(&mcpEndpoint, "mcp-endpoint", "", "MCP server endpoint URL (overrides K8S_CLUSTER_MCP_ENDPOINT env var)")
	rootCmd.Flags().StringVar(&workspaceRoot, "workspace-root", "", "Workspace root directory (overrides WORKSPACE_ROOT env var, default: ./incidents)")
	rootCmd.Flags().StringVar(&scriptPath, "script-path", "", "Path to agent script (default: ./scripts/stub-agent.sh)")
	rootCmd.Flags().StringVar(&logLevel, "log-level", "", "Log level (overrides LOG_LEVEL env var, default: info)")
}

func run(cmd *cobra.Command, args []string) error {
	// Override environment variables with flags if provided
	if mcpEndpoint != "" {
		os.Setenv("K8S_CLUSTER_MCP_ENDPOINT", mcpEndpoint)
	}
	if workspaceRoot != "" {
		os.Setenv("WORKSPACE_ROOT", workspaceRoot)
	}
	if logLevel != "" {
		os.Setenv("LOG_LEVEL", logLevel)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Setup structured logging
	setupLogging(cfg.LogLevel)

	slog.Info("starting kubernetes-mcp-alerts-event-runner",
		"mcp_endpoint", cfg.MCPEndpoint,
		"workspace_root", cfg.WorkspaceRoot,
		"log_level", cfg.LogLevel,
		"agent_script", cfg.AgentScriptPath,
		"agent_model", cfg.AgentModel,
		"agent_timeout", cfg.AgentTimeout)

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
	mcpClient := events.NewClient(cfg.MCPEndpoint)
	workspaceMgr := agent.NewWorkspaceManager(cfg.WorkspaceRoot)
	executor := agent.NewExecutorWithConfig(agent.ExecutorConfig{
		ScriptPath:       agentScript,
		SystemPromptFile: cfg.AgentSystemPromptFile,
		AllowedTools:     cfg.AgentAllowedTools,
		Model:            cfg.AgentModel,
		Timeout:          cfg.AgentTimeout,
	})

	// Create Slack notifier (optional - only if webhook URL configured)
	var slackNotifier *reporting.SlackNotifier
	if cfg.SlackWebhookURL != "" {
		slackNotifier = reporting.NewSlackNotifier(cfg.SlackWebhookURL)
		slog.Info("slack notifications enabled")
	}

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
			if err := processEvent(ctx, event, workspaceMgr, executor, slackNotifier); err != nil {
				slog.Error("failed to process event", "error", err)
			}
		}
	}
}

func processEvent(ctx context.Context, event *events.FaultEvent, workspaceMgr *agent.WorkspaceManager, executor *agent.Executor, slackNotifier *reporting.SlackNotifier) error {
	incidentID := uuid.New().String()
	startedAt := time.Now()

	slog.Info("processing fault event",
		"incident_id", incidentID,
		"cluster", event.Cluster,
		"namespace", event.GetNamespace(),
		"resource", fmt.Sprintf("%s/%s", event.GetResourceKind(), event.GetResourceName()),
		"reason", event.Event.Reason,
		"severity", event.GetSeverity())

	// Create workspace
	workspacePath, err := workspaceMgr.Create(incidentID)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}
	slog.Info("created workspace", "path", workspacePath)

	// Write event context
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
			IncidentID:  incidentID,
			Cluster:     event.Cluster,
			Namespace:   event.GetNamespace(),
			Resource:    fmt.Sprintf("%s/%s", event.GetResourceKind(), event.GetResourceName()),
			Reason:      event.Event.Reason,
			Status:      status,
			RootCause:   rootCause,
			Confidence:  confidence,
			Duration:    duration,
			ReportPath:  filepath.Join(workspacePath, "output", "investigation.md"),
		}

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
