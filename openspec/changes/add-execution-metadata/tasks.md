## 1. Database Schema

- [ ] 1.1 Create migration `000002_add_execution_metadata.up.sql` adding `agent_cli`, `agent_model`, and `cluster_name` columns to `agent_executions`
- [ ] 1.2 Create migration `000002_add_execution_metadata.down.sql` to drop the new columns
- [ ] 1.3 Add indexes on new columns for query performance

## 2. Storage Layer

- [ ] 2.1 Update `AgentExecution` struct in `internal/storage/statestore.go` with new fields
- [ ] 2.2 Update SQLite `RecordAgentExecution` to insert new columns
- [ ] 2.3 Update PostgreSQL `RecordAgentExecution` to insert new columns
- [ ] 2.4 Update any query methods that return `AgentExecution` to select new columns

## 3. Integration

- [ ] 3.1 Update executor to pass agent CLI, model, and cluster name when recording executions
- [ ] 3.2 Ensure metadata is captured at agent start time (not just completion)

## 4. Testing

- [ ] 4.1 Add unit tests for SQLite storage with new metadata fields
- [ ] 4.2 Add unit tests for PostgreSQL storage with new metadata fields
- [ ] 4.3 Test migration up/down cycle
- [ ] 4.4 Integration test: verify metadata persisted and retrievable
