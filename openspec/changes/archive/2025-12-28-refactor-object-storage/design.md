# Design: Go CDK Object Storage Abstraction

## Context

The current storage implementation uses Azure Blob Storage directly via the Azure SDK. Operators need flexibility to deploy with S3-compatible storage (AWS S3, MinIO, RustFS). Go CDK provides a production-ready abstraction with native signed URL support.

## Goals

1. Support Azure Blob and S3-compatible storage backends
2. Unify signed URL generation across providers
3. Simplify configuration with URL-based provider selection
4. Enable easy unit testing with in-memory storage
5. Keep existing filesystem storage unchanged

## Non-Goals

1. Google Cloud Storage support (add later if needed)
2. Prometheus metrics instrumentation
3. Replacing existing filesystem storage implementation

## Decisions

### Decision 1: Use Go CDK blob package

**Choice**: Use `gocloud.dev/blob` as the storage abstraction layer.

**Rationale**:
- Native `SignedURL()` support - critical for sharing investigation reports
- URL-based configuration aligns with 12-factor app principles
- In-memory implementation (`memblob`) for testing
- Active maintenance and Google backing

**Alternatives Considered**:

| Option | Pros | Cons |
|--------|------|------|
| Thanos objstore | Battle-tested, Prometheus metrics | No signed URL support |
| Custom abstraction | Full control | Maintenance burden |
| Direct SDK per provider | No abstraction overhead | Duplicated logic |

### Decision 2: URL-based configuration

**Choice**: Configure cloud storage via `OBJECT_STORAGE_URL` environment variable.

**Examples**:
```bash
# Azure Blob Storage
OBJECT_STORAGE_URL="azblob://incidents"

# AWS S3
OBJECT_STORAGE_URL="s3://my-bucket?region=us-east-1"

# S3-compatible (MinIO/RustFS)
OBJECT_STORAGE_URL="s3://incidents?endpoint=http://minio:9000&use_path_style=true&disable_https=true&awssdk=v2"
```

**Rationale**:
- Single configuration point
- Provider selection explicit in URL scheme
- Query parameters handle provider-specific options
- Aligns with Go CDK design philosophy

### Decision 3: Selective imports for build trimming

**Choice**: Use selective blank imports to include only needed drivers.

```go
// internal/storage/objectstore.go
package storage

import (
    "gocloud.dev/blob"
    _ "gocloud.dev/blob/azureblob"  // Azure Blob support
    _ "gocloud.dev/blob/s3blob"     // S3/MinIO/RustFS support
    _ "gocloud.dev/blob/memblob"    // In-memory for testing
    // NOT importing gcsblob - GCS code not compiled into binary
)
```

**Rationale**:
- Simple, no code generation needed
- Go's linker eliminates unused code
- Unused drivers (gcsblob, etc.) not compiled into binary
- Wire not needed for this use case

### Decision 4: Credential handling via config file with overrides

**Choice**: Credentials are configured in config file, with env var and CLI flag overrides. Never in the storage URL.

**Precedence (highest to lowest)**:
1. Environment variables (e.g., `AWS_ACCESS_KEY_ID`, `AZURE_STORAGE_KEY`)
2. Config file (e.g., `object_storage.aws_access_key_id`)

**Config file structure**:
```yaml
object_storage:
  url: "s3://bucket?endpoint=http://minio:9000&use_path_style=true"
  signed_url_expiry: "168h"

  # S3/MinIO/RustFS credentials
  aws_access_key_id: "minioadmin"
  aws_secret_access_key: "minioadmin"

  # OR Azure credentials
  # azure_storage_account: "myaccount"
  # azure_storage_key: "mykey"
```

**Credential requirements by provider**:

| Provider | Config Keys | Env Var Overrides | Required |
|----------|-------------|-------------------|----------|
| Azure | `azure_storage_account`, `azure_storage_key` | `AZURE_STORAGE_ACCOUNT`, `AZURE_STORAGE_KEY` | Both |
| S3/MinIO/RustFS | `aws_access_key_id`, `aws_secret_access_key` | `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` | Both (or IAM role) |
| Memory | None | None | None |

**Rationale**:
- Config file provides defaults (matches 12-factor principles)
- Env vars override config for container deployments, CI/CD (12-factor compliance)
- Credentials never in URL (URLs are logged)
- IAM roles still work on AWS (credential chain fallback)

### Decision 3: Thin wrapper over blob.Bucket

**Choice**: Create `ObjectStore` struct wrapping `blob.Bucket` with domain helpers.

```go
type ObjectStore struct {
    bucket   *blob.Bucket
    provider string      // "azblob", "s3", "mem"
    expiry   time.Duration
}

func (s *ObjectStore) Upload(ctx context.Context, key string, data []byte) error
func (s *ObjectStore) SignedURL(ctx context.Context, key string) (string, time.Time, error)
func (s *ObjectStore) CanonicalURL(key string) string
func (s *ObjectStore) Close() error
```

**Rationale**:
- Provides canonical URL generation (not in Go CDK)
- Simplifies common operations
- Insulates from Go CDK API changes

### Decision 4: Preserve existing Storage interface

**Choice**: Keep the domain-level `Storage` interface unchanged.

```go
type Storage interface {
    SaveIncident(ctx context.Context, incidentID string, artifacts *IncidentArtifacts) (*SaveResult, error)
}
```

**Rationale**:
- No changes needed in reporting, notification code
- `ObjectStore` is an implementation detail

### Decision 5: Storage selection logic

**Choice**: Simple priority-based selection.

```go
func NewStorage(cfg *Config) (Storage, error) {
    // Cloud storage if OBJECT_STORAGE_URL is set
    if cfg.GetObjectStorageURL() != "" {
        bucket, err := blob.OpenBucket(ctx, cfg.GetObjectStorageURL())
        if err != nil {
            return nil, err
        }
        return NewObjectStoreStorage(bucket, cfg.GetObjectStorageExpiry()), nil
    }

    // Default to existing filesystem storage
    return NewFilesystemStorage(cfg.GetWorkspaceRoot()), nil
}
```

### Decision 6: S3-compatible endpoint configuration

**Choice**: Use Go CDK URL query parameters.

```bash
# MinIO/RustFS
OBJECT_STORAGE_URL="s3://bucket?endpoint=http://localhost:9000&use_path_style=true&disable_https=true&awssdk=v2"
```

**Key query parameters**:
- `endpoint` - Custom S3 endpoint URL
- `use_path_style=true` - Required for most S3-compatible services
- `disable_https=true` - Allow HTTP (development)
- `awssdk=v2` - Use AWS SDK v2 (recommended)
- `region` - AWS region (often ignored by S3-compatible services)

### Decision 7: Three-tier URL model

**Choice**: Implement three distinct URL types for different use cases.

**URL Types**:

| Type | Purpose | Example |
|------|---------|---------|
| Canonical URL | Reference, storage, re-signing | `http://minio:9000/bucket/incident-123/report.html` |
| Signed URL | Clickable access (temporary) | `http://minio:9000/bucket/...?X-Amz-Signature=...` |
| Key | Internal reference | `incident-123/report.html` |

**Storage in result.json**:
```json
{
  "canonical_urls": {
    "investigation.html": "http://minio:9000/bucket/incident-123/investigation.html",
    "incident.json": "http://minio:9000/bucket/incident-123/incident.json"
  },
  "presigned_urls": {
    "investigation.html": "http://minio:9000/bucket/incident-123/investigation.html?X-Amz-...",
    "incident.json": "http://minio:9000/bucket/incident-123/incident.json?X-Amz-..."
  },
  "presigned_urls_expire_at": "2024-01-15T10:00:00Z"
}
```

**Implementation**:

```go
// CanonicalURL returns the base URL without authentication
func (s *ObjectStore) CanonicalURL(key string) string {
    switch s.provider {
    case "azblob":
        return fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s",
            s.account, s.container, key)
    case "s3":
        if s.customEndpoint != "" {
            return fmt.Sprintf("%s/%s/%s", s.customEndpoint, s.bucket, key)
        }
        return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s",
            s.bucket, s.region, key)
    default:
        return key
    }
}

// SignedURL returns a clickable URL with embedded authentication
func (s *ObjectStore) SignedURL(ctx context.Context, key string) (string, time.Time, error) {
    expiry := time.Now().Add(s.expiry)
    url, err := s.bucket.SignedURL(ctx, key, &blob.SignedURLOptions{
        Expiry: s.expiry,
        Method: "GET",
    })
    return url, expiry, err
}

// SignCanonicalURL generates a signed URL from a canonical URL (for re-signing)
func (s *ObjectStore) SignCanonicalURL(ctx context.Context, canonicalURL string) (string, time.Time, error) {
    key := s.extractKeyFromCanonicalURL(canonicalURL)
    return s.SignedURL(ctx, key)
}
```

**Rationale**:
- Canonical URLs are stable references that can be stored long-term
- Signed URLs are temporary but clickable (essential for MinIO/RustFS private buckets)
- Re-signing capability allows regenerating expired links from stored canonical URLs
- index.html uses signed URLs for immediate clickability
- result.json stores both for flexibility

## Risks / Trade-offs

### Risk: S3-compatible service quirks

**Mitigation**:
- Test with RustFS in CI
- Go CDK's `UseLegacyList` handles services without ListObjectsV2

### Risk: Signed URL expiration differences

| Provider | Max Expiry |
|----------|------------|
| AWS S3 (no IAM role) | 7 days |
| AWS S3 (with IAM role) | 1 hour |
| Azure | Configurable (years) |

**Mitigation**: Default to 7 days, document limits.

### Trade-off: Additional dependency

**Accept**: Go CDK adds binary size but significantly reduces code.

## Open Questions

None - scope is well-defined.
