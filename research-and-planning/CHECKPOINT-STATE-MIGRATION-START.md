# Checkpoint: State Migration - Ready to Implement

## Current Status
*   **Worktree Created:** `/Users/rbias/code/worktrees/feature/state-migration`
*   **Branch:** `feature/state-migration`
*   **Phase:** Phase 1 (State Migration) - Step 1 (Storage Abstraction)

## Objective
Implement the SQL-based state persistence layer to replace the file-system dependency and enable the future Worker Queue architecture.

## Instructions for Resuming (YOLO Mode)

1.  **Switch Context:** Ensure you are operating within the worktree:
    `cd /Users/rbias/code/worktrees/feature/state-migration`

2.  **Authorize Execution:** Run the agent with permissions to execute `mkdir`, `cat`, `go get`, and `go build` commands without constant confirmation.

3.  **Next Tasks (The Agent will perform these):**
    *   **Define Interface:** Create `internal/storage/repository.go` with the `StateStore` interface.
    *   **Implement SQLite:** Create `internal/storage/sqlite/adapter.go` using `database/sql` and `go-sqlite3` (or a pure-Go alternative like `modernc.org/sqlite` if CGO is a concern).
    *   **Implement Schema:** Write the SQL migration/init scripts to create `incidents`, `fault_events`, `agent_executions`, `triage_reports`.
    *   **Wire Up Main:** Modify `cmd/nightcrier/main.go` to initialize this store.

## Verification
After the agent completes these steps, you should be able to run `go build ./...` and see no compilation errors.
