# Tasks: Add Panic Capture

## Implementation Order

### Phase 1: Database Schema

- [ ] Create migration `000006_add_panic_reports.up.sql`
- [ ] Create corresponding down migration
- [ ] Add `PanicReport` struct to `internal/storage/types.go`

### Phase 2: StateStore Interface

- [ ] Add `SavePanicReport(ctx, report)` to StateStore interface
- [ ] Add `GetUnanalyzedPanicReports(ctx)` to StateStore interface
- [ ] Add `MarkPanicAnalyzed(ctx, id)` to StateStore interface
- [ ] Implement PostgreSQL methods
- [ ] Implement SQLite methods

### Phase 3: Recovery Package

- [ ] Create `internal/recovery/recovery.go`
- [ ] Add `PanicReport` struct with JSON tags
- [ ] Add `CaptureAndSave(db, component)` function that returns a defer-able function
- [ ] Add `SafeGo(component, fn, onPanic)` for goroutine recovery (optional)

### Phase 4: Integration

- [ ] Wrap `main()` with panic capture defer
- [ ] Ensure database connection is established early enough to capture panics
- [ ] Handle case where DB itself causes the panic (best-effort logging to stderr)

### Phase 5: Testing

- [ ] Add unit tests for panic report storage
- [ ] Add unit tests for query methods
- [ ] Test panic capture with intentional panic

## Notes

- Panic capture must be best-effort - if DB is down, log to stderr
- Instance ID reuses the same ID generated for cluster locks
- The `analyzed` flag is for future use (automated analysis)
