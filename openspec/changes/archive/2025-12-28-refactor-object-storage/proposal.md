# Change: Refactor Object Storage to Go CDK

## Why

The current object storage implementation is tightly coupled to Azure Blob Storage. Operators need flexibility to deploy with S3-compatible storage (AWS S3, MinIO, RustFS) for on-premises, air-gapped, or cost-conscious deployments. Go CDK provides a mature abstraction layer with native signed URL generation.

## What Changes

### Core Changes

- **Replace Azure-specific storage implementation** with Go CDK's `blob.Bucket` abstraction
- **Add S3-compatible storage support** via Go CDK's `s3blob` driver
- **Introduce URL-based storage configuration** (`OBJECT_STORAGE_URL`) for cloud providers
- **Keep existing filesystem implementation** unchanged (default fallback)
- **Remove Azure SDK direct dependency** in favor of Go CDK drivers

### Configuration Changes

- **REMOVE**: All `AZURE_STORAGE_*` environment variables
- **NEW**: `OBJECT_STORAGE_URL` environment variable using Go CDK URL format:
  - Azure: `azblob://container-name`
  - S3: `s3://bucket-name?region=us-east-1`
  - S3-compatible: `s3://bucket?endpoint=http://minio:9000&use_path_style=true&disable_https=true`
- **NEW**: `OBJECT_STORAGE_SIGNED_URL_EXPIRY` duration (default: 168h)
- Filesystem storage continues using `WORKSPACE_ROOT` (no `OBJECT_STORAGE_URL` needed)

### S3-Compatible Endpoint Support

- Custom S3 endpoints via URL query parameters for MinIO/RustFS:
  - `endpoint` - Custom endpoint URL
  - `use_path_style=true` - Required for most S3-compatible services
  - `disable_https=true` - For local development
  - `awssdk=v2` - Use AWS SDK v2

## Impact

### Affected Specs

- `cloud-storage` - Replace Azure-specific requirements with provider-agnostic
- `configuration` - New storage URL configuration

### Affected Code

- `internal/storage/storage.go` - Interface unchanged, factory updated
- `internal/storage/azure.go` - Delete, replace with Go CDK wrapper
- `internal/storage/objectstore.go` - New Go CDK wrapper
- `internal/config/config.go` - Replace Azure config with `OBJECT_STORAGE_URL`

### Dependencies

- **Add**: `gocloud.dev/blob`, `gocloud.dev/blob/azureblob`, `gocloud.dev/blob/s3blob`, `gocloud.dev/blob/memblob`
- **Remove**: `github.com/Azure/azure-sdk-for-go/sdk/storage/azblob`

### Testing

- Unit tests use `mem://` in-memory bucket
- Integration tests can use RustFS container

## Non-Goals

- Google Cloud Storage support (add later if needed)
- Prometheus metrics
- Replacing existing filesystem storage implementation

## Build Optimization

Use selective blank imports to keep binary trim:
- Only import needed drivers: `azureblob`, `s3blob`, `memblob`
- Go linker eliminates unused code
- Unused drivers (gcsblob, etc.) not compiled into binary
