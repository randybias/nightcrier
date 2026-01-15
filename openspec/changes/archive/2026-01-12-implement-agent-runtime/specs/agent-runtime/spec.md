## ADDED Requirements

### Requirement: Workspace Isolation
The runner SHALL create a unique, isolated directory (workspace) for each incident investigation with a standardized structure.

#### Scenario: Workspace creation
- **WHEN** an incident is accepted for processing
- **THEN** a directory named `incident-{uuid}` is created under the workspace root
- **AND** the directory contains subdirectories: `.claude/`, `context/`, and `output/`
- **AND** all agent artifacts are stored within this workspace

#### Scenario: Workspace uniqueness
- **WHEN** multiple incidents are processed concurrently
- **THEN** each incident gets its own isolated workspace directory
- **AND** workspaces do not share files or state

#### Scenario: Workspace cleanup
- **WHEN** an agent invocation completes (success or failure)
- **THEN** the workspace is preserved for the reporting layer
- **AND** cleanup occurs based on the configured retention policy (e.g., 72 hours)

### Requirement: Skill Loading
The runner SHALL load the k8s-troubleshooter skill into each workspace to enable Kubernetes diagnostics.

#### Scenario: Skill installation
- **WHEN** a workspace is created
- **THEN** the k8s-troubleshooter skill is copied to `.claude/skills/k8s-troubleshooter/`
- **AND** the skill includes all required files (SKILL.md, references, scripts)

#### Scenario: Skill isolation
- **WHEN** skills are loaded into a workspace
- **THEN** each workspace contains its own copy of the skill
- **AND** skill updates do not affect running agents in other workspaces

### Requirement: Context Bundle Creation
The runner SHALL construct a context bundle containing all information needed for the agent to perform triage.

#### Scenario: Context file generation
- **WHEN** preparing to invoke the agent
- **THEN** the runner creates files in the `context/` directory:
  - `incident.json` with structured incident metadata
  - `event.json` with the raw kubernetes-mcp-server event payload
  - `logs.txt` with enriched log excerpts
  - `cluster-info.json` with cluster name, namespace, and involved resources
  - `system-instructions.txt` with read-only enforcement instructions

#### Scenario: Context environment variables
- **WHEN** invoking the agent
- **THEN** the runner sets environment variables:
  - `INCIDENT_ID` with the unique incident identifier
  - `INCIDENT_WORKSPACE` with the absolute path to the workspace
  - `KUBERNETES_CLUSTER` with the cluster name
  - `KUBERNETES_NAMESPACE` with the affected namespace
  - `ANTHROPIC_API_KEY` or `CLAUDE_API_KEY` for authentication
  - `CLAUDE_READ_ONLY_MODE=true` as a read-only marker

### Requirement: Prompt Construction
The runner SHALL generate a structured prompt that guides the agent to perform read-only triage.

#### Scenario: Prompt generation
- **WHEN** preparing agent invocation
- **THEN** the runner creates a `PROMPT.md` file containing:
  - Incident summary and severity
  - Reference to context files
  - Explicit read-only instructions
  - Expected outputs and artifact locations

#### Scenario: Prompt customization
- **WHEN** different incident severities are processed
- **THEN** the prompt emphasizes urgency for high-severity incidents
- **AND** includes appropriate triage depth instructions

### Requirement: Headless Agent Invocation
The runner SHALL invoke the AI agent via CLI command in headless/non-interactive mode using `exec.CommandContext`.

#### Scenario: Command construction
- **WHEN** the workspace and context are ready
- **THEN** the runner constructs a command like:
  `claude -p "$(cat PROMPT.md)" --output-format stream-json --allowedTools "Read,Grep,Glob,Bash(kubectl:get*,kubectl:describe*,kubectl:logs*)" --append-system-prompt-file context/system-instructions.txt`

#### Scenario: Process group management
- **WHEN** spawning the agent process
- **THEN** the runner configures the process with `Setpgid: true` for process group isolation
- **AND** signals are sent to the entire process group to prevent orphaned children

#### Scenario: Working directory
- **WHEN** executing the agent command
- **THEN** the working directory is set to the workspace root
- **AND** the agent has access to all context files via relative paths

### Requirement: Read-Only Enforcement
The runner SHALL configure the agent environment to restrict capabilities to read-only Kubernetes operations through multiple enforcement layers.

#### Scenario: RBAC enforcement
- **WHEN** the agent is invoked
- **THEN** the kubeconfig provided has a ServiceAccount limited to get, list, and watch verbs
- **AND** no create, update, delete, or patch permissions are granted

#### Scenario: Tool restriction
- **WHEN** invoking the agent with `--allowedTools` flag
- **THEN** only Read, Grep, and Glob file tools are permitted
- **AND** Bash tool is restricted to read-only kubectl patterns: `kubectl:get*,kubectl:describe*,kubectl:logs*`
- **AND** Write, Edit, and unrestricted Bash access are prohibited

#### Scenario: System prompt enforcement
- **WHEN** the system instructions file is created
- **THEN** it contains explicit text: "You are in READ-ONLY triage mode. Do NOT attempt to modify cluster state, apply changes, or run remediation commands. Your role is ANALYSIS ONLY."

#### Scenario: Environment marker
- **WHEN** environment variables are set
- **THEN** `CLAUDE_READ_ONLY_MODE=true` is included as a runtime marker
- **AND** this can be checked by skills or custom scripts

### Requirement: Process Lifecycle Management
The runner SHALL manage the agent process lifecycle with timeout, cancellation, and graceful shutdown capabilities.

#### Scenario: Timeout enforcement
- **WHEN** an agent invocation starts
- **THEN** a context with timeout (e.g., 10 minutes) is created
- **AND** if the timeout expires, the process is terminated
- **AND** the status is set to `timeout`

#### Scenario: Graceful shutdown
- **WHEN** the context is cancelled or times out
- **THEN** the runner sends SIGINT to the process group
- **AND** waits for the configured WaitDelay (e.g., 30 seconds)
- **AND** sends SIGKILL if the process has not exited

#### Scenario: State transitions
- **WHEN** an agent invocation progresses
- **THEN** the status transitions through states: `created` → `starting` → `running` → [`success` | `failed` | `timeout` | `cancelled`]
- **AND** each transition is logged with timestamp and incident ID

#### Scenario: Manual cancellation
- **WHEN** a cancellation request is received for an active agent
- **THEN** the context is cancelled triggering graceful shutdown
- **AND** the final status is set to `cancelled`

### Requirement: Output Capture
The runner SHALL capture all agent stdout and stderr output for debugging and audit purposes.

#### Scenario: Output logging
- **WHEN** the agent process is running
- **THEN** stdout and stderr are redirected to `output/agent.log`
- **AND** the log file is created before process start
- **AND** output is flushed periodically to prevent loss

#### Scenario: Streaming output support
- **WHEN** `--output-format stream-json` is used
- **THEN** the runner can optionally parse JSON messages in real-time
- **AND** progressive status updates are available

### Requirement: Exit Code Handling
The runner SHALL capture and interpret the agent process exit code to determine the invocation outcome.

#### Scenario: Success detection
- **WHEN** the agent process exits with code 0
- **THEN** the status is set to `success`
- **AND** artifacts are collected from the `output/` directory

#### Scenario: Failure detection
- **WHEN** the agent process exits with a non-zero code
- **THEN** the status is set to `failed`
- **AND** the exit code is recorded in the result
- **AND** the workspace is preserved for debugging

#### Scenario: Signal-based termination
- **WHEN** the agent process is terminated by a signal (SIGTERM, SIGKILL)
- **THEN** the exit code reflects the signal
- **AND** the status is set to `timeout` or `cancelled` based on context

### Requirement: Artifact Collection
The runner SHALL collect all files generated by the agent in the workspace for downstream consumption.

#### Scenario: Artifact discovery
- **WHEN** an agent invocation completes
- **THEN** the runner scans the `output/artifacts/` directory
- **AND** all files are cataloged with paths in the AgentResult

#### Scenario: Artifact validation
- **WHEN** collecting artifacts
- **THEN** the runner validates that files are within the workspace boundary
- **AND** path traversal attempts are rejected

### Requirement: Error Handling and Recovery
The runner SHALL handle errors at each lifecycle stage and ensure workspace cleanup on all exit paths.

#### Scenario: Workspace creation failure
- **WHEN** workspace creation fails (disk full, permissions error)
- **THEN** an error is returned immediately
- **AND** no partial workspace is left behind

#### Scenario: Agent execution failure
- **WHEN** the agent command fails to start
- **THEN** the error is captured with full context (command, working directory, environment)
- **AND** the workspace is preserved with status `failed`

#### Scenario: Cleanup guarantee
- **WHEN** any error occurs during invocation
- **THEN** cleanup is attempted via `defer` statement
- **AND** cleanup errors are logged but do not override the original error

### Requirement: Configuration Management
The runner SHALL load configuration from environment variables and config files to control agent runtime behavior.

#### Scenario: Configuration precedence
- **WHEN** both config file and environment variables are present
- **THEN** environment variables override config file values
- **AND** defaults are used for unspecified values

#### Scenario: Agent command configuration
- **WHEN** the `agent_command` is configured as "claude"
- **THEN** the runner uses the `claude` binary from PATH
- **AND** the command can be overridden via `AGENT_RUNTIME_COMMAND` environment variable

#### Scenario: Workspace root configuration
- **WHEN** `workspace_root` is set to "/var/lib/event-runner/workspaces"
- **THEN** all workspaces are created under this directory
- **AND** the directory is created if it does not exist

### Requirement: Observability
The runner SHALL emit metrics and structured logs to enable monitoring and debugging of agent invocations.

#### Scenario: Metrics emission
- **WHEN** agent invocations occur
- **THEN** the runner emits metrics:
  - `agent_runtime_invocations_total{cluster, status}` counter
  - `agent_runtime_duration_seconds{cluster, status}` histogram
  - `agent_runtime_active_agents{cluster}` gauge
  - `agent_runtime_workspace_size_bytes{cluster}` gauge

#### Scenario: Structured logging
- **WHEN** lifecycle events occur
- **THEN** the runner logs in JSON format with fields:
  - `timestamp`, `level`, `component`, `incident_id`, `cluster`, `event`, `workspace`, `pid`
- **AND** log level is configurable (info, debug, warn, error)

#### Scenario: Error logging
- **WHEN** errors occur
- **THEN** errors are logged with full context:
  - Error message and type
  - Stack trace if available
  - Incident ID and cluster
  - Command and environment (sanitized)

### Requirement: Security and Isolation
The runner SHALL implement security controls to protect the host system and prevent unauthorized access.

#### Scenario: Workspace permissions
- **WHEN** a workspace is created
- **THEN** permissions are set to owner-only read/write (0700)
- **AND** the workspace is owned by the event-runner process user

#### Scenario: Credential isolation
- **WHEN** setting up the agent environment
- **THEN** only read-only kubeconfig is provided
- **AND** no write-capable credentials are placed in the workspace
- **AND** API keys are passed via environment variables, not files

#### Scenario: Path validation
- **WHEN** constructing workspace paths
- **THEN** incident IDs are validated to be UUIDs only
- **AND** `filepath.Join` and `filepath.Clean` are used for path construction
- **AND** all operations verify paths are under workspace_root

### Requirement: Resource Limits
The runner SHALL enforce resource limits to prevent agent invocations from exhausting host resources.

#### Scenario: Execution timeout
- **WHEN** an agent is invoked
- **THEN** a maximum duration limit (e.g., 10 minutes) is enforced via context timeout
- **AND** exceeding the limit triggers graceful shutdown and timeout status

#### Scenario: Workspace size monitoring
- **WHEN** workspaces are created and used
- **THEN** workspace disk usage is tracked via metrics
- **AND** alerts can be configured for excessive usage

### Requirement: Status Reporting
The runner SHALL provide real-time status information for agent invocations to upstream consumers.

#### Scenario: Status query
- **WHEN** a status request is made for an incident ID
- **THEN** the current agent status is returned (created, starting, running, success, failed, timeout, cancelled)
- **AND** the response includes start time, current duration, and workspace path

#### Scenario: Status persistence
- **WHEN** an agent invocation is in progress
- **THEN** the status is tracked in memory
- **AND** completed statuses are available for the retention period
