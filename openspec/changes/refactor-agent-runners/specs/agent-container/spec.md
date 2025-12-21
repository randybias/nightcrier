# agent-container Spec Delta

## ADDED Requirements

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

## MODIFIED Requirements

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
