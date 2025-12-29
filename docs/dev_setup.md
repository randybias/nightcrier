# Developer Setup Guide

This guide covers setting up a local development environment for Nightcrier.

## Prerequisites

- Go 1.23+
- kubectl
- kind (Kubernetes in Docker)
- make

## Quick Start

```bash
# Clone and build
git clone https://github.com/rbias/nightcrier.git
cd nightcrier
make build

# Set up local kind cluster
./scripts/dev-setup.sh

# Build and load agent image
cd nc-agent-runner
make build
make load-kind
cd ..

# Run Nightcrier
./bin/nightcrier --config configs/local-dev.yaml
```

## Local Kind Cluster

The `scripts/dev-setup.sh` script:
- Creates a kind cluster (`nightcrier-dev`)
- Loads the `nc-agent-runner` image
- Applies namespace and RBAC manifests
- Creates secrets for API keys

## Configuration

Create `configs/local-dev.yaml` for local development. See [configuration.md](configuration.md) for options.

## Running Tests

```bash
make test              # Unit tests
make test-integration  # Integration tests
make test-coverage     # With coverage report
```

## Iterative Development

1. Make code changes
2. Rebuild: `make build`
3. Restart Nightcrier

For agent container changes:
1. Modify `nc-agent-runner/`
2. Rebuild: `cd nc-agent-runner && make build`
3. Reload: `make load-kind`

## Troubleshooting

See [troubleshooting.md](troubleshooting.md) for common issues.

## Additional Resources

- [contributing.md](contributing.md) - Contribution guidelines
- [configuration.md](configuration.md) - Configuration options
- [README](../README.md) - Project overview
