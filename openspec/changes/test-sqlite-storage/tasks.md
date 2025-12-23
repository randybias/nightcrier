# Tasks: test-sqlite-storage

## Phase 1: Unit Test Expansion

- [ ] Add edge case tests for NULL and empty values in all fields
- [ ] Add tests for special characters and Unicode in string fields
- [ ] Add tests for very long strings (approaching TEXT column limits)
- [ ] Add tests for timestamp edge cases (epoch, far future, timezone handling)
- [ ] Add tests for boundary conditions in filters (Limit=0, Offset beyond data)
- [ ] Add error condition tests (invalid incident ID formats, constraint violations)
- [ ] Add tests for CreateIncident with nil Resource pointer
- [ ] Add tests for RecordAgentExecution with nil optional fields

## Phase 2: File-Based Database Tests

- [ ] Create test helper for temp database file creation/cleanup
- [ ] Add test for file-based database creation in new directory
- [ ] Add test for WAL mode journal creation and behavior
- [ ] Add test for database file permissions
- [ ] Add test for reopening existing database file
- [ ] Add test for database path resolution (relative vs absolute)
- [ ] Add test for database busy timeout behavior

## Phase 3: Migration Tests

- [ ] Create migration test infrastructure
- [ ] Add test for clean migration on fresh database
- [ ] Add test for migration idempotency (running migrations twice)
- [ ] Add test for migration version tracking
- [ ] Add test for migration rollback capability (if supported)

## Phase 4: Integration Tests

- [ ] Create integration test package in `tests/integration/sqlite/`
- [ ] Add test for SQLite store creation from config
- [ ] Add test for full incident lifecycle (create, update, complete)
- [ ] Add test for agent execution recording through full pipeline
- [ ] Add test for triage report storage
- [ ] Add test for ListIncidents with various filter combinations
- [ ] Add test for concurrent incident creation from multiple goroutines

## Phase 5: Live Test Validation

- [ ] Update `run-live-test.sh` to verify SQLite database after test
- [ ] Add database inspection step to test report
- [ ] Create `verify-sqlite-storage.sh` script to query and validate stored data
- [ ] Add test case for verifying incident was persisted to SQLite
- [ ] Add test case for verifying triage report was persisted

## Phase 6: Documentation and Coverage

- [ ] Document SQLite testing approach in tests/README.md
- [ ] Run coverage report and verify 80%+ coverage for sqlite package
- [ ] Add coverage report to CI if not already present
