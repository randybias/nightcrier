# Change: Add Developer Admin UI

## Why

During development and operations, there's no quick way to see the current state of triage runs and incidents without querying the database directly. A minimal admin console would provide immediate visibility into:
- Which triage runs are currently active
- The incident queue and their status
- Quick access to investigation artifacts

## What Changes

- Add a new `--admin-listen` flag to the nightcrier binary
- Create a single-page HTML dashboard served via Go's `html/template`
- Dashboard displays two panes:
  - **Top pane**: Running triages (active agent executions)
  - **Bottom pane**: All incidents with status indicators
- Each incident has a "View" button linking to the object storage index.html
- Auto-detect database backend from existing config (supports both SQLite and PostgreSQL)
- No authentication (localhost binding by default)
- No database schema modifications required

## Impact

- Affected specs: None (new capability)
- Affected code:
  - `cmd/nightcrier/main.go` - Add flag and server startup
  - `internal/adminui/` - New package (handlers, store, templates)
