# cloud-storage Spec Delta

## MODIFIED Requirements

### Requirement: Cloud Storage Backend

The system SHALL support uploading incident artifacts to cloud storage using Go CDK's blob abstraction, with support for Azure Blob Storage and S3-compatible storage backends.

#### Scenario: Cloud storage mode enabled via OBJECT_STORAGE_URL
- **GIVEN** `OBJECT_STORAGE_URL` environment variable is configured with a valid Go CDK URL
- **WHEN** the runner starts
- **THEN** cloud storage mode is activated using the specified provider
- **AND** signed URLs are generated for uploaded artifacts

#### Scenario: S3-compatible storage
- **GIVEN** `OBJECT_STORAGE_URL` is set to an S3 URL (e.g., `s3://bucket?region=us-east-1`)
- **WHEN** the runner starts
- **THEN** S3 storage mode is activated
- **AND** AWS credentials are loaded from the standard credential chain

#### Scenario: S3-compatible with custom endpoint
- **GIVEN** `OBJECT_STORAGE_URL` is set with custom endpoint (e.g., `s3://bucket?endpoint=http://minio:9000&use_path_style=true`)
- **WHEN** the runner starts
- **THEN** the custom endpoint is used for S3 operations
- **AND** path-style URLs are used instead of virtual-hosted style

#### Scenario: Azure Blob storage
- **GIVEN** `OBJECT_STORAGE_URL` is set to an Azure URL (e.g., `azblob://container`)
- **WHEN** the runner starts
- **THEN** Azure Blob Storage mode is activated
- **AND** Azure credentials are loaded from environment variables

#### Scenario: Filesystem storage mode (default)
- **GIVEN** `OBJECT_STORAGE_URL` is not set
- **WHEN** the runner starts
- **THEN** local filesystem storage is used
- **AND** cloud storage is not attempted

#### Scenario: Storage interface extensibility
- **GIVEN** the storage subsystem uses Go CDK's blob abstraction
- **WHEN** a new storage backend is needed
- **THEN** only a new Go CDK driver import is required
- **AND** no changes are needed to reporting or notification code

### Requirement: Incident Artifact Upload

The system SHALL upload individual incident artifacts to cloud storage with a consistent path structure.

#### Scenario: Artifact path structure
- **GIVEN** `OBJECT_STORAGE_URL` is set to `azblob://my-container`
- **AND** Azure credentials are available via environment
- **WHEN** the runner starts
- **THEN** configuration validation passes
- **AND** the runner proceeds with Azure storage

#### Scenario: Valid S3 URL
- **GIVEN** `OBJECT_STORAGE_URL` is set to `s3://my-bucket?region=us-east-1`
- **AND** AWS credentials are available via environment or IAM
- **WHEN** the runner starts
- **THEN** configuration validation passes
- **AND** the runner proceeds with S3 storage

#### Scenario: Valid S3-compatible URL with endpoint
- **GIVEN** `OBJECT_STORAGE_URL` is set to `s3://bucket?endpoint=http://localhost:9000&use_path_style=true&disable_https=true`
- **AND** S3-compatible credentials are available
- **WHEN** the runner starts
- **THEN** configuration validation passes
- **AND** the runner connects to the custom endpoint

#### Scenario: Invalid OBJECT_STORAGE_URL scheme
- **GIVEN** `OBJECT_STORAGE_URL` is set to an unsupported scheme (e.g., `gcs://bucket`)
- **WHEN** the runner starts
- **THEN** a configuration error is logged
- **AND** the runner exits with a non-zero status code

#### Scenario: Malformed OBJECT_STORAGE_URL
- **GIVEN** `OBJECT_STORAGE_URL` is set to an invalid URL
- **WHEN** the runner starts
- **THEN** a configuration error is logged
- **AND** the runner exits with a non-zero status code

#### Scenario: Artifact path structure
- **GIVEN** an incident with ID `abc-123`
- **WHEN** artifacts are uploaded to cloud storage
- **THEN** objects SHALL be stored with paths following the pattern `<incident-id>/<filename>`
- **AND** incident.json is uploaded to `abc-123/incident.json`
- **AND** investigation.md is uploaded to `abc-123/investigation.md`
- **AND** investigation.html is uploaded to `abc-123/investigation.html`

#### Scenario: Successful artifact upload
- **GIVEN** cloud storage mode is enabled
- **AND** credentials are valid
- **WHEN** an investigation completes
- **THEN** all incident artifacts SHALL be uploaded to the configured bucket/container
- **AND** upload status SHALL be logged

#### Scenario: Upload failure handling
- **GIVEN** cloud storage mode is enabled
- **WHEN** an upload fails (network error, permission denied, etc.)
- **THEN** the error SHALL be logged with full context
- **AND** the investigation result SHALL NOT be marked as failed
- **AND** the runner SHALL continue processing other incidents

### Requirement: Signed URL Generation

The system SHALL generate signed URLs for uploaded artifacts to allow authenticated access without credentials.

#### Scenario: Signed URL creation
- **GIVEN** an artifact is successfully uploaded to cloud storage
- **WHEN** the upload completes
- **THEN** a signed URL SHALL be generated for the artifact using Go CDK's SignedURL method
- **AND** the URL SHALL be valid for the configured expiration period

#### Scenario: Default signed URL expiration
- **GIVEN** `OBJECT_STORAGE_SIGNED_URL_EXPIRY` is not configured
- **WHEN** generating a signed URL
- **THEN** the URL SHALL expire after 7 days (168 hours)

#### Scenario: Custom signed URL expiration
- **GIVEN** `OBJECT_STORAGE_SIGNED_URL_EXPIRY` is set to `24h`
- **WHEN** generating a signed URL
- **THEN** the URL SHALL expire after 24 hours

#### Scenario: Signed URL access
- **GIVEN** a valid signed URL for an uploaded artifact
- **WHEN** an operator accesses the URL
- **THEN** the artifact content SHALL be returned without additional authentication

### Requirement: Slack Notification with Report Link

The system SHALL include a clickable signed URL in Slack notifications when cloud storage is enabled.

#### Scenario: Slack message with report URL
- **GIVEN** cloud storage mode is enabled
- **AND** investigation.html is successfully uploaded
- **WHEN** a Slack notification is sent
- **THEN** the message SHALL include a clickable link to the investigation report
- **AND** the link SHALL be a signed URL

#### Scenario: Slack message format with button
- **GIVEN** cloud storage mode is enabled
- **WHEN** formatting the Slack notification
- **THEN** the report link SHALL be presented as a "View Report" button or hyperlink
- **AND** the button/link SHALL use the signed URL

#### Scenario: Slack message without cloud storage
- **GIVEN** filesystem storage mode is active (OBJECT_STORAGE_URL not configured)
- **WHEN** a Slack notification is sent
- **THEN** the message SHALL include the local filesystem path (existing behavior)
- **AND** no signed URL SHALL be generated

### Requirement: Result Metadata with URLs

The system SHALL store signed URLs in the result.json for programmatic access.

#### Scenario: URLs in result.json
- **GIVEN** cloud storage mode is enabled
- **AND** artifacts are successfully uploaded
- **WHEN** result.json is generated
- **THEN** it SHALL include a `presigned_urls` object
- **AND** the object SHALL contain keys for each artifact
- **AND** each value SHALL be the signed URL for that artifact

#### Scenario: URL expiration metadata
- **GIVEN** signed URLs are generated
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
- **GIVEN** cloud storage mode is enabled
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
- **GIVEN** cloud storage mode is enabled
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

## ADDED Requirements

### Requirement: Canonical URL Generation

The system SHALL provide canonical (non-authenticated) URLs for uploaded artifacts for reference, logging, and re-signing purposes.

#### Scenario: Azure canonical URL format
- **GIVEN** an artifact is uploaded to Azure Blob Storage
- **WHEN** a canonical URL is requested
- **THEN** the URL SHALL follow the format `https://{account}.blob.core.windows.net/{container}/{key}`

#### Scenario: S3 canonical URL format
- **GIVEN** an artifact is uploaded to AWS S3
- **WHEN** a canonical URL is requested
- **THEN** the URL SHALL follow the format `https://{bucket}.s3.{region}.amazonaws.com/{key}`

#### Scenario: S3-compatible canonical URL format
- **GIVEN** an artifact is uploaded to an S3-compatible service with custom endpoint
- **WHEN** a canonical URL is requested
- **THEN** the URL SHALL follow the format `{endpoint}/{bucket}/{key}`

#### Scenario: Canonical URL stored in result metadata
- **GIVEN** artifacts are successfully uploaded
- **WHEN** result.json is generated
- **THEN** it SHALL include a `canonical_urls` object mapping artifact names to canonical URLs
- **AND** canonical URLs SHALL be stored separately from signed URLs

### Requirement: Signed URL for Clickable Access

The system SHALL generate signed URLs that include authentication for direct clickable access to artifacts in private buckets.

#### Scenario: Signed URL includes authentication
- **GIVEN** a canonical URL for an artifact in a private bucket
- **WHEN** a signed URL is generated
- **THEN** the URL SHALL include provider-specific authentication parameters
- **AND** the URL SHALL be directly clickable without additional authentication

#### Scenario: S3-compatible signed URL format
- **GIVEN** an artifact in an S3-compatible bucket (MinIO/RustFS)
- **WHEN** a signed URL is generated
- **THEN** the URL SHALL include `X-Amz-Algorithm`, `X-Amz-Credential`, `X-Amz-Signature` query parameters
- **AND** clicking the URL SHALL return the artifact content

#### Scenario: Azure signed URL format
- **GIVEN** an artifact in Azure Blob Storage
- **WHEN** a signed URL is generated
- **THEN** the URL SHALL include SAS token query parameters
- **AND** clicking the URL SHALL return the artifact content

#### Scenario: Signed URLs in index.html
- **GIVEN** index.html is generated for an incident
- **WHEN** artifact links are rendered
- **THEN** all links SHALL use signed URLs
- **AND** clicking any link SHALL directly display/download the artifact

### Requirement: URL Re-signing Capability

The system SHALL support generating new signed URLs from stored canonical URLs when existing signed URLs expire.

#### Scenario: Re-sign from canonical URL
- **GIVEN** a canonical URL stored in result.json
- **AND** the original signed URL has expired
- **WHEN** a new signed URL is requested for that canonical URL
- **THEN** the system SHALL generate a fresh signed URL with new expiration

#### Scenario: Re-sign preserves artifact identity
- **GIVEN** a canonical URL `http://minio:9000/bucket/incident-123/report.html`
- **WHEN** a signed URL is generated
- **THEN** the signed URL SHALL point to the same artifact
- **AND** the canonical URL portion SHALL be preserved in the signed URL

#### Scenario: Batch re-signing
- **GIVEN** a result.json with multiple canonical URLs
- **WHEN** re-signing is requested for all artifacts
- **THEN** new signed URLs SHALL be generated for each canonical URL
- **AND** expiration times SHALL be consistent across all URLs

### Requirement: In-Memory Storage for Testing

The system SHALL support an in-memory storage backend for unit testing.

#### Scenario: Memory URL scheme
- **GIVEN** `OBJECT_STORAGE_URL` is set to `mem://`
- **WHEN** the storage is initialized
- **THEN** an in-memory bucket SHALL be created
- **AND** all operations SHALL work without external dependencies

#### Scenario: Memory storage isolation
- **GIVEN** multiple tests use `mem://` storage
- **WHEN** each test creates its own storage instance
- **THEN** data SHALL NOT leak between test instances

## REMOVED Requirements

### Requirement: Azure Configuration

**Reason**: Replaced by provider-agnostic Storage URL Configuration requirement. Azure-specific environment variables (`AZURE_STORAGE_CONNECTION_STRING`, `AZURE_STORAGE_ACCOUNT`, `AZURE_STORAGE_KEY`, `AZURE_STORAGE_CONTAINER`) are replaced by the unified `OBJECT_STORAGE_URL` configuration.

**Migration**: Use `OBJECT_STORAGE_URL=azblob://container-name` with Azure credentials in environment.
