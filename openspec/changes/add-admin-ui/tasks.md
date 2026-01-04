# Tasks: Add Developer Admin UI

## 1. Core Implementation

- [x] 1.1 Create `internal/adminui/` package structure
- [x] 1.2 Create `store.go` with database query helpers (support both SQLite and PostgreSQL)
- [x] 1.3 Create `handlers.go` with HTTP handlers for GET /admin
- [x] 1.4 Create `templates/admin.html` with split-pane layout

## 2. Database Queries

- [x] 2.1 Implement `GetRunningTriages()` - active agent executions (job_completed_at IS NULL)
- [x] 2.2 Implement `GetIncidents()` - all incidents ordered by created_at DESC
- [x] 2.3 Implement `GetLatestExecutionForIncident()` - for triage indicator

## 3. UI Components

- [x] 3.1 Top pane: Running triages table (cluster, incident_id, run state, age)
- [x] 3.2 Bottom pane: Incidents table (created, cluster, severity, status, triage indicator)
- [x] 3.3 View button linking to object storage index.html with signed URLs
- [x] 3.4 Auto-refresh via meta refresh tag (5 second interval)
- [x] 3.5 Copyable incident IDs with click-to-copy
- [x] 3.6 Nightcrier headshot logo in header

## 4. Integration

- [x] 4.1 Add `--admin-listen` flag to cmd/nightcrier/main.go
- [x] 4.2 Start HTTP server when flag is provided
- [x] 4.3 Pass database connection to adminui handlers
- [x] 4.4 Pass ObjectSigner for Azure Blob Storage signed URLs

## 5. Testing

- [x] 5.1 Manual testing with PostgreSQL backend
- [x] 5.2 Verify view links work with signed URLs for Azure Blob Storage
