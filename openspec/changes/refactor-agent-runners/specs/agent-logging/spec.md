# agent-logging Spec Delta

## MODIFIED Requirements

### Requirement: Claude Session Archive Capture
The system SHALL capture and persist AI agent session archives in DEBUG mode using agent-specific extraction scripts.

#### Scenario: Session archive created in DEBUG mode
- **Given** an agent executes in DEBUG mode
- **When** the agent completes
- **Then** the system SHALL invoke the agent-specific post-run script (`runners/{agent}-post.sh`)
- **And** the post-run script SHALL extract the agent's session directory from the container
- **And** the system SHALL create `{workspace}/logs/agent-session.tar.gz`
- **And** the archive SHALL contain the complete session history and internal logs for that agent

#### Scenario: Claude-specific session extraction
- **Given** an agent executes in DEBUG mode
- **And** the agent is Claude (`AGENT_CLI=claude`)
- **When** `runners/claude-post.sh` executes
- **Then** the system SHALL extract `/home/agent/.claude` from the container
- **And** the archive SHALL contain Claude's session JSONL files

#### Scenario: Codex-specific session extraction
- **Given** an agent executes in DEBUG mode
- **And** the agent is Codex (`AGENT_CLI=codex`)
- **When** `runners/codex-post.sh` executes
- **Then** the system SHALL extract `/home/agent/.codex` from the container
- **And** the archive SHALL contain Codex's session JSONL files

#### Scenario: Gemini-specific session extraction
- **Given** an agent executes in DEBUG mode
- **And** the agent is Gemini (`AGENT_CLI=gemini`)
- **When** `runners/gemini-post.sh` executes
- **Then** the system SHALL extract `/home/agent/.gemini` from the container
- **And** the archive SHALL contain Gemini's session JSON files in `tmp/*/chats/session-*.json` format

#### Scenario: Session archive graceful handling
- **Given** a post-run hook attempts session extraction
- **When** the session directory doesn't exist or extraction fails
- **Then** the system SHALL NOT fail the incident
- **And** the system SHALL log a debug message
- **And** the incident SHALL complete successfully without the session archive

#### Scenario: Session archive not captured in production
- **Given** nightcrier is running in production mode (DEBUG != true)
- **When** an agent executes
- **Then** the system SHALL NOT attempt to extract the session
- **And** no session archive SHALL be created

### Requirement: Agent Commands Extraction
The system SHALL extract and log all commands executed by the agent during investigation in DEBUG mode using agent-specific extraction logic.

#### Scenario: Commands extracted from Claude session JSONL
- **Given** an agent executes in DEBUG mode
- **And** the agent is Claude (`AGENT_CLI=claude`)
- **When** the agent completes and session files are extracted
- **Then** `runners/claude-post.sh` SHALL parse the session JSONL files
- **And** it SHALL extract all Bash tool calls with their commands
- **And** it SHALL write commands to `{workspace}/logs/agent-commands-executed.log`

#### Scenario: Commands extracted from Codex session
- **Given** an agent executes in DEBUG mode
- **And** the agent is Codex (`AGENT_CLI=codex`)
- **When** the agent completes
- **Then** `runners/codex-post.sh` SHALL parse the Codex session JSONL files
- **And** it SHALL extract all Bash tool calls with their commands
- **And** it SHALL write commands to `{workspace}/logs/agent-commands-executed.log`

#### Scenario: Commands extracted from Gemini session
- **Given** an agent executes in DEBUG mode
- **And** the agent is Gemini (`AGENT_CLI=gemini`)
- **When** the agent completes
- **Then** `runners/gemini-post.sh` SHALL parse the Gemini session JSON files
- **And** it SHALL extract all bash tool executions from the messages array
- **And** it SHALL write commands to `{workspace}/logs/agent-commands-executed.log`

#### Scenario: Commands log format
- **Given** commands are being extracted from any agent's session
- **When** the commands log file is generated
- **Then** the file SHALL include a header with:
  - Agent type (`# Agent: {agent_cli}`)
  - Timestamp (`# Generated: {ISO8601}`)
  - Incident ID (`# Incident: {incident_id}`)
- **And** each command SHALL be prefixed with `$ `
- **And** each command MAY include its description as a comment (if available from session data)

#### Scenario: Commands extraction graceful handling
- **Given** session extraction completed
- **When** command parsing fails or no commands are found
- **Then** the system SHALL NOT fail the incident
- **And** the system SHALL log a debug message
- **And** the commands log SHALL be empty or not created

### Requirement: Post-Run Hooks Architecture
The system SHALL provide a modular, pluggable post-run hooks mechanism that dispatches to agent-specific scripts.

#### Scenario: Post-run hook execution
- **Given** an agent completes execution
- **When** post-run hooks are invoked in run-agent.sh
- **Then** the orchestrator SHALL check for `runners/${AGENT_CLI}-post.sh`
- **And** if the script exists, it SHALL be sourced and executed
- **And** if the script does not exist, execution SHALL continue without error

#### Scenario: Hook simplicity and maintainability
- **Given** post-run hooks are implemented as separate scripts per agent
- **When** code is reviewed
- **Then** each agent's post-run script SHALL be self-contained
- **And** scripts SHALL be under 100 lines each
- **And** scripts SHALL use consistent "Post-run:" debug logging

#### Scenario: Hook extensibility
- **Given** the modular post-run hooks system is in place
- **When** a new AI agent is added
- **Then** developers SHALL create a new `runners/{agent}-post.sh` script
- **And** the main orchestrator SHALL automatically dispatch to it without code changes
- **And** the new script SHALL follow the standardized artifact naming conventions
