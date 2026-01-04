-- Migration 000003: Add run lifecycle tracking
-- Separates Job lifecycle (Go) from Run lifecycle (container/entrypoint.sh)
--
-- Timeline:
--   job_started_at    -> Go creates K8s Job
--   run_started_at    -> entrypoint.sh publishes run.started (after preflight)
--   run_completed_at  -> entrypoint.sh publishes run.completed (after agent exits)
--   job_completed_at  -> Go finishes cleanup and file uploads

-- Rename existing columns to reflect Job lifecycle
ALTER TABLE agent_executions
  RENAME COLUMN started_at TO job_started_at;

ALTER TABLE agent_executions
  RENAME COLUMN completed_at TO job_completed_at;

-- Add run lifecycle columns (separate statements for SQLite compatibility)
ALTER TABLE agent_executions ADD COLUMN run_started_at TIMESTAMP;
ALTER TABLE agent_executions ADD COLUMN run_completed_at TIMESTAMP;
ALTER TABLE agent_executions ADD COLUMN run_exit_code INTEGER;

-- Create indexes for run lifecycle queries
CREATE INDEX IF NOT EXISTS idx_agent_executions_run_started_at
  ON agent_executions(run_started_at);

CREATE INDEX IF NOT EXISTS idx_agent_executions_run_completed_at
  ON agent_executions(run_completed_at);

-- Update existing index name for consistency
DROP INDEX IF EXISTS idx_agent_executions_started_at;
CREATE INDEX IF NOT EXISTS idx_agent_executions_job_started_at
  ON agent_executions(job_started_at);
