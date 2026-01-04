// Package adminui provides a minimal developer dashboard for viewing triage status.
package adminui

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Store provides database access for the admin UI.
type Store struct {
	db *sql.DB
}

// NewStore creates a new Store with the given database connection.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// RunningTriage represents an active agent execution.
type RunningTriage struct {
	ExecutionID              string
	IncidentID               string
	Cluster                  string
	Namespace                string
	FaultType                string
	JobStartedAt             time.Time
	RunStartedAt             *time.Time
	RunCompletedAt           *time.Time
	CurrentActivity          string
	CurrentActivityStartedAt *time.Time
}

// RunState returns the derived run state based on timestamps.
func (r *RunningTriage) RunState() string {
	if r.RunStartedAt == nil {
		return "queued"
	}
	if r.RunCompletedAt == nil {
		return "running"
	}
	return "finishing"
}

// Age returns the duration since job started.
func (r *RunningTriage) Age() time.Duration {
	return time.Now().UTC().Sub(r.JobStartedAt.UTC())
}

// Incident represents an incident for display.
type Incident struct {
	IncidentID     string
	FaultID        string
	Cluster        string
	Namespace      string
	FaultType      string
	Severity       string
	Status         string
	CreatedAt      time.Time
	JobStartedAt   *time.Time
	JobCompletedAt *time.Time
	ExitCode       *int
	FailureReason  string

	// Execution info (from latest execution)
	HasExecution     bool
	ExecutionRunning bool
	RunExitCode      *int
	ErrorMessage     string
}

// TriageIndicator returns the color for the triage indicator.
func (i *Incident) TriageIndicator() string {
	if !i.HasExecution {
		return "gray"
	}
	if i.ExecutionRunning {
		return "blue"
	}
	// Check for failure conditions
	if i.RunExitCode != nil && *i.RunExitCode != 0 {
		return "red"
	}
	if i.ErrorMessage != "" {
		return "red"
	}
	if i.Status == "failed" || i.Status == "agent_failed" {
		return "red"
	}
	// Success
	if i.RunExitCode != nil && *i.RunExitCode == 0 {
		return "green"
	}
	// Still running or unknown
	return "blue"
}

// GetRunningTriages returns all active agent executions.
func (s *Store) GetRunningTriages(ctx context.Context) ([]RunningTriage, error) {
	query := `
		SELECT
			ae.execution_id,
			ae.incident_id,
			i.cluster,
			COALESCE(i.namespace, '') as namespace,
			i.fault_type,
			ae.job_started_at,
			ae.run_started_at,
			ae.run_completed_at,
			COALESCE(ae.current_activity, '') as current_activity,
			ae.current_activity_started_at
		FROM agent_executions ae
		JOIN incidents i ON ae.incident_id = i.incident_id
		WHERE ae.job_completed_at IS NULL
		ORDER BY ae.job_started_at DESC
		LIMIT 50
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var triages []RunningTriage
	for rows.Next() {
		var t RunningTriage
		err := rows.Scan(
			&t.ExecutionID,
			&t.IncidentID,
			&t.Cluster,
			&t.Namespace,
			&t.FaultType,
			&t.JobStartedAt,
			&t.RunStartedAt,
			&t.RunCompletedAt,
			&t.CurrentActivity,
			&t.CurrentActivityStartedAt,
		)
		if err != nil {
			return nil, err
		}
		triages = append(triages, t)
	}

	return triages, rows.Err()
}

// GetIncidents returns all incidents ordered by creation time.
func (s *Store) GetIncidents(ctx context.Context, limit int) ([]Incident, error) {
	if limit <= 0 {
		limit = 100
	}

	// Main query for incidents
	query := `
		SELECT
			i.incident_id,
			i.fault_id,
			i.cluster,
			COALESCE(i.namespace, '') as namespace,
			i.fault_type,
			i.severity,
			i.status,
			i.created_at,
			i.job_started_at,
			i.job_completed_at,
			i.exit_code,
			COALESCE(i.failure_reason, '') as failure_reason
		FROM incidents i
		ORDER BY i.created_at DESC
		LIMIT $1
	`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidents []Incident
	for rows.Next() {
		var inc Incident
		err := rows.Scan(
			&inc.IncidentID,
			&inc.FaultID,
			&inc.Cluster,
			&inc.Namespace,
			&inc.FaultType,
			&inc.Severity,
			&inc.Status,
			&inc.CreatedAt,
			&inc.JobStartedAt,
			&inc.JobCompletedAt,
			&inc.ExitCode,
			&inc.FailureReason,
		)
		if err != nil {
			return nil, err
		}
		incidents = append(incidents, inc)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Fetch latest execution for each incident
	for i := range incidents {
		exec, err := s.getLatestExecution(ctx, incidents[i].IncidentID)
		if err != nil {
			continue // Skip on error, leave HasExecution as false
		}
		if exec != nil {
			incidents[i].HasExecution = true
			incidents[i].ExecutionRunning = exec.JobCompletedAt == nil
			incidents[i].RunExitCode = exec.RunExitCode
			incidents[i].ErrorMessage = exec.ErrorMessage
		}
	}

	return incidents, nil
}

// executionInfo holds execution details for an incident.
type executionInfo struct {
	JobCompletedAt *time.Time
	RunExitCode    *int
	ErrorMessage   string
}

// getLatestExecution returns the most recent execution for an incident.
func (s *Store) getLatestExecution(ctx context.Context, incidentID string) (*executionInfo, error) {
	query := `
		SELECT
			job_completed_at,
			run_exit_code,
			COALESCE(error_message, '') as error_message
		FROM agent_executions
		WHERE incident_id = $1
		ORDER BY job_started_at DESC
		LIMIT 1
	`

	var exec executionInfo
	err := s.db.QueryRowContext(ctx, query, incidentID).Scan(
		&exec.JobCompletedAt,
		&exec.RunExitCode,
		&exec.ErrorMessage,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &exec, nil
}

// DeleteIncident removes an incident and all associated data from the database.
// This deletes: triage_reports, agent_executions, and the incident itself.
// Faults are NOT deleted (they may be shared across incidents).
func (s *Store) DeleteIncident(ctx context.Context, incidentID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete triage_reports first (depends on both incidents and agent_executions)
	_, err = tx.ExecContext(ctx, `DELETE FROM triage_reports WHERE incident_id = $1`, incidentID)
	if err != nil {
		return fmt.Errorf("failed to delete triage reports: %w", err)
	}

	// Delete agent_executions (depends on incidents)
	_, err = tx.ExecContext(ctx, `DELETE FROM agent_executions WHERE incident_id = $1`, incidentID)
	if err != nil {
		return fmt.Errorf("failed to delete agent executions: %w", err)
	}

	// Delete the incident
	result, err := tx.ExecContext(ctx, `DELETE FROM incidents WHERE incident_id = $1`, incidentID)
	if err != nil {
		return fmt.Errorf("failed to delete incident: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("incident not found: %s", incidentID)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetRunningTriageByExecutionID returns a running triage by its execution ID.
func (s *Store) GetRunningTriageByExecutionID(ctx context.Context, executionID string) (*RunningTriage, error) {
	query := `
		SELECT
			ae.execution_id,
			ae.incident_id,
			i.cluster,
			COALESCE(i.namespace, '') as namespace,
			i.fault_type,
			ae.job_started_at,
			ae.run_started_at,
			ae.run_completed_at,
			COALESCE(ae.current_activity, '') as current_activity,
			ae.current_activity_started_at
		FROM agent_executions ae
		JOIN incidents i ON ae.incident_id = i.incident_id
		WHERE ae.execution_id = $1 AND ae.job_completed_at IS NULL
	`

	var t RunningTriage
	err := s.db.QueryRowContext(ctx, query, executionID).Scan(
		&t.ExecutionID,
		&t.IncidentID,
		&t.Cluster,
		&t.Namespace,
		&t.FaultType,
		&t.JobStartedAt,
		&t.RunStartedAt,
		&t.RunCompletedAt,
		&t.CurrentActivity,
		&t.CurrentActivityStartedAt,
	)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

// CancelExecution marks an execution as cancelled by setting job_completed_at and error_message.
func (s *Store) CancelExecution(ctx context.Context, executionID string) error {
	now := time.Now().UTC()
	result, err := s.db.ExecContext(ctx, `
		UPDATE agent_executions
		SET job_completed_at = $1, error_message = 'cancelled by user'
		WHERE execution_id = $2 AND job_completed_at IS NULL
	`, now, executionID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("execution not found or already completed: %s", executionID)
	}

	return nil
}
