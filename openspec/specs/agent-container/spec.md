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

#### Scenario: Agent selection
- **WHEN** invoking run-agent.sh with `-a claude`
- **THEN** the Claude Code CLI is used with the sonnet model by default
- **WHEN** invoking run-agent.sh with `-a codex`
- **THEN** the OpenAI Codex CLI is used with proper authentication
- **WHEN** invoking run-agent.sh with `-a gemini`
- **THEN** the Google Gemini CLI is used

#### Scenario: Default agent
- **WHEN** invoking run-agent.sh without the `-a` flag
- **THEN** Claude Code is used as the default agent

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

