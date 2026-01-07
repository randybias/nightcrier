package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/randybias/nightcrier/internal/adminui"
	"github.com/randybias/nightcrier/internal/agent"
	"github.com/randybias/nightcrier/internal/agent/k8s"
	"github.com/randybias/nightcrier/internal/bootstrap"
	"github.com/randybias/nightcrier/internal/cluster"
	"github.com/randybias/nightcrier/internal/config"
	"github.com/randybias/nightcrier/internal/dispatcher"
	"github.com/randybias/nightcrier/internal/events"
	"github.com/randybias/nightcrier/internal/health"
	"github.com/randybias/nightcrier/internal/incident"
	"github.com/randybias/nightcrier/internal/nats"
	"github.com/randybias/nightcrier/internal/reload"
	"github.com/randybias/nightcrier/internal/reporting"
	"github.com/randybias/nightcrier/internal/storage"
	"github.com/randybias/nightcrier/internal/storage/postgres"
	"github.com/randybias/nightcrier/internal/storage/sqlite"
	"github.com/spf13/cobra"
)

// AgentExecutor is the interface that both Docker and K8s executors implement
type AgentExecutor interface {
	Execute(ctx context.Context, workspacePath string, incidentID string) (int, agent.LogPaths, error)
	ExecuteWithPrompt(ctx context.Context, workspacePath string, incidentID string, prompt string) (int, agent.LogPaths, error)
}

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
	adminListen   string
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

	// Admin UI flags
	rootCmd.Flags().StringVar(&adminListen, "admin-listen", "", "Address for admin UI server (e.g., 127.0.0.1:8847)")

	// Test mode flags
	rootCmd.Flags().Bool("single-run", false, "Process one fault event then exit (for test harnesses)")

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

	// Handle --single-run flag (test mode: process one event then exit)
	singleRun, _ := cmd.Flags().GetBool("single-run")

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

	// Verify system prompt file exists (required for agent operation)
	if cfg.Agent.SystemPromptFile != "" {
		if _, err := os.Stat(cfg.Agent.SystemPromptFile); os.IsNotExist(err) {
			return fmt.Errorf("agent system prompt file not found: %s\n\nThe system prompt file is required for agent operation. Please ensure:\n  1. The file exists at the specified path\n  2. The path in config (agent_system_prompt_file) is correct\n  3. The path is readable by the nightcrier process", cfg.Agent.SystemPromptFile)
		}
	} else {
		return fmt.Errorf("agent system prompt file not configured (agent_system_prompt_file is required)")
	}

	// Create ConnectionManager for multi-cluster support
	mgrConfig := &cluster.ManagerConfig{
		Clusters:                   cfg.MonitoredClusters,
		SubscribeMode:              cfg.SubscribeMode,
		GlobalQueueSize:            cfg.GlobalQueueSize,
		QueueOverflowPolicy:        cfg.QueueOverflowPolicy,
		MCPReconnectInitialBackoff: cfg.MCPReconnectInitialBackoff,
	}
	connectionMgr, err := cluster.NewConnectionManager(mgrConfig)
	if err != nil {
		return fmt.Errorf("failed to create connection manager: %w", err)
	}

	// Create and inject MCP clients for each cluster
	for _, clusterCfg := range cfg.MonitoredClusters {
		mcpClient := events.NewClient(clusterCfg.MCP.Endpoint, cfg.SubscribeMode, tuning)
		if err := connectionMgr.SetClusterClient(clusterCfg.Name, mcpClient); err != nil {
			return fmt.Errorf("failed to set client for cluster %s: %w", clusterCfg.Name, err)
		}
		slog.Info("mcp client created for cluster",
			"cluster", clusterCfg.Name,
			"endpoint", clusterCfg.MCP.Endpoint)
	}

	workspaceMgr := agent.NewWorkspaceManager(cfg.WorkspaceRoot)

	// Create Slack notifier (optional - only if webhook URL configured)
	var slackNotifier *reporting.SlackNotifier
	if cfg.SlackWebhookURL != "" {
		slackNotifier = reporting.NewSlackNotifier(cfg.SlackWebhookURL, tuning)
		slog.Info("slack notifications enabled")
	}

	// Create circuit breaker with configured threshold
	circuitBreaker := reporting.NewCircuitBreaker(cfg.FailureThresholdForAlert, tuning)
	slog.Info("circuit breaker initialized", "threshold", cfg.FailureThresholdForAlert)

	// Setup context with cancellation for graceful shutdown (needed for object storage and postgres.New)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize artifact storage backend (for investigation reports and logs)
	storageBackend, err := storage.NewStorage(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize artifact storage backend: %w", err)
	}
	artifactStorageMode := "local_filesystem"
	if cfg.ObjectStorage.URL != "" {
		artifactStorageMode = "object_storage"
	}
	slog.Info("artifact storage initialized", "backend", artifactStorageMode)

	// Initialize Object Store (required for artifact uploads)
	if cfg.ObjectStorage.URL == "" {
		return fmt.Errorf("object_storage.url must be configured")
	}

	expiry, err := time.ParseDuration(cfg.ObjectStorage.SignedURLExpiry)
	if err != nil {
		return fmt.Errorf("invalid signed URL expiry duration: %w", err)
	}
	objectStore, err := storage.NewObjectStore(ctx, cfg.ObjectStorage.URL, expiry)
	if err != nil {
		return fmt.Errorf("failed to initialize object store: %w", err)
	}
	slog.Info("object store initialized",
		"url", cfg.ObjectStorage.URL,
		"signed_url_expiry", expiry)

	// Initialize K8s client for executor (requires at least one execution cluster)
	if len(cfg.ExecutionClusters) == 0 {
		return fmt.Errorf("at least one execution_cluster must be configured for agent Job execution")
	}
	execCluster := cfg.ExecutionClusters[0]
	k8sClient, err := k8s.NewClient(k8s.ClientConfig{
		Kubeconfig: execCluster.KubeconfigPath,
	})
	if err != nil {
		return fmt.Errorf("failed to create K8s client: %w", err)
	}
	slog.Info("K8s client initialized",
		"kubeconfig", execCluster.KubeconfigPath,
		"context", "",
		"namespace", cfg.ExecutionDefaults.Namespace)

	// Bootstrap Kubernetes resources (namespace, RBAC, secrets)
	slog.Info("bootstrapping kubernetes resources...")
	bootstrapCtx, bootstrapCancel := context.WithTimeout(ctx, 2*time.Minute)
	defer bootstrapCancel()

	bootstrapClusters := make([]bootstrap.MonitoredClusterConfig, 0, len(cfg.MonitoredClusters))
	for _, c := range cfg.MonitoredClusters {
		if c.Triage.Enabled && c.Triage.Kubeconfig != "" {
			bootstrapClusters = append(bootstrapClusters, bootstrap.MonitoredClusterConfig{
				Name:                 c.Name,
				TargetKubeconfigPath: c.Triage.Kubeconfig,
			})
		}
	}

	bootstrapConfig := bootstrap.Config{
		Namespace:         cfg.ExecutionDefaults.Namespace,
		AnthropicAPIKey:   cfg.AnthropicAPIKey,
		OpenAIAPIKey:      cfg.OpenAIAPIKey,
		GeminiAPIKey:      cfg.GeminiAPIKey,
		MonitoredClusters: bootstrapClusters,
	}

	bootstrapMgr := bootstrap.NewManager(k8sClient.Clientset(), bootstrapConfig)
	bootstrapResult, err := bootstrapMgr.Bootstrap(bootstrapCtx)
	if err != nil {
		slog.Error("kubernetes bootstrap failed",
			"error", err,
			"remediation", "check permissions: ensure kubeconfig user can create namespaces, RBAC, and secrets")
		return fmt.Errorf("kubernetes bootstrap failed: %w", err)
	}

	// Log bootstrap results
	createdCount := bootstrapResult.CreatedCount()
	existingCount := bootstrapResult.ExistingCount()
	if createdCount > 0 {
		slog.Info("kubernetes bootstrap complete",
			"created", createdCount,
			"existing", existingCount,
			"namespace_created", bootstrapResult.NamespaceCreated)
	} else {
		slog.Debug("kubernetes resources already exist, skipping creation",
			"resources", existingCount)
	}

	// Create ExecutionClusterManager for managing execution clusters
	execClusterConfigs := make([]cluster.ExecutionClusterConfig, 0, len(cfg.ExecutionClusters))
	for _, ec := range cfg.ExecutionClusters {
		execClusterConfigs = append(execClusterConfigs, cluster.ExecutionClusterConfig{
			Name:                ec.Name,
			KubeconfigPath:      ec.KubeconfigPath,
			Namespace:           ec.Namespace,
			RunnerImage:         ec.RunnerImage,
			ImagePullPolicy:     ec.ImagePullPolicy,
			Timeout:             ec.Timeout,
			MemoryLimit:         ec.MemoryLimit,
			CPULimit:            ec.CPULimit,
			CleanupTTL:          ec.CleanupTTL,
			MaxConcurrentAgents: ec.MaxConcurrentAgents,
		})
	}
	execMgrConfig := &cluster.ExecutionManagerConfig{
		Clusters: execClusterConfigs,
		Defaults: &cluster.ExecutionDefaults{
			Namespace:           cfg.ExecutionDefaults.Namespace,
			RunnerImage:         cfg.ExecutionDefaults.RunnerImage,
			ImagePullPolicy:     cfg.ExecutionDefaults.ImagePullPolicy,
			Timeout:             cfg.ExecutionDefaults.Timeout,
			MemoryLimit:         cfg.ExecutionDefaults.MemoryLimit,
			CPULimit:            cfg.ExecutionDefaults.CPULimit,
			CleanupTTL:          cfg.ExecutionDefaults.CleanupTTL,
			MaxConcurrentAgents: cfg.ExecutionDefaults.MaxConcurrentAgents,
		},
	}
	executionMgr, err := cluster.NewExecutionClusterManager(execMgrConfig)
	if err != nil {
		return fmt.Errorf("failed to create execution cluster manager: %w", err)
	}

	// Build execution clusters map for the K8s executor (temporary until K8sExecutor uses ExecutionClusterManager)
	executionClustersMap := make(map[string]*config.ExecutionClusterConfig)
	for i := range cfg.ExecutionClusters {
		ec := &cfg.ExecutionClusters[i]
		executionClustersMap[ec.Name] = ec
	}
	defaultExecutionCluster := executionMgr.DefaultClusterName()

	// Create shared K8s executor config (agent-specific settings)
	k8sExecCfg := agent.K8sExecutorConfig{
		AgentCLI:         cfg.Agent.CLI,
		Model:            cfg.Agent.Model,
		SystemPromptFile: cfg.Agent.SystemPromptFile,
		Debug:            cfg.LogLevel == "debug",
		NATSEnabled:      cfg.NATS.Enabled,
		NATSServer:       cfg.NATS.Server,
		NATSToken:        cfg.NATS.Token,
	}

	// Create single K8s executor that handles all execution clusters
	k8sExec := agent.NewK8sExecutor(
		k8sExecCfg,
		executionClustersMap,
		defaultExecutionCluster,
		k8sClient,
		objectStore,
		nil, // stateStore will be set later after it's initialized
		storageBackend,
		tuning,
	)
	slog.Info("K8s executor initialized",
		"execution_clusters", len(executionClustersMap),
		"default_cluster", defaultExecutionCluster)

	// Create executors map for compatibility (all monitored clusters use the same executor)
	executors := make(map[string]AgentExecutor)
	for _, clusterCfg := range cfg.MonitoredClusters {
		executors[clusterCfg.Name] = k8sExec
	}

	// Create configuration reloader for SIGHUP handling
	reloader := reload.NewReloader(&reload.ReloaderConfig{
		ConfigFile:    config.GetConfigFile(),
		ConnectionMgr: connectionMgr,
		ExecutionMgr:  executionMgr,
		ClusterStore:  nil, // Will be set after stateStore is initialized if it implements ClusterStore
		CurrentConfig: cfg,
	})

	// Handle signals: SIGHUP for config reload, SIGINT/SIGTERM for shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		for sig := range sigChan {
			switch sig {
			case syscall.SIGHUP:
				slog.Info("received SIGHUP, reloading configuration")
				result := reloader.Reload(ctx)
				if result.Error != nil {
					slog.Error("failed to reload configuration", "error", result.Error)
				} else {
					slog.Info("configuration reloaded successfully",
						"monitored_added", len(result.MonitoredAdded),
						"monitored_removed", len(result.MonitoredRemoved),
						"execution_added", len(result.ExecutionAdded),
						"execution_removed", len(result.ExecutionRemoved))
				}
			case syscall.SIGINT, syscall.SIGTERM:
				slog.Info("received shutdown signal", "signal", sig)
				cancel()
				return
			}
		}
	}()

	// Initialize state store (SQL persistence) based on configuration
	var stateStore storage.StateStore
	storageType := cfg.GetStateStorageType()

	switch storageType {
	case "filesystem":
		// No SQL backend needed for filesystem storage
		slog.Info("state store disabled (using filesystem storage)")

	case "sqlite":
		dbPath := cfg.StateStorage.SQLitePath
		migrationsPath := cfg.StateStorage.MigrationsPath
		slog.Info("initializing SQLite state store", "path", dbPath, "migrations", migrationsPath)

		// Run migrations
		slog.Info("running database migrations", "driver", "sqlite", "path", migrationsPath)
		migrationCfg := &storage.MigrationConfig{
			MigrationsPath: migrationsPath,
			DatabaseType:   "sqlite",
			DatabasePath:   dbPath,
		}
		if err := storage.RunMigrations(migrationCfg); err != nil {
			return fmt.Errorf("failed to run SQLite migrations: %w", err)
		}

		// Create SQLite store
		sqliteCfg := &sqlite.Config{
			Path: dbPath,
		}
		stateStore, err = sqlite.New(sqliteCfg)
		if err != nil {
			return fmt.Errorf("failed to create SQLite store: %w", err)
		}
		defer stateStore.Close()
		slog.Info("SQLite state store initialized successfully")

	case "postgres":
		var connStr string
		if cfg.StateStorage.PostgresConnectionString != "" {
			connStr = cfg.StateStorage.PostgresConnectionString
		} else {
			// URL-encode credentials to handle special characters
			connStr = fmt.Sprintf(
				"postgres://%s:%s@%s:%d/%s?sslmode=disable",
				url.QueryEscape(cfg.StateStorage.PostgresUser),
				url.QueryEscape(cfg.StateStorage.PostgresPassword),
				cfg.StateStorage.PostgresHost,
				cfg.StateStorage.PostgresPort,
				cfg.StateStorage.PostgresDatabase,
			)
		}
		migrationsPath := cfg.StateStorage.MigrationsPath

		slog.Info("initializing PostgreSQL state store",
			"host", cfg.StateStorage.PostgresHost,
			"database", cfg.StateStorage.PostgresDatabase,
			"migrations", migrationsPath)

		// Run migrations
		slog.Info("running database migrations", "driver", "postgres", "path", migrationsPath)
		migrationCfg := &storage.MigrationConfig{
			MigrationsPath: migrationsPath,
			DatabaseType:   "postgres",
			DatabaseURL:    connStr,
		}
		if err := storage.RunMigrations(migrationCfg); err != nil {
			return fmt.Errorf("failed to run PostgreSQL migrations: %w", err)
		}

		// Create PostgreSQL store
		postgresCfg := &postgres.Config{
			ConnectionString: connStr,
		}
		stateStore, err = postgres.New(ctx, postgresCfg)
		if err != nil {
			return fmt.Errorf("failed to create PostgreSQL store: %w", err)
		}
		defer stateStore.Close()
		slog.Info("PostgreSQL state store initialized successfully")

	default:
		return fmt.Errorf("unknown state storage type: %s", storageType)
	}

	// Inject stateStore into K8s executor now that it's initialized
	if stateStore != nil {
		k8sExec.SetStateStore(stateStore)
		slog.Debug("stateStore injected into K8s executor")

		// Check if stateStore implements ClusterStore interface for dynamic cluster loading
		if clusterStore, ok := stateStore.(reload.ClusterStore); ok {
			reloader.SetClusterStore(clusterStore)
			slog.Debug("cluster store injected into reloader")

			// Perform initial sync of YAML clusters to database
			// This ensures database is the single source of truth
			result := reloader.Reload(ctx)
			if result.Error != nil {
				slog.Warn("initial cluster sync to database failed", "error", result.Error)
			} else {
				slog.Info("initial cluster sync complete",
					"monitored_synced", len(result.MonitoredAdded),
					"execution_synced", len(result.ExecutionAdded))
			}

			// Start database polling if no clusters are configured
			// This allows the system to start with zero clusters and pick them up from the database
			reloader.StartDatabasePolling(ctx)
		}
	}

	// Initialize NATS client and listener if enabled
	var natsClient *nats.Client
	var natsListener *nats.Listener
	if cfg.NATS.Enabled {
		slog.Info("initializing NATS client",
			"server", cfg.NATS.Server,
			"connect_timeout", cfg.NATS.ConnectTimeout,
			"reconnect_wait", cfg.NATS.ReconnectWait)

		var err error
		natsClient, err = nats.Connect(
			cfg.NATS.Server,
			cfg.NATS.Token,
			nats.WithName("nightcrier"),
			nats.WithTimeout(cfg.NATS.ConnectTimeout),
			nats.WithReconnectWait(cfg.NATS.ReconnectWait),
		)
		if err != nil {
			slog.Warn("failed to connect to NATS server (continuing without progress tracking)",
				"error", err,
				"server", cfg.NATS.Server)
		} else {
			slog.Info("NATS client connected successfully")

			// Only create listener if we have a stateStore
			if stateStore != nil {
				natsListener = nats.NewListener(natsClient, stateStore)
				go func() {
					if err := natsListener.Start(ctx); err != nil {
						slog.Warn("NATS listener stopped", "error", err)
					}
				}()
				slog.Info("NATS listener started, subscribing to progress events")
			} else {
				slog.Warn("NATS enabled but no stateStore available, listener not started")
			}
		}
	} else {
		slog.Info("NATS progress tracking disabled")
	}

	// Phase 3: Initialize connection manager (validates cluster permissions)
	// This runs kubectl auth can-i checks for all clusters with triage enabled
	slog.Info("initializing connection manager - validating permissions")
	initCtx, initCancel := context.WithTimeout(ctx, 30*time.Second)
	defer initCancel()
	if err := connectionMgr.Initialize(initCtx); err != nil {
		return fmt.Errorf("failed to initialize connection manager: %w", err)
	}

	// Build kubeconfig and permissions maps for the event handler
	// These are static per-cluster and are used by the dispatcher's event handler
	kubeconfigMap := make(map[string]string)
	permissionsMap := make(map[string]*cluster.ClusterPermissions)
	for _, clusterCfg := range cfg.MonitoredClusters {
		kubeconfigMap[clusterCfg.Name] = clusterCfg.Triage.Kubeconfig
		// Permissions will be looked up from the event since they're set during Initialize
	}

	// Create the event handler closure that captures all dependencies
	eventHandler := func(ctx context.Context, event *events.FaultEvent, clusterName string) error {
		// Look up cluster-specific data
		kubeconfig := kubeconfigMap[clusterName]
		permissions := permissionsMap[clusterName]

		// Get executor for this cluster
		executor, ok := executors[clusterName]
		if !ok {
			return fmt.Errorf("no executor found for cluster: %s", clusterName)
		}

		// Process the event
		return processEvent(ctx, event, clusterName, kubeconfig, permissions, workspaceMgr, executor, slackNotifier, storageBackend, stateStore, circuitBreaker, cfg, tuning)
	}

	// Create dispatcher with the event handler
	eventDispatcher := dispatcher.NewDispatcher(cfg, eventHandler)
	slog.Info("dispatcher initialized",
		"max_concurrent_agents", cfg.MaxConcurrentAgents,
		"drop_events_while_busy", *cfg.DropEventsWhileBusy,
		"cluster_failure_event_queue_size", cfg.ClusterFailureEventQueueSize,
		"event_ttl_seconds", cfg.EventTTLSeconds)

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

	// Start admin UI server if enabled
	if adminListen != "" && stateStore != nil {
		// Get the underlying sql.DB from the state store
		var adminDB *sql.DB
		switch s := stateStore.(type) {
		case *sqlite.Store:
			adminDB = s.DB()
		case *postgres.Store:
			adminDB = s.DB()
		}

		if adminDB != nil {
			// Build cluster info for admin UI display
			clusterInfos := make([]adminui.ClusterInfo, len(cfg.MonitoredClusters))
			for i, c := range cfg.MonitoredClusters {
				clusterInfos[i] = adminui.ClusterInfo{
					Name:          c.Name,
					Environment:   c.Environment,
					MCPEndpoint:   c.MCP.Endpoint,
					TriageEnabled: c.Triage.Enabled,
				}
			}

			adminCfg := adminui.Config{
				DB:           adminDB,
				ListenAddr:   adminListen,
				ObjectSigner: objectStore,
				JobCanceller: k8sClient,
				Namespace:    cfg.ExecutionDefaults.Namespace,
				Clusters:     clusterInfos,
			}
			adminServer, err := adminui.NewServer(adminCfg)
			if err != nil {
				slog.Error("failed to create admin UI server", "error", err)
			} else {
				go func() {
					slog.Info("starting admin UI server",
						"addr", adminListen,
						"endpoint", fmt.Sprintf("http://%s/admin", adminListen))
					if err := adminServer.Start(); err != nil && err != http.ErrServerClosed {
						slog.Error("admin UI server failed", "error", err)
					}
				}()
			}
		} else {
			slog.Warn("admin UI disabled: could not get database connection from state store")
		}
	} else if adminListen != "" {
		slog.Warn("admin UI disabled: requires SQL state storage (sqlite or postgres)")
	}

	// Start the ConnectionManager and get event channel
	eventChan := connectionMgr.Start(ctx)
	defer connectionMgr.Stop()

	slog.Info("connection manager started, processing events",
		"cluster_count", len(cfg.MonitoredClusters))

	// Track if we've processed an event in single-run mode
	singleRunProcessed := false

	// Event processing loop
	for {
		select {
		case <-ctx.Done():
			slog.Info("shutting down, waiting for in-flight agents...")

			// Graceful shutdown: wait for dispatcher to complete in-flight agents
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer shutdownCancel()

			if err := eventDispatcher.Shutdown(shutdownCtx); err != nil {
				slog.Warn("dispatcher shutdown timed out", "error", err)
			}

			// Stop database polling
			reloader.StopDatabasePolling()

			// Shutdown NATS listener and client
			if natsListener != nil {
				slog.Info("shutting down NATS listener")
				natsListener.Stop()
			}
			if natsClient != nil {
				slog.Info("closing NATS client")
				natsClient.Close()
			}

			return nil

		case event, ok := <-eventChan:
			if !ok {
				slog.Info("event channel closed, waiting for in-flight agents...")

				// Event channel closed, wait for dispatcher to complete
				shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer shutdownCancel()

				if err := eventDispatcher.Shutdown(shutdownCtx); err != nil {
					slog.Warn("dispatcher shutdown timed out", "error", err)
				}

				return nil
			}

			// Single-run mode: skip any events after the first one
			if singleRun && singleRunProcessed {
				slog.Debug("single-run mode: dropping event, shutdown in progress")
				continue
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

			// Extract and cache cluster permissions from the event
			// These are set during connectionMgr.Initialize() and included with each event
			if permissions, ok := clusterEvent["Permissions"].(*cluster.ClusterPermissions); ok && permissions != nil {
				permissionsMap[clusterName] = permissions
			}

			// Extract the FaultEvent
			faultEvent, ok := clusterEvent["Event"].(*events.FaultEvent)
			if !ok {
				slog.Error("missing or invalid Event in cluster event",
					"cluster", clusterName,
					"type", fmt.Sprintf("%T", clusterEvent["Event"]))
				continue
			}

			// Dispatch the event to the dispatcher (non-blocking)
			// The dispatcher handles concurrency limits and per-cluster serialization
			eventDispatcher.Dispatch(ctx, faultEvent, clusterName)

			// Single-run mode: exit after first event is dispatched
			// Note: We wait for the dispatcher to process it during shutdown
			if singleRun {
				singleRunProcessed = true
				slog.Info("single-run mode: event dispatched, initiating shutdown")
				cancel()
			}
		}
	}
}

func processEvent(ctx context.Context, event *events.FaultEvent, clusterName string, kubeconfig string, permissions *cluster.ClusterPermissions, workspaceMgr *agent.WorkspaceManager, executor AgentExecutor, slackNotifier *reporting.SlackNotifier, storageBackend storage.Storage, stateStore storage.StateStore, circuitBreaker *reporting.CircuitBreaker, cfg *config.Config, tuning *config.TuningConfig) error {
	// Create incident from event
	incidentID := uuid.New().String()
	inc := incident.NewFromEvent(incidentID, event)

	// Override cluster name with the one from ClusterEvent (Phase 2: multi-cluster support)
	inc.Cluster = clusterName

	// Persist incident to state store (SQL database)
	if stateStore != nil {
		if err := stateStore.CreateIncident(ctx, inc, event); err != nil {
			slog.Error("failed to create incident in state store", "incident_id", incidentID, "error", err)
			// Continue processing - don't fail the incident if database write fails
		}
	}

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

	// Mark agent start time (use UTC for database consistency)
	startedAt := time.Now().UTC()
	inc.StartedAt = &startedAt

	// Update incident status to investigating in state store
	if stateStore != nil {
		if err := stateStore.UpdateIncidentStatus(ctx, incidentID, incident.StatusInvestigating, &startedAt); err != nil {
			slog.Error("failed to update incident status in state store", "incident_id", incidentID, "error", err)
		}
	}

	// Execute agent
	exitCode, logPaths, execErr := executor.Execute(ctx, workspacePath, incidentID)

	// Update incident with completion info
	inc.MarkCompleted(exitCode, execErr)

	// Detect agent failures (exit code 0 but missing or invalid output)
	agentFailed, failureReason := detectAgentFailure(exitCode, execErr)
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

	// Get artifact info from K8s executor (artifacts already uploaded by processor)
	reportURL := logPaths.ReportURL
	rootCause := logPaths.RootCause
	confidence := logPaths.Confidence

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
			// Extract root cause and confidence if not already available from K8s executor
			if rootCause == "" {
				var err error
				rootCause, confidence, err = reporting.ExtractSummaryFromReport(workspacePath)
				if err != nil {
					slog.Warn("failed to extract report summary for slack", "error", err)
					rootCause = "See investigation report"
					confidence = "UNKNOWN"
				}
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
// 2. No execution error
//
// Note: Artifact validation (report exists, proper size) is handled by the K8s executor
// during artifact retrieval from blob storage.
//
// Returns (failed bool, reason string)
func detectAgentFailure(exitCode int, err error) (bool, string) {
	// Check if there was an execution error
	if err != nil {
		return true, fmt.Sprintf("agent execution error: %v", err)
	}

	// Check exit code
	if exitCode != 0 {
		return true, fmt.Sprintf("agent exited with non-zero code: %d", exitCode)
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

// printStartupBanner displays configuration summary at startup
func printStartupBanner(cfg *config.Config, configFile string) {
	// Determine artifact storage mode (for reports/logs)
	artifactStorage := "local_filesystem"
	if cfg.IsAzureStorageEnabled() {
		artifactStorage = "azure_blob"
	}

	// Determine state storage mode (for incident metadata)
	stateStorage := cfg.GetStateStorageType()
	if stateStorage == "" {
		stateStorage = "filesystem"
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
	fmt.Printf("║  Clusters:       %-45s ║\n", fmt.Sprintf("%d configured", len(cfg.MonitoredClusters)))
	fmt.Printf("║  Subscribe Mode: %-45s ║\n", cfg.SubscribeMode)
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Agent CLI:      %-45s ║\n", cfg.Agent.CLI)
	fmt.Printf("║  Agent Model:    %-45s ║\n", cfg.Agent.Model)
	fmt.Printf("║  Agent Timeout:  %-45s ║\n", fmt.Sprintf("%ds", cfg.Agent.Timeout))
	fmt.Printf("║  Allowed Tools:  %-45s ║\n", truncateString(cfg.Agent.AllowedTools, 45))
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Workspace Root:     %-41s ║\n", truncateString(cfg.WorkspaceRoot, 41))
	fmt.Printf("║  Artifact Storage:   %-41s ║\n", artifactStorage)
	fmt.Printf("║  State Storage:      %-41s ║\n", stateStorage)
	fmt.Printf("║  Slack:              %-41s ║\n", slackStatus)
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Log Level:      %-45s ║\n", cfg.LogLevel)
	fmt.Printf("║  Max Concurrent: %-45s ║\n", fmt.Sprintf("%d agents", cfg.MaxConcurrentAgents))
	fmt.Printf("║  Queue Size:     %-45s ║\n", fmt.Sprintf("%d events/cluster", cfg.ClusterFailureEventQueueSize))
	fmt.Printf("║  Event TTL:      %-45s ║\n", fmt.Sprintf("%ds", cfg.EventTTLSeconds))
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

// buildObjectBaseURL constructs the base URL for accessing artifacts in object storage.
// For Azure Blob Storage: https://<account>.blob.core.windows.net/<container>
// For S3: https://<bucket>.s3.<region>.amazonaws.com (or custom endpoint)
func buildObjectBaseURL(cfg *config.Config) string {
	storageURL := cfg.ObjectStorage.URL
	if storageURL == "" {
		return ""
	}

	// Azure Blob Storage: azblob://container-name
	if strings.HasPrefix(storageURL, "azblob://") {
		container := strings.TrimPrefix(storageURL, "azblob://")
		account := cfg.ObjectStorage.AzureStorageAccount
		if account != "" {
			return fmt.Sprintf("https://%s.blob.core.windows.net/%s", account, container)
		}
	}

	// S3: s3://bucket-name?region=us-east-1&endpoint=...
	if strings.HasPrefix(storageURL, "s3://") {
		// Parse the URL to extract bucket and endpoint
		parsed, err := url.Parse(storageURL)
		if err != nil {
			return ""
		}
		bucket := parsed.Host

		// Check for custom endpoint (e.g., MinIO, RustFS)
		endpoint := parsed.Query().Get("endpoint")
		if endpoint != "" {
			// Custom endpoint: http://endpoint/bucket
			return fmt.Sprintf("%s/%s", strings.TrimSuffix(endpoint, "/"), bucket)
		}

		// Standard S3
		region := parsed.Query().Get("region")
		if region == "" {
			region = "us-east-1"
		}
		return fmt.Sprintf("https://%s.s3.%s.amazonaws.com", bucket, region)
	}

	return ""
}
