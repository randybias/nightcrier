-- Add activity tracking columns to agent_executions table
-- These columns track the agent's current and last activity for progress reporting

ALTER TABLE agent_executions ADD COLUMN current_activity TEXT;
ALTER TABLE agent_executions ADD COLUMN current_activity_started_at TIMESTAMP;
ALTER TABLE agent_executions ADD COLUMN last_activity TEXT;
ALTER TABLE agent_executions ADD COLUMN last_activity_finished_at TIMESTAMP;
