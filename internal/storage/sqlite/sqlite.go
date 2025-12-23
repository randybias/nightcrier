// Package sqlite provides a SQLite implementation of the StateStore interface.
// It uses an embedded SQLite database with WAL mode for better concurrency.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // SQLite driver

	"github.com/rbias/nightcrier/internal/events"
	"github.com/rbias/nightcrier/internal/incident"
	"github.com/rbias/nightcrier/internal/storage"
)

// Store implements the StateStore interface using SQLite.
// It provides an embedded database solution with connection pooling
// and WAL mode for improved concurrency performance.
type Store struct {
	db *sql.DB
}

// Config holds configuration options for the SQLite store.
type Config struct {
	// Path is the filesystem path to the SQLite database file.
	// Use ":memory:" for an in-memory database (useful for testing).
	// Default: "./nightcrier.db"
	Path string

	// BusyTimeout is the maximum time to wait for a locked database.
	// Default: 5 seconds
	BusyTimeout time.Duration

	// MaxOpenConns is the maximum number of open connections to the database.
	// Default: 25
	MaxOpenConns int

	// MaxIdleConns is the maximum number of idle connections.
	// Default: 5
	MaxIdleConns int

	// ConnMaxLifetime is the maximum amount of time a connection may be reused.
	// Default: 1 hour
	ConnMaxLifetime time.Duration
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Path:            "./nightcrier.db",
		BusyTimeout:     5 * time.Second,
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	}
}

// New creates a new SQLite store with the provided configuration.
// It initializes the database connection, configures connection pooling,
// and enables WAL mode for better concurrency.
//
// Example usage:
//
//	cfg := sqlite.DefaultConfig()
//	cfg.Path = "/var/lib/nightcrier/nightcrier.db"
//	store, err := sqlite.New(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer store.Close()
func New(cfg *Config) (*Store, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Resolve absolute path (except for :memory:)
	dbPath := cfg.Path
	if dbPath != ":memory:" {
		absPath, err := filepath.Abs(dbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve database path: %w", err)
		}
		dbPath = absPath
	}

	// Open database connection with pragmas
	// Enable WAL mode and busy timeout in connection string
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=%d&_foreign_keys=on",
		dbPath, int(cfg.BusyTimeout.Milliseconds()))

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Verify connection works
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Store{db: db}, nil
}

// CreateIncident creates a new incident from a fault event.
// It inserts both the fault event and incident records in a transaction.
// This ensures data consistency between the events and incidents tables.
func (s *Store) CreateIncident(ctx context.Context, inc *incident.Incident, event *events.FaultEvent) error {
	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert fault event first (due to foreign key constraint)
	// Use incident fields (inc) for consistency - these have been processed and enriched
	_, err = tx.ExecContext(ctx, `
		INSERT INTO fault_events (
			fault_id, subscription_id, cluster, received_at,
			resource_api_version, resource_kind, resource_name, resource_namespace, resource_uid,
			fault_type, severity, context, timestamp
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(fault_id) DO NOTHING
	`,
		event.FaultID,
		event.SubscriptionID,
		inc.Cluster,
		event.ReceivedAt,
		safeResourceField(inc.Resource, func(r *incident.ResourceInfo) string { return r.APIVersion }),
		safeResourceField(inc.Resource, func(r *incident.ResourceInfo) string { return r.Kind }),
		safeResourceField(inc.Resource, func(r *incident.ResourceInfo) string { return r.Name }),
		safeResourceField(inc.Resource, func(r *incident.ResourceInfo) string { return r.Namespace }),
		safeResourceField(inc.Resource, func(r *incident.ResourceInfo) string { return r.UID }),
		inc.FaultType,
		inc.Severity,
		inc.Context,
		inc.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to insert fault event: %w", err)
	}

	// Insert incident
	_, err = tx.ExecContext(ctx, `
		INSERT INTO incidents (
			incident_id, fault_id, triggering_event_id,
			status, created_at, started_at, completed_at,
			exit_code, failure_reason,
			cluster, namespace, fault_type, severity, context, timestamp,
			resource_api_version, resource_kind, resource_name, resource_namespace, resource_uid
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		inc.IncidentID,
		inc.FaultID,
		inc.TriggeringEventID,
		inc.Status,
		inc.CreatedAt,
		inc.StartedAt,
		inc.CompletedAt,
		inc.ExitCode,
		inc.FailureReason,
		inc.Cluster,
		inc.Namespace,
		inc.FaultType,
		inc.Severity,
		inc.Context,
		inc.Timestamp,
		safeResourceField(inc.Resource, func(r *incident.ResourceInfo) string { return r.APIVersion }),
		safeResourceField(inc.Resource, func(r *incident.ResourceInfo) string { return r.Kind }),
		safeResourceField(inc.Resource, func(r *incident.ResourceInfo) string { return r.Name }),
		safeResourceField(inc.Resource, func(r *incident.ResourceInfo) string { return r.Namespace }),
		safeResourceField(inc.Resource, func(r *incident.ResourceInfo) string { return r.UID }),
	)
	if err != nil {
		return fmt.Errorf("failed to insert incident: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UpdateIncidentStatus updates the status of an existing incident.
// This is called during state transitions (pending -> investigating, investigating -> resolved, etc.).
// The startedAt timestamp is set when transitioning to investigating status.
func (s *Store) UpdateIncidentStatus(ctx context.Context, incidentID string, status string, startedAt *time.Time) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE incidents
		SET status = ?, started_at = ?
		WHERE incident_id = ?
	`, status, startedAt, incidentID)
	if err != nil {
		return fmt.Errorf("failed to update incident status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("incident not found: %s", incidentID)
	}

	return nil
}

// CompleteIncident marks an incident as complete with final result information.
// This is called when the agent finishes execution (success or failure).
// Records the exit code, completion time, and any failure reason.
func (s *Store) CompleteIncident(ctx context.Context, incidentID string, exitCode int, failureReason string) error {
	now := time.Now()

	// Determine status based on exit code
	status := incident.StatusResolved
	if exitCode != 0 {
		status = incident.StatusFailed
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE incidents
		SET status = ?, completed_at = ?, exit_code = ?, failure_reason = ?
		WHERE incident_id = ?
	`, status, now, exitCode, failureReason, incidentID)
	if err != nil {
		return fmt.Errorf("failed to complete incident: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("incident not found: %s", incidentID)
	}

	return nil
}

// RecordAgentExecution records details of an agent execution attempt.
// This is called when starting and completing agent execution.
// Links the execution to its parent incident.
func (s *Store) RecordAgentExecution(ctx context.Context, exec *storage.AgentExecution) error {
	// Serialize log paths to JSON
	var logPathsJSON []byte
	var err error
	if exec.LogPaths != nil && len(exec.LogPaths) > 0 {
		logPathsJSON, err = json.Marshal(exec.LogPaths)
		if err != nil {
			return fmt.Errorf("failed to marshal log paths: %w", err)
		}
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO agent_executions (
			execution_id, incident_id,
			started_at, completed_at, exit_code, error_message,
			log_paths
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(execution_id) DO UPDATE SET
			completed_at = excluded.completed_at,
			exit_code = excluded.exit_code,
			error_message = excluded.error_message,
			log_paths = excluded.log_paths
	`,
		exec.ExecutionID,
		exec.IncidentID,
		exec.StartedAt,
		exec.CompletedAt,
		exec.ExitCode,
		exec.ErrorMessage,
		logPathsJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to record agent execution: %w", err)
	}

	return nil
}

// RecordTriageReport stores the investigation report generated by the agent.
// This is called after the agent produces investigation.md output.
// The report content is stored in markdown format.
func (s *Store) RecordTriageReport(ctx context.Context, report *storage.TriageReport) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO triage_reports (
			report_id, incident_id, execution_id,
			generated_at,
			report_markdown, report_html
		) VALUES (?, ?, ?, ?, ?, ?)
	`,
		report.ReportID,
		report.IncidentID,
		report.ExecutionID,
		report.GeneratedAt,
		report.ReportMarkdown,
		report.ReportHTML,
	)
	if err != nil {
		return fmt.Errorf("failed to record triage report: %w", err)
	}

	return nil
}

// GetIncident retrieves an incident by its ID.
// Returns nil if the incident is not found.
func (s *Store) GetIncident(ctx context.Context, incidentID string) (*incident.Incident, error) {
	var inc incident.Incident
	var startedAt, completedAt sql.NullTime
	var exitCode sql.NullInt64
	var failureReason sql.NullString
	var resourceAPIVersion, resourceKind, resourceName, resourceNamespace, resourceUID sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT
			incident_id, fault_id, triggering_event_id,
			status, created_at, started_at, completed_at,
			exit_code, failure_reason,
			cluster, namespace, fault_type, severity, context, timestamp,
			resource_api_version, resource_kind, resource_name, resource_namespace, resource_uid
		FROM incidents
		WHERE incident_id = ?
	`, incidentID).Scan(
		&inc.IncidentID,
		&inc.FaultID,
		&inc.TriggeringEventID,
		&inc.Status,
		&inc.CreatedAt,
		&startedAt,
		&completedAt,
		&exitCode,
		&failureReason,
		&inc.Cluster,
		&inc.Namespace,
		&inc.FaultType,
		&inc.Severity,
		&inc.Context,
		&inc.Timestamp,
		&resourceAPIVersion,
		&resourceKind,
		&resourceName,
		&resourceNamespace,
		&resourceUID,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get incident: %w", err)
	}

	// Handle nullable fields
	if startedAt.Valid {
		inc.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		inc.CompletedAt = &completedAt.Time
	}
	if exitCode.Valid {
		exitCodeInt := int(exitCode.Int64)
		inc.ExitCode = &exitCodeInt
	}
	if failureReason.Valid {
		inc.FailureReason = failureReason.String
	}

	// Reconstruct resource info if any fields are present
	if resourceKind.Valid || resourceName.Valid {
		inc.Resource = &incident.ResourceInfo{}
		if resourceAPIVersion.Valid {
			inc.Resource.APIVersion = resourceAPIVersion.String
		}
		if resourceKind.Valid {
			inc.Resource.Kind = resourceKind.String
		}
		if resourceName.Valid {
			inc.Resource.Name = resourceName.String
		}
		if resourceNamespace.Valid {
			inc.Resource.Namespace = resourceNamespace.String
		}
		if resourceUID.Valid {
			inc.Resource.UID = resourceUID.String
		}
	}

	return &inc, nil
}

// ListIncidents returns incidents matching the provided filters.
// Supports filtering by status, cluster, namespace, fault type, severity, and time range.
// Supports pagination via limit and offset.
func (s *Store) ListIncidents(ctx context.Context, filters *storage.IncidentFilters) ([]*incident.Incident, error) {
	query := `
		SELECT
			incident_id, fault_id, triggering_event_id,
			status, created_at, started_at, completed_at,
			exit_code, failure_reason,
			cluster, namespace, fault_type, severity, context, timestamp,
			resource_api_version, resource_kind, resource_name, resource_namespace, resource_uid
		FROM incidents
		WHERE 1=1
	`
	args := []interface{}{}

	// Apply filters
	if filters != nil {
		if len(filters.Status) > 0 {
			query += " AND status IN ("
			for i, status := range filters.Status {
				if i > 0 {
					query += ", "
				}
				query += "?"
				args = append(args, status)
			}
			query += ")"
		}
		if filters.Cluster != "" {
			query += " AND cluster = ?"
			args = append(args, filters.Cluster)
		}
		if filters.Namespace != "" {
			query += " AND namespace = ?"
			args = append(args, filters.Namespace)
		}
		if filters.FaultType != "" {
			query += " AND fault_type = ?"
			args = append(args, filters.FaultType)
		}
		if filters.Severity != "" {
			query += " AND severity = ?"
			args = append(args, filters.Severity)
		}
		if filters.CreatedAfter != nil {
			query += " AND created_at > ?"
			args = append(args, *filters.CreatedAfter)
		}
		if filters.CreatedBefore != nil {
			query += " AND created_at < ?"
			args = append(args, *filters.CreatedBefore)
		}
	}

	// Order by created_at descending (newest first)
	query += " ORDER BY created_at DESC"

	// Apply pagination
	if filters != nil {
		if filters.Limit > 0 {
			query += " LIMIT ?"
			args = append(args, filters.Limit)
		}
		if filters.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filters.Offset)
		}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list incidents: %w", err)
	}
	defer rows.Close()

	var incidents []*incident.Incident
	for rows.Next() {
		var inc incident.Incident
		var startedAt, completedAt sql.NullTime
		var exitCode sql.NullInt64
		var failureReason sql.NullString
		var resourceAPIVersion, resourceKind, resourceName, resourceNamespace, resourceUID sql.NullString

		err := rows.Scan(
			&inc.IncidentID,
			&inc.FaultID,
			&inc.TriggeringEventID,
			&inc.Status,
			&inc.CreatedAt,
			&startedAt,
			&completedAt,
			&exitCode,
			&failureReason,
			&inc.Cluster,
			&inc.Namespace,
			&inc.FaultType,
			&inc.Severity,
			&inc.Context,
			&inc.Timestamp,
			&resourceAPIVersion,
			&resourceKind,
			&resourceName,
			&resourceNamespace,
			&resourceUID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan incident row: %w", err)
		}

		// Handle nullable fields
		if startedAt.Valid {
			inc.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			inc.CompletedAt = &completedAt.Time
		}
		if exitCode.Valid {
			exitCodeInt := int(exitCode.Int64)
			inc.ExitCode = &exitCodeInt
		}
		if failureReason.Valid {
			inc.FailureReason = failureReason.String
		}

		// Reconstruct resource info if any fields are present
		if resourceKind.Valid || resourceName.Valid {
			inc.Resource = &incident.ResourceInfo{}
			if resourceAPIVersion.Valid {
				inc.Resource.APIVersion = resourceAPIVersion.String
			}
			if resourceKind.Valid {
				inc.Resource.Kind = resourceKind.String
			}
			if resourceName.Valid {
				inc.Resource.Name = resourceName.String
			}
			if resourceNamespace.Valid {
				inc.Resource.Namespace = resourceNamespace.String
			}
			if resourceUID.Valid {
				inc.Resource.UID = resourceUID.String
			}
		}

		incidents = append(incidents, &inc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating incident rows: %w", err)
	}

	return incidents, nil
}

// Close releases resources held by the store.
// Should be called during application shutdown.
func (s *Store) Close() error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}
	return nil
}

// safeResourceField safely extracts a field from a resource pointer.
// Returns empty string if the resource is nil.
func safeResourceField[T any](resource *T, field func(*T) string) string {
	if resource == nil {
		return ""
	}
	return field(resource)
}
