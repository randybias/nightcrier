# Proposal: Add Panic Capture

## Problem Statement

When nightcrier panics, the process exits and Kubernetes restarts it. However:
1. The panic stack trace is only visible in container logs (which may rotate)
2. There's no persistent record of what caused the crash
3. Analyzing panics requires manual log retrieval and interpretation

## Proposed Solution

Capture panic information to the database before the process exits. This provides:
- Persistent record of all panics across restarts
- Structured data (timestamp, error, stack trace, component)
- Foundation for future automated analysis

## Scope

1. Add `panic_reports` table to store panic information
2. Add `recovery` package with panic capture utilities
3. Wrap `main()` with deferred panic recovery that saves to database
4. Optionally wrap critical goroutines with recoverable panic handling

## Database Schema

```sql
CREATE TABLE IF NOT EXISTS panic_reports (
    id TEXT PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL,
    error TEXT NOT NULL,
    stack TEXT NOT NULL,
    component TEXT NOT NULL,        -- "main", "dispatcher", "mcp-client", etc.
    instance_id TEXT,               -- Which nightcrier instance
    analyzed BOOLEAN DEFAULT FALSE, -- For future analysis tracking
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Implementation Approach

### Main Function Recovery

```go
func main() {
    // Initialize minimal logging and DB connection first
    db := initDatabase()

    defer func() {
        if r := recover(); r != nil {
            report := PanicReport{
                ID:        uuid.New().String(),
                Timestamp: time.Now(),
                Error:     fmt.Sprintf("%v", r),
                Stack:     string(debug.Stack()),
                Component: "main",
            }
            // Best-effort save - may fail if DB caused the panic
            db.SavePanicReport(context.Background(), report)
            // Re-panic to trigger K8s restart
            panic(r)
        }
    }()

    run()
}
```

### Goroutine Recovery (Optional)

For non-fatal panics in goroutines, allow recovery without full process exit:

```go
recovery.SafeGo("mcp-client-westeu", func() {
    client.Subscribe(ctx)
}, func(report PanicReport) {
    stateStore.SavePanicReport(ctx, report)
    // Goroutine died but process continues
})
```

## Out of Scope

- Automated analysis of panics (future enhancement)
- Alerting/notifications on panic (use existing monitoring)
- Panic prevention (addressed case-by-case)

## Requirements Changed

- **state-persistence**: Add panic report storage requirement

## Success Criteria

1. Panics are captured to database before process exits
2. Panic reports include full stack trace and timestamp
3. Reports persist across restarts
4. No performance impact during normal operation
