## 1. Configuration

- [x] 1.1 Add Azure storage configuration fields to `internal/config/config.go` (connection string, account, key, container, SAS expiry)
- [x] 1.2 Implement storage mode detection (Azure if `AZURE_STORAGE_ACCOUNT` or `AZURE_STORAGE_CONNECTION_STRING` set, else filesystem)
- [x] 1.3 Validate Azure configuration on startup (required fields present, connection string parseable)
- [x] 1.4 Add unit tests for Azure configuration loading and validation

## 2. Storage Interface

- [x] 2.1 Create `internal/storage/storage.go` with `Storage` interface
- [x] 2.2 Define interface: `SaveIncident(ctx, incidentID, artifacts) (*SaveResult, error)`
- [x] 2.3 Define `IncidentArtifacts` struct (EventJSON, ResultJSON, InvestigationMD as []byte)
- [x] 2.4 Define `SaveResult` struct (ReportURL, ArtifactURLs map, ExpiresAt)
- [x] 2.5 Create storage factory function `NewStorage(cfg) (Storage, error)`

## 3. Filesystem Storage Adapter

- [x] 3.1 Create `internal/storage/filesystem.go` implementing `Storage` interface
- [x] 3.2 Wrap existing workspace write logic in adapter
- [x] 3.3 Return filesystem paths (not URLs) in `SaveResult` for local storage mode
- [x] 3.4 Add unit tests for filesystem adapter

## 4. Azure Blob Storage Client

- [x] 4.1 Add Azure SDK dependency to `go.mod` (`github.com/Azure/azure-sdk-for-go/sdk/storage/azblob`)
- [x] 4.2 Create `internal/storage/azure.go` with AzureStorage struct
- [x] 4.3 Implement constructor supporting both connection string and account+key auth
- [x] 4.4 Implement `uploadBlob(ctx, blobPath, data)` method
- [x] 4.5 Implement `generateSASURL(blobPath, expiry)` method using Service SAS
- [x] 4.6 Add unit tests with Azurite emulator or mocks

## 5. Incident Upload Implementation

- [x] 5.1 Implement `SaveIncident` for AzureStorage
- [x] 5.2 Upload each artifact to `<container>/<incident-id>/<filename>`
- [x] 5.3 Generate SAS URLs for all uploaded artifacts
- [x] 5.4 Populate SaveResult with URLs and expiration time
- [x] 5.5 Handle upload errors gracefully (log, return partial results)
- [x] 5.6 Add integration test with Azurite

## 6. Slack Integration

- [x] 6.1 Add `ReportURL` field to `IncidentSummary` struct
- [x] 6.2 Update `SendIncidentNotification` to include clickable link when URL available
- [x] 6.3 Format SAS URL as Slack button or hyperlink in Block Kit
- [x] 6.4 Add "View Report" button to Slack message template
- [x] 6.5 Preserve existing behavior (filesystem path) when cloud storage not configured
- [x] 6.6 Add unit tests for Slack message formatting with URL

## 7. Result Metadata

- [x] 7.1 Add `presigned_urls` field to result.json structure
- [x] 7.2 Add `presigned_urls_expire_at` timestamp field
- [x] 7.3 Populate URL fields when cloud storage is used
- [x] 7.4 Update result.go to include new fields

## 8. Orchestration

- [x] 8.1 Update `cmd/runner/main.go` to initialize storage based on config
- [x] 8.2 Inject storage implementation into reporting flow
- [x] 8.3 Call `storage.SaveIncident()` after agent execution
- [x] 8.4 Pass report URL from SaveResult to Slack notifier
- [x] 8.5 Verify filesystem fallback works when Azure not configured

## 9. Documentation and Testing

- [x] 9.1 Add Azure configuration section to README
- [x] 9.2 Document Azurite setup for local development
- [x] 9.3 Add example docker-compose with Azurite for local testing
- [x] 9.4 End-to-end test: trigger incident, verify Azure upload, verify SAS URL works
