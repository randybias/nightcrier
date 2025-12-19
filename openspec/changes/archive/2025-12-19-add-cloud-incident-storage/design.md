## Context

The current system stores incident artifacts on the local filesystem. Operators viewing Slack notifications must SSH into the host to access reports. This proposal adds cloud storage with authenticated URLs so operators can click a link in Slack and immediately view incident data.

The initial implementation targets Azure Blob Storage, but the interface is designed to support future backends (S3, GCS) without breaking changes.

## Goals / Non-Goals

**Goals:**
- Upload incident artifacts to Azure Blob Storage
- Generate SAS (Shared Access Signature) URLs for secure, time-limited access
- Include clickable links in Slack notifications
- Simple configuration via environment variables
- Design extensible interface for future storage backends

**Non-Goals:**
- Supporting both cloud and local storage simultaneously (either/or)
- Container lifecycle management (retention policies managed externally)
- Automatic container creation (container must exist)
- Implementing S3/GCS backends in this change (interface only)

## Decisions

### Decision: Azure SDK for Go

Use the official Azure SDK for Go (`github.com/Azure/azure-sdk-for-go/sdk/storage/azblob`) for Blob Storage operations.

**Rationale:** Official SDK, well-maintained, native SAS URL generation, handles authentication properly.

### Decision: Storage Interface

Define an abstract `Storage` interface that Azure implements, allowing future backends.

```go
type Storage interface {
    // SaveIncident uploads all artifacts for an incident
    SaveIncident(ctx context.Context, incidentID string, artifacts *IncidentArtifacts) (*SaveResult, error)
}

type SaveResult struct {
    // ReportURL is the authenticated URL to the investigation report
    ReportURL string
    // ArtifactURLs maps artifact names to their authenticated URLs
    ArtifactURLs map[string]string
    // ExpiresAt is when the URLs expire
    ExpiresAt time.Time
}

type IncidentArtifacts struct {
    EventJSON       []byte
    ResultJSON      []byte
    InvestigationMD []byte
}
```

**Rationale:** Adding S3 or GCS later requires only implementing this interface, no changes to calling code.

### Decision: Individual File Uploads

Upload each artifact as a separate blob rather than bundling into an archive.

**Structure:**
```
<container>/<incident-id>/event.json
<container>/<incident-id>/result.json
<container>/<incident-id>/output/investigation.md
```

**Rationale:** Individual files allow SAS URLs to specific artifacts. Operators typically only need the investigation report.

### Decision: SAS Token Authentication

Use Azure Shared Access Signatures (SAS) for authenticated access.

**SAS Types:**
- **Service SAS** - Scoped to specific blob, uses storage account key
- **User Delegation SAS** - Uses Azure AD credentials (more secure, requires AAD setup)

**Initial Implementation:** Service SAS with storage account key (simpler setup).

**Rationale:**
- Single-click access from Slack (no login required)
- Time-limited (7-day default) for security
- No operator credential management needed
- Can upgrade to User Delegation SAS later for enhanced security

### Decision: Either/Or Storage Mode

Configure either cloud storage OR local filesystem, not both.

**Detection logic:**
- If `AZURE_STORAGE_ACCOUNT` is set, use Azure Blob Storage
- Otherwise, use local filesystem storage

**Rationale:** Simplifies implementation and avoids dual-write complexity.

### Decision: Connection String vs Individual Credentials

Support both Azure connection string and individual credential configuration.

**Option A - Connection String (recommended for simplicity):**
```bash
AZURE_STORAGE_CONNECTION_STRING="DefaultEndpointsProtocol=https;AccountName=...;AccountKey=..."
```

**Option B - Individual credentials:**
```bash
AZURE_STORAGE_ACCOUNT="mystorageaccount"
AZURE_STORAGE_KEY="base64encodedkey..."
```

**Rationale:** Connection strings are common in Azure deployments and contain all needed info. Individual credentials offer more flexibility.

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| Azure upload failure loses incident data | Log error but don't fail investigation; artifacts still in container filesystem during execution |
| SAS URLs shared beyond intended audience | 7-day expiration limits exposure; operators should treat URLs as sensitive |
| Storage account key in environment | Standard practice; can upgrade to Managed Identity in production |
| Network latency for uploads | Upload happens after investigation completes; doesn't block triage |

## Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `AZURE_STORAGE_CONNECTION_STRING` | Yes* | - | Full Azure connection string (alternative to individual creds) |
| `AZURE_STORAGE_ACCOUNT` | Yes* | - | Storage account name |
| `AZURE_STORAGE_KEY` | Yes* | - | Storage account access key |
| `AZURE_STORAGE_CONTAINER` | Yes | - | Blob container name |
| `AZURE_SAS_EXPIRY` | No | `168h` | SAS URL expiration (7 days) |

*Either connection string OR account+key required when using Azure storage

## Future Extensibility

To add S3 support later:

1. Implement `Storage` interface in `internal/storage/s3.go`
2. Add S3 configuration detection (e.g., `S3_ENDPOINT` set)
3. Update storage factory to return S3 implementation
4. No changes needed to reporting or Slack integration

```go
// Future: storage factory
func NewStorage(cfg *config.Config) (Storage, error) {
    if cfg.AzureStorageAccount != "" || cfg.AzureConnectionString != "" {
        return NewAzureStorage(cfg)
    }
    if cfg.S3Endpoint != "" {
        return NewS3Storage(cfg)  // Future implementation
    }
    return NewFilesystemStorage(cfg)
}
```

## Open Questions

1. Should we support Azure Managed Identity for authentication?
   - **Proposed:** Not in initial implementation; storage account key is simpler
   - Can add later as an enhancement

2. Should the container be auto-created if it doesn't exist?
   - **Proposed:** No - container must exist; auto-creation adds permission complexity
