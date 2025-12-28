# agent-container Spec Delta

## REMOVED Requirements

### Requirement: Workspace Isolation (Docker-specific)
The Docker-specific workspace mounting requirements are removed as Docker execution is eliminated.

#### Scenario: Workspace mounting removal
- **WHEN** the refactor is complete
- **THEN** the Docker volume mount logic in run-agent.sh SHALL be removed
- **AND** workspace isolation SHALL be handled by K8s ConfigMap/Secret mounts instead

### Requirement: Modular Agent Runners (Shell Scripts)
The shell-based runner scripts are removed in favor of a unified container entrypoint.

#### Scenario: Runner script removal
- **WHEN** the refactor is complete
- **THEN** `agent-container/run-agent.sh` SHALL be deleted
- **AND** `agent-container/runners/*.sh` SHALL be deleted
- **AND** agent execution logic SHALL be handled by the container entrypoint

### Requirement: Agent-Agnostic Post-Run Hooks (Docker cp)
The `docker cp` based post-run extraction is removed as containers are stateless.

#### Scenario: Post-run hook removal
- **WHEN** the refactor is complete
- **THEN** `runners/*-post.sh` scripts SHALL be deleted
- **AND** session extraction SHALL be handled by container teardown upload to Object Store

## MODIFIED Requirements

### Requirement: Multi-Agent Container
The container SHALL support multiple AI CLI agents but execution is now K8s-native.

#### Scenario: Container image naming
- **WHEN** building the agent container
- **THEN** the image SHALL be named `nc-agent-runner`
- **AND** the image includes kubectl 1.31, helm 3.x, and search tools (ripgrep, fd, fzf)
- **AND** the image includes Claude Code, OpenAI Codex, Google Gemini, and Goose CLIs
- **AND** the image includes jq and sqlite3 for session data extraction
- **AND** the image includes the k8s-troubleshooter skill baked in
- **AND** the image uses `entrypoint.sh` as the container entrypoint

#### Scenario: Agent selection via environment
- **WHEN** a K8s Job is created with `AGENT_CLI=claude`
- **THEN** the container entrypoint invokes the Claude Code CLI
- **WHEN** a K8s Job is created with `AGENT_CLI=codex`
- **THEN** the container entrypoint invokes the OpenAI Codex CLI
- **WHEN** a K8s Job is created with `AGENT_CLI=gemini`
- **THEN** the container entrypoint invokes the Google Gemini CLI
- **WHEN** a K8s Job is created with `AGENT_CLI=goose`
- **THEN** the container entrypoint invokes the Goose CLI

### Requirement: Output Capture
The container SHALL upload outputs to Object Store instead of local file capture.

#### Scenario: Output upload
- **WHEN** an agent invocation completes (success or failure)
- **THEN** the container SHALL upload `/home/agent/output/report.md` to `OUTPUT_URL_REPORT`
- **AND** the container SHALL upload `/home/agent/logs/agent.log` to `OUTPUT_URL_LOG`
- **AND** uploads SHALL use HTTP PUT with presigned URLs

#### Scenario: Session archive upload
- **WHEN** an agent invocation completes
- **THEN** the container SHALL archive the session directory to `/tmp/session.tar.gz`
- **AND** the container SHALL upload the archive to `OUTPUT_URL_SESSION`
- **AND** the session directory location SHALL be agent-specific

#### Scenario: Result metadata upload
- **WHEN** an agent invocation completes
- **THEN** the container SHALL create `/tmp/result.json` with exit code and metadata
- **AND** the container SHALL upload the result to `OUTPUT_URL_RESULT`

#### Scenario: Teardown via trap
- **WHEN** the container receives SIGTERM or exits normally
- **THEN** the teardown function SHALL run via bash trap
- **AND** all outputs SHALL be uploaded before container termination
- **AND** upload failures SHALL NOT prevent other uploads from attempting

### Requirement: API Key Authentication
API keys SHALL be injected via K8s Secrets as environment variables.

#### Scenario: Claude authentication via Secret
- **WHEN** a K8s Job is created for Claude
- **THEN** `ANTHROPIC_API_KEY` SHALL be injected from Secret `ai-api-keys`
- **AND** the key SHALL be available as an environment variable

#### Scenario: Codex authentication via Secret
- **WHEN** a K8s Job is created for Codex
- **THEN** `OPENAI_API_KEY` SHALL be injected from Secret `ai-api-keys`
- **AND** the entrypoint SHALL perform `codex login --with-api-key` before execution

#### Scenario: Gemini authentication via Secret
- **WHEN** a K8s Job is created for Gemini
- **THEN** `GEMINI_API_KEY` SHALL be injected from Secret `ai-api-keys`

### Requirement: Built-in Skills
Skills SHALL be baked into the container image with agent-specific symlinks.

#### Scenario: Skill location
- **WHEN** the container starts
- **THEN** skills SHALL be available at `/home/agent/skills/`
- **AND** the entrypoint SHALL create symlinks to agent-specific locations

#### Scenario: Claude skill symlink
- **WHEN** `AGENT_CLI=claude`
- **THEN** the entrypoint SHALL create symlink `~/.claude/skills` pointing to `/home/agent/skills`

#### Scenario: Codex skill symlink
- **WHEN** `AGENT_CLI=codex`
- **THEN** the entrypoint SHALL create symlink `~/.codex/skills` pointing to `/home/agent/skills`

#### Scenario: Goose skill symlinks
- **WHEN** `AGENT_CLI=goose`
- **THEN** the entrypoint SHALL create symlink `~/.config/agents/skills` pointing to `/home/agent/skills`
- **AND** the entrypoint SHALL create symlink `~/.config/goose/skills` pointing to `/home/agent/skills`
- **AND** Goose searches multiple locations with priority ordering

### Requirement: Configurable Execution
Execution parameters SHALL be configured via K8s Job spec.

#### Scenario: Timeout configuration
- **WHEN** a K8s Job is created
- **THEN** `spec.activeDeadlineSeconds` SHALL control the job timeout
- **AND** the default timeout SHALL be 600 seconds
- **AND** SIGTERM SHALL be sent before SIGKILL to allow trap execution

#### Scenario: Resource limits
- **WHEN** a K8s Job is created
- **THEN** `resources.limits.memory` SHALL default to 2Gi
- **AND** `resources.limits.cpu` SHALL default to 1
- **AND** `resources.requests.memory` SHALL default to 512Mi

### Requirement: Context Preloading
Context SHALL be delivered via ConfigMap mounts instead of shell script preloading.

#### Scenario: Incident context delivery
- **WHEN** a K8s Job is created
- **THEN** `incident.json` SHALL be mounted from ConfigMap at `/home/agent/incident.json`
- **AND** the file SHALL be read-only

#### Scenario: Permissions context delivery
- **WHEN** a K8s Job is created
- **THEN** `permissions.json` SHALL be mounted from ConfigMap at `/home/agent/incident_cluster_permissions.json`
- **AND** the file SHALL be read-only

#### Scenario: System prompt delivery
- **WHEN** a K8s Job is created
- **THEN** `system-prompt.md` SHALL be mounted from ConfigMap at `/home/agent/system-prompt.md`
- **AND** the file SHALL be read-only
- **AND** the agent SHALL use this file via append-system-prompt-file or equivalent

#### Scenario: Kubeconfig delivery
- **WHEN** a K8s Job is created
- **THEN** kubeconfig SHALL be mounted from Secret at `/home/agent/.kube/config`
- **AND** the file SHALL be read-only
- **AND** the kubeconfig SHALL use a read-only ServiceAccount with TTL

## ADDED Requirements

### Requirement: Unified Container Entrypoint
The container SHALL have a unified entrypoint that handles all agent types.

#### Scenario: Entrypoint structure
- **WHEN** the container starts
- **THEN** `entrypoint.sh` SHALL be the container entrypoint
- **AND** it SHALL perform agent-specific setup based on `AGENT_CLI` environment variable
- **AND** it SHALL execute the agent and capture output
- **AND** it SHALL upload outputs on exit via trap

#### Scenario: Agent path setup
- **WHEN** the entrypoint runs
- **THEN** it SHALL create agent-specific directories
- **AND** it SHALL create symlinks for skills to agent-specific locations
- **AND** it SHALL export `SESSION_DIR` for teardown to know what to archive

#### Scenario: Agent invocation
- **WHEN** the entrypoint invokes the agent
- **THEN** it SHALL construct the CLI command based on `AGENT_CLI`
- **AND** it SHALL pass `PROMPT` as the investigation prompt
- **AND** it SHALL pass `LLM_MODEL` as the model selection
- **AND** stdout and stderr SHALL be captured to agent.log and displayed

### Requirement: Stateless Container Operation
The container SHALL operate without host volume mounts or persistent state.

#### Scenario: No host mounts
- **WHEN** inspecting the K8s Job spec
- **THEN** there SHALL be no hostPath volumes
- **AND** all volumes SHALL be ConfigMaps or Secrets

#### Scenario: Ephemeral filesystem
- **WHEN** the container runs
- **THEN** all writes SHALL be to ephemeral container filesystem
- **AND** outputs SHALL be uploaded to Object Store before termination
- **AND** container filesystem contents SHALL NOT be preserved after termination

#### Scenario: Idempotent execution
- **WHEN** a Job is created with the same incident ID
- **THEN** execution SHALL be independent of any previous runs
- **AND** there SHALL be no shared state between executions

### Requirement: In-Container Command Extraction
The container SHALL extract commands executed by the agent before uploading to Object Store.

#### Scenario: Command extraction from Claude
- **WHEN** `AGENT_CLI=claude` and the agent completes
- **THEN** the entrypoint SHALL parse JSONL files from `~/.claude/projects/*/`
- **AND** extract Bash tool invocations
- **AND** create `commands-executed.log` with each command prefixed by `$ `

#### Scenario: Command extraction from Codex
- **WHEN** `AGENT_CLI=codex` and the agent completes
- **THEN** the entrypoint SHALL parse JSONL files from `~/.codex/sessions/`
- **AND** extract shell_command function calls
- **AND** create `commands-executed.log` with each command prefixed by `$ `

#### Scenario: Command extraction from Gemini
- **WHEN** `AGENT_CLI=gemini` and the agent completes
- **THEN** the entrypoint SHALL parse JSON files from `~/.gemini/tmp/*/chats/session-*.json`
- **AND** extract bash tool invocations
- **AND** create `commands-executed.log` with each command prefixed by `$ `

#### Scenario: Command extraction from Goose
- **WHEN** `AGENT_CLI=goose` and the agent completes
- **THEN** the entrypoint SHALL query SQLite database at `~/.config/goose/sessions.db`
- **AND** extract shell commands from the messages table
- **AND** create `commands-executed.log` with each command prefixed by `$ `

#### Scenario: Command extraction upload
- **WHEN** commands have been extracted
- **THEN** the container SHALL upload `commands-executed.log` to `OUTPUT_URL_COMMANDS`
- **AND** extraction failures SHALL NOT prevent other uploads

### Requirement: Real-Time Logging
The container SHALL provide basic real-time logging during agent execution.

#### Scenario: Log capture and display
- **WHEN** the agent runs
- **THEN** stdout and stderr SHALL be displayed to the container console in real-time
- **AND** stdout and stderr SHALL be captured to `/home/agent/logs/agent.log`
- **AND** both capture and display SHALL use tee or equivalent

#### Scenario: Log streaming via kubectl
- **WHEN** the K8s Job is running
- **THEN** logs SHALL be streamable via `kubectl logs -f`
- **AND** users can observe agent progress in real-time
