# Usage

## Running Nightcrier

```bash
# With configuration file
./bin/nightcrier --config configs/config.yaml

# With environment variables
export WORKSPACE_ROOT="./workspaces"
export AGENT_CLI="claude"
export AGENT_MODEL="claude-sonnet-4-5-20250929"
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
./bin/nightcrier --config configs/config.yaml

# With debug logging
./bin/nightcrier --config configs/config.yaml --log-level debug
```

## Command-line Flags

Configuration can be overridden via CLI flags:
- `--config` - Path to configuration file
- `--log-level` - Log level (debug, info, warn, error)
- `--workspace-root` - Workspace root directory

## Local Development and Testing

### Using MinIO (S3-compatible)

For local development and testing with S3-compatible storage, use MinIO:

```yaml
version: '3.8'

services:
  minio:
    image: minio/minio:latest
    ports:
      - "9000:9000"  # API
      - "9001:9001"  # Console
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    volumes:
      - minio-data:/data
    command: server /data --console-address ":9001"

volumes:
  minio-data:
```

Start MinIO:
```bash
docker-compose up -d
```

Configure Nightcrier for MinIO:
```bash
export OBJECT_STORAGE_URL="s3://incident-reports?endpoint=http://localhost:9000&disable_https=true&use_path_style=true&region=us-east-1"
export AWS_ACCESS_KEY_ID="minioadmin"
export AWS_SECRET_ACCESS_KEY="minioadmin"
```

Create the bucket using MinIO Console (http://localhost:9001) or AWS CLI:
```bash
# Using AWS CLI configured for MinIO
aws --endpoint-url http://localhost:9000 s3 mb s3://incident-reports
```

### Using Azurite (Azure-compatible)

For local development and testing with Azure Blob Storage compatibility, use Azurite:

```yaml
version: '3.8'

services:
  azurite:
    image: mcr.microsoft.com/azure-storage/azurite:latest
    ports:
      - "10000:10000"  # Blob service
      - "10001:10001"  # Queue service
      - "10002:10002"  # Table service
    volumes:
      - azurite-data:/data
    command: azurite-blob --blobHost 0.0.0.0 --blobPort 10000 --location /data --debug /data/debug.log

volumes:
  azurite-data:
```

Start Azurite:
```bash
docker-compose up -d
```

Configure Nightcrier for Azurite:
```bash
export OBJECT_STORAGE_URL="azblob://incident-reports"
export AZURE_STORAGE_ACCOUNT="devstoreaccount1"
export AZURE_STORAGE_KEY="Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw=="
```

Create the container (one-time setup):
```bash
# Install Azure CLI or use Azure Storage Explorer
az storage container create --name incident-reports \
  --connection-string "DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://127.0.0.1:10000/devstoreaccount1;"
```

Or using curl:
```bash
curl -X PUT "http://127.0.0.1:10000/devstoreaccount1/incident-reports?restype=container" \
  -H "x-ms-date: $(date -u '+%a, %d %b %Y %H:%M:%S GMT')" \
  -H "x-ms-version: 2021-08-06"
```

Verify Azurite Setup:
```bash
# List containers
az storage container list \
  --connection-string "DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://127.0.0.1:10000/devstoreaccount1;" \
  --output table

# After running an incident, list blobs
az storage blob list --container-name incident-reports \
  --connection-string "DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://127.0.0.1:10000/devstoreaccount1;" \
  --output table
```

### Using In-Memory Storage (Testing Only)

For unit tests and quick testing without any external storage service:

```bash
export OBJECT_STORAGE_URL="mem://"
```

Note: Data is lost when the process restarts.

## Testing

### Unit Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/storage/... -v
go test ./internal/config/... -v
```

### Integration Tests

```bash
# Test storage backends
go test ./internal/storage/... -v

# Test with in-memory storage (no external dependencies)
export OBJECT_STORAGE_URL="mem://"
go test ./internal/storage/... -v

# Test with Azurite (requires Azurite running)
export OBJECT_STORAGE_URL="azblob://test-incidents"
export AZURE_STORAGE_ACCOUNT="devstoreaccount1"
export AZURE_STORAGE_KEY="Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw=="
go test ./internal/storage/... -tags=integration

# Test with MinIO (requires MinIO running)
export OBJECT_STORAGE_URL="s3://test-incidents?endpoint=http://localhost:9000&disable_https=true&use_path_style=true&region=us-east-1"
export AWS_ACCESS_KEY_ID="minioadmin"
export AWS_SECRET_ACCESS_KEY="minioadmin"
go test ./internal/storage/... -tags=integration
```
