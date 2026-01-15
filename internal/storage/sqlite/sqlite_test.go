package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/randybias/nightcrier/internal/events"
	"github.com/randybias/nightcrier/internal/incident"
	"github.com/randybias/nightcrier/internal/storage"
)

// setupTestStore creates a new in-memory SQLite store for testing.
// Uses MaxOpenConns: 1 to ensure all connections use the same :memory: database.
func setupTestStore(t *testing.T) *Store {
	t.Helper()

	cfg := &Config{
		Path:            ":memory:",
		BusyTimeout:     5 * time.Second,
		MaxOpenConns:    1, // Single connection ensures all goroutines use the same in-memory DB
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Hour,
	}

	store, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}

	// Run migrations
	if err := runTestMigrations(store.db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return store
}

// runTestMigrations applies the schema to the test database.
func runTestMigrations(db *sql.DB) error {
	schema := `
-- fault_events table stores the raw fault events received from kubernetes-mcp-server
CREATE TABLE IF NOT EXISTS fault_events (
    fault_id TEXT PRIMARY KEY,
    subscription_id TEXT NOT NULL,
    cluster TEXT NOT NULL,
    received_at TIMESTAMP NOT NULL,
    resource_api_version TEXT,
    resource_kind TEXT,
    resource_name TEXT,
    resource_namespace TEXT,
    resource_uid TEXT,
    fault_type TEXT NOT NULL,
    severity TEXT NOT NULL,
    context TEXT NOT NULL,
    timestamp TEXT NOT NULL,
    CONSTRAINT idx_fault_events_cluster CHECK (cluster <> ''),
    CONSTRAINT idx_fault_events_fault_type CHECK (fault_type <> '')
);

CREATE INDEX IF NOT EXISTS idx_fault_events_cluster ON fault_events(cluster);
CREATE INDEX IF NOT EXISTS idx_fault_events_received_at ON fault_events(received_at);
CREATE INDEX IF NOT EXISTS idx_fault_events_fault_type ON fault_events(fault_type);
CREATE INDEX IF NOT EXISTS idx_fault_events_severity ON fault_events(severity);

-- incidents table stores the investigation incidents created from fault events
CREATE TABLE IF NOT EXISTS incidents (
    incident_id TEXT PRIMARY KEY,
    fault_id TEXT NOT NULL,
    triggering_event_id TEXT,
    status TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    job_started_at TIMESTAMP,
    job_completed_at TIMESTAMP,
    exit_code INTEGER,
    failure_reason TEXT,
    cluster TEXT NOT NULL,
    namespace TEXT,
    fault_type TEXT NOT NULL,
    severity TEXT NOT NULL,
    context TEXT NOT NULL,
    timestamp TEXT NOT NULL,
    resource_api_version TEXT,
    resource_kind TEXT,
    resource_name TEXT,
    resource_namespace TEXT,
    resource_uid TEXT,
    FOREIGN KEY (fault_id) REFERENCES fault_events(fault_id),
    CONSTRAINT chk_incidents_status CHECK (status IN ('pending', 'investigating', 'resolved', 'failed', 'agent_failed')),
    CONSTRAINT chk_incidents_cluster CHECK (cluster <> ''),
    CONSTRAINT chk_incidents_fault_type CHECK (fault_type <> '')
);

CREATE INDEX IF NOT EXISTS idx_incidents_fault_id ON incidents(fault_id);
CREATE INDEX IF NOT EXISTS idx_incidents_status ON incidents(status);
CREATE INDEX IF NOT EXISTS idx_incidents_cluster ON incidents(cluster);
CREATE INDEX IF NOT EXISTS idx_incidents_created_at ON incidents(created_at);
CREATE INDEX IF NOT EXISTS idx_incidents_namespace ON incidents(namespace);
CREATE INDEX IF NOT EXISTS idx_incidents_fault_type ON incidents(fault_type);
CREATE INDEX IF NOT EXISTS idx_incidents_severity ON incidents(severity);

-- agent_executions table stores details of agent execution attempts
CREATE TABLE IF NOT EXISTS agent_executions (
    execution_id TEXT PRIMARY KEY,
    incident_id TEXT NOT NULL,
    job_started_at TIMESTAMP NOT NULL,
    job_completed_at TIMESTAMP,
    run_started_at TIMESTAMP,
    run_completed_at TIMESTAMP,
    run_exit_code INTEGER,
    exit_code INTEGER,
    error_message TEXT,
    log_paths TEXT,
    agent_cli TEXT NOT NULL DEFAULT 'unknown',
    agent_model TEXT NOT NULL DEFAULT 'unknown',
    cluster_name TEXT NOT NULL DEFAULT 'unknown',
    current_activity TEXT,
    current_activity_started_at TIMESTAMP,
    last_activity TEXT,
    last_activity_finished_at TIMESTAMP,
    FOREIGN KEY (incident_id) REFERENCES incidents(incident_id),
    CONSTRAINT chk_agent_executions_incident_id CHECK (incident_id <> '')
);

CREATE INDEX IF NOT EXISTS idx_agent_executions_incident_id ON agent_executions(incident_id);
CREATE INDEX IF NOT EXISTS idx_agent_executions_job_started_at ON agent_executions(job_started_at);
CREATE INDEX IF NOT EXISTS idx_agent_executions_run_started_at ON agent_executions(run_started_at);
CREATE INDEX IF NOT EXISTS idx_agent_executions_run_completed_at ON agent_executions(run_completed_at);
CREATE INDEX IF NOT EXISTS idx_agent_executions_agent_cli ON agent_executions(agent_cli);
CREATE INDEX IF NOT EXISTS idx_agent_executions_agent_model ON agent_executions(agent_model);
CREATE INDEX IF NOT EXISTS idx_agent_executions_cluster_name ON agent_executions(cluster_name);

-- triage_reports table stores the investigation reports generated by agents
CREATE TABLE IF NOT EXISTS triage_reports (
    report_id TEXT PRIMARY KEY,
    incident_id TEXT NOT NULL,
    execution_id TEXT NOT NULL,
    generated_at TIMESTAMP NOT NULL,
    report_markdown TEXT NOT NULL,
    report_html TEXT,
    FOREIGN KEY (incident_id) REFERENCES incidents(incident_id),
    FOREIGN KEY (execution_id) REFERENCES agent_executions(execution_id),
    CONSTRAINT chk_triage_reports_incident_id CHECK (incident_id <> ''),
    CONSTRAINT chk_triage_reports_execution_id CHECK (execution_id <> '')
);

CREATE INDEX IF NOT EXISTS idx_triage_reports_incident_id ON triage_reports(incident_id);
CREATE INDEX IF NOT EXISTS idx_triage_reports_execution_id ON triage_reports(execution_id);
CREATE INDEX IF NOT EXISTS idx_triage_reports_generated_at ON triage_reports(generated_at);

-- Monitored clusters: where fault events are detected
CREATE TABLE IF NOT EXISTS monitored_clusters (
    name TEXT PRIMARY KEY,
    environment TEXT,
    labels TEXT,  -- JSON object stored as TEXT for SQLite compatibility
    mcp_endpoint TEXT NOT NULL,
    mcp_api_key TEXT,
    triage_enabled INTEGER NOT NULL DEFAULT 0,
    target_kubeconfig TEXT,
    allow_secrets_access INTEGER NOT NULL DEFAULT 0,
    execution_cluster TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    source TEXT NOT NULL DEFAULT 'database',
    -- Runtime reachability fields
    connection_status TEXT NOT NULL DEFAULT 'disconnected',
    unreachable INTEGER NOT NULL DEFAULT 0,
    unreachable_reason TEXT,
    last_status_check TIMESTAMP,
    last_error TEXT
);

-- Execution clusters: where agent Jobs run
CREATE TABLE IF NOT EXISTS execution_clusters (
    name TEXT PRIMARY KEY,
    kubeconfig TEXT NOT NULL,
    namespace TEXT NOT NULL DEFAULT 'nightcrier',
    runner_image TEXT NOT NULL DEFAULT 'nc-agent-runner:latest',
    image_pull_policy TEXT NOT NULL DEFAULT 'IfNotPresent',
    timeout INTEGER NOT NULL DEFAULT 600,
    memory_limit TEXT NOT NULL DEFAULT '2Gi',
    cpu_limit TEXT NOT NULL DEFAULT '1',
    cleanup_ttl INTEGER NOT NULL DEFAULT 3600,
    max_concurrent_agents INTEGER NOT NULL DEFAULT 10,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    source TEXT NOT NULL DEFAULT 'database'
);

CREATE INDEX IF NOT EXISTS idx_monitored_clusters_environment ON monitored_clusters(environment);
CREATE INDEX IF NOT EXISTS idx_monitored_clusters_source ON monitored_clusters(source);
CREATE INDEX IF NOT EXISTS idx_execution_clusters_source ON execution_clusters(source);
`
	_, err := db.Exec(schema)
	return err
}

// createTestEvent creates a test fault event.
func createTestEvent(faultID string) *events.FaultEvent {
	return &events.FaultEvent{
		FaultID:        faultID,
		SubscriptionID: "sub-123",
		Cluster:        "test-cluster",
		ReceivedAt:     time.Now(),
		Resource: &events.ResourceInfo{
			APIVersion: "v1",
			Kind:       "Pod",
			Name:       "test-pod",
			Namespace:  "default",
			UID:        "pod-uid-123",
		},
		FaultType: "PodCrashLoop",
		Severity:  "critical",
		Context:   "Pod is crash looping",
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// createTestIncident creates a test incident from an event.
func createTestIncident(incidentID string, event *events.FaultEvent) *incident.Incident {
	return incident.NewFromEvent(incidentID, event)
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "default config",
			cfg:  DefaultConfig(),
		},
		{
			name: "in-memory database",
			cfg: &Config{
				Path:            ":memory:",
				BusyTimeout:     5 * time.Second,
				MaxOpenConns:    10,
				MaxIdleConns:    2,
				ConnMaxLifetime: time.Hour,
			},
		},
		{
			name: "nil config uses defaults",
			cfg:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := New(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if store != nil {
				defer store.Close()
			}
		})
	}
}

func TestCreateIncident(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()
	event := createTestEvent("fault-001")
	inc := createTestIncident("inc-001", event)

	// Create incident
	err := store.CreateIncident(ctx, inc, event)
	if err != nil {
		t.Fatalf("CreateIncident() error = %v", err)
	}

	// Verify incident was created
	retrieved, err := store.GetIncident(ctx, inc.IncidentID)
	if err != nil {
		t.Fatalf("GetIncident() error = %v", err)
	}
	if retrieved == nil {
		t.Fatal("GetIncident() returned nil")
	}

	// Verify fields
	if retrieved.IncidentID != inc.IncidentID {
		t.Errorf("IncidentID = %v, want %v", retrieved.IncidentID, inc.IncidentID)
	}
	if retrieved.FaultID != inc.FaultID {
		t.Errorf("FaultID = %v, want %v", retrieved.FaultID, inc.FaultID)
	}
	if retrieved.Status != inc.Status {
		t.Errorf("Status = %v, want %v", retrieved.Status, inc.Status)
	}
	if retrieved.Cluster != inc.Cluster {
		t.Errorf("Cluster = %v, want %v", retrieved.Cluster, inc.Cluster)
	}
	if retrieved.Resource == nil {
		t.Fatal("Resource is nil")
	}
	if retrieved.Resource.Kind != "Pod" {
		t.Errorf("Resource.Kind = %v, want Pod", retrieved.Resource.Kind)
	}
}

func TestCreateIncident_Duplicate(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()
	event := createTestEvent("fault-002")
	inc := createTestIncident("inc-002", event)

	// Create incident first time
	err := store.CreateIncident(ctx, inc, event)
	if err != nil {
		t.Fatalf("CreateIncident() first call error = %v", err)
	}

	// Try to create same incident again - should fail
	err = store.CreateIncident(ctx, inc, event)
	if err == nil {
		t.Fatal("CreateIncident() second call should have failed")
	}
}

func TestUpdateIncidentStatus(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()
	event := createTestEvent("fault-003")
	inc := createTestIncident("inc-003", event)

	// Create incident
	err := store.CreateIncident(ctx, inc, event)
	if err != nil {
		t.Fatalf("CreateIncident() error = %v", err)
	}

	// Update status
	startedAt := time.Now()
	err = store.UpdateIncidentStatus(ctx, inc.IncidentID, incident.StatusResolved, &startedAt)
	if err != nil {
		t.Fatalf("UpdateIncidentStatus() error = %v", err)
	}

	// Verify update
	retrieved, err := store.GetIncident(ctx, inc.IncidentID)
	if err != nil {
		t.Fatalf("GetIncident() error = %v", err)
	}
	if retrieved.Status != incident.StatusResolved {
		t.Errorf("Status = %v, want %v", retrieved.Status, incident.StatusResolved)
	}
	if retrieved.StartedAt == nil {
		t.Fatal("StartedAt is nil")
	}
}

func TestUpdateIncidentStatus_NotFound(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()
	startedAt := time.Now()

	// Try to update non-existent incident
	err := store.UpdateIncidentStatus(ctx, "nonexistent", incident.StatusResolved, &startedAt)
	if err == nil {
		t.Fatal("UpdateIncidentStatus() should have failed for non-existent incident")
	}
}

func TestCompleteIncident(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()
	event := createTestEvent("fault-004")
	inc := createTestIncident("inc-004", event)

	// Create incident
	err := store.CreateIncident(ctx, inc, event)
	if err != nil {
		t.Fatalf("CreateIncident() error = %v", err)
	}

	tests := []struct {
		name          string
		exitCode      int
		failureReason string
		wantStatus    string
	}{
		{
			name:       "successful completion",
			exitCode:   0,
			wantStatus: incident.StatusResolved,
		},
		{
			name:          "failed completion",
			exitCode:      1,
			failureReason: "agent failed",
			wantStatus:    incident.StatusFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new incident for each test case
			testEvent := createTestEvent("fault-" + tt.name)
			testInc := createTestIncident("inc-"+tt.name, testEvent)
			err := store.CreateIncident(ctx, testInc, testEvent)
			if err != nil {
				t.Fatalf("CreateIncident() error = %v", err)
			}

			// Complete incident
			err = store.CompleteIncident(ctx, testInc.IncidentID, tt.exitCode, tt.failureReason)
			if err != nil {
				t.Fatalf("CompleteIncident() error = %v", err)
			}

			// Verify completion
			retrieved, err := store.GetIncident(ctx, testInc.IncidentID)
			if err != nil {
				t.Fatalf("GetIncident() error = %v", err)
			}
			if retrieved.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", retrieved.Status, tt.wantStatus)
			}
			if retrieved.ExitCode == nil {
				t.Fatal("ExitCode is nil")
			}
			if *retrieved.ExitCode != tt.exitCode {
				t.Errorf("ExitCode = %v, want %v", *retrieved.ExitCode, tt.exitCode)
			}
			if retrieved.CompletedAt == nil {
				t.Fatal("CompletedAt is nil")
			}
			if tt.failureReason != "" && retrieved.FailureReason != tt.failureReason {
				t.Errorf("FailureReason = %v, want %v", retrieved.FailureReason, tt.failureReason)
			}
		})
	}
}

func TestRecordAgentExecution(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()
	event := createTestEvent("fault-005")
	inc := createTestIncident("inc-005", event)

	// Create incident first
	err := store.CreateIncident(ctx, inc, event)
	if err != nil {
		t.Fatalf("CreateIncident() error = %v", err)
	}

	// Record execution
	exec := &storage.AgentExecution{
		ExecutionID: "exec-001",
		IncidentID:  inc.IncidentID,
		StartedAt:   time.Now(),
		LogPaths: map[string]string{
			"stdout": "/path/to/stdout.log",
			"stderr": "/path/to/stderr.log",
		},
	}

	err = store.RecordAgentExecution(ctx, exec)
	if err != nil {
		t.Fatalf("RecordAgentExecution() error = %v", err)
	}

	// Verify execution was recorded
	var retrievedExecID string
	var logPathsJSON []byte
	err = store.db.QueryRowContext(ctx, "SELECT execution_id, log_paths FROM agent_executions WHERE execution_id = ?", exec.ExecutionID).
		Scan(&retrievedExecID, &logPathsJSON)
	if err != nil {
		t.Fatalf("Failed to retrieve execution: %v", err)
	}
	if retrievedExecID != exec.ExecutionID {
		t.Errorf("ExecutionID = %v, want %v", retrievedExecID, exec.ExecutionID)
	}

	// Verify log paths JSON
	var logPaths map[string]string
	err = json.Unmarshal(logPathsJSON, &logPaths)
	if err != nil {
		t.Fatalf("Failed to unmarshal log paths: %v", err)
	}
	if logPaths["stdout"] != "/path/to/stdout.log" {
		t.Errorf("stdout log path = %v, want /path/to/stdout.log", logPaths["stdout"])
	}
}

func TestRecordAgentExecution_Update(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()
	event := createTestEvent("fault-006")
	inc := createTestIncident("inc-006", event)

	// Create incident first
	err := store.CreateIncident(ctx, inc, event)
	if err != nil {
		t.Fatalf("CreateIncident() error = %v", err)
	}

	// Record execution start
	exec := &storage.AgentExecution{
		ExecutionID: "exec-002",
		IncidentID:  inc.IncidentID,
		StartedAt:   time.Now(),
	}

	err = store.RecordAgentExecution(ctx, exec)
	if err != nil {
		t.Fatalf("RecordAgentExecution() start error = %v", err)
	}

	// Update execution with completion info
	completedAt := time.Now()
	exitCode := 0
	exec.CompletedAt = &completedAt
	exec.ExitCode = &exitCode

	err = store.RecordAgentExecution(ctx, exec)
	if err != nil {
		t.Fatalf("RecordAgentExecution() update error = %v", err)
	}

	// Verify update
	var retrievedExitCode sql.NullInt64
	err = store.db.QueryRowContext(ctx, "SELECT exit_code FROM agent_executions WHERE execution_id = ?", exec.ExecutionID).
		Scan(&retrievedExitCode)
	if err != nil {
		t.Fatalf("Failed to retrieve execution: %v", err)
	}
	if !retrievedExitCode.Valid || retrievedExitCode.Int64 != 0 {
		t.Errorf("ExitCode = %v, want 0", retrievedExitCode.Int64)
	}
}

func TestRecordTriageReport(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()
	event := createTestEvent("fault-007")
	inc := createTestIncident("inc-007", event)

	// Create incident and execution
	err := store.CreateIncident(ctx, inc, event)
	if err != nil {
		t.Fatalf("CreateIncident() error = %v", err)
	}

	exec := &storage.AgentExecution{
		ExecutionID: "exec-003",
		IncidentID:  inc.IncidentID,
		StartedAt:   time.Now(),
	}
	err = store.RecordAgentExecution(ctx, exec)
	if err != nil {
		t.Fatalf("RecordAgentExecution() error = %v", err)
	}

	// Record triage report
	report := &storage.TriageReport{
		ReportID:       "report-001",
		IncidentID:     inc.IncidentID,
		ExecutionID:    exec.ExecutionID,
		GeneratedAt:    time.Now(),
		ReportMarkdown: "# Investigation Report\n\nDetails here...",
		ReportHTML:     "<h1>Investigation Report</h1><p>Details here...</p>",
	}

	err = store.RecordTriageReport(ctx, report)
	if err != nil {
		t.Fatalf("RecordTriageReport() error = %v", err)
	}

	// Verify report was recorded
	var retrievedReportID string
	var markdown string
	err = store.db.QueryRowContext(ctx, "SELECT report_id, report_markdown FROM triage_reports WHERE report_id = ?", report.ReportID).
		Scan(&retrievedReportID, &markdown)
	if err != nil {
		t.Fatalf("Failed to retrieve report: %v", err)
	}
	if retrievedReportID != report.ReportID {
		t.Errorf("ReportID = %v, want %v", retrievedReportID, report.ReportID)
	}
	if markdown != report.ReportMarkdown {
		t.Errorf("ReportMarkdown = %v, want %v", markdown, report.ReportMarkdown)
	}
}

func TestUpdateExecutionActivity(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()

	t.Run("update activity with rotation", func(t *testing.T) {
		event := createTestEvent("fault-008")
		inc := createTestIncident("inc-008", event)

		// Create incident first
		if err := store.CreateIncident(ctx, inc, event); err != nil {
			t.Fatalf("failed to create incident: %v", err)
		}

		// Create execution
		exec := &storage.AgentExecution{
			ExecutionID: "exec-activity-001",
			IncidentID:  inc.IncidentID,
			StartedAt:   time.Now(),
		}
		if err := store.RecordAgentExecution(ctx, exec); err != nil {
			t.Fatalf("failed to record execution: %v", err)
		}

		// Mark run as started (required for UpdateExecutionActivity to find the execution)
		if err := store.UpdateRunStarted(ctx, inc.IncidentID, time.Now()); err != nil {
			t.Fatalf("failed to update run started: %v", err)
		}

		// First activity update - uses incident_id, not execution_id
		activity1Time := time.Now()
		err := store.UpdateExecutionActivity(ctx, inc.IncidentID, "analyzing logs", activity1Time)
		if err != nil {
			t.Fatalf("failed to update activity: %v", err)
		}

		// Verify first activity is set as current
		var currentActivity sql.NullString
		var currentActivityStartedAt sql.NullTime
		var lastActivity sql.NullString
		var lastActivityFinishedAt sql.NullTime

		err = store.db.QueryRowContext(ctx, `
			SELECT current_activity, current_activity_started_at, last_activity, last_activity_finished_at
			FROM agent_executions
			WHERE execution_id = ?
		`, exec.ExecutionID).Scan(&currentActivity, &currentActivityStartedAt, &lastActivity, &lastActivityFinishedAt)
		if err != nil {
			t.Fatalf("failed to query execution: %v", err)
		}

		if !currentActivity.Valid || currentActivity.String != "analyzing logs" {
			t.Errorf("expected current_activity 'analyzing logs', got %v", currentActivity)
		}
		if !currentActivityStartedAt.Valid {
			t.Error("expected current_activity_started_at to be set")
		}
		if lastActivity.Valid {
			t.Error("expected last_activity to be NULL after first update")
		}
		if lastActivityFinishedAt.Valid {
			t.Error("expected last_activity_finished_at to be NULL after first update")
		}

		// Second activity update - this should rotate first activity to last
		time.Sleep(10 * time.Millisecond) // Ensure time difference
		activity2Time := time.Now()
		err = store.UpdateExecutionActivity(ctx, inc.IncidentID, "generating report", activity2Time)
		if err != nil {
			t.Fatalf("failed to update activity second time: %v", err)
		}

		// Verify rotation occurred
		err = store.db.QueryRowContext(ctx, `
			SELECT current_activity, current_activity_started_at, last_activity, last_activity_finished_at
			FROM agent_executions
			WHERE execution_id = ?
		`, exec.ExecutionID).Scan(&currentActivity, &currentActivityStartedAt, &lastActivity, &lastActivityFinishedAt)
		if err != nil {
			t.Fatalf("failed to query execution after second update: %v", err)
		}

		if !currentActivity.Valid || currentActivity.String != "generating report" {
			t.Errorf("expected current_activity 'generating report', got %v", currentActivity)
		}
		if !lastActivity.Valid || lastActivity.String != "analyzing logs" {
			t.Errorf("expected last_activity 'analyzing logs', got %v", lastActivity)
		}
		if !lastActivityFinishedAt.Valid {
			t.Error("expected last_activity_finished_at to be set after rotation")
		}

		// Verify timestamps rotated correctly
		if lastActivityFinishedAt.Time.After(currentActivityStartedAt.Time) {
			t.Error("last_activity_finished_at should be before current_activity_started_at")
		}
	})

	t.Run("update nonexistent execution", func(t *testing.T) {
		err := store.UpdateExecutionActivity(ctx, "nonexistent-id", "activity", time.Now())
		if err == nil {
			t.Fatal("expected error for nonexistent execution")
		}
	})

	t.Run("multiple activity updates", func(t *testing.T) {
		event := createTestEvent("fault-009")
		inc := createTestIncident("inc-009", event)

		// Create incident first
		if err := store.CreateIncident(ctx, inc, event); err != nil {
			t.Fatalf("failed to create incident: %v", err)
		}

		// Create execution
		exec := &storage.AgentExecution{
			ExecutionID: "exec-activity-002",
			IncidentID:  inc.IncidentID,
			StartedAt:   time.Now(),
		}
		if err := store.RecordAgentExecution(ctx, exec); err != nil {
			t.Fatalf("failed to record execution: %v", err)
		}

		// Mark run as started (required for UpdateExecutionActivity to find the execution)
		if err := store.UpdateRunStarted(ctx, inc.IncidentID, time.Now()); err != nil {
			t.Fatalf("failed to update run started: %v", err)
		}

		// Perform multiple activity updates - uses incident_id, not execution_id
		activities := []string{"starting", "analyzing", "investigating", "reporting"}
		for _, activity := range activities {
			time.Sleep(5 * time.Millisecond)
			err := store.UpdateExecutionActivity(ctx, inc.IncidentID, activity, time.Now())
			if err != nil {
				t.Fatalf("failed to update activity to %s: %v", activity, err)
			}
		}

		// Verify last two activities
		var currentActivity sql.NullString
		var lastActivity sql.NullString
		err := store.db.QueryRowContext(ctx, `
			SELECT current_activity, last_activity
			FROM agent_executions
			WHERE execution_id = ?
		`, exec.ExecutionID).Scan(&currentActivity, &lastActivity)
		if err != nil {
			t.Fatalf("failed to query execution: %v", err)
		}

		if !currentActivity.Valid || currentActivity.String != "reporting" {
			t.Errorf("expected current_activity 'reporting', got %v", currentActivity)
		}
		if !lastActivity.Valid || lastActivity.String != "investigating" {
			t.Errorf("expected last_activity 'investigating', got %v", lastActivity)
		}
	})
}

func TestGetIncident_NotFound(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()

	// Try to get non-existent incident
	retrieved, err := store.GetIncident(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetIncident() error = %v", err)
	}
	if retrieved != nil {
		t.Error("GetIncident() should return nil for non-existent incident")
	}
}

func TestListIncidents(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()

	// Create multiple incidents
	for i := 0; i < 5; i++ {
		event := createTestEvent(fmt.Sprintf("fault-%03d", i))
		inc := createTestIncident(fmt.Sprintf("inc-%03d", i), event)
		if i < 3 {
			inc.Status = incident.StatusResolved
		} else {
			inc.Status = incident.StatusInvestigating
		}
		err := store.CreateIncident(ctx, inc, event)
		if err != nil {
			t.Fatalf("CreateIncident() error = %v", err)
		}
	}

	tests := []struct {
		name    string
		filters *storage.IncidentFilters
		want    int
	}{
		{
			name:    "list all",
			filters: nil,
			want:    5,
		},
		{
			name: "filter by status",
			filters: &storage.IncidentFilters{
				Status: []string{incident.StatusResolved},
			},
			want: 3,
		},
		{
			name: "filter by cluster",
			filters: &storage.IncidentFilters{
				Cluster: "test-cluster",
			},
			want: 5,
		},
		{
			name: "limit results",
			filters: &storage.IncidentFilters{
				Limit: 2,
			},
			want: 2,
		},
		{
			name: "pagination",
			filters: &storage.IncidentFilters{
				Limit:  2,
				Offset: 2,
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			incidents, err := store.ListIncidents(ctx, tt.filters)
			if err != nil {
				t.Fatalf("ListIncidents() error = %v", err)
			}
			if len(incidents) != tt.want {
				t.Errorf("ListIncidents() returned %d incidents, want %d", len(incidents), tt.want)
			}
		})
	}
}

func TestListIncidents_TimeRange(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	tomorrow := now.Add(24 * time.Hour)

	// Create incidents
	event := createTestEvent("fault-time")
	inc := createTestIncident("inc-time", event)
	err := store.CreateIncident(ctx, inc, event)
	if err != nil {
		t.Fatalf("CreateIncident() error = %v", err)
	}

	tests := []struct {
		name    string
		filters *storage.IncidentFilters
		want    int
	}{
		{
			name: "created after yesterday",
			filters: &storage.IncidentFilters{
				CreatedAfter: &yesterday,
			},
			want: 1,
		},
		{
			name: "created before tomorrow",
			filters: &storage.IncidentFilters{
				CreatedBefore: &tomorrow,
			},
			want: 1,
		},
		{
			name: "created after tomorrow",
			filters: &storage.IncidentFilters{
				CreatedAfter: &tomorrow,
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			incidents, err := store.ListIncidents(ctx, tt.filters)
			if err != nil {
				t.Fatalf("ListIncidents() error = %v", err)
			}
			if len(incidents) != tt.want {
				t.Errorf("ListIncidents() returned %d incidents, want %d", len(incidents), tt.want)
			}
		})
	}
}

func TestConcurrentAccess(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()
	numGoroutines := 10
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			event := createTestEvent(fmt.Sprintf("fault-concurrent-%d", id))
			inc := createTestIncident(fmt.Sprintf("inc-concurrent-%d", id), event)
			err := store.CreateIncident(ctx, inc, event)
			if err != nil {
				t.Errorf("CreateIncident() error = %v", err)
			}
		}(i)
	}

	wg.Wait()

	// Verify all incidents were created
	incidents, err := store.ListIncidents(ctx, nil)
	if err != nil {
		t.Fatalf("ListIncidents() error = %v", err)
	}
	if len(incidents) != numGoroutines {
		t.Errorf("ListIncidents() returned %d incidents, want %d", len(incidents), numGoroutines)
	}
}

func TestClose(t *testing.T) {
	store := setupTestStore(t)

	err := store.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Verify database is closed by trying to use it
	ctx := context.Background()
	_, err = store.GetIncident(ctx, "test")
	if err == nil {
		t.Error("GetIncident() should fail after Close()")
	}
}

// =============================================================================
// Cluster Storage Tests
// =============================================================================

func TestMonitoredCluster_CRUD(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()

	t.Run("create and get", func(t *testing.T) {
		cluster := &storage.MonitoredClusterRecord{
			Name:               "test-cluster-1",
			Environment:        "production",
			Labels:             map[string]string{"region": "us-east-1", "tier": "critical"},
			MCPEndpoint:        "http://mcp-server:8080/mcp",
			MCPAPIKey:          "test-api-key",
			TriageEnabled:      true,
			TargetKubeconfig:   "apiVersion: v1\nkind: Config\n...",
			AllowSecretsAccess: false,
			ExecutionCluster:   "exec-cluster-1",
			Source:             "yaml",
		}

		err := store.UpsertMonitoredCluster(ctx, cluster)
		if err != nil {
			t.Fatalf("UpsertMonitoredCluster() error = %v", err)
		}

		retrieved, err := store.GetMonitoredCluster(ctx, cluster.Name)
		if err != nil {
			t.Fatalf("GetMonitoredCluster() error = %v", err)
		}
		if retrieved == nil {
			t.Fatal("GetMonitoredCluster() returned nil")
		}

		// Verify fields
		if retrieved.Name != cluster.Name {
			t.Errorf("Name = %v, want %v", retrieved.Name, cluster.Name)
		}
		if retrieved.Environment != cluster.Environment {
			t.Errorf("Environment = %v, want %v", retrieved.Environment, cluster.Environment)
		}
		if retrieved.MCPEndpoint != cluster.MCPEndpoint {
			t.Errorf("MCPEndpoint = %v, want %v", retrieved.MCPEndpoint, cluster.MCPEndpoint)
		}
		if retrieved.TriageEnabled != cluster.TriageEnabled {
			t.Errorf("TriageEnabled = %v, want %v", retrieved.TriageEnabled, cluster.TriageEnabled)
		}
		if retrieved.Labels["region"] != "us-east-1" {
			t.Errorf("Labels[region] = %v, want us-east-1", retrieved.Labels["region"])
		}
		if retrieved.Source != "yaml" {
			t.Errorf("Source = %v, want yaml", retrieved.Source)
		}
	})

	t.Run("update existing", func(t *testing.T) {
		cluster := &storage.MonitoredClusterRecord{
			Name:          "test-cluster-2",
			Environment:   "staging",
			MCPEndpoint:   "http://mcp-staging:8080/mcp",
			TriageEnabled: false,
			Source:        "yaml",
		}

		// Create
		err := store.UpsertMonitoredCluster(ctx, cluster)
		if err != nil {
			t.Fatalf("Initial UpsertMonitoredCluster() error = %v", err)
		}

		// Update
		cluster.Environment = "production"
		cluster.TriageEnabled = true
		err = store.UpsertMonitoredCluster(ctx, cluster)
		if err != nil {
			t.Fatalf("Update UpsertMonitoredCluster() error = %v", err)
		}

		// Verify update
		retrieved, err := store.GetMonitoredCluster(ctx, cluster.Name)
		if err != nil {
			t.Fatalf("GetMonitoredCluster() error = %v", err)
		}
		if retrieved.Environment != "production" {
			t.Errorf("Environment = %v, want production", retrieved.Environment)
		}
		if !retrieved.TriageEnabled {
			t.Error("TriageEnabled should be true after update")
		}
	})

	t.Run("get non-existent", func(t *testing.T) {
		retrieved, err := store.GetMonitoredCluster(ctx, "non-existent-cluster")
		if err != nil {
			t.Fatalf("GetMonitoredCluster() error = %v", err)
		}
		if retrieved != nil {
			t.Error("GetMonitoredCluster() should return nil for non-existent cluster")
		}
	})

	t.Run("delete", func(t *testing.T) {
		cluster := &storage.MonitoredClusterRecord{
			Name:        "test-cluster-delete",
			MCPEndpoint: "http://mcp:8080/mcp",
			Source:      "database",
		}

		err := store.UpsertMonitoredCluster(ctx, cluster)
		if err != nil {
			t.Fatalf("UpsertMonitoredCluster() error = %v", err)
		}

		err = store.DeleteMonitoredCluster(ctx, cluster.Name)
		if err != nil {
			t.Fatalf("DeleteMonitoredCluster() error = %v", err)
		}

		retrieved, err := store.GetMonitoredCluster(ctx, cluster.Name)
		if err != nil {
			t.Fatalf("GetMonitoredCluster() after delete error = %v", err)
		}
		if retrieved != nil {
			t.Error("Cluster should be nil after delete")
		}
	})
}

func TestMonitoredCluster_List(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()

	// Create multiple clusters
	clusters := []*storage.MonitoredClusterRecord{
		{Name: "cluster-a", Environment: "production", MCPEndpoint: "http://a:8080/mcp", Source: "yaml"},
		{Name: "cluster-b", Environment: "staging", MCPEndpoint: "http://b:8080/mcp", Source: "yaml"},
		{Name: "cluster-c", Environment: "production", MCPEndpoint: "http://c:8080/mcp", Source: "database"},
	}

	for _, c := range clusters {
		if err := store.UpsertMonitoredCluster(ctx, c); err != nil {
			t.Fatalf("UpsertMonitoredCluster() error = %v", err)
		}
	}

	list, err := store.ListMonitoredClusters(ctx)
	if err != nil {
		t.Fatalf("ListMonitoredClusters() error = %v", err)
	}

	if len(list) != 3 {
		t.Errorf("ListMonitoredClusters() returned %d clusters, want 3", len(list))
	}

	// Verify ordering (by name)
	if list[0].Name != "cluster-a" {
		t.Errorf("First cluster name = %v, want cluster-a", list[0].Name)
	}
}

func TestExecutionCluster_CRUD(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()

	t.Run("create and get", func(t *testing.T) {
		cluster := &storage.ExecutionClusterRecord{
			Name:                "exec-cluster-1",
			Kubeconfig:          "apiVersion: v1\nkind: Config\n...",
			Namespace:           "nightcrier",
			RunnerImage:         "nc-agent-runner:v1.0.0",
			ImagePullPolicy:     "Always",
			Timeout:             900,
			MemoryLimit:         "4Gi",
			CPULimit:            "2",
			CleanupTTL:          7200,
			MaxConcurrentAgents: 5,
			Source:              "yaml",
		}

		err := store.UpsertExecutionCluster(ctx, cluster)
		if err != nil {
			t.Fatalf("UpsertExecutionCluster() error = %v", err)
		}

		retrieved, err := store.GetExecutionCluster(ctx, cluster.Name)
		if err != nil {
			t.Fatalf("GetExecutionCluster() error = %v", err)
		}
		if retrieved == nil {
			t.Fatal("GetExecutionCluster() returned nil")
		}

		// Verify fields
		if retrieved.Name != cluster.Name {
			t.Errorf("Name = %v, want %v", retrieved.Name, cluster.Name)
		}
		if retrieved.Namespace != cluster.Namespace {
			t.Errorf("Namespace = %v, want %v", retrieved.Namespace, cluster.Namespace)
		}
		if retrieved.RunnerImage != cluster.RunnerImage {
			t.Errorf("RunnerImage = %v, want %v", retrieved.RunnerImage, cluster.RunnerImage)
		}
		if retrieved.Timeout != cluster.Timeout {
			t.Errorf("Timeout = %v, want %v", retrieved.Timeout, cluster.Timeout)
		}
		if retrieved.MemoryLimit != cluster.MemoryLimit {
			t.Errorf("MemoryLimit = %v, want %v", retrieved.MemoryLimit, cluster.MemoryLimit)
		}
		if retrieved.MaxConcurrentAgents != cluster.MaxConcurrentAgents {
			t.Errorf("MaxConcurrentAgents = %v, want %v", retrieved.MaxConcurrentAgents, cluster.MaxConcurrentAgents)
		}
	})

	t.Run("update existing", func(t *testing.T) {
		cluster := &storage.ExecutionClusterRecord{
			Name:            "exec-cluster-2",
			Kubeconfig:      "initial-config",
			Namespace:       "default",
			RunnerImage:     "runner:v1",
			ImagePullPolicy: "IfNotPresent",
			Timeout:         600,
			MemoryLimit:     "2Gi",
			CPULimit:        "1",
			CleanupTTL:      3600,
			Source:          "yaml",
		}

		// Create
		if err := store.UpsertExecutionCluster(ctx, cluster); err != nil {
			t.Fatalf("Initial UpsertExecutionCluster() error = %v", err)
		}

		// Update
		cluster.Namespace = "nightcrier-prod"
		cluster.RunnerImage = "runner:v2"
		cluster.MemoryLimit = "4Gi"
		if err := store.UpsertExecutionCluster(ctx, cluster); err != nil {
			t.Fatalf("Update UpsertExecutionCluster() error = %v", err)
		}

		// Verify update
		retrieved, err := store.GetExecutionCluster(ctx, cluster.Name)
		if err != nil {
			t.Fatalf("GetExecutionCluster() error = %v", err)
		}
		if retrieved.Namespace != "nightcrier-prod" {
			t.Errorf("Namespace = %v, want nightcrier-prod", retrieved.Namespace)
		}
		if retrieved.RunnerImage != "runner:v2" {
			t.Errorf("RunnerImage = %v, want runner:v2", retrieved.RunnerImage)
		}
		if retrieved.MemoryLimit != "4Gi" {
			t.Errorf("MemoryLimit = %v, want 4Gi", retrieved.MemoryLimit)
		}
	})

	t.Run("get non-existent", func(t *testing.T) {
		retrieved, err := store.GetExecutionCluster(ctx, "non-existent-exec")
		if err != nil {
			t.Fatalf("GetExecutionCluster() error = %v", err)
		}
		if retrieved != nil {
			t.Error("GetExecutionCluster() should return nil for non-existent cluster")
		}
	})

	t.Run("delete", func(t *testing.T) {
		cluster := &storage.ExecutionClusterRecord{
			Name:            "exec-cluster-delete",
			Kubeconfig:      "config",
			Namespace:       "test",
			RunnerImage:     "runner:latest",
			ImagePullPolicy: "Never",
			Timeout:         300,
			MemoryLimit:     "1Gi",
			CPULimit:        "500m",
			CleanupTTL:      1800,
			Source:          "database",
		}

		if err := store.UpsertExecutionCluster(ctx, cluster); err != nil {
			t.Fatalf("UpsertExecutionCluster() error = %v", err)
		}

		if err := store.DeleteExecutionCluster(ctx, cluster.Name); err != nil {
			t.Fatalf("DeleteExecutionCluster() error = %v", err)
		}

		retrieved, err := store.GetExecutionCluster(ctx, cluster.Name)
		if err != nil {
			t.Fatalf("GetExecutionCluster() after delete error = %v", err)
		}
		if retrieved != nil {
			t.Error("Cluster should be nil after delete")
		}
	})
}

func TestExecutionCluster_List(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()

	// Create multiple clusters
	clusters := []*storage.ExecutionClusterRecord{
		{Name: "exec-a", Kubeconfig: "config-a", Namespace: "ns-a", RunnerImage: "img", ImagePullPolicy: "Always", Timeout: 600, MemoryLimit: "2Gi", CPULimit: "1", CleanupTTL: 3600, Source: "yaml"},
		{Name: "exec-b", Kubeconfig: "config-b", Namespace: "ns-b", RunnerImage: "img", ImagePullPolicy: "Always", Timeout: 600, MemoryLimit: "2Gi", CPULimit: "1", CleanupTTL: 3600, Source: "database"},
	}

	for _, c := range clusters {
		if err := store.UpsertExecutionCluster(ctx, c); err != nil {
			t.Fatalf("UpsertExecutionCluster() error = %v", err)
		}
	}

	list, err := store.ListExecutionClusters(ctx)
	if err != nil {
		t.Fatalf("ListExecutionClusters() error = %v", err)
	}

	if len(list) != 2 {
		t.Errorf("ListExecutionClusters() returned %d clusters, want 2", len(list))
	}

	// Verify ordering (by name)
	if list[0].Name != "exec-a" {
		t.Errorf("First cluster name = %v, want exec-a", list[0].Name)
	}
}

func TestSyncMonitoredClustersFromYAML(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()

	t.Run("initial sync", func(t *testing.T) {
		clusters := []storage.MonitoredClusterRecord{
			{Name: "yaml-cluster-1", MCPEndpoint: "http://1:8080/mcp"},
			{Name: "yaml-cluster-2", MCPEndpoint: "http://2:8080/mcp"},
		}

		err := store.SyncMonitoredClustersFromYAML(ctx, clusters)
		if err != nil {
			t.Fatalf("SyncMonitoredClustersFromYAML() error = %v", err)
		}

		list, err := store.ListMonitoredClusters(ctx)
		if err != nil {
			t.Fatalf("ListMonitoredClusters() error = %v", err)
		}

		if len(list) != 2 {
			t.Errorf("Expected 2 clusters, got %d", len(list))
		}

		// Verify source is set to yaml
		for _, c := range list {
			if c.Source != "yaml" {
				t.Errorf("Cluster %s source = %v, want yaml", c.Name, c.Source)
			}
		}
	})

	t.Run("sync removes old yaml clusters", func(t *testing.T) {
		// Initial sync with 2 clusters
		initial := []storage.MonitoredClusterRecord{
			{Name: "keep-cluster", MCPEndpoint: "http://keep:8080/mcp"},
			{Name: "remove-cluster", MCPEndpoint: "http://remove:8080/mcp"},
		}
		if err := store.SyncMonitoredClustersFromYAML(ctx, initial); err != nil {
			t.Fatalf("Initial sync error = %v", err)
		}

		// Sync with only 1 cluster (remove-cluster should be deleted)
		updated := []storage.MonitoredClusterRecord{
			{Name: "keep-cluster", MCPEndpoint: "http://keep:8080/mcp"},
		}
		if err := store.SyncMonitoredClustersFromYAML(ctx, updated); err != nil {
			t.Fatalf("Updated sync error = %v", err)
		}

		// Verify remove-cluster was deleted
		removed, err := store.GetMonitoredCluster(ctx, "remove-cluster")
		if err != nil {
			t.Fatalf("GetMonitoredCluster() error = %v", err)
		}
		if removed != nil {
			t.Error("remove-cluster should have been deleted")
		}

		// Verify keep-cluster still exists
		kept, err := store.GetMonitoredCluster(ctx, "keep-cluster")
		if err != nil {
			t.Fatalf("GetMonitoredCluster() error = %v", err)
		}
		if kept == nil {
			t.Error("keep-cluster should still exist")
		}
	})

	t.Run("sync preserves database clusters", func(t *testing.T) {
		// Create a database-sourced cluster
		dbCluster := &storage.MonitoredClusterRecord{
			Name:        "db-cluster",
			MCPEndpoint: "http://db:8080/mcp",
			Source:      "database",
		}
		if err := store.UpsertMonitoredCluster(ctx, dbCluster); err != nil {
			t.Fatalf("UpsertMonitoredCluster() error = %v", err)
		}

		// Sync with yaml clusters (should not affect db-cluster)
		yamlClusters := []storage.MonitoredClusterRecord{
			{Name: "yaml-only", MCPEndpoint: "http://yaml:8080/mcp"},
		}
		if err := store.SyncMonitoredClustersFromYAML(ctx, yamlClusters); err != nil {
			t.Fatalf("SyncMonitoredClustersFromYAML() error = %v", err)
		}

		// Verify db-cluster is preserved
		db, err := store.GetMonitoredCluster(ctx, "db-cluster")
		if err != nil {
			t.Fatalf("GetMonitoredCluster() error = %v", err)
		}
		if db == nil {
			t.Error("db-cluster should be preserved after yaml sync")
		}
	})
}

func TestSyncExecutionClustersFromYAML(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()

	t.Run("initial sync", func(t *testing.T) {
		clusters := []storage.ExecutionClusterRecord{
			{Name: "yaml-exec-1", Kubeconfig: "c1", Namespace: "ns1", RunnerImage: "img", ImagePullPolicy: "Always", Timeout: 600, MemoryLimit: "2Gi", CPULimit: "1", CleanupTTL: 3600},
			{Name: "yaml-exec-2", Kubeconfig: "c2", Namespace: "ns2", RunnerImage: "img", ImagePullPolicy: "Always", Timeout: 600, MemoryLimit: "2Gi", CPULimit: "1", CleanupTTL: 3600},
		}

		err := store.SyncExecutionClustersFromYAML(ctx, clusters)
		if err != nil {
			t.Fatalf("SyncExecutionClustersFromYAML() error = %v", err)
		}

		list, err := store.ListExecutionClusters(ctx)
		if err != nil {
			t.Fatalf("ListExecutionClusters() error = %v", err)
		}

		if len(list) != 2 {
			t.Errorf("Expected 2 clusters, got %d", len(list))
		}

		// Verify source is set to yaml
		for _, c := range list {
			if c.Source != "yaml" {
				t.Errorf("Cluster %s source = %v, want yaml", c.Name, c.Source)
			}
		}
	})

	t.Run("sync removes old yaml clusters", func(t *testing.T) {
		// Create a new store for isolation
		store2 := setupTestStore(t)
		defer store2.Close()

		initial := []storage.ExecutionClusterRecord{
			{Name: "keep-exec", Kubeconfig: "c", Namespace: "ns", RunnerImage: "img", ImagePullPolicy: "Always", Timeout: 600, MemoryLimit: "2Gi", CPULimit: "1", CleanupTTL: 3600},
			{Name: "remove-exec", Kubeconfig: "c", Namespace: "ns", RunnerImage: "img", ImagePullPolicy: "Always", Timeout: 600, MemoryLimit: "2Gi", CPULimit: "1", CleanupTTL: 3600},
		}
		if err := store2.SyncExecutionClustersFromYAML(ctx, initial); err != nil {
			t.Fatalf("Initial sync error = %v", err)
		}

		updated := []storage.ExecutionClusterRecord{
			{Name: "keep-exec", Kubeconfig: "c", Namespace: "ns", RunnerImage: "img", ImagePullPolicy: "Always", Timeout: 600, MemoryLimit: "2Gi", CPULimit: "1", CleanupTTL: 3600},
		}
		if err := store2.SyncExecutionClustersFromYAML(ctx, updated); err != nil {
			t.Fatalf("Updated sync error = %v", err)
		}

		removed, _ := store2.GetExecutionCluster(ctx, "remove-exec")
		if removed != nil {
			t.Error("remove-exec should have been deleted")
		}

		kept, _ := store2.GetExecutionCluster(ctx, "keep-exec")
		if kept == nil {
			t.Error("keep-exec should still exist")
		}
	})

	t.Run("sync preserves database clusters", func(t *testing.T) {
		// Create a new store for isolation
		store3 := setupTestStore(t)
		defer store3.Close()

		dbCluster := &storage.ExecutionClusterRecord{
			Name:            "db-exec",
			Kubeconfig:      "c",
			Namespace:       "ns",
			RunnerImage:     "img",
			ImagePullPolicy: "Always",
			Timeout:         600,
			MemoryLimit:     "2Gi",
			CPULimit:        "1",
			CleanupTTL:      3600,
			Source:          "database",
		}
		if err := store3.UpsertExecutionCluster(ctx, dbCluster); err != nil {
			t.Fatalf("UpsertExecutionCluster() error = %v", err)
		}

		yamlClusters := []storage.ExecutionClusterRecord{
			{Name: "yaml-exec-only", Kubeconfig: "c", Namespace: "ns", RunnerImage: "img", ImagePullPolicy: "Always", Timeout: 600, MemoryLimit: "2Gi", CPULimit: "1", CleanupTTL: 3600},
		}
		if err := store3.SyncExecutionClustersFromYAML(ctx, yamlClusters); err != nil {
			t.Fatalf("SyncExecutionClustersFromYAML() error = %v", err)
		}

		db, _ := store3.GetExecutionCluster(ctx, "db-exec")
		if db == nil {
			t.Error("db-exec should be preserved after yaml sync")
		}
	})
}

func TestMonitoredCluster_LabelsJSON(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()

	t.Run("empty labels", func(t *testing.T) {
		cluster := &storage.MonitoredClusterRecord{
			Name:        "no-labels-cluster",
			MCPEndpoint: "http://mcp:8080/mcp",
			Labels:      nil,
			Source:      "yaml",
		}

		if err := store.UpsertMonitoredCluster(ctx, cluster); err != nil {
			t.Fatalf("UpsertMonitoredCluster() error = %v", err)
		}

		retrieved, err := store.GetMonitoredCluster(ctx, cluster.Name)
		if err != nil {
			t.Fatalf("GetMonitoredCluster() error = %v", err)
		}

		// Labels should be nil or empty, not cause an error
		if retrieved.Labels != nil && len(retrieved.Labels) > 0 {
			t.Errorf("Expected empty labels, got %v", retrieved.Labels)
		}
	})

	t.Run("complex labels", func(t *testing.T) {
		cluster := &storage.MonitoredClusterRecord{
			Name:        "complex-labels-cluster",
			MCPEndpoint: "http://mcp:8080/mcp",
			Labels: map[string]string{
				"region":      "us-west-2",
				"environment": "production",
				"team":        "platform",
				"cost-center": "engineering",
			},
			Source: "yaml",
		}

		if err := store.UpsertMonitoredCluster(ctx, cluster); err != nil {
			t.Fatalf("UpsertMonitoredCluster() error = %v", err)
		}

		retrieved, err := store.GetMonitoredCluster(ctx, cluster.Name)
		if err != nil {
			t.Fatalf("GetMonitoredCluster() error = %v", err)
		}

		if len(retrieved.Labels) != 4 {
			t.Errorf("Expected 4 labels, got %d", len(retrieved.Labels))
		}
		if retrieved.Labels["region"] != "us-west-2" {
			t.Errorf("Labels[region] = %v, want us-west-2", retrieved.Labels["region"])
		}
		if retrieved.Labels["cost-center"] != "engineering" {
			t.Errorf("Labels[cost-center] = %v, want engineering", retrieved.Labels["cost-center"])
		}
	})
}
