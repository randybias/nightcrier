# Tasks: Refactor to K8s-Native Stateless Agent Execution

## Phase 1: Foundation (K8s Executor Core)

### 1.1 K8s Client Setup
- [ ] Add client-go dependency to go.mod
- [ ] Create `internal/agent/k8s/client.go` with K8s client initialization
- [ ] Implement in-cluster config detection
- [ ] Implement out-of-cluster config (kubeconfig file)
- [ ] Add unit tests for client initialization

### 1.2 ConfigMap Generator
- [ ] Create `internal/agent/k8s/configmap.go`
- [ ] Implement `CreateIncidentConfigMap()` function
- [ ] Add incident.json to ConfigMap data
- [ ] Add permissions.json to ConfigMap data
- [ ] Add system-prompt.md to ConfigMap data
- [ ] Add labels for incident identification
- [ ] Implement `DeleteConfigMap()` function
- [ ] Add unit tests with fake clientset

### 1.3 Job Generator
- [ ] Create `internal/agent/k8s/job.go`
- [ ] Implement `CreateJob()` function
- [ ] Configure container spec with environment variables
- [ ] Configure volume mounts for ConfigMap and Secrets
- [ ] Configure resource limits and requests
- [ ] Set TTL, activeDeadlineSeconds, backoffLimit
- [ ] Add labels for incident identification
- [ ] Use image name `nc-agent-runner`
- [ ] Add unit tests with fake clientset

### 1.4 Presigned URL Generation for Outputs
- [ ] Create `internal/agent/k8s/urls.go`
- [ ] Implement `GenerateOutputURLs()` function using existing ObjectStore
- [ ] Generate presigned PUT URL for `report.md`
- [ ] Generate presigned PUT URL for `agent.log`
- [ ] Generate presigned PUT URL for `session.tar.gz`
- [ ] Generate presigned PUT URL for `result.json`
- [ ] Generate presigned PUT URL for `commands-executed.log` (new)
- [ ] Set URL expiration based on job timeout + buffer
- [ ] Storage paths: `incidents/{incident-id}/results/{filename}`
- [ ] Add unit tests

## Phase 2: Container Image and Entrypoint

### 2.1 Container Directory Structure
- [ ] Create `nc-agent-runner/` directory (new name)
- [ ] Move/copy relevant files from `agent-container/`
- [ ] Create `nc-agent-runner/Dockerfile`
- [ ] Create `nc-agent-runner/entrypoint.sh`

### 2.2 Agent Skills Setup (Research Applied)
- [ ] Implement Claude skill symlink: `~/.claude/skills/` → `/home/agent/skills/`
- [ ] Implement Codex skill symlink: `~/.codex/skills/` → `/home/agent/skills/`
- [ ] Implement Gemini context: Ensure `GEMINI.md` exists at `/home/agent/`
- [ ] Implement Goose skill symlinks (multiple paths):
  - `~/.config/agents/skills/` → `/home/agent/skills/`
  - `~/.config/goose/skills/` → `/home/agent/skills/`
- [ ] Verify skill loading works for each agent type
- [ ] Document skill setup in entrypoint comments

### 2.3 Entrypoint Script
- [ ] Create `nc-agent-runner/entrypoint.sh`
- [ ] Implement `setup_agent_paths()` function with all agent types
- [ ] Export `SESSION_DIR` for each agent type
- [ ] Implement agent-specific initialization:
  - Claude: symlinks only
  - Codex: symlinks + `codex login --with-api-key`
  - Gemini: context file check
  - Goose: symlinks + `GOOSE_DISABLE_KEYRING=1`

### 2.4 Agent Invocation with Real-Time Logging
- [ ] Implement `run_agent()` function in entrypoint
- [ ] Create `/home/agent/logs/` directory
- [ ] Use `tee` to capture stdout/stderr AND display in real-time
- [ ] Add Claude CLI invocation with correct flags
- [ ] Add Codex CLI invocation with correct flags
- [ ] Add Gemini CLI invocation with correct flags
- [ ] Add Goose CLI invocation with correct flags
- [ ] Capture exit code for result.json

### 2.5 In-Container Command Extraction
- [ ] Implement `extract_commands()` function
- [ ] Claude extraction: Parse JSONL from `~/.claude/projects/*/`
- [ ] Codex extraction: Parse JSONL from `~/.codex/sessions/`
- [ ] Gemini extraction: Parse JSON from `~/.gemini/tmp/*/chats/session-*.json`
- [ ] Goose extraction: Query SQLite from `~/.config/goose/sessions.db`
- [ ] Create `commands-executed.log` with `$ ` prefix per command
- [ ] Handle missing session data gracefully

### 2.6 Teardown and Upload
- [ ] Implement `teardown()` function with trap
- [ ] Call `extract_commands()` before uploads
- [ ] Archive session directory to `/tmp/session.tar.gz`
- [ ] Upload `report.md` via `curl -X PUT`
- [ ] Upload `agent.log` via `curl -X PUT`
- [ ] Upload `session.tar.gz` via `curl -X PUT`
- [ ] Upload `commands-executed.log` via `curl -X PUT` (new)
- [ ] Create and upload `result.json` with exit code
- [ ] Register trap for EXIT signal
- [ ] Handle upload failures gracefully (log error, continue)

### 2.7 Dockerfile
- [ ] Create `nc-agent-runner/Dockerfile`
- [ ] Install all agent CLIs (claude, codex, gemini, goose)
- [ ] Install kubectl 1.31+, helm 3.x
- [ ] Install utilities: curl, jq, sqlite3, tar
- [ ] Install search tools: ripgrep, fd, fzf
- [ ] Bake in k8s-troubleshooter skill at `/home/agent/skills/`
- [ ] Set ENTRYPOINT to `entrypoint.sh`
- [ ] Test image build

### 2.8 Image Build Scripts
- [ ] Create `scripts/build-agent-image.sh`
- [ ] Create `scripts/update-agent-versions.sh` for checking/updating CLI versions
- [ ] Add `make build-agent-image` target to Makefile
- [ ] Document manual update process
- [ ] (Future) Create `.github/workflows/update-agent-image.yml` for weekly builds

## Phase 3: Job Lifecycle Management

### 3.1 Job Monitoring
- [ ] Create `internal/agent/k8s/watcher.go`
- [ ] Implement `WatchJob()` using K8s watch API
- [ ] Detect Job completion (Succeeded/Failed)
- [ ] Implement timeout for watch operation
- [ ] Handle watch connection drops and reconnection

### 3.2 Basic Log Streaming (Minimalist)
- [ ] Implement optional log tailing via K8s API
- [ ] Log key events: Job started, Pod running, Job completed
- [ ] Support `kubectl logs -f` for detailed streaming
- [ ] Note: Advanced streaming is out of scope, can expand later

### 3.3 Result Retrieval from Object Store
- [ ] Create `internal/agent/k8s/results.go`
- [ ] Implement `RetrieveResults()` function
- [ ] Download `result.json` from Object Store to get exit code
- [ ] Download `report.md` (investigation markdown) from Object Store
- [ ] Download `agent.log` from Object Store
- [ ] Download `commands-executed.log` from Object Store (new)
- [ ] Optionally download `session.tar.gz` for debugging
- [ ] Handle missing files (Job failed before upload completed)
- [ ] Return structured result with all artifacts

### 3.4 Cleanup
- [ ] Implement ConfigMap cleanup after Job completion
- [ ] Implement orphan detection on startup
- [ ] Delete orphaned ConfigMaps older than 24 hours with matching labels
- [ ] Add cleanup logging

## Phase 4: Artifact Processing and Database Integration

### 4.1 Report Processing (Markdown to HTML)
- [ ] After downloading `report.md`, convert to HTML using existing `ConvertMarkdownToHTML()`
- [ ] Create `InvestigationHTML` from the markdown content
- [ ] Ensure incident ID is passed for report header

### 4.2 Storage Integration
- [ ] Update or create method to upload processed artifacts
- [ ] Upload `investigation.html` to Object Store alongside raw markdown
- [ ] Upload `incident.json` to Object Store (for completeness)
- [ ] Ensure `SaveResult` contains all artifact URLs (canonical and signed)
- [ ] Verify URLs are accessible for downstream consumers (Slack, dashboard)

### 4.3 Database Updates (StateStore)
- [ ] Update `RecordTriageReport()` call with markdown and HTML content
- [ ] Store report markdown in `triage_reports.report_markdown`
- [ ] Store report HTML in `triage_reports.report_html`
- [ ] Update `RecordAgentExecution()` with log URLs instead of file paths
- [ ] Update `agent_executions.log_paths` to store Object Store URLs
- [ ] Add migration if schema changes needed for URL storage

### 4.4 Incident Completion
- [ ] Call `CompleteIncident()` with exit code from `result.json`
- [ ] Set failure_reason if exit code is non-zero
- [ ] Ensure incident status transitions correctly (investigating → resolved/failed)

## Phase 5: K8s Executor Integration

### 5.1 K8s Executor Implementation
- [ ] Create `internal/agent/k8s_executor.go`
- [ ] Implement `AgentExecutor` interface
- [ ] Orchestrate full flow:
  1. Generate presigned PUT URLs for outputs (including commands-executed.log)
  2. Create ConfigMap with incident data
  3. Create Job referencing ConfigMap and Secrets
  4. Watch Job for completion (with basic logging)
  5. Retrieve artifacts from Object Store
  6. Convert markdown to HTML
  7. Save all artifacts via Storage interface
  8. Record execution and report in StateStore
  9. Complete incident with result
  10. Cleanup ConfigMap
- [ ] Integrate with existing incident handler

### 5.2 Configuration
- [ ] Add K8s executor config to `internal/config/config.go`
- [ ] Add `k8s.namespace` option (default: `nightcrier`)
- [ ] Add `k8s.image` option (default: `nc-agent-runner:latest`)
- [ ] Add `k8s.timeout` option (default: 600)
- [ ] Add `k8s.memory_limit` option (default: `2Gi`)
- [ ] Add `k8s.cpu_limit` option (default: `1`)
- [ ] Add `k8s.cleanup_ttl` option (default: 3600)
- [ ] Update config.example.yaml with K8s options

### 5.3 Wire Up
- [ ] Update `internal/agent/executor.go` to use K8s executor directly (no Docker fallback)
- [ ] Remove Docker execution logic
- [ ] Update incident handler to use new executor
- [ ] Add error handling for K8s API failures

## Phase 6: Local Development Setup

### 6.1 Kind Setup
- [ ] Create `deploy/dev/kind-config.yaml` for local cluster
- [ ] Create `deploy/dev/namespace.yaml`
- [ ] Create `deploy/dev/secrets.yaml` (template for API keys)
- [ ] Create `deploy/dev/rbac.yaml` for executor permissions
- [ ] Create `deploy/dev/kubeconfig-secret.yaml` (template for triage kubeconfig)

### 6.2 Development Scripts
- [ ] Create `scripts/dev-setup.sh` to bootstrap kind cluster
- [ ] Create `scripts/dev-teardown.sh` to clean up
- [ ] Update Makefile with dev targets
- [ ] Add `make dev-cluster` target
- [ ] Add `make dev-secrets` target

### 6.3 Documentation
- [ ] Update README.md with kind-based local dev instructions
- [ ] Document Secret provisioning for kubeconfigs
- [ ] Document API key Secret setup
- [ ] Document Object Store configuration for local dev (can use `mem://` or local minio)

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
