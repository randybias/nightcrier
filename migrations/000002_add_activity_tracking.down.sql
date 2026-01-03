-- Rollback activity tracking columns from agent_executions table
-- Removes columns in reverse order of addition

ALTER TABLE agent_executions DROP COLUMN IF EXISTS last_activity_finished_at;
ALTER TABLE agent_executions DROP COLUMN IF EXISTS last_activity;
ALTER TABLE agent_executions DROP COLUMN IF EXISTS current_activity_started_at;
ALTER TABLE agent_executions DROP COLUMN IF EXISTS current_activity;
