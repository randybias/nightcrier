# PostgreSQL StateStore Implementation

This package provides a PostgreSQL implementation of the `StateStore` interface for persisting nightcrier incident state.

## Features

- Full implementation of the `StateStore` interface
- Connection pooling with configurable limits
- Transaction support for atomic operations
- Context-aware operations supporting cancellation and timeouts
- JSON encoding for complex fields (log_paths)
- Comprehensive error handling with detailed context
- Thread-safe concurrent operations
- Health check support

## Configuration

### Connection String Format

The PostgreSQL connection string follows the standard format:

```
postgres://username:password@host:port/database?sslmode=disable
```

### SSL Modes

Supported SSL modes:
- `disable` - No SSL (development only)
- `require` - SSL required but no certificate verification
- `verify-ca` - SSL with CA certificate verification
- `verify-full` - SSL with full certificate verification (recommended for production)

Example with SSL:
```
postgres://username:password@host:port/database?sslmode=verify-full
```

### Connection Pool Configuration

The `Config` struct supports the following connection pool settings:

- `MaxOpenConns` - Maximum number of open connections (default: 25)
- `MaxIdleConns` - Maximum number of idle connections (default: 5)
- `ConnMaxLifetime` - Maximum lifetime of a connection (default: 5 minutes)
- `ConnMaxIdleTime` - Maximum idle time for a connection (default: 10 minutes)

## Usage

### Basic Setup

```go
import (
    "context"
    "github.com/rbias/nightcrier/internal/storage/postgres"
)

func main() {
    ctx := context.Background()

    cfg := &postgres.Config{
        ConnectionString: "postgres://user:pass@localhost:5432/nightcrier?sslmode=disable",
        MaxOpenConns: 25,
        MaxIdleConns: 5,
    }

    store, err := postgres.New(ctx, cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()

    // Use the store...
}
```

### Creating an Incident

```go
event := &events.FaultEvent{
    FaultID: "fault-123",
    Cluster: "production",
    // ... other fields
}

inc := incident.NewFromEvent("incident-123", event)

err := store.CreateIncident(ctx, inc, event)
if err != nil {
    log.Printf("failed to create incident: %v", err)
}
```

### Updating Incident Status

```go
startedAt := time.Now()
err := store.UpdateIncidentStatus(ctx, "incident-123", incident.StatusInvestigating, &startedAt)
if err != nil {
    log.Printf("failed to update status: %v", err)
}
```

### Completing an Incident

```go
exitCode := 0
failureReason := ""
err := store.CompleteIncident(ctx, "incident-123", exitCode, failureReason)
if err != nil {
    log.Printf("failed to complete incident: %v", err)
}
```

### Recording Agent Execution

```go
exec := &storage.AgentExecution{
    ExecutionID: "exec-123",
    IncidentID:  "incident-123",
    StartedAt:   time.Now(),
    LogPaths: map[string]string{
        "stdout": "/path/to/stdout.log",
        "stderr": "/path/to/stderr.log",
    },
}

err := store.RecordAgentExecution(ctx, exec)
if err != nil {
    log.Printf("failed to record execution: %v", err)
}
```

### Recording a Triage Report

```go
report := &storage.TriageReport{
    ReportID:       "report-123",
    IncidentID:     "incident-123",
    ExecutionID:    "exec-123",
    GeneratedAt:    time.Now(),
    ReportMarkdown: "# Investigation Report\n\n...",
}

err := store.RecordTriageReport(ctx, report)
if err != nil {
    log.Printf("failed to record report: %v", err)
}
```

### Querying Incidents

```go
// Get a single incident
inc, err := store.GetIncident(ctx, "incident-123")
if err != nil {
    log.Printf("failed to get incident: %v", err)
}

// List incidents with filters
filters := &storage.IncidentFilters{
    Status:  []string{incident.StatusInvestigating},
    Cluster: "production",
    Limit:   10,
}

incidents, err := store.ListIncidents(ctx, filters)
if err != nil {
    log.Printf("failed to list incidents: %v", err)
}
```

### Health Check

```go
err := store.Health(ctx)
if err != nil {
    log.Printf("database health check failed: %v", err)
}
```

## Testing

### Running Integration Tests

Integration tests require a PostgreSQL database. Set the connection string via environment variable:

```bash
export NIGHTCRIER_TEST_POSTGRES_URL="postgres://postgres:password@localhost:5432/nightcrier_test?sslmode=disable"
go test -v ./internal/storage/postgres/...
```

Tests will be skipped if the environment variable is not set.

### Test Coverage

The test suite includes:

- Connection validation and error handling
- CRUD operations for all StateStore methods
- Concurrent access patterns (thread safety)
- Transaction integrity
- Context cancellation and timeouts
- Edge cases and error conditions
- Performance benchmarks

### Benchmarks

Run benchmarks to measure performance:

```bash
go test -bench=. -benchmem ./internal/storage/postgres/...
```

## Database Schema

The PostgreSQL adapter works with the following schema:

### Tables

1. **fault_events** - Raw fault events from kubernetes-mcp-server
2. **incidents** - Investigation incidents with lifecycle tracking
3. **agent_executions** - Agent execution attempts
4. **triage_reports** - Investigation reports generated by agents

See `migrations/000001_initial_schema.up.sql` for the complete schema definition.

### Indexes

Indexes are created on commonly queried columns:
- `fault_events`: cluster, received_at, fault_type, severity
- `incidents`: fault_id, status, cluster, created_at, namespace, fault_type, severity
- `agent_executions`: incident_id, started_at
- `triage_reports`: incident_id, execution_id, generated_at

## Error Handling

All methods return detailed errors with context:

```go
err := store.CreateIncident(ctx, inc, event)
if err != nil {
    // Errors are wrapped with context
    // Example: "failed to insert incident: <underlying error>"
}
```

Common error scenarios:
- Connection failures
- Context cancellation/timeout
- Constraint violations (foreign keys, unique constraints)
- Not found errors (GetIncident, UpdateIncidentStatus, CompleteIncident)

## Best Practices

### Connection Pooling

For production deployments:
- Set `MaxOpenConns` based on expected load (typically 25-100)
- Set `MaxIdleConns` to 10-20% of `MaxOpenConns`
- Configure `ConnMaxLifetime` to handle connection resets (5-10 minutes)

### Context Usage

Always pass context with appropriate timeouts:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

err := store.CreateIncident(ctx, inc, event)
```

### Transaction Safety

`CreateIncident` uses transactions internally to ensure atomic writes to both `fault_events` and `incidents` tables. Other operations that modify single tables use single statements.

### Graceful Shutdown

Always close the store during application shutdown:

```go
defer store.Close()
```

## Performance Considerations

### Connection Pool Tuning

Monitor these metrics:
- Connection wait time
- Active connections
- Idle connections

Adjust pool settings based on application load.

### Query Optimization

The adapter uses:
- Prepared statements via `QueryContext`/`ExecContext`
- Indexes on all filter columns
- Efficient pagination with LIMIT/OFFSET

### JSONB Fields

The `log_paths` field in `agent_executions` is stored as TEXT with JSON encoding. For large deployments, consider using PostgreSQL's native JSONB type and update the schema migration.

## Troubleshooting

### Connection Refused

```
failed to ping database: dial tcp: connect: connection refused
```

Check that:
- PostgreSQL is running
- Host and port are correct
- Firewall allows connections

### Authentication Failed

```
failed to open database: pq: password authentication failed
```

Verify:
- Username and password are correct
- User has proper database permissions

### SSL Required

```
pq: SSL is not enabled on the server
```

Either enable SSL on PostgreSQL or use `sslmode=disable` (not recommended for production).

### Foreign Key Constraint Violation

```
failed to insert agent_execution: pq: insert or update on table "agent_executions" violates foreign key constraint
```

Ensure the referenced incident exists before creating related records.

## Migration

### Initial Setup

Run migrations to create the schema:

```go
import "github.com/rbias/nightcrier/internal/storage"

cfg := &storage.MigrationConfig{
    MigrationsPath: "./migrations",
    DatabaseType:   "postgres",
    DatabaseURL:    "postgres://...",
}

err := storage.RunMigrations(cfg)
if err != nil {
    log.Fatal(err)
}
```

### Schema Updates

Future schema changes should be added as new migration files following the naming convention:
- `000002_add_feature.up.sql`
- `000002_add_feature.down.sql`

## Comparison with SQLite

| Feature | PostgreSQL | SQLite |
|---------|-----------|--------|
| Network access | Yes | No (embedded) |
| Concurrency | Excellent | Good (WAL mode) |
| Scalability | Horizontal | Single file |
| Setup complexity | Higher | Lower |
| Use case | Production, multi-instance | Development, single-instance |

## License

Part of the nightcrier project. See project LICENSE for details.
