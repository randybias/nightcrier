# Change: Add Cloud Incident Storage with Authenticated URL Access

## Why

Operators need a way to access incident data directly from Slack notifications without SSH access to the host filesystem. By uploading incident artifacts to cloud storage (Azure Blob Storage) and including authenticated URLs in Slack messages, operators can click a link and immediately view the incident report.

## What Changes

- Add new `cloud-storage` capability as an alternative to local filesystem storage
- Either/or configuration: use cloud storage OR local filesystem, not both simultaneously
- Implement Azure Blob Storage backend with SAS (Shared Access Signature) URLs
- Upload individual incident files (event.json, investigation.md, result.json) with incident ID prefix
- Generate SAS URLs (7-day default expiration) for each uploaded file
- Include SAS URL for investigation.md in Slack notifications
- Design storage interface to allow future backends (S3, GCS) without breaking changes

## Impact

- **New Capabilities**: `cloud-storage`
- **Modified Code**:
  - `internal/config/config.go` - Cloud storage configuration options
  - `internal/storage/` - New package with Storage interface and Azure implementation
  - `internal/reporting/slack.go` - Add report URL to notifications
- **Dependencies**:
  - Azure SDK for Go (`github.com/Azure/azure-sdk-for-go/sdk/storage/azblob`)
- **Breaking Changes**: None - cloud storage is opt-in via configuration
