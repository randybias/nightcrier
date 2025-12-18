# Implementation Tasks (Phase 2)

## 1. Core Data Structures and Interfaces

- [ ] 1.1 Define `Incident` struct with ID, cluster, namespace, severity, timestamp, event data, logs, and resources
- [ ] 1.2 Define `AgentResult` struct with incident ID, status, workspace dir, timestamps, exit code, output log, artifacts, and error
- [ ] 1.3 Define `AgentStatus` type with constants: created, starting, running, success, failed, timeout, cancelled
- [ ] 1.4 Define `AgentRuntime` interface with methods: RunAgent, GetStatus, Cancel
- [ ] 1.5 Define `WorkspaceManager` interface with methods: CreateWorkspace, SetupSkills, CleanupWorkspace
- [ ] 1.6 Define `ContextBuilder` interface with methods: BuildContextBundle, WriteContextFiles
- [ ] 1.7 Define `AgentExecutor` interface with methods: Execute, Monitor
- [ ] 1.8 Define `ContextBundle` struct with Files map, EnvVars map, SystemPrompt, UserPrompt
- [ ] 1.9 Define `ExecutionConfig` struct with workspace dir, context bundle, timeout, command, and args
- [ ] 1.10 Define `CleanupPolicy` enum for workspace retention strategies

## 2. Configuration Management

- [ ] 2.1 Define configuration struct for agent_runtime with all required fields
- [ ] 2.2 Implement config file parser (YAML) for agent_runtime section
- [ ] 2.3 Implement environment variable override logic with precedence
- [ ] 2.4 Add validation for required configuration fields (agent_command, workspace_root, etc.)
- [ ] 2.5 Implement default values for optional configuration (timeout, retention, etc.)
- [ ] 2.6 Add configuration test coverage for all scenarios

## 3. Workspace Management

- [ ] 3.1 Implement workspace path construction with UUID validation
- [ ] 3.2 Implement workspace directory creation with structure: `.claude/`, `context/`, `output/`
- [ ] 3.3 Set workspace permissions to 0700 (owner-only read/write)
- [ ] 3.4 Implement workspace uniqueness validation (prevent overwrites)
- [ ] 3.5 Implement workspace cleanup with retention policy support
- [ ] 3.6 Add path traversal protection using filepath.Join and filepath.Clean
- [ ] 3.7 Verify all workspace operations stay within workspace_root boundary
- [ ] 3.8 Add workspace manager unit tests for all scenarios

## 4. Skill Loading

**Note:** Skills are now built into the `k8s-triage-agent` Docker container at `/skills/`.
The container clones from https://github.com/randybias/k8s4agents during build.
Claude accesses skills via `/skills/k8s-troubleshooter/SKILL.md` or the `/k8s-troubleshooter` slash command.

- [x] 4.1 Implement skill source directory configuration and validation
      **DONE**: Skills at `/skills/` in container, cloned from GitHub during build
- [x] 4.2 Implement k8s-troubleshooter skill copying to `.claude/skills/k8s-troubleshooter/`
      **DONE**: Skills at `/skills/k8s-troubleshooter/`, slash command at `/root/.claude/commands/`
- [x] 4.3 Verify SKILL.md, references/, and scripts/ are copied correctly
      **DONE**: Full skill directory cloned from GitHub
- [x] 4.4 Add error handling for missing skill source directory
      **DONE**: Build fails if GitHub clone fails
- [ ] 4.5 Implement skill version tracking in workspace metadata (optional)
- [ ] 4.6 Add skill loading unit tests with mock filesystem

## 5. Context Bundle Creation

- [ ] 5.1 Implement incident.json generation with structured metadata
- [ ] 5.2 Implement event.json generation with raw event payload
- [ ] 5.3 Implement logs.txt generation with enriched log excerpts
- [ ] 5.4 Implement cluster-info.json generation with cluster name, namespace, resources
- [ ] 5.5 Implement system-instructions.txt generation with read-only enforcement text
- [ ] 5.6 Implement environment variable builder with all required vars (INCIDENT_ID, INCIDENT_WORKSPACE, etc.)
- [ ] 5.7 Add ANTHROPIC_API_KEY or CLAUDE_API_KEY from config/env
- [ ] 5.8 Add CLAUDE_READ_ONLY_MODE=true environment marker
- [ ] 5.9 Implement context file writing to `context/` directory
- [ ] 5.10 Add context bundle unit tests with sample incidents

## 6. Prompt Construction

- [ ] 6.1 Implement PROMPT.md template with incident summary, severity, context references
- [ ] 6.2 Add explicit read-only instructions to prompt
- [ ] 6.3 Add expected outputs and artifact location guidance
- [ ] 6.4 Implement severity-based prompt customization (urgency emphasis)
- [ ] 6.5 Add prompt generation unit tests for different severities

## 7. Agent Command Construction

- [ ] 7.1 Implement command builder to construct `claude` command with all flags
- [ ] 7.2 Add `-p "$(cat PROMPT.md)"` for prompt passing
- [ ] 7.3 Add `--output-format stream-json` flag
- [ ] 7.4 Add `--allowedTools` flag with read-only tool restrictions
- [ ] 7.5 Add `--append-system-prompt-file context/system-instructions.txt` flag
- [ ] 7.6 Make command configurable via agent_command config
- [ ] 7.7 Add command builder unit tests

## 8. Process Execution and Lifecycle

- [ ] 8.1 Implement exec.CommandContext creation with timeout context
- [ ] 8.2 Configure process working directory to workspace root
- [ ] 8.3 Configure process environment variables from context bundle
- [ ] 8.4 Configure process group with `SysProcAttr{Setpgid: true}`
- [ ] 8.5 Implement graceful shutdown with SIGINT to process group
- [ ] 8.6 Set cmd.Cancel to send SIGINT to negative PID (process group)
- [ ] 8.7 Set cmd.WaitDelay to 30 seconds before SIGKILL
- [ ] 8.8 Implement stdout/stderr capture to `output/agent.log`
- [ ] 8.9 Implement log file creation before process start
- [ ] 8.10 Add output flushing logic to prevent data loss
- [ ] 8.11 Implement process start with error handling
- [ ] 8.12 Implement process wait with error handling
- [ ] 8.13 Capture and record exit code
- [ ] 8.14 Add process execution integration tests (with dummy script)

## 9. State Management and Lifecycle Tracking

- [ ] 9.1 Implement in-memory state tracker for active agents
- [ ] 9.2 Implement state transition logic: created → starting → running → final states
- [ ] 9.3 Log each state transition with timestamp and incident ID
- [ ] 9.4 Implement GetStatus method to query current agent status
- [ ] 9.5 Return status with start time, duration, and workspace path
- [ ] 9.6 Implement Cancel method to trigger graceful shutdown
- [ ] 9.7 Handle concurrent access to state map with mutex
- [ ] 9.8 Add state management unit tests

## 10. Exit Code Interpretation

- [ ] 10.1 Implement exit code capture from process
- [ ] 10.2 Map exit code 0 to success status
- [ ] 10.3 Map non-zero exit codes to failed status with code recorded
- [ ] 10.4 Detect signal-based termination (SIGTERM, SIGKILL)
- [ ] 10.5 Map timeout context to timeout status
- [ ] 10.6 Map cancellation context to cancelled status
- [ ] 10.7 Add exit code handling unit tests

## 11. Artifact Collection

- [ ] 11.1 Implement artifact scanning in `output/artifacts/` directory
- [ ] 11.2 Catalog all artifact files with absolute paths
- [ ] 11.3 Validate artifacts are within workspace boundary (path traversal check)
- [ ] 11.4 Reject files outside workspace with error
- [ ] 11.5 Return artifact list in AgentResult
- [ ] 11.6 Add artifact collection unit tests with mock files

## 12. Error Handling and Cleanup

- [ ] 12.1 Implement comprehensive error wrapping with context
- [ ] 12.2 Add defer statement in RunAgent to guarantee cleanup
- [ ] 12.3 Ensure cleanup runs on all exit paths (success, failure, panic)
- [ ] 12.4 Log cleanup errors without overriding original error
- [ ] 12.5 Preserve workspace on all error types for debugging
- [ ] 12.6 Implement workspace retention policy enforcement
- [ ] 12.7 Add error handling unit tests for all failure modes

## 13. Read-Only Enforcement

- [ ] 13.1 Create read-only kubeconfig with ServiceAccount (get, list, watch verbs only)
- [ ] 13.2 Document RBAC setup requirements for read-only ServiceAccount
- [ ] 13.3 Implement kubeconfig path configuration and validation
- [ ] 13.4 Ensure no write-capable credentials in workspace
- [ ] 13.5 Add `--allowedTools` flag with read-only restrictions to command
- [ ] 13.6 Write system-instructions.txt with explicit read-only enforcement text
- [ ] 13.7 Add CLAUDE_READ_ONLY_MODE=true to environment
- [ ] 13.8 Add read-only enforcement verification tests

## 14. Observability - Metrics

- [ ] 14.1 Define Prometheus metrics for agent runtime
- [ ] 14.2 Implement `agent_runtime_invocations_total` counter with cluster and status labels
- [ ] 14.3 Implement `agent_runtime_duration_seconds` histogram with cluster and status labels
- [ ] 14.4 Implement `agent_runtime_active_agents` gauge with cluster label
- [ ] 14.5 Implement `agent_runtime_workspace_size_bytes` gauge with cluster label
- [ ] 14.6 Add `agent_runtime_errors_total` counter with cluster and error_type labels
- [ ] 14.7 Emit metrics at appropriate lifecycle points
- [ ] 14.8 Expose metrics endpoint for Prometheus scraping

## 15. Observability - Structured Logging

- [ ] 15.1 Configure structured JSON logging library
- [ ] 15.2 Implement log entry with standard fields: timestamp, level, component, incident_id, cluster, event
- [ ] 15.3 Log workspace creation with workspace path
- [ ] 15.4 Log agent start with command, working directory, PID
- [ ] 15.5 Log state transitions
- [ ] 15.6 Log agent completion with status and duration
- [ ] 15.7 Log errors with full context (command, environment sanitized)
- [ ] 15.8 Make log level configurable (info, debug, warn, error)
- [ ] 15.9 Add structured logging tests

## 16. Security and Validation

- [ ] 16.1 Implement incident ID validation (UUID format only)
- [ ] 16.2 Implement path construction with filepath.Join and filepath.Clean
- [ ] 16.3 Add workspace boundary validation for all file operations
- [ ] 16.4 Reject path traversal attempts with clear error
- [ ] 16.5 Set workspace permissions to 0700 on creation
- [ ] 16.6 Ensure API keys passed via environment variables, not files
- [ ] 16.7 Add security validation unit tests

## 17. Resource Limits

- [ ] 17.1 Implement timeout configuration (default 10 minutes)
- [ ] 17.2 Create context with timeout for agent execution
- [ ] 17.3 Handle context deadline exceeded with timeout status
- [ ] 17.4 Implement workspace size calculation
- [ ] 17.5 Emit workspace size metrics
- [ ] 17.6 Document alerting thresholds for resource usage

## 18. Integration with Event Processing Layer

- [ ] 18.1 Define integration interface between event processor and agent runtime
- [ ] 18.2 Implement AgentRuntime factory/constructor
- [ ] 18.3 Wire AgentRuntime into event processor on "Active" state
- [ ] 18.4 Pass Incident struct from event processor to AgentRuntime.RunAgent
- [ ] 18.5 Return AgentResult to event processor for reporting layer
- [ ] 18.6 Handle agent runtime errors in event processor
- [ ] 18.7 Add integration tests with mock event processor

## 19. Testing and Validation

- [ ] 19.1 Create test fixtures: sample incidents, events, logs
- [ ] 19.2 Create dummy agent script for testing (exit 0, write test file)
- [ ] 19.3 Write unit tests for all component interfaces
- [ ] 19.4 Write integration test: end-to-end workspace creation to cleanup
- [ ] 19.5 Write integration test: successful agent invocation with dummy script
- [ ] 19.6 Write integration test: agent timeout handling
- [ ] 19.7 Write integration test: agent cancellation
- [ ] 19.8 Write integration test: agent failure (non-zero exit)
- [ ] 19.9 Write integration test: artifact collection
- [ ] 19.10 Write integration test: concurrent workspace isolation
- [ ] 19.11 Verify metrics are emitted correctly in tests
- [ ] 19.12 Verify structured logs are emitted correctly in tests
- [ ] 19.13 Run go vet and go fmt on all code
- [ ] 19.14 Achieve 80%+ test coverage for agent-runtime package

## 20. Documentation

- [ ] 20.1 Document workspace structure in code comments
- [ ] 20.2 Document configuration options in config schema
- [ ] 20.3 Document RBAC setup requirements for read-only kubeconfig
- [ ] 20.4 Document skill installation and source directory setup
- [ ] 20.5 Document metrics and alerting recommendations
- [ ] 20.6 Add inline godoc comments for all exported types and functions
- [ ] 20.7 Create example configuration file in `configs/agent-runtime.yaml`

## 21. Optional Enhancements (Post-MVP)

- [ ] 21.1 Implement progressive output streaming with `stream-json` parsing
- [x] 21.2 Add support for multiple agent backends (Codex, Goose, etc.)
      **DONE**: Implemented in `agent-container/`. Supports Claude (default), Codex, Gemini.
      Goose disabled due to X11 dependency. See `agent-container/README.md`.
- [ ] 21.3 Implement skill versioning and update management
- [ ] 21.4 Add workspace GC service with configurable retention
- [ ] 21.5 Implement workspace compression for long-term retention
- [ ] 21.6 Add agent health checks before invocation
- [ ] 21.7 Implement retry logic for transient agent failures
- [ ] 21.8 Add distributed tracing support with OpenTelemetry

## 22. Agent Container Implementation (COMPLETED)

The following tasks were implemented in `agent-container/`:

- [x] 22.1 Docker container with kubectl, helm, and AI CLI tools
- [x] 22.2 Multi-agent support: Claude (default/sonnet), Codex, Gemini
- [x] 22.3 Built-in k8s-troubleshooter skill from https://github.com/randybias/k8s4agents
- [x] 22.4 Workspace isolation: `-w` flag required, prevents mounting source code
- [x] 22.5 Output capture: Timestamped logs in `<workspace>/output/`
- [x] 22.6 Run script `run-agent.sh` with full configuration options
- [x] 22.7 Makefile for build, test, and debug targets

**Reference:** `agent-container/README.md`
