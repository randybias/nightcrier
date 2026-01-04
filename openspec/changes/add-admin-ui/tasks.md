# Tasks: Add Developer Admin UI

## 1. Core Implementation

- [ ] 1.1 Create `internal/adminui/` package structure
- [ ] 1.2 Create `store.go` with database query helpers (support both SQLite and PostgreSQL)
- [ ] 1.3 Create `handlers.go` with HTTP handlers for GET /admin
- [ ] 1.4 Create `templates/admin.html` with split-pane layout

## 2. Database Queries

- [ ] 2.1 Implement `GetRunningTriages()` - active agent executions (job_completed_at IS NULL)
- [ ] 2.2 Implement `GetIncidents()` - all incidents ordered by created_at DESC
- [ ] 2.3 Implement `GetLatestExecutionForIncident()` - for triage indicator

## 3. UI Components

- [ ] 3.1 Top pane: Running triages table (cluster, incident_id, run state, age)
- [ ] 3.2 Bottom pane: Incidents table (created, cluster, severity, status, triage indicator)
- [ ] 3.3 View button linking to object storage index.html
- [ ] 3.4 Auto-refresh via meta refresh tag (simple approach)

## 4. Integration

- [ ] 4.1 Add `--admin-listen` flag to cmd/nightcrier/main.go
- [ ] 4.2 Start HTTP server when flag is provided
- [ ] 4.3 Pass database connection to adminui handlers

## 5. Testing

- [ ] 5.1 Manual testing with PostgreSQL backend
- [ ] 5.2 Manual testing with SQLite backend
- [ ] 5.3 Verify view links work with object storage
