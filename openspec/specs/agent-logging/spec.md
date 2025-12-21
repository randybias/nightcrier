# agent-logging Specification

## Purpose
TBD - created by archiving change capture-agent-logs. Update Purpose after archive.
## Requirements
### Requirement: Conditional Debug Mode Logging

The system SHALL only capture and persist agent logs when running in DEBUG mode.

#### Scenario: Log capture enabled in DEBUG mode
- **Given** nightcrier is running with `log_level: debug`
- **When** an agent is executed for an incident
- **Then** the system SHALL create a `{workspace}/logs/` directory
- **And** the system SHALL write complete stdout and stderr to log files
- **And** the system SHALL upload logs to cloud storage

#### Scenario: No log capture in production mode
- **Given** nightcrier is running with `log_level: info/warn/error`
- **When** an agent is executed for an incident
- **Then** the system SHALL NOT create log files
- **And** the system SHALL NOT persist logs to disk
- **And** the system SHALL NOT upload logs to storage
- **And** the executor SHALL use `io.Discard` for efficiency (zero overhead)

#### Scenario: Log files created in DEBUG mode
- **Given** an agent is executed in DEBUG mode
- **When** the agent produces output to stdout or stderr
- **Then** the output SHALL be written to `{workspace}/logs/agent-stdout.log`
- **And** the output SHALL be written to `{workspace}/logs/agent-stderr.log`
- **And** a combined log SHALL be written to `{workspace}/logs/agent-full.log`
- **And** extracted commands SHALL be written to `{workspace}/logs/agent-commands-executed.log`

#### Scenario: Timestamped combined log
- **Given** an agent produces interleaved stdout and stderr output in DEBUG mode
- **When** the output is written to the combined log file
- **Then** each line SHALL be prefixed with an ISO8601 timestamp
- **And** each line SHALL be prefixed with a stream indicator (STDOUT/STDERR)

#### Scenario: Real-time visibility maintained
- **Given** log capture is enabled in DEBUG mode
- **When** the agent produces output
- **Then** the output SHALL also be logged via slog for real-time visibility
- **And** log file writing SHALL NOT block agent execution

### Requirement: Log Storage Upload

The system SHALL upload agent logs to cloud storage when configured.

#### Scenario: Azure Blob Storage log upload
- **Given** Azure storage is configured
- **When** an incident's artifacts are saved
- **Then** log files SHALL be uploaded to `{incident-id}/logs/` path
- **And** SAS URLs SHALL be generated for each log file
- **And** the log URLs SHALL be included in the SaveResult

#### Scenario: Empty log handling
- **Given** a log file is empty (zero bytes)
- **When** artifacts are saved to storage
- **Then** the empty log file SHALL NOT be uploaded
- **And** no error SHALL be raised for empty logs

#### Scenario: Filesystem storage log handling
- **Given** filesystem storage is configured (no Azure)
- **When** an incident's artifacts are saved
- **Then** log files SHALL be copied to the incident storage directory
- **And** local file paths SHALL be returned as log URLs

### Requirement: Log Notification Strategy

The system SHALL include logs in index.html but NOT in Slack notifications (keep Slack focused on investigation results).

#### Scenario: Index.html displays logs in DEBUG mode
- **Given** logs were captured in DEBUG mode
- **When** index.html is generated
- **Then** the HTML SHALL include links to all log files
- **And** each log SHALL be labeled "(DEBUG mode only)"
- **And** logs SHALL appear at the end of the file list

#### Scenario: Index.html clean in production mode
- **Given** no logs were captured (production mode)
- **When** index.html is generated
- **Then** the HTML SHALL NOT include log links
- **And** only core artifacts SHALL be displayed (report, incident, permissions)

#### Scenario: Slack notification stays clean
- **Given** an incident has log files (DEBUG mode)
- **When** a Slack notification is sent
- **Then** the notification SHALL NOT include log links
- **And** the notification SHALL focus on investigation results
- **And** operators access logs via index.html link

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

#### Scenario: Goose-specific session extraction
- **Given** an agent executes in DEBUG mode
- **And** the agent is Goose (`AGENT_CLI=goose`)
- **When** `runners/goose-post.sh` executes
- **Then** the system SHALL extract `/home/agent/.config/goose` from the container
- **And** the archive SHALL contain Goose's SQLite session database (`sessions.db`)

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

#### Scenario: Commands extracted from Goose session
- **Given** an agent executes in DEBUG mode
- **And** the agent is Goose (`AGENT_CLI=goose`)
- **When** the agent completes
- **Then** `runners/goose-post.sh` SHALL create a minimal commands log
- **And** the log SHALL note that Goose uses SQLite database storage
- **And** if `sqlite3` is available, it SHALL extract basic session metadata
- **And** it SHALL write the log to `{workspace}/logs/agent-commands-executed.log`

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

### Requirement: Log Metadata in Results

The system SHALL include log file metadata in the incident result.

#### Scenario: Result includes log paths
- **Given** an agent execution completes with log capture in DEBUG mode
- **When** the incident.json is written
- **Then** it SHALL include log file paths for local reference
- **And** it SHALL include presigned URLs for log files (when available)

