# Storage Configuration

When object storage is configured, incident artifacts are automatically uploaded to cloud storage and signed URLs are generated for secure access. If object storage is not configured, the system falls back to filesystem storage.

Nightcrier supports multiple storage backends through Go Cloud Development Kit (CDK):
- **Azure Blob Storage** - Azure's cloud object storage
- **AWS S3** - Amazon's Simple Storage Service
- **S3-compatible storage** - MinIO, RustFS, and other S3-compatible providers

## Configuration via URL

Object storage is configured using a single URL that specifies the provider and bucket/container:

- `OBJECT_STORAGE_URL` - Storage URL (required for cloud storage)
- `OBJECT_STORAGE_SIGNED_URL_EXPIRY` - Signed URL expiration duration (default: `168h` / 7 days)

### Azure Blob Storage
```bash
export OBJECT_STORAGE_URL="azblob://mycontainer"
export AZURE_STORAGE_ACCOUNT="mystorageaccount"
export AZURE_STORAGE_KEY="your-account-key"
export OBJECT_STORAGE_SIGNED_URL_EXPIRY="168h"
```

### AWS S3
```bash
export OBJECT_STORAGE_URL="s3://mybucket?region=us-east-1"
export AWS_ACCESS_KEY_ID="your-access-key-id"
export AWS_SECRET_ACCESS_KEY="your-secret-access-key"
export OBJECT_STORAGE_SIGNED_URL_EXPIRY="168h"
```

### S3-compatible (MinIO, RustFS)
```bash
export OBJECT_STORAGE_URL="s3://mybucket?endpoint=http://minio:9000&disable_https=true&use_path_style=true&region=us-east-1"
export AWS_ACCESS_KEY_ID="minioadmin"
export AWS_SECRET_ACCESS_KEY="minioadmin"
export OBJECT_STORAGE_SIGNED_URL_EXPIRY="168h"
```

### In-memory (testing only)
```bash
export OBJECT_STORAGE_URL="mem://"
```

## Credential Requirements by Provider

**Azure Blob Storage** (azblob://):
- `AZURE_STORAGE_ACCOUNT` - Storage account name (required)
- `AZURE_STORAGE_KEY` - Storage account access key (required)

**AWS S3** (s3://):
- `AWS_ACCESS_KEY_ID` - AWS access key ID (required)
- `AWS_SECRET_ACCESS_KEY` - AWS secret access key (required)

**S3-compatible** (s3:// with endpoint):
- `AWS_ACCESS_KEY_ID` - Access key ID for the S3-compatible service (required)
- `AWS_SECRET_ACCESS_KEY` - Secret access key for the S3-compatible service (required)
- URL parameters (Go CDK uses snake_case):
  - `endpoint` - Service endpoint URL (e.g., `http://minio:9000`)
  - `disable_https=true` - Disable SSL for local/development setups
  - `use_path_style=true` - Use path-style addressing (required for MinIO)
  - `region` - Region name (can be any value for MinIO, typically `us-east-1`)

**In-memory** (mem://):
- No credentials required (testing only, data lost on restart)

## Storage Mode Detection

The system automatically detects which storage backend to use:
- **Object Storage**: Used when `OBJECT_STORAGE_URL` is set
- **Filesystem Storage**: Used as fallback when object storage is not configured

## What the System Does

When object storage is configured, the system will:
- Upload all artifacts to `<bucket/container>/<incident-id>/` structure
- Generate signed URLs with read-only access
- Include signed URLs in Slack notifications and result.json
- Set URL expiration based on `OBJECT_STORAGE_SIGNED_URL_EXPIRY`
- Store both signed URLs (temporary) and canonical URLs (permanent) in result.json

## Bucket/Container Structure

The bucket/container must have the following structure:
```
<bucket or container>/
  <incident-id>/
    event.json              # Original fault event
    result.json             # Execution result with URLs
    output/
      investigation.md      # AI-generated investigation report
```
