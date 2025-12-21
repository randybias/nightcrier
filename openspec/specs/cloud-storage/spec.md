# cloud-storage Specification

## Purpose
TBD - created by archiving change add-cloud-incident-storage. Update Purpose after archive.
## Requirements
### Requirement: Cloud Storage Backend

The system SHALL support uploading incident artifacts to cloud storage as an alternative to local filesystem storage, with Azure Blob Storage as the initial implementation.

#### Scenario: Azure storage mode enabled
- **GIVEN** `AZURE_STORAGE_ACCOUNT` or `AZURE_STORAGE_CONNECTION_STRING` environment variable is configured
- **WHEN** the runner starts
- **THEN** Azure Blob Storage mode is activated
- **AND** local filesystem storage is not used for incident artifacts

#### Scenario: Filesystem storage mode (default)
- **GIVEN** no cloud storage environment variables are set
- **WHEN** the runner starts
- **THEN** local filesystem storage is used
- **AND** cloud storage is not attempted

#### Scenario: Storage interface extensibility
- **GIVEN** the storage subsystem uses an abstract interface
- **WHEN** a new storage backend is needed (e.g., S3, GCS)
- **THEN** only the new backend implementation is required
- **AND** no changes are needed to reporting or notification code

### Requirement: Azure Configuration

The system SHALL require valid Azure credentials and container configuration when Azure storage mode is enabled.

#### Scenario: Valid Azure configuration with connection string
- **GIVEN** Azure storage mode is enabled
- **AND** `AZURE_STORAGE_CONNECTION_STRING` is set with a valid connection string
- **AND** `AZURE_STORAGE_CONTAINER` is set
- **WHEN** the runner starts
- **THEN** configuration validation passes
- **AND** the runner proceeds with Azure storage

#### Scenario: Valid Azure configuration with account and key
- **GIVEN** Azure storage mode is enabled
- **AND** `AZURE_STORAGE_ACCOUNT` and `AZURE_STORAGE_KEY` are set
- **AND** `AZURE_STORAGE_CONTAINER` is set
- **WHEN** the runner starts
- **THEN** configuration validation passes
- **AND** the runner proceeds with Azure storage

#### Scenario: Missing required Azure configuration
- **GIVEN** Azure storage mode is enabled
- **AND** `AZURE_STORAGE_CONTAINER` is missing
- **WHEN** the runner starts
- **THEN** a configuration error is logged
- **AND** the runner exits with a non-zero status code

#### Scenario: Invalid Azure connection string
- **GIVEN** Azure storage mode is enabled
- **AND** `AZURE_STORAGE_CONNECTION_STRING` is malformed
- **WHEN** the runner starts
- **THEN** a configuration error is logged
- **AND** the runner exits with a non-zero status code

### Requirement: Incident Artifact Upload

The system SHALL upload individual incident artifacts to Azure Blob Storage with a consistent path structure.

#### Scenario: Artifact path structure
- **GIVEN** an incident with ID `abc-123`
- **WHEN** artifacts are uploaded to Azure
- **THEN** blobs SHALL be stored with paths following the pattern `<incident-id>/<filename>`
- **AND** event.json is uploaded to `abc-123/event.json`
- **AND** result.json is uploaded to `abc-123/result.json`
- **AND** investigation.md is uploaded to `abc-123/output/investigation.md`

#### Scenario: Successful artifact upload
- **GIVEN** Azure storage mode is enabled
- **AND** Azure credentials are valid
- **WHEN** an investigation completes
- **THEN** all incident artifacts SHALL be uploaded to the configured container
- **AND** upload status SHALL be logged

#### Scenario: Upload failure handling
- **GIVEN** Azure storage mode is enabled
- **WHEN** an Azure upload fails (network error, permission denied, etc.)
- **THEN** the error SHALL be logged with full context
- **AND** the investigation result SHALL NOT be marked as failed
- **AND** the runner SHALL continue processing other incidents

### Requirement: SAS URL Generation

The system SHALL generate Shared Access Signature (SAS) URLs for uploaded artifacts to allow authenticated access without credentials.

#### Scenario: SAS URL creation
- **GIVEN** an artifact is successfully uploaded to Azure Blob Storage
- **WHEN** the upload completes
- **THEN** a SAS URL SHALL be generated for the artifact
- **AND** the URL SHALL be valid for the configured expiration period

#### Scenario: Default SAS URL expiration
- **GIVEN** `AZURE_SAS_EXPIRY` is not configured
- **WHEN** generating a SAS URL
- **THEN** the URL SHALL expire after 7 days (168 hours)

#### Scenario: Custom SAS URL expiration
- **GIVEN** `AZURE_SAS_EXPIRY` is set to `24h`
- **WHEN** generating a SAS URL
- **THEN** the URL SHALL expire after 24 hours

#### Scenario: SAS URL access
- **GIVEN** a valid SAS URL for an uploaded artifact
- **WHEN** an operator accesses the URL
- **THEN** the artifact content SHALL be returned without additional authentication

### Requirement: Slack Notification with Report Link

The system SHALL include a clickable SAS URL in Slack notifications when cloud storage is enabled.

#### Scenario: Slack message with report URL
- **GIVEN** Azure storage mode is enabled
- **AND** investigation.md is successfully uploaded
- **WHEN** a Slack notification is sent
- **THEN** the message SHALL include a clickable link to the investigation report
- **AND** the link SHALL be a SAS URL

#### Scenario: Slack message format with button
- **GIVEN** Azure storage mode is enabled
- **WHEN** formatting the Slack notification
- **THEN** the report link SHALL be presented as a "View Report" button or hyperlink
- **AND** the button/link SHALL use the SAS URL

#### Scenario: Slack message without cloud storage
- **GIVEN** filesystem storage mode is active (Azure not configured)
- **WHEN** a Slack notification is sent
- **THEN** the message SHALL include the local filesystem path (existing behavior)
- **AND** no SAS URL SHALL be generated

### Requirement: Result Metadata with URLs

The system SHALL store SAS URLs in the result.json for programmatic access.

#### Scenario: URLs in result.json
- **GIVEN** Azure storage mode is enabled
- **AND** artifacts are successfully uploaded
- **WHEN** result.json is generated
- **THEN** it SHALL include a `presigned_urls` object
- **AND** the object SHALL contain keys for each artifact (`event_json`, `result_json`, `investigation_md`)
- **AND** each value SHALL be the SAS URL for that artifact

#### Scenario: URL expiration metadata
- **GIVEN** SAS URLs are generated
- **WHEN** result.json is written
- **THEN** it SHALL include `presigned_urls_expire_at` timestamp
- **AND** the timestamp SHALL reflect when the URLs will expire

### Requirement: Prompt Capture Artifact

The system SHALL capture and store the prompt sent to the agent for auditability.

#### Scenario: Prompt capture before execution
- **GIVEN** an incident investigation is starting
- **WHEN** the agent executor is about to launch the subprocess
- **THEN** the full prompt (system + additional) SHALL be written to `prompt-sent.md` in the workspace
- **AND** the file SHALL be written before the subprocess starts

#### Scenario: Prompt capture metadata
- **GIVEN** prompt-sent.md is being generated
- **WHEN** the file content is created
- **THEN** it SHALL include metadata: timestamp, incident ID, cluster name, agent CLI, and model
- **AND** it SHALL include the full system prompt content
- **AND** it SHALL include the additional prompt content (or "None provided" if empty)

#### Scenario: Prompt artifact upload
- **GIVEN** Azure storage mode is enabled
- **AND** an investigation completes
- **WHEN** artifacts are uploaded
- **THEN** prompt-sent.md SHALL be uploaded alongside other artifacts
- **AND** it SHALL appear in the index.html file listing

#### Scenario: Prompt artifact optional
- **GIVEN** prompt-sent.md does not exist in the workspace
- **WHEN** artifacts are read for upload
- **THEN** the upload SHALL succeed without prompt-sent.md
- **AND** no error SHALL be logged (prompt is optional for backwards compatibility)

### Requirement: Debug Log Artifacts

The system SHALL upload debug log artifacts when running in DEBUG mode.

#### Scenario: Debug log files uploaded
- **GIVEN** Azure storage mode is enabled
- **AND** the system is running in DEBUG mode
- **WHEN** an investigation completes
- **THEN** the following log files SHALL be uploaded to `{incident-id}/logs/`:
  - `agent-stdout.log` - Agent standard output
  - `agent-stderr.log` - Agent standard error
  - `agent-full.log` - Combined timestamped log
  - `agent-commands-executed.log` - Extracted Bash commands from session
  - `claude-session.tar.gz` - Complete Claude Code session archive

#### Scenario: Debug logs in index.html
- **GIVEN** debug logs were captured and uploaded
- **WHEN** index.html is generated
- **THEN** all log files SHALL appear in the file listing
- **AND** each log SHALL be labeled "(DEBUG mode only)"
- **AND** logs SHALL appear after core artifacts in the display order

#### Scenario: Empty debug logs skipped
- **GIVEN** a debug log file is empty (zero bytes)
- **WHEN** artifacts are uploaded
- **THEN** the empty file SHALL NOT be uploaded
- **AND** no error SHALL be logged

