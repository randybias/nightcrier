# agent-container Specification

## Purpose
TBD - created by archiving change implement-agent-container. Update Purpose after archive.
## Requirements
### Requirement: Multi-Agent Container
The system SHALL provide a Docker container capable of running multiple AI CLI agents for Kubernetes incident triage.

#### Scenario: Container build
- **WHEN** building the k8s-triage-agent container
- **THEN** the image includes kubectl 1.31, helm 3.x, and search tools (ripgrep, fd, fzf)
- **AND** the image includes Claude Code, OpenAI Codex, and Google Gemini CLIs
- **AND** the image includes the k8s-troubleshooter skill from k8s4agents
- **AND** the image includes the `runners/` directory with all sub-runner scripts

#### Scenario: Agent selection
- **WHEN** invoking run-agent.sh with `-a claude`
- **THEN** the Claude Code CLI is used via the `runners/claude.sh` sub-runner
- **WHEN** invoking run-agent.sh with `-a codex`
- **THEN** the OpenAI Codex CLI is used via the `runners/codex.sh` sub-runner
- **WHEN** invoking run-agent.sh with `-a gemini`
- **THEN** the Google Gemini CLI is used via the `runners/gemini.sh` sub-runner
- **WHEN** invoking run-agent.sh with `-a goose`
- **THEN** the Goose CLI is used via the `runners/goose.sh` sub-runner

#### Scenario: Default agent
- **WHEN** invoking run-agent.sh without the `-a` flag
- **THEN** Claude Code is used as the default agent via `runners/claude.sh`

### Requirement: Workspace Isolation
The container SHALL enforce workspace isolation to prevent access to unauthorized host directories.

#### Scenario: Required workspace flag
- **WHEN** invoking run-agent.sh without the `-w` flag
- **THEN** the script exits with an error message
- **AND** the message instructs the user to provide an incident workspace directory

#### Scenario: Workspace mounting
- **WHEN** invoking run-agent.sh with `-w /path/to/incident`
- **THEN** the incident directory is mounted at /workspace in the container
- **AND** an output subdirectory is created and mounted at /output
- **AND** no other host directories are accessible (except kubeconfig read-only)

#### Scenario: Kubeconfig access
- **WHEN** the container is started
- **THEN** the host kubeconfig is mounted read-only at /root/.kube/config
- **AND** the agent can execute kubectl commands against the configured cluster

### Requirement: Built-in Skills
The container SHALL include the k8s-troubleshooter skill for Kubernetes diagnostics.

#### Scenario: Skill availability
- **WHEN** the container starts
- **THEN** the k8s-troubleshooter skill is available at /skills/k8s-troubleshooter/
- **AND** the skill includes SKILL.md, references/, and scripts/ directories

#### Scenario: Claude skill access
- **WHEN** Claude Code runs in the container
- **THEN** the /k8s-troubleshooter slash command is available
- **AND** Claude can read /skills/k8s-troubleshooter/SKILL.md for guidance

### Requirement: Output Capture
The container SHALL capture all agent output to timestamped log files.

#### Scenario: Output logging
- **WHEN** an agent invocation completes
- **THEN** all stdout and stderr is captured to a log file
- **AND** the log file is named triage_<agent>_<timestamp>.log
- **AND** the log file is saved in the workspace output directory

#### Scenario: Real-time output
- **WHEN** an agent is running
- **THEN** output is displayed in real-time to the terminal
- **AND** simultaneously written to the log file via tee

#### Scenario: Agent-specific session archive
- **WHEN** an agent completes in DEBUG mode
- **THEN** the agent-specific post-run script SHALL extract session data
- **AND** the archive SHALL be named `agent-session.tar.gz` regardless of agent type
- **AND** the archive contents SHALL be agent-specific (`.claude/` for Claude, `.codex/` for Codex, etc.)

### Requirement: API Key Authentication
The container SHALL support API key authentication for each agent backend.

#### Scenario: Claude authentication
- **WHEN** ANTHROPIC_API_KEY is set in the environment
- **THEN** Claude Code uses this key for API authentication

#### Scenario: Codex authentication
- **WHEN** OPENAI_API_KEY is set in the environment
- **THEN** the script performs `codex login --with-api-key` before execution
- **AND** Codex uses this key for API authentication

#### Scenario: Gemini authentication
- **WHEN** GEMINI_API_KEY or GOOGLE_API_KEY is set in the environment
- **THEN** Gemini CLI uses this key for API authentication

#### Scenario: Missing API key
- **WHEN** the required API key for the selected agent is not set
- **THEN** the script exits with an error message indicating which key is required

### Requirement: Configurable Execution
The container SHALL support configurable execution parameters.

#### Scenario: Timeout configuration
- **WHEN** CONTAINER_TIMEOUT is set or --timeout flag is provided
- **THEN** the container execution is limited to the specified duration in seconds
- **AND** the default timeout is 600 seconds (10 minutes)

#### Scenario: Memory limit
- **WHEN** CONTAINER_MEMORY is set or --memory flag is provided
- **THEN** the container memory is limited to the specified amount
- **AND** the default memory limit is 2g

#### Scenario: Claude tool restrictions
- **WHEN** invoking Claude with `-t "Read,Grep,Glob,Bash"`
- **THEN** Claude is restricted to only those tools via --allowedTools flag

#### Scenario: Claude model selection
- **WHEN** invoking run-agent.sh with `-m opus`
- **THEN** Claude uses the opus model instead of the default sonnet

### Requirement: Agent Context File Integration
The system SHALL provide agent-specific context files for skill integration where native skill systems are unavailable.

#### Scenario: Gemini context file
- **WHEN** the container is built
- **THEN** a GEMINI.md file SHALL be created in /home/agent/
- **AND** it SHALL contain instructions for Kubernetes incident triage
- **AND** it SHALL reference the k8s-troubleshooter skill location and available scripts
- **AND** Gemini CLI SHALL automatically load this file as system context

#### Scenario: Context file hierarchy
- **WHEN** Gemini CLI starts in /home/agent
- **THEN** it SHALL load context from ~/.gemini/GEMINI.md (if exists)
- **AND** it SHALL load context from /home/agent/GEMINI.md (project level)
- **AND** all loaded context files SHALL be concatenated and provided to the model

#### Scenario: Goose context file
- **WHEN** the container is built
- **THEN** a .goosehints file SHALL be created in /home/agent/
- **AND** it SHALL contain instructions for Kubernetes incident triage
- **AND** it SHALL reference the k8s-troubleshooter skill location and available scripts
- **AND** Goose CLI SHALL automatically load this file as additional context

#### Scenario: Goose configuration
- **WHEN** the container is built
- **THEN** a config.yaml file SHALL be created in /home/agent/.config/goose/
- **AND** it SHALL be pre-configured with GOOSE_PROVIDER: openai
- **AND** it SHALL be pre-configured with GOOSE_MODEL: gpt-4.1
- **AND** the Goose runner SHALL set GOOSE_DISABLE_KEYRING=1 for headless operation

### Requirement: Modular Agent Runners
The system SHALL provide modular sub-runner scripts for each supported AI CLI agent.

#### Scenario: Sub-runner directory structure
- **WHEN** inspecting the agent-container directory
- **THEN** a `runners/` subdirectory SHALL exist
- **AND** it SHALL contain `common.sh` with shared functions
- **AND** it SHALL contain `{agent}.sh` for each supported agent (claude, codex, gemini, goose)
- **AND** it SHALL contain `{agent}-post.sh` for each agent's post-run hooks

#### Scenario: Sub-runner invocation
- **WHEN** run-agent.sh is invoked with `-a claude`
- **THEN** the script SHALL source `runners/claude.sh` to build the CLI command
- **AND** after execution, it SHALL source `runners/claude-post.sh` for artifact extraction
- **WHEN** run-agent.sh is invoked with `-a codex`
- **THEN** the script SHALL source `runners/codex.sh` to build the CLI command
- **AND** after execution, it SHALL source `runners/codex-post.sh` for artifact extraction

#### Scenario: Sub-runner contract
- **WHEN** a sub-runner script is sourced
- **THEN** it SHALL have access to standardized environment variables (AGENT_CLI, PROMPT, LLM_MODEL, AGENT_HOME, etc.)
- **AND** it SHALL output the complete agent CLI command string to stdout
- **AND** it SHALL NOT execute the command itself (orchestrator handles execution)

### Requirement: Agent-Agnostic Post-Run Hooks
The system SHALL dispatch post-run artifact extraction to agent-specific scripts.

#### Scenario: Post-run dispatch
- **WHEN** an agent execution completes
- **THEN** the orchestrator SHALL check for `runners/${AGENT_CLI}-post.sh`
- **AND** if the script exists, it SHALL be sourced for execution
- **AND** if the script does not exist, the orchestrator SHALL log a warning and continue

#### Scenario: Standardized artifact paths
- **WHEN** a post-run script extracts session artifacts in DEBUG mode
- **THEN** it SHALL create `{workspace}/logs/agent-session.tar.gz` for the session archive
- **AND** it SHALL create `{workspace}/logs/agent-commands-executed.log` for extracted commands
- **AND** the commands log SHALL use a standardized header format with agent name, timestamp, and incident ID

#### Scenario: Graceful failure handling
- **WHEN** a post-run script cannot extract session data (e.g., session directory missing)
- **THEN** it SHALL log a debug message indicating the failure
- **AND** it SHALL NOT cause the overall script to exit with non-zero code
- **AND** the incident SHALL complete successfully without the missing artifacts

### Requirement: Context Preloading
The agent runner SHALL preload incident context before agent execution to minimize redundant operations.

#### Scenario: Incident context preloading
- **WHEN** run-agent.sh is invoked with a workspace containing incident.json
- **THEN** the incident.json contents are read from the host
- **AND** the contents are wrapped in `<incident>` XML tags
- **AND** the tagged content is included in the agent's initial prompt

#### Scenario: Permissions preloading
- **WHEN** run-agent.sh is invoked with a workspace containing incident_cluster_permissions.json
- **THEN** the permissions file contents are read from the host
- **AND** the contents are wrapped in `<kubernetes_cluster_access_permissions>` XML tags
- **AND** the tagged content is included in the agent's initial prompt

#### Scenario: Baseline triage preloading
- **WHEN** run-agent.sh is invoked and skill triage script is available
- **THEN** the script is executed before agent starts
- **AND** the triage output is wrapped in `<initial_triage_report>` XML tags
- **AND** the tagged output is included in the agent's initial prompt
- **AND** execution is limited to 30 seconds timeout

#### Scenario: Graceful triage failure
- **WHEN** triage script execution fails or times out
- **THEN** a warning is logged indicating triage unavailable
- **AND** the agent proceeds without triage context
- **AND** the failure does not prevent agent execution

#### Scenario: Context size monitoring
- **WHEN** preloaded context exceeds 8,000 tokens (estimated)
- **THEN** a warning is logged indicating large context size
- **AND** triage output is truncated if total exceeds 10,000 tokens
- **AND** incident.json and permissions are never truncated

#### Scenario: Context injection location
- **WHEN** the agent command is constructed
- **THEN** preloaded context is inserted between system prompt and user prompt
- **AND** the system prompt remains generic (domain-agnostic)
- **AND** the preloaded context provides domain-specific data

#### Scenario: Audit trail accuracy
- **WHEN** preloaded context is assembled
- **THEN** the full preloaded context is appended to prompt-sent.md
- **AND** the audit trail shows the complete prompt sent to the agent
- **AND** this happens after preloading completes but before agent execution

