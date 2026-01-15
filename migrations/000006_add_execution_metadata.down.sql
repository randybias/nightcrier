-- Rollback execution metadata columns from agent_executions table

-- Drop indexes first
DROP INDEX IF EXISTS idx_agent_executions_cluster_name;
DROP INDEX IF EXISTS idx_agent_executions_agent_model;
DROP INDEX IF EXISTS idx_agent_executions_agent_cli;

-- Drop columns
ALTER TABLE agent_executions DROP COLUMN cluster_name;
ALTER TABLE agent_executions DROP COLUMN agent_model;
ALTER TABLE agent_executions DROP COLUMN agent_cli;
