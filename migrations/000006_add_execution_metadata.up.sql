-- Add execution metadata columns to agent_executions table
-- These columns track which agent CLI, model, and cluster were used for each execution

-- Add agent_cli column (claude, codex, gemini, goose)
ALTER TABLE agent_executions ADD COLUMN agent_cli TEXT NOT NULL DEFAULT 'unknown';

-- Add agent_model column (sonnet, opus, haiku, etc.)
ALTER TABLE agent_executions ADD COLUMN agent_model TEXT NOT NULL DEFAULT 'unknown';

-- Add cluster_name column (name of the cluster being investigated)
ALTER TABLE agent_executions ADD COLUMN cluster_name TEXT NOT NULL DEFAULT 'unknown';

-- Create indexes for filtering and reporting
CREATE INDEX IF NOT EXISTS idx_agent_executions_agent_cli ON agent_executions(agent_cli);
CREATE INDEX IF NOT EXISTS idx_agent_executions_agent_model ON agent_executions(agent_model);
CREATE INDEX IF NOT EXISTS idx_agent_executions_cluster_name ON agent_executions(cluster_name);
