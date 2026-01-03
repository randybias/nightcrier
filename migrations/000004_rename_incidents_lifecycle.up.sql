-- Migration 000004: Rename lifecycle columns in incidents table
-- Matches the agent_executions table naming from migration 000003

ALTER TABLE incidents
  RENAME COLUMN started_at TO job_started_at;

ALTER TABLE incidents
  RENAME COLUMN completed_at TO job_completed_at;
