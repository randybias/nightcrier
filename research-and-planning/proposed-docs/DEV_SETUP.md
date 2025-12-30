# Developer Setup Guide

This guide covers setting up a local development environment for Nightcrier.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Local Development with kind](#local-development-with-kind)
- [Building from Source](#building-from-source)
- [Running Tests](#running-tests)
- [Configuration](#configuration)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### Required Tools

- **Go 1.21+** - [Install Go](https://go.dev/doc/install)
- **Docker** - [Install Docker](https://docs.docker.com/get-docker/)
- **kubectl** - [Install kubectl](https://kubernetes.io/docs/tasks/tools/)
- **kind** (Kubernetes in Docker) - [Install kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
- **make** - Usually pre-installed on Unix systems

### Optional Tools

- **PostgreSQL client** - For database operations
- **Azure CLI** - If using Azure Blob Storage
- **AWS CLI** - If using S3 storage

## Quick Start

1. **Clone the repository**
   ```bash
   git clone https://github.com/randybias/nightcrier.git
   cd nightcrier
   ```

2. **Set up local kind cluster**
   ```bash
   ./scripts/dev-setup.sh
   ```

   This script will:
   - Create a kind cluster (`nightcrier-dev`)
   - Load the `nc-agent-runner` image
   - Apply namespace and RBAC manifests
   - Prompt for API keys (or use test values)
   - Create kubeconfig secret for self-triage

3. **Build Nightcrier**
   ```bash
   make build
   ```

4. **Run Nightcrier**
   ```bash
   ./bin/nightcrier --config configs/local-dev.yaml
   ```

## Local Development with kind

### Cluster Setup

The `scripts/dev-setup.sh` script automates kind cluster setup:

```bash
./scripts/dev-setup.sh
```

**What it does:**
- Checks prerequisites (kind, kubectl, docker)
- Creates kind cluster with config from `deploy/dev/kind-config.yaml`
- Loads `nc-agent-runner:latest` image into kind
- Applies Kubernetes manifests:
  - Namespace: `nightcrier`
  - ServiceAccount: `nightcrier-executor`
  - RBAC: Role and RoleBinding for Job/ConfigMap management
- Creates secrets:
  - `agent-api-keys` - AI agent API keys
  - `kubeconfig-nightcrier-dev` - Kubeconfig for self-triage

### API Keys Setup

During `dev-setup.sh`, you'll be prompted for API keys:
- **Anthropic API Key** - For Claude agents
- **OpenAI API Key** - For Codex agents
- **Gemini API Key** - For Gemini agents

Alternatively, store keys in `~/dev-secrets/api-keys.env`:
```bash
export ANTHROPIC_API_KEY="sk-ant-..."
export OPENAI_API_KEY="sk-proj-..."
export GEMINI_API_KEY="AIza..."
```

### Manual Secret Management

**Update API keys:**
```bash
kubectl edit secret agent-api-keys -n nightcrier
```

**Update kubeconfig:**
```bash
kubectl edit secret kubeconfig-nightcrier-dev -n nightcrier
```

### Cluster Teardown

```bash
./scripts/dev-teardown.sh
```

This will:
- Delete the kind cluster
- Optionally remove local secrets files

## Building from Source

### Build Nightcrier Binary

```bash
make build
```

Output: `bin/nightcrier`

### Build Agent Container Image

#### Single-arch (for local testing)
```bash
cd nc-agent-runner
make build
```

#### Multi-arch (for registry push)
```bash
cd nc-agent-runner
make buildx-push
```

**Supported platforms:**
- `linux/amd64`
- `linux/arm64`

### Load Image into kind

```bash
cd nc-agent-runner
make load-kind
```

Or manually:
```bash
kind load docker-image nc-agent-runner:latest --name nightcrier-dev
```

## Running Tests

### Unit Tests

```bash
make test
```

### Integration Tests

```bash
make test-integration
```

### Specific Package Tests

```bash
go test ./internal/agent/...
go test ./internal/storage/...
```

### Coverage Report

```bash
make test-coverage
```

## Configuration

### Local Development Config

Create `configs/local-dev.yaml`:

```yaml
# Cluster Configuration
clusters:
  - name: nightcrier-dev
    mcp:
      endpoint: "http://localhost:30870/mcp"
    triage:
      enabled: true
      kubeconfig: "~/.kube/config"

# Workspace
workspace_root: "./workspaces"

# Logging
log_level: "debug"

# Agent Configuration
agent_cli: "claude"
agent_model: "claude-sonnet-4-5-20250929"
agent_timeout: 600
agent_system_prompt_file: "./configs/triage-system-prompt.md"

# K8s Executor (use kind cluster)
k8s_namespace: "nightcrier"
k8s_image: "nc-agent-runner:latest"
k8s_timeout: 600
k8s_memory_limit: "2Gi"
k8s_cpu_limit: "1"
k8s_cleanup_ttl: 3600
kubeconfig_path: "~/.kube/config"
kubernetes_context: "kind-nightcrier-dev"

# Object Storage (in-memory for local dev)
object_storage:
  url: "mem://"
  signed_url_expiry: "1h"

# State Storage (SQLite for local dev)
state_storage:
  type: "sqlite"
  sqlite_path: "./nightcrier.db"
  migrations_path: "./migrations"

# Event Processing
max_concurrent_agents: 3
global_queue_size: 100
```

### Environment Variables

Nightcrier supports environment variable overrides:

```bash
export LOG_LEVEL=debug
export WORKSPACE_ROOT=./workspaces
export AGENT_CLI=claude
export AGENT_MODEL=claude-sonnet-4-5-20250929
export ANTHROPIC_API_KEY=sk-ant-...
```

See `internal/config/config.go` for all available environment variables.

### Using PostgreSQL Instead of SQLite

```yaml
state_storage:
  type: "postgres"
  postgres_host: "localhost"
  postgres_port: 5432
  postgres_database: "nightcrier"
  postgres_user: "nightcrier"
  postgres_password: "your-password"
  migrations_path: "./migrations"
```

Or use connection string:
```yaml
state_storage:
  type: "postgres"
  postgres_connection_string: "postgres://user:pass@localhost:5432/nightcrier?sslmode=disable"
  migrations_path: "./migrations"
```

## Troubleshooting

### kind Cluster Won't Start

**Problem:** `failed to create cluster: context deadline exceeded`

**Solutions:**
1. Check Docker is running: `docker ps`
2. Increase Docker resources (CPU/Memory in Docker Desktop)
3. Try simpler kind config: `deploy/dev/kind-config.yaml`

### Image Not Found in kind

**Problem:** `ErrImagePull` when Job starts

**Solution:**
```bash
# Rebuild and reload image
cd nc-agent-runner
make build
make load-kind

# Verify image is loaded
docker exec nightcrier-dev-control-plane crictl images | grep nc-agent-runner
```

### API Keys Not Working

**Problem:** Agent fails with authentication error

**Solution:**
```bash
# Check secrets exist
kubectl get secrets -n nightcrier

# View secret (base64 encoded)
kubectl get secret agent-api-keys -n nightcrier -o yaml

# Update secret
kubectl delete secret agent-api-keys -n nightcrier
kubectl create secret generic agent-api-keys \
  --from-literal=ANTHROPIC_API_KEY="sk-ant-..." \
  --from-literal=OPENAI_API_KEY="sk-proj-..." \
  --from-literal=GEMINI_API_KEY="AIza..." \
  -n nightcrier
```

### Database Migration Errors

**Problem:** `failed to run migrations`

**Solution:**
```bash
# Check migrations directory exists
ls -la migrations/

# Run migrations manually
cd migrations
# For SQLite:
sqlite3 ../nightcrier.db < 001_initial_schema.sql

# For PostgreSQL:
psql -h localhost -U nightcrier -d nightcrier -f 001_initial_schema.sql
```

### Permission Denied on Scripts

**Problem:** `permission denied: ./scripts/dev-setup.sh`

**Solution:**
```bash
chmod +x scripts/*.sh
```

### Port Already in Use

**Problem:** `bind: address already in use`

**Solution:**
```bash
# Find process using port
lsof -i :8080

# Kill process or change port
./bin/nightcrier --health-port 8081
```

## Development Workflow

### Iterative Development

1. **Make code changes**
2. **Rebuild:**
   ```bash
   make build
   ```
3. **Restart Nightcrier:**
   ```bash
   ./bin/nightcrier --config configs/local-dev.yaml
   ```

### Agent Container Development

1. **Modify** `nc-agent-runner/entrypoint.sh` or `Dockerfile`
2. **Rebuild:**
   ```bash
   cd nc-agent-runner
   make build
   ```
3. **Reload into kind:**
   ```bash
   make load-kind
   ```
4. **Test:** Create test incident or wait for real event

### Testing Agent Changes

Create a test Job:
```bash
kubectl apply -f deploy/dev/test-job.yaml
```

Watch logs:
```bash
kubectl logs -n nightcrier -l job-name=nc-agent-test-001 -f
```

## IDE Setup

### VS Code

Recommended extensions:
- Go (golang.go)
- Kubernetes (ms-kubernetes-tools.vscode-kubernetes-tools)
- YAML (redhat.vscode-yaml)

### GoLand

1. Open project
2. Enable Go modules support
3. Configure run configuration for `cmd/nightcrier/main.go`

## Additional Resources

- [CONTRIBUTING.md](CONTRIBUTING.md) - Contribution guidelines
- [README.md](../README.md) - Project overview
- [OpenSpec Changes](../openspec/changes/) - Implementation specs
- [Research & Planning](../research-and-planning/) - Design docs

## Getting Help

- **Issues:** [GitHub Issues](https://github.com/randybias/nightcrier/issues)
- **Discussions:** [GitHub Discussions](https://github.com/randybias/nightcrier/discussions)
- **Slack:** Join #nightcrier channel (if available)
