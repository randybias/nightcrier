-- Rollback migration 000003: Remove run lifecycle tracking

-- Drop indexes for run lifecycle
DROP INDEX IF EXISTS idx_agent_executions_run_started_at;
DROP INDEX IF EXISTS idx_agent_executions_run_completed_at;

-- Remove run lifecycle columns
ALTER TABLE agent_executions
  DROP COLUMN IF EXISTS run_started_at,
  DROP COLUMN IF EXISTS run_completed_at,
  DROP COLUMN IF EXISTS run_exit_code;

-- Restore original column names
ALTER TABLE agent_executions
  RENAME COLUMN job_started_at TO started_at;

ALTER TABLE agent_executions
  RENAME COLUMN job_completed_at TO completed_at;

-- Restore original index name
DROP INDEX IF EXISTS idx_agent_executions_job_started_at;
CREATE INDEX IF NOT EXISTS idx_agent_executions_started_at
  ON agent_executions(started_at);
