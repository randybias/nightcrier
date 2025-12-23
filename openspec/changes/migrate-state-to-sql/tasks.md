# Tasks

1.  [x] Define `StateStore` interface in `internal/storage`.
2.  [x] Create initial SQL schema migration files (up/down).
3.  [x] Implement `internal/storage/sqlite` adapter.
4.  [x] Implement `internal/storage/postgres` adapter.
5.  [x] Update `internal/config` to support `storage` configuration (type, connection details).
6.  [x] Refactor `cmd/nightcrier` to initialize `StateStore` based on config.
7.  [x] Integrate `StateStore` into `processEvent` loop for incident creation.
8.  [x] Update `agent` execution logic to record status and completion via `StateStore`.
9.  [x] Add integration tests for SQLite adapter.
10. [x] Add integration tests for Postgres adapter.
11. [x] Validate locally with Docker Postgres instance.
