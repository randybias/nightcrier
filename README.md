# Kubernetes MCP Alerts Event Runner

An AI-powered incident triage system that automatically investigates Kubernetes faults using Claude AI agents through the Model Context Protocol (MCP).

## Overview

This system listens for fault events from a Kubernetes MCP server and spawns AI agents to autonomously investigate and triage incidents. It provides automated root cause analysis with configurable storage backends (filesystem or Azure Blob Storage) and Slack notifications.

## Architecture

```
kubernetes-mcp-server -> MCP Events -> Event Runner -> AI Agent -> Investigation Report
                                            |
                                            v
                                    Storage (Azure/Filesystem)
                                            |
                                            v
                                    Slack Notification (with Report URL)
```

## Features

- Automated incident detection via MCP server integration
- AI-powered root cause analysis using Claude agents
- Multi-backend storage support (filesystem and Azure Blob Storage)
- Slack notifications with investigation reports
- Secure artifact storage with SAS URL generation (Azure mode)
- Containerized agent execution environment

## Prerequisites

- Go 1.23 or later
- Docker (for containerized agent execution)
- Kubernetes cluster with MCP server (for production use)
- Azure Storage Account (optional, for cloud storage)
- Slack webhook (optional, for notifications)

## Installation

### Build from source

```bash
# Clone the repository
git clone https://github.com/rbias/kubernetes-mcp-alerts-event-runner.git
cd kubernetes-mcp-alerts-event-runner

# Build the runner
go build -o runner ./cmd/runner

# Build the agent container
cd agent-container
docker build -t k8s-triage-agent:latest .
```

## Configuration

### Environment Variables

#### Required

- `K8S_CLUSTER_MCP_ENDPOINT` - MCP server endpoint URL (e.g., `http://localhost:8080`)

#### Optional - Core Settings

- `WORKSPACE_ROOT` - Directory for incident artifacts (default: `./incidents`)
- `LOG_LEVEL` - Logging level: `debug`, `info`, `warn`, `error` (default: `info`)
- `AGENT_SCRIPT_PATH` - Path to agent execution script (default: `./agent-container/run-agent.sh`)
- `AGENT_SYSTEM_PROMPT_FILE` - Path to system prompt file (default: `./configs/triage-system-prompt.md`)
- `AGENT_ALLOWED_TOOLS` - Comma-separated list of allowed tools (default: `Read,Write,Grep,Glob,Bash,Skill`)
- `AGENT_MODEL` - Claude model to use: `sonnet`, `opus`, `haiku` (default: `sonnet`)
- `AGENT_TIMEOUT` - Agent timeout in seconds (default: `300`)

#### Optional - Slack Notifications

- `SLACK_WEBHOOK_URL` - Slack webhook URL for notifications (if not set, Slack notifications are disabled)

#### Optional - Azure Blob Storage

When Azure storage is configured, incident artifacts are automatically uploaded to Azure Blob Storage and SAS URLs are generated for secure access. If Azure is not configured, the system falls back to filesystem storage.

##### Option 1: Connection String (Recommended)

- `AZURE_STORAGE_CONNECTION_STRING` - Full Azure connection string
- `AZURE_STORAGE_CONTAINER` - Blob container name (required)
- `AZURE_SAS_EXPIRY` - SAS URL expiration duration (default: `168h` / 7 days)

Example:
```bash
export AZURE_STORAGE_CONNECTION_STRING="DefaultEndpointsProtocol=https;AccountName=myaccount;AccountKey=xxxxx;EndpointSuffix=core.windows.net"
export AZURE_STORAGE_CONTAINER="incident-reports"
export AZURE_SAS_EXPIRY="168h"
```

##### Option 2: Account + Key

- `AZURE_STORAGE_ACCOUNT` - Storage account name
- `AZURE_STORAGE_KEY` - Storage account access key
- `AZURE_STORAGE_CONTAINER` - Blob container name (required)
- `AZURE_SAS_EXPIRY` - SAS URL expiration duration (default: `168h` / 7 days)

Example:
```bash
export AZURE_STORAGE_ACCOUNT="myaccount"
export AZURE_STORAGE_KEY="xxxxx"
export AZURE_STORAGE_CONTAINER="incident-reports"
```

### Storage Mode Detection

The system automatically detects which storage backend to use:
- **Azure Storage**: Used when `AZURE_STORAGE_ACCOUNT` or `AZURE_STORAGE_CONNECTION_STRING` is set
- **Filesystem Storage**: Used as fallback when Azure is not configured

### Azure Blob Storage Setup

1. Create a storage account in Azure Portal
2. Create a blob container for incident reports
3. Get connection string or account keys from the Azure Portal
4. Set environment variables as shown above

The system will:
- Upload all artifacts to `<container>/<incident-id>/` structure
- Generate SAS URLs with read-only access
- Include URLs in Slack notifications and result.json
- Set URL expiration based on `AZURE_SAS_EXPIRY`

### Container Requirements

The container must have the following structure:
```
<container>/
  <incident-id>/
    event.json              # Original fault event
    result.json             # Execution result with URLs
    output/
      investigation.md      # AI-generated investigation report
```

## Usage

### Running the Runner

```bash
# With environment variables
export K8S_CLUSTER_MCP_ENDPOINT="http://localhost:8080"
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
./runner

# With command-line flags
./runner --mcp-endpoint http://localhost:8080 --workspace-root ./incidents --log-level debug
```

### Command-line Flags

All configuration can be overridden via CLI flags:
- `--mcp-endpoint` - MCP server endpoint URL
- `--workspace-root` - Workspace root directory
- `--script-path` - Path to agent script
- `--log-level` - Log level (debug, info, warn, error)

## Local Development with Azurite

For local development and testing without an Azure account, use Azurite (Azure Storage Emulator).

### Using Docker Compose

Create `docker-compose.yml`:

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

### Configure for Azurite

Use the default Azurite connection string:

```bash
export AZURE_STORAGE_CONNECTION_STRING="DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://127.0.0.1:10000/devstoreaccount1;"
export AZURE_STORAGE_CONTAINER="incident-reports"
```

Create the container (one-time setup):
```bash
# Install Azure CLI or use Azure Storage Explorer
az storage container create --name incident-reports --connection-string "$AZURE_STORAGE_CONNECTION_STRING"
```

Or using curl:
```bash
curl -X PUT "http://127.0.0.1:10000/devstoreaccount1/incident-reports?restype=container" \
  -H "x-ms-date: $(date -u '+%a, %d %b %Y %H:%M:%S GMT')" \
  -H "x-ms-version: 2021-08-06"
```

### Verify Azurite Setup

```bash
# List containers
az storage container list --connection-string "$AZURE_STORAGE_CONNECTION_STRING" --output table

# After running an incident, list blobs
az storage blob list --container-name incident-reports --connection-string "$AZURE_STORAGE_CONNECTION_STRING" --output table
```

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

# Test with Azurite (requires Azurite running)
export AZURE_STORAGE_CONNECTION_STRING="DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://127.0.0.1:10000/devstoreaccount1;"
export AZURE_STORAGE_CONTAINER="test-incidents"
go test ./internal/storage/... -tags=integration
```

### End-to-End Testing

Since we can't trigger real Kubernetes incidents in a test environment, here's the recommended E2E testing approach:

#### 1. Setup Test Environment

```bash
# Start Azurite
docker-compose up -d

# Configure for Azurite
export AZURE_STORAGE_CONNECTION_STRING="DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://127.0.0.1:10000/devstoreaccount1;"
export AZURE_STORAGE_CONTAINER="incident-reports"
export K8S_CLUSTER_MCP_ENDPOINT="http://localhost:8080"
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"

# Create container
az storage container create --name incident-reports --connection-string "$AZURE_STORAGE_CONNECTION_STRING"

# Build and run
go build -o runner ./cmd/runner
./runner --log-level debug
```

#### 2. Trigger Test Event

Use the MCP server to send a test fault event. The runner will:
1. Receive the event
2. Create a workspace
3. Execute the AI agent
4. Upload artifacts to Azure/filesystem
5. Send Slack notification with report URL

#### 3. Verify Results

**Check Logs:**
```bash
# Look for these log entries:
# - "storage backend initialized" (mode: azure or filesystem)
# - "incident artifacts saved to storage"
# - "slack notification sent"
```

**Check Azure Storage:**
```bash
# List uploaded artifacts
az storage blob list \
  --container-name incident-reports \
  --connection-string "$AZURE_STORAGE_CONNECTION_STRING" \
  --output table

# Download and view investigation report
az storage blob download \
  --container-name incident-reports \
  --name "<incident-id>/output/investigation.md" \
  --file investigation.md \
  --connection-string "$AZURE_STORAGE_CONNECTION_STRING"
```

**Check Slack:**
- Verify notification received
- Click "View Report" button
- Confirm SAS URL works and report is accessible

**Check Result JSON:**
```bash
# View result with presigned URLs
cat ./incidents/<incident-id>/result.json

# Should contain:
# - "presigned_urls": {"event.json": "...", "investigation.md": "..."}
# - "presigned_urls_expire_at": "2025-12-25T..."
```

#### 4. Test URL Expiration

Wait for SAS URL to expire (or set short expiry for testing):
```bash
export AZURE_SAS_EXPIRY="1m"
```

Then verify URL becomes inaccessible after expiration.

#### 5. Test Filesystem Fallback

```bash
# Unset Azure config
unset AZURE_STORAGE_CONNECTION_STRING
unset AZURE_STORAGE_ACCOUNT
unset AZURE_STORAGE_KEY

# Run again
./runner

# Verify:
# - Logs show "mode: filesystem"
# - Artifacts saved to local filesystem
# - Slack notification shows file path instead of URL
```

## Output Structure

### Filesystem Storage
```
./incidents/
  <incident-id>/
    event.json              # Original fault event
    result.json             # Execution result
    output/
      investigation.md      # AI-generated investigation report
```

### Azure Storage
```
<container>/
  <incident-id>/
    event.json
    result.json
    output/
      investigation.md
```

Plus SAS URLs in result.json and Slack notifications.

## Slack Notification Format

When Slack is configured, notifications include:
- Incident metadata (cluster, namespace, resource)
- Root cause analysis with confidence level
- Investigation duration
- **"View Report" button** (when Azure storage is enabled)
- File path (when filesystem storage is used)

## Troubleshooting

### Azure Storage Issues

**Problem**: "failed to initialize Azure storage"
- Check connection string format
- Verify account name and key are correct
- Ensure container exists
- Check network connectivity to Azure

**Problem**: "failed to upload blob"
- Verify container exists and has correct permissions
- Check storage account access keys
- Ensure container name is lowercase (Azure requirement)

**Problem**: SAS URL returns 403 Forbidden
- Check URL hasn't expired
- Verify SAS token permissions (should include read)
- Ensure blob exists at the path

### Azurite Issues

**Problem**: Connection refused to Azurite
- Ensure Azurite is running: `docker-compose ps`
- Check port 10000 is accessible
- Verify connection string uses `http://` not `https://`

**Problem**: Container not found
- Create container: `az storage container create --name incident-reports --connection-string "$AZURE_STORAGE_CONNECTION_STRING"`
- Verify with: `az storage container list --connection-string "$AZURE_STORAGE_CONNECTION_STRING"`

### Debug Mode

Enable debug logging to see detailed storage operations:
```bash
./runner --log-level debug
```

Look for these log entries:
- "storage backend initialized" - Shows which backend is active
- "incident artifacts saved to storage" - Shows successful upload
- "failed to save incident to storage" - Shows upload errors

## Contributing

See [openspec/AGENTS.md](openspec/AGENTS.md) for development workflow and contribution guidelines.

## License

[Add your license here]

## Related Projects

- [kubernetes-mcp-server](https://github.com/rbias/kubernetes-mcp-server) - MCP server for Kubernetes fault events
- [Model Context Protocol](https://github.com/anthropics/mcp) - Protocol specification
