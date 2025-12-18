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
	sseEndpoint   string
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
	Long:  "Listens to SSE events from kubernetes-mcp-server and processes alerts using Claude agents",
	RunE:  run,
}

func init() {
	rootCmd.Flags().StringVar(&sseEndpoint, "sse-endpoint", "", "SSE endpoint URL (overrides SSE_ENDPOINT env var)")
	rootCmd.Flags().StringVar(&workspaceRoot, "workspace-root", "", "Workspace root directory (overrides WORKSPACE_ROOT env var, default: ./incidents)")
	rootCmd.Flags().StringVar(&scriptPath, "script-path", "", "Path to agent script (default: ./scripts/stub-agent.sh)")
	rootCmd.Flags().StringVar(&logLevel, "log-level", "", "Log level (overrides LOG_LEVEL env var, default: info)")
}

func run(cmd *cobra.Command, args []string) error {
	// Override environment variables with flags if provided
	if sseEndpoint != "" {
		os.Setenv("SSE_ENDPOINT", sseEndpoint)
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
		"sse_endpoint", cfg.SSEEndpoint,
		"workspace_root", cfg.WorkspaceRoot,
		"log_level", cfg.LogLevel)

	// Determine script path
	agentScript := scriptPath
	if agentScript == "" {
		agentScript = "./scripts/stub-agent.sh"
	}

	// Verify script exists
	if _, err := os.Stat(agentScript); os.IsNotExist(err) {
		// Try absolute path from executable location
		execPath, _ := os.Executable()
		agentScript = filepath.Join(filepath.Dir(execPath), "scripts", "stub-agent.sh")
		if _, err := os.Stat(agentScript); os.IsNotExist(err) {
			return fmt.Errorf("agent script not found: %s", scriptPath)
		}
	}

	// Create components
	sseClient := events.NewClient(cfg.SSEEndpoint)
	workspaceMgr := agent.NewWorkspaceManager(cfg.WorkspaceRoot)
	executor := agent.NewExecutor(agentScript)

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

	// Subscribe to SSE events
	eventChan, err := sseClient.Subscribe(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to SSE: %w", err)
	}

	slog.Info("connected to SSE endpoint, waiting for events...")

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
			if err := processEvent(ctx, event, workspaceMgr, executor); err != nil {
				slog.Error("failed to process event", "error", err)
			}
		}
	}
}

func processEvent(ctx context.Context, event *events.FaultEvent, workspaceMgr *agent.WorkspaceManager, executor *agent.Executor) error {
	incidentID := uuid.New().String()
	startedAt := time.Now()

	slog.Info("processing event",
		"incident_id", incidentID,
		"cluster_id", event.ClusterID,
		"namespace", event.Namespace,
		"resource", fmt.Sprintf("%s/%s", event.ResourceType, event.ResourceName),
		"severity", event.Severity)

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

	slog.Info("event processed",
		"incident_id", incidentID,
		"status", status,
		"exit_code", exitCode,
		"duration", completedAt.Sub(startedAt))

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
