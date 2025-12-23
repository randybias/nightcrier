# Proposal: test-sqlite-storage

## Summary

Create comprehensive test coverage for the SQLite storage backend to validate it works correctly in all scenarios before relying on it in production. The SQLite implementation was created during the migrate-state-to-sql change but was not exhaustively tested.

## Problem Statement

The SQLite storage option in `internal/storage/sqlite/` has unit tests covering basic CRUD operations but lacks:

1. **Edge case coverage** - NULL values, empty strings, special characters, very long strings
2. **Error condition testing** - Database corruption, disk full, permission errors
3. **Migration testing** - Schema migrations apply correctly on fresh and existing databases
4. **File-based database testing** - All existing tests use `:memory:` mode only
5. **WAL mode validation** - WAL journal mode and concurrency behavior verification
6. **Integration testing** - SQLite works correctly when nightcrier runs end-to-end
7. **Live test validation** - SQLite storage functions correctly during real incident triage

## Scope

### In Scope

- Add comprehensive unit tests for edge cases and error conditions
- Add file-based database tests (vs only in-memory)
- Add migration tests to verify schema evolves correctly
- Add integration tests verifying SQLite with the main application
- Update live testing harness to verify SQLite storage during end-to-end tests
- Document testing approach in tests/README.md

### Out of Scope

- Fixing the existing TestConcurrentAccess bug (separate change)
- PostgreSQL testing (separate effort)
- Performance benchmarking (future work)

## Motivation

SQLite is configured as the default storage backend in the example configs. Without exhaustive testing, there's risk of data loss or corruption in production. This testing work validates the implementation before it handles real incident data.

## Dependencies

- Existing `internal/storage/sqlite/` implementation
- Existing `migrations/000001_initial_schema.up.sql`
- Live testing infrastructure in `tests/live-tests/`

## Risks

- **Low**: Some edge cases may expose bugs that need fixing
- **Low**: File-based tests require careful cleanup to avoid polluting the filesystem

## Success Criteria

1. All new unit tests pass
2. Migration tests verify schema applies correctly to fresh and existing databases
3. Integration tests pass with SQLite storage
4. Live testing validates SQLite stores incident data correctly
5. Test coverage for `internal/storage/sqlite/` exceeds 80%
