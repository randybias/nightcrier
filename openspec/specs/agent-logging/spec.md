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

The system SHALL capture and persist Claude Code session archives in DEBUG mode using a single, known path.

#### Scenario: Session archive created in DEBUG mode
- **Given** an agent executes in DEBUG mode
- **And** the agent uses Claude Code CLI
- **When** the agent completes
- **Then** the system SHALL extract `~/.claude` from `/home/agent/.claude` in the container
- **And** the system SHALL create `{workspace}/logs/claude-session.tar.gz`
- **And** the archive SHALL contain complete session history and internal logs

#### Scenario: Session archive graceful handling
- **Given** a post-run hook attempts session extraction
- **When** the session directory doesn't exist
- **Then** the system SHALL NOT fail the incident
- **And** the system SHALL log a debug message
- **And** the incident SHALL complete successfully without logs

#### Scenario: Session archive not captured in production
- **Given** nightcrier is running in production mode
- **When** an agent executes
- **Then** the system SHALL NOT attempt to extract the session
- **And** no session archive SHALL be created

### Requirement: Agent Commands Extraction

The system SHALL extract and log all Bash commands executed by the agent during investigation in DEBUG mode.

#### Scenario: Commands extracted from session JSONL
- **Given** an agent executes in DEBUG mode
- **And** the agent uses Claude Code CLI
- **When** the agent completes and session files are extracted
- **Then** the system SHALL parse the session JSONL files
- **And** the system SHALL extract all Bash tool calls with their commands
- **And** the system SHALL write commands to `{workspace}/logs/agent-commands-executed.log`

#### Scenario: Commands log format
- **Given** commands are being extracted from session JSONL
- **When** the commands log file is generated
- **Then** each command SHALL be prefixed with `$ `
- **And** each command SHALL include its description as a comment (if available)
- **And** the file SHALL include a header with timestamp, incident ID, and session ID

#### Scenario: Commands log upload
- **Given** Azure storage is configured
- **And** commands were extracted in DEBUG mode
- **When** artifacts are uploaded
- **Then** `agent-commands-executed.log` SHALL be uploaded to `{incident-id}/logs/`
- **And** a SAS URL SHALL be generated for the commands log
- **And** the log SHALL appear in index.html file listing

#### Scenario: Commands extraction graceful handling
- **Given** session extraction completed
- **When** no JSONL files are found or jq parsing fails
- **Then** the system SHALL NOT fail the incident
- **And** the system SHALL log a debug message
- **And** the commands log SHALL be empty or not created

### Requirement: Post-Run Hooks Architecture

The system SHALL provide a simple, pluggable post-run hooks mechanism for extensibility with minimal code.

#### Scenario: Post-run hook execution
- **Given** an agent completes execution
- **When** post-run hooks are configured in run-agent.sh
- **Then** each hook function SHALL be called after agent exits
- **And** hooks SHALL return 0 on success or graceful failure
- **And** hooks MAY skip execution based on conditions (e.g., debug mode)

#### Scenario: Hook simplicity and maintainability
- **Given** post-run hooks are implemented
- **When** code is reviewed
- **Then** the implementation SHALL be minimal (< 30 lines per hook)
- **And** single-path logic SHALL be preferred over fallbacks
- **And** hooks SHALL use consistent "Post-run:" debug logging

#### Scenario: Hook extensibility
- **Given** the post-run hooks system is simple and working
- **When** new post-execution tasks are needed (cleanup, metrics, etc.)
- **Then** developers SHALL add new hook functions
- **And** each function SHALL have single responsibility
- **And** hooks SHALL be added to the execution list

### Requirement: Log Metadata in Results

The system SHALL include log file metadata in the incident result.

#### Scenario: Result includes log paths
- **Given** an agent execution completes with log capture in DEBUG mode
- **When** the incident.json is written
- **Then** it SHALL include log file paths for local reference
- **And** it SHALL include presigned URLs for log files (when available)

