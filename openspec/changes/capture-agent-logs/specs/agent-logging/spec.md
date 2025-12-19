# Agent Logging Specification

## ADDED Requirements

### Requirement: Agent Log Capture

The system SHALL capture complete stdout and stderr from agent container executions and persist them to the workspace.

#### Scenario: Log files created during execution
- **Given** an agent is executed for an incident
- **When** the agent produces output to stdout or stderr
- **Then** the output SHALL be written to `{workspace}/logs/agent-stdout.log`
- **And** the output SHALL be written to `{workspace}/logs/agent-stderr.log`
- **And** a combined log SHALL be written to `{workspace}/logs/agent-full.log`

#### Scenario: Timestamped combined log
- **Given** an agent produces interleaved stdout and stderr output
- **When** the output is written to the combined log file
- **Then** each line SHALL be prefixed with an ISO8601 timestamp
- **And** each line SHALL be prefixed with a stream indicator (STDOUT/STDERR)

#### Scenario: Real-time visibility maintained
- **Given** log capture is enabled
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

### Requirement: Log Notification

The system SHALL include log URLs in incident notifications when available.

#### Scenario: Slack notification with logs
- **Given** log files were captured and uploaded
- **When** a Slack notification is sent for an incident
- **Then** the notification SHALL include a link to view the logs
- **And** the link SHALL point to the combined log file URL

#### Scenario: Notification without logs
- **Given** no log files were captured (logs are optional)
- **When** a Slack notification is sent for an incident
- **Then** the notification SHALL be sent without log links
- **And** no error SHALL be raised for missing logs

### Requirement: Log Metadata in Results

The system SHALL include log file metadata in the incident result.

#### Scenario: Result includes log paths
- **Given** an agent execution completes with log capture
- **When** the result.json is written
- **Then** it SHALL include a `log_paths` field with local file paths
- **And** it SHALL include a `log_urls` field with presigned URLs (when available)
