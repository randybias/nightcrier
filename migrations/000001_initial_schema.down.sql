-- Rollback initial schema
-- Drop tables in reverse order of dependencies

DROP INDEX IF EXISTS idx_triage_reports_generated_at;
DROP INDEX IF EXISTS idx_triage_reports_execution_id;
DROP INDEX IF EXISTS idx_triage_reports_incident_id;
DROP TABLE IF EXISTS triage_reports;

DROP INDEX IF EXISTS idx_agent_executions_started_at;
DROP INDEX IF EXISTS idx_agent_executions_incident_id;
DROP TABLE IF EXISTS agent_executions;

DROP INDEX IF EXISTS idx_incidents_severity;
DROP INDEX IF EXISTS idx_incidents_fault_type;
DROP INDEX IF EXISTS idx_incidents_namespace;
DROP INDEX IF EXISTS idx_incidents_created_at;
DROP INDEX IF EXISTS idx_incidents_cluster;
DROP INDEX IF EXISTS idx_incidents_status;
DROP INDEX IF EXISTS idx_incidents_fault_id;
DROP TABLE IF EXISTS incidents;

DROP INDEX IF EXISTS idx_fault_events_severity;
DROP INDEX IF EXISTS idx_fault_events_fault_type;
DROP INDEX IF EXISTS idx_fault_events_received_at;
DROP INDEX IF EXISTS idx_fault_events_cluster;
DROP TABLE IF EXISTS fault_events;
