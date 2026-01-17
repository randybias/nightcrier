# Change: Add Execution Metadata to Agent Executions Table

## Why

The `agent_executions` table currently lacks critical metadata needed for observability and analysis. When reviewing past investigations, operators need to know:
- Which agent CLI was used (claude, codex, gemini, goose)
- Which LLM model was used (sonnet, opus, haiku for Claude; various models for other providers)
- Which cluster was being investigated

This metadata is essential for:
1. **Cost analysis** - Different models have different costs
2. **Performance comparison** - Comparing agent effectiveness across models
3. **Debugging** - Correlating agent behavior with model/CLI choice
4. **Audit trail** - Complete record of what ran where

## What Changes

- Add `agent_cli` column to `agent_executions` table (e.g., "claude", "codex", "gemini", "goose")
- Add `agent_model` column to `agent_executions` table (e.g., "sonnet", "opus", "haiku")
- Add `cluster_name` column to `agent_executions` table (cluster being investigated)
- Update `AgentExecution` struct to include new fields
- Update `RecordAgentExecution` callers to populate new fields
- Add database migration for schema change

## Impact

- **Affected Specs:**
  - `state-persistence` (Add metadata columns to execution records)
- **Affected Code:**
  - `migrations/` (New migration file)
  - `internal/storage/statestore.go` (Update `AgentExecution` struct)
  - `internal/storage/sqlite/sqlite.go` (Update SQL queries)
  - `internal/storage/postgres/postgres.go` (Update SQL queries)
  - Agent execution callers (Pass metadata when recording)
