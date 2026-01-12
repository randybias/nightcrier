## Phase 1: Foundation - NATS Infrastructure and Run Events ✅ COMPLETE

### 1.1 Configuration ✅
- [x] 1.1.1 Add `NATSConfig` struct to `internal/config/config.go` with fields: `Enabled`, `Server`, `Token`, `ConnectTimeout`, `ReconnectWait`
- [x] 1.1.2 Add environment variable bindings for NATS config fields in `bindEnvVars()`
- [x] 1.1.3 Add `ValidateNATSConfig()` method to validate NATS settings when enabled
- [x] 1.1.4 Update `configs/config.example.yaml` with NATS configuration section
- [x] 1.1.5 Write unit tests for NATS config validation (7 test functions, all passing)

### 1.2 Database Schema ✅
- [x] 1.2.1 Create migration `000002_add_activity_tracking.up.sql`:
  - Add `current_activity` (TEXT) column to `agent_executions`
  - Add `current_activity_started_at` (TIMESTAMP) column to `agent_executions`
  - Add `last_activity` (TEXT) column to `agent_executions`
  - Add `last_activity_finished_at` (TIMESTAMP) column to `agent_executions`
- [x] 1.2.2 Create migration `000002_add_activity_tracking.down.sql` for rollback
- [x] 1.2.3 Update `internal/storage/statestore.go` interface with:
  - `UpdateExecutionActivity(ctx context.Context, executionID, activity string, activityTime time.Time) error`
  - Method rotates current → last before updating current
- [x] 1.2.4 Implement method in postgres and sqlite storage backends
- [x] 1.2.5 Write unit tests for activity rotation (3 test cases per backend, all passing)

### 1.3 NATS Client Package ✅
- [x] 1.3.1 Create `internal/nats/client.go` with:
  - `Client` struct wrapping nats.Conn
  - `Connect(server, token string, opts ...Option) (*Client, error)` constructor
  - `Publish(subject string, data []byte) error` method (fire-and-forget, 3s timeout)
  - `Close()` method
- [x] 1.3.2 Create `internal/nats/listener.go` with:
  - `Listener` struct with NATS client and StateStore
  - `Start(ctx context.Context) error` method to subscribe to "triage.>" wildcard
  - `handleProgressEvent(msg *nats.Msg)` for parsing and updating activity
- [x] 1.3.3 Create `internal/nats/events.go` with:
  - `ProgressEvent` struct matching JSON schema
  - JSON marshal/unmarshal methods
  - `SubjectForEvent(incidentID, eventType string) string` helper
- [x] 1.3.4 Add nats.go dependency: `github.com/nats-io/nats.go v1.48.0`
- [x] 1.3.5 Write unit tests for NATS client with embedded NATS server (17 tests, all passing)

### 1.4 Nightcrier Integration ✅
- [x] 1.4.1 Update `cmd/nightcrier/main.go` to:
  - Initialize NATS client if `cfg.NATS.Enabled` (main.go:397-436)
  - Start listener goroutine before dispatcher
  - Pass NATS config to K8s executor via K8sExecutorConfig
- [x] 1.4.2 Update `internal/agent/k8s_executor.go` to:
  - Accept NATS config in K8sExecutorConfig struct
  - Add NATS_SERVER, NATS_TOKEN, NATS_ENABLED to Job env vars when enabled
  - Add CLUSTER env var for event context
- [x] 1.4.3 Add graceful shutdown for NATS listener in main shutdown sequence (main.go:518-526)
- [x] 1.4.4 Code compiles successfully, ready for integration testing

### 1.5 Container Updates - Run Events ✅
- [x] 1.5.1 Update `nc-agent-runner/Dockerfile`:
  - Multi-stage build with golang:1.24-bookworm to compile nats-cli
  - Copy nats binary to final image at /usr/local/bin/nats
  - Verified multi-architecture support (amd64, arm64)
- [x] 1.5.2 Create `nc-agent-runner/scripts/nats-publish.sh`:
  - Helper function `publish_event()` to publish JSON to NATS subject
  - Checks NATS_ENABLED, exits silently if disabled
  - 3-second timeout with graceful failure handling
- [x] 1.5.3 Update `nc-agent-runner/entrypoint.sh`:
  - Sources nats-publish.sh at line 17
  - Calls `publish_run_started()` after preflight validation at line 579
  - Calls `publish_run_completed()` in teardown at line 447
- [x] 1.5.4 Create JSON payload builders in nats-publish.sh:
  - `build_run_started_event()` - incident_id, cluster, agent info
  - `build_run_completed_event()` - adds exit_code
- [x] 1.5.5 Container builds successfully, nats-cli verified in PATH

### 1.6 Run Lifecycle Schema ✅
- [x] 1.6.1 Create migration `000003_add_run_lifecycle.up.sql`:
  - Rename `started_at` → `job_started_at` in agent_executions
  - Rename `completed_at` → `job_completed_at` in agent_executions
  - Add `run_started_at` TIMESTAMP column
  - Add `run_completed_at` TIMESTAMP column
  - Add `run_exit_code` INTEGER column
  - Add indexes for new columns
- [x] 1.6.2 Create migration `000003_add_run_lifecycle.down.sql` for rollback
- [x] 1.6.3 Create migration `000004_rename_incidents_lifecycle.up.sql`:
  - Rename `started_at` → `job_started_at` in incidents table
  - Rename `completed_at` → `job_completed_at` in incidents table
- [x] 1.6.4 Create migration `000004_rename_incidents_lifecycle.down.sql` for rollback
- [x] 1.6.5 Update StateStore interface with:
  - `UpdateRunStarted(ctx, incidentID, runStartedAt) error`
  - `UpdateRunCompleted(ctx, incidentID, runCompletedAt, runExitCode) error`
- [x] 1.6.6 Implement methods in postgres and sqlite storage backends
- [x] 1.6.7 Update all SQL queries to use new column names (job_started_at, job_completed_at)
- [x] 1.6.8 Update NATS listener to call UpdateRunStarted/UpdateRunCompleted on events
- [x] 1.6.9 Update test schemas in postgres_test.go and sqlite_test.go

### 1.7 Phase 1 Validation ✅ COMPLETE
- [x] 1.7.1 Deploy NATS server to test cluster (nats.ospo-dev.miralabs.dev:18453)
- [x] 1.7.2 Deploy nightcrier with NATS enabled (config-multicluster.yaml)
- [x] 1.7.3 Trigger test incidents, verify:
  - run.started events received and run_started_at populated
  - run.completed events received with run_completed_at and run_exit_code
  - Job lifecycle (job_started_at, job_completed_at) tracked separately
- [x] 1.7.4 Verify database schema correctly distinguishes:
  - Job lifecycle (Go orchestration): job_started_at, job_completed_at
  - Run lifecycle (container): run_started_at, run_completed_at, run_exit_code
- [x] 1.7.5 Test NATS reconnection: verify listener handles disconnects gracefully

### 1.8 Timing Fix ✅
- [x] 1.8.1 Fix race condition: agent_execution record must exist before NATS run.started arrives
- [x] 1.8.2 Update k8s_executor.go to call RecordAgentExecution immediately after Job creation
- [x] 1.8.3 Processor's RecordAgentExecution call now updates (UPSERT) rather than inserts
- [x] 1.8.4 Remove legacy executor code from main.go (duplicate RecordAgentExecution, CompleteIncident calls)

**Implementation Status:** Phase 1 COMPLETE. All tasks implemented and validated end-to-end. Run lifecycle events working correctly with proper Job/Run separation.

## Phase 2: Claude Hooks Integration ✅ COMPLETE

### 2.1 Hook Script ✅
- [x] 2.1.1 Create `nc-agent-runner/hooks/nats-executing.sh`:
  - Read JSON from stdin (Claude PreToolUse hook format)
  - Extract command from tool_input.command using jq
  - Truncate command to first 100 characters for activity field
  - Publish to `triage.$INCIDENT_ID.executing`
- [x] 2.1.2 Create JSON payload builder for executing event (inline jq)
- [x] 2.1.3 Handle NATS_ENABLED check (exit 0 immediately if disabled)
- [x] 2.1.4 Shellcheck validation passed

### 2.2 Claude Configuration ✅
- [x] 2.2.1 Create `nc-agent-runner/hooks/claude-settings.json.template`:
  - PreToolUse hook on Bash matcher
  - 5 second timeout for hook execution
- [x] 2.2.2 Update `nc-agent-runner/entrypoint.sh` setup_agent_paths():
  - If AGENT_CLI=claude AND NATS_ENABLED=true
  - Copy settings.json.template to ~/.claude/settings.json
- [x] 2.2.3 Update Dockerfile to include hooks directory (COPY hooks/)
- [x] 2.2.4 Ensure hook scripts are executable (chmod +x in Dockerfile)

### 2.3 Phase 2 Validation ✅
- [x] 2.3.1 Build updated container with hooks
- [x] 2.3.2 Run Claude agent with NATS enabled, verify:
  - run.started/completed events still work
  - "executing" events for each Bash tool call
  - current_activity and last_activity updated correctly in database
- [x] 2.3.3 Hook failure isolation verified (fire-and-forget pattern)
- [x] 2.3.4 No noticeable performance impact (3s timeout, async publish)
- [x] 2.3.5 Documentation updated

### 2.4 SQLite Compatibility Fix ✅
- [x] 2.4.1 Fix UpdateExecutionActivity to use incident_id (consistent with other methods)
- [x] 2.4.2 Fix SQLite syntax: UPDATE...ORDER BY LIMIT not supported, use subquery
- [x] 2.4.3 Apply same fix to UpdateRunStarted and UpdateRunCompleted
- [x] 2.4.4 Update tests to use incident_id and set run_started_at before activity updates

**Implementation Status:** Phase 2 COMPLETE. Claude hooks working, executing events tracked.

## Phase 3: Documentation and Cleanup

### 3.1 NATS Graceful Fallback Fix ✅
- [x] 3.1.1 Fix `publish_event()` to return 0 on failure (was returning 1, causing script exit with `set -e`)
- [x] 3.1.2 Add `check_nats_connectivity()` function that runs once at script source time
- [x] 3.1.3 Add `NATS_AVAILABLE` global flag set by connectivity check (prevents repeated 3s timeouts)
- [x] 3.1.4 Update `publish_event()` to check `NATS_AVAILABLE` flag and skip instantly if unavailable
- [x] 3.1.5 Run shellcheck validation on updated script
- [x] 3.1.6 Rebuild container image with fix

**Bug context:** Agent Jobs were failing immediately after NATS timeout because `publish_event()` returned 1
on failure, which caused the script to exit due to `set -euo pipefail` in both entrypoint.sh and nats-publish.sh,
despite the "continuing anyway" message. The fix ensures NATS failures don't affect agent execution.

### 3.2 Observability
- [ ] 3.2.1 Add structured logging for NATS events in listener
- [ ] 3.2.2 Log last_activity updates at debug level

### 3.3 Documentation
- [ ] 3.3.1 Add NATS deployment guide to docs/ (Helm chart or manifests)
- [ ] 3.3.2 Document NATS configuration options in config.example.yaml comments
- [ ] 3.3.3 Add troubleshooting section for NATS connectivity issues
- [ ] 3.3.4 Document event schema and subscription patterns
