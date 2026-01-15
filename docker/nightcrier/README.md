# Nightcrier Docker Image

This directory contains the Docker build system for the Nightcrier controller.

## Quick Start

```bash
# Build for local testing
make build

# Build and push to registry
make docker-push
```

## Build Targets

| Target | Description |
|--------|-------------|
| `make build` | Build single-arch image for local testing |
| `make build-clean` | Build without cache (fresh pull) |
| `make buildx` | Build multi-arch image (amd64 + arm64) |
| `make buildx-push` | Build multi-arch and push to registry |
| `make release` | Full release workflow (setup + build + push) |

## Registry Examples

### GitHub Container Registry (default)

```bash
make buildx-push
# Pushes to: ghcr.io/randybias/nightcrier:latest
```

### DockerHub

```bash
REGISTRY=docker.io/myuser make buildx-push
# Pushes to: docker.io/myuser/nightcrier:latest
```

### Google Container Registry

```bash
REGISTRY=gcr.io/my-project make buildx-push
# Pushes to: gcr.io/my-project/nightcrier:latest
```

### AWS ECR

```bash
# First, authenticate to ECR
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin 123456789.dkr.ecr.us-east-1.amazonaws.com

# Then push
REGISTRY=123456789.dkr.ecr.us-east-1.amazonaws.com make buildx-push
```

### Custom Version Tag

```bash
VERSION=v1.0.0 make buildx-push
# Creates tags: latest, v1.0.0, and git commit hash
```

## Testing

```bash
# Test version command
make test-version

# Test all debugging tools
make test-tools

# Interactive shell
make shell
```

## Image Details

- **Base**: `debian:bookworm-slim`
- **Size**: ~97MB
- **User**: `nightcrier` (non-root, UID 999)
- **Platforms**: `linux/amd64`, `linux/arm64`

### Included Tools

| Tool | Purpose |
|------|---------|
| curl | API debugging |
| jq | JSON processing |
| sqlite3 | Database inspection |
| procps | Process monitoring (ps, top) |
| netcat | Network debugging |
| dnsutils | DNS debugging (dig, nslookup) |

### Directory Structure

```
/usr/local/bin/nightcrier  # Main binary
/migrations/               # Database migrations
/data/                     # Working directory (writable)
/configs/                  # Config mount point
```

## Running the Container

### Basic Usage

```bash
docker run --rm ghcr.io/randybias/nightcrier:latest --help
```

### With Configuration

```bash
docker run --rm \
  -v $(pwd)/configs:/configs:ro \
  -v $(pwd)/data:/data \
  ghcr.io/randybias/nightcrier:latest \
  --config /configs/config.yaml
```

### With Environment Variables

```bash
docker run --rm \
  -e ANTHROPIC_API_KEY="${ANTHROPIC_API_KEY}" \
  -e SLACK_WEBHOOK_URL="${SLACK_WEBHOOK_URL}" \
  -v $(pwd)/configs:/configs:ro \
  ghcr.io/randybias/nightcrier:latest \
  --config /configs/config.yaml
```

## Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `IMAGE_NAME` | `nightcrier` | Image name |
| `IMAGE_TAG` | `latest` | Image tag |
| `REGISTRY` | `ghcr.io/randybias` | Container registry |
| `VERSION` | auto from git | Version tag |
| `PLATFORMS` | `linux/amd64,linux/arm64` | Build platforms |

## From Project Root

You can also use the root Makefile:

```bash
# From project root
make docker-build    # Build locally
make docker-push     # Build and push
make docker-release  # Full release
```
