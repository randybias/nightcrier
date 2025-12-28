# Tasks: Refactor Object Storage to Go CDK

## 1. Dependencies and Setup

- [x] 1.1 Add Go CDK blob dependencies to go.mod:
  - `gocloud.dev/blob`
  - `gocloud.dev/blob/azureblob`
  - `gocloud.dev/blob/s3blob`
  - `gocloud.dev/blob/memblob`
- [x] 1.2 Run `go mod tidy` to resolve transitive dependencies
- [x] 1.3 Verify AWS SDK v2 is pulled in (transitive via s3blob)

## 2. Configuration Changes

- [x] 2.1 Add `ObjectStorage` struct to Config with fields:
  - `URL` (string) - Go CDK storage URL
  - `SignedURLExpiry` (string) - duration like "168h"
  - `AWSAccessKeyID` (string) - S3/MinIO credentials
  - `AWSSecretAccessKey` (string)
  - `AzureStorageAccount` (string) - Azure credentials
  - `AzureStorageKey` (string)
- [x] 2.2 Add environment variable bindings:
  - `OBJECT_STORAGE_URL`
  - `OBJECT_STORAGE_SIGNED_URL_EXPIRY`
  - `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`
  - `AZURE_STORAGE_ACCOUNT`, `AZURE_STORAGE_KEY`
- [x] 2.3 Add CLI flag bindings for credential overrides
- [x] 2.4 Remove old Azure-specific config fields from root Config:
  - `AzureStorageConnectionString`
  - `AzureStorageAccount` (moved to ObjectStorage)
  - `AzureStorageKey` (moved to ObjectStorage)
  - `AzureStorageContainer`
  - `AzureSASExpiry`
- [x] 2.5 Remove old Azure config methods:
  - `IsAzureStorageEnabled()`
  - `ValidateAzureConfig()`
  - `validateConnectionString()`
  - Old Azure getter methods
- [x] 2.6 Add `ValidateObjectStorageConfig()` method:
  - Validate URL scheme is supported (azblob, s3, mem)
  - Validate credentials present for URL scheme
  - Validate both credentials present (not partial)
  - Skip credential check for mem:// and when URL not set
- [x] 2.7 Update `configs/config.example.yaml` with new `object_storage:` section

## 3. ObjectStore Implementation

- [x] 3.1 Create `internal/storage/objectstore.go` with ObjectStore struct
- [x] 3.2 Implement `NewObjectStore(ctx, url, expiry)` constructor using `blob.OpenBucket`
- [x] 3.3 Implement `Upload(ctx, key, data, contentType)` method
- [x] 3.4 Implement `SignedURL(ctx, key)` returning signed URL and expiration time
- [x] 3.5 Implement `CanonicalURL(key)` for provider-specific base URLs (no auth)
- [x] 3.6 Implement `SignCanonicalURL(ctx, canonicalURL)` for re-signing expired URLs
- [x] 3.7 Implement `extractKeyFromCanonicalURL(canonicalURL)` helper
- [x] 3.8 Implement `Close()` method
- [x] 3.9 Add provider detection from URL scheme (azblob, s3)
- [x] 3.10 Extract and store bucket/container name, endpoint, account, region for URL generation

## 4. Storage Interface Integration

- [x] 4.1 Create `internal/storage/objectstore_storage.go` implementing `Storage` interface
- [x] 4.2 Implement `SaveIncident()` using ObjectStore methods
- [x] 4.3 Port artifact upload logic from azure.go (content-type mapping, path structure)
- [x] 4.4 Port index.html generation from azure.go (use signed URLs for clickable links)
- [x] 4.5 Update `SaveResult` struct to include `CanonicalURLs` map
- [x] 4.6 Populate both `ArtifactURLs` (signed) and `CanonicalURLs` in SaveResult

## 5. Factory Update

- [x] 5.1 Update `NewStorage()` in `internal/storage/storage.go`:
  - Check for object storage URL via `GetObjectStorageURL()`
  - Create ObjectStore-based storage if URL present
  - Fall back to FilesystemStorage if no URL
  - Set provider environment variables before bucket initialization
- [x] 5.2 Remove Azure-specific factory logic
- [x] 5.3 Update `StorageConfig` interface (add credential getter methods)

## 6. Cleanup

- [x] 6.1 Delete `internal/storage/azure.go`
- [x] 6.2 Delete `internal/storage/azure_test.go`
- [x] 6.3 Remove Azure SDK import from go.mod (`github.com/Azure/azure-sdk-for-go/sdk/storage/azblob`)
- [x] 6.4 Run `go mod tidy` to clean up unused dependencies

## 7. Testing

- [x] 7.1 Create `internal/storage/objectstore_test.go` with unit tests using `mem://`
- [x] 7.2 Test Upload and SignedURL operations
- [x] 7.3 Test CanonicalURL generation for azblob and s3 schemes
- [x] 7.4 Test SignCanonicalURL re-signing functionality
- [x] 7.5 Test extractKeyFromCanonicalURL for various URL formats
- [x] 7.6 Create `internal/storage/objectstore_storage_test.go`
- [x] 7.7 Test SaveIncident with in-memory bucket
- [x] 7.8 Test artifact path structure and content-type mapping
- [x] 7.9 Verify SaveResult contains both signed and canonical URLs
- [x] 7.10 Verify index.html links are clickable signed URLs
- [x] 7.11 Update any existing integration tests that referenced Azure

## 8. Documentation

- [x] 8.1 Update README with new storage configuration examples
- [x] 8.2 Document S3-compatible endpoint configuration (MinIO, RustFS)
- [x] 8.3 Document credential requirements per provider

## 9. Validation

- [x] 9.1 Run `go build ./...` to verify compilation
- [x] 9.2 Run `go test ./internal/storage/...` to verify tests pass
- [x] 9.3 Run `go vet ./...` to check for issues
- [x] 9.4 Test with `STORAGE_URL=mem://` to verify in-memory operation
- [x] 9.5 Test with RustFS or MinIO container for S3 compatibility (if available)
