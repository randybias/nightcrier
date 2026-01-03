-- Rollback migration 000004: Restore original column names in incidents table

ALTER TABLE incidents
  RENAME COLUMN job_started_at TO started_at;

ALTER TABLE incidents
  RENAME COLUMN job_completed_at TO completed_at;
