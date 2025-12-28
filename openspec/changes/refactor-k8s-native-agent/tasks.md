# Tasks: Refactor to K8s-Native Stateless Agent Execution

## Phase 1: Foundation (K8s Executor Core)

### 1.1 K8s Client Setup
- [x] Add client-go dependency to go.mod
- [x] Create `internal/agent/k8s/client.go` with K8s client initialization
- [x] Implement in-cluster config detection
- [x] Implement out-of-cluster config (kubeconfig file)
- [x] Add unit tests for client initialization

### 1.2 ConfigMap Generator
- [x] Create `internal/agent/k8s/configmap.go`
- [x] Implement `CreateIncidentConfigMap()` function
- [x] Add incident.json to ConfigMap data
- [x] Add permissions.json to ConfigMap data
- [x] Add system-prompt.md to ConfigMap data
- [x] Add labels for incident identification
- [x] Implement `DeleteConfigMap()` function
- [x] Add unit tests with fake clientset

### 1.3 Job Generator
- [x] Create `internal/agent/k8s/job.go`
- [x] Implement `CreateJob()` function
- [x] Configure container spec with environment variables
- [x] Configure volume mounts for ConfigMap and Secrets
- [x] Configure resource limits and requests
- [x] Set TTL, activeDeadlineSeconds, backoffLimit
- [x] Add labels for incident identification
- [x] Use image name `nc-agent-runner`
- [x] Add unit tests with fake clientset

### 1.4 Presigned URL Generation for Outputs
- [x] Create `internal/agent/k8s/urls.go`
- [x] Implement `GenerateOutputURLs()` function using existing ObjectStore
- [x] Generate presigned PUT URL for `report.md`
- [x] Generate presigned PUT URL for `agent.log`
- [x] Generate presigned PUT URL for `session.tar.gz`
- [x] Generate presigned PUT URL for `result.json`
- [x] Generate presigned PUT URL for `commands-executed.log` (new)
- [x] Set URL expiration based on job timeout + buffer
- [x] Storage paths: `incidents/{incident-id}/results/{filename}`
- [x] Add unit tests

## Phase 2: Container Image and Entrypoint

### 2.1 Container Directory Structure
- [x] Create `nc-agent-runner/` directory (new name)
- [x] Move/copy relevant files from `agent-container/`
- [x] Create `nc-agent-runner/Dockerfile`
- [x] Create `nc-agent-runner/entrypoint.sh`

### 2.2 Agent Skills Setup (Research Applied)
- [x] Implement dynamic skill cloning from GitHub (`clone_skills()` function)
- [x] Clone https://github.com/randybias/k8s4agents at container startup
- [x] Implement Claude skill symlink: `~/.claude/skills/` → `/home/agent/skills/k8s4agents/skills`
- [x] Implement Codex skill symlink: `~/.codex/skills/` → `/home/agent/skills/k8s4agents/skills`
- [x] Implement Gemini context: Ensure `GEMINI.md` exists at `/home/agent/`
- [x] Implement Goose skill symlinks (multiple paths):
  - [x] `~/.config/agents/skills/` → `/home/agent/skills/k8s4agents/skills`
  - [x] `~/.config/goose/skills/` → `/home/agent/skills/k8s4agents/skills`
- [x] Verify skill loading works for each agent type
- [x] Document skill setup in entrypoint comments

### 2.3 Entrypoint Script
- [x] Create `nc-agent-runner/entrypoint.sh`
- [x] Implement `setup_agent_paths()` function with all agent types
- [x] Export `SESSION_DIR` for each agent type
- [x] Implement agent-specific initialization:
  - [x] Claude: symlinks only
  - [x] Codex: symlinks + `codex login --with-api-key`
  - [x] Gemini: context file check
  - [x] Goose: symlinks + `GOOSE_DISABLE_KEYRING=1`

### 2.4 Agent Invocation with Real-Time Logging
- [x] Implement `run_agent()` function in entrypoint
- [x] Create `/home/agent/logs/` directory
- [x] Use `tee` to capture stdout/stderr AND display in real-time
- [x] Add Claude CLI invocation with correct flags
- [x] Add Codex CLI invocation with correct flags
- [x] Add Gemini CLI invocation with correct flags
- [x] Add Goose CLI invocation with correct flags
- [x] Capture exit code for result.json

### 2.5 In-Container Command Extraction
- [x] Implement `extract_commands()` function
- [x] Claude extraction: Parse JSONL from `~/.claude/projects/*/`
- [x] Codex extraction: Parse JSONL from `~/.codex/sessions/`
- [x] Gemini extraction: Parse JSON from `~/.gemini/tmp/*/chats/session-*.json`
- [x] Goose extraction: Query SQLite from `~/.config/goose/sessions.db`
- [x] Create `commands-executed.log` with `$ ` prefix per command
- [x] Handle missing session data gracefully

### 2.6 Teardown and Upload
- [x] Implement `teardown()` function with trap
- [x] Call `extract_commands()` before uploads
- [x] Archive session directory to `/tmp/session.tar.gz`
- [x] Upload `report.md` via `curl -X PUT`
- [x] Upload `agent.log` via `curl -X PUT`
- [x] Upload `session.tar.gz` via `curl -X PUT`
- [x] Upload `commands-executed.log` via `curl -X PUT` (new)
- [x] Create and upload `result.json` with exit code
- [x] Register trap for EXIT signal
- [x] Handle upload failures gracefully (log error, continue)

### 2.7 Dockerfile
- [x] Create `nc-agent-runner/Dockerfile`
- [x] Install all agent CLIs (claude, codex, gemini, goose)
- [x] Install kubectl 1.31+, helm 3.x
- [x] Install utilities: curl, jq, sqlite3, tar
- [x] Install search tools: ripgrep, fd, fzf
- [x] Create empty `/home/agent/skills/` directory (cloned at runtime)
- [x] Add Codex managed config at `/etc/codex/managed_config.toml`
- [x] Set ENTRYPOINT to `entrypoint.sh`
- [x] Test image build

### 2.7.1 Codex-Specific Fixes
- [x] Fix Codex prompt passing: concatenate system prompt + investigation prompt into single argument
- [x] Add managed config with sandbox bypass, approval policy "never", network access enabled
- [x] Fix Codex authentication: pipe API key via stdin to `codex login --with-api-key`
- [x] Update system prompt to be agent-agnostic with multiple skill paths
- [x] Add debug logging for prompt construction and command execution
- [x] Document known issues: commands-executed.log extraction needs Codex session format fix

### 2.8 Image Build Scripts
- [x] Create `scripts/build-agent-image.sh`
- [x] Create `scripts/update-agent-versions.sh` for checking/updating CLI versions
- [x] Add `make build-agent-image` target to Makefile
- [x] Document manual update process
- [x] (Future) Create `.github/workflows/update-agent-image.yml` for weekly builds

## Phase 3: Job Lifecycle Management

### 3.1 Job Monitoring
- [x] Create `internal/agent/k8s/watcher.go`
- [x] Implement `WatchJob()` using K8s watch API
- [x] Detect Job completion (Succeeded/Failed)
- [x] Implement timeout for watch operation
- [x] Handle watch connection drops and reconnection

### 3.2 Basic Log Streaming (Minimalist)
- [x] Implement optional log tailing via K8s API
- [x] Log key events: Job started, Pod running, Job completed
- [x] Support `kubectl logs -f` for detailed streaming
- [x] Note: Advanced streaming is out of scope, can expand later

### 3.3 Result Retrieval from Object Store
- [x] Create `internal/agent/k8s/results.go`
- [x] Implement `RetrieveResults()` function
- [x] Download `result.json` from Object Store to get exit code
- [x] Download `report.md` (investigation markdown) from Object Store
- [x] Download `agent.log` from Object Store
- [x] Download `commands-executed.log` from Object Store (new)
- [x] Optionally download `session.tar.gz` for debugging
- [x] Handle missing files (Job failed before upload completed)
- [x] Return structured result with all artifacts

### 3.4 Cleanup
- [x] Implement ConfigMap cleanup after Job completion
- [x] Implement orphan detection on startup
- [x] Delete orphaned ConfigMaps older than 24 hours with matching labels
- [x] Add cleanup logging

## Phase 4: Artifact Processing and Database Integration

### 4.1 Report Processing (Markdown to HTML)
- [x] After downloading `report.md`, convert to HTML using existing `ConvertMarkdownToHTML()`
- [x] Create `InvestigationHTML` from the markdown content
- [x] Ensure incident ID is passed for report header

### 4.2 Storage Integration
- [x] Update or create method to upload processed artifacts
- [x] Upload `investigation.html` to Object Store alongside raw markdown
- [x] Upload `incident.json` to Object Store (for completeness)
- [x] Ensure `SaveResult` contains all artifact URLs (canonical and signed)
- [x] Verify URLs are accessible for downstream consumers (Slack, dashboard)

### 4.3 Database Updates (StateStore)
- [x] Update `RecordTriageReport()` call with markdown and HTML content
- [x] Store report markdown in `triage_reports.report_markdown`
- [x] Store report HTML in `triage_reports.report_html`
- [x] Update `RecordAgentExecution()` with log URLs instead of file paths
- [x] Update `agent_executions.log_paths` to store Object Store URLs
- [x] Add migration if schema changes needed for URL storage

### 4.4 Incident Completion
- [x] Call `CompleteIncident()` with exit code from `result.json`
- [x] Set failure_reason if exit code is non-zero
- [x] Ensure incident status transitions correctly (investigating → resolved/failed)

## Phase 5: K8s Executor Integration

### 5.1 K8s Executor Implementation
- [x] Create `internal/agent/k8s_executor.go`
- [x] Implement `AgentExecutor` interface
- [x] Orchestrate full flow:
  1. [x] Generate presigned PUT URLs for outputs (including commands-executed.log)
  2. [x] Create ConfigMap with incident data
  3. [x] Create Job referencing ConfigMap and Secrets
  4. [x] Watch Job for completion (with basic logging)
  5. [x] Retrieve artifacts from Object Store
  6. [x] Convert markdown to HTML
  7. [x] Save all artifacts via Storage interface
  8. [x] Record execution and report in StateStore
  9. [x] Complete incident with result
  10. [x] Cleanup ConfigMap
- [x] Integrate with existing incident handler

### 5.2 Configuration
- [x] Add K8s executor config to `internal/config/config.go`
- [x] Add `k8s.namespace` option (default: `nightcrier`)
- [x] Add `k8s.image` option (default: `nc-agent-runner:latest`)
- [x] Add `k8s.timeout` option (default: 600)
- [x] Add `k8s.memory_limit` option (default: `2Gi`)
- [x] Add `k8s.cpu_limit` option (default: `1`)
- [x] Add `k8s.cleanup_ttl` option (default: 3600)
- [x] Update config.example.yaml with K8s options

### 5.3 Wire Up
- [x] Update `internal/agent/executor.go` to use K8s executor directly (no Docker fallback)
- [x] Remove Docker execution logic
- [x] Update incident handler to use new executor
- [x] Add error handling for K8s API failures

## Phase 6: Local Development Setup

### 6.1 Kind Setup
- [x] Create `deploy/dev/kind-config.yaml` for local cluster
- [x] Create `deploy/dev/namespace.yaml`
- [x] Create `deploy/dev/secrets.yaml` (template for API keys)
- [x] Create `deploy/dev/rbac.yaml` for executor permissions
- [x] Create `deploy/dev/kubeconfig-secret.yaml` (template for triage kubeconfig)

### 6.2 Development Scripts
- [x] Create `scripts/dev-setup.sh` to bootstrap kind cluster
- [x] Create `scripts/dev-teardown.sh` to clean up
- [x] Update Makefile with dev targets (`make build`, `make load-kind`)
- [ ] Add `make dev-cluster` target (optional - scripts work well)
- [ ] Add `make dev-secrets` target (optional - handled by dev-setup.sh)

### 6.3 Documentation
- [x] Create `docs/DEV_SETUP.md` with kind-based local dev instructions
- [x] Create `docs/CONTRIBUTING.md` with contribution guidelines
- [x] Document Secret provisioning for kubeconfigs
- [x] Document API key Secret setup
- [x] Document Object Store configuration for local dev (can use `mem://` or local minio)

## Phase 7: Cleanup and Migration

### 7.1 Move Docker Scripts to Reference
- [ ] Create `agent-container/reference/` directory
- [ ] Move `agent-container/run-agent.sh` to reference
- [ ] Move `agent-container/runners/` to reference
- [ ] Move `agent-container/test_*.sh` to reference
- [ ] Add README in reference/ explaining these are historical

### 7.2 Remove Docker Execution Code
- [ ] Remove Docker execution logic from `internal/agent/executor.go`
- [ ] Remove `agent.runtime` config option if present
- [ ] Update any imports/dependencies

### 7.3 Update Tests
- [ ] Add K8s executor unit tests with fake clientset
- [ ] Add entrypoint.sh tests (bats or similar)
- [ ] Update CI to use kind for integration tests
- [ ] Add tests for artifact upload/download cycle
- [ ] Add tests for command extraction per agent type

### 7.4 Documentation Updates
- [ ] Update agent-container/README.md (point to nc-agent-runner/)
- [ ] Update main README.md

## Phase 8: Validation (Artifact Verification)

### 8.1 End-to-End Testing
- [ ] Test with Claude agent on kind cluster
- [ ] Test with Codex agent on kind cluster
- [ ] Test with Gemini agent on kind cluster
- [ ] Test with Goose agent on kind cluster
- [ ] Verify all outputs exist in Object Store after each test

### 8.2 Artifact Verification Tests
- [ ] **Verify `report.md` exists** in Object Store after successful run
- [ ] **Verify `report.md` contains valid markdown** (non-empty, has headers)
- [ ] **Verify `investigation.html` exists** in Object Store
- [ ] **Verify HTML is valid** (has DOCTYPE, title includes incident ID)
- [ ] **Verify `agent.log` exists** in Object Store
- [ ] **Verify `session.tar.gz` exists** for agents with session state
- [ ] **Verify `result.json` exists** with exit_code field
- [ ] **Verify `commands-executed.log` exists** and contains `$` prefixed commands
- [ ] **Verify database has report** - query `triage_reports` for incident
- [ ] **Verify database has execution** - query `agent_executions` for incident
- [ ] **Verify URLs are accessible** - GET request on signed URLs returns 200
- [ ] **Verify incident status** - incident marked resolved/failed appropriately

### 8.3 Command Extraction Verification
- [ ] Verify Claude command extraction from JSONL
- [ ] Verify Codex command extraction from JSONL
- [ ] Verify Gemini command extraction from JSON
- [ ] Verify Goose command extraction from SQLite
- [ ] Test graceful handling when session data is missing

### 8.4 Failure Mode Testing
- [ ] Test Job timeout behavior (trap fires, partial artifacts uploaded)
- [ ] Test OOM behavior (trap may not fire, verify what we can recover)
- [ ] Test Object Store upload failures (verify other artifacts still uploaded)
- [ ] Test K8s API unavailability (verify graceful failure with logging)
- [ ] Test missing report (agent crashed before producing report)
- [ ] Verify database records failure reason when artifacts missing

### 8.5 Performance Validation
- [ ] Measure Job startup latency (ConfigMap create → Pod running)
- [ ] Measure output upload time (large agent.log)
- [ ] Compare with previous Docker execution time
- [ ] Document any regressions

### 8.6 Artifact Retention Verification
- [ ] Verify artifacts are accessible after 1 hour (within signed URL expiry)
- [ ] Verify canonical URLs can be re-signed for later access
- [ ] Verify old incidents can have reports regenerated from stored markdown
