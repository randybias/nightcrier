## ADDED Requirements

### Requirement: Markdown Report Generation
The runner SHALL generate a summary report in Markdown format at the end of each investigation.

#### Scenario: Successful investigation report
- **WHEN** the agent completes its task successfully (exit code 0)
- **THEN** a `report.md` file is generated in the incident workspace
- **AND** it contains the incident ID, timestamp, severity, cluster name, and namespace
- **AND** it contains a summary section with one-paragraph executive overview
- **AND** it contains a findings section with detailed analysis
- **AND** it contains a recommendations section with suggested next steps
- **AND** it contains a metadata section with agent version, duration, and workspace path

#### Scenario: Failed investigation report
- **WHEN** the agent exits with a non-zero exit code
- **THEN** a `report.md` file is generated
- **AND** it includes the failure status, exit code, and any captured error output
- **AND** it contains metadata about the failure

#### Scenario: Timeout investigation report
- **WHEN** the agent times out before completing
- **THEN** a `report.md` file is generated
- **AND** it includes the timeout status and duration
- **AND** it contains any partial output captured before timeout

### Requirement: Report Template Structure
The runner SHALL use a consistent template structure for all reports with required sections.

#### Scenario: Required sections present
- **WHEN** generating any report
- **THEN** the report MUST include a header section with metadata
- **AND** it MUST include a summary section
- **AND** it MUST include a findings section
- **AND** it MUST include a recommendations section
- **AND** it MUST include a metadata footer

#### Scenario: Template rendering with special characters
- **WHEN** agent output contains Markdown special characters
- **THEN** the template MUST escape these characters to prevent formatting issues
- **AND** the rendered report displays the content correctly

### Requirement: Artifact Persistence
The runner SHALL save relevant artifacts (logs, command output) to the incident workspace on disk.

#### Scenario: Workspace directory structure
- **WHEN** an investigation begins
- **THEN** a directory named with the incident ID is created under the configured root path
- **AND** the directory path follows the pattern `/var/lib/event-runner/incidents/<incident-id>/`

#### Scenario: Core artifact files
- **WHEN** the agent completes
- **THEN** a `report.md` file is written to the workspace
- **AND** an `agent-output.log` file containing raw stdout/stderr is written
- **AND** an `agent-context.json` file with input context is written
- **AND** a `metadata.json` file with runner metadata is written

#### Scenario: Additional agent artifacts
- **WHEN** the agent produces additional output files
- **THEN** these are saved in an `artifacts/` subdirectory within the workspace
- **AND** original filenames are preserved

#### Scenario: Disk write failure
- **WHEN** writing a report file fails due to disk issues
- **THEN** the error is logged with full context
- **AND** the runner attempts to write a minimal `metadata.json` with error details
- **AND** the failure does not prevent Slack notification

### Requirement: Artifact Retention
The runner SHALL retain all artifacts for audit purposes with no automatic deletion.

#### Scenario: Audit trail preservation
- **WHEN** multiple investigations complete
- **THEN** all workspace directories are preserved
- **AND** no automatic cleanup or deletion occurs
- **AND** operators can review any past investigation

### Requirement: Slack Notification on Success
The runner SHALL send a notification to a configured Slack webhook upon successful completion of an investigation.

#### Scenario: Successful investigation notification
- **WHEN** the report is generated successfully
- **THEN** a POST request is sent to the Slack webhook URL
- **AND** the payload uses Block Kit format with message attachments
- **AND** the attachment color matches the incident severity (red/yellow/green)
- **AND** the message includes incident ID, severity, cluster, resource, and duration
- **AND** the message includes key findings (up to 3 bullet points)
- **AND** the message includes the filesystem path to the full report

#### Scenario: Severity color mapping
- **WHEN** sending a notification for a critical/high severity incident
- **THEN** the message attachment color is `#E01E5A` (red)
- **WHEN** sending a notification for a warning/medium severity incident
- **THEN** the message attachment color is `#ECB22E` (yellow)
- **WHEN** sending a notification for an info/low severity incident
- **THEN** the message attachment color is `#2EB67D` (green)

### Requirement: Slack Notification on Failure
The runner SHALL send a notification to Slack when an investigation fails or times out.

#### Scenario: Agent failure notification
- **WHEN** the agent exits with a non-zero exit code
- **THEN** a POST request is sent to the Slack webhook
- **AND** the message attachment color is `#611F69` (purple)
- **AND** the message indicates the failure status and exit code
- **AND** the message includes any captured error output
- **AND** the message includes the filesystem path to the failure report

#### Scenario: Agent timeout notification
- **WHEN** the agent exceeds the configured timeout
- **THEN** a POST request is sent to the Slack webhook
- **AND** the message indicates the timeout status
- **AND** the message includes the timeout duration

### Requirement: Slack Retry Logic
The runner SHALL implement retry logic with exponential backoff for transient Slack failures.

#### Scenario: Rate limit retry
- **WHEN** Slack responds with HTTP 429 (rate limited)
- **THEN** the runner waits 1 second and retries
- **AND** if it fails again, waits 2 seconds and retries
- **AND** if it fails again, waits 4 seconds and retries
- **AND** after 3 attempts, logs final failure and continues

#### Scenario: Server error retry
- **WHEN** Slack responds with HTTP 5xx (server error)
- **THEN** the runner retries with exponential backoff (1s, 2s, 4s)
- **AND** after 3 attempts, logs final failure and continues

#### Scenario: Network timeout retry
- **WHEN** the HTTP request times out
- **THEN** the runner retries with exponential backoff (1s, 2s, 4s)
- **AND** after 3 attempts, logs final failure and continues

#### Scenario: Client error no retry
- **WHEN** Slack responds with HTTP 4xx (client error)
- **THEN** the error is logged with the response details
- **AND** no retry is attempted (likely configuration issue)

#### Scenario: Success response
- **WHEN** Slack responds with HTTP 200-299
- **THEN** the response is logged as successful
- **AND** no retry is needed

### Requirement: Slack Configuration
The runner SHALL require a Slack webhook URL to be configured and SHALL validate it on startup.

#### Scenario: Missing webhook URL
- **WHEN** the runner starts without a SLACK_WEBHOOK_URL environment variable
- **THEN** the runner logs a fatal error
- **AND** exits with a non-zero status code

#### Scenario: Malformed webhook URL
- **WHEN** the SLACK_WEBHOOK_URL is not a valid HTTPS URL
- **THEN** the runner logs a fatal error
- **AND** exits with a non-zero status code

#### Scenario: Valid webhook URL
- **WHEN** the SLACK_WEBHOOK_URL is a valid HTTPS URL starting with https://hooks.slack.com/
- **THEN** the runner logs successful configuration
- **AND** continues startup

### Requirement: Report Root Directory Configuration
The runner SHALL support configurable report root directory with validation on startup.

#### Scenario: Default report directory
- **WHEN** no REPORT_ROOT_DIR environment variable is set
- **THEN** the runner uses `/var/lib/event-runner/incidents` as the default
- **AND** validates the directory is writable on startup

#### Scenario: Custom report directory
- **WHEN** REPORT_ROOT_DIR environment variable is set
- **THEN** the runner uses the specified directory
- **AND** creates the directory if it does not exist
- **AND** validates write permissions before processing incidents

#### Scenario: Non-writable report directory
- **WHEN** the report root directory is not writable
- **THEN** the runner logs a fatal error
- **AND** exits with a non-zero status code

### Requirement: Non-Blocking Notification
The runner SHALL treat Slack notifications as best-effort and SHALL NOT block incident processing on notification failures.

#### Scenario: Report generation continues on Slack failure
- **WHEN** report generation succeeds but Slack notification fails
- **THEN** the report is preserved on disk
- **AND** the failure is logged
- **AND** the runner continues processing other incidents

#### Scenario: Notification happens after persistence
- **WHEN** an investigation completes
- **THEN** the report and artifacts are written to disk first
- **AND** the Slack notification is attempted afterward
- **AND** disk persistence is the source of truth

### Requirement: Notification Timing
The runner SHALL send notifications immediately after report generation without batching or delays.

#### Scenario: Immediate notification
- **WHEN** a report is successfully generated
- **THEN** the Slack notification is sent within 1 second
- **AND** no artificial delays or batching occurs

### Requirement: Report Data Completeness
The runner SHALL capture all relevant investigation data for inclusion in reports and notifications.

#### Scenario: Capturing agent output
- **WHEN** the agent produces stdout and stderr output
- **THEN** both streams are captured separately
- **AND** combined into agent-output.log
- **AND** relevant portions are included in the report

#### Scenario: Capturing timing metadata
- **WHEN** an investigation completes
- **THEN** the start timestamp is recorded
- **AND** the end timestamp is recorded
- **AND** the total duration is calculated and included in metadata

#### Scenario: Capturing context
- **WHEN** the agent is invoked
- **THEN** the input context (incident details, prompts) is saved to agent-context.json
- **AND** this context is available for report generation
