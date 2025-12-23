// Package postgres provides a PostgreSQL implementation of the StateStore interface.
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/rbias/nightcrier/internal/events"
	"github.com/rbias/nightcrier/internal/incident"
	"github.com/rbias/nightcrier/internal/storage"
)

// Config holds PostgreSQL-specific configuration options.
type Config struct {
	// ConnectionString is the PostgreSQL connection string.
	// Format: postgres://username:password@host:port/database?sslmode=disable
	// Example: postgres://postgres:password@localhost:5432/nightcrier?sslmode=disable
	ConnectionString string

	// MaxOpenConns sets the maximum number of open connections to the database.
	// Default: 25
	MaxOpenConns int

	// MaxIdleConns sets the maximum number of connections in the idle connection pool.
	// Default: 5
	MaxIdleConns int

	// ConnMaxLifetime sets the maximum amount of time a connection may be reused.
	// Default: 5 minutes
	ConnMaxLifetime time.Duration

	// ConnMaxIdleTime sets the maximum amount of time a connection may be idle.
	// Default: 10 minutes
	ConnMaxIdleTime time.Duration
}

// Store implements the StateStore interface using PostgreSQL as the backend.
type Store struct {
	db *sql.DB
}

// New creates a new PostgreSQL StateStore with the provided configuration.
// It establishes a connection pool and validates connectivity.
//
// Example usage:
//
//	cfg := &postgres.Config{
//	    ConnectionString: "postgres://postgres:password@localhost:5432/nightcrier?sslmode=disable",
//	    MaxOpenConns: 25,
//	    MaxIdleConns: 5,
//	}
//	store, err := postgres.New(ctx, cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer store.Close()
func New(ctx context.Context, cfg *Config) (*Store, error) {
	if cfg.ConnectionString == "" {
		return nil, fmt.Errorf("connection string is required")
	}

	// Set defaults
	if cfg.MaxOpenConns == 0 {
		cfg.MaxOpenConns = 25
	}
	if cfg.MaxIdleConns == 0 {
		cfg.MaxIdleConns = 5
	}
	if cfg.ConnMaxLifetime == 0 {
		cfg.ConnMaxLifetime = 5 * time.Minute
	}
	if cfg.ConnMaxIdleTime == 0 {
		cfg.ConnMaxIdleTime = 10 * time.Minute
	}

	// Open database connection
	db, err := sql.Open("postgres", cfg.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Verify connectivity
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Store{db: db}, nil
}

// CreateIncident creates a new incident from a fault event.
// This method creates records in both the fault_events and incidents tables
// within a transaction to ensure consistency.
func (s *Store) CreateIncident(ctx context.Context, inc *incident.Incident, event *events.FaultEvent) error {
	// Start a transaction for atomicity
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert fault_event first
	// Use incident fields (inc) for consistency - these have been processed and enriched
	_, err = tx.ExecContext(ctx, `
		INSERT INTO fault_events (
			fault_id, subscription_id, cluster, received_at,
			resource_api_version, resource_kind, resource_name, resource_namespace, resource_uid,
			fault_type, severity, context, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (fault_id) DO NOTHING`,
		event.FaultID,
		event.SubscriptionID,
		inc.Cluster,
		event.ReceivedAt,
		nullString(inc.Resource, func(r *incident.ResourceInfo) string { return r.APIVersion }),
		nullString(inc.Resource, func(r *incident.ResourceInfo) string { return r.Kind }),
		nullString(inc.Resource, func(r *incident.ResourceInfo) string { return r.Name }),
		nullString(inc.Resource, func(r *incident.ResourceInfo) string { return r.Namespace }),
		nullString(inc.Resource, func(r *incident.ResourceInfo) string { return r.UID }),
		inc.FaultType,
		inc.Severity,
		inc.Context,
		inc.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to insert fault_event: %w", err)
	}

	// Insert incident
	_, err = tx.ExecContext(ctx, `
		INSERT INTO incidents (
			incident_id, fault_id, triggering_event_id,
			status, created_at, started_at, completed_at,
			exit_code, failure_reason,
			cluster, namespace, fault_type, severity, context, timestamp,
			resource_api_version, resource_kind, resource_name, resource_namespace, resource_uid
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)`,
		inc.IncidentID,
		inc.FaultID,
		nullStringValue(inc.TriggeringEventID),
		inc.Status,
		inc.CreatedAt,
		inc.StartedAt,
		inc.CompletedAt,
		inc.ExitCode,
		nullStringValue(inc.FailureReason),
		inc.Cluster,
		nullStringValue(inc.Namespace),
		inc.FaultType,
		inc.Severity,
		inc.Context,
		inc.Timestamp,
		nullString(inc.Resource, func(r *incident.ResourceInfo) string { return r.APIVersion }),
		nullString(inc.Resource, func(r *incident.ResourceInfo) string { return r.Kind }),
		nullString(inc.Resource, func(r *incident.ResourceInfo) string { return r.Name }),
		nullString(inc.Resource, func(r *incident.ResourceInfo) string { return r.Namespace }),
		nullString(inc.Resource, func(r *incident.ResourceInfo) string { return r.UID }),
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
// The startedAt timestamp is set when transitioning to investigating status.
func (s *Store) UpdateIncidentStatus(ctx context.Context, incidentID string, status string, startedAt *time.Time) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE incidents
		SET status = $1, started_at = $2
		WHERE incident_id = $3`,
		status,
		startedAt,
		incidentID,
	)
	if err != nil {
		return fmt.Errorf("failed to update incident status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("incident not found: %s", incidentID)
	}

	return nil
}

// CompleteIncident marks an incident as complete with final result information.
// This updates the incident record with completion time, exit code, and failure reason.
func (s *Store) CompleteIncident(ctx context.Context, incidentID string, exitCode int, failureReason string) error {
	now := time.Now()

	// Determine status based on exit code and failure reason
	status := incident.StatusResolved
	if failureReason != "" || exitCode != 0 {
		status = incident.StatusFailed
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE incidents
		SET status = $1, completed_at = $2, exit_code = $3, failure_reason = $4
		WHERE incident_id = $5`,
		status,
		now,
		exitCode,
		nullStringValue(failureReason),
		incidentID,
	)
	if err != nil {
		return fmt.Errorf("failed to complete incident: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("incident not found: %s", incidentID)
	}

	return nil
}

// RecordAgentExecution records details of an agent execution attempt.
// The log_paths field is stored as JSON in the database.
func (s *Store) RecordAgentExecution(ctx context.Context, exec *storage.AgentExecution) error {
	// Marshal log_paths to JSON
	var logPathsJSON sql.NullString
	if len(exec.LogPaths) > 0 {
		data, err := json.Marshal(exec.LogPaths)
		if err != nil {
			return fmt.Errorf("failed to marshal log_paths: %w", err)
		}
		logPathsJSON = sql.NullString{String: string(data), Valid: true}
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO agent_executions (
			execution_id, incident_id, started_at, completed_at,
			exit_code, error_message, log_paths
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (execution_id) DO UPDATE SET
			completed_at = EXCLUDED.completed_at,
			exit_code = EXCLUDED.exit_code,
			error_message = EXCLUDED.error_message,
			log_paths = EXCLUDED.log_paths`,
		exec.ExecutionID,
		exec.IncidentID,
		exec.StartedAt,
		exec.CompletedAt,
		exec.ExitCode,
		nullStringValue(exec.ErrorMessage),
		logPathsJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to insert/update agent_execution: %w", err)
	}

	return nil
}

// RecordTriageReport stores the investigation report generated by the agent.
func (s *Store) RecordTriageReport(ctx context.Context, report *storage.TriageReport) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO triage_reports (
			report_id, incident_id, execution_id, generated_at,
			report_markdown, report_html
		) VALUES ($1, $2, $3, $4, $5, $6)`,
		report.ReportID,
		report.IncidentID,
		report.ExecutionID,
		report.GeneratedAt,
		report.ReportMarkdown,
		nullStringValue(report.ReportHTML),
	)
	if err != nil {
		return fmt.Errorf("failed to insert triage_report: %w", err)
	}

	return nil
}

// GetIncident retrieves an incident by its ID.
func (s *Store) GetIncident(ctx context.Context, incidentID string) (*incident.Incident, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT
			incident_id, fault_id, triggering_event_id,
			status, created_at, started_at, completed_at,
			exit_code, failure_reason,
			cluster, namespace, fault_type, severity, context, timestamp,
			resource_api_version, resource_kind, resource_name, resource_namespace, resource_uid
		FROM incidents
		WHERE incident_id = $1`,
		incidentID,
	)

	inc := &incident.Incident{}
	var triggeringEventID, failureReason, namespace sql.NullString
	var startedAt, completedAt sql.NullTime
	var exitCode sql.NullInt64
	var resourceAPIVersion, resourceKind, resourceName, resourceNamespace, resourceUID sql.NullString

	err := row.Scan(
		&inc.IncidentID,
		&inc.FaultID,
		&triggeringEventID,
		&inc.Status,
		&inc.CreatedAt,
		&startedAt,
		&completedAt,
		&exitCode,
		&failureReason,
		&inc.Cluster,
		&namespace,
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
		return nil, fmt.Errorf("incident not found: %s", incidentID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query incident: %w", err)
	}

	// Convert nullable fields
	if triggeringEventID.Valid {
		inc.TriggeringEventID = triggeringEventID.String
	}
	if failureReason.Valid {
		inc.FailureReason = failureReason.String
	}
	if namespace.Valid {
		inc.Namespace = namespace.String
	}
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

	// Reconstruct resource info
	if resourceKind.Valid && resourceName.Valid {
		inc.Resource = &incident.ResourceInfo{
			APIVersion: resourceAPIVersion.String,
			Kind:       resourceKind.String,
			Name:       resourceName.String,
			Namespace:  resourceNamespace.String,
			UID:        resourceUID.String,
		}
	}

	return inc, nil
}

// ListIncidents returns incidents matching the provided filters.
func (s *Store) ListIncidents(ctx context.Context, filters *storage.IncidentFilters) ([]*incident.Incident, error) {
	if filters == nil {
		filters = &storage.IncidentFilters{}
	}

	// Build query dynamically based on filters
	query := `
		SELECT
			incident_id, fault_id, triggering_event_id,
			status, created_at, started_at, completed_at,
			exit_code, failure_reason,
			cluster, namespace, fault_type, severity, context, timestamp,
			resource_api_version, resource_kind, resource_name, resource_namespace, resource_uid
		FROM incidents
		WHERE 1=1`

	args := []interface{}{}
	argIndex := 1

	// Apply filters
	if len(filters.Status) > 0 {
		query += fmt.Sprintf(" AND status = ANY($%d)", argIndex)
		args = append(args, pq.Array(filters.Status))
		argIndex++
	}
	if filters.Cluster != "" {
		query += fmt.Sprintf(" AND cluster = $%d", argIndex)
		args = append(args, filters.Cluster)
		argIndex++
	}
	if filters.Namespace != "" {
		query += fmt.Sprintf(" AND namespace = $%d", argIndex)
		args = append(args, filters.Namespace)
		argIndex++
	}
	if filters.FaultType != "" {
		query += fmt.Sprintf(" AND fault_type = $%d", argIndex)
		args = append(args, filters.FaultType)
		argIndex++
	}
	if filters.Severity != "" {
		query += fmt.Sprintf(" AND severity = $%d", argIndex)
		args = append(args, filters.Severity)
		argIndex++
	}
	if filters.CreatedAfter != nil {
		query += fmt.Sprintf(" AND created_at > $%d", argIndex)
		args = append(args, filters.CreatedAfter)
		argIndex++
	}
	if filters.CreatedBefore != nil {
		query += fmt.Sprintf(" AND created_at < $%d", argIndex)
		args = append(args, filters.CreatedBefore)
		argIndex++
	}

	// Add ordering and pagination
	query += " ORDER BY created_at DESC"
	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filters.Limit)
		argIndex++
	}
	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filters.Offset)
		argIndex++
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query incidents: %w", err)
	}
	defer rows.Close()

	incidents := []*incident.Incident{}
	for rows.Next() {
		inc := &incident.Incident{}
		var triggeringEventID, failureReason, namespace sql.NullString
		var startedAt, completedAt sql.NullTime
		var exitCode sql.NullInt64
		var resourceAPIVersion, resourceKind, resourceName, resourceNamespace, resourceUID sql.NullString

		err := rows.Scan(
			&inc.IncidentID,
			&inc.FaultID,
			&triggeringEventID,
			&inc.Status,
			&inc.CreatedAt,
			&startedAt,
			&completedAt,
			&exitCode,
			&failureReason,
			&inc.Cluster,
			&namespace,
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
			return nil, fmt.Errorf("failed to scan incident: %w", err)
		}

		// Convert nullable fields
		if triggeringEventID.Valid {
			inc.TriggeringEventID = triggeringEventID.String
		}
		if failureReason.Valid {
			inc.FailureReason = failureReason.String
		}
		if namespace.Valid {
			inc.Namespace = namespace.String
		}
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

		// Reconstruct resource info
		if resourceKind.Valid && resourceName.Valid {
			inc.Resource = &incident.ResourceInfo{
				APIVersion: resourceAPIVersion.String,
				Kind:       resourceKind.String,
				Name:       resourceName.String,
				Namespace:  resourceNamespace.String,
				UID:        resourceUID.String,
			}
		}

		incidents = append(incidents, inc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating incidents: %w", err)
	}

	return incidents, nil
}

// Close releases any resources held by the StateStore.
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Health performs a health check on the database connection.
// Returns nil if the connection is healthy, an error otherwise.
func (s *Store) Health(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Helper functions

// nullString safely extracts a string value from a pointer, returning sql.NullString.
func nullString[T any](ptr *T, getter func(*T) string) sql.NullString {
	if ptr == nil {
		return sql.NullString{Valid: false}
	}
	val := getter(ptr)
	if val == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: val, Valid: true}
}

// nullStringValue converts a string to sql.NullString, treating empty strings as NULL.
func nullStringValue(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
